package http

import (
	"os"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/users"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/activity"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/family/invites"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
)

// Auth / OTP (backend/routes/api.php, OtpAuthController). Middleware is attached per route
// so nothing leaks onto other domains' /api/v1 routes.
func init() {
	Register("auth", func(r fiber.Router, d *Deps) {
		m := auth.MustModule(r, auth.Deps{Config: d.Config, DB: d.DB, Cache: d.Cache, Logger: d.Logger})
		if m == nil {
			return // the server refuses to start (FailStartup)
		}
		activity.Install(r, d.DB, d.Cache, d.Logger) // T-M10-30: user-local activity days (async writer, no query on the request)
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h, guard := m.Handlers, m.Guard.RequireUser
		h.OnSignup(learningSignupHook(d)) // B-N8-01: pending course grants of the new number become active
		// T-M11-08 (F-11): open phone invites of the new number bind to the account as pending acceptances.
		if d.DB != nil {
			h.OnSignup(invites.NewService(d.DB, nil, invites.Options{Pepper: []byte(d.Config.Companion.CodePepper)}).SignupHook())
		}
		// T-M10-32 (A-09): the admin reference code of a new account (no secret outside local/testing = no code).
		if rc, err := users.NewRefCoder(d.DB, d.Config.App.Env, os.Getenv(users.RefSecretEnv), d.Logger); err == nil {
			h.OnSignup(rc.SignupHook())
		}
		// T-M10-23 (D-174): a new account must accept the mandatory signup consents (verify-otp `consents`).
		h.RequireSignupConsents(consent.NewSignup(consent.NewService(d.DB)))

		// T-M12-01: named per-IP limiters (IPv6 on its /64); per-number caps live in the handlers (OTPGuard).
		r.Post("/api/v1/auth/send-otp", locale, m.ThrottleOTPSend(), h.SendOTP)
		r.Post("/api/v1/auth/verify-otp", locale, m.ThrottleOTPVerify(), h.VerifyOTP)

		r.Post("/api/v1/auth/logout", locale, guard, h.Logout)
		r.Get("/api/v1/auth/user", locale, guard, h.User)
		r.Post("/api/v1/auth/refresh-session", locale, guard, m.Throttle(10), h.RefreshSession)
	})
}
