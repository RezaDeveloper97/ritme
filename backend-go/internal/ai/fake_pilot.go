package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// fakeManifest is the part of the autopilot manifest the fake reads.
type fakeManifest struct {
	Targets []struct {
		ID      string `json:"id"`
		Kind    string `json:"kind"`
		Label   string `json:"label"`
		Options []struct {
			Value string `json:"value"`
			Label string `json:"label"`
		} `json:"options"`
	} `json:"targets"`
}

// PilotScript implements Scripter without a model: a control is set when one of its option labels (single /
// multi) or its own label (bool) is said as a whole word. Same transcript and manifest → same script. A transcript
// containing the fixture word of "error" never reaches here (Transcribe fails first).
func (f *Fake) PilotScript(_ context.Context, req PilotRequest) (string, Usage, error) {
	u := Usage{Provider: "fake", Model: FakeModel, InputTokens: (utf8.RuneCountInString(req.Transcript) + len(req.Manifest)) / 4}
	var m fakeManifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return "", u, fmt.Errorf("%w: manifest", ErrInvalidRequest)
	}
	text := NormalizeText(req.Transcript)
	steps := []map[string]any{}
	for _, t := range m.Targets {
		switch t.Kind {
		case "single", "multi":
			picked := []string{}
			for _, o := range t.Options {
				if label := NormalizeText(o.Label); label != "" && wordAt(text, label, nil) {
					picked = append(picked, o.Value)
				}
			}
			if len(picked) == 0 {
				continue
			}
			var value any = picked
			if t.Kind == "single" {
				value = picked[0]
			}
			steps = append(steps, map[string]any{"do": "set", "target": t.ID, "value": value})
		case "bool":
			if label := NormalizeText(t.Label); label != "" && wordAt(text, label, nil) {
				steps = append(steps, map[string]any{"do": "set", "target": t.ID, "value": true})
			}
		}
	}
	raw, err := json.Marshal(map[string]any{"version": 1, "steps": steps})
	if err != nil {
		return "", u, fmt.Errorf("%w: encode script", ErrUpstream)
	}
	u.OutputTokens = len(raw) / 4
	return string(raw), u, nil
}
