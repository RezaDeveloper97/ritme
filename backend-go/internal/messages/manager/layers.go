package manager

import (
	"context"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// correlationLayer is CorrelationLayer (Layer 3,
// backend/app/Services/MessageSystem/Layers/CorrelationLayer.php).
type correlationLayer struct {
	content Content
	locale  string
}

type correlationRule struct {
	group, typ   string
	premiumOnly  bool
	applies      func(mc *Context) bool
	cycleOnly    bool
	pregnantOnly bool
}

// correlationRules keep the PHP evaluation order: cycle or pregnancy rules, then common ones.
var correlationRules = []correlationRule{
	{group: "correlation_cycle", typ: "sleep_mood", cycleOnly: true, applies: func(mc *Context) bool {
		return mc.HasSymptom("poor_sleep") && mc.HasSymptom("mood_sad", "mood_angry")
	}},
	{group: "correlation_cycle", typ: "stress_physical", cycleOnly: true, applies: func(mc *Context) bool {
		return mc.HasSymptom("mood_anxious") && mc.HasSymptom("headache", "cramps")
	}},
	{group: "correlation_cycle", typ: "pms_symptoms", cycleOnly: true, applies: func(mc *Context) bool {
		return mc.IsPmsWindow && mc.HasSymptom("bloating", "mood_sad")
	}},
	{group: "correlation_cycle", typ: "fertile_window_ttc", cycleOnly: true, applies: func(mc *Context) bool {
		return mc.IsTTC() && mc.IsFertileWindow
	}},
	{group: "correlation_cycle", typ: "luteal_energy_drop", premiumOnly: true, cycleOnly: true, applies: func(mc *Context) bool {
		return mc.CyclePhase == "luteal" && mc.HasSymptom("low_energy")
	}},
	{group: "correlation_cycle", typ: "hormone_skin", premiumOnly: true, cycleOnly: true, applies: func(mc *Context) bool {
		return mc.HasSymptom("acne") && (mc.CyclePhase == "luteal" || mc.CyclePhase == "menstruation")
	}},
	{group: "correlation_pregnancy", typ: "first_trimester_combo", pregnantOnly: true, applies: func(mc *Context) bool {
		return trimesterIs(mc, 1) && mc.HasSymptom("nausea") && mc.HasSymptom("fatigue")
	}},
	{group: "correlation_pregnancy", typ: "third_trimester_discomfort", pregnantOnly: true, applies: func(mc *Context) bool {
		return trimesterIs(mc, 3) && mc.HasSymptom("backache", "fatigue")
	}},
	{group: "correlation_pregnancy", typ: "pregnancy_mood", premiumOnly: true, pregnantOnly: true, applies: func(mc *Context) bool {
		return mc.HasSymptom("mood_sad", "mood_anxious")
	}},
	{group: "correlation_common", typ: "dehydration_indicator", applies: func(mc *Context) bool {
		return mc.HasSymptom("headache") && mc.HasSymptom("fatigue")
	}},
	{group: "correlation_common", typ: "sleep_deprivation", applies: func(mc *Context) bool {
		return mc.HasSymptom("poor_sleep") && mc.HasSymptom("low_energy")
	}},
}

// trimesterIs is `$trimester === n`.
func trimesterIs(mc *Context, n int) bool { return mc.Trimester != nil && *mc.Trimester == n }

// analyze is CorrelationLayer::analyze (unfiltered; the manager drops premium-only entries for
// free users).
func (l correlationLayer) analyze(ctx context.Context, mc *Context) ([]*jsonx.OrderedMap, error) {
	out := []*jsonx.OrderedMap{}
	if len(mc.Symptoms) == 0 {
		return out, nil
	}
	for _, r := range correlationRules {
		if (r.cycleOnly && !mc.IsCycleMode()) || (r.pregnantOnly && !mc.IsPregnancyMode()) || !r.applies(mc) {
			continue
		}
		p, err := l.content.Resolve(ctx, r.group, r.typ, l.locale)
		if err != nil {
			return nil, err
		}
		out = append(out, jsonx.Obj(
			"type", r.typ,
			"insight_message", p.Or("insight_message", ""),
			"action", p.Or("action", ""),
			"is_premium_only", r.premiumOnly,
		))
	}
	return out, nil
}

// patternLayer is PatternLayer (Layer 4, premium only,
// backend/app/Services/MessageSystem/Layers/PatternLayer.php). The detectors read the logs'
// toArray(); most fields they read do not exist (D-05), so on real data only
// insufficient_data, ttc_tracking and chronic_fatigue fire.
type patternLayer struct {
	content Content
	locale  string
}

func (l patternLayer) analyze(ctx context.Context, mc *Context) ([]*jsonx.OrderedMap, error) {
	logs := mc.RecentLogs
	type hit struct{ typ, level string }
	var hits []hit
	add := func(typ, level string) { hits = append(hits, hit{typ, level}) }

	if len(logs) < 14 {
		add("insufficient_data", "info")
	} else {
		if mc.IsCycleMode() {
			if typ := detectPMSPattern(logs); typ != "" {
				add(typ, "info")
			}
			if typ, level := detectPainPattern(logs); typ != "" {
				add(typ, level)
			}
			if typ, level := detectMoodPattern(logs); typ != "" {
				add(typ, level)
			}
			// detectCycleLengthPattern always returns null.
			if mc.IsTTC() {
				add("ttc_tracking", "info") // detectTTCPattern: unconditional placeholder
			}
		} else if mc.IsPregnancyMode() {
			if typ := detectSymptomProgression(logs, mc.trimesterOr1()); typ != "" {
				add(typ, "info")
			}
		}
		if countLogs(logs, func(g Log) bool {
			return g.isString("sleep_quality", "poor") || g.isString("sleep_quality", "very_poor")
		}) >= 10 {
			add("poor_sleep", "warning")
		}
		if countLogs(logs, func(g Log) bool { return g.isString("energy_level", "low") || g.isString("energy_level", "very_low") }) >= 15 {
			add("chronic_fatigue", "warning")
		}
	}

	out := make([]*jsonx.OrderedMap, 0, len(hits))
	for _, h := range hits {
		p, err := l.content.Resolve(ctx, "pattern", h.typ, l.locale)
		if err != nil {
			return nil, err
		}
		out = append(out, jsonx.Obj(
			"pattern_type", h.typ,
			"alert_level", h.level,
			"message", p.Or("message", ""),
			"recommendation", p.Or("recommendation", ""),
		))
	}
	return out, nil
}

func countLogs(logs []Log, pred func(Log) bool) int {
	n := 0
	for _, g := range logs {
		if pred(g) {
			n++
		}
	}
	return n
}

// detectPMSPattern: `has_` + symptom without the `mood_` prefix, summed over every log.
func detectPMSPattern(logs []Log) string {
	fields := []string{"has_bloating", "has_sad", "has_angry", "has_headache", "has_breast_tenderness"}
	n := 0
	for _, g := range logs {
		for _, f := range fields {
			if g.truthy(f) {
				n++
			}
		}
	}
	if n > 10 {
		return "recurring_pms"
	}
	return ""
}

func detectPainPattern(logs []Log) (string, string) {
	pain, severe := 0, 0
	for _, g := range logs {
		if g.truthy("has_cramps") {
			pain++
			if g.isString("cramp_severity", "severe") {
				severe++
			}
		}
	}
	switch {
	case severe >= 3:
		return "severe_pain", "warning"
	case pain >= 7:
		return "frequent_pain", "info"
	}
	return "", ""
}

func detectMoodPattern(logs []Log) (string, string) {
	sad := countLogs(logs, func(g Log) bool { return g.isString("mood", "sad") || g.isString("mood", "depressed") })
	anxious := countLogs(logs, func(g Log) bool { return g.isString("mood", "anxious") || g.isString("mood", "stressed") })
	switch {
	case sad >= 10:
		return "low_mood", "warning"
	case anxious >= 7:
		return "anxiety_pattern", "info"
	}
	return "", ""
}

// detectSymptomProgression compares has_nausea in the newest 7 logs with the 7 before.
func detectSymptomProgression(logs []Log, trimester int) string {
	if len(logs) < 14 {
		return ""
	}
	nausea := func(part []Log) int { return countLogs(part, func(g Log) bool { return g.truthy("has_nausea") }) }
	recent, older := nausea(logs[:7]), nausea(logs[7:14])
	switch {
	case trimester == 1 && recent > older+2:
		return "increasing_nausea"
	case trimester == 2 && recent < older-2:
		return "improving_symptoms"
	}
	return ""
}
