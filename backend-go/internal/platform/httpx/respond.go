// Package httpx holds the HTTP shapes every Ritme endpoint shares with Laravel:
// the controller envelopes ({success, message, data}), the framework error bodies
// (422 validation, pretty-printed 404/405/429/500/503), the Fiber error handler that
// maps returned errors onto them, and the three paginator shapes.
//
// Handlers return errors; ErrorHandler renders them. A handler that must answer with
// a non-2xx controller body returns Fail(...).
package httpx

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Send writes body as JSON with the given json_encode flags.
// response()->json() uses flags 0; framework error pages use jsonx.Framework.
func Send(c fiber.Ctx, status int, body any, flags jsonx.Flags) error {
	b, err := jsonx.Marshal(body, flags)
	if err != nil {
		return fmt.Errorf("httpx: encode response: %w", err)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Status(status).Send(b)
}

// JSON is response()->json($body, $status): compact, PHP default escaping. Use it
// with jsonx.Obj for bodies whose literal key order differs from the helpers below.
func JSON(c fiber.Ctx, status int, body any) error { return Send(c, status, body, 0) }

// Envelope builds {"success": true[, "message": msg], "data": data} in that key order.
func Envelope(data any, msg ...string) *jsonx.OrderedMap {
	m := jsonx.Obj("success", true)
	if len(msg) > 0 {
		m.Set("message", msg[0])
	}
	return m.Set("data", data)
}

// OK answers 200 {"success":true[,"message"],"data":…}.
func OK(c fiber.Ctx, data any, msg ...string) error {
	return JSON(c, fiber.StatusOK, Envelope(data, msg...))
}

// Created answers 201 with the same envelope (health-log / reminder / pregnancy creates).
func Created(c fiber.Ctx, data any, msg ...string) error {
	return JSON(c, fiber.StatusCreated, Envelope(data, msg...))
}

// Message answers 200 {"success":true,"message":msg} (no data key, e.g. logout).
func Message(c fiber.Ctx, msg string) error {
	return JSON(c, fiber.StatusOK, jsonx.Obj("success", true, "message", msg))
}

// FailError is a controller error body: {"success":false,"message":…, <extras…>}.
// Return it from a handler (see Fail); ErrorHandler renders it.
type FailError struct {
	Status  int
	Msg     string
	Extras  []any             // key/value pairs appended after "message", in order
	Headers map[string]string // optional response headers
}

// Fail returns a controller failure, e.g.
//
//	return httpx.Fail(422, "Invalid date format. Use YYYY-MM-DD.")
//	return httpx.Fail(429, msg, "data", jsonx.Obj("retry_after", 42))
//	return httpx.Fail(422, msg, "errors", errs, "code", "open_period_exists")
//
// extras are key/value pairs (string keys) appended in order after "message".
func Fail(status int, msg string, extras ...any) *FailError {
	return &FailError{Status: status, Msg: msg, Extras: extras}
}

// WithHeader sets a response header on the failure and returns it.
func (e *FailError) WithHeader(key, value string) *FailError {
	if e.Headers == nil {
		e.Headers = map[string]string{}
	}
	e.Headers[key] = value
	return e
}

func (e *FailError) Error() string { return fmt.Sprintf("httpx: %d %s", e.Status, e.Msg) }

// Body returns the ordered JSON body.
func (e *FailError) Body() *jsonx.OrderedMap {
	m := jsonx.Obj("success", false, "message", e.Msg)
	extras := jsonx.Obj(e.Extras...)
	for _, k := range extras.Keys() {
		v, _ := extras.Get(k)
		m.Set(k, v)
	}
	return m
}

// Render implements Renderer.
func (e *FailError) Render(c fiber.Ctx) error {
	for k, v := range e.Headers {
		c.Set(k, v)
	}
	return JSON(c, e.Status, e.Body())
}

// HTTPStatus implements StatusCoder.
func (e *FailError) HTTPStatus() int { return e.Status }
