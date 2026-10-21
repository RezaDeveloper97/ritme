package cache_test

import (
	"strconv"
	"testing"

	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/benchfixtures/cyclesnap"
	"github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/service"
)

// T-M9-08 baseline: the engine cache key (JSON of every input row + SHA-256) that every cached cycle GET hashes
// before it can even look in Redis. Inputs mirror Snapshot.cacheKey: profile, all history rows, the log range,
// its rows and the recommendations signature.

func key(in cyclesnap.Inputs, scope string) cache.Key {
	sig := "recommendations:0"
	return cache.Key{
		UserID: cyclesnap.UserID, Version: 7, Locale: "fa", Today: benchfixtures.Today, Scope: scope,
		Inputs: []any{in.Profile, in.HistoryRows, in.From, in.To, in.LogRows, &sig},
	}
}

func BenchmarkKeyString(b *testing.B) {
	today := benchfixtures.Today
	from, to := service.MonthRange(today.Year, int(today.Month))
	for _, h := range benchfixtures.Sizes {
		for _, scope := range []string{"day", "month"} {
			b.Run(scope+"/H="+strconv.Itoa(h), func(b *testing.B) {
				in := cyclesnap.Load(h, today, today)
				if scope == "month" {
					in = cyclesnap.Load(h, from, to)
				}
				k := key(in, scope)
				b.ReportAllocs()
				for b.Loop() {
					if _, err := k.String(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
