// Package safety is the safety & clinical content dashboard of the admin panel (bloom B-N9-10, D-90): aggregated
// urgent-signal counts (no user ids), the unified clinical review queue over the existing review states, the urgent-path
// health checks and the nightly urgent-scenario test run.
package safety

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum"
	"github.com/ritme/backend-go/internal/postpartum/guide"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/tools"
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
	{"epds_high_total_urgent", func(civildate.Date) bool { return epds(postpartum.KindFull, 3, 0).Urgent }},
	{"epds_low_not_urgent", func(civildate.Date) bool { return !epds(postpartum.KindFull, 0, 0).Urgent }},
	{"epds_short_never_urgent", func(civildate.Date) bool { return !epds(postpartum.KindShort, 3, 0).Urgent }},
	{"contractions_511_met", func(civildate.Date) bool {
		return labor.FiveOneOne(contractions(14, 5*time.Minute, time.Minute), labor.DefaultParams()).Met
	}},
	{"contractions_sparse_not_met", func(civildate.Date) bool {
		return !labor.FiveOneOne(contractions(8, 12*time.Minute, time.Minute), labor.DefaultParams()).Met
	}},
	{"postpartum_heavy_bleeding_alert", func(civildate.Date) bool {
		return slices.Contains(postpartum.Recovery{LochiaAmount: "heavy"}.Alerts(), guide.AlertHeavyBleeding)
	}},
	{"postpartum_light_bleeding_no_alert", func(civildate.Date) bool {
		return len(postpartum.Recovery{LochiaAmount: "light"}.Alerts()) == 0
	}},
	{"reduced_kicks", func(civildate.Date) bool { return ReducedKicks(tools.KickTarget-4, 2*time.Hour) }},
	{"enough_kicks_not_reduced", func(civildate.Date) bool { return !ReducedKicks(tools.KickTarget, 30*time.Minute) }},
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

// NightlyHour is when the in-process run repeats (Asia/Tehran).
const NightlyHour = 3

func nextNightly(now time.Time) time.Time {
	t := now.In(civildate.Tehran)
	next := time.Date(t.Year(), t.Month(), t.Day(), NightlyHour, 0, 0, 0, civildate.Tehran)
	if !next.After(t) {
		next = next.AddDate(0, 0, 1)
	}
	return next
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

// Loop runs the scenarios at start-up and every night at NightlyHour until ctx ends (the CI workflow
// .github/workflows/safety-scenarios-nightly.yml runs the same set as TestUrgentScenarios).
func Loop(ctx context.Context, logger *slog.Logger, now func() time.Time) {
	RunAndRecord(ctx, logger, now())
	for {
		t := time.NewTimer(time.Until(nextNightly(now())))
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			RunAndRecord(ctx, logger, now())
		}
	}
}
