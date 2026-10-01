package analysis

// Golden tests: every report of every seeded history (fixtures_test.go) rendered with the seeded fa copy,
// compared with testdata/golden/<fixture>_<report>.json. Re-record with UPDATE_GOLDEN=1 go test ./internal/analysis/
// and review the diff by hand.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func render(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := jsonx.Marshal(v, jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, json.Indent(&buf, raw, "", "  "))
	buf.WriteByte('\n')
	return buf.Bytes()
}

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name+".json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, got, 0o600))
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // G304: test golden
	require.NoError(t, err, "missing golden (UPDATE_GOLDEN=1)")
	assert.Equal(t, string(want), string(got), name)
}

// reports renders every endpoint body of one fixture.
func reports(t *testing.T, in *Input, c *Copy) map[string]any {
	t.Helper()
	in.Copy = c
	six := ranged(in, Range6M)
	cur, prev := MonthOf(2026, 9, CalendarGregorian, in.Today)
	month := *in
	month.Range = Range{Key: "month", From: cur.From, To: cur.To}
	jcur, jprev := MonthOf(1405, 6, CalendarJalali, in.Today) // Shahrivar 1405
	jmonth := *in
	jmonth.Range = Range{Key: "month", From: jcur.From, To: jcur.To}
	corr := any(jsonx.Obj("locked", true))
	if in.DeepAnalysis {
		corr = BuildCorrelations(six).JSON(c)
	}
	return map[string]any{
		"summary_6m":       withRange(six.Range, BuildSummary(six).JSON(c)),
		"summary_1y":       withRange(ranged(in, Range1Y).Range, BuildSummary(ranged(in, Range1Y)).JSON(c)),
		"cycle_6m":         withRange(six.Range, BuildCycle(six).JSON()),
		"period_6m":        withRange(six.Range, BuildPeriod(six).JSON(c)),
		"symptoms_2w":      withRange(ranged(in, Range2W).Range, BuildSymptoms(ranged(in, Range2W)).JSON(c)),
		"symptoms_6m":      withRange(six.Range, BuildSymptoms(six).JSON(c)),
		"correlations_6m":  corr,
		"body_1m":          withRange(ranged(in, Range1M).Range, BuildBody(ranged(in, Range1M)).JSON()),
		"monthly_2026_09":  BuildMonthly(&month, cur, prev, false).JSON(c),
		"monthly_j1405_06": BuildMonthly(&jmonth, jcur, jprev, true).JSON(c),
	}
}

func TestGolden_SeededHistories(t *testing.T) {
	fa := copyFor(t, "fa")
	names := make([]string, 0, len(fixtures))
	for n := range fixtures {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			for report, v := range reports(t, fixtures[name](), fa) {
				golden(t, name+"_"+report, render(t, v))
			}
		})
	}
}

func TestGolden_EnglishSummary(t *testing.T) {
	in := ranged(regularFixture(), Range6M)
	in.Copy = copyFor(t, "en")
	golden(t, "regular_summary_6m.en", render(t, withRange(in.Range, BuildSummary(in).JSON(in.Copy))))
}

// The goldens are deterministic: the same input renders the same bytes (map iteration never leaks).
func TestReports_Deterministic(t *testing.T) {
	fa := copyFor(t, "fa")
	a := reports(t, regularFixture(), fa)
	for range 5 {
		b := reports(t, regularFixture(), fa)
		for k := range a {
			assert.Equal(t, string(render(t, a[k])), string(render(t, b[k])), k)
		}
	}
}

func TestGolden_RegularHeadlines(t *testing.T) {
	in := ranged(regularFixture(), Range6M)
	in.Copy = copyFor(t, "fa")
	s := BuildSummary(in)
	assert.Equal(t, FindingCycleRegular, s.Finding.Kind)
	f := s.Finding.JSON(in.Copy)
	text, _ := f.Get("text")
	assert.Contains(t, text, "منظم")
	assert.Contains(t, text, "سردرد")
	assert.Contains(t, text, "قبل از پریود")

	ir := ranged(irregularFixture(), Range1Y)
	assert.Equal(t, FindingCycleIrregular, BuildSummary(ir).Finding.Kind)
	assert.Equal(t, FindingNoData, BuildSummary(ranged(emptyFixture(), Range6M)).Finding.Kind)
	assert.Equal(t, FindingNotEnoughData, BuildSummary(ranged(teenFixture(), Range6M)).Finding.Kind)
}
