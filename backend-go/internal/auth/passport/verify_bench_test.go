package passport_test

import (
	"context"
	"testing"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// T-M9-08 baseline: auth Verify under concurrent requests (the shared client cache and the RS256 verify on every
// core). The single-goroutine BenchmarkVerify (README §8 ceiling ≤ 150 µs) lives in verify_hotpath_test.go.
func BenchmarkVerifyParallel(b *testing.B) {
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live", ClientID: clientID}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	hdr := "Bearer " + mint(&testing.T{}, testKey, claims("live", ts(now), ts(now.AddDate(1, 0, 0))))
	ctx := context.Background()
	if _, err := v.Verify(ctx, hdr); err != nil { // warm the client cache
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := v.Verify(ctx, hdr); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
