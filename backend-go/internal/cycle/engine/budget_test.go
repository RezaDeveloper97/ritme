package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Locked performance budgets (README §8, T-M9-22). Every engine benchmark is defined here from one table:
//
//   - time: the ceiling of README §8 where it names one (Prepare, Resolve, Range31, the §2.35 encode), else a
//     T-M9-22 ceiling (FutureCycles, TodayUI, TodayPayload = the sum of its parts). A benchmark run with b.N ≥
//     budgetMinN fails when its mean ns/op is above the ceiling, so `go test -bench . -benchtime 2000x` (the task
//     verify) and the CI bench job enforce it. Measured medians are 10–40× below these ceilings even under load
//     (docs/engine-v6/perf-baseline.md), so a failure is a real regression, not noise.
//   - allocs: the count measured when the budgets were locked (deterministic, checked by TestBudgets_Allocs in every
//     `go test` run). README §8 allows Resolve ≤ 3 and Range31 ≤ 40; the lock is the tighter measured value, and
//     bench/gate holds the same numbers against bench/baseline.txt.
//
// Raising a lock is a decision: change it here, re-record the engine rows of bench/baseline.txt and say why in the
// task report.

// budgetMinN is the smallest b.N whose mean time is checked (warm-up runs and -benchtime 1x smoke runs are not).
const budgetMinN = 1000

type budget struct {
	name    string
	fixture string
	ceiling time.Duration
	allocs  float64
	// op returns the measured operation, built once outside the timer.
	op func() func()
}

func budgets() []budget {
	return []budget{
		{"Prepare", "60 periods, a skip, declared values", 300 * time.Microsecond, 8, func() func() {
			in, p := benchInput(), DefaultParams()
			return func() { _ = Prepare(in, p) }
		}},
		{"Prepare_Markers", "60 periods + 400 BBT readings + 60 LH tests", 300 * time.Microsecond, 10, func() func() {
			in, p := markerBenchInput(), DefaultParams()
			return func() { _ = Prepare(in, p) }
		}},
		{"Resolve", "24 periods, 330 targets around today", 3 * time.Microsecond, 0, func() func() {
			s, cur := bench24()
			i := 0
			return func() {
				_ = s.Resolve(cur-300+Day(i%330), cur)
				i++
			}
		}},
		{"Resolve_Hormonal", "24 periods on the pill, 330 targets", 3 * time.Microsecond, 0, func() func() {
			_, cur := bench24()
			in := withPeriods(bench24Periods())
			in.Hormonal = true
			s := prep(in)
			i := 0
			return func() {
				_ = s.Resolve(cur-300+Day(i%330), cur)
				i++
			}
		}},
		{"Range31", "24 periods, 31 days around today (crosses into future_cycles)", 150 * time.Microsecond, 1,
			func() func() {
				s, cur := bench24()
				today := cur + 20
				return func() { _ = s.Range(today-15, today+15, today) }
			}},
		{"FutureCycles", "24 periods, 6 future cycles", 5 * time.Microsecond, 1, func() func() {
			s, cur := bench24()
			return func() { _ = s.FutureCycles(cur + 20) }
		}},
		{"TodayUI", "one status", time.Microsecond, 0, func() func() {
			cur := day(julyStart)
			s := prep(withPeriods(periodsEnding(cur, 5, 27, 29, 30)))
			st := s.Resolve(cur+17, cur+17)
			return func() { _ = s.TodayUI(st) }
		}},
		{"EncodeDay", "the §2.35 today payload (status + 6 future cycles + today_ui), reused buffer",
			20 * time.Microsecond, 0, func() func() {
				s := prep(withPeriods(periodsEnding(day("2026-07-01"), 5, 27, 29, 30)))
				st, ui := s.Today(day("2026-07-18"))
				buf := make([]byte, 0, 4096)
				return func() { buf = AppendJSON(buf[:0], &st, &ui) }
			}},
		{"TodayPayload", "Resolve + FutureCycles + TodayUI + encode after Prepare, 24 periods",
			30 * time.Microsecond, 1, func() func() {
				s, cur := bench24()
				buf := make([]byte, 0, 4096)
				return func() {
					st, ui := s.Today(cur + 20)
					buf = AppendJSON(buf[:0], &st, &ui)
				}
			}},
	}
}

func budgetNamed(tb testing.TB, name string) budget {
	tb.Helper()
	for _, b := range budgets() {
		if b.name == name {
			return b
		}
	}
	tb.Fatalf("no budget %q", name)
	return budget{}
}

// runBudget is the body of every engine benchmark: the op in a b.N loop, then the time ceiling.
func runBudget(b *testing.B, name string) {
	bg := budgetNamed(b, name)
	op := bg.op()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		op()
	}
	b.StopTimer()
	if b.N >= budgetMinN {
		if per := b.Elapsed() / time.Duration(b.N); per > bg.ceiling {
			b.Errorf("%s (%s): %v/op is over the locked ceiling %v (README §8, T-M9-22)", name, bg.fixture, per,
				bg.ceiling)
		}
	}
}

// TestBudgets_Allocs holds every benchmark's allocation lock (deterministic, so it runs in every go test).
func TestBudgets_Allocs(t *testing.T) {
	seen := map[string]bool{}
	for _, bg := range budgets() {
		require.False(t, seen[bg.name], "duplicate budget %s", bg.name)
		seen[bg.name] = true
		op := bg.op()
		op() // warm the op's own lazy state (buffers) like the benchmark's first iterations
		assert.LessOrEqual(t, testing.AllocsPerRun(200, op), bg.allocs, "%s (%s)", bg.name, bg.fixture)
	}
	// README §8 ceilings the locks must stay inside.
	assert.LessOrEqual(t, budgetNamed(t, "Resolve").allocs, 3.0)
	assert.LessOrEqual(t, budgetNamed(t, "Range31").allocs, 40.0)
}

// markerBenchInput is benchInput plus 400 BBT readings (a confirmed shift every 9th cycle) and an LH test per cycle.
func markerBenchInput() Input {
	in := benchInput()
	for i, p := range in.Periods {
		if i%9 == 0 && i+1 < len(in.Periods) {
			in.BBT = append(in.BBT, shiftAt(in.Periods[i+1].Start-13)...)
		}
		in.LH = append(in.LH, lhOn(p.Start+12, i%2 == 0))
	}
	for len(in.BBT) < 400 {
		in.BBT = append(in.BBT, BBTReading{Date: in.Periods[0].Start + Day(len(in.BBT)%20), Centi: 3650})
	}
	return in
}

// bench24Periods are bench24's 24 closed periods (cycles 26–30, last start 2026-07-01).
func bench24Periods() []Period {
	lengths := make([]int, 23)
	for i := range lengths {
		lengths[i] = 26 + i%5
	}
	return periodsEnding(day("2026-07-01"), 5, lengths...)
}

func BenchmarkPrepare(b *testing.B)          { runBudget(b, "Prepare") }
func BenchmarkPrepare_Markers(b *testing.B)  { runBudget(b, "Prepare_Markers") }
func BenchmarkResolve(b *testing.B)          { runBudget(b, "Resolve") }
func BenchmarkResolve_Hormonal(b *testing.B) { runBudget(b, "Resolve_Hormonal") }
func BenchmarkRange31(b *testing.B)          { runBudget(b, "Range31") }
func BenchmarkFutureCycles(b *testing.B)     { runBudget(b, "FutureCycles") }
func BenchmarkTodayUI(b *testing.B)          { runBudget(b, "TodayUI") }
func BenchmarkEncodeDay(b *testing.B)        { runBudget(b, "EncodeDay") }
func BenchmarkTodayPayload(b *testing.B)     { runBudget(b, "TodayPayload") }
