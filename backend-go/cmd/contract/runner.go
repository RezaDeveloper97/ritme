package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ritme/backend-go/cmd/contract/internal/ojson"
)

// CanonicalHost is the host every request claims (Host header) and every absolute
// URL in a body is rewritten to, so goldens don't depend on CONTRACT_PORT.
const CanonicalHost = "127.0.0.1:8090"

// CanonicalBaseURL is CanonicalHost as a URL (the Go server's APP_URL).
const CanonicalBaseURL = "http://" + CanonicalHost

// Target is a server under test plus the database behind it.
type Target interface {
	Name() string
	BaseURL() string
	// Reset restores the fixture database (dump.sql) and the persona sessions.
	Reset(ctx context.Context) error
	// LatestOTP returns the newest OTP code stored for mobile.
	LatestOTP(ctx context.Context, mobile string) (string, error)
	// Session returns the persona's bearer token ("" + true for anon).
	Session(persona string) (string, bool)
}

// Result is what one step produced.
type Result struct {
	Method string
	URL    string
	Body   []byte // request body sent
	// GoldenBody is the request body before {{var}} substitution (what goldens show:
	// captured values such as OTP codes are random).
	GoldenBody []byte
	Status     int
	Headers    http.Header
	Raw        []byte // response body, base URL canonicalised
}

// Runner executes cases against a target, resetting the DB when needed.
type Runner struct {
	T      Target
	Client *http.Client
	dirty  bool
}

// NewRunner returns a runner; the first case always starts from a fresh DB.
func NewRunner(t Target) *Runner {
	return &Runner{
		T: t,
		Client: &http.Client{
			Timeout:       90 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		dirty: true,
	}
}

// Run executes every step of c.
func (r *Runner) Run(ctx context.Context, c *Case) ([]Result, error) {
	if r.dirty || c.Reset {
		if err := r.T.Reset(ctx); err != nil {
			return nil, fmt.Errorf("reset %s: %w", r.T.Name(), err)
		}
		r.dirty = false
	}
	if c.Writes() {
		r.dirty = true
	}
	vars := map[string]string{}
	out := make([]Result, 0, len(c.Steps))
	for i := range c.Steps {
		s := &c.Steps[i]
		res, err := r.step(ctx, s, vars)
		if err != nil {
			return out, fmt.Errorf("step %d (%s %s): %w", i+1, s.Method, s.Path, err)
		}
		out = append(out, res)
		if len(s.Capture) > 0 {
			v, err := ojson.Parse(res.Raw)
			if err != nil {
				return out, fmt.Errorf("step %d: capture: body is not JSON", i+1)
			}
			for name, sel := range s.Capture {
				got, err := lookup(v, sel)
				if err != nil {
					return out, fmt.Errorf("step %d: capture %s: %w", i+1, name, err)
				}
				vars[name] = got
			}
		}
	}
	return out, nil
}

func (r *Runner) step(ctx context.Context, s *Step, vars map[string]string) (Result, error) {
	if s.OTPFor != "" {
		code, err := r.T.LatestOTP(ctx, s.OTPFor)
		if err != nil {
			return Result{}, fmt.Errorf("otp for %s: %w", s.OTPFor, err)
		}
		vars["otp"] = code
	}
	var body, goldenBody []byte
	contentType := ""
	switch {
	case s.Body != nil:
		body = ojson.Encode(substValue(s.Body.V, vars), "")
		goldenBody = ojson.Encode(s.Body.V, "")
		contentType = "application/json"
	case s.RawBody != nil:
		body = []byte(subst(*s.RawBody, vars))
		goldenBody = []byte(*s.RawBody)
		contentType = firstString(s.ContentType, "application/json")
	}
	target := s.URL(vars)
	req, err := http.NewRequestWithContext(ctx, s.Method, r.T.BaseURL()+target, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Host = CanonicalHost
	req.Header.Set("Accept", "application/json")
	if s.Now != "none" {
		req.Header.Set("X-Test-Now", s.Now)
	}
	if s.Locale != "" {
		req.Header.Set("Accept-Language", s.Locale)
	}
	tok, ok := r.T.Session(s.Persona)
	if !ok {
		return Result{}, fmt.Errorf("persona %q has no session on %s", s.Persona, r.T.Name())
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, h := range s.Headers {
		if h.Value == "" {
			req.Header.Del(h.Key)
			continue
		}
		req.Header.Set(h.Key, subst(h.Value, vars))
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Method: s.Method, URL: target, Body: body, GoldenBody: goldenBody,
		Status: resp.StatusCode, Headers: resp.Header,
		Raw: canonicalise(raw, r.T.BaseURL()),
	}, nil
}

// canonicalise rewrites the target's own base URL (plain and PHP-escaped) to
// CanonicalBaseURL.
func canonicalise(raw []byte, base string) []byte {
	if base == CanonicalBaseURL {
		return raw
	}
	escaped := strings.ReplaceAll(base, "/", `\/`)
	raw = bytes.ReplaceAll(raw, []byte(base), []byte(CanonicalBaseURL))
	return bytes.ReplaceAll(raw, []byte(escaped), []byte(strings.ReplaceAll(CanonicalBaseURL, "/", `\/`)))
}

var selectorIndex = regexp.MustCompile(`\[(\d+)\]`)

// lookup resolves a plain selector (no wildcards) to a scalar.
func lookup(v *ojson.Value, sel string) (string, error) {
	norm := strings.TrimPrefix(strings.TrimPrefix(selectorIndex.ReplaceAllString(sel, ".$1"), "$"), ".")
	cur := v
	for _, seg := range strings.Split(norm, ".") {
		switch cur.Kind {
		case ojson.Object:
			cur = cur.Get(seg)
		case ojson.Array:
			var i int
			if _, err := fmt.Sscanf(seg, "%d", &i); err != nil || i < 0 || i >= len(cur.Arr) {
				return "", fmt.Errorf("%s: no index %q", sel, seg)
			}
			cur = cur.Arr[i]
		default:
			cur = nil
		}
		if cur == nil {
			return "", fmt.Errorf("%s: not found", sel)
		}
	}
	return cur.Scalar(), nil
}
