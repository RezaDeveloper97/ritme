package civildate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func zoneCtx(t *testing.T, name string) context.Context {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return WithZone(context.Background(), func(context.Context) *time.Location { return loc })
}

// T-M9-26 (D-126): the instant 2026-07-17T21:00Z is 00:30 on the 18th in Tehran and 23:00 on the 17th in Berlin.
func TestLocalToday_FollowsTheRequestZone(t *testing.T) {
	at := fixedNow(time.Date(2026, 7, 17, 21, 0, 0, 0, time.UTC))
	assert.Equal(t, "2026-07-18", LocalToday(zoneCtx(t, "Asia/Tehran"), at).String())
	assert.Equal(t, "2026-07-17", LocalToday(zoneCtx(t, "Europe/Berlin"), at).String())
	assert.Equal(t, "Europe/Berlin", LocalNow(zoneCtx(t, "Europe/Berlin"), at).Location().String())
}

// No resolver (no usertz middleware, a background job) → Tehran, identical to Today.
func TestLocalToday_DefaultsToTehran(t *testing.T) {
	at := fixedNow(time.Date(2026, 7, 17, 21, 0, 0, 0, time.UTC))
	assert.Equal(t, "2026-07-18", LocalToday(context.Background(), at).String())
	assert.Equal(t, Today(at), LocalToday(context.Background(), at))
	//nolint:staticcheck // nil context is part of the contract
	assert.Equal(t, Tehran, Zone(nil))
	// A resolver that has no answer falls back to Tehran too.
	ctx := WithZone(context.Background(), func(context.Context) *time.Location { return nil })
	assert.Equal(t, "2026-07-18", LocalToday(ctx, at).String())
}

// §6.1: 00:30 Tehran on a server whose clock reports UTC is still the Tehran date.
func TestLocalToday_UTCServerClock(t *testing.T) {
	at := fixedNow(time.Date(2026, 9, 22, 21, 0, 0, 0, time.UTC)) // 2026-09-23 00:30 Tehran
	assert.Equal(t, "2026-09-23", LocalToday(context.Background(), at).String())
	assert.Equal(t, "2026-09-23", LocalToday(zoneCtx(t, "Asia/Tehran"), at).String())
}

// Nowruz 1405 (1 Farvardin = 2026-03-21) starts at 2026-03-20T20:30Z in Tehran (+03:30, no DST since 2022).
func TestLocalToday_NowruzBoundary(t *testing.T) {
	tehran := zoneCtx(t, "Asia/Tehran")
	assert.Equal(t, "2026-03-20", LocalToday(tehran, fixedNow(time.Date(2026, 3, 20, 20, 29, 0, 0, time.UTC))).String())
	assert.Equal(t, "2026-03-21", LocalToday(tehran, fixedNow(time.Date(2026, 3, 20, 20, 30, 0, 0, time.UTC))).String())
	assert.Equal(t, "2026-03-21", LocalToday(tehran, fixedNow(time.Date(2026, 3, 20, 20, 59, 0, 0, time.UTC))).String())
}
