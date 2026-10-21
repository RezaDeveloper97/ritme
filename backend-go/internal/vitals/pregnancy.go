package vitals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Vitals in pregnancy (T-M9-05, D-134). When the reading's day falls in the user's active pregnancy (from the dated
// start, LMP-equivalent, to today), the reading is classified with the pregnancy table instead of ACC/AHA / ADA:
//
//   - BP: below pregnancy.bp.high (140/90) normal, from it stage2, from pregnancy.bp.severe (160/110) crisis — the
//     existing class codes, so clients and reports keep their labels; urgent = the severe tier;
//   - glucose: the GDM targets (pregnancy.gdm.targets) replace the target ceiling of the fasting / before-meal
//     (fasting 95) and after-meal (= 2 h, 120) contexts; bedtime / random keep the ADA bands. There is no 1 h vitals
//     context (the weekly log's post-meal value is the 1 h slot);
//   - the reading is handed to the pregnancy alert engine (pregnancyalerts.EvaluateReadings: rules bp_high and
//     sugar_high), which persists one alert row per fact; an urgent one is the save's modal instead of the vitals
//     copy, so one reading never makes two alerts.
//
// Outside pregnancy nothing changes.

// pregnancyDays is the user's active pregnancy as a day range (ok false when none).
type pregnancyDays struct {
	from, to civildate.Date
	ok       bool
}

func (p pregnancyDays) contains(d civildate.Date) bool {
	return p.ok && !d.Before(p.from) && !d.After(p.to)
}

// mark sets Pregnant on the readings of rs that fall in p.
func (p pregnancyDays) mark(rs []Reading) {
	if !p.ok {
		return
	}
	for i := range rs {
		rs[i].Pregnant = p.contains(rs[i].Date)
	}
}

// pregnancy loads the user's active pregnancy (today = now in Tehran).
func (s *Service) pregnancy(ctx context.Context, userID uint64) (pregnancyDays, error) {
	if s.preg == nil {
		return pregnancyDays{}, nil
	}
	p, err := s.preg.GetProfileByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return pregnancyDays{}, nil
	}
	if err != nil {
		return pregnancyDays{}, fmt.Errorf("vitals: pregnancy profile: %w", err)
	}
	if !p.PregnancyMode {
		return pregnancyDays{}, nil
	}
	today := civildate.InTehran(s.now())
	d, ok := v2.Resolve(&p, today)
	if !ok {
		return pregnancyDays{}, nil
	}
	return pregnancyDays{from: d.Start(), to: today, ok: true}, nil
}

func (s *Service) markPregnancy(ctx context.Context, userID uint64, rs []Reading) error {
	p, err := s.pregnancy(ctx, userID)
	if err != nil {
		return err
	}
	p.mark(rs)
	return nil
}

// pregnancyBP is the pregnancy class of a BP reading.
func pregnancyBP(systolic, diastolic float64, day civildate.Date) string {
	switch pregnancyalerts.BPTier(int(systolic), int(diastolic), day) {
	case "severe":
		return ClassCrisis
	case "high":
		return ClassStage2
	}
	return ClassNormal
}

// GDMSlot is the GDM slot of a glucose context ("" when the GDM targets do not apply).
func GDMSlot(context string) string {
	switch context {
	case ContextFasting, ContextBeforeMeal:
		return pregnancyalerts.GlucoseFasting
	case ContextAfterMeal:
		return pregnancyalerts.GlucoseTwoHour
	}
	return ""
}

// pregnancyGlucose is the class of a glucose reading in pregnancy: the ADA class with the GDM target as ceiling.
func (t Thresholds) pregnancyGlucose(mgdl float64, context string) string {
	slot := GDMSlot(context)
	if slot == "" {
		return t.ClassifyGlucose(mgdl, context)
	}
	b := t.GlucoseBands[context]
	b.TargetMax = pregnancyalerts.GDMTarget(slot, t.On)
	if b.VeryHighFrom <= b.TargetMax {
		b.VeryHighFrom = b.TargetMax + 1
	}
	bands := make(map[string]GlucoseBand, len(t.GlucoseBands))
	for k, v := range t.GlucoseBands {
		bands[k] = v
	}
	bands[context] = b
	t.GlucoseBands = bands
	return t.ClassifyGlucose(mgdl, context)
}

// PregnancyReading is the reading as the pregnancy alert engine reads it (ok false for heart rate / no GDM slot).
func PregnancyReading(r Reading) (pregnancyalerts.Reading, bool) {
	switch r.Type {
	case TypeBP:
		return pregnancyalerts.Reading{Date: r.Date, Systolic: int(r.Systolic), Diastolic: int(r.Diastolic)}, true
	case TypeGlucose:
		if slot := GDMSlot(r.Context); slot != "" {
			return pregnancyalerts.Reading{Date: r.Date, MgDl: r.MgDl, Slot: slot}, true
		}
	}
	return pregnancyalerts.Reading{}, false
}

// PregnancyAlerts hands a pregnant user's reading to the pregnancy alert engine and returns its alerts (created now
// or already there for the same fact), rendered in l. Nil for a reading outside pregnancy.
func (s *Service) PregnancyAlerts(ctx context.Context, userID uint64, r Reading, now time.Time, l v2.Lang,
) ([]*jsonx.OrderedMap, error) {
	in, ok := PregnancyReading(r)
	if !r.Pregnant || !ok || s.alerts == nil {
		return nil, nil
	}
	out, err := s.alerts.EvaluateReadings(ctx, userID, []pregnancyalerts.Reading{in}, now, l)
	if err != nil {
		return nil, fmt.Errorf("vitals: pregnancy alerts: %w", err)
	}
	return out, nil
}

// emergencyPhone is the number of the call action of the vitals urgent copy (bp_crisis), "" when none.
func (c AlertCopy) emergencyPhone(locale, defaultLocale string) string {
	m := c.pick(RuleBPCrisis, locale, defaultLocale)
	list, _ := m["actions"].([]any)
	for _, a := range list {
		am, _ := a.(map[string]any)
		if phone, _ := am["phone"].(string); phone != "" {
			return phone
		}
	}
	return ""
}

// PregnancyModalJSON is the save's urgent modal for a pregnancy alert (the shape of AlertJSON, rule
// "pregnancy_<rule_key>"), or nil when the alert is not urgent. The call action dials the alert's contact phone, else
// the vitals urgent copy's number.
func (c AlertCopy) PregnancyModalJSON(a *jsonx.OrderedMap, locale, defaultLocale string) any {
	get := func(k string) any { v, _ := a.Get(k); return v }
	if lvl, _ := get("level").(string); lvl != "urgent" {
		return nil
	}
	rule, _ := get("rule_key").(string)
	var contact any
	phone := c.emergencyPhone(locale, defaultLocale)
	switch x := get("contact").(type) {
	case string:
		contact = x
	case *jsonx.OrderedMap:
		contact, _ = x.Get("text")
		if p, _ := x.Get("phone"); p != nil {
			if ps, ok := p.(string); ok && ps != "" {
				phone = ps
			}
		}
	}
	actions := []any{}
	list, _ := get("actions").([]any)
	for _, raw := range list {
		am, _ := raw.(*jsonx.OrderedMap)
		if am == nil {
			continue
		}
		key, _ := am.Get("key")
		label, _ := am.Get("label")
		ks, _ := key.(string)
		ls, _ := label.(string)
		if ks == "" || ls == "" || (ks != "call" && ks != "ack") {
			continue
		}
		act := jsonx.Obj("key", ks, "label", ls)
		if ks == "call" && phone != "" {
			act.Set("phone", phone)
		}
		actions = append(actions, act)
	}
	return jsonx.Obj(
		"rule", "pregnancy_"+rule,
		"level", AlertLevel,
		"modal", true,
		"title", get("title"),
		"what_we_saw", get("what_we_saw"),
		"advice", get("advice"),
		"contact", contact,
		"actions", actions,
	)
}
