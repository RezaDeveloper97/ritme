package auth

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// RunWorkersEnv names the switch that lets the API process run its background work (T-M9-12). "1" (the default:
// local dev, tests, production) starts the job worker, the scheduler and the pollers when the server listens; "0"
// (the stage and beta API containers, which have an `api worker` container next to them) starts none of them. The
// API still dispatches either way.
const RunWorkersEnv = "API_RUN_WORKERS"

// ParseRunWorkers reads an API_RUN_WORKERS value: "" means the default (true); otherwise strconv.ParseBool.
func ParseRunWorkers(v string) (bool, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return true, nil
	}
	on, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s=%q: want 1 or 0", RunWorkersEnv, v)
	}
	return on, nil
}

// RunWorkersFromEnv is ParseRunWorkers(os.Getenv(API_RUN_WORKERS)).
func RunWorkersFromEnv() (bool, error) { return ParseRunWorkers(os.Getenv(RunWorkersEnv)) }

// FailStartup makes the server refuse to start because of err: on the *fiber.App the
// registrars receive, an OnListen hook returns err, which makes Listen abort before serving
// a single request, and StartBackground (the worker role) returns it too. Tests that only call
// app.Test never listen, so an incomplete test environment (no keys) does not break unrelated
// route tests. On any other router it panics.
func FailStartup(r fiber.Router, logger *slog.Logger, err error) {
	if logger != nil {
		logger.Error("auth: refusing to start", slog.String("error", err.Error()))
	}
	app, ok := r.(*fiber.App)
	if !ok {
		panic(err)
	}
	backgroundOf(app).fail(err)
	app.Hooks().OnListen(func(fiber.ListenData) error { return err })
}

// OnLifecycle registers background work (job worker + scheduler, pollers, start-up runs): start runs when the server
// starts listening and stop before it shuts down, both only while the app runs its workers (SetRunWorkers, default
// on); the worker role starts them without a listener (StartBackground). On a router that is not the *fiber.App,
// start runs immediately and stop is never called.
func OnLifecycle(r fiber.Router, start func() error, stop func()) {
	app, ok := r.(*fiber.App)
	if !ok {
		if err := start(); err != nil {
			panic(err)
		}
		return
	}
	backgroundOf(app).add(start, stop)
}

// SetRunWorkers decides whether app starts its background work when it listens (API_RUN_WORKERS). Call it before
// Listen; StartBackground ignores it.
func SetRunWorkers(app *fiber.App, on bool) { backgroundOf(app).setEnabled(on) }

// RunsWorkers reports whether app starts its background work when it listens.
func RunsWorkers(app *fiber.App) bool { return backgroundOf(app).isEnabled() }

// StartBackground starts every hook registered with OnLifecycle, without a listener (`api worker`), whatever
// SetRunWorkers says. It returns the FailStartup error, if any, before starting anything; when a hook fails the ones
// already started are stopped. Idempotent.
func StartBackground(app *fiber.App) error { return backgroundOf(app).start() }

// StopBackground stops the hooks StartBackground (or the listener) started, in registration order. Idempotent.
func StopBackground(app *fiber.App) { backgroundOf(app).stop() }

// BackgroundHooks is the number of hooks registered with OnLifecycle on app.
func BackgroundHooks(app *fiber.App) int { return backgroundOf(app).count() }

// backgroundStateKey holds the app's *background in its State.
const backgroundStateKey = "ritme.background"

var backgroundMu sync.Mutex

type hook struct {
	start func() error
	stop  func()
}

// background is the app's set of lifecycle hooks; one per *fiber.App, hooked to its listener once.
type background struct {
	mu       sync.Mutex
	hooks    []hook
	disabled bool
	failed   error
	running  []hook
	started  bool
}

func backgroundOf(app *fiber.App) *background {
	backgroundMu.Lock()
	defer backgroundMu.Unlock()
	if v, ok := app.State().Get(backgroundStateKey); ok {
		if b, ok := v.(*background); ok {
			return b
		}
	}
	b := &background{}
	app.State().Set(backgroundStateKey, b)
	app.Hooks().OnListen(func(fiber.ListenData) error {
		if !b.isEnabled() {
			return nil
		}
		return b.start()
	})
	app.Hooks().OnPreShutdown(func() error {
		b.stop()
		return nil
	})
	return b
}

func (b *background) add(start func() error, stop func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hooks = append(b.hooks, hook{start: start, stop: stop})
}

func (b *background) fail(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failed == nil {
		b.failed = err
	}
}

func (b *background) setEnabled(on bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.disabled = !on
}

func (b *background) isEnabled() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.disabled
}

func (b *background) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.hooks)
}

func (b *background) start() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failed != nil {
		return b.failed
	}
	if b.started {
		return nil
	}
	for _, h := range b.hooks {
		if err := h.start(); err != nil {
			stopAll(b.running)
			b.running = nil
			return err
		}
		b.running = append(b.running, h)
	}
	b.started = true
	return nil
}

func (b *background) stop() {
	b.mu.Lock()
	running := b.running
	b.running, b.started = nil, false
	b.mu.Unlock()
	stopAll(running)
}

func stopAll(hooks []hook) {
	for _, h := range hooks {
		if h.stop != nil {
			h.stop()
		}
	}
}

// NewJobQueue builds a job queue on the shared Redis (QUEUE_CONNECTION picks async or sync); without a cache it is a
// sync queue (tests).
func NewJobQueue(c *cache.Client, logger *slog.Logger) *queue.Queue {
	if c == nil {
		return queue.New(nil, queue.Options{Mode: queue.ModeSync, Logger: logger})
	}
	return queue.New(c.Redis(), queue.Options{Name: queue.NameFromPrefix(c.Prefix()), Mode: queue.ModeFromEnv(), Logger: logger})
}

// jobsStateKey holds the process queue in the app's State.
const jobsStateKey = "ritme.jobs"

// Jobs is the process job queue and scheduler (T-M9-11, docs/engine-v6/jobs.md), one per *fiber.App: the first
// registrar that asks builds it and hooks it to the app's lifecycle (worker + scheduler start with the background
// work — on listen when the API runs its workers, or in `api worker` — and drain before shutdown); later registrars
// get the same instance. Register jobs and scheduler entries while mounting. Dispatch works whether or not this
// process runs the worker (async mode enqueues on Redis for whichever process does).
//
// On a router that is not the *fiber.App every call returns a fresh, never-started queue: Dispatch still works (sync
// without a cache), nothing is scheduled.
func Jobs(r fiber.Router, c *cache.Client, logger *slog.Logger) *queue.Queue {
	app, ok := r.(*fiber.App)
	if !ok {
		return NewJobQueue(c, logger)
	}
	if q, ok := app.State().Get(jobsStateKey); ok {
		if jq, ok := q.(*queue.Queue); ok {
			return jq
		}
	}
	q := NewJobQueue(c, logger)
	app.State().Set(jobsStateKey, q)
	OnLifecycle(r, q.Start, q.Shutdown)
	return q
}
