package pregnancyalerts

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Handlers serve GET /pregnancy/v2/alerts and POST /pregnancy/v2/alerts/{id}/actions/{action};
// mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	eng   *Engine
	clock clock.Clock
	kv    *cache.Client // nil = EvaluateFirst always evaluates
	build string
}

// NewHandlers wires the handlers; base is the fallback clock.
func NewHandlers(q store.Querier, base clock.Clock) *Handlers {
	return &Handlers{eng: New(q), clock: base}
}

// WithCache keeps the last daily evaluation per user in kv, so EvaluateFirst skips a repeat evaluation whose inputs
// have not changed and whose next due instant has not passed (T-M12-30c, evalcache.go). build identifies the binary
// (etag.Build()). nil kv = always evaluate.
func (h *Handlers) WithCache(kv *cache.Client, build string) *Handlers {
	h.kv, h.build = kv, build
	return h
}

// Engine returns the handlers' engine (the log-save hooks share it).
func (h *Handlers) Engine() *Engine { return h.eng }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// LangOf is the request locale + the site's default language.
func LangOf(c fiber.Ctx) v2.Lang {
	return v2.Lang{Locale: i18n.Locale(c), Default: i18n.LanguagesOf(c).DefaultCode()}
}

func mapErr(err error, locale string) error {
	switch {
	case errors.Is(err, ErrNotActive):
		return httpx.Fail(fiber.StatusConflict, v2.T("messages.not_active", locale), "error_code", "pregnancy_not_active")
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound()
	}
	return err
}

// Index is GET /pregnancy/v2/alerts.
func (h *Handlers) Index(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	l := LangOf(c)
	// GET answers HEAD too (Fiber); HEAD must stay read-only.
	out, err := h.eng.List(c, uid, h.now(c), l, c.Method() != fiber.MethodHead)
	if err != nil {
		return mapErr(err, l.Locale)
	}
	return httpx.OK(c, out)
}

// EvaluateFirst runs the daily evaluation (every rule, deduplicated per window, so idempotent)
// before the next handler — mounted in front of GET /pregnancy/v2/today so its `unread_alerts`
// already counts the calendar alerts (week_entered, weight_missing_week) without the Alerts screen
// being opened first (review #10). HEAD stays read-only. Best effort: a failure is logged and the
// screen is still served (it only shows the badge without the new alerts).
//
// With a cache (WithCache) a repeat read skips the evaluation only when the input revision is unchanged and the
// request clock is before the last evaluation's next due instant (T-M12-30c, evalcache.go); anything uncertain
// evaluates.
func (h *Handlers) EvaluateFirst(log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		uid, ok := auth.CurrentUserID(c)
		if ok && c.Method() != fiber.MethodHead {
			if err := h.evaluateDaily(c, uid, h.now(c), LangOf(c)); err != nil && log != nil {
				log.WarnContext(c, "pregnancy alerts: evaluation before today failed", "user_id", uid, "error", err)
			}
		}
		return c.Next()
	}
}

// evaluateDaily is the daily evaluation of EvaluateFirst behind the per-user skip.
func (h *Handlers) evaluateDaily(c fiber.Ctx, uid uint64, now time.Time, l v2.Lang) error {
	ctx := context.Context(c)
	revs, hasRevs := auth.CurrentRevisions(c)
	rv, hasRules := installedRulesVersioner()
	if h.kv == nil || !hasRevs || !hasRules {
		_, err := h.eng.EvaluateAll(ctx, uid, now, l)
		return err
	}
	today := civildate.InTehran(now)
	// The rule version is read once: a rule edit during the evaluation leaves the stored revision behind the next
	// read's, which then evaluates again.
	rules, rulesErr := rv.RulesVersion(ctx)
	rev := func(st Stamp) string { return revisionOf(revs, st, rules, h.build, today) }
	pre, err := h.eng.InputStamp(ctx, uid, now)
	revOK := err == nil && rulesErr == nil
	preRev := rev(pre)
	if revOK {
		if v, err := h.kv.Get(ctx, evalKey(uid)); err == nil {
			if e, ok := parseEntry(v); ok && e.fresh(preRev, now) {
				cache.Hit(cacheName)
				return nil
			}
		}
		cache.Miss(cacheName)
	}
	due, err := h.eng.EvaluateDaily(ctx, uid, now, l)
	if err != nil || !revOK || due.IsZero() || !due.After(now) {
		return err
	}
	// The evaluation wrote its own inbox rows: store the stamp read after it, but only when no other input moved
	// while it ran (else the next read evaluates again).
	post, err := h.eng.InputStamp(ctx, uid, now)
	if err != nil || post.Data != pre.Data {
		return nil //nolint:nilerr // fail safe: nothing cached, the next read evaluates
	}
	postRev := rev(post)
	ttl := min(due.Sub(now), 25*time.Hour) + time.Minute
	e := entry{rev: postRev, at: now.Unix(), dueSec: due.Unix()}
	if err := h.kv.Set(context.WithoutCancel(ctx), evalKey(uid), e.String(), ttl); err != nil {
		cache.Error(cacheName)
	}
	return nil
}

// cacheName labels the skip's cache_requests_total counters.
const cacheName = "pregnancy_alert_eval"

func installedRulesVersioner() (RulesVersioner, bool) {
	b, err := backendOf()
	if err != nil {
		return nil, false
	}
	rv, ok := b.(RulesVersioner)
	return rv, ok
}

// Action is POST /pregnancy/v2/alerts/{id}/actions/{action} (ack).
func (h *Handlers) Action(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	l := LangOf(c)
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || !contains(Actions, c.Params("action")) {
		return httpx.NotFound()
	}
	act := c.Params("action")
	out, err := h.eng.Act(c, uid, id, act, h.now(c), l)
	if err != nil {
		return mapErr(err, l.Locale)
	}
	return httpx.OK(c, out, T("messages.action_done", l.Locale))
}

// AfterSave is the hook the v1 log-save handlers call (errors bubble up as 500).
func (e *Engine) AfterSave(ctx context.Context, userID uint64, locale, defaultLocale string, now time.Time) error {
	_, err := e.Evaluate(ctx, userID, now, v2.Lang{Locale: locale, Default: defaultLocale})
	return err
}
