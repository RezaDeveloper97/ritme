package resolver_test

import (
	"strconv"
	"testing"

	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
)

// T-M9-08 baseline: the v1.1 status resolver and the three-layer metrics over H = 6 / 24 / 60 / 120 periods.
// /cycle/today runs Calculate once and Resolve once per request; a month view runs Resolve per day.

func BenchmarkResolve(b *testing.B) {
	for _, h := range benchfixtures.Sizes {
		b.Run("H="+strconv.Itoa(h), func(b *testing.B) {
			hist, profile := benchfixtures.Histories(h), benchfixtures.Profile()
			m := metrics.Calculate(hist, profile)
			today := benchfixtures.Today
			b.ReportAllocs()
			for b.Loop() {
				_ = resolver.Resolve(hist, profile, today, today, m)
			}
		})
	}
}

// BenchmarkResolveMonth is one calendar month (31 days) resolved day by day over one metrics value.
func BenchmarkResolveMonth(b *testing.B) {
	hist, profile := benchfixtures.Histories(24), benchfixtures.Profile()
	m := metrics.Calculate(hist, profile)
	today := benchfixtures.Today
	first := today.AddDays(-20)
	b.ReportAllocs()
	for b.Loop() {
		for i := range 31 {
			_ = resolver.Resolve(hist, profile, first.AddDays(i), today, m)
		}
	}
}

func BenchmarkCalculate(b *testing.B) {
	for _, h := range benchfixtures.Sizes {
		b.Run("H="+strconv.Itoa(h), func(b *testing.B) {
			hist, profile := benchfixtures.Histories(h), benchfixtures.Profile()
			b.ReportAllocs()
			for b.Loop() {
				_ = metrics.Calculate(hist, profile)
			}
		})
	}
}
