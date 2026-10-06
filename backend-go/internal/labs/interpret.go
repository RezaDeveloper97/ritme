package labs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/labs/store"
)

// Evaluated is a marker read against its range and the catalog.
type Evaluated struct {
	Row        store.LabMarker
	Value      *float64
	Confidence *float64
	Range      Range
	State      string
	Marker     *Marker // nil = not catalogued
}

// LowConfidence reports whether the extractor was unsure about the value.
func (e Evaluated) LowConfidence() bool { return e.Confidence != nil && *e.Confidence < LowConfidence }

// sheetRange is the range printed on the sheet.
func sheetRange(m store.LabMarker) Range {
	r := Range{Low: number(m.RefLow.String, m.RefLow.Valid), High: number(m.RefHigh.String, m.RefHigh.Valid),
		Text: m.RefText.String}
	if !r.Empty() {
		r.Source = RangeSheet
	}
	return r
}

// evaluate classifies m: the sheet's range first; the catalog's typical range only when the sheet has none and the
// units match (labs and methods differ, so the sheet always wins).
func evaluate(m store.LabMarker, cat *Catalog) Evaluated {
	e := Evaluated{Row: m, Value: number(m.Value.String, m.Value.Valid), Confidence: number(m.Confidence.String, m.Confidence.Valid)}
	if m.Code.Valid {
		if mk, ok := cat.ByCode(m.Code.String); ok {
			e.Marker = &mk
		}
	}
	e.Range = sheetRange(m)
	if e.Range.Empty() && e.Marker != nil && UnitsMatch(m.Unit.String, e.Marker.Meta.Unit) {
		e.Range = e.Marker.TypicalRange()
	}
	if e.Range.Empty() && e.Range.Text == "" {
		e.Range.Text = m.RefText.String
	}
	e.State = Classify(e.Value, e.Range)
	return e
}

// RedFlag is a value beyond a catalog threshold.
type RedFlag struct {
	MarkerID  uint64
	Code      string
	Name      string
	Severity  string // urgent | soon
	Direction string // low | high
}

// Severities.
const (
	SeverityUrgent = "urgent"
	SeveritySoon   = "soon"
)

// redFlag is the red flag of e, if any (only in the catalog unit: a threshold never applies across units).
func redFlag(e Evaluated, name string) (RedFlag, bool) {
	if e.Marker == nil || e.Value == nil || !UnitsMatch(e.Row.Unit.String, e.Marker.Meta.Unit) {
		return RedFlag{}, false
	}
	v, c := *e.Value, e.Marker.Meta.Critical
	f := RedFlag{MarkerID: e.Row.ID, Code: e.Marker.Code, Name: name}
	switch {
	case c.UrgentLow != nil && v < *c.UrgentLow:
		f.Severity, f.Direction = SeverityUrgent, "low"
	case c.UrgentHigh != nil && v > *c.UrgentHigh:
		f.Severity, f.Direction = SeverityUrgent, "high"
	case c.SoonLow != nil && v < *c.SoonLow:
		f.Severity, f.Direction = SeveritySoon, "low"
	case c.SoonHigh != nil && v >= *c.SoonHigh:
		f.Severity, f.Direction = SeveritySoon, "high"
	default:
		return RedFlag{}, false
	}
	return f, true
}

// Counts are the result tallies (nbl_Lab_Result «۲۱/۲۴ طبیعی · ۳ مورد نیاز به توجه · ۲ پایین · ۱ پایین مرزی»).
type Counts struct {
	Total, Normal, Attention, Unknown        int
	Low, BorderlineLow, High, BorderlineHigh int
}

func countStates(evals []Evaluated) Counts {
	c := Counts{Total: len(evals)}
	for _, e := range evals {
		switch e.State {
		case StateNormal:
			c.Normal++
		case StateUnknown:
			c.Unknown++
		case StateLow:
			c.Low++
		case StateBorderlineLow:
			c.BorderlineLow++
		case StateHigh:
			c.High++
		case StateBorderlineHigh:
			c.BorderlineHigh++
		}
	}
	c.Attention = c.Low + c.BorderlineLow + c.High + c.BorderlineHigh
	return c
}

// loc picks translations for one response.
type loc struct {
	Locale, Default string
}

func (l loc) pick(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	return strings.TrimSpace(i18n.PickString(raw, l.Locale, l.Default))
}

// name is the display name of e: the catalog title, else the printed name.
func (l loc) name(e Evaluated) string {
	if e.Marker != nil {
		if s := l.pick(e.Marker.Title); s != "" {
			return s
		}
	}
	return e.Row.Name
}

// MaxQuestions caps the doctor questions of a result.
const MaxQuestions = 5

// questions are the doctor questions: the catalog's for each out-of-range marker (clear lows / highs first,
// borderline after), then a generic follow-up, deduplicated.
func questions(evals []Evaluated, l loc) []string {
	out, seen := []string{}, map[string]bool{}
	add := func(s string) {
		if s != "" && !seen[s] && len(out) < MaxQuestions {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, pass := range [][]string{{StateLow, StateHigh}, {StateBorderlineLow, StateBorderlineHigh}} {
		for _, e := range evals {
			if e.State != pass[0] && e.State != pass[1] || e.Marker == nil {
				continue
			}
			if adv := e.Marker.AdviceFor(e.State); adv != nil {
				for _, q := range adv.Questions {
					add(l.pick(q))
				}
			}
		}
	}
	if countStates(evals).Attention > 0 {
		add(T("questions.repeat", l.Locale, nil))
	} else {
		add(T("questions.routine", l.Locale, nil))
	}
	return out
}

// Interpretation is the stored part of a lab's interpretation (lab_reports.interpretation).
type Interpretation struct {
	Summary     string    `json:"summary"`
	Source      string    `json:"source"` // ai | rules
	Locale      string    `json:"locale"`
	GeneratedAt time.Time `json:"generated_at"`
	Context     struct {
		Mode        string `json:"mode"`
		Phase       string `json:"phase,omitempty"`
		Medications int    `json:"medications"`
	} `json:"context"`
}

// Summary sources.
const (
	SummaryAI    = "ai"
	SummaryRules = "rules"
)

// rulesSummary is the summary without AI: what is out of range, then «talk to your doctor».
func rulesSummary(evals []Evaluated, l loc) string {
	c := countStates(evals)
	switch {
	case c.Total == 0:
		return T("summary.empty", l.Locale, nil)
	case c.Attention == 0 && c.Normal == 0:
		return T("summary.no_ranges", l.Locale, nil) + " " + T("summary.doctor", l.Locale, nil)
	case c.Attention == 0:
		return T("summary.all_normal", l.Locale, map[string]string{"count": num(c.Normal, l.Locale)}) + " " +
			T("summary.doctor", l.Locale, nil)
	}
	var parts []string
	for _, e := range evals {
		if Attention(e.State) {
			parts = append(parts, T("summary.item", l.Locale, map[string]string{"name": l.name(e), "state": T("states."+e.State, l.Locale, nil)}))
		}
	}
	return T("summary.attention", l.Locale, map[string]string{
		"count": num(c.Attention, l.Locale), "list": strings.Join(parts, T("summary.separator", l.Locale, nil)),
	}) + " " + T("summary.doctor", l.Locale, nil)
}

// Interpretation prompt limits.
const (
	summaryMaxTokens = 400
	summaryMaxRunes  = 1500
	interpretTimeout = 60 * time.Second
)

// systemPrompt are the guardrails of the summary (B-N6-06): plain words, non-diagnostic, «talk to your doctor».
const systemPrompt = `You explain a woman's lab test results in plain, warm, simple words, like a knowledgeable friend.
You are not a doctor. Never give a diagnosis or say she has a disease; use careful wording such as "may" or "can be related to".
Never recommend a medicine, supplement dose or treatment. Use only the values given; do not invent values or ranges.
Each value is compared with the reference range printed on her own lab sheet; say when a hormone depends on the cycle day.
Take her life stage, approximate cycle phase and medications into account when they are relevant.
If a value is marked URGENT, say clearly that she should contact a doctor today.
Always end by suggesting she discusses the results with her doctor.
At most 120 words, one or two short paragraphs, no lists, no headings, no greeting, no names.
Reply in the language whose ISO code is %s.`

func modeLabel(m enums.LifeMode) string {
	switch m {
	case enums.LifeModeTTC:
		return "trying to conceive"
	case enums.LifeModePregnancy:
		return "pregnant"
	case enums.LifeModePostpartum:
		return "postpartum"
	case enums.LifeModeMenopause:
		return "menopause"
	case enums.LifeModeTeen:
		return "teenager tracking her cycle"
	}
	return "tracking her menstrual cycle"
}

func fmtNum(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

// promptLine is one result line of the prompt.
func promptLine(e Evaluated, flags map[uint64]RedFlag) string {
	var b strings.Builder
	name := e.Row.Name
	if e.Marker != nil && e.Marker.Code != "" {
		name += " (" + e.Marker.Code + ")"
	}
	b.WriteString("- " + name + ": ")
	switch {
	case e.Value != nil:
		b.WriteString(fmtNum(e.Value))
	case e.Row.ValueText.Valid:
		b.WriteString(e.Row.ValueText.String)
	}
	if e.Row.Unit.Valid {
		b.WriteString(" " + e.Row.Unit.String)
	}
	switch {
	case e.Range.Source == RangeSheet && e.Range.Text != "":
		b.WriteString(" [lab range " + e.Range.Text + "]")
	case !e.Range.Empty():
		b.WriteString(" [range " + fmtNum(e.Range.Low) + "–" + fmtNum(e.Range.High) + "]")
	}
	b.WriteString(" → " + strings.ReplaceAll(e.State, "_", " "))
	if f, ok := flags[e.Row.ID]; ok && f.Severity == SeverityUrgent {
		b.WriteString(" URGENT")
	}
	return b.String()
}

// buildPrompt is the summary request: guardrails in System, context + results as the one user message. No name,
// no id, no date of birth (age band only).
func buildPrompt(evals []Evaluated, uc UserContext, locale string, flags map[uint64]RedFlag) ai.ChatRequest {
	var b strings.Builder
	b.WriteString("Context: life stage: " + modeLabel(uc.Mode))
	if band := ageBand(uc.Age); band != "" {
		b.WriteString("; age " + band)
	}
	if uc.Phase != "" {
		b.WriteString("; approximate cycle phase today: " + uc.Phase)
	}
	if len(uc.Medications) > 0 {
		b.WriteString("; medications: " + strings.Join(uc.Medications, ", "))
	}
	b.WriteString(".\nResults:\n")
	for _, e := range evals {
		b.WriteString(promptLine(e, flags) + "\n")
	}
	return ai.ChatRequest{
		System:          fmt.Sprintf(systemPrompt, locale),
		Messages:        []ai.ChatMessage{{Role: ai.RoleUser, Text: b.String()}},
		Language:        locale,
		MaxOutputTokens: summaryMaxTokens,
	}
}

// errNoSummary: the provider answered nothing usable (blocked, cut off, empty).
var errNoSummary = errors.New("labs: no usable summary")

// aiSummary asks the AI client for the summary (non-streaming for the caller: the stream is read to the end here,
// bounded by interpretTimeout, and always closed).
func aiSummary(ctx context.Context, client *ai.Client, req ai.ChatRequest, names []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, interpretTimeout)
	defer cancel()
	stream, err := client.Chat(ctx, ai.FeatureLabAnalysis, req)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	var b strings.Builder
	for ev := range stream.Events {
		switch {
		case ev.Err != nil:
			return "", ev.Err
		case ev.Done:
			if ev.Finish == ai.FinishSafety {
				return "", errNoSummary
			}
			text := strings.TrimSpace(ai.Redact(b.String(), names))
			if text == "" {
				return "", errNoSummary
			}
			if utf8.RuneCountInString(text) > summaryMaxRunes {
				text = string([]rune(text)[:summaryMaxRunes]) + "…"
			}
			return text, nil
		default:
			if b.Len() < summaryMaxRunes*4 {
				b.WriteString(ev.Delta)
			}
		}
	}
	return "", errNoSummary
}
