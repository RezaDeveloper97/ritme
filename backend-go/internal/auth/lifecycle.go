package auth

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
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
