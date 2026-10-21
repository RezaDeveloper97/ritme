package legacy_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/benchfixtures/cyclesnap"
	"github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// T-M9-08 baseline: the legacy month (Snapshot.Month → 28–31 HealthDataEngine days + the summary) in calendar
// and full mode, a fresh engine per iteration as per request, and the month body's JSON encode.

var monthYear, monthNum = benchfixtures.Today.Year, int(benchfixtures.Today.Month)

func BenchmarkSnapshotMonth(b *testing.B) {
	from, to := service.MonthRange(monthYear, monthNum)
	ctx := context.Background()
	for _, mode := range []string{"calendar", "full"} {
		for _, h := range benchfixtures.Sizes {
			b.Run(mode+"/H="+strconv.Itoa(h), func(b *testing.B) {
				in := cyclesnap.Load(h, from, to)
				b.ReportAllocs()
				for b.Loop() {
					if _, _, err := in.Snapshot().Month(ctx, monthYear, monthNum, mode == "full"); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkMonthJSON is the cached /cycle/month body ({"calculations":[…],"month_summary":{…}}, unescaped).
func BenchmarkMonthJSON(b *testing.B) {
	from, to := service.MonthRange(monthYear, monthNum)
	for _, mode := range []string{"calendar", "full"} {
		b.Run(mode, func(b *testing.B) {
			calcs, summary, err := cyclesnap.Load(24, from, to).Snapshot().Month(context.Background(), monthYear, monthNum, mode == "full")
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := jsonx.Marshal(jsonx.Obj("calculations", calcs, "month_summary", summary), jsonx.UnescapedSlashes|jsonx.UnescapedUnicode); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
