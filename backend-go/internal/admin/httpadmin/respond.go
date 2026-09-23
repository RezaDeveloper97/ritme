package httpadmin

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Machine-readable error codes (the admin-web translates these; messages are English).
const (
	CodeUnauthenticated = "unauthenticated" // no / unknown session cookie
	CodeSessionExpired  = "session_expired" // cookie present, session gone
	CodeAdminInactive   = "admin_inactive"  // account deactivated or deleted mid-session
	CodeInvalidLogin    = "invalid_credentials"
	CodeCSRF            = "csrf_mismatch"    // 419
	CodeForbidden       = "forbidden"        // 403 (role)
	CodeOrigin          = "origin_forbidden" // 403 (cross-site Origin)
	CodeNotFound        = "not_found"
	CodeValidation      = "validation_failed"
	CodeThrottled       = "too_many_attempts"
	CodeUnsupported     = "unsupported_media_type"
	CodeSelf            = "cannot_modify_self"
)

// StatusCSRF is Laravel's TokenMismatchException status ("Page Expired").
const StatusCSRF = 419

// OK answers 200 {"success":true[,"message"],"data":…}.
func OK(c fiber.Ctx, data any, msg ...string) error { return httpx.OK(c, data, msg...) }

// Created answers 201 with the same envelope.
func Created(c fiber.Ctx, data any, msg ...string) error { return httpx.Created(c, data, msg...) }

// Fail is {"success":false,"message":msg,"error_code":code[, extras…]} with status.
func Fail(status int, code, msg string, extras ...any) *httpx.FailError {
	return httpx.Fail(status, msg, append([]any{"error_code", code}, extras...)...)
}

// Unauthenticated is the admin 401.
func Unauthenticated(code string) *httpx.FailError {
	return Fail(fiber.StatusUnauthorized, code, "Unauthenticated.")
}

// Forbidden is the 403 for a role the admin does not have.
func Forbidden() *httpx.FailError {
	return Fail(fiber.StatusForbidden, CodeForbidden, "This action is restricted to super admins.")
}

// NotFound is the 404 for a missing record.
func NotFound(what string) *httpx.FailError {
	return Fail(fiber.StatusNotFound, CodeNotFound, what+" not found.")
}

// Invalid is the admin 422: {"success":false,"message":summary,"error_code":"validation_failed",
// "errors":{field:[messages…]}} (Laravel's field messages, in the default language).
func Invalid(ve *httpx.ValidationError) *httpx.FailError {
	errs, _ := ve.Body().Get("errors")
	return Fail(fiber.StatusUnprocessableEntity, CodeValidation, ve.Summary(), "errors", errs)
}

// FieldError is Invalid for a single field message.
func FieldError(field, msg string) *httpx.FailError {
	ve := httpx.NewValidationError()
	ve.Add(field, msg)
	return Invalid(ve)
}

// Throttled is the 429 of the login limiter.
func Throttled(retryAfter int) *httpx.FailError {
	return Fail(fiber.StatusTooManyRequests, CodeThrottled,
		"Too many attempts. Try again in "+strconv.Itoa(retryAfter)+" seconds.",
		"retry_after", retryAfter).WithHeader(fiber.HeaderRetryAfter, strconv.Itoa(retryAfter))
}

// Page is the list shape: data = {"items":[…],"meta":{current_page,last_page,per_page,total}}.
func Page[T any](items []T, p Pagination, total int) *jsonx.OrderedMap {
	pg := httpx.NewPage(items, total, p.PerPage, p.Page, "")
	return jsonx.Obj("items", jsonx.List(items), "meta", pg.Meta())
}

// Time is a nullable timestamp as ISO 8601 in Asia/Tehran ("2026-09-23T13:00:00+03:30")
// or null.
func Time(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.ISO8601(t.Time)
}

// NullString is a nullable string or null.
func NullString(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

// AdminJSON is the public shape of an admin account (never the password hash or remember token).
func AdminJSON(a *store.Admin) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", a.ID,
		"name", a.Name,
		"email", a.Email,
		"role", a.Role,
		"is_super", a.Role == RoleSuper,
		"is_active", a.IsActive,
		"last_login_at", Time(a.LastLoginAt),
		"created_at", Time(a.CreatedAt),
		"updated_at", Time(a.UpdatedAt),
	)
}
