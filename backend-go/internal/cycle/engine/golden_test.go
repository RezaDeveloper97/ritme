package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Engine goldens (T-M9-22): one synthetic persona per testdata/personas/<name>.json (an Input and its today), one
// golden per persona in <name>.golden.jsonl:
//
//	line 1      the today payload (State.Today + AppendJSON: today's status, future_cycles and today_ui)
//	line 2      Range day 1 (today − 182) as a full §2.35 object
//	lines 3…366 Range days 2…365: only the keys whose value differs from the previous day (date always)
//
// Regenerate with `go test ./internal/cycle/engine -run TestGoldens -update` and review the diff. The §2.36
// (21/28/35) and §2.35 anchors are also asserted by hand below, so a wrong engine cannot pass by regenerating.

const (
	personaDir     = "testdata/personas"
	goldenDaysBack = 182
	goldenDays     = 365
)

// persona is the JSON form of one persona: dates are ISO strings, absent values null or omitted.
type persona struct {
	Name  string       `json:"name"`
	About string       `json:"about"`
	Today string       `json:"today"`
	Input personaInput `json:"input"`
}

type personaInput struct {
	Periods []struct {
		Start  string  `json:"start"`
		End    *string `json:"end"`
		Source string  `json:"source"`
	} `json:"periods"`
	Skips    []string `json:"skips"`
	Declared struct {
		LastPeriodStart *string `json:"last_period_start"`
		CycleLength     *int    `json:"cycle_length"`
		PeriodLength    *int    `json:"period_length"`
		Manual          bool    `json:"manual"`
	} `json:"declared"`
	BirthDate     *string `json:"birth_date"`
	TeenBandOnly  bool    `json:"teen_band_only"`
	Mode          string  `json:"mode"`
	Hormonal      bool    `json:"hormonal"`
	HormonalFrom  *string `json:"hormonal_from"`
	DeliveryDate  *string `json:"delivery_date"`
	Breastfeeding bool    `json:"breastfeeding"`
	LH            []struct {
		Date     string `json:"date"`
		Positive bool   `json:"positive"`
	} `json:"lh"`
	// BBT is a list of daily series: Centi[i] is the reading of From + i (0 = not measured).
	BBT []struct {
		From  string  `json:"from"`
		Centi []int32 `json:"centi"`
	} `json:"bbt"`
	PregnancyTests []struct {
		Date     string `json:"date"`
		Positive bool   `json:"positive"`
	} `json:"pregnancy_tests"`
}

func optDay(s *string) Opt[Day] {
	if s == nil {
		return None[Day]()
	}
	return Some(day(*s))
}

// input converts the persona to an engine Input.
func (p *persona) input(t *testing.T) Input {
	w := &p.Input
	in := none()
	for _, r := range w.Periods {
		pr := Period{Start: day(r.Start), End: optDay(r.End).Or(NoDay)}
		switch r.Source {
		case "", "user_logged":
		case "bleeding_log":
			pr.Source = PeriodBleedingLog
		default:
			t.Fatalf("%s: unknown period source %q", p.Name, r.Source)
		}
		in.Periods = append(in.Periods, pr)
	}
	for _, s := range w.Skips {
		in.Skips = append(in.Skips, day(s))
	}
	in.Declared = Declared{LastPeriodStart: optDay(w.Declared.LastPeriodStart), CycleLength: optInt(w.Declared.CycleLength),
		PeriodLength: optInt(w.Declared.PeriodLength), Manual: w.Declared.Manual}
	in.BirthDate, in.TeenBandOnly = optDay(w.BirthDate), w.TeenBandOnly
	if w.Mode != "" {
		m, ok := ParseMode(w.Mode)
		require.True(t, ok, "%s: mode %q", p.Name, w.Mode)
		in.Mode = m
	}
	in.Hormonal, in.DeliveryDate, in.Breastfeeding = w.Hormonal, optDay(w.DeliveryDate), w.Breastfeeding
	in.HormonalFrom = optDay(w.HormonalFrom)
	for _, x := range w.LH {
		in.LH = append(in.LH, LHTest{Date: day(x.Date), Positive: x.Positive})
	}
	for _, series := range w.BBT {
		from := day(series.From)
		for i, c := range series.Centi {
			if c != 0 {
				in.BBT = append(in.BBT, BBTReading{Date: from + Day(i), Centi: c})
			}
		}
	}
	for _, x := range w.PregnancyTests {
		in.PregnancyTests = append(in.PregnancyTests, PregnancyTest{Date: day(x.Date), Positive: x.Positive})
	}
	return in
}

func loadPersonas(t *testing.T) []persona {
	paths, err := filepath.Glob(filepath.Join(personaDir, "*.json"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(paths), 20, "personas")
	out := make([]persona, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path) //nolint:gosec // test fixture
		require.NoError(t, err)
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var p persona
		require.NoError(t, dec.Decode(&p), path)
		require.Equal(t, strings.TrimSuffix(filepath.Base(path), ".json"), p.Name, path)
		out = append(out, p)
	}
	return out
}

// goldenOf is the golden text of a persona (see the file comment), and the statuses behind it.
func goldenOf(t *testing.T, s *State, today Day) ([]byte, []Status) {
	var out bytes.Buffer
	st, ui := s.Today(today)
	out.Write(AppendJSON(nil, &st, &ui))
	out.WriteByte('\n')

	days := s.Range(today-goldenDaysBack, today-goldenDaysBack+goldenDays-1, today)
	require.Len(t, days, goldenDays)
	var prev map[string]json.RawMessage
	var keys []string
	for i := range days {
		b := AppendJSON(nil, &days[i], nil)
		var cur map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(b, &cur))
		if prev == nil {
			keys = objectKeys(t, b)
			out.Write(b)
		} else {
			out.WriteByte('{')
			first := true
			for _, k := range keys {
				if k != "date" && bytes.Equal(cur[k], prev[k]) {
					continue
				}
				if !first {
					out.WriteByte(',')
				}
				first = false
				fmt.Fprintf(&out, "%q:%s", k, cur[k])
			}
			out.WriteByte('}')
		}
		out.WriteByte('\n')
		prev = cur
	}
	return out.Bytes(), days
}

// TestGoldens checks every persona against its golden (or rewrites it with -update), runs the invariant checks on
// every golden day, and asserts the §2.36 and §2.35 anchors by hand.
func TestGoldens(t *testing.T) {
	byName := map[string]*State{}
	todays := map[string]Day{}
	for _, p := range loadPersonas(t) {
		t.Run(p.Name, func(t *testing.T) {
			in := p.input(t)
			today := day(p.Today)
			s := prep(in)
			byName[p.Name], todays[p.Name] = s, today
			got, days := goldenOf(t, s, today)
			for i := range days {
				require.Empty(t, violations(s, &days[i], today), "%s", days[i].Date)
			}
			require.Empty(t, futureViolations(s, s.FutureCycles(today), today))

			path := filepath.Join(personaDir, p.Name+".golden.jsonl")
			if *updateGolden {
				require.NoError(t, os.WriteFile(path, got, 0o600))
			}
			want, err := os.ReadFile(path) //nolint:gosec // test fixture
			require.NoError(t, err, "missing golden: run go test ./internal/cycle/engine -run TestGoldens -update")
			if !bytes.Equal(want, got) {
				wl, gl := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
				for i := range min(len(wl), len(gl)) {
					if wl[i] != gl[i] {
						t.Fatalf("%s line %d differs (regenerate with -update and review):\nwant %s\ngot  %s", path, i+1,
							wl[i], gl[i])
					}
				}
				t.Fatalf("%s: %d lines, want %d", path, len(gl), len(wl))
			}
		})
	}

	t.Run("§2.36 by hand", func(t *testing.T) {
		for _, c := range []struct {
			persona string
			length  int
			table   map[int]Subphase // cycle day → subphase; main phase only where the table gives a range
			ov      int
			fertile [2]int // display window, cycle days
		}{
			{"regular_21", 21, map[int]Subphase{6: SubphaseHighFertility, 7: SubphaseHighFertility,
				8: SubphaseOvulationLikely, 9: SubphasePostOvulation, 10: SubphaseEarlyLuteal, 12: SubphaseEarlyLuteal,
				13: SubphaseMidLuteal, 15: SubphaseMidLuteal, 16: SubphaseLateLuteal, 18: SubphaseLateLuteal,
				19: SubphasePMSPossible, 21: SubphasePMSPossible}, 8, [2]int{6, 8}},
			{"regular_28", 28, map[int]Subphase{6: SubphaseEarlyFollicular, 7: SubphaseMidFollicular,
				8: SubphaseLateFollicularTransition, 9: SubphaseLateFollicularTransition, 10: SubphaseFertileRising,
				12: SubphaseFertileRising, 13: SubphaseHighFertility, 14: SubphaseHighFertility,
				15: SubphaseOvulationLikely, 16: SubphasePostOvulation, 17: SubphaseEarlyLuteal, 19: SubphaseEarlyLuteal,
				20: SubphaseMidLuteal, 22: SubphaseMidLuteal, 23: SubphaseLateLuteal, 25: SubphaseLateLuteal,
				26: SubphasePMSPossible, 28: SubphasePMSPossible}, 15, [2]int{10, 15}},
			{"regular_35", 35, map[int]Subphase{17: SubphaseFertileRising, 19: SubphaseFertileRising,
				20: SubphaseHighFertility, 21: SubphaseHighFertility, 22: SubphaseOvulationLikely,
				23: SubphasePostOvulation, 24: SubphaseEarlyLuteal, 26: SubphaseEarlyLuteal, 27: SubphaseMidLuteal,
				29: SubphaseMidLuteal, 30: SubphaseLateLuteal, 32: SubphaseLateLuteal, 33: SubphasePMSPossible,
				35: SubphasePMSPossible}, 22, [2]int{17, 22}},
		} {
			s, today := byName[c.persona], todays[c.persona]
			require.NotNil(t, s, c.persona)
			start := s.Periods()[len(s.Periods())-1].Start
			require.LessOrEqual(t, start, today, c.persona)
			for cd := 1; cd <= c.length; cd++ {
				st := s.Resolve(start+Day(cd-1), today)
				require.Equal(t, Some(cd), st.CycleDay, "%s day %d", c.persona, cd)
				assert.Equal(t, start+Day(c.ov-1), st.Anchors.EstimatedOvulationDate, "%s ovulation", c.persona)
				assert.Equal(t, start+Day(c.length), st.Anchors.PredictedNextPeriodStart, "%s next", c.persona)
				switch want, ok := c.table[cd]; {
				case cd <= 5:
					assert.Equal(t, SubphaseMenstrual, st.Subphase, "%s day %d", c.persona, cd)
				case ok:
					assert.Equal(t, want, st.Subphase, "%s day %d", c.persona, cd)
				case c.persona == "regular_35" && cd <= 16:
					assert.Equal(t, MainPhaseFollicular, st.MainPhase, "%s day %d", c.persona, cd)
				}
				fertile := cd >= c.fertile[0] && cd <= c.fertile[1]
				assert.Equal(t, fertile, st.MainPhase == MainPhaseFertile, "%s day %d fertile", c.persona, cd)
			}
			// period_expected from day L + 1, seen on that day.
			d := start + Day(c.length)
			st := s.Resolve(d, d)
			assert.Equal(t, MainPhasePeriodExpected, st.MainPhase, "%s day %d", c.persona, c.length+1)
		}
	})

	t.Run("§2.35 by hand", func(t *testing.T) {
		s, today := byName["spec_2_35"], todays["spec_2_35"]
		require.NotNil(t, s)
		require.Equal(t, day("2026-07-18"), today)
		st, ui := s.Today(today)
		assert.Equal(t, Some(18), st.CycleDay)
		assert.Equal(t, SubphaseEarlyLuteal, st.Subphase)
		assert.Equal(t, MainPhaseLuteal, st.MainPhase)
		assert.Equal(t, FertilityLow, st.FertilityLevel)
		assert.Equal(t, Some(-2), st.DaysToOvulation)
		assert.Equal(t, Some(12), st.DaysToPeriod)
		assert.Equal(t, Some(0), st.DaysLate)
		assert.Equal(t, Anchors{
			CurrentPeriodStart: day("2026-07-01"), CurrentPeriodStartSource: StartUserLogged,
			CurrentPeriodEnd: day("2026-07-05"), CurrentPeriodEndSource: EndUserLogged, CurrentPeriodEndIsConfirmed: true,
			PredictedNextPeriodStart: day("2026-07-30"), EstimatedOvulationDate: day("2026-07-16"),
			OvulationSource: OvulationCalendar,
		}, st.Anchors)
		assert.Equal(t, Metrics{EffectiveCycleLength: 29, EffectivePeriodLength: 5, CycleVariability: Some(3),
			CycleRegularity: RegularityRegular}, st.Metrics)
		assert.Equal(t, ClinicalFlags{Frequency: Some(FrequencyNormal), Duration: Some(DurationNormal),
			Absent: Some(false)}, st.ClinicalFlags)
		assert.Equal(t, ConfidenceHigh, st.Confidence)
		assert.Equal(t, DataQualityGood, st.DataQuality)
		assert.Equal(t, ResolutionPrediction, st.ResolutionSource)
		assert.Zero(t, st.ConfidenceReasons)
		assert.Zero(t, st.Warnings)
		assert.False(t, st.RequiresUserInput)
		assert.False(t, st.SuggestPregnancyTest)
		require.Len(t, st.FutureCycles, 6)
		k2 := st.FutureCycles[1]
		assert.Equal(t, day("2026-08-28"), k2.PeriodStart)
		assert.Equal(t, day("2026-08-14"), k2.Ovulation)
		assert.Equal(t, RingEdit, ui.RingAction)
		assert.Equal(t, BannerNone, ui.Banner)
	})

	t.Run("persona coverage", func(t *testing.T) {
		names := make([]string, 0, len(byName))
		for n := range byName {
			names = append(names, n)
		}
		slices.Sort(names)
		for _, n := range []string{"regular_28", "regular_21", "regular_35", "irregular", "teen_16", "age_46", "pill",
			"copper_iud", "postpartum_breastfeeding", "pregnant", "onboarding_fresh", "onboarding_stale", "skip",
			"bbt_strict_pass", "bbt_strict_fail", "lh_bbt_conflict", "open_past_hard_cap", "late_8", "late_15",
			"contradictory"} {
			assert.Contains(t, names, n)
		}
	})
}
