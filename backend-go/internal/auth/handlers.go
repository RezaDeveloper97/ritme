package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// OTP settings (backend/config/sms.php → otp). There is deliberately no test/bypass mode:
// the code is always random and has to be read from otp_verifications in dev.
const (
	OTPLength      = 4
	OTPExpiresIn   = 2 * time.Minute
	OTPMaxAttempts = 5
	OTPResendAfter = 60 * time.Second
)

// JobSendOTPSMS is the queue job type of SendOtpSmsJob.
const JobSendOTPSMS = "send_otp_sms"

// SendOTPJob is the job payload.
type SendOTPJob struct {
	Mobile   string `json:"mobile"`
	Code     string `json:"code"`
	Template string `json:"template"`
}

// Dispatcher enqueues jobs (satisfied by *queue.Queue).
type Dispatcher interface {
	Dispatch(ctx context.Context, jobType string, payload any) error
}

// Querier is everything the auth handlers read and write (satisfied by *store.Queries).
type Querier interface {
	passport.VerifierStore
	passport.IssuerStore
	ProfileQuerier
	TokenRevoker
	GetUserByMobile(ctx context.Context, mobile sql.NullString) (store.User, error)
	CreateUser(ctx context.Context, arg store.CreateUserParams) (sql.Result, error)
	GetRecentOtp(ctx context.Context, arg store.GetRecentOtpParams) (store.GetRecentOtpRow, error)
	DeleteOtpsForMobile(ctx context.Context, mobile string) error
	InsertOtp(ctx context.Context, arg store.InsertOtpParams) error
	GetLatestUnverifiedOtp(ctx context.Context, mobile string) (store.GetLatestUnverifiedOtpRow, error)
	ClaimOtpAttempt(ctx context.Context, arg store.ClaimOtpAttemptParams) (int64, error)
	MarkOtpVerified(ctx context.Context, arg store.MarkOtpVerifiedParams) error
}

// Handlers is Api\V1\OtpAuthController.
type Handlers struct {
	q                 Querier
	issuer            *passport.Issuer
	jobs              Dispatcher
	clock             clock.Clock // fallback when the request has no clock
	refreshWindowDays int
	logger            *slog.Logger
	signupHooks       []SignupHook
}

// SignupHook is told about an account verify-otp has just created (now = the request clock) (e.g. learning: pending course grants of that
// number become active). Hooks run after the token is issued, each bounded by SignupHookTimeout and isolated: an
// error or panic is logged (user id only) and never changes the login response.
type SignupHook func(ctx context.Context, userID uint64, mobile string, now time.Time) error

// SignupHookTimeout bounds one signup hook (keeps the OTP path's latency predictable).
const SignupHookTimeout = 2 * time.Second

// OnSignup registers a signup hook. Call it while wiring routes, before the server serves requests.
func (h *Handlers) OnSignup(fn SignupHook) {
	if fn != nil {
		h.signupHooks = append(h.signupHooks, fn)
	}
}

func (h *Handlers) runSignupHooks(ctx context.Context, userID uint64, mobile string, now time.Time) {
	for _, fn := range h.signupHooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					h.logger.ErrorContext(ctx, "auth: signup hook panicked", slog.Uint64("user_id", userID), slog.Any("panic", r))
				}
			}()
			hctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), SignupHookTimeout)
			defer cancel()
			if err := fn(hctx, userID, mobile, now); err != nil {
				h.logger.ErrorContext(ctx, "auth: signup hook failed", slog.Uint64("user_id", userID), slog.String("error", err.Error()))
			}
		}()
	}
}

// NewHandlers wires the controller.
func NewHandlers(q Querier, issuer *passport.Issuer, jobs Dispatcher, base clock.Clock, refreshWindowDays int, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{q: q, issuer: issuer, jobs: jobs, clock: base, refreshWindowDays: refreshWindowDays, logger: logger}
}

func (h *Handlers) now(c fiber.Ctx) time.Time { return clock.FromContext(c, h.clock).Now() }

var (
	sendRules = validation.Rules{validation.F("mobile", "required", "string", "regex:/^09[0-9]{9}$/")}
	sendMsgs  = validation.Messages("mobile.regex", "Mobile number must be a valid Iranian mobile number (e.g., 09123456789)")

	verifyRules = validation.Rules{
		validation.F("mobile", "required", "string", "regex:/^09[0-9]{9}$/"),
		validation.F("code", "required|string|size:4"),
	}
)

// validate runs Validator::make and returns the controller's 422 body on failure.
func validate(c fiber.Ctx, now time.Time, rules validation.Rules, opts ...validation.Option) (phpval.Map, error) {
	data := validation.Input(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), data, rules, append(opts, validation.Now(now))...)
	if v.Fails() {
		return nil, httpx.Fail(fiber.StatusUnprocessableEntity, "Validation failed", "errors", v.ErrorBag())
	}
	return data, nil
}

func str(m phpval.Map, key string) string {
	v, _ := m.Get(key)
	s, _ := v.(string)
	return s
}

// SendOTP is POST /auth/send-otp (sendOtp). It never reveals whether the mobile is a user.
func (h *Handlers) SendOTP(c fiber.Ctx) error {
	ctx := c.Context()
	now := h.now(c)
	data, err := validate(c, now, sendRules, sendMsgs)
	if err != nil {
		return err
	}
	mobile := str(data, "mobile")

	recent, err := h.q.GetRecentOtp(ctx, store.GetRecentOtpParams{Mobile: mobile, CreatedAt: dbTime(now.Add(-OTPResendAfter))})
	switch {
	case err == nil:
		// $resendAfter - now()->diffInSeconds($recent->created_at): Carbon 3's signed float
		// diff (created_at is in the past, so the diff is negative and retry_after > 60 —
		// Laravel's quirk, locked by the golden: 20 s later → 80).
		diff := float64(recent.CreatedAt.Time.Sub(now).Microseconds()) / 1000 / 1000
		retry := OTPResendAfter.Seconds() - diff
		return httpx.Fail(fiber.StatusTooManyRequests, "Please wait before requesting a new OTP",
			"data", jsonx.Obj("retry_after", jsonx.Float(retry)))
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("auth: recent otp: %w", err)
	}

	code, err := GenerateOTP(OTPLength)
	if err != nil {
		return err
	}
	if err := h.q.DeleteOtpsForMobile(ctx, mobile); err != nil {
		return fmt.Errorf("auth: delete otps: %w", err)
	}
	if err := h.q.InsertOtp(ctx, store.InsertOtpParams{
		Mobile:    mobile,
		Code:      code,
		ExpiresAt: dbTime(now.Add(OTPExpiresIn)).Time,
		CreatedAt: dbTime(now),
		UpdatedAt: dbTime(now),
	}); err != nil {
		return fmt.Errorf("auth: insert otp: %w", err)
	}

	// Delivery is queued; a failing dispatch (or a failing inline job in sync mode) must not
	// change the response — the OTP row is written either way.
	if err := h.jobs.Dispatch(ctx, JobSendOTPSMS, SendOTPJob{Mobile: mobile, Code: code, Template: sms.TemplateLoginOTP}); err != nil {
		h.logger.ErrorContext(ctx, "OTP SMS dispatch failed inline",
			slog.String("mobile", sms.MaskMobile(mobile)), slog.String("error", err.Error()))
	}

	return httpx.OK(c, jsonx.Obj("expires_in", int(OTPExpiresIn/time.Second)), "OTP sent successfully")
}

// GenerateOTP is SmsService::generateOtp: random_int(10^(n-1), 10^n - 1) from crypto/rand.
func GenerateOTP(length int) (string, error) {
	lo := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length-1)), nil)
	hi := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	n, err := rand.Int(rand.Reader, new(big.Int).Sub(hi, lo))
	if err != nil {
		return "", fmt.Errorf("auth: otp: %w", err)
	}
	return n.Add(n, lo).String(), nil
}

// VerifyOTP is POST /auth/verify-otp (verifyOtp).
func (h *Handlers) VerifyOTP(c fiber.Ctx) error {
	ctx := c.Context()
	now := h.now(c)
	data, err := validate(c, now, verifyRules)
	if err != nil {
		return err
	}
	mobile, code := str(data, "mobile"), str(data, "code")

	otp, err := h.q.GetLatestUnverifiedOtp(ctx, mobile)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.Fail(fiber.StatusBadRequest, "No OTP found. Please request a new one.")
	}
	if err != nil {
		return fmt.Errorf("auth: load otp: %w", err)
	}
	if otp.ExpiresAt.Before(now) { // $otp->expires_at->isPast()
		return httpx.Fail(fiber.StatusBadRequest, "OTP has expired. Please request a new one.")
	}

	// Atomic attempt claim: concurrent guesses cannot all pass a read-then-check.
	claimed, err := h.q.ClaimOtpAttempt(ctx, store.ClaimOtpAttemptParams{UpdatedAt: dbTime(now), ID: otp.ID, Attempts: OTPMaxAttempts})
	if err != nil {
		return fmt.Errorf("auth: claim otp attempt: %w", err)
	}
	if claimed == 0 {
		return httpx.Fail(fiber.StatusTooManyRequests, "Too many attempts. Please request a new OTP.")
	}
	if subtle.ConstantTimeCompare([]byte(otp.Code), []byte(code)) != 1 {
		return httpx.Fail(fiber.StatusUnprocessableEntity, "Invalid OTP code")
	}
	if err := h.q.MarkOtpVerified(ctx, store.MarkOtpVerifiedParams{VerifiedAt: dbTime(now), UpdatedAt: dbTime(now), ID: otp.ID}); err != nil {
		return fmt.Errorf("auth: mark otp verified: %w", err)
	}

	user, isNew, err := h.firstOrCreateUser(ctx, mobile, now)
	if err != nil {
		return err
	}
	if !isNew && Blocked(&user) {
		return httpx.Fail(fiber.StatusForbidden, "حساب کاربری شما مسدود شده است.")
	}
	// Laravel's `$user->update(['mobile_verified_at' => now()])` is a no-op
	// (mobile_verified_at is not $fillable), so it is intentionally not written here.

	issued, err := h.issuer.Issue(ctx, user.ID, now)
	if err != nil {
		return err
	}
	if isNew {
		h.runSignupHooks(ctx, user.ID, mobile, now)
	}
	fresh, err := h.q.GetUserByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("auth: reload user: %w", err)
	}
	completed, err := ProfileCompleted(ctx, h.q, &fresh)
	if err != nil {
		return fmt.Errorf("auth: profile exists: %w", err)
	}
	msg := "Login successful"
	if isNew {
		msg = "Registration successful"
	}
	return httpx.OK(c, jsonx.Obj(
		"user", UserJSON(&fresh),
		"new_user", isNew,
		"profile_completed", completed,
		"access_token", issued.AccessToken,
		"token_type", "Bearer",
	), msg)
}

// firstOrCreateUser finds the user by mobile or creates it (User::create(['mobile' => …])).
// A concurrent create of the same mobile (unique index) re-reads the winner's row.
func (h *Handlers) firstOrCreateUser(ctx context.Context, mobile string, now time.Time) (store.User, bool, error) {
	nm := sql.NullString{String: mobile, Valid: true}
	u, err := h.q.GetUserByMobile(ctx, nm)
	if err == nil {
		return u, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return u, false, fmt.Errorf("auth: load user: %w", err)
	}
	res, err := h.q.CreateUser(ctx, store.CreateUserParams{Mobile: nm, CreatedAt: dbTime(now), UpdatedAt: dbTime(now)})
	if err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			u, err := h.q.GetUserByMobile(ctx, nm)
			if err != nil {
				return u, false, fmt.Errorf("auth: load user: %w", err)
			}
			return u, false, nil
		}
		return u, false, fmt.Errorf("auth: create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return u, false, fmt.Errorf("auth: create user: %w", err)
	}
	u, err = h.q.GetUserByID(ctx, uint64(id)) //nolint:gosec // G115: auto-increment ids are positive
	if err != nil {
		return u, false, fmt.Errorf("auth: load new user: %w", err)
	}
	return u, true, nil
}

// Logout is POST /auth/logout: revokes the current token.
func (h *Handlers) Logout(c fiber.Ctx) error {
	tok := CurrentToken(c)
	if tok == nil {
		return &UnauthenticatedError{Code: CodeUnauthenticated}
	}
	if err := RevokeToken(c.Context(), h.q, tok.ID, h.now(c)); err != nil {
		return fmt.Errorf("auth: revoke: %w", err)
	}
	return httpx.Message(c, "Successfully logged out")
}

// User is GET /auth/user.
func (h *Handlers) User(c fiber.Ctx) error {
	u := CurrentUser(c)
	if u == nil {
		return &UnauthenticatedError{Code: CodeUnauthenticated}
	}
	completed, err := ProfileCompleted(c.Context(), h.q, u)
	if err != nil {
		return fmt.Errorf("auth: profile exists: %w", err)
	}
	return httpx.OK(c, jsonx.Obj("user", UserJSON(u), "profile_completed", completed))
}

// RefreshSession is POST /auth/refresh-session: when the current token expires within the
// refresh window (30 days) — or has no expiry — issue a new token first, then revoke the
// old one; otherwise answer refreshed:false and change nothing.
func (h *Handlers) RefreshSession(c fiber.Ctx) error {
	u, tok := CurrentUser(c), CurrentToken(c)
	if u == nil || tok == nil {
		return &UnauthenticatedError{Code: CodeUnauthenticated}
	}
	ctx := c.Context()
	now := h.now(c)
	refreshBefore := now.AddDate(0, 0, h.refreshWindowDays)
	if tok.ExpiresAt.Valid && tok.ExpiresAt.Time.After(refreshBefore) {
		return httpx.OK(c, jsonx.Obj("refreshed", false, "expires_at", jsonx.ISO8601(tok.ExpiresAt.Time)))
	}

	// Issue first, revoke second: if issuing fails the client keeps a working session.
	issued, err := h.issuer.Issue(ctx, u.ID, now)
	if err != nil {
		return err
	}
	if err := RevokeToken(ctx, h.q, tok.ID, now); err != nil {
		return fmt.Errorf("auth: revoke: %w", err)
	}
	return httpx.OK(c, jsonx.Obj(
		"refreshed", true,
		"access_token", issued.AccessToken,
		"token_type", "Bearer",
		"expires_at", jsonx.ISO8601(issued.ExpiresAt),
	))
}
