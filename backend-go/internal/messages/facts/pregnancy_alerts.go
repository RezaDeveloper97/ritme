package facts

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Pregnancy alert facts (T-M10-08, D-165, DECISIONS.md M-10; SPEC §5.4.1): the pregnancy alert detectors
// (messages/pregnancyalerts) are the producers. Each condition fact is a number, class or set computed from the user's
// own logs of the last 7 days; the thresholds of BP, GDM, kicks and 5-1-1 come from the clinical registry (T-M9-05),
// every other threshold is a condition of the seeded rule (rule_key preg.<name>), so the admin edits it there.
//
// Each alert also gets its evidence as text variables (preg_*, Variable: true): «what we saw» is built from the
// values the user logged («این الگو از ثبتهای خودته، نه تشخیص»), and the inbox keys a row by the variables of its
// evidence texts, so the same data never raises an alert twice and new data (a new reading, a new session, a new
// day) does. Dates and times are ISO (Y-m-d, Y-m-d H:i) and symptom / status / basis values are codes: language-free
// evidence, localized when an alert is rendered.
//
// The safety facts (fetal_movement, contractions_511, critical_symptom) are present only while they hold — an urgent
// alert is never resolved by the data moving on, only acknowledged by the user (SPEC §5.4.1). The other facts are
// present whenever the pregnancy is (false / 0 / a class), so their alerts resolve when the pattern ends. Outside an
// active, dated pregnancy every fact is absent.

// PregnancyAlertInput is what the producer reads: the detectors' input for the target day, plus the two inputs the
// detectors do not load (the last weight log, the next visit).
type PregnancyAlertInput struct {
	Facts pregnancyalerts.Facts
	// LastWeight is the day of the newest logged weight (zero = none); Start the first day of the pregnancy.
	LastWeight, Start civildate.Date
	// NextVisit is the next upcoming visit (zero = none).
	NextVisit time.Time
}

// PregnancyAlertSource reads the pregnancy alert input of the source's user on day (nil = no active, dated
// pregnancy). A PregnancySource that also implements it produces the group; without it the facts are absent.
type PregnancyAlertSource interface {
	PregnancyAlertInput(ctx context.Context, day civildate.Date) (*PregnancyAlertInput, error)
}

// GroupPregnancyAlerts is the pregnancy alert producer group.
const GroupPregnancyAlerts = "pregnancy_alerts"

// Fetal movement states (preg.fetal_movement).
const (
	FetalReduced    = "reduced"     // manual log «reduced», from pregnancy.fetal_movement.manual_from_week
	FetalNone       = "none"        // manual log «no movement», same weeks
	FetalLowSession = "low_session" // a kick session short of pregnancy.kicks.target within the window, from week 28
)

// Classes of preg.bp_last_category / preg.gdm_last / preg.bp_high_level.
const (
	BPNormal      = "normal"
	BPHigh        = "high"
	BPSevere      = "severe"
	GDMInTarget   = "in_target"
	GDMAbove      = "above_target"
	LevelFollowUp = "follow_up"
	LevelUrgent   = "urgent"
)

// CriticalSymptoms are the symptoms of preg.critical_symptom (spotting only up to week 12 or when severe).
var CriticalSymptoms = []string{"bleeding", "fluid_leakage", "severe_sudden_pain", "spotting"}

// SevereSymptoms are the symptoms preg.severe_symptoms_7 counts: the v2 severe_symptom_count list plus the v1
// countSevereSymptoms fields it lacked (dizziness, lower abdominal pain, cramping, pelvic pressure), so no v1
// escalation is lost.
var SevereSymptoms = []string{"nausea", "vomiting", "fatigue", "headache", "back_pain", "breast_pain", "heartburn",
	"constipation", "dizziness", "lower_abdominal_pain", "cramping", "pelvic_pressure"}

// SpottingUntilWeek is the last week a mild spotting log is critical (the v2 / v1 rule).
const SpottingUntilWeek = 12

// noValue is a missing number inside a reading (a BP with one number, a GDM log with one slot).
const noValue = "—"

func init() {
	g := GroupPregnancyAlerts
	cond := func(name string, t Type, enum []string, fa, en string) Def {
		return Def{Name: name, Type: t, Enum: enum, Group: g, Description: desc(fa, en)}
	}
	unit := func(d Def, u string) Def { d.Unit = u; return d }
	v := func(name, fa, en string) Def {
		return Def{Name: name, Type: TypeEnum, Group: g, Variable: true, Description: desc(fa, en)}
	}
	n := func(name, fa, en string) Def {
		return Def{Name: name, Type: TypeInt, Group: g, Variable: true, Description: desc(fa, en)}
	}
	Register(
		unit(cond("preg.vomiting_days_7", TypeInt, nil, "روزهای دارای استفراغ در ۷ روز اخیر",
			"Days with vomiting in the last 7 days."), "day"),
		cond("preg.vomiting_severe_7", TypeInt, nil, "روزهای استفراغ «شدید» در ۷ روز اخیر",
			"Days with severe vomiting in the last 7 days."),
		unit(cond("preg.vomiting_streak", TypeInt, nil, "روزهای پشت سر هم استفراغ تا امروز (یا دیروز)",
			"Consecutive days of vomiting up to today (or yesterday)."), "day"),
		Def{Name: "preg.cant_keep_fluids", Type: TypeBool, Group: g, Pending: "no log field yet",
			Description: desc("«مایعات را نگه نمی\u200cدارم» ثبت شده", "«I can't keep fluids down» was logged.")},
		cond("preg.severe_symptoms_7", TypeInt, nil, "تعداد علائم «شدید» در ۷ روز اخیر",
			"Severe symptoms logged in the last 7 days."),
		unit(cond("preg.days_since_weight", TypeInt, nil, "روزهای گذشته از آخرین ثبت وزن (یا از شروع بارداری)",
			"Days since the last weight log (or since the pregnancy started)."), "day"),
		cond("preg.week_day", TypeInt, nil, "روز هفتهٔ بارداری (۰ تا ۶)", "Day of the pregnancy week (0–6)."),
		cond("preg.week_started_today", TypeBool, nil, "هفتهٔ بارداری امروز شروع شده", "The pregnancy week started today."),
		unit(cond("preg.next_visit_in_days", TypeInt, nil, "روزهای مانده تا ویزیت بعدی",
			"Days until the next visit."), "day"),
		cond("preg.fetal_movement", TypeEnum, []string{FetalReduced, FetalNone, FetalLowSession},
			"حرکات جنین: کم (ثبت دستی)، بدون حرکت، یا شمارش کمتر از هدف (فقط وقتی برقرار است)",
			"Fetal movement: reduced or none (manual log), or a kick count short of the target (present only when it holds)."),
		cond("preg.bp_last_category", TypeEnum, []string{BPNormal, BPHigh, BPSevere},
			"بالاترین ردهٔ فشار خون بارداری در ۷ روز اخیر (ثبت هفتگی و علائم حیاتی)",
			"The highest pregnancy BP class of the last 7 days (weekly logs and vitals)."),
		cond("preg.bp_high_level", TypeEnum, []string{LevelFollowUp, LevelUrgent},
			"سطح فشار ≥ ۱۴۰/۹۰ از تنظیم بالینی pregnancy.bp.high.urgent",
			"The level of a ≥ 140/90 reading from the clinical config pregnancy.bp.high.urgent."),
		cond("preg.gdm_last", TypeEnum, []string{GDMInTarget, GDMAbove},
			"قند خون ۷ روز اخیر نسبت به اهداف دیابت بارداری", "Blood sugar of the last 7 days against the GDM targets."),
		cond("preg.contractions_511", TypeBool, nil, "انقباض\u200cهای جلسهٔ جاری به الگوی ۵-۱-۱ رسیده (فقط وقتی برقرار است)",
			"The running contraction session met 5-1-1 (present only when it holds)."),
		cond("preg.critical_symptom", TypeSet, CriticalSymptoms,
			"علائم بحرانی ثبت\u200cشده در ۷ روز اخیر، از تازه\u200cترین روز (فقط وقتی هست)",
			"Critical symptoms logged in the last 7 days, from the newest such day (present only when any)."),
		cond("preg.depression_severe", TypeBool, nil, "احساس افسردگی «شدید» در ثبت هفتگی ۷ روز اخیر",
			"Severe depression feelings in a weekly log of the last 7 days."),

		n("preg_vomit_days", "متغیر متن: روزهای استفراغ", "Text variable: vomiting days."),
		n("preg_vomit_severe", "متغیر متن: روزهای استفراغ شدید", "Text variable: severe vomiting days."),
		n("preg_severe_count", "متغیر متن: تعداد علائم شدید", "Text variable: severe symptom count."),
		n("preg_week", "متغیر متن: هفتهٔ بارداری", "Text variable: the pregnancy week."),
		v("preg_week_on", "متغیر متن: روز شروع هفتهٔ بارداری", "Text variable: the first day of the pregnancy week."),
		v("preg_basis", "متغیر متن: مبنای سن بارداری (lmp / ultrasound / manual)", "Text variable: the dating basis."),
		v("preg_visit_on", "متغیر متن: تاریخ ویزیت بعدی", "Text variable: the next visit's date."),
		v("preg_fetal_status", "متغیر متن: وضعیت حرکات ثبت\u200cشده", "Text variable: the logged movement status."),
		v("preg_fetal_on", "متغیر متن: روز ثبت حرکات", "Text variable: the day of the movement log."),
		n("preg_kick_count", "متغیر متن: حرکت\u200cهای شمرده\u200cشده", "Text variable: the kicks counted."),
		n("preg_kick_target", "متغیر متن: هدف شمارش", "Text variable: the kick target."),
		n("preg_kick_minutes", "متغیر متن: پنجرهٔ شمارش (دقیقه)", "Text variable: the kick window in minutes."),
		v("preg_kick_at", "متغیر متن: زمان شروع شمارش", "Text variable: when the kick session started."),
		v("preg_bp_sys", "متغیر متن: فشار سیستولیک", "Text variable: systolic pressure."),
		v("preg_bp_dia", "متغیر متن: فشار دیاستولیک", "Text variable: diastolic pressure."),
		v("preg_bp_on", "متغیر متن: روز ثبت فشار", "Text variable: the day of the BP reading."),
		v("preg_fasting", "متغیر متن: قند ناشتا", "Text variable: fasting sugar."),
		v("preg_post_meal", "متغیر متن: قند بعد از غذا", "Text variable: after-meal sugar."),
		v("preg_gdm_on", "متغیر متن: روز ثبت قند", "Text variable: the day of the sugar reading."),
		n("preg_labor_count", "متغیر متن: تعداد انقباض\u200cها", "Text variable: the contractions timed."),
		n("preg_labor_minutes", "متغیر متن: بازهٔ انقباض\u200cها (دقیقه)", "Text variable: the minutes they span."),
		v("preg_labor_interval", "متغیر متن: فاصلهٔ میانگین", "Text variable: the average interval."),
		v("preg_labor_duration", "متغیر متن: مدت میانگین", "Text variable: the average duration."),
		v("preg_labor_at", "متغیر متن: شروع جلسهٔ انقباض", "Text variable: when the contraction session started."),
		Def{Name: "preg_symptoms", Type: TypeSet, Enum: CriticalSymptoms, Group: g, Variable: true,
			Description: desc("متغیر متن: علائم بحرانی", "Text variable: the critical symptoms.")},
		v("preg_symptom_on", "متغیر متن: روز ثبت علامت", "Text variable: the day of the symptom log."),
		v("preg_mood_on", "متغیر متن: روز ثبت هفتگی", "Text variable: the day of the weekly log."),
	)
	RegisterGroup(g, func(ctx context.Context, f *Facts) (Values, error) {
		src, ok := f.src.Pregnancy.(PregnancyAlertSource)
		if !ok || src == nil {
			return nil, nil
		}
		in, err := src.PregnancyAlertInput(ctx, f.today)
		if err != nil || in == nil {
			return nil, err
		}
		return PregnancyAlertValues(in), nil
	})
}

// detector configs: the detectors run with their widest setting and the rule conditions hold the thresholds.
func detect(rule string, window int, params map[string]any, f pregnancyalerts.Facts) []pregnancyalerts.Hit {
	if params == nil {
		params = map[string]any{}
	}
	return pregnancyalerts.Detect(rule, pregnancyalerts.Config{Enabled: true, Level: "info", WindowDays: window,
		Params: params}, f)
}

func anyList(s []string) []any {
	out := make([]any, len(s))
	for i, x := range s {
		out[i] = x
	}
	return out
}

func varOf(h pregnancyalerts.Hit, name string) string {
	for _, kv := range h.Vars {
		if kv[0] == name {
			return kv[1]
		}
	}
	return ""
}

// num is a detector's ASCII number var (missing → noValue).
func num(h pregnancyalerts.Hit, name string) string {
	s := varOf(h, name)
	if s == "" || strings.ContainsRune(s, 0) {
		return noValue
	}
	return s
}

// dayOfDedupe is the leading Y-m-d of a reading hit's dedupe key ("2026-09-23:150/95").
func dayOfDedupe(h pregnancyalerts.Hit, today civildate.Date) string {
	if d, _, ok := strings.Cut(h.Dedupe, ":"); ok {
		if _, err := civildate.Parse(d); err == nil {
			return d
		}
	}
	return today.String()
}

const minuteLayout = "2006-01-02 15:04"

// PregnancyAlertValues are the facts and evidence variables of one input (pure; the parity table and the safety
// scenarios call it directly).
func PregnancyAlertValues(in *PregnancyAlertInput) Values {
	f := in.Facts
	today := f.Today
	out := Values{
		"preg.week_day":           float64(f.WeekDay),
		"preg.week_started_today": f.WeekDay == 0,
		"preg_week":               float64(f.Week),
		"preg_week_on":            today.AddDays(-f.WeekDay).String(),
		"preg_basis":              f.Source,
		"preg.cant_keep_fluids":   false, // no log field yet: the rule is seeded, the fact stays false
	}

	// Vomiting: days in the 7-day window, severe days, and the v2 streak (ending today, or yesterday when today is not
	// logged yet).
	days, severe := 0, 0
	for d := today.AddDays(-6); !d.After(today); d = d.AddDays(1) {
		if sev, ok := f.Days[d]["vomiting"]; ok {
			days++
			if sev == "severe" {
				severe++
			}
		}
	}
	streak := 0
	if hits := detect("vomiting_streak", 7, map[string]any{"min_streak_days": 1}, f); len(hits) > 0 {
		streak, _ = strconv.Atoi(varOf(hits[0], "days"))
	}
	out["preg.vomiting_days_7"], out["preg.vomiting_severe_7"], out["preg.vomiting_streak"] =
		float64(days), float64(severe), float64(streak)
	out["preg_vomit_days"], out["preg_vomit_severe"] = float64(days), float64(severe)

	// Severe symptoms of the 7 days (the v2 list).
	count := 0
	if hits := detect("severe_symptom_count", 7, map[string]any{"min_count": 1, "symptoms": anyList(SevereSymptoms)}, f); len(hits) > 0 {
		count, _ = strconv.Atoi(varOf(hits[0], "count"))
	}
	out["preg.severe_symptoms_7"], out["preg_severe_count"] = float64(count), float64(count)

	// Weight: days since the newest weight (since the pregnancy start when none was ever logged).
	from := in.LastWeight
	if from.IsZero() {
		from = in.Start
	}
	if !from.IsZero() {
		out["preg.days_since_weight"] = float64(max(0, from.DiffDays(today)))
	}

	// Next visit (today or later).
	if !in.NextVisit.IsZero() {
		day := civildate.InTehran(in.NextVisit)
		if !day.Before(today) {
			out["preg.next_visit_in_days"] = float64(today.DiffDays(day))
			out["preg_visit_on"] = day.String()
		}
	}

	// Critical symptoms logged in the 7-day window (spotting only up to week 12, or when severe): a log saved for an
	// earlier day still raises the alert, as the v1 save-time rule did. The evidence is the newest day with one.
	crit, critOn := map[string]bool{}, ""
	for _, h := range detect("critical_symptom", pregnancyalerts.FactsWindowDays, map[string]any{
		"symptoms": anyList(CriticalSymptoms), "spotting_until_week": SpottingUntilWeek}, f) {
		_, on, _ := strings.Cut(h.Dedupe, "@")
		if on > critOn {
			critOn, crit = on, map[string]bool{}
		}
		if on == critOn {
			crit[varOf(h, "symptom")] = true
		}
	}
	if len(crit) > 0 {
		set := setOf(crit)
		out["preg.critical_symptom"], out["preg_symptoms"], out["preg_symptom_on"] = set, set, critOn
	}

	// Fetal movement: a reduced kick session leads, then no movement, then reduced (manual logs of today).
	fetal, fetalOn := "", ""
	for _, h := range detect("fetal_movement", 1, map[string]any{"statuses": anyList([]string{FetalReduced, FetalNone})}, f) {
		if h.Variant == pregnancyalerts.VariantKicks {
			if fetal != FetalLowSession {
				fetal = FetalLowSession
				out["preg_kick_count"], _ = strconv.ParseFloat(varOf(h, "count"), 64)
				out["preg_kick_target"], _ = strconv.ParseFloat(varOf(h, "target"), 64)
				out["preg_kick_minutes"], _ = strconv.ParseFloat(varOf(h, "minutes"), 64)
				out["preg_kick_at"] = kickStart(f, h)
			}
			continue
		}
		st := varOf(h, "status")
		if fetal == "" || (fetal == FetalReduced && st == FetalNone) {
			fetal, fetalOn = st, today.String()
		}
	}
	if fetal != "" {
		out["preg.fetal_movement"] = fetal
		if fetal != FetalLowSession {
			out["preg_fetal_status"], out["preg_fetal_on"] = fetal, fetalOn
		}
	}

	// BP: the highest class of the window's readings; its values are the evidence (the newest of that class).
	bpHits := detect("bp_high", 7, nil, f)
	cls, worst := "", -1
	for i, h := range bpHits {
		c := BPHigh
		if h.Variant == pregnancyalerts.VariantSevere {
			c = BPSevere
		}
		if worst < 0 || bpRank(c) > bpRank(cls) || (bpRank(c) == bpRank(cls) && dayOfDedupe(h, today) >= dayOfDedupe(bpHits[worst], today)) {
			cls, worst = c, i
		}
	}
	switch {
	case worst >= 0:
		h := bpHits[worst]
		out["preg.bp_last_category"] = cls
		out["preg_bp_sys"], out["preg_bp_dia"], out["preg_bp_on"] = num(h, "systolic"), num(h, "diastolic"), dayOfDedupe(h, today)
	case hasBP(f):
		out["preg.bp_last_category"] = BPNormal
	}
	out["preg.bp_high_level"] = LevelFollowUp
	if pregnancyalerts.HighBPLevel(today) == "urgent" {
		out["preg.bp_high_level"] = LevelUrgent
	}

	// GDM: any value of the window at or above its target (clinical config pregnancy.gdm.targets).
	if hits := detect("sugar_high", 7, nil, f); len(hits) > 0 {
		h := hits[len(hits)-1]
		out["preg.gdm_last"] = GDMAbove
		out["preg_fasting"], out["preg_post_meal"], out["preg_gdm_on"] = num(h, "fasting"), num(h, "post_meal"), dayOfDedupe(h, today)
	} else if hasGlucose(f) {
		out["preg.gdm_last"] = GDMInTarget
	}

	// 5-1-1 of the running contraction session (once per session: the session start keys the evidence).
	if hits := detect("contractions_511", 1, nil, f); len(hits) > 0 {
		h := hits[0]
		out["preg.contractions_511"] = true
		out["preg_labor_count"], _ = strconv.ParseFloat(varOf(h, "count"), 64)
		out["preg_labor_minutes"], _ = strconv.ParseFloat(varOf(h, "minutes"), 64)
		out["preg_labor_interval"], out["preg_labor_duration"] = varOf(h, "interval"), varOf(h, "duration")
		if len(f.Contractions) > 0 {
			out["preg_labor_at"] = f.Contractions[0].Start.In(civildate.Tehran).Format(minuteLayout)
		}
	}

	// Severe depression feelings in a weekly log of the window (the v1 weekly rule, kept as a follow-up).
	out["preg.depression_severe"] = false
	for _, w := range f.Weekly {
		if w.LogDate.Before(today.AddDays(-6)) || w.LogDate.After(today) {
			continue
		}
		if w.HasDepressionFeelings.Valid && w.HasDepressionFeelings.Bool && w.DepressionSeverity.String == "severe" {
			out["preg.depression_severe"] = true
			if on, _ := out["preg_mood_on"].(string); w.LogDate.String() > on {
				out["preg_mood_on"] = w.LogDate.String()
			}
		}
	}
	return out
}

func bpRank(c string) int {
	switch c {
	case BPSevere:
		return 2
	case BPHigh:
		return 1
	}
	return 0
}

// kickStart is the start of the kick session of a hit (Dedupe "kick:<id>").
func kickStart(f pregnancyalerts.Facts, h pregnancyalerts.Hit) string {
	id := strings.TrimPrefix(h.Dedupe, "kick:")
	for _, k := range f.Kicks {
		if strconv.FormatUint(k.ID, 10) == id {
			return k.Start.In(civildate.Tehran).Format(minuteLayout)
		}
	}
	return h.On.String()
}

func inWeek(f pregnancyalerts.Facts, d civildate.Date) bool {
	return !d.Before(f.Today.AddDays(-6)) && !d.After(f.Today)
}

func hasBP(f pregnancyalerts.Facts) bool {
	for _, w := range f.Weekly {
		if inWeek(f, w.LogDate) && (w.SystolicPressure.Valid || w.DiastolicPressure.Valid) {
			return true
		}
	}
	return slices.ContainsFunc(f.Readings, func(r pregnancyalerts.Reading) bool {
		return inWeek(f, r.Date) && (r.Systolic > 0 || r.Diastolic > 0)
	})
}

func hasGlucose(f pregnancyalerts.Facts) bool {
	for _, w := range f.Weekly {
		if inWeek(f, w.LogDate) && (w.FastingBloodSugar.Valid || w.PostMealBloodSugar.Valid) {
			return true
		}
	}
	return slices.ContainsFunc(f.Readings, func(r pregnancyalerts.Reading) bool {
		return inWeek(f, r.Date) && r.MgDl > 0 && pregnancyalerts.GDMTarget(r.Slot, r.Date) > 0
	})
}

// keep the clinical registry import explicit: the BP / GDM / kick thresholds are read by the detectors through it.
var _ = clinicalconfig.KeyPregnancyBPHigh
