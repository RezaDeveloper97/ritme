// Package safety is the safety & clinical content dashboard of the admin panel (bloom B-N9-10, D-90): aggregated
// urgent-signal counts (no user ids), the unified clinical review queue over the existing review states, the urgent-path
// health checks and the nightly urgent-scenario test run.
package safety

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/postpartum"
	"github.com/ritme/backend-go/internal/postpartum/guide"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/pregnancy/tools"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
	"github.com/ritme/backend-go/internal/vitals"
)

// ReducedKicks reports a kick session that ended without reaching tools.KickTarget within the counting window
// (nbl_Log_Kick: «اگر در ۲ ساعت کمتر از ۱۰ حرکت حس کردی …»). The signal count uses the same rule in SQL.
func ReducedKicks(kicks int, counted time.Duration) bool {
	return kicks < tools.KickTarget && counted >= tools.KickWindowMinutes*time.Minute
}

// Scenario is one urgent-path scenario: a synthetic input run through the real classifier of its domain with the
// clinical config in force on day, and the verdict it must give.
type Scenario struct {
	Name string
	Run  func(day civildate.Date) bool
}

func contractions(n int, every, last time.Duration) []labor.Contraction {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	out := make([]labor.Contraction, n)
	for i := range out {
		s := start.Add(time.Duration(i) * every)
		out[i] = labor.Contraction{Start: s, End: s.Add(last)}
	}
	return out
}

// epdsTotal is a full check with item 10 = 0 and the other nine items summing to total (≤ 27) on day.
func epdsTotal(day civildate.Date, total int) postpartum.Result {
	answers := map[string]int{postpartum.SelfHarmItem: 0}
	left := total
	for _, it := range postpartum.Items {
		if it.Code == postpartum.SelfHarmItem {
			continue
		}
		v := min(left, postpartum.ItemMax)
		answers[it.Code], left = v, left-v
	}
	return postpartum.ScoreOn(postpartum.KindFull, answers, day)
}

// hasCall reports whether a safety message carries a call action to number.
func hasCall(msg any, number string) bool {
	m, ok := msg.(*jsonx.OrderedMap)
	if !ok {
		return false
	}
	raw, _ := m.Get("actions")
	actions, _ := raw.([]*jsonx.OrderedMap)
	for _, a := range actions {
		if n, _ := a.Get("number"); n == number {
			return true
		}
	}
	return false
}

// redFlag runs the real recovery alert builder on r on day and reports whether it raised key as urgent with 115.
func redFlag(day civildate.Date, r postpartum.Recovery, key string) bool {
	for _, loc := range []string{"fa", "en"} {
		alerts, err := postpartum.RecoveryAlerts(context.Background(), guide.NewCopy(nil, loc, loc), r, day)
		if err != nil || len(alerts) == 0 {
			return false
		}
		a, ok := alerts[0].(*jsonx.OrderedMap)
		if !ok {
			return false
		}
		k, _ := a.Get("key")
		level, _ := a.Get("level")
		title, _ := a.Get("title")
		if k != key || level != postpartum.LevelUrgent || title == "" {
			return false
		}
		act, _ := a.Get("action")
		am, _ := act.(*jsonx.OrderedMap)
		if am == nil {
			return false
		}
		if n, _ := am.Get("number"); n != postpartum.EmergencyNumber {
			return false
		}
	}
	return true
}

func epds(kind string, score int, q10 int) postpartum.Result {
	answers := map[string]int{}
	for _, it := range postpartum.ItemsOf(kind) {
		answers[it.Code] = score
	}
	if kind == postpartum.KindFull {
		answers[postpartum.SelfHarmItem] = q10
	}
	return postpartum.ScoreOn(kind, answers, civildate.Date{})
}

func bp(day civildate.Date, sys, dia float64) string {
	return vitals.AlertRule(vitals.Reading{Type: vitals.TypeBP, Date: day, Systolic: sys, Diastolic: dia})
}

func glucose(day civildate.Date, mgdl float64) string {
	return vitals.AlertRule(vitals.Reading{Type: vitals.TypeGlucose, Date: day, MgDl: mgdl})
}

// pregnant are the alert-engine facts of a pregnancy at weeks+0 on day, evaluated at 12:00.
func pregnant(day civildate.Date, weeks int) pregnancyalerts.Facts {
	return pregnancyalerts.Facts{Today: day, Week: weeks + 1, TotalDays: weeks * 7, Days: map[civildate.Date]pregnancyalerts.Day{},
		Now: day.TehranMidnight().Add(12 * time.Hour)}
}

// rule is a live, enabled rule row of the alert engine.
func rule(level string, window int, params map[string]any) pregnancyalerts.Config {
	return pregnancyalerts.Config{Enabled: true, Level: level, WindowDays: window, Params: params}
}

// kickSession: kicks counted in a session started at 09:00 on day, ended after `after`, with the target-th kick at
// targetAt (0 = not reached).
func kickSession(day civildate.Date, kicks int, after, targetAt time.Duration) pregnancyalerts.Kick {
	start := day.TehranMidnight().Add(9 * time.Hour)
	k := pregnancyalerts.Kick{ID: 1, Start: start, End: start.Add(after), Count: kicks}
	if targetAt > 0 {
		k.TargetAt = start.Add(targetAt)
	}
	return k
}

// kickAlert runs the real fetal_movement rule on one session at week 28+0 and reports whether it raised the urgent
// alert.
func kickAlert(day civildate.Date, k pregnancyalerts.Kick) bool {
	f := pregnant(day, pregnancyalerts.KickParamsOn(day).FromWeek)
	f.Kicks = []pregnancyalerts.Kick{k}
	for _, h := range pregnancyalerts.Detect("fetal_movement", rule("urgent", 1, map[string]any{}), f) {
		if h.Level == "urgent" {
			return true
		}
	}
	return false
}

// urgentHit reports whether rule raised an urgent hit on f.
func urgentHit(name string, f pregnancyalerts.Facts) bool {
	for _, h := range pregnancyalerts.Detect(name, rule("follow_up", 7, map[string]any{}), f) {
		if h.Level == "urgent" {
			return true
		}
	}
	return false
}

// Scenarios are the urgent paths the nightly run checks (the «آخرین تست خودکار سناریوها» row).
var Scenarios = []Scenario{
	{"bp_crisis_systolic", func(d civildate.Date) bool {
		t := vitals.ThresholdsOn(d)
		return bp(d, t.BPCrisisSystolic+1, 70) == vitals.RuleBPCrisis
	}},
	{"bp_crisis_diastolic", func(d civildate.Date) bool {
		t := vitals.ThresholdsOn(d)
		return bp(d, 110, t.BPCrisisDiastolic+1) == vitals.RuleBPCrisis
	}},
	{"bp_below_crisis_not_urgent", func(d civildate.Date) bool {
		t := vitals.ThresholdsOn(d)
		return bp(d, t.BPCrisisSystolic-1, t.BPCrisisDiastolic-1) == ""
	}},
	{"glucose_urgent_low", func(d civildate.Date) bool {
		return glucose(d, vitals.ThresholdsOn(d).GlucoseUrgentBelow-1) == vitals.RuleGlucoseLow
	}},
	{"glucose_at_threshold_not_urgent", func(d civildate.Date) bool {
		return glucose(d, vitals.ThresholdsOn(d).GlucoseUrgentBelow) == ""
	}},
	{"epds_q10_positive_urgent", func(civildate.Date) bool {
		r := epds(postpartum.KindFull, 0, 1)
		return r.Urgent && slices.Contains(r.Reasons, postpartum.ReasonSelfHarm)
	}},
	// T-M9-06 / H-01: the score path reads epds.urgent_total (13 today; 31 = off — then no total without item 10
	// is urgent, the top one being 27).
	{"epds_high_total_urgent", func(d civildate.Date) bool {
		at := postpartum.CutoffsOn(d).UrgentTotal
		top := len(postpartum.Items[:9]) * postpartum.ItemMax
		if at > top {
			return !epdsTotal(d, top).Urgent
		}
		return epdsTotal(d, at).Urgent && !epdsTotal(d, at-1).Urgent
	}},
	{"epds_q10_low_total", func(d civildate.Date) bool {
		r := postpartum.ScoreOn(postpartum.KindFull, map[string]int{"q1": 1, "q2": 1, "q3": 1, "q10": 1}, d)
		if r.Total != 4 || !r.Urgent {
			return false
		}
		msg, err := postpartum.SafetyJSON(context.Background(), guide.NewCopy(nil, "fa", "fa"), r)
		return err == nil && hasCall(msg, postpartum.EmergencyNumber) && hasCall(msg, postpartum.SocialEmergencyNumber)
	}},
	{"epds_low_not_urgent", func(civildate.Date) bool { return !epds(postpartum.KindFull, 0, 0).Urgent }},
	{"epds_short_never_urgent", func(civildate.Date) bool { return !epds(postpartum.KindShort, 3, 0).Urgent }},
	{"contractions_511_met", func(d civildate.Date) bool {
		return labor.FiveOneOne(contractions(14, 5*time.Minute, time.Minute), labor.ParamsOn(d)).Met
	}},
	{"contractions_sparse_not_met", func(d civildate.Date) bool {
		return !labor.FiveOneOne(contractions(8, 12*time.Minute, time.Minute), labor.ParamsOn(d)).Met
	}},
	{"postpartum_heavy_bleeding_alert", func(civildate.Date) bool {
		return slices.Contains(postpartum.Recovery{LochiaAmount: "heavy"}.Alerts(), guide.AlertHeavyBleeding)
	}},
	{"postpartum_light_bleeding_no_alert", func(civildate.Date) bool {
		return len(postpartum.Recovery{LochiaAmount: "light"}.Alerts()) == 0
	}},
	// T-M9-06: the postpartum red flags are urgent with 115 (fever from postpartum.redflags.fever_c).
	{"postpartum_fever", func(d civildate.Date) bool {
		fever := postpartum.FeverOn(d)
		below := fever - 0.1
		return redFlag(d, postpartum.Recovery{TemperatureC: &fever}, guide.AlertFever) &&
			len(postpartum.Recovery{TemperatureC: &below}.AlertsOn(d)) == 0
	}},
	{"postpartum_pad_per_hour", func(d civildate.Date) bool {
		yes := true
		return redFlag(d, postpartum.Recovery{PadSoakedWithinHour: &yes}, guide.AlertPadPerHour)
	}},
	{"postpartum_breathless", func(d civildate.Date) bool {
		yes := true
		return redFlag(d, postpartum.Recovery{Breathless: &yes}, guide.AlertBreathless)
	}},
	// T-M9-05: the kick paths run the real alert rule (pregnancyalerts fetal_movement), not a counter.
	{"reduced_kicks", func(d civildate.Date) bool {
		p := pregnancyalerts.KickParamsOn(d)
		return kickAlert(d, kickSession(d, p.Target-1, p.Window, 0))
	}},
	{"enough_kicks_not_reduced", func(d civildate.Date) bool {
		p := pregnancyalerts.KickParamsOn(d)
		return !kickAlert(d, kickSession(d, p.Target, 30*time.Minute, 30*time.Minute))
	}},
	{"pregnancy_bp_severe", func(d civildate.Date) bool {
		f := pregnant(d, 30)
		sys := int32(pregnancyalerts.BPTierThreshold("severe", "systolic", d)) //nolint:gosec // mmHg
		f.Weekly = []store.PregnancyWeeklyLog{{LogDate: d,
			SystolicPressure: sql.NullInt32{Int32: sys, Valid: true}, DiastolicPressure: sql.NullInt32{Int32: 80, Valid: true}}}
		return urgentHit("bp_high", f)
	}},
	{"pregnancy_vitals_bp", func(d civildate.Date) bool {
		r := vitals.Reading{Type: vitals.TypeBP, Date: d, Systolic: 165, Diastolic: 112, Pregnant: true}
		in, ok := vitals.PregnancyReading(r)
		f := pregnant(d, 30)
		f.Readings = []pregnancyalerts.Reading{in}
		return ok && r.Class() == vitals.ClassCrisis && r.Urgent() && vitals.AlertRule(r) == "" && urgentHit("bp_high", f)
	}},
	{"gdm_two_hour", func(d civildate.Date) bool {
		hits := func(mgdl float64) int {
			in, _ := vitals.PregnancyReading(vitals.Reading{Type: vitals.TypeGlucose, Date: d, MgDl: mgdl,
				Context: vitals.ContextAfterMeal, Pregnant: true})
			f := pregnant(d, 30)
			f.Readings = []pregnancyalerts.Reading{in}
			return len(pregnancyalerts.Detect("sugar_high", rule("follow_up", 7, map[string]any{}), f))
		}
		at := pregnancyalerts.GDMTarget(pregnancyalerts.GlucoseTwoHour, d)
		return hits(at) == 1 && hits(at-1) == 0
	}},
	{"emergency_rule_without_copy", func(d civildate.Date) bool {
		p := pregnancyalerts.KickParamsOn(d)
		f := pregnant(d, p.FromWeek)
		f.Kicks = []pregnancyalerts.Kick{kickSession(d, 2, time.Hour, 0)}
		for _, loc := range []string{"fa", "en"} {
			level, title, contact := pregnancyalerts.FiresWithoutCopy("fetal_movement", f, v2.Lang{Locale: loc, Default: loc})
			if level != "urgent" || title == "" || contact == "" {
				return false
			}
		}
		g := pregnant(d, 30)
		g.Days[d] = pregnancyalerts.Day{"fluid_leakage": "mild"}
		level, _, _ := pregnancyalerts.FiresWithoutCopy("critical_symptom", g, v2.Lang{Locale: "en", Default: "en"})
		return level == "urgent"
	}},
}

// Run is one scenario run.
type Run struct {
	RanAt  time.Time
	Passed int
	Total  int
	Failed []string
}

// RunScenarios runs every scenario with the clinical config in force on day.
func RunScenarios(day civildate.Date, now time.Time) Run {
	r := Run{RanAt: now, Total: len(Scenarios)}
	for _, s := range Scenarios {
		if ok := safeRun(s, day); ok {
			r.Passed++
		} else {
			r.Failed = append(r.Failed, s.Name)
		}
	}
	return r
}

func safeRun(s Scenario, day civildate.Date) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return s.Run(day)
}

var (
	lastMu  sync.RWMutex
	lastRun *Run
)

// Last is the latest in-process run (nil before the first).
func Last() *Run {
	lastMu.RLock()
	defer lastMu.RUnlock()
	return lastRun
}

func record(r Run) {
	lastMu.Lock()
	lastRun = &r
	lastMu.Unlock()
}

// Background job (T-M9-11, docs/engine-v6/jobs.md): the nightly run at 03:00 Asia/Tehran on the low queue. The result
// stays in the memory of the process that ran it until T-M10-24 persists it; the API also runs the set once at
// start-up (RunAndRecord) so its own dashboard row is filled.
const (
	JobNightly  = "safety:nightly_scenarios"
	NightlyCron = "0 3 * * *"
)

// RegisterJobs puts the nightly job on q.
func RegisterJobs(q *queue.Queue, logger *slog.Logger, now func() time.Time) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobNightly, Priority: queue.Low, MaxTries: 1, Timeout: time.Minute,
	}, NightlyJob(logger, now), NightlyCron, queue.Unique(time.Hour))
}

// NightlyJob is the safety:nightly_scenarios handler. Idempotent: the scenarios are pure functions of the clinical
// config in force today; a rerun replaces the recorded run with an equal one.
func NightlyJob(logger *slog.Logger, now func() time.Time) queue.Handler {
	return func(ctx context.Context, _ []byte) error {
		RunAndRecord(ctx, logger, now())
		return nil
	}
}

// RunAndRecord runs the scenarios now and keeps the result for the dashboard (failures are logged by name only).
func RunAndRecord(ctx context.Context, logger *slog.Logger, now time.Time) Run {
	r := RunScenarios(civildate.InTehran(now), now)
	record(r)
	if len(r.Failed) > 0 {
		logger.ErrorContext(ctx, "safety: urgent scenarios failed", slog.Any("failed", r.Failed))
	}
	return r
}
