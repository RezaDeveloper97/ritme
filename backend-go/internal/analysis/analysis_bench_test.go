package analysis_test

import (
	"testing"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/resources/translations"
)

// T-M9-08 baseline: GET /analysis/summary's pure part (BuildSummary + the localized body + its JSON encode) for
// a cycle user with 24 periods and a year of log-sheet days, per range tab.

func benchInput(rangeKey string) *analysis.Input {
	rows := make([]analysis.DayEntries, 0, benchfixtures.LogDays)
	for _, d := range benchfixtures.Days(benchfixtures.LogDays) {
		rows = append(rows, analysis.DayEntries{Date: d.Date, Entries: d.Entries})
	}
	// The seeded copy, read with the bundle's own default; the language list itself stays data.
	bundles := i18n.NewTranslationStore(translations.FS, "")
	const locale = "fa"
	return &analysis.Input{
		Today: benchfixtures.Today, Range: analysis.NewRange(rangeKey, benchfixtures.Today),
		Histories: benchfixtures.Histories(24), Profile: benchfixtures.Profile(),
		HeightCM: 165, Birthday: benchfixtures.Birthday, DeepAnalysis: true,
		Days: analysis.BuildDays(rows),
		Copy: analysis.NewCopy(bundles.NamespaceMessages(locale, analysis.Namespace, locale),
			bundles.NamespaceMessages(locale, healthlog.TaxonomyNamespace, locale)),
	}
}

func BenchmarkSummary(b *testing.B) {
	for _, key := range []string{analysis.Range1M, analysis.Range3M, analysis.Range1Y} {
		b.Run(key, func(b *testing.B) {
			in := benchInput(key)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := jsonx.Marshal(analysis.BuildSummary(in).JSON(in.Copy), jsonx.UnescapedUnicode); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
