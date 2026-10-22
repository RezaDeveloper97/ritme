package clinicalconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func mustDef(t *testing.T, key string) *Def {
	t.Helper()
	d, ok := Lookup(key)
	require.True(t, ok, key)
	return d
}

// with returns the v1 rows of key with row i's column set to v.
func with(t *testing.T, key string, i int, col string, v any) []Row {
	t.Helper()
	rows := mustDef(t, key).DefaultRows()
	rows[i][col] = v
	return rows
}

func TestRowsKeys_Schema(t *testing.T) {
	for _, def := range Defs {
		if !def.IsRows() {
			assert.Nil(t, def.Table, def.Key)
			assert.Equal(t, KindNumbers, def.KindName(), def.Key)
			continue
		}
		require.NotNil(t, def.Table, def.Key)
		assert.Empty(t, def.Fields, def.Key)
		assert.Equal(t, "rows", def.Validate(Value{}), "a rows key never takes named numbers")
		for _, vals := range def.Table.Default {
			assert.Len(t, vals, len(def.Table.Columns), "%s default row width", def.Key)
		}
	}
}

func TestRowsValidation_PerKey(t *testing.T) {
	cases := []struct {
		name string
		key  string
		rows []Row
		want string
	}{
		// pregnancy.redating: ascending bands, only the last open, bounds.
		{"redating unsorted", KeyPregnancyRedating, with(t, KeyPregnancyRedating, 1, "ga_until_days", 60.0), "sort:1:ga_until_days"},
		{"redating duplicate band", KeyPregnancyRedating, with(t, KeyPregnancyRedating, 1, "ga_until_days", 62.0), "sort:1:ga_until_days"},
		{"redating open in the middle", KeyPregnancyRedating, with(t, KeyPregnancyRedating, 2, "ga_until_days", nil), "open:2:ga_until_days"},
		{"redating limit too big", KeyPregnancyRedating, with(t, KeyPregnancyRedating, 0, "max_diff_days", 43.0), "range:0:max_diff_days"},
		{"redating fractional", KeyPregnancyRedating, with(t, KeyPregnancyRedating, 0, "max_diff_days", 5.5), "integer:0:max_diff_days"},
		{"redating limit null", KeyPregnancyRedating, with(t, KeyPregnancyRedating, 0, "max_diff_days", nil), "null:0:max_diff_days"},
		{"redating string", KeyPregnancyRedating, with(t, KeyPregnancyRedating, 0, "max_diff_days", "5"), "type:0:max_diff_days"},
		{"redating no rows", KeyPregnancyRedating, []Row{}, "count"},

		// pregnancy.iom: contiguous BMI bands, lo < hi.
		{"iom gap", KeyPregnancyIOM, with(t, KeyPregnancyIOM, 1, "bmi_from", 19.0), "gap:1"},
		{"iom overlap", KeyPregnancyIOM, with(t, KeyPregnancyIOM, 0, "bmi_to", 20.0), "overlap:1"},
		{"iom open band not last", KeyPregnancyIOM, with(t, KeyPregnancyIOM, 1, "bmi_to", nil), "open:1:bmi_to"},
		{"iom total order", KeyPregnancyIOM, with(t, KeyPregnancyIOM, 2, "total_lo", 11.5), "order:2:total_lo"},
		{"iom weekly order", KeyPregnancyIOM, with(t, KeyPregnancyIOM, 3, "weekly_hi", 0.1), "order:3:weekly_lo"},
		{"iom weekly bound", KeyPregnancyIOM, with(t, KeyPregnancyIOM, 3, "weekly_hi", 2.5), "range:3:weekly_hi"},
		{"iom empty band", KeyPregnancyIOM, with(t, KeyPregnancyIOM, 0, "bmi_to", 5.0), "range:0:bmi_to"},

		// screening.uspstf: enum, overlapping ages of one type and kind, a null kind overlaps every kind.
		{"screening bad kind", KeyScreeningUSPSTF, with(t, KeyScreeningUSPSTF, 1, "last_test_kind", "pap"), "enum:1:last_test_kind"},
		{"screening overlap", KeyScreeningUSPSTF, with(t, KeyScreeningUSPSTF, 0, "age_max", 30.0), "overlap:1"},
		{"screening any-kind overlap", KeyScreeningUSPSTF, with(t, KeyScreeningUSPSTF, 2, "last_test_kind", nil), "overlap:2"},
		{"screening open bp overlaps", KeyScreeningUSPSTF, with(t, KeyScreeningUSPSTF, 6, "age_min", 39.0), "overlap:6"},
		{"screening age bound", KeyScreeningUSPSTF, with(t, KeyScreeningUSPSTF, 4, "age_min", 130.0), "range:4:age_min"},
		{"screening empty type", KeyScreeningUSPSTF, with(t, KeyScreeningUSPSTF, 4, "type_key", ""), "type:4:type_key"},
		{"screening interval 0", KeyScreeningUSPSTF, with(t, KeyScreeningUSPSTF, 8, "interval_months", 0.0), "range:8:interval_months"},

		// child.vaccine_schedule: one row per catalog code, ascending ages.
		{"vaccine duplicate", KeyChildVaccineSchedule, with(t, KeyChildVaccineSchedule, 1, "vaccine_code", "bcg"), "unique:1"},
		{"vaccine unsorted", KeyChildVaccineSchedule, with(t, KeyChildVaccineSchedule, 0, "age_months", 3.0), "sort:1:age_months"},
		{"vaccine code too long", KeyChildVaccineSchedule, with(t, KeyChildVaccineSchedule, 0, "vaccine_code", strings.Repeat("x", 65)), "length:0:vaccine_code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, mustDef(t, tc.key).ValidateRows(tc.rows))
		})
	}

	// Reversed ages: age_min 75 is in range (0–120) but after age_max 74.
	s := mustDef(t, KeyScreeningUSPSTF)
	rows := s.DefaultRows()
	rows[4]["age_min"] = 75.0
	assert.Equal(t, "order:4:age_min", s.ValidateRows(rows))
	// An inclusive band may be a single age.
	rows[4]["age_min"] = 74.0
	assert.Empty(t, s.ValidateRows(rows))
}

func TestParseRows(t *testing.T) {
	def := mustDef(t, KeyPregnancyRedating)
	rows, p := def.ParseRows([]byte(`{"rows":[{"ga_until_days":62,"max_diff_days":5},{"max_diff_days":21}]}`))
	require.Empty(t, p)
	assert.Equal(t, []Row{{"ga_until_days": 62.0, "max_diff_days": 5.0}, {"ga_until_days": nil, "max_diff_days": 21.0}}, rows,
		"a missing nullable column reads null")

	for raw, want := range map[string]string{
		`{"rows":[{"ga_until_days":62,"max_diff_days":5,"x":1}]}`: "column:0:x",
		`{"rows":[{"ga_until_days":62}]}`:                         "null:0:max_diff_days",
		`{"rows":[1]}`:                                            "row:0",
		`{"rows":{}}`:                                             "rows",
		`{"rows":[],"x":1}`:                                       "rows",
		`{"days":3}`:                                              "rows",
		`[]`:                                                      "rows",
	} {
		_, p := def.ParseRows([]byte(raw))
		assert.Equal(t, want, p, raw)
	}

	// Round trip: the stored form parses back to the same rows, columns in schema order.
	for _, def := range Defs {
		if !def.IsRows() {
			continue
		}
		raw, err := def.MarshalRows(def.DefaultRows())
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(raw), `{"rows":[{"`+def.Table.Columns[0].Name+`":`), def.Key)
		back, p := def.ParseRows(raw)
		assert.Empty(t, p, def.Key)
		assert.Equal(t, def.DefaultRows(), back, def.Key)
	}
	_, p = mustDef(t, KeyBPCrisis).ParseRows([]byte(`{"rows":[]}`))
	assert.Equal(t, "rows", p, "a numbers key has no rows")
}

func TestRowsAndSource_AroundEffectiveFrom(t *testing.T) {
	t.Cleanup(Reset)
	eff := civildate.MustParse("2026-11-01")
	v2 := []Row{{"ga_until_days": 97.0, "max_diff_days": 7.0}, {"ga_until_days": nil, "max_diff_days": 21.0}}
	SetVersions([]Version{
		{Key: KeyPregnancyRedating, Version: 2, Rows: v2, Source: "ACOG CO 700 (advisor review 2026)", EffectiveFrom: eff},
		{Key: KeyLabVitD, Version: 2, Value: Value{"deficient_below": 12, "sufficient_from": 30}, EffectiveFrom: eff},
		{Key: KeyChildVaccineSchedule, Version: 2, Rows: nil, EffectiveFrom: eff}, // a rows version without rows is never used
	})
	before, on, after := eff.AddDays(-1), eff, eff.AddDays(1)

	assert.Equal(t, 1, At(KeyPregnancyRedating, before).Version)
	assert.Equal(t, mustDef(t, KeyPregnancyRedating).DefaultRows(), Rows(KeyPregnancyRedating, before))
	assert.Equal(t, srcRedating, SourceOf(KeyPregnancyRedating, before))
	for _, d := range []civildate.Date{on, after} {
		assert.Equal(t, 2, At(KeyPregnancyRedating, d).Version, d.String())
		assert.Equal(t, v2, Rows(KeyPregnancyRedating, d), d.String())
		assert.Equal(t, "ACOG CO 700 (advisor review 2026)", SourceOf(KeyPregnancyRedating, d), d.String())
	}

	// A numbers key's stored version without a source cites the key's source.
	assert.Equal(t, srcVitD, SourceOf(KeyLabVitD, on))
	assert.Equal(t, 30.0, Num(KeyLabVitD, "sufficient_from", on))
	assert.Equal(t, 20.0, Num(KeyLabVitD, "sufficient_from", before))
	assert.Nil(t, Rows(KeyLabVitD, on), "a numbers key has no rows")

	assert.Equal(t, 1, At(KeyChildVaccineSchedule, after).Version)

	// Fingerprint covers rows keys.
	assert.Empty(t, Fingerprint(before, KeyPregnancyRedating, KeyLabVitD))
	assert.Equal(t, "pregnancy.redating@2,lab.vitd@2", Fingerprint(on, KeyPregnancyRedating, KeyLabVitD, KeyChildVaccineSchedule))

	assert.Nil(t, Rows("nope", on))
	assert.Empty(t, SourceOf("nope", on))
	assert.Equal(t, srcBP, SourceOf(KeyBPCrisis, on), "version 1 cites Def.Source")
}

// The refactor leaves every pre-T-M10-16 key a numbers key with the same fingerprint, and adds nothing to the engine
// group (the cycle payload fingerprint, cycle/service engineConfigKeys, is unchanged).
func TestNumbersKeys_FingerprintsUnchanged(t *testing.T) {
	t.Cleanup(Reset)
	before := []string{
		KeyPeriodHardCap, KeyWarningCap, KeyMaxLateDays, KeyFertileWindow, KeyLuteal, KeyBBT, KeyLHBBTConflict,
		KeyPregnancyTest, KeyStaleAnchor, KeyFIGOFrequency, KeyFIGODuration, KeyFIGORegularity, KeyTeenFrequency,
		KeyBPElevated, KeyBPStage1, KeyBPStage2, KeyBPCrisis, KeyGlucoseUrgentLow, KeyGlucoseLow, KeyGlucoseFasting,
		KeyGlucosePostMeal, KeyHRResting, KeyEPDSLikely, KeyEPDSPossible, KeyEPDSShortElevated, KeyEPDSUrgentTotal,
		KeyPostpartumRedFlags, KeyPostpartumVisits, KeyPregnancyBPHigh, KeyPregnancyBPSevere, KeyPregnancyGDMTargets,
		KeyPregnancyKicks, KeyPregnancyManualFetalWeek, KeyPregnancyContractions511, KeyFerritinLow,
	}
	require.Len(t, baseDefs, len(before))
	for i, def := range baseDefs {
		assert.Equal(t, before[i], def.Key, "order of the pre-T-M10-16 keys")
		assert.False(t, def.IsRows(), def.Key)
	}
	for _, def := range coverageDefs {
		assert.NotEqual(t, GroupEngine, def.Group, def.Key)
		assert.False(t, strings.HasPrefix(def.Key, "engine.") || strings.HasPrefix(def.Key, "cycle."), def.Key)
	}

	d := civildate.MustParse("2026-10-21")
	Reset()
	assert.Empty(t, Fingerprint(d, before...))
	vs := make([]Version, 0, len(before))
	for i, k := range before {
		vs = append(vs, Version{Key: k, Version: 2 + i%3, Value: mustDef(t, k).DefaultValue(), EffectiveFrom: d})
	}
	SetVersions(vs)
	want := make([]string, 0, len(before))
	for i, k := range before {
		want = append(want, k+"@"+string(rune('2'+i%3)))
	}
	assert.Equal(t, strings.Join(want, ","), Fingerprint(d, before...), "key@version, in the caller's order")
}

// Numbers coverage keys: v1 passes, each field's bounds and the ordered pairs reject.
func TestCoverageNumbersKeys_Bounds(t *testing.T) {
	for _, def := range coverageDefs {
		if def.IsRows() {
			continue
		}
		assert.Empty(t, def.Validate(def.DefaultValue()), def.Key)
		for _, f := range def.Fields {
			assert.LessOrEqual(t, f.Min, f.Default, "%s.%s", def.Key, f.Name)
			assert.GreaterOrEqual(t, f.Max, f.Default, "%s.%s", def.Key, f.Name)
			low := def.DefaultValue()
			low[f.Name] = f.Min - 1
			assert.Equal(t, "range:"+f.Name, def.Validate(low), def.Key)
			high := def.DefaultValue()
			high[f.Name] = f.Max + 1
			assert.Equal(t, "range:"+f.Name, def.Validate(high), def.Key)
		}
		for _, p := range def.Order {
			v := def.DefaultValue()
			v[p[0]] = v[p[1]]
			if def.Validate(v) == "range:"+p[0] {
				continue // the pair's bounds already keep it ordered
			}
			assert.Equal(t, "order:"+p[0], def.Validate(v), def.Key)
		}
	}
}

// Sources are stored in a varchar(255) and pre-filled into the proposal form, so every v1 source fits.
func TestSources_FitTheColumn(t *testing.T) {
	for _, def := range Defs {
		assert.NotEmpty(t, def.Source, def.Key)
		assert.LessOrEqual(t, len([]rune(def.Source)), 255, def.Key)
	}
}
