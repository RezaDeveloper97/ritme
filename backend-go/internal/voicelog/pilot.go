package voicelog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Voice → autopilot («بگو، ریتمی پر می‌کند»): the recording is transcribed, then a model writes a UI autopilot
// script (docs/autopilot.md) from the transcript and the manifest the client sent — the fields of the log sheet
// and the values each accepts. Nothing is saved here and nothing is run here: the client validates the script
// again and plays it on the user's own screen, and the user presses save herself.
//
// The manifest is client input and the script is model output: neither is trusted. The model never sees the
// client's manifest as sent — only a copy rebuilt here (the rules and the action list are this package's, each
// target is reduced to its id, kind, short labels, option codes and bounds), so the request cannot be turned into
// a general prompt. Of the answer only `set` steps survive, on a listed target, with a value that target accepts.

// Pilot limits.
const (
	// MaxManifestBytes caps the manifest part of the upload (a real log sheet is about 10 KB).
	MaxManifestBytes = 48 << 10
	// MaxManifestTargets caps the controls of one manifest.
	MaxManifestTargets = 300
	// MaxPilotSteps caps the script returned (the engine's own limit is 60).
	MaxPilotSteps = 40
	// maxTargetLen bounds a target id; maxLabelRunes a label; maxOptions the choices of one target.
	maxTargetLen  = 120
	maxLabelRunes = 80
	maxOptions    = 120
	maxCodeLen    = 64
	// maxTextRunes bounds a text value when the target names no maxLength.
	maxTextRunes = 2000
)

// pilotGuide are the rules of the script the model is asked for. Server-owned: a client cannot rewrite them.
const pilotGuide = `Reply with ONE JSON object and nothing else: {"version":1,"steps":[...]}.
Each step is {"do":"set","target":<id>,"value":<value>}. No other action exists.
Use only the target ids listed here. Never invent an id.
Values by kind — single: one option "value"; multi: an array of option values to add; bool: true/false; number/integer: a number inside min..max; text: a string. Use option "value" codes, never labels.
Only fill what the user actually said.
If nothing in the request maps to this list, reply {"version":1,"steps":[]}.`

// ErrManifest: the manifest part is not a usable manifest.
var ErrManifest = errors.New("voicelog: invalid manifest")

// PilotResult is one recording turned into a script.
type PilotResult struct {
	Transcript string
	Language   string
	Steps      []PilotStep
}

// PilotStep is one validated `set` step of the script.
type PilotStep struct {
	Target string
	Value  json.RawMessage
}

type pilotOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// pilotTarget is one field as the model is told about it (and what its values are checked against).
type pilotTarget struct {
	ID        string        `json:"id"`
	Kind      string        `json:"kind"`
	Label     string        `json:"label"`
	Group     string        `json:"group,omitempty"`
	Options   []pilotOption `json:"options,omitempty"`
	Min       *float64      `json:"min,omitempty"`
	Max       *float64      `json:"max,omitempty"`
	Unit      string        `json:"unit,omitempty"`
	MaxLength int           `json:"maxLength,omitempty"`
}

// pilotKinds are the kinds a voice note may set.
var pilotKinds = map[string]bool{"single": true, "multi": true, "bool": true, "number": true, "integer": true, "text": true}

// readManifest reads the client's manifest and returns its targets by id plus the manifest the model is sent,
// rebuilt from scratch: only the known fields of each target, trimmed, under this package's own rules.
func readManifest(manifest []byte) (map[string]pilotTarget, []byte, error) {
	var m struct {
		Targets []pilotTarget `json:"targets"`
	}
	if err := json.Unmarshal(manifest, &m); err != nil || len(m.Targets) == 0 || len(m.Targets) > MaxManifestTargets {
		return nil, nil, ErrManifest
	}
	byID := make(map[string]pilotTarget, len(m.Targets))
	clean := make([]pilotTarget, 0, len(m.Targets))
	for _, t := range m.Targets {
		if t.ID == "" || len(t.ID) > maxTargetLen || !pilotKinds[t.Kind] {
			return nil, nil, ErrManifest
		}
		// A very long list of her own custom items is cut, not refused: voice still works for the rest.
		if len(t.Options) > maxOptions {
			t.Options = t.Options[:maxOptions]
		}
		if _, dup := byID[t.ID]; dup {
			return nil, nil, ErrManifest
		}
		t.Label, t.Group, t.Unit = truncate(t.Label, maxLabelRunes), truncate(t.Group, maxLabelRunes), truncate(t.Unit, 20)
		if t.MaxLength < 0 || t.MaxLength > maxTextRunes {
			t.MaxLength = 0
		}
		for i, o := range t.Options {
			if o.Value == "" || len(o.Value) > maxCodeLen {
				return nil, nil, ErrManifest
			}
			t.Options[i].Label = truncate(o.Label, maxLabelRunes)
		}
		byID[t.ID] = t
		clean = append(clean, t)
	}
	raw, err := json.Marshal(struct {
		Version int           `json:"version"`
		Guide   string        `json:"guide"`
		Actions []string      `json:"actions"`
		Targets []pilotTarget `json:"targets"`
	}{Version: 1, Guide: pilotGuide, Actions: []string{"set"}, Targets: clean})
	if err != nil {
		return nil, nil, ErrManifest
	}
	return byID, raw, nil
}

// accepts reports whether value is something target can be set to.
func (t pilotTarget) accepts(value json.RawMessage) bool {
	has := func(code string) bool {
		for _, o := range t.Options {
			if o.Value == code {
				return true
			}
		}
		return false
	}
	switch t.Kind {
	case "single":
		var code string
		return json.Unmarshal(value, &code) == nil && has(code)
	case "multi":
		var codes []string
		if json.Unmarshal(value, &codes) != nil || len(codes) == 0 || len(codes) > len(t.Options) {
			return false
		}
		for _, c := range codes {
			if !has(c) {
				return false
			}
		}
		return true
	case "bool":
		var b bool
		return json.Unmarshal(value, &b) == nil && (string(value) == "true" || string(value) == "false")
	case "number", "integer":
		var n float64
		if string(value) == "null" || json.Unmarshal(value, &n) != nil {
			return false
		}
		if t.Kind == "integer" && n != float64(int64(n)) {
			return false
		}
		return (t.Min == nil || n >= *t.Min) && (t.Max == nil || n <= *t.Max)
	case "text":
		var text string
		if json.Unmarshal(value, &text) != nil || strings.TrimSpace(text) == "" {
			return false
		}
		limit := t.MaxLength
		if limit == 0 {
			limit = maxTextRunes
		}
		return utf8.RuneCountInString(text) <= limit
	}
	return false
}

// filterScript keeps the `set` steps of a model's answer that put an accepted value on a listed target. An answer
// that is not the JSON asked for yields no steps (the provider was paid: the use is kept, like a silent clip).
// A model's captions (`say`) are never passed on: they would be shown as Ritme's own words.
func filterScript(raw string, targets map[string]pilotTarget) []PilotStep {
	var parsed struct {
		Steps []struct {
			Do     string          `json:"do"`
			Target json.RawMessage `json:"target"`
			Value  json.RawMessage `json:"value"`
		} `json:"steps"`
	}
	out := []PilotStep{}
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		return out
	}
	for _, s := range parsed.Steps {
		if len(out) >= MaxPilotSteps {
			break
		}
		var id string
		if s.Do != "set" || json.Unmarshal(s.Target, &id) != nil {
			continue // {role,name} / {css} locators are never accepted from a model
		}
		t, ok := targets[id]
		if !ok || len(s.Value) == 0 || !t.accepts(s.Value) {
			continue
		}
		out = append(out, PilotStep{Target: id, Value: s.Value})
	}
	return out
}

// Pilot transcribes audio (wiping it as soon as the provider answered) and asks for the script. An empty
// transcript is a valid result without steps.
func (s *Service) Pilot(ctx context.Context, locale string, manifest []byte, audio *ai.Audio, now time.Time) (PilotResult, error) {
	targets, rebuilt, err := readManifest(manifest)
	if err != nil {
		audio.Wipe()
		return PilotResult{}, err
	}
	t, err := s.ai.Transcribe(ctx, ai.FeatureVoiceSTT, ai.TranscribeRequest{Audio: *audio, Language: locale})
	audio.Wipe()
	if err != nil {
		return PilotResult{}, err
	}
	res := PilotResult{Transcript: truncate(strings.TrimSpace(t.Text), MaxTranscriptRunes), Language: t.Language, Steps: []PilotStep{}}
	if res.Language == "" {
		res.Language = locale
	}
	if res.Transcript == "" {
		return res, nil
	}
	raw, err := s.ai.PilotScript(ctx, ai.FeatureVoiceLog, ai.PilotRequest{Transcript: res.Transcript, Manifest: rebuilt,
		Language: res.Language, Today: civildate.InTehran(now).String()})
	if err != nil {
		return PilotResult{}, err
	}
	res.Steps = filterScript(raw, targets)
	return res, nil
}
