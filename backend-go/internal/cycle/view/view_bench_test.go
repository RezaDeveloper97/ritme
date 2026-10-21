package view_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/benchfixtures/cyclesnap"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// T-M9-08 baseline: the §19 cycle_view of /cycle/today (metrics + resolve + prediction + daily card) over
// H = 6 / 24 / 60 / 120 periods, and the day payload's JSON encode.

func BenchmarkBuild(b *testing.B) {
	today := benchfixtures.Today
	for _, h := range benchfixtures.Sizes {
		b.Run("H="+strconv.Itoa(h), func(b *testing.B) {
			sn := cyclesnap.Load(h, today, today).Snapshot()
			calc, err := sn.Engine().CalculateForDisplay(context.Background(), today)
			if err != nil {
				b.Fatal(err)
			}
			profile := sn.EngineProfile()
			b.ReportAllocs()
			for b.Loop() {
				_ = view.Build(sn.Histories, profile, today, today, "fa", calc)
			}
		})
	}
}

// BenchmarkDayJSON is the cached /cycle/today body: {"calculation":…,"cycle_view":…} encoded unescaped.
func BenchmarkDayJSON(b *testing.B) {
	today := benchfixtures.Today
	sn := cyclesnap.Load(24, today, today).Snapshot()
	calc, cv, err := sn.Day(context.Background(), today, "fa")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := jsonx.Marshal(jsonx.Obj("calculation", calc, "cycle_view", cv), jsonx.UnescapedSlashes|jsonx.UnescapedUnicode); err != nil {
			b.Fatal(err)
		}
	}
}
