package auth_test

import (
	"context"
	"errors"
	"net"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/queue"
)

func TestMustGuard_MissingKeysRefusesToStart(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	g := auth.MustGuard(app, &config.Config{StoragePath: t.TempDir()}, nil, quiet)
	app.Get("/x", g.RequireUser, func(c fiber.Ctx) error { return c.SendString("ok") })

	// Requests (only possible in tests) are 500, never a 401 that would log clients out.
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/x", nil))
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, 500, resp.StatusCode)

	// Listen aborts before serving.
	assert.Panics(t, func() { _ = app.Listen("127.0.0.1:0", fiber.ListenConfig{DisableStartupMessage: true}) })
}

func TestMustModule_MissingKeysRefusesToStart(t *testing.T) {
	app := fiber.New()
	m := auth.MustModule(app, auth.Deps{Config: &config.Config{StoragePath: t.TempDir()}, Logger: quiet})
	assert.Nil(t, m)
	assert.Panics(t, func() { _ = app.Listen("127.0.0.1:0", fiber.ListenConfig{DisableStartupMessage: true}) })
}

// One job queue per app, shared by every registrar; the OTP job goes on the critical queue (T-M9-11).
func TestJobs_OnePerAppAndOTPIsCritical(t *testing.T) {
	app := fiber.New()
	a := auth.Jobs(app, nil, quiet)
	assert.Same(t, a, auth.Jobs(app, nil, quiet))
	assert.NotSame(t, a, auth.Jobs(app.Group("/x"), nil, quiet), "a group gets its own never-started queue")
	assert.NotSame(t, a, auth.Jobs(fiber.New(), nil, quiet))
	assert.Equal(t, queue.ModeSync, a.Mode(), "no cache → sync")
	assert.Equal(t, queue.Critical, auth.SendOTPJobSpec.Priority)
}

// listenOnce serves app on a free loopback port until its OnListen hooks ran, then shuts it down (pre-shutdown hooks
// run).
func listenOnce(t *testing.T, app *fiber.App) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	listening := make(chan struct{})
	app.Hooks().OnListen(func(fiber.ListenData) error { close(listening); return nil })
	done := make(chan error, 1)
	go func() { done <- app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	select {
	case <-listening:
	case err := <-done:
		t.Fatalf("listener stopped early: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("listener never started")
	}
	require.NoError(t, app.Shutdown())
	// OnListen can fire before fasthttp enters Serve; a Shutdown that lands in that gap leaves Serve running.
	// Closing the listener ends Serve either way (the pre-shutdown hooks already ran in Shutdown).
	_ = ln.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("listener never stopped")
	}
}

// counted registers one lifecycle hook that counts its starts and stops.
func counted(app *fiber.App) (starts, stops *atomic.Int32) {
	starts, stops = new(atomic.Int32), new(atomic.Int32)
	auth.OnLifecycle(app, func() error { starts.Add(1); return nil }, func() { stops.Add(1) })
	return starts, stops
}

// API_RUN_WORKERS=0: the listener starts no lifecycle hook — no job worker, no scheduler, no poller (T-M9-12).
func TestRunWorkers_OffStartsNoHook(t *testing.T) {
	t.Setenv(auth.RunWorkersEnv, "0")
	on, err := auth.RunWorkersFromEnv()
	require.NoError(t, err)
	require.False(t, on)

	app := fiber.New()
	q := auth.Jobs(app, nil, quiet)
	starts, stops := counted(app)
	auth.SetRunWorkers(app, on)
	assert.False(t, auth.RunsWorkers(app))

	listenOnce(t, app)
	assert.Zero(t, starts.Load(), "no poller started")
	assert.Zero(t, stops.Load(), "nothing to stop")
	assert.False(t, q.Scheduler().Leader(), "the scheduler never ran")

	// Dispatch still works (sync without a cache here; async enqueues for the worker container).
	ran := false
	q.Register(queueJob("t912:dispatch"), func(_ context.Context, _ []byte) error { ran = true; return nil })
	require.NoError(t, q.Dispatch(t.Context(), "t912:dispatch", nil))
	assert.True(t, ran)
}

// API_RUN_WORKERS=1 (and unset, the default): the listener starts every hook and stops it on shutdown.
func TestRunWorkers_OnStartsEveryHook(t *testing.T) {
	for _, v := range []string{"1", ""} {
		t.Run("API_RUN_WORKERS="+v, func(t *testing.T) {
			t.Setenv(auth.RunWorkersEnv, v)
			on, err := auth.RunWorkersFromEnv()
			require.NoError(t, err)
			require.True(t, on)

			app := fiber.New()
			q := auth.Jobs(app, nil, quiet)
			leader := false
			auth.OnLifecycle(app, func() error { leader = q.Scheduler().Leader(); return nil }, func() {})
			starts, stops := counted(app)
			auth.SetRunWorkers(app, on)

			listenOnce(t, app)
			assert.True(t, leader, "the scheduler ran (it starts before later hooks)")
			assert.Equal(t, int32(1), starts.Load())
			assert.Equal(t, int32(1), stops.Load())
			assert.False(t, q.Scheduler().Leader(), "stopped on shutdown")
			assert.Equal(t, 3, auth.BackgroundHooks(app))
		})
	}
}

func TestParseRunWorkers(t *testing.T) {
	for in, want := range map[string]bool{"": true, "1": true, "true": true, "0": false, "false": false} {
		got, err := auth.ParseRunWorkers(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	padded, err := auth.ParseRunWorkers("\t0\n")
	require.NoError(t, err, "surrounding whitespace is ignored")
	assert.False(t, padded)
	_, err = auth.ParseRunWorkers("yes please")
	assert.Error(t, err)
}

// The worker role starts the hooks without a listener, whatever API_RUN_WORKERS says; idempotent both ways; a hook
// that fails stops the ones already started.
func TestStartBackground_WorkerRole(t *testing.T) {
	app := fiber.New()
	q := auth.Jobs(app, nil, quiet)
	starts, stops := counted(app)
	auth.SetRunWorkers(app, false) // the worker container gets the API's env, API_RUN_WORKERS=0 included

	require.NoError(t, auth.StartBackground(app))
	require.NoError(t, auth.StartBackground(app))
	assert.Equal(t, int32(1), starts.Load())
	assert.True(t, q.Scheduler().Leader())
	auth.StopBackground(app)
	auth.StopBackground(app)
	assert.Equal(t, int32(1), stops.Load())
	assert.False(t, q.Scheduler().Leader())

	bad := fiber.New()
	firstStarts, firstStops := counted(bad)
	boom := errors.New("boom")
	auth.OnLifecycle(bad, func() error { return boom }, func() {})
	require.ErrorIs(t, auth.StartBackground(bad), boom)
	assert.Equal(t, int32(1), firstStarts.Load())
	assert.Equal(t, int32(1), firstStops.Load(), "rolled back")
}

// A FailStartup error (missing keys, bad SMS config) stops the worker role too, before any hook starts.
func TestStartBackground_RefusesAfterFailStartup(t *testing.T) {
	app := fiber.New()
	starts, _ := counted(app)
	boom := errors.New("no keys")
	auth.FailStartup(app, quiet, boom)
	require.ErrorIs(t, auth.StartBackground(app), boom)
	assert.Zero(t, starts.Load())
}

func queueJob(typ string) queue.Job { return queue.Job{Type: typ, MaxTries: 1} }
