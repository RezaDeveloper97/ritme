package idempotency_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/idempotency"
)

const rawKey = "client-key-6f1d2c"

func user(fiber.Ctx) (uint64, bool) { return 42, true }

type harness struct {
	mr    *miniredis.Miniredis
	app   *fiber.App
	calls atomic.Int32
	fail  atomic.Bool  // the handler returns a 422 error
	code  atomic.Int32 // status the handler answers with
	logs  *bytes.Buffer
}

func newHarness(t *testing.T, opts ...idempotency.Option) *harness {
	t.Helper()
	mr := miniredis.RunT(t)
	h := &harness{mr: mr, logs: &bytes.Buffer{}}
	h.code.Store(fiber.StatusCreated)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	logger := slog.New(slog.NewJSONHandler(h.logs, nil))
	mw := idempotency.New(c, user, logger, opts...).Handler()
	h.app = fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(slog.New(slog.DiscardHandler))})
	h.app.Post("/w", mw, func(c fiber.Ctx) error {
		n := h.calls.Add(1)
		if h.fail.Load() {
			return httpx.Fail(fiber.StatusUnprocessableEntity, "nope")
		}
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		return c.Status(int(h.code.Load())).SendString(`{"n":` + strconv.Itoa(int(n)) + `,"fa":"سلام"}`)
	})
	return h
}

func (h *harness) post(t *testing.T, key, body string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/w", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set(idempotency.Header, key)
	}
	res, err := h.app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	require.NoError(t, err)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res, string(b)
}

func TestKey_HashesTheClientKeyPerUser(t *testing.T) {
	sum := sha256.Sum256([]byte(rawKey))
	assert.Equal(t, "idem:42:"+hex.EncodeToString(sum[:]), idempotency.Key(42, rawKey))
	assert.NotContains(t, idempotency.Key(42, rawKey), rawKey)
	assert.NotEqual(t, idempotency.Key(42, rawKey), idempotency.Key(43, rawKey), "keys are scoped per user")
}

func TestFingerprint_DependsOnMethodURLAndBody(t *testing.T) {
	base := idempotency.Fingerprint("POST", "/a", []byte(`{"x":1}`))
	assert.Equal(t, base, idempotency.Fingerprint("POST", "/a", []byte(`{"x":1}`)))
	assert.NotEqual(t, base, idempotency.Fingerprint("PUT", "/a", []byte(`{"x":1}`)))
	assert.NotEqual(t, base, idempotency.Fingerprint("POST", "/b", []byte(`{"x":1}`)))
	assert.NotEqual(t, base, idempotency.Fingerprint("POST", "/a", []byte(`{"x":2}`)))
}

func TestReplay_ReturnsTheStoredResponseByteForByte(t *testing.T) {
	h := newHarness(t)
	first, b1 := h.post(t, rawKey, `{"date":"2026-10-01"}`)
	second, b2 := h.post(t, rawKey, `{"date":"2026-10-01"}`)

	assert.Equal(t, fiber.StatusCreated, first.StatusCode)
	assert.Equal(t, fiber.StatusCreated, second.StatusCode)
	assert.Equal(t, b1, b2)
	assert.Equal(t, first.Header.Get("Content-Type"), second.Header.Get("Content-Type"))
	assert.Equal(t, "true", second.Header.Get(idempotency.ReplayedHeader))
	assert.Empty(t, first.Header.Get(idempotency.ReplayedHeader))
	assert.EqualValues(t, 1, h.calls.Load(), "the handler ran once")
}

func TestStoredResponse_TTLIs24Hours(t *testing.T) {
	h := newHarness(t)
	h.post(t, rawKey, `{}`)
	key := "ritme-go:" + idempotency.Key(42, rawKey)
	require.True(t, h.mr.Exists(key))
	assert.Equal(t, idempotency.TTL, h.mr.TTL(key))

	h.mr.FastForward(idempotency.TTL + time.Second)
	h.post(t, rawKey, `{}`)
	assert.EqualValues(t, 2, h.calls.Load(), "an expired key is processed again")
}

func TestSameKeyDifferentBody_Is422(t *testing.T) {
	h := newHarness(t)
	h.post(t, rawKey, `{"date":"2026-10-01"}`)
	res, body := h.post(t, rawKey, `{"date":"2026-10-02"}`)

	assert.Equal(t, fiber.StatusUnprocessableEntity, res.StatusCode)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	assert.Equal(t, false, got["success"])
	assert.Equal(t, idempotency.CodeReused, got["code"])
	assert.NotEmpty(t, got["message"])
	assert.NotContains(t, got["message"], "idempotency.", "the embedded copy is loaded")
	assert.EqualValues(t, 1, h.calls.Load())
}

func TestHandlerError_ReleasesTheKey(t *testing.T) {
	h := newHarness(t)
	h.fail.Store(true)
	res, _ := h.post(t, rawKey, `{}`)
	assert.Equal(t, fiber.StatusUnprocessableEntity, res.StatusCode)
	assert.False(t, h.mr.Exists("ritme-go:"+idempotency.Key(42, rawKey)), "lock released on handler error")

	h.fail.Store(false)
	res, _ = h.post(t, rawKey, `{}`)
	assert.Equal(t, fiber.StatusCreated, res.StatusCode)
	assert.EqualValues(t, 2, h.calls.Load(), "the retry ran the handler")
}

func TestServerErrorStatus_ReleasesTheKey(t *testing.T) {
	h := newHarness(t)
	h.code.Store(fiber.StatusInternalServerError)
	h.post(t, rawKey, `{}`)
	assert.False(t, h.mr.Exists("ritme-go:"+idempotency.Key(42, rawKey)))
}

func TestInFlight_WaitsThenAnswers409(t *testing.T) {
	h := newHarness(t, idempotency.WithTimings(time.Hour, time.Minute, 150*time.Millisecond))
	fp := idempotency.Fingerprint(http.MethodPost, "/w", []byte(`{}`))
	pending, _ := json.Marshal(map[string]any{"state": "pending", "fp": fp, "owner": "someone"})
	require.NoError(t, h.mr.Set("ritme-go:"+idempotency.Key(42, rawKey), string(pending)))

	res, body := h.post(t, rawKey, `{}`)
	assert.Equal(t, fiber.StatusConflict, res.StatusCode)
	assert.Contains(t, body, idempotency.CodeInFlight)
	assert.EqualValues(t, 0, h.calls.Load())
}

func TestInFlight_ReleasedWhileWaiting_RunsTheHandler(t *testing.T) {
	h := newHarness(t, idempotency.WithTimings(time.Hour, time.Minute, 3*time.Second))
	key := "ritme-go:" + idempotency.Key(42, rawKey)
	fp := idempotency.Fingerprint(http.MethodPost, "/w", []byte(`{}`))
	pending, _ := json.Marshal(map[string]any{"state": "pending", "fp": fp, "owner": "someone"})
	require.NoError(t, h.mr.Set(key, string(pending)))
	go func() {
		time.Sleep(200 * time.Millisecond)
		h.mr.Del(key)
	}()

	res, _ := h.post(t, rawKey, `{}`)
	assert.Equal(t, fiber.StatusCreated, res.StatusCode)
	assert.EqualValues(t, 1, h.calls.Load())
}

func TestNoHeader_PassesThrough(t *testing.T) {
	h := newHarness(t)
	h.post(t, "", `{}`)
	h.post(t, "", `{}`)
	assert.EqualValues(t, 2, h.calls.Load())
	assert.Empty(t, h.mr.Keys())
}

func TestRedisDown_ProcessesNormallyAndNeverLogsTheKey(t *testing.T) {
	h := newHarness(t)
	h.mr.Close()
	res, _ := h.post(t, rawKey, `{}`)
	assert.Equal(t, fiber.StatusCreated, res.StatusCode)
	res, _ = h.post(t, rawKey, `{}`)
	assert.Equal(t, fiber.StatusCreated, res.StatusCode)
	assert.EqualValues(t, 2, h.calls.Load())
	assert.Contains(t, h.logs.String(), "idempotency")
	assert.NotContains(t, h.logs.String(), rawKey)
}

func TestNilCache_PassesThrough(t *testing.T) {
	app := fiber.New()
	var calls int
	app.Post("/w", idempotency.New(nil, user, nil).Handler(), func(c fiber.Ctx) error { calls++; return c.SendStatus(200) })
	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/w", nil)
		req.Header.Set(idempotency.Header, rawKey)
		_, err := app.Test(req)
		require.NoError(t, err)
	}
	assert.Equal(t, 2, calls)
}
