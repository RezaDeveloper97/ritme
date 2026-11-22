package auth_test

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// T-M12-01 (S-01/S-02): per-number send caps, the failure budget across codes, the atomic send lock and the split
// per-IP limiters, against the real test MariaDB + Redis, time-travelling with X-Test-Now.

func testCache(t *testing.T) *cache.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr}), "ritme-go-test-"+hex.EncodeToString(b)+":")
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// guarded is setup with the OTP guard on (what auth.NewModule wires with Redis).
func guarded(t *testing.T) *env {
	t.Helper()
	c := testCache(t)
	e := setup(t, clock.Real{})
	e.h.UseOTPGuard(auth.NewOTPGuard(c, testLimits, quiet))
	return e
}

var tehran = time.FixedZone("+0330", 3*3600+1800)

var base = time.Date(2026, 11, 18, 10, 0, 0, 0, tehran)

func at(d time.Duration) string { return base.Add(d).Format(time.RFC3339) }

func (e *env) send(t *testing.T, mobile string, d time.Duration) response {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"`+mobile+`"}`, at(d))
}

func (e *env) verify(t *testing.T, mobile, code string, d time.Duration) response {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "", `{"mobile":"`+mobile+`","code":"`+code+`"}`, at(d))
}

func wrongOf(code string) string {
	if code == "0000" {
		return "0001"
	}
	return "0000"
}

// assertThrottle429 is the framework throttle envelope.
func assertThrottle429(t *testing.T, r response, limit string) {
	t.Helper()
	require.Equal(t, 429, r.status, r.raw)
	assert.Equal(t, "{\n    \"message\": \"Too Many Attempts.\"\n}", r.raw)
	assert.Equal(t, limit, r.header.Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", r.header.Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, r.header.Get("Retry-After"))
}

// (a) 3 send + 5-wrong cycles 61 s apart → the 4th send is refused by the per-number cap.
func TestOTPGuard_SendCapAfterThreeCycles(t *testing.T) {
	e := guarded(t)
	const mobile = "09121230001"
	for i := range 3 {
		d := time.Duration(i) * 61 * time.Second
		r := e.send(t, mobile, d)
		require.Equal(t, 200, r.status, "cycle %d: %s", i+1, r.raw)
		wrong := wrongOf(e.otp(t, mobile))
		for range 5 {
			r = e.verify(t, mobile, wrong, d)
			assert.NotEqual(t, 200, r.status)
		}
	}
	r := e.send(t, mobile, 183*time.Second)
	assertThrottle429(t, r, "3")
	assert.Equal(t, "717", r.header.Get("Retry-After"), "until the first send leaves the 15 min window")
	assert.Len(t, e.jobs.jobs, 3, "no SMS for the refused send")

	// Boundary: another number is not affected, and the number sends again once the window moved on.
	assert.Equal(t, 200, e.send(t, "09121230002", 183*time.Second).status)
	assert.Equal(t, 200, e.send(t, mobile, 15*time.Minute).status)
}

// (b) The failure budget spans codes: the 10th wrong code locks the number for an hour, the right code included.
func TestOTPGuard_FailureBudgetLocksTheNumber(t *testing.T) {
	e := guarded(t)
	const mobile = "09121230003"
	for i := range 2 {
		d := time.Duration(i) * 61 * time.Second
		require.Equal(t, 200, e.send(t, mobile, d).status)
		wrong := wrongOf(e.otp(t, mobile))
		for j := range 5 {
			r := e.verify(t, mobile, wrong, d)
			require.Equal(t, 422, r.status, "failure %d: %s", i*5+j+1, r.raw)
		}
	}
	lockedAt := 61 * time.Second
	// The 11th try is the right code: refused.
	r := e.verify(t, mobile, e.otp(t, mobile), lockedAt)
	assertThrottle429(t, r, "10")
	assert.Equal(t, "3600", r.header.Get("Retry-After"))

	// A fresh code just before the hour ends is still refused.
	require.Equal(t, 200, e.send(t, mobile, lockedAt+59*time.Minute).status)
	assertThrottle429(t, e.verify(t, mobile, e.otp(t, mobile), lockedAt+59*time.Minute), "10")
	// A fresh code once the hour is over logs in.
	require.Equal(t, 200, e.send(t, mobile, lockedAt+time.Hour+time.Second).status)
	r = e.verify(t, mobile, e.otp(t, mobile), lockedAt+time.Hour+time.Second)
	assert.Equal(t, 200, r.status, r.raw)
}

// Boundary of (b): 9 wrong codes across two codes do not lock; the right code still logs in.
func TestOTPGuard_NineFailuresDoNotLock(t *testing.T) {
	e := guarded(t)
	const mobile = "09121230004"
	require.Equal(t, 200, e.send(t, mobile, 0).status)
	wrong := wrongOf(e.otp(t, mobile))
	for range 5 {
		require.Equal(t, 422, e.verify(t, mobile, wrong, 0).status)
	}
	require.Equal(t, 200, e.send(t, mobile, 61*time.Second).status)
	wrong = wrongOf(e.otp(t, mobile))
	for range 4 {
		require.Equal(t, 422, e.verify(t, mobile, wrong, 61*time.Second).status)
	}
	r := e.verify(t, mobile, e.otp(t, mobile), 61*time.Second)
	assert.Equal(t, 200, r.status, r.raw)
}

// (c) Parallel sends for one number dispatch exactly one SMS job.
func TestOTPGuard_ParallelSendsOneJob(t *testing.T) {
	e := guarded(t)
	const mobile = "09121230005"
	const n = 5
	statuses := make(chan int, n)
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- e.send(t, mobile, 0).status
		}()
	}
	wg.Wait()
	close(statuses)
	count := map[int]int{}
	for s := range statuses {
		count[s]++
	}
	assert.Equal(t, map[int]int{200: 1, 429: n - 1}, count)
	assert.Len(t, e.jobs.jobs, 1)
	var rows int
	require.NoError(t, e.db.QueryRow("SELECT COUNT(*) FROM otp_verifications WHERE mobile=?", mobile).Scan(&rows))
	assert.Equal(t, 1, rows)

	// The lock ends with the resend window: the next send after it goes out.
	assert.Equal(t, 200, e.send(t, mobile, 61*time.Second).status)
	assert.Len(t, e.jobs.jobs, 2)
}

// (d) IPv6 clients in one /64 share the otp_send bucket; send and verify count separately.
func TestOTPGuard_IPv6SlashSixtyFourSharesTheIPBucket(t *testing.T) {
	e := guarded(t)
	m := &auth.Module{Limiter: ratelimit.New(testCache(t), clock.Real{})}
	app := fiber.New(fiber.Config{
		ErrorHandler:       httpx.ErrorHandler(quiet),
		ProxyHeader:        fiber.HeaderXForwardedFor,
		TrustProxy:         true,
		TrustProxyConfig:   fiber.TrustProxyConfig{Proxies: []string{"0.0.0.0/0", "::/0"}},
		EnableIPValidation: true,
	})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/api/v1/auth/send-otp", m.ThrottleOTPSend(), e.h.SendOTP)
	app.Post("/api/v1/auth/verify-otp", m.ThrottleOTPVerify(), e.h.VerifyOTP)
	post := func(path, ip, body string) int {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(fiber.HeaderXForwardedFor, ip)
		req.Header.Set(clock.Header, at(0))
		resp, err := app.Test(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	// 10 verify tries first: with one shared counter they would use up send's 5/min too.
	for i := range auth.OTPVerifyPerIPPerMinute {
		assert.NotEqual(t, 429, post("/api/v1/auth/verify-otp", "2001:db8:1:2::"+hex.EncodeToString([]byte{byte(i + 1)}),
			`{"mobile":"09121239999","code":"1234"}`), "verify %d", i+1)
	}
	assert.Equal(t, 429, post("/api/v1/auth/verify-otp", "2001:db8:1:2::ff", `{"mobile":"09121239999","code":"1234"}`))

	for i := range auth.OTPSendPerIPPerMinute {
		ip := "2001:db8:1:2:" + hex.EncodeToString([]byte{byte(i + 1)}) + "::1" // rotating addresses, same /64
		assert.Equal(t, 200, post("/api/v1/auth/send-otp", ip, `{"mobile":"0912123100`+string(rune('0'+i))+`"}`), "send %d", i+1)
	}
	assert.Equal(t, 429, post("/api/v1/auth/send-otp", "2001:db8:1:2:ffff:ffff:ffff:ffff", `{"mobile":"09121231009"}`),
		"a new address in the same /64 is the same client")
	assert.Equal(t, 200, post("/api/v1/auth/send-otp", "2001:db8:1:3::1", `{"mobile":"09121231008"}`),
		"the next /64 is another client")
}

// (e) Registered and unknown numbers get the same bodies on send, on a wrong code and at the cap.
func TestOTPGuard_NoEnumeration(t *testing.T) {
	e := guarded(t)
	const known, unknown = "09121230006", "09121230007"
	_, err := e.db.Exec(`INSERT INTO users (mobile, created_at, updated_at) VALUES (?, '2026-01-01 10:00:00', '2026-01-01 10:00:00')`, known)
	require.NoError(t, err)

	for i := range 3 {
		d := time.Duration(i) * 61 * time.Second
		k, u := e.send(t, known, d), e.send(t, unknown, d)
		require.Equal(t, 200, k.status)
		assert.Equal(t, k.raw, u.raw)
		kw, uw := e.verify(t, known, wrongOf(e.otp(t, known)), d), e.verify(t, unknown, wrongOf(e.otp(t, unknown)), d)
		assert.Equal(t, 422, kw.status)
		assert.Equal(t, kw.raw, uw.raw)
	}
	k, u := e.send(t, known, 183*time.Second), e.send(t, unknown, 183*time.Second)
	assertThrottle429(t, k, "3")
	assert.Equal(t, k.raw, u.raw)
	assert.Equal(t, k.header.Get("Retry-After"), u.header.Get("Retry-After"))
}

// T-M12-01b (R-02): caps and the failure lock count per (number, client network). netApp serves the guarded handlers
// behind a trusted proxy so each request names its client address in X-Forwarded-For.
func netApp(e *env) func(t *testing.T, path, ip, body string, d time.Duration) response {
	app := fiber.New(fiber.Config{
		ErrorHandler:       httpx.ErrorHandler(quiet),
		ProxyHeader:        fiber.HeaderXForwardedFor,
		TrustProxy:         true,
		TrustProxyConfig:   fiber.TrustProxyConfig{Proxies: []string{"0.0.0.0/0", "::/0"}},
		EnableIPValidation: true,
	})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/api/v1/auth/send-otp", e.h.SendOTP)
	app.Post("/api/v1/auth/verify-otp", e.h.VerifyOTP)
	return func(t *testing.T, path, ip, body string, d time.Duration) response {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(fiber.HeaderXForwardedFor, ip)
		req.Header.Set(clock.Header, at(d))
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		return response{status: resp.StatusCode, raw: string(raw), header: resp.Header}
	}
}

const attackerNet, ownerNet = "203.0.113.7", "198.51.100.9"

func sendBody(mobile string) string { return `{"mobile":"` + mobile + `"}` }
func verifyBody(mobile, code string) string {
	return `{"mobile":"` + mobile + `","code":"` + code + `"}`
}

// A stranger on net A uses up the send caps; the owner on net B still gets and verifies a code. Net A stays capped.
func TestOTPGuard_StrangerSendCapDoesNotBlockOwner(t *testing.T) {
	e := guarded(t)
	post := netApp(e)
	const mobile = "09121230101"
	for i := range 3 {
		r := post(t, "/api/v1/auth/send-otp", attackerNet, sendBody(mobile), time.Duration(i)*61*time.Second)
		require.Equal(t, 200, r.status, "stranger send %d: %s", i+1, r.raw)
	}
	r := post(t, "/api/v1/auth/send-otp", attackerNet, sendBody(mobile), 183*time.Second)
	assertThrottle429(t, r, "3")

	r = post(t, "/api/v1/auth/send-otp", ownerNet, sendBody(mobile), 183*time.Second)
	require.Equal(t, 200, r.status, "the owner on another network gets a code: %s", r.raw)
	r = post(t, "/api/v1/auth/verify-otp", ownerNet, verifyBody(mobile, e.otp(t, mobile)), 183*time.Second)
	assert.Equal(t, 200, r.status, r.raw)

	assertThrottle429(t, post(t, "/api/v1/auth/send-otp", attackerNet, sendBody(mobile), 245*time.Second), "3")
	assert.Len(t, e.jobs.jobs, 4)
}

// A stranger on net A trips the failure lock; the owner on net B still logs in. Net A stays locked, even with the right
// code and after the account is deleted (keys carry no user id).
func TestOTPGuard_StrangerLockDoesNotBlockOwner(t *testing.T) {
	e := guarded(t)
	post := netApp(e)
	const mobile = "09121230102"
	for i := range 2 {
		d := time.Duration(i) * 61 * time.Second
		require.Equal(t, 200, post(t, "/api/v1/auth/send-otp", attackerNet, sendBody(mobile), d).status)
		wrong := wrongOf(e.otp(t, mobile))
		for range 5 {
			require.Equal(t, 422, post(t, "/api/v1/auth/verify-otp", attackerNet, verifyBody(mobile, wrong), d).status)
		}
	}
	d := 122 * time.Second
	r := post(t, "/api/v1/auth/send-otp", ownerNet, sendBody(mobile), d)
	require.Equal(t, 200, r.status, r.raw)
	code := e.otp(t, mobile)
	assertThrottle429(t, post(t, "/api/v1/auth/verify-otp", attackerNet, verifyBody(mobile, code), d), "10")
	r = post(t, "/api/v1/auth/verify-otp", ownerNet, verifyBody(mobile, code), d)
	require.Equal(t, 200, r.status, "the owner on another network logs in: %s", r.raw)

	_, err := e.db.Exec("DELETE FROM users WHERE mobile=?", mobile)
	require.NoError(t, err)
	require.Equal(t, 200, post(t, "/api/v1/auth/send-otp", ownerNet, sendBody(mobile), d+61*time.Second).status)
	assertThrottle429(t, post(t, "/api/v1/auth/verify-otp", attackerNet, verifyBody(mobile, e.otp(t, mobile)), d+61*time.Second), "10")
}

// 30 sends from 30 networks pass; the 31st from a fresh network is refused by the global ceiling.
func TestOTPGuard_GlobalSendCeilingAcrossNetworks(t *testing.T) {
	e := guarded(t)
	post := netApp(e)
	const mobile = "09121230103"
	for i := range 30 {
		r := post(t, "/api/v1/auth/send-otp", "10.1.0."+strconv.Itoa(i+1), sendBody(mobile), time.Duration(i)*61*time.Second)
		require.Equal(t, 200, r.status, "send %d: %s", i+1, r.raw)
	}
	r := post(t, "/api/v1/auth/send-otp", "10.1.1.1", sendBody(mobile), 30*61*time.Second)
	assertThrottle429(t, r, "30")
	assert.Len(t, e.jobs.jobs, 30)
}

// 49 failures across networks (none locked on its own) leave the right code working; the 50th locks the number on
// every network for an hour.
func TestOTPGuard_GlobalFailureCeilingAcrossNetworks(t *testing.T) {
	e := guarded(t)
	post := netApp(e)
	const mobile = "09121230104"
	failures := 0
	for i := 0; failures < 49; i++ {
		d := time.Duration(i) * 61 * time.Second
		net := "10.2.0." + strconv.Itoa(i+1)
		require.Equal(t, 200, post(t, "/api/v1/auth/send-otp", net, sendBody(mobile), d).status)
		wrong := wrongOf(e.otp(t, mobile))
		for j := 0; j < 5 && failures < 49; j++ {
			require.Equal(t, 422, post(t, "/api/v1/auth/verify-otp", net, verifyBody(mobile, wrong), d).status)
			failures++
		}
	}
	d := 10 * 61 * time.Second // 10th code (49 = 9×5 + 4): one attempt left on it
	r := post(t, "/api/v1/auth/verify-otp", ownerNet, verifyBody(mobile, e.otp(t, mobile)), d)
	require.Equal(t, 200, r.status, "boundary: 49 failures do not lock: %s", r.raw)

	d += 61 * time.Second
	require.Equal(t, 200, post(t, "/api/v1/auth/send-otp", "10.2.1.1", sendBody(mobile), d).status)
	code := e.otp(t, mobile)
	require.Equal(t, 422, post(t, "/api/v1/auth/verify-otp", "10.2.1.1", verifyBody(mobile, wrongOf(code)), d).status)
	r = post(t, "/api/v1/auth/verify-otp", ownerNet, verifyBody(mobile, code), d)
	assertThrottle429(t, r, "50")
	assert.Equal(t, "3600", r.header.Get("Retry-After"))
}
