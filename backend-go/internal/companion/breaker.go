package companion

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// AcceptBreaker is the global failed-accept circuit breaker (B-N4-08b, CMP-M3). The per-account and per-IP accept
// throttles do not stop a guesser who spreads blind guesses of code-only invites over many cheap accounts and IPs;
// the breaker counts every refused accept across all callers and, past its thresholds, pauses accepts for everyone
// for a cool-down (the same generic 429 as the throttles) and logs an alert. internal/http implements it on Redis
// and documents the thresholds; nil = no breaker (unit tests).
type AcceptBreaker interface {
	// Open reports whether accepts are paused, and for how many more seconds.
	Open(ctx context.Context) (open bool, retryAfter int, err error)
	// Failed records one refused accept and reports whether it tripped the breaker.
	Failed(ctx context.Context) (tripped bool, err error)
}

// acceptPaused is the 429 while the breaker is open (nil when closed or without a breaker).
func (h *Handlers) acceptPaused(c fiber.Ctx, locale string) error {
	if h.opt.Breaker == nil {
		return nil
	}
	open, retry, err := h.opt.Breaker.Open(c.Context())
	if err != nil {
		return err // fail closed, like the SMS gate
	}
	if !open {
		return nil
	}
	return TooManyAttempts(locale, retry, map[string]string{fiber.HeaderRetryAfter: strconv.Itoa(retry)})
}

// acceptRefused counts a refused accept against the breaker (best effort; never fails the response).
func (h *Handlers) acceptRefused(c fiber.Ctx) {
	if h.opt.Breaker == nil {
		return
	}
	tripped, err := h.opt.Breaker.Failed(c.Context())
	if err != nil {
		h.logger.WarnContext(c.Context(), "companion accept breaker count failed", slog.String("error", err.Error()))
		return
	}
	if tripped {
		// The alert: a burst of refused invite codes across accounts (blind guessing). No code, no user data.
		h.logger.ErrorContext(c.Context(), "companion accept circuit breaker tripped: accepts paused for everyone")
	}
}
