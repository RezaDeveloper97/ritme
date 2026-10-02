package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Renderer is an error that writes its own response. Later platform packages
// (validation, auth 401s) implement it; ErrorHandler calls Render first.
type Renderer interface {
	error
	Render(c fiber.Ctx) error
}

// StatusCoder exposes the HTTP status an error renders with (for access logs).
type StatusCoder interface {
	HTTPStatus() int
}

// StatusOf returns the status ErrorHandler will answer err with.
func StatusOf(err error) int {
	var sc StatusCoder
	if errors.As(err, &sc) {
		return sc.HTTPStatus()
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return fe.Code
	}
	return fiber.StatusInternalServerError
}

// ---------------------------------------------------------------------------
// Framework errors (Illuminate\Foundation\Exceptions\Handler::prepareJsonResponse):
// pretty-printed, unescaped slashes, {"message": …} only.

// HTTPError is a Symfony HttpException rendered by Laravel: pretty JSON
// {"message": Message} with Status and optional Headers.
type HTTPError struct {
	Status  int
	Message string
	Headers map[string]string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("httpx: %d %s", e.Status, e.Message) }

// HTTPStatus implements StatusCoder.
func (e *HTTPError) HTTPStatus() int { return e.Status }

// Render implements Renderer.
func (e *HTTPError) Render(c fiber.Ctx) error {
	for k, v := range e.Headers {
		c.Set(k, v)
	}
	return Send(c, e.Status, jsonx.Obj("message", e.Message), jsonx.Framework)
}

// Abort is abort($status): an HttpException with an empty message ({"message": ""}).
func Abort(status int) *HTTPError { return &HTTPError{Status: status} }

// NotFound is abort(404).
func NotFound() *HTTPError { return Abort(fiber.StatusNotFound) }

// RouteNotFound is Symfony's NotFoundHttpException for an unmatched route.
// path is the raw request path; Laravel prints $request->path() (slashes trimmed,
// percent-encoding kept, "/" for the root).
func RouteNotFound(path string) *HTTPError {
	return &HTTPError{
		Status:  fiber.StatusNotFound,
		Message: "The route " + laravelPath(path) + " could not be found.",
	}
}

// ModelNotFound is an implicit route-model-binding miss / findOrFail():
// "No query results for model [App\Models\TaskTemplate] 99" (ids joined by ", "),
// or "No query results for model [App\Models\TaskTemplate]." without ids.
func ModelNotFound(model string, ids ...any) *HTTPError {
	msg := "No query results for model [" + model + "]"
	if len(ids) == 0 {
		msg += "."
	} else {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = fmt.Sprint(id)
		}
		msg += " " + strings.Join(parts, ", ")
	}
	return &HTTPError{Status: fiber.StatusNotFound, Message: msg}
}

// laravelVerbs is Router::$verbs, the order Laravel lists supported methods in.
var laravelVerbs = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

// MethodNotAllowed is RouteCollection::methodNotAllowed(): 405 with an Allow header
// and "The GET method is not supported for route api/v1/x. Supported methods: POST."
// allowed is reordered to Laravel's verb order.
func MethodNotAllowed(method, path string, allowed []string) *HTTPError {
	ordered := make([]string, 0, len(allowed))
	for _, v := range laravelVerbs {
		for _, a := range allowed {
			if strings.EqualFold(strings.TrimSpace(a), v) {
				ordered = append(ordered, v)
				break
			}
		}
	}
	list := strings.Join(ordered, ", ")
	return &HTTPError{
		Status: fiber.StatusMethodNotAllowed,
		Message: fmt.Sprintf("The %s method is not supported for route %s. Supported methods: %s.",
			method, laravelPath(path), list),
		Headers: map[string]string{fiber.HeaderAllow: list},
	}
}

// TooManyRequests is ThrottleRequestsException: 429 "Too Many Attempts." with
// Retry-After and the X-RateLimit-* headers (reset is a unix timestamp).
func TooManyRequests(retryAfter, limit, remaining int, reset int64) *HTTPError {
	return &HTTPError{
		Status:  fiber.StatusTooManyRequests,
		Message: "Too Many Attempts.",
		Headers: map[string]string{
			fiber.HeaderRetryAfter:  strconv.Itoa(retryAfter),
			"X-RateLimit-Limit":     strconv.Itoa(limit),
			"X-RateLimit-Remaining": strconv.Itoa(remaining),
			"X-RateLimit-Reset":     strconv.FormatInt(reset, 10),
		},
	}
}

// ServiceUnavailable is the maintenance-mode 503.
func ServiceUnavailable() *HTTPError {
	return &HTTPError{Status: fiber.StatusServiceUnavailable, Message: "Service Unavailable"}
}

// ServerError is the production (APP_DEBUG=false) 500 body.
func ServerError() *HTTPError {
	return &HTTPError{Status: fiber.StatusInternalServerError, Message: "Server Error"}
}

// laravelPath is Request::path(): trim slashes, "/" when empty.
func laravelPath(p string) string {
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	p = strings.Trim(p, "/")
	if p == "" {
		return "/"
	}
	return p
}

// ---------------------------------------------------------------------------
// 422 validation (ValidationException rendered by Handler::invalidJson): compact
// {"message": summary, "errors": {field: [messages…]}}.

// ValidationError collects field messages in rule order. T-M2-06 fills it from the
// rule engine; handlers may also build one directly (Validator::make + errors()->add).
type ValidationError struct {
	fields []string
	msgs   map[string][]string
}

// NewValidationError returns an empty ValidationError.
func NewValidationError() *ValidationError {
	return &ValidationError{msgs: map[string][]string{}}
}

// Add appends msg to field (fields keep first-seen order).
func (v *ValidationError) Add(field, msg string) {
	if v.msgs == nil {
		v.msgs = map[string][]string{}
	}
	if _, ok := v.msgs[field]; !ok {
		v.fields = append(v.fields, field)
	}
	v.msgs[field] = append(v.msgs[field], msg)
}

// Empty reports whether no message was added.
func (v *ValidationError) Empty() bool { return len(v.fields) == 0 }

// Messages returns the messages for field.
func (v *ValidationError) Messages(field string) []string { return v.msgs[field] }

// Summary is ValidationException::summarize(): the first message plus
// " (and N more error)" / " (and N more errors)". The key is translated through
// the app translator, and neither lang/fa nor lang/en has a JSON translation for it,
// so it stays English for every locale (captured: testdata/laravel/validation_422_fa.body).
func (v *ValidationError) Summary() string {
	var all []string
	for _, f := range v.fields {
		all = append(all, v.msgs[f]...)
	}
	if len(all) == 0 {
		return "The given data was invalid."
	}
	msg := all[0]
	if n := len(all) - 1; n > 0 {
		word := "errors"
		if n == 1 {
			word = "error"
		}
		msg += fmt.Sprintf(" (and %d more %s)", n, word)
	}
	return msg
}

func (v *ValidationError) Error() string { return "httpx: validation failed: " + v.Summary() }

// HTTPStatus implements StatusCoder.
func (v *ValidationError) HTTPStatus() int { return fiber.StatusUnprocessableEntity }

// Body returns {"message": summary, "errors": {…}}.
func (v *ValidationError) Body() *jsonx.OrderedMap {
	errs := jsonx.NewObject()
	for _, f := range v.fields {
		errs.Set(f, v.msgs[f])
	}
	return jsonx.Obj("message", v.Summary(), "errors", errs)
}

// Render implements Renderer.
func (v *ValidationError) Render(c fiber.Ctx) error {
	return JSON(c, fiber.StatusUnprocessableEntity, v.Body())
}

// ---------------------------------------------------------------------------

// ErrorHandler is the Fiber error handler: Renderer errors render themselves,
// Fiber's route misses become Laravel's 404/405 pages, any other *fiber.Error an
// HttpException with an empty message, and everything else a logged 500
// "Server Error".
func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var r Renderer
		if errors.As(err, &r) {
			return r.Render(c)
		}
		var fe *fiber.Error
		if errors.As(err, &fe) {
			switch fe.Code {
			case fiber.StatusNotFound:
				return RouteNotFound(rawPath(c)).Render(c)
			case fiber.StatusMethodNotAllowed:
				allowed := strings.Split(string(c.Response().Header.Peek(fiber.HeaderAllow)), ",")
				return MethodNotAllowed(c.Method(), rawPath(c), allowed).Render(c)
			default:
				return Abort(fe.Code).Render(c)
			}
		}
		if logger != nil {
			logger.ErrorContext(c.Context(), "unhandled error",
				slog.String("method", c.Method()), slog.String("path", LogPath(c.Path())), slog.String("error", err.Error()))
		}
		return ServerError().Render(c)
	}
}

// rawPath is the request path as sent (percent-encoding kept), like Symfony's
// getPathInfo().
func rawPath(c fiber.Ctx) string {
	return string(c.Request().URI().PathOriginal())
}
