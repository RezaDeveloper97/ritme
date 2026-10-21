package engine

import (
	"bytes"
	"encoding/json"
	"flag"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "rewrite testdata/*.json goldens from the encoder")

const specGolden = "testdata/spec_2_35_today.json"

// specExample is the §2.35 worked example exactly as SPEC.md prints it (comments removed). Its future_cycles list is
// partial (k = 2, three keys); E-05 makes every item k = 1..6 with eight keys.
const specExample = `{
  "date": "2026-07-18",
  "cycle_day": 18,
  "main_phase": "luteal",
  "subphase": "early_luteal",
  "fertility_level": "low",
  "days_to_ovulation": -2,
  "days_to_period": 12,
  "days_late": 0,
  "anchors": {
    "current_period_start": "2026-07-01", "current_period_start_source": "user_logged",
    "current_period_end": "2026-07-05",   "current_period_end_source": "user_logged",
    "current_period_end_is_confirmed": true,
    "predicted_next_period_start": "2026-07-30",
    "estimated_ovulation_date": "2026-07-16",
    "ovulation_source": "calendar"
  },
  "metrics": { "effective_cycle_length": 29, "effective_period_length": 5,
               "cycle_variability": 3, "cycle_regularity": "regular" },
  "clinical_flags": { "frequency": "normal", "duration": "normal", "absent": false },
  "future_cycles": [ { "k": 2, "period_start": "2026-08-28", "ovulation": "2026-08-14" } ],
  "suggest_pregnancy_test": false,
  "confidence": "high", "confidence_reasons": [], "data_quality": "good",
  "resolution_source": "prediction", "is_predicted": true,
  "requires_user_input": false, "warnings": []
}`

// specToday is the §2.35 example as the today payload: periods 07-01..07-05 confirmed after cycles 27, 29, 30
// (ECL 29, variability 3, ≥ 2 closed periods of 5), target = today = 07-18.
func specToday() []byte {
	s := prep(withPeriods(periodsEnding(day("2026-07-01"), 5, 27, 29, 30)))
	st, ui := s.Today(day("2026-07-18"))
	return AppendJSON(nil, &st, &ui)
}

// TestEncode_SpecGolden fixes the bytes: the encoder output equals the committed golden (stored indented; compacting
// it gives the exact encoder bytes). Run with -update to rewrite it.
func TestEncode_SpecGolden(t *testing.T) {
	got := specToday()
	require.True(t, json.Valid(got))
	if *updateGolden {
		var ind bytes.Buffer
		require.NoError(t, json.Indent(&ind, got, "", "  "))
		ind.WriteByte('\n')
		require.NoError(t, os.MkdirAll(filepath.Dir(specGolden), 0o755))
		require.NoError(t, os.WriteFile(specGolden, ind.Bytes(), 0o600))
	}
	golden, err := os.ReadFile(specGolden)
	require.NoError(t, err)
	var want bytes.Buffer
	require.NoError(t, json.Compact(&want, golden))
	assert.Equal(t, want.String(), string(got))
}

// TestEncode_MatchesSpecExample checks every §2.35 field against the spec text: same values, same key order at
// every level (the additions limits / future_k / today_ui come after the last spec key), and the spec's partial k = 2
// item is a subset of the encoder's k = 2 item.
func TestEncode_MatchesSpecExample(t *testing.T) {
	got := specToday()
	var spec, out map[string]any
	require.NoError(t, json.Unmarshal([]byte(specExample), &spec))
	require.NoError(t, json.Unmarshal(got, &out))

	for k, v := range spec {
		if k == "future_cycles" {
			continue
		}
		assert.Equal(t, v, out[k], k)
	}
	fc, ok := out["future_cycles"].([]any)
	require.True(t, ok)
	require.Len(t, fc, 6)
	for i, item := range fc {
		assert.InDelta(t, float64(i+1), item.(map[string]any)["k"], 0, "k = 1..6")
	}
	for _, item := range spec["future_cycles"].([]any) {
		want := item.(map[string]any)
		k := int(want["k"].(float64))
		have := fc[k-1].(map[string]any)
		for key, v := range want {
			assert.Equal(t, v, have[key], "future_cycles[k=%d].%s", k, key)
		}
	}

	specKeys, outKeys := objectKeys(t, []byte(specExample)), objectKeys(t, got)
	assert.Equal(t, specKeys, outKeys[:len(specKeys)], "spec keys keep their positions")
	assert.Equal(t, []string{"limits", "future_k", "today_ui"}, outKeys[len(specKeys):], "documented additions")
	for _, nested := range []string{"anchors", "metrics", "clinical_flags"} {
		assert.Equal(t, objectKeys(t, rawField(t, []byte(specExample), nested)), objectKeys(t, rawField(t, got, nested)),
			nested)
	}
}

// objectKeys are the top-level keys of a JSON object in document order.
func objectKeys(t *testing.T, data []byte) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	require.NoError(t, err)
	require.Equal(t, json.Delim('{'), tok)
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		require.NoError(t, err)
		keys = append(keys, tok.(string))
		var skip json.RawMessage
		require.NoError(t, dec.Decode(&skip))
	}
	return keys
}

func rawField(t *testing.T, data []byte, key string) []byte {
	t.Helper()
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))
	return m[key]
}

// TestEncode_Nulls covers the null / [] rules: Opt none, NoDay, SuppressionNone and BannerNone are null; an empty
// future_cycles is [] and a nil one (Resolve, Range) omits the key; skip anchors have a null end with source
// "unknown"; future targets carry future_k.
func TestEncode_Nulls(t *testing.T) {
	cur := day(julyStart)

	st := prep(regular(cur, 28, 5)).Resolve(cur+10, cur+10)
	b := string(AppendJSON(nil, &st, nil))
	assert.NotContains(t, b, `"future_cycles"`)
	assert.NotContains(t, b, `"today_ui"`)
	assert.Contains(t, b, `"limits":{"fertility_suppressed":false,"suppression_reason":null,"engine_suspended":false,`+
		`"prompt_pregnancy_mode":false},"future_k":0}`)

	// Pregnancy (E-25): no countdowns, empty future list, suspended, null card.
	in := regular(cur, 28, 5)
	in.Mode = ModePregnancy
	st, ui := prep(in).Today(cur + 10)
	b = string(AppendJSON(nil, &st, &ui))
	for _, want := range []string{
		`"cycle_day":null`, `"days_to_ovulation":null`, `"days_to_period":null`, `"days_late":null`,
		`"predicted_next_period_start":null`, `"estimated_ovulation_date":null,"ovulation_source":"unknown"`,
		`"future_cycles":[]`, `"suppression_reason":"pregnancy","engine_suspended":true`,
		`"banner":null`, `"prediction_card":null`, `"warnings":[]`, `"confidence_reasons":[]`,
	} {
		assert.Contains(t, b, want)
	}

	// No data at all: null anchors, null clinical flags and variability.
	st, ui = prep(none()).Today(cur)
	b = string(AppendJSON(nil, &st, &ui))
	for _, want := range []string{
		`"current_period_start":null,"current_period_start_source":"unknown"`, `"cycle_variability":null`,
		`"clinical_flags":{"frequency":null,"duration":null,"absent":null}`, `"warnings":["insufficient_anchor_data"]`,
	} {
		assert.Contains(t, b, want)
	}

	// Skip anchor (E-15): no period, so no period end.
	in = withPeriods(periodsEnding(cur, 5, 28, 28))
	in.Skips = []Day{cur + 28}
	st = prep(in).Resolve(cur+40, cur+40)
	require.Equal(t, StartPredictedReference, st.Anchors.CurrentPeriodStartSource)
	b = string(AppendJSON(nil, &st, nil))
	assert.Contains(t, b, `"current_period_end":null,"current_period_end_source":"unknown"`)

	// A future target (E-03) and the pill's future cycles (E-13: fertile and ovulation null).
	st = prep(regular(cur, 28, 5)).Resolve(cur+30, cur+10)
	require.Equal(t, 1, st.FutureK)
	assert.Contains(t, string(AppendJSON(nil, &st, nil)), `"future_k":1}`)
	st, ui = prep(pill()).Today(cur + 10)
	b = string(AppendJSON(nil, &st, &ui))
	assert.Contains(t, b, `"fertile_start":null,"fertile_end":null,"ovulation":null,"ovulation_source":"unknown"`)
	assert.Contains(t, b, `"suppression_reason":"hormonal_contraception"`)
	assert.Contains(t, b, `"fertile":null,"ovulation":null`)
}

// TestEncode_EnumOrder: warnings and reasons follow the enum order whatever order they were added in.
func TestEncode_EnumOrder(t *testing.T) {
	var st Status
	st.Date = day("2026-07-18")
	st.Warnings.Add(WarnContradictoryAnchorData)
	st.Warnings.Add(WarnPeriodEndMissing)
	st.Warnings.Add(WarnLowHistory)
	st.ConfidenceReasons.Add(ReasonHormonalContraceptionActive)
	st.ConfidenceReasons.Add(ReasonLowHistory)
	b := string(AppendJSON(nil, &st, nil))
	assert.Contains(t, b, `"warnings":["period_end_missing","low_history","contradictory_anchor_data"]`)
	assert.Contains(t, b, `"confidence_reasons":["low_history","hormonal_contraception_active"]`)
}

func TestEncode_DayOutOfRangeIsNull(t *testing.T) {
	assert.Equal(t, "null", string(appendDay(nil, NoDay)))
	assert.Equal(t, `x"2026-07-18"`, string(appendDay([]byte("x"), day("2026-07-18"))))
	assert.Equal(t, "xnull", string(appendDay([]byte("x"), Day(-800000))))
}

// randomDay is a valid day or (1 in 5) NoDay.
func randomDay(r *rand.Rand) Day {
	if r.IntN(5) == 0 {
		return NoDay
	}
	return day("2020-01-01") + Day(r.IntN(4000))
}

func randomOptInt(r *rand.Rand) Opt[int] {
	if r.IntN(4) == 0 {
		return None[int]()
	}
	return Some(r.IntN(200) - 100)
}

// randomStatus fills every field with valid values, independent of the engine's rules.
func randomStatus(r *rand.Rand) (Status, Opt[TodayUI]) {
	st := Status{
		Date: randomDay(r), CycleDay: randomOptInt(r),
		MainPhase: MainPhase(r.IntN(6)), Subphase: Subphase(r.IntN(15)), FertilityLevel: FertilityLevel(r.IntN(5)),
		DaysToOvulation: randomOptInt(r), DaysToPeriod: randomOptInt(r), DaysLate: randomOptInt(r),
		Anchors: Anchors{
			CurrentPeriodStart: randomDay(r), CurrentPeriodStartSource: StartSource(r.IntN(4)),
			CurrentPeriodEnd: randomDay(r), CurrentPeriodEndSource: EndSource(r.IntN(3)),
			CurrentPeriodEndIsConfirmed: r.IntN(2) == 0,
			PredictedNextPeriodStart:    randomDay(r), EstimatedOvulationDate: randomDay(r),
			OvulationSource: OvulationSource(r.IntN(4)),
		},
		Metrics: Metrics{EffectiveCycleLength: r.IntN(60), EffectivePeriodLength: r.IntN(12),
			CycleVariability: randomOptInt(r), CycleRegularity: CycleRegularity(r.IntN(3))},
		SuggestPregnancyTest: r.IntN(2) == 0,
		Confidence:           Confidence(r.IntN(4)),
		ConfidenceReasons:    ReasonSet(r.Uint32() & (1<<reasonCount - 1)),
		DataQuality:          DataQuality(r.IntN(4)),
		ResolutionSource:     ResolutionSource(r.IntN(6)),
		RequiresUserInput:    r.IntN(2) == 0,
		Warnings:             WarningSet(r.Uint32() & (1<<warningCount - 1)),
		Limits: Limits{FertilitySuppressed: r.IntN(2) == 0, SuppressionReason: SuppressionReason(r.IntN(4)),
			EngineSuspended: r.IntN(2) == 0, PromptPregnancyMode: r.IntN(2) == 0},
		FutureK: r.IntN(7),
	}
	if r.IntN(3) > 0 {
		st.ClinicalFlags.Frequency = Some(Frequency(r.IntN(3)))
	}
	if r.IntN(3) > 0 {
		st.ClinicalFlags.Duration = Some(Duration(r.IntN(2)))
	}
	if r.IntN(3) > 0 {
		st.ClinicalFlags.Absent = Some(r.IntN(2) == 0)
	}
	switch r.IntN(3) {
	case 0: // nil: the key is omitted
	case 1:
		st.FutureCycles = []FutureCycle{}
	default:
		st.FutureCycles = make([]FutureCycle, 1+r.IntN(6))
		for i := range st.FutureCycles {
			st.FutureCycles[i] = FutureCycle{K: i + 1, PeriodStart: randomDay(r), PeriodEnd: randomDay(r),
				FertileStart: randomDay(r), FertileEnd: randomDay(r), Ovulation: randomDay(r),
				OvulationSource: OvulationSource(r.IntN(4)), Confidence: Confidence(r.IntN(4))}
		}
	}
	if r.IntN(3) == 0 {
		return st, None[TodayUI]()
	}
	ui := TodayUI{RingAction: RingAction(r.IntN(3)), Banner: Banner(r.IntN(4)), ShowPregnancyTestButton: r.IntN(2) == 0}
	if r.IntN(4) > 0 {
		c := PredictionCard{Period: DayRange{randomDay(r), randomDay(r)}, PMS: DayRange{randomDay(r), randomDay(r)},
			CycleLength: r.IntN(60), CycleRegularity: CycleRegularity(r.IntN(3))}
		if r.IntN(2) == 0 {
			c.Fertile = Some(DayRange{randomDay(r), randomDay(r)})
		}
		if r.IntN(2) == 0 {
			c.Ovulation = Some(OvulationMark{Date: randomDay(r), Source: OvulationSource(r.IntN(4))})
		}
		ui.PredictionCard = Some(c)
	}
	return st, Some(ui)
}

// TestEncode_RoundTrip: decode(encode(x)) == x over 1 000 statuses — 500 from the engine (random histories, random
// targets, half of them today payloads with future_cycles and today_ui) and 500 hand-random ones covering every
// enum value and null.
func TestEncode_RoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(21, 2026))
	today := day("2026-10-21")
	check := func(i int, st Status, ui Opt[TodayUI]) {
		var uip *TodayUI
		if u, ok := ui.Get(); ok {
			uip = &u
		}
		b := AppendJSON(nil, &st, uip)
		require.True(t, json.Valid(b), "status %d: %s", i, b)
		got, gotUI, err := DecodeJSON(b)
		require.NoError(t, err, "status %d: %s", i, b)
		require.Equal(t, st, got, "status %d: %s", i, b)
		require.Equal(t, ui, gotUI, "status %d", i)
	}
	for i := range 500 {
		s := prep(randomInput(r, today))
		if i%2 == 0 {
			st, ui := s.Today(today)
			check(i, st, Some(ui))
			continue
		}
		check(i, s.Resolve(today-200+Day(r.IntN(400)), today), None[TodayUI]())
	}
	for i := range 500 {
		st, ui := randomStatus(r)
		check(500+i, st, ui)
	}
}

func TestDecode_Errors(t *testing.T) {
	good := specToday()
	_, _, err := DecodeJSON(good)
	require.NoError(t, err)

	for name, b := range map[string]string{
		"unknown key":  `{"date":"2026-07-18","extra":1}`,
		"bad date":     string(bytes.Replace(good, []byte(`"2026-07-18"`), []byte(`"2026-02-30"`), 1)),
		"short date":   string(bytes.Replace(good, []byte(`"2026-07-18"`), []byte(`"2026-7-18"`), 1)),
		"digit date":   string(bytes.Replace(good, []byte(`"2026-07-18"`), []byte(`"2026-0x-18"`), 1)),
		"bad enum":     string(bytes.Replace(good, []byte(`"early_luteal"`), []byte(`"lutealish"`), 1)),
		"empty enum":   string(bytes.Replace(good, []byte(`"suppression_reason":null`), []byte(`"suppression_reason":""`), 1)),
		"bad warning":  string(bytes.Replace(good, []byte(`"warnings":[]`), []byte(`"warnings":["nope"]`), 1)),
		"is_predicted": string(bytes.Replace(good, []byte(`"is_predicted":true`), []byte(`"is_predicted":false`), 1)),
		"not json":     `{`,
	} {
		_, ui, err := DecodeJSON([]byte(b))
		assert.Error(t, err, name)
		assert.False(t, ui.OK(), name)
	}
}

// TestEncode_NoAllocsWithBuffer: with a reused buffer the encoder allocates nothing.
func TestEncode_NoAllocsWithBuffer(t *testing.T) {
	s := prep(withPeriods(periodsEnding(day("2026-07-01"), 5, 27, 29, 30)))
	st, ui := s.Today(day("2026-07-18"))
	buf := make([]byte, 0, 4096)
	assert.Zero(t, testing.AllocsPerRun(200, func() { buf = AppendJSON(buf[:0], &st, &ui) }))
}
