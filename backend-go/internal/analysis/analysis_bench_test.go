package analysis_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
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

// T-M10-28 budgets: a summary hit (Redis GET of the stored body) against a cold summary (compute + encode + store),
// through Cache.Body on miniredis. The DB part of a cold request (the snapshot and log reads) is not in this number.
func BenchmarkAnalysisSummary(b *testing.B) {
	mr := miniredis.RunT(b)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	b.Cleanup(func() { _ = rdb.Close() })
	c := analysis.NewCache(cache.NewFromClient(rdb, "ritme-go-bench:"), "bench", nil)
	in := benchInput(analysis.DefaultRange)
	compute := func(context.Context) (any, error) { return analysis.BuildSummary(in).JSON(in.Copy), nil }
	key := func(rev uint64) analysis.CacheKey {
		return analysis.CacheKey{UserID: 1, AnalysisRev: rev, Endpoint: analysis.EndpointSummary, Range: analysis.DefaultRange,
			Plus: "d1l1p1", Today: benchfixtures.Today, Locale: "fa", ConfigFP: analysis.ConfigFingerprint(benchfixtures.Today)}
	}
	ctx := context.Background()
	b.Run("hit", func(b *testing.B) {
		if _, err := c.Body(ctx, key(1), compute); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		for b.Loop() {
			if _, err := c.Body(ctx, key(1), compute); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("cold", func(b *testing.B) {
		rev := uint64(1000)
		b.ReportAllocs()
		for b.Loop() {
			rev++ // a new revision: always a miss
			if _, err := c.Body(ctx, key(rev), compute); err != nil {
				b.Fatal(err)
			}
		}
	})
}
