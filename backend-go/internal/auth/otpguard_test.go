package auth_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
)

// testLimits are the production defaults (config.go).
var testLimits = config.OTPLimits{
	SendShortMax: 3, SendShortWindow: 15 * time.Minute,
	SendDailyMax: 10, SendDailyWindow: 24 * time.Hour, SendGlobalMax: 30,
	FailureMax: 10, FailureWindow: 24 * time.Hour, LockFor: time.Hour,
	FailureGlobalMax: 50, FailureGlobalWindow: 24 * time.Hour,
	SendLock: 60 * time.Second,
}

// netA / netB are two client networks (ratelimit.ClientNet buckets).
const netA, netB = "203.0.113.7", "198.51.100.9"

func newGuard(t *testing.T) (*auth.OTPGuard, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return auth.NewOTPGuard(cache.NewFromClient(rdb, "ritme-go:"), testLimits, quiet), mr
}

var g0 = time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)

func TestOTPGuard_NilWithoutCache(t *testing.T) {
	assert.Nil(t, auth.NewOTPGuard(nil, testLimits, nil))
}

func TestOTPGuard_SendLock(t *testing.T) {
	g, _ := newGuard(t)
	ctx := context.Background()
	ok, _, err := g.AcquireSend(ctx, "09121234567", g0)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, retry, err := g.AcquireSend(ctx, "09121234567", g0.Add(59*time.Second))
	require.NoError(t, err)
	assert.False(t, ok, "held until the lock ends (request clock)")
	assert.Equal(t, 1, retry)

	ok, _, err = g.AcquireSend(ctx, "09129999999", g0)
	require.NoError(t, err)
	assert.True(t, ok, "another number has its own lock")

	ok, _, err = g.AcquireSend(ctx, "09121234567", g0.Add(60*time.Second))
	require.NoError(t, err)
	assert.True(t, ok, "free once the lock ends")

	g.ReleaseSend(ctx, "09121234567")
	ok, _, err = g.AcquireSend(ctx, "09121234567", g0.Add(61*time.Second))
	require.NoError(t, err)
	assert.True(t, ok, "free after a release")
}

func TestOTPGuard_ShortSendCap(t *testing.T) {
	g, _ := newGuard(t)
	ctx := context.Background()
	for i := range 3 {
		r, err := g.CountSend(ctx, "09121234567", netA, g0.Add(time.Duration(i)*61*time.Second))
		require.NoError(t, err)
		assert.Nil(t, r, "send %d is under the cap", i+1)
	}
	at := g0.Add(183 * time.Second)
	r, err := g.CountSend(ctx, "09121234567", netA, at)
	require.NoError(t, err)
	require.NotNil(t, r, "4th code in 15 min")
	assert.Equal(t, auth.OTPRefusal{Limit: 3, RetryAfter: int((15*time.Minute - 183*time.Second).Seconds())}, *r)

	// Refused sends are not counted: the window frees when the first send leaves it.
	r, err = g.CountSend(ctx, "09121234567", netA, g0.Add(15*time.Minute-time.Second))
	require.NoError(t, err)
	assert.NotNil(t, r, "boundary: one second before the first send leaves the window")
	r, err = g.CountSend(ctx, "09121234567", netA, g0.Add(15*time.Minute))
	require.NoError(t, err)
	assert.Nil(t, r)
}

func TestOTPGuard_DailySendCap(t *testing.T) {
	g, _ := newGuard(t)
	ctx := context.Background()
	step := 5*time.Minute + time.Second // never more than 3 in 15 min
	for i := range 10 {
		r, err := g.CountSend(ctx, "09121234567", netA, g0.Add(time.Duration(i)*step))
		require.NoError(t, err)
		require.Nil(t, r, "send %d is under the daily cap", i+1)
	}
	at := g0.Add(10 * step)
	r, err := g.CountSend(ctx, "09121234567", netA, at)
	require.NoError(t, err)
	require.NotNil(t, r, "11th code in 24 h")
	assert.Equal(t, 10, r.Limit)
	assert.Equal(t, int((24*time.Hour - 10*step).Seconds()), r.RetryAfter)

	r, err = g.CountSend(ctx, "09129999999", netA, at)
	require.NoError(t, err)
	assert.Nil(t, r, "the cap is per number")
	r, err = g.CountSend(ctx, "09121234567", netA, g0.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Nil(t, r, "the first send has left the 24 h window")
}

func TestOTPGuard_FailureBudget(t *testing.T) {
	g, mr := newGuard(t)
	ctx := context.Background()
	for i := range 9 {
		require.NoError(t, g.RecordFailure(ctx, "09121234567", netA, g0.Add(time.Duration(i)*time.Minute)))
	}
	r, err := g.Locked(ctx, "09121234567", netA, g0.Add(9*time.Minute))
	require.NoError(t, err)
	assert.Nil(t, r, "boundary: 9 failures do not lock")

	at := g0.Add(10 * time.Minute)
	require.NoError(t, g.RecordFailure(ctx, "09121234567", netA, at))
	r, err = g.Locked(ctx, "09121234567", netA, at)
	require.NoError(t, err)
	require.NotNil(t, r, "the 10th failure locks")
	assert.Equal(t, auth.OTPRefusal{Limit: 10, RetryAfter: 3600}, *r)

	r, err = g.Locked(ctx, "09121234567", netA, at.Add(time.Hour-time.Second))
	require.NoError(t, err)
	assert.NotNil(t, r, "still locked one second before the hour")
	r, err = g.Locked(ctx, "09121234567", netA, at.Add(time.Hour))
	require.NoError(t, err)
	assert.Nil(t, r, "unlocked after the hour")
	r, err = g.Locked(ctx, "09129999999", netA, at)
	require.NoError(t, err)
	assert.Nil(t, r, "the budget is per number")

	// Old failures leave the 24 h window.
	require.NoError(t, g.RecordFailure(ctx, "09121234567", netA, g0.Add(25*time.Hour)))
	r, err = g.Locked(ctx, "09121234567", netA, g0.Add(25*time.Hour))
	require.NoError(t, err)
	assert.Nil(t, r, "one failure in the new window does not lock")

	for _, k := range mr.Keys() {
		assert.NotContains(t, k, "09121234567", "keys never hold the number")
		assert.NotContains(t, k, netA, "keys never hold the address")
		assert.True(t, strings.HasPrefix(k, "ritme-go:otp:"), k)
	}
}

// T-M12-01b (R-02): the send caps count per (number, network) — a stranger on net A using them up leaves net B free.
func TestOTPGuard_SendCapsArePerNetwork(t *testing.T) {
	g, _ := newGuard(t)
	ctx := context.Background()
	const mobile = "09121234567"
	step := 5*time.Minute + time.Second // never more than 3 in 15 min
	for i := range 10 {
		r, err := g.CountSend(ctx, mobile, netA, g0.Add(time.Duration(i)*step))
		require.NoError(t, err)
		require.Nil(t, r, "net A send %d", i+1)
	}
	at := g0.Add(10 * step)
	r, err := g.CountSend(ctx, mobile, netA, at)
	require.NoError(t, err)
	require.NotNil(t, r, "net A is capped as before")
	assert.Equal(t, 10, r.Limit)

	r, err = g.CountSend(ctx, mobile, netB, at)
	require.NoError(t, err)
	assert.Nil(t, r, "the owner on net B still gets a code")

	// Short cap per network too.
	for i := range 2 {
		r, err = g.CountSend(ctx, mobile, netB, at.Add(time.Duration(i+1)*time.Minute))
		require.NoError(t, err)
		require.Nil(t, r, "net B send %d", i+2)
	}
	r, err = g.CountSend(ctx, mobile, netB, at.Add(3*time.Minute))
	require.NoError(t, err)
	require.NotNil(t, r, "4th code in 15 min from net B")
	assert.Equal(t, 3, r.Limit)
}

// T-M12-01b: the number's global send ceiling counts every network; the 30th send passes, the 31st is refused.
func TestOTPGuard_GlobalSendCeiling(t *testing.T) {
	g, _ := newGuard(t)
	ctx := context.Background()
	const mobile = "09121234567"
	for i := range 30 {
		net := "10.0.0." + strconv.Itoa(i+1) // one send per network: never a per-network cap
		r, err := g.CountSend(ctx, mobile, net, g0.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
		require.Nil(t, r, "send %d is under the global ceiling", i+1)
	}
	at := g0.Add(30 * time.Minute)
	r, err := g.CountSend(ctx, mobile, "10.0.1.1", at)
	require.NoError(t, err)
	require.NotNil(t, r, "31st code in 24 h, from a fresh network")
	assert.Equal(t, auth.OTPRefusal{Limit: 30, RetryAfter: int((24*time.Hour - 30*time.Minute).Seconds())}, *r)

	r, err = g.CountSend(ctx, "09129999999", "10.0.1.1", at)
	require.NoError(t, err)
	assert.Nil(t, r, "the ceiling is per number")
	r, err = g.CountSend(ctx, mobile, "10.0.1.1", g0.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Nil(t, r, "the first send has left the 24 h window")
}

// T-M12-01b: the failure lock is per (number, network); 50 failures across networks lock the number everywhere.
func TestOTPGuard_FailureLockPerNetworkAndGlobal(t *testing.T) {
	g, _ := newGuard(t)
	ctx := context.Background()
	const mobile = "09121234567"
	for i := range 10 {
		require.NoError(t, g.RecordFailure(ctx, mobile, netA, g0.Add(time.Duration(i)*time.Minute)))
	}
	at := g0.Add(10 * time.Minute)
	r, err := g.Locked(ctx, mobile, netA, at)
	require.NoError(t, err)
	require.NotNil(t, r, "net A locked as before")
	assert.Equal(t, 10, r.Limit)
	r, err = g.Locked(ctx, mobile, netB, at)
	require.NoError(t, err)
	assert.Nil(t, r, "net B is not locked by net A's failures")

	// 39 more failures on 13 other networks (3 each, no network lock): 49 in all → not locked.
	for i := range 39 {
		require.NoError(t, g.RecordFailure(ctx, mobile, "10.0.0."+strconv.Itoa(i/3+1), at.Add(time.Duration(i)*time.Second)))
	}
	at = at.Add(time.Minute)
	r, err = g.Locked(ctx, mobile, netB, at)
	require.NoError(t, err)
	assert.Nil(t, r, "boundary: 49 failures on every network do not lock")

	require.NoError(t, g.RecordFailure(ctx, mobile, "10.0.1.1", at))
	r, err = g.Locked(ctx, mobile, netB, at)
	require.NoError(t, err)
	require.NotNil(t, r, "the 50th failure locks the number on every network")
	assert.Equal(t, auth.OTPRefusal{Limit: 50, RetryAfter: 3600}, *r)
	r, err = g.Locked(ctx, mobile, netB, at.Add(time.Hour))
	require.NoError(t, err)
	assert.Nil(t, r, "unlocked after the hour")
}
