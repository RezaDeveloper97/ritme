package manager

import (
	"context"
	"strconv"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// modules are the supplementary NutritionModule, SleepModule and ExerciseModule
// (backend/app/Services/MessageSystem/Modules).
type modules struct {
	content Content
	locale  string
}

// phaseItem is `($phase !== null && array_key_exists($phase, $tips)) ? $phase : 'follicular'`.
func phaseItem(group, phase string) string {
	if phase == "" {
		return "follicular"
	}
	return itemOr(group, phase, "follicular")
}

// trimesterItem is `(string) (array_key_exists($trimester, $tips) ? $trimester : 1)`.
func trimesterItem(group string, trimester int) string {
	return itemOr(group, strconv.Itoa(trimester), "1")
}

// nutrition is NutritionModule::getTips.
func (m modules) nutrition(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
	if mc.IsPregnancyMode() {
		const group = "nutrition_trimester"
		p, err := m.content.Resolve(ctx, group, trimesterItem(group, mc.trimesterOr1()), m.locale)
		if err != nil {
			return nil, err
		}
		return jsonx.Obj(
			"focus", p.Or("focus", ""),
			"essential", p.Or("essential", emptyList()),
			"foods", p.Or("foods", emptyList()),
			"avoid", p.Or("avoid", emptyList()),
			"tip", p.Or("tip", ""),
		), nil
	}

	const group = "nutrition_cycle"
	p, err := m.content.Resolve(ctx, group, phaseItem(group, mc.CyclePhase), m.locale)
	if err != nil {
		return nil, err
	}
	out := jsonx.Obj(
		"focus", p.Or("focus", ""),
		"foods", p.Or("foods", emptyList()),
		"avoid", p.Or("avoid", emptyList()),
		"tip", p.Or("tip", ""),
	)
	if mc.IsTTC() {
		t, err := m.content.Resolve(ctx, "nutrition_ttc", "base", m.locale)
		if err != nil {
			return nil, err
		}
		ttc := jsonx.Obj(
			"essential", t.Or("essential", emptyList()),
			"foods", t.Or("foods", emptyList()),
			"avoid", t.Or("avoid", emptyList()),
		)
		if mc.CyclePhase == "ovulation" {
			ttc.Set("fertile_window_tip", t.Or("fertile_window_tip", ""))
		}
		out.Set("ttc_additions", ttc)
	}
	if len(mc.Symptoms) > 0 {
		tips := jsonx.NewArray()
		for _, s := range []struct {
			key, tip string
			applies  bool
		}{
			{"for_cramps", "cramps", mc.HasSymptom("cramps")},
			{"for_bloating", "bloating", mc.HasSymptom("bloating")},
			{"for_fatigue", "fatigue", mc.HasSymptom("fatigue", "low_energy")},
			{"for_headache", "headache", mc.HasSymptom("headache")},
		} {
			if !s.applies {
				continue
			}
			p, err := m.content.Resolve(ctx, "nutrition_symptom", s.tip, m.locale)
			if err != nil {
				return nil, err
			}
			tips.Set(s.key, p.Or("tip", ""))
		}
		if tips.Len() > 0 {
			out.Set("symptom_specific", tips)
		}
	}
	return out, nil
}

// sleep is SleepModule::getTips.
func (m modules) sleep(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
	if mc.IsPregnancyMode() {
		const group = "sleep_trimester"
		p, err := m.content.Resolve(ctx, group, trimesterItem(group, mc.trimesterOr1()), m.locale)
		if err != nil {
			return nil, err
		}
		return jsonx.Obj(
			"recommended_hours", p.Or("recommended_hours", ""),
			"quality_focus", p.Or("quality_focus", ""),
			"tips", p.Or("tips", emptyList()),
			"avoid", p.Or("avoid", emptyList()),
			"position", p.Or("position", ""),
		), nil
	}

	const group = "sleep_cycle"
	p, err := m.content.Resolve(ctx, group, phaseItem(group, mc.CyclePhase), m.locale)
	if err != nil {
		return nil, err
	}
	out := jsonx.Obj(
		"recommended_hours", p.Or("recommended_hours", ""),
		"quality_focus", p.Or("quality_focus", ""),
		"tips", p.Or("tips", emptyList()),
		"avoid", p.Or("avoid", emptyList()),
	)
	if mc.HasSymptom("poor_sleep") {
		s, err := m.content.Resolve(ctx, "sleep_symptom", "poor_sleep", m.locale)
		if err != nil {
			return nil, err
		}
		out.Set("symptom_tip", s.Or("tip", ""))
	}
	return out, nil
}

// exercise is ExerciseModule::getTips.
func (m modules) exercise(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
	if mc.IsPregnancyMode() {
		const group = "exercise_trimester"
		p, err := m.content.Resolve(ctx, group, trimesterItem(group, mc.trimesterOr1()), m.locale)
		if err != nil {
			return nil, err
		}
		return jsonx.Obj(
			"intensity", p.Or("intensity", ""),
			"recommended", p.Or("recommended", emptyList()),
			"avoid", p.Or("avoid", emptyList()),
			"tip", p.Or("tip", ""),
			"duration", p.Or("duration", ""),
			"precautions", p.Or("precautions", emptyList()),
		), nil
	}

	const group = "exercise_cycle"
	p, err := m.content.Resolve(ctx, group, phaseItem(group, mc.CyclePhase), m.locale)
	if err != nil {
		return nil, err
	}
	out := jsonx.Obj(
		"intensity", p.Or("intensity", ""),
		"recommended", p.Or("recommended", emptyList()),
		"avoid", p.Or("avoid", emptyList()),
		"tip", p.Or("tip", ""),
		"duration", p.Or("duration", ""),
	)
	// The fatigue tip overwrites the pain tip (same key, first position kept).
	for _, s := range []struct {
		key     string
		applies bool
	}{
		{"pain", mc.HasSymptom("cramps", "pain")},
		{"fatigue", mc.HasSymptom("low_energy", "fatigue")},
	} {
		if !s.applies {
			continue
		}
		t, err := m.content.Resolve(ctx, "exercise_symptom", s.key, m.locale)
		if err != nil {
			return nil, err
		}
		out.Set("symptom_modification", t.Or("tip", ""))
	}
	if mc.IsTTC() {
		t, err := m.content.Resolve(ctx, "exercise_ttc", "note", m.locale)
		if err != nil {
			return nil, err
		}
		out.Set("ttc_note", t.Or("tip", ""))
	}
	return out, nil
}
