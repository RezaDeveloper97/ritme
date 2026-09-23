// Package auth is the Passport-compatible authentication of the Go API: the `auth:api`
// guard (RequireUser), OTP login, logout, /auth/user and refresh-session.
//
// Public API for other domains (routes_<domain>.go):
//
//	guard := auth.MustGuard(r, d.Config, d.DB, d.Logger) // loads the Passport keys; refuses to start without them
//	r.Get("/api/v1/banners", guard.RequireUser, handler)   // attach per route (see RequireUser)
//
//	u := auth.CurrentUser(c)          // *auth.User ($request->user()), nil outside RequireUser
//	id, ok := auth.CurrentUserID(c)
//	tok := auth.CurrentToken(c)       // jti / client / expires_at of the presented token
//	auth.UserJSON(u)                  // the User model's JSON (ordered map; add relations)
//	auth.ProfileCompleted(ctx, q, u)  // hasCompletedProfile()
//	auth.Blocked(u)                   // blocked_at set
//	auth.RevokeUserTokens(ctx, q, id, now) // admin block / account deletion
//	&auth.UnauthenticatedError{Code: auth.CodeUnauthenticated} // the only allowed 401
package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// Deps are what the auth module needs (a subset of http.Deps).
type Deps struct {
	Config *config.Config
	DB     *sql.DB
	Cache  *cache.Client // nil → no limiter, sync queue (tests only)
	Logger *slog.Logger
}

// Module is the wired auth domain.
type Module struct {
	Guard    *Guard
	Handlers *Handlers
	Limiter  *ratelimit.Limiter
	Queue    *queue.Queue
}

// SendOTPJobSpec is SendOtpSmsJob: 3 tries, backoff 5 s then 15 s.
var SendOTPJobSpec = queue.Job{
	Type:     JobSendOTPSMS,
	MaxTries: 3,
	Backoff:  []time.Duration{5 * time.Second, 15 * time.Second},
	Timeout:  60 * time.Second,
}

// NewModule wires the auth domain. It fails when the keys are missing/unreadable or the
// SMS provider configuration is invalid.
func NewModule(d Deps) (*Module, error) {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}
	keys, err := passport.LoadKeys(d.Config.StoragePath)
	if err != nil {
		return nil, err
	}
	smsSvc, err := sms.NewService(d.Config.SMS, d.Config.App.Env, &http.Client{Timeout: 30 * time.Second}, logger)
	if err != nil {
		return nil, err
	}
	q := store.New(d.DB)

	var rdbQueue *queue.Queue
	var limiter *ratelimit.Limiter
	if d.Cache != nil {
		rdbQueue = queue.New(d.Cache.Redis(), queue.Options{
			Name: queue.NameFromPrefix(d.Cache.Prefix()), Mode: queue.ModeFromEnv(), Logger: logger,
		})
		limiter = ratelimit.New(d.Cache, clock.Real{})
	} else {
		rdbQueue = queue.New(nil, queue.Options{Mode: queue.ModeSync, Logger: logger})
	}
	rdbQueue.Register(SendOTPJobSpec, SendOTPSMSHandler(smsSvc))
	rdbQueue.OnFailed(func(ctx context.Context, _ string, payload []byte, err error) {
		var job SendOTPJob
		_ = json.Unmarshal(payload, &job)
		logger.ErrorContext(ctx, "OTP SMS job exhausted all retries",
			slog.String("mobile", sms.MaskMobile(job.Mobile)), slog.String("error", err.Error()))
	})

	issuer := passport.NewIssuer(keys.Private, q, clock.Real{}, d.Config.Passport.TokenLifetimeDays)
	return &Module{
		Guard:    NewGuardWith(keys.Public, q, clock.Real{}, logger),
		Handlers: NewHandlers(q, issuer, rdbQueue, clock.Real{}, d.Config.Passport.RefreshWindowDays, logger),
		Limiter:  limiter,
		Queue:    rdbQueue,
	}, nil
}

// MustModule is NewModule for the auth registrar: on error the server refuses to start
// (FailStartup) and nil is returned. On success the queue worker starts with the listener
// and stops (draining running jobs) before shutdown.
func MustModule(r fiber.Router, d Deps) *Module {
	m, err := NewModule(d)
	if err != nil {
		FailStartup(r, d.Logger, err)
		return nil
	}
	OnLifecycle(r, m.Queue.Start, m.Queue.Shutdown)
	return m
}

// Throttle is `throttle:n,1` (per user behind RequireUser, else per client IP).
func (m *Module) Throttle(n int) fiber.Handler {
	if m.Limiter == nil {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	return m.Limiter.Middleware(n, time.Minute, ThrottleIdentity)
}

// SendOTPSMSHandler is SendOtpSmsJob::handle: fails the try when every provider failed.
func SendOTPSMSHandler(svc *sms.Service) queue.Handler {
	return func(ctx context.Context, payload []byte) error {
		var job SendOTPJob
		if err := json.Unmarshal(payload, &job); err != nil {
			return fmt.Errorf("auth: send_otp_sms payload: %w", err)
		}
		if err := svc.SendOTP(ctx, job.Mobile, job.Code, job.Template); err != nil {
			return fmt.Errorf("OTP SMS delivery failed for %s: %w", sms.MaskMobile(job.Mobile), err)
		}
		return nil
	}
}
