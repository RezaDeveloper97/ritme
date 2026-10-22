package calc

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// D-169 (SPEC v6 §3.1): T1 ≤ 13+6, T2 14+0 … 27+6, T3 ≥ 28+0, by completed days.
func TestTrimesterOn_CompletedDays(t *testing.T) {
	clinicalconfig.Reset()
	for _, tc := range []struct{ w, d, want int }{
		{0, 0, 1}, {13, 6, 1}, {14, 0, 2}, {27, 6, 2}, {28, 0, 3}, {41, 3, 3},
	} {
		assert.Equal(t, tc.want, TrimesterOn(tc.w*7+tc.d, today), "%d+%d", tc.w, tc.d)
	}
	assert.Equal(t, 1, TrimesterOn(-3, today), "a negative age is the first trimester")
	assert.Equal(t, 1, TrimesterOfWeek(14, today), "week 14 = 13+0 … 13+6")
	assert.Equal(t, 2, TrimesterOfWeek(15, today))
	assert.Equal(t, [3]int{0, 98, 196}, TrimesterStartDays(today))

	// The gestational age carries the same rule: LMP 98 days before today = 14+0.
	p := &store.PregnancyProfile{AgeSource: ns("lmp"), LmpDate: civildate.NullDate{Date: today.AddDays(-97), Valid: true}}
	assert.Equal(t, 1, New(p, "en", today).GestationalAge().Trimester, "13+6")
	p.LmpDate.Date = today.AddDays(-98)
	assert.Equal(t, 2, New(p, "en", today).GestationalAge().Trimester, "14+0")
}

// The trimester bounds are the registry version valid on the day.
func TestTrimesterOn_ReadsTheVersionOnTheDay(t *testing.T) {
	from := civildate.MustParse("2026-10-01")
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyPregnancyTrimester, Version: 2,
		Value: clinicalconfig.Value{"t2_from_days": 91, "t3_from_days": 189}, EffectiveFrom: from}})
	t.Cleanup(clinicalconfig.Reset)
	assert.Equal(t, 1, TrimesterOn(91, from.AddDays(-1)))
	assert.Equal(t, 2, TrimesterOn(91, from))
	assert.Equal(t, 3, TrimesterOn(189, from))
}

func TestEDD_NaegeleAndCycleCorrection(t *testing.T) {
	clinicalconfig.Reset()
	lmp := civildate.MustParse("2026-01-01")
	assert.Equal(t, "2026-10-08", EDDFromLMP(lmp, 0).String())
	assert.Equal(t, "2026-10-08", EDDFromLMP(lmp, 35).String(), "the correction is off by default")
	p := &store.PregnancyProfile{AgeSource: ns("lmp"), LmpDate: civildate.NullDate{Date: lmp, Valid: true}}
	edd, ok := New(p, "en", today).WithCycleLength(35).EDDDate()
	require.True(t, ok)
	assert.Equal(t, "2026-10-08", edd.String())

	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyPregnancyEDDCycleCorrect, Version: 2,
		Value: clinicalconfig.Value{"enabled": 1}, EffectiveFrom: civildate.MustParse("2025-12-01")}})
	t.Cleanup(clinicalconfig.Reset)
	assert.Equal(t, "2026-10-15", EDDFromLMP(lmp, 35).String())
	assert.Equal(t, "2026-10-08", EDDFromLMP(lmp, 0).String(), "unknown cycle length: no correction")
	edd, _ = New(p, "en", today).WithCycleLength(35).EDDDate()
	assert.Equal(t, "2026-10-15", edd.String())
	// A scan's due date is never corrected by the cycle length.
	us := &store.PregnancyProfile{AgeSource: ns("ultrasound"), UltrasoundDate: nd("2026-02-26"), UltrasoundWeeks: ni(8), UltrasoundDays: ni(0)}
	edd, _ = New(us, "en", today).WithCycleLength(35).EDDDate()
	assert.Equal(t, "2026-10-08", edd.String())
}

// ACOG CO 700 Table 1: re-date only when the discrepancy is MORE than the band's limit; the band is the GA by LMP on
// the scan day.
func TestRedate_Bands(t *testing.T) {
	clinicalconfig.Reset()
	scan := civildate.MustParse("2026-09-01")
	for _, tc := range []struct {
		w, d, diff int
		replaced   bool
		until      any
		limit      int
	}{
		{8, 6, 5, false, 62, 5}, {8, 6, 6, true, 62, 5},
		{9, 0, 7, false, 97, 7}, {9, 0, 8, true, 97, 7},
		{15, 6, 8, true, 111, 7},
		{16, 0, 10, false, 153, 10}, {16, 0, 11, true, 153, 10},
		{27, 6, 14, false, 195, 14}, {27, 6, 15, true, 195, 14},
		{28, 0, 21, false, nil, 21}, {28, 0, 22, true, nil, 21},
	} {
		byLMP := tc.w*7 + tc.d
		lmp := scan.AddDays(-byLMP)
		for _, sign := range []int{1, -1} { // the scan may say older or younger
			r := Redate(lmp, true, scan, byLMP+sign*tc.diff)
			name := []any{tc.w, tc.d, tc.diff, sign}
			assert.Equal(t, tc.replaced, r.Replaced, "%v", name)
			want := BasisLMP
			if tc.replaced {
				want = BasisUltrasound
			}
			assert.Equal(t, want, r.Basis, "%v", name)
			assert.Equal(t, tc.diff, *r.DiscrepancyDays, "%v", name)
			assert.Equal(t, tc.limit, *r.LimitDays, "%v", name)
			assert.Equal(t, tc.until, intPtr(r.BandUntilDays), "%v", name)
		}
	}
	// Without an LMP the scan dates the pregnancy.
	assert.Equal(t, Redating{Basis: BasisUltrasound}, Redate(civildate.Date{}, false, scan, 60))
	assert.JSONEq(t, `{"basis":"ultrasound","replaced":false,"discrepancy_days":null,"band_until_days":null,"limit_days":null}`,
		mustJSON(t, Redate(civildate.Date{}, false, scan, 60).JSON()))
	assert.JSONEq(t, `{"basis":"ultrasound","replaced":true,"discrepancy_days":6,"band_until_days":62,"limit_days":5}`,
		mustJSON(t, Redate(scan.AddDays(-62), true, scan, 68).JSON()))
}

func TestRedate_ReadsTheVersionOnTheScanDay(t *testing.T) {
	scan := civildate.MustParse("2026-09-01")
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyPregnancyRedating, Version: 2,
		Rows: []clinicalconfig.Row{{"ga_until_days": nil, "max_diff_days": 3.0}}, EffectiveFrom: scan}})
	t.Cleanup(clinicalconfig.Reset)
	assert.True(t, Redate(scan.AddDays(-62), true, scan, 66).Replaced, "v2: 4 > 3")
	assert.False(t, Redate(scan.AddDays(-63), true, scan.AddDays(-1), 67).Replaced, "v1 the day before: 4 ≤ 5")
}

func TestCalculatorDating(t *testing.T) {
	clinicalconfig.Reset()
	assert.Equal(t, Redating{}, New(nil, "en", today).Dating())
	lmpOnly := &store.PregnancyProfile{AgeSource: ns("lmp"), LmpDate: nd("2026-07-01")}
	assert.Equal(t, Redating{Basis: BasisLMP}, New(lmpOnly, "en", today).Dating())
	// LMP 2026-07-01, scan 2026-09-01 (GA by LMP 62 = 8+6) at 9+5 (68): 6 days > 5 → the scan wins.
	both := &store.PregnancyProfile{AgeSource: ns("ultrasound"), LmpDate: nd("2026-07-01"),
		UltrasoundDate: nd("2026-09-01"), UltrasoundWeeks: ni(9), UltrasoundDays: ni(5)}
	d := New(both, "en", today).Dating()
	assert.Equal(t, BasisUltrasound, d.Basis)
	assert.True(t, d.Replaced)
	assert.Equal(t, 6, *d.DiscrepancyDays)
	// The basis is the profile's (a user-chosen LMP stays), the comparison is still reported.
	both.AgeSource = ns("lmp")
	d = New(both, "en", today).Dating()
	assert.Equal(t, BasisLMP, d.Basis)
	assert.True(t, d.Replaced)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// trimesterRule finds a week compared against a trimester bound (12/13/14/27/28) or a hard-coded trimester start
// table: the rule lives in this package only (D-169).
var trimesterRule = regexp.MustCompile(`(?i)\b\w*weeks?\b\s*(<=|<|>=|>)\s*(12|13|14|27|28)\b|\{\s*0\s*,\s*1[34]\s*,\s*28\s*\}`)

// notTrimesterRules are matches that are not a trimester rule.
var notTrimesterRules = map[string]string{
	// The v1 spotting alert window (pregnancy.spotting / spotting_until_week, SPEC v6 §3.1), not a trimester.
	"internal/pregnancy/alerts/alerts.go:" + `if ctx.Week <= 12 || severity == "severe" {`: "spotting window",
}

// No second copy of the trimester rule may remain outside calc (T-M10-18 Scope 3).
func TestNoSecondTrimesterRule(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	var found []string
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "internal/pregnancy/calc/") || strings.HasPrefix(rel, "internal/clinicalconfig/") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			code := strings.TrimSpace(line)
			if strings.HasPrefix(code, "//") || !trimesterRule.MatchString(code) {
				continue
			}
			if _, ok := notTrimesterRules[rel+":"+code]; ok {
				continue
			}
			found = append(found, rel+":"+strconv.Itoa(i+1)+": "+code)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Empty(t, found, "trimester rules outside internal/pregnancy/calc — use calc.TrimesterOn")
}
