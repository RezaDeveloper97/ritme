package auth

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// FailStartup makes the server refuse to start because of err: on the *fiber.App the
// registrars receive, an OnListen hook returns err, which makes Listen abort before serving
// a single request. Tests that only call app.Test never listen, so an incomplete test
// environment (no keys) does not break unrelated route tests. On any other router it panics.
func FailStartup(r fiber.Router, logger *slog.Logger, err error) {
	if logger != nil {
		logger.Error("auth: refusing to start", slog.String("error", err.Error()))
	}
	app, ok := r.(*fiber.App)
	if !ok {
		panic(err)
	}
	app.Hooks().OnListen(func(fiber.ListenData) error { return err })
}

// OnLifecycle runs start when the server starts listening and stop before it shuts down
// (background workers). On a router that is not the *fiber.App, start runs immediately and
// stop is never called.
func OnLifecycle(r fiber.Router, start func() error, stop func()) {
	app, ok := r.(*fiber.App)
	if !ok {
		if err := start(); err != nil {
			panic(err)
		}
		return
	}
	app.Hooks().OnListen(func(fiber.ListenData) error { return start() })
	app.Hooks().OnPreShutdown(func() error {
		stop()
		return nil
	})
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
// registrar that asks builds it and hooks it to the listener (worker + scheduler start on listen and drain before
// shutdown); later registrars get the same instance. Register jobs and scheduler entries while mounting.
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
