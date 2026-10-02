package home

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy"
)

// Model class names of the route-model-binding 404s (kept as Laravel prints them).
const (
	modelTaskTemplate     = `App\Models\TaskTemplate`
	modelChallenge        = `App\Models\Challenge`
	modelUserNotification = `App\Models\UserNotification`
)

// Handlers are the HomeController actions. Mount them behind auth RequireUser and the locale
// middleware.
type Handlers struct {
	deps  *Deps
	cycle *cycleservice.Service
	page  *Page
	clock clock.Clock
}

// NewHandlers wires the handlers; cycle loads the engine inputs (no engine cache: HomeContext
// computes directly), base is the fallback clock (clock.Middleware's request clock wins).
func NewHandlers(d Deps, cycle *cycleservice.Service, base clock.Clock) *Handlers {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Handlers{deps: &d, cycle: cycle, page: NewPage(d.Logger), clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time { return clock.FromContext(c, h.clock).Now() }

func currentUser(c fiber.Ctx) (*auth.User, error) {
	u := auth.CurrentUser(c)
	if u == nil {
		return nil, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return u, nil
}

// buildContext is HomeController::buildContext(): `date` nullable|date (framework 422), the
// locale, the date, the pregnancy profile → mode, and the cycle snapshot of the log window.
func (h *Handlers) buildContext(c fiber.Ctx) (*Context, error) {
	now := h.now(c)
	query := validation.Query(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), query,
		validation.Rules{validation.F("date", "nullable", "date")}, validation.Now(now))
	if v.Fails() {
		return nil, v.Errors()
	}
	user, err := currentUser(c)
	if err != nil {
		return nil, err
	}
	today := civildate.InTehran(now)
	date := today
	if raw, _ := query.Get("date"); phpval.Truthy(raw) {
		t, err := civildate.ParseLenient(phpval.ToString(raw), now, civildate.Tehran)
		if err != nil {
			return nil, fmt.Errorf("home: parse date: %w", err) // unreachable: the date rule passed
		}
		date = civildate.FromTime(t)
	}
	return h.newContext(c.Context(), user, date, today, i18n.ResolveLocale(c, ""), i18n.LanguagesOf(c).DefaultCode())
}

// newContext loads the per-request inputs of a Context.
func (h *Handlers) newContext(ctx context.Context, user *auth.User, date, today civildate.Date, locale, def string) (*Context, error) {
	hc := newContext(ctx, h.deps)
	hc.UserID, hc.UserName = user.ID, user.Name
	hc.Date, hc.Today, hc.Locale, hc.DefaultLocale = date, today, locale, def

	preg, err := pregnancy.LoadProfile(ctx, h.deps.Pregnancy, user.ID)
	if err != nil {
		return nil, err
	}
	hc.Pregnancy = preg
	hc.Mode = enums.MessageModeCycle
	if preg != nil && preg.PregnancyMode {
		hc.Mode = enums.MessageModePregnancy
	}
	from, to := LogWindow(date)
	if hc.Snapshot, err = h.cycle.Load(ctx, user.ID, from, to, today); err != nil {
		return nil, err
	}
	return hc, nil
}

// refuseCompanion is the 409 companion_account of a male account (bloom B-N4-03): his home is GET /companion/home.
func (h *Handlers) refuseCompanion(c fiber.Ctx) error {
	if h.deps.Accounts == nil {
		return nil
	}
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return nil // the guard answers 401 first
	}
	is, err := h.deps.Accounts.IsCompanion(c.Context(), uid)
	if err != nil {
		return err
	}
	if is {
		return companion.AccountConflict(i18n.Locale(c))
	}
	return nil
}

// Index is GET /home.
func (h *Handlers) Index(c fiber.Ctx) error {
	if err := h.refuseCompanion(c); err != nil {
		return err
	}
	hc, err := h.buildContext(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj(
		"mode", string(hc.Mode),
		"mode_label", hc.Mode.Label(hc.Locale),
		"date", hc.Date.String(),
		"locale", hc.Locale,
		"profile_complete", hc.HasCycleData(),
		"sections", h.page.Build(hc),
	))
}

// Section is GET /home/sections/{section}: unknown key → 404 with available_sections (before
// any validation), otherwise `section` (null when unsupported or empty).
func (h *Handlers) Section(c fiber.Ctx) error {
	key := param(c, "section")
	if !h.page.Has(key) {
		msg := "Unknown section"
		if i18n.ResolveLocale(c, "") == "fa" {
			msg = "بخش موردنظر یافت نشد"
		}
		return httpx.Fail(fiber.StatusNotFound, msg, "available_sections", h.page.Keys())
	}
	if err := h.refuseCompanion(c); err != nil {
		return err
	}
	hc, err := h.buildContext(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("section", h.page.BuildSection(key, hc)))
}

// ToggleTask is POST /home/tasks/{task}/toggle.
func (h *Handlers) ToggleTask(c fiber.Ctx) error {
	user, err := currentUser(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	q := h.deps.Queries
	raw := param(c, "task")
	taskID, err := bindModel(raw, func(id uint64) (uint64, error) { return q.GetTaskTemplateID(ctx, id) })
	if err != nil {
		if errors.Is(err, errModelMissing) {
			return httpx.ModelNotFound(modelTaskTemplate, raw)
		}
		return fmt.Errorf("home: bind task: %w", err)
	}

	now := h.now(c)
	today := civildate.InTehran(now)
	isCompleted := false
	existing, err := q.GetTaskCompletionOn(ctx, store.GetTaskCompletionOnParams{UserID: user.ID, TaskTemplateID: taskID, CompletionDate: today})
	switch {
	case err == nil:
		if err := q.DeleteTaskCompletion(ctx, existing); err != nil {
			return fmt.Errorf("home: delete task completion: %w", err)
		}
	case errors.Is(err, sql.ErrNoRows):
		ts := sql.NullTime{Time: dbNow(now), Valid: true}
		if err := q.InsertTaskCompletion(ctx, store.InsertTaskCompletionParams{
			UserID: user.ID, TaskTemplateID: taskID, CompletionDate: today, CompletedAt: ts, CreatedAt: ts, UpdatedAt: ts,
		}); err != nil {
			return fmt.Errorf("home: insert task completion: %w", err)
		}
		isCompleted = true
	default:
		return fmt.Errorf("home: task completion: %w", err)
	}

	// taskProgress(): a fresh context (its phase), completions counted on today.
	hc, err := h.buildContext(c)
	if err != nil {
		return err
	}
	phase, err := hc.Phase()
	if err != nil {
		return err
	}
	templates, err := taskTemplates(hc, phase)
	if err != nil {
		return err
	}
	done, err := completedTaskIDs(hc, today)
	if err != nil {
		return err
	}
	completed := 0
	for _, t := range templates {
		if done[t.ID] {
			completed++
		}
	}
	total := len(templates)
	return httpx.OK(c, jsonx.Obj(
		"task_id", taskID,
		"is_completed", isCompleted,
		"progress", jsonx.Obj("completed", completed, "total", total, "percent", percentOf(completed, total)),
	))
}

// ToggleChallenge is POST /home/challenges/{challenge}/toggle.
func (h *Handlers) ToggleChallenge(c fiber.Ctx) error {
	user, err := currentUser(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	q := h.deps.Queries
	raw := param(c, "challenge")
	id, err := bindModel(raw, func(id uint64) (uint64, error) { return q.GetChallengeID(ctx, id) })
	if err != nil {
		if errors.Is(err, errModelMissing) {
			return httpx.ModelNotFound(modelChallenge, raw)
		}
		return fmt.Errorf("home: bind challenge: %w", err)
	}
	now := h.now(c)
	done, err := ToggleChallenge(ctx, q, user.ID, id, civildate.InTehran(now), now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("challenge_id", id, "is_completed", done))
}

// Notifications is GET /home/notifications: per_page (int) cast clamped to 1..100 (default 20).
func (h *Handlers) Notifications(c fiber.Ctx) error {
	user, err := currentUser(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	q := h.deps.Queries
	locale := i18n.ResolveLocale(c, "")
	def := i18n.LanguagesOf(c).DefaultCode()

	perPage := int64(20)
	if raw, ok := validation.Query(c).Get("per_page"); ok {
		perPage = phpIntCast(raw)
	}
	perPage = max(1, min(perPage, 100))
	page := int64(httpx.PageParam(c))

	total, err := q.CountUserNotifications(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("home: count notifications: %w", err)
	}
	items := []*jsonx.OrderedMap{}
	if total > 0 {
		rows, err := q.ListUserNotifications(ctx, store.ListUserNotificationsParams{
			UserID: user.ID, Limit: int32(perPage), Offset: int32(min((page-1)*perPage, math.MaxInt32)), //nolint:gosec // clamped
		})
		if err != nil {
			return fmt.Errorf("home: list notifications: %w", err)
		}
		for _, n := range rows {
			items = append(items, jsonx.Obj(
				"id", n.ID,
				"type", n.Type,
				"title", localized(n.Title, locale, def),
				"body", localizedNull(n.Body, locale, def),
				"action_url", nullString(n.ActionUrl),
				"is_read", n.ReadAt.Valid,
				"read_at", isoOrNil(n.ReadAt),
				"created_at", isoOrNil(n.CreatedAt),
			))
		}
	}
	unread, err := q.CountUnreadNotifications(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("home: unread notifications: %w", err)
	}
	lastPage := max(int64(math.Ceil(float64(total)/float64(perPage))), 1)
	return httpx.OK(c, jsonx.Obj(
		"unread_count", unread,
		"items", items,
		"pagination", jsonx.Obj("current_page", page, "last_page", lastPage, "per_page", perPage, "total", total),
	))
}

// MarkNotificationRead is POST /home/notifications/{notification}/read: another user's
// notification → abort(404).
func (h *Handlers) MarkNotificationRead(c fiber.Ctx) error {
	user, err := currentUser(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	q := h.deps.Queries
	raw := param(c, "notification")
	var row store.GetNotificationOwnerRow
	_, err = bindModel(raw, func(id uint64) (uint64, error) {
		r, err := q.GetNotificationOwner(ctx, id)
		row = r
		return r.ID, err
	})
	if err != nil {
		if errors.Is(err, errModelMissing) {
			return httpx.ModelNotFound(modelUserNotification, raw)
		}
		return fmt.Errorf("home: bind notification: %w", err)
	}
	if row.UserID != user.ID {
		return httpx.NotFound()
	}
	if !row.ReadAt.Valid {
		ts := sql.NullTime{Time: dbNow(h.now(c)), Valid: true}
		if err := q.MarkNotificationRead(ctx, store.MarkNotificationReadParams{ReadAt: ts, UpdatedAt: ts, ID: row.ID}); err != nil {
			return fmt.Errorf("home: mark notification read: %w", err)
		}
	}
	unread, err := q.CountUnreadNotifications(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("home: unread notifications: %w", err)
	}
	return httpx.OK(c, jsonx.Obj("unread_count", unread))
}

// MarkAllNotificationsRead is POST /home/notifications/read-all.
func (h *Handlers) MarkAllNotificationsRead(c fiber.Ctx) error {
	user, err := currentUser(c)
	if err != nil {
		return err
	}
	ts := sql.NullTime{Time: dbNow(h.now(c)), Valid: true}
	marked, err := h.deps.Queries.MarkAllNotificationsRead(c.Context(), store.MarkAllNotificationsReadParams{ReadAt: ts, UpdatedAt: ts, UserID: user.ID})
	if err != nil {
		return fmt.Errorf("home: mark all notifications read: %w", err)
	}
	return httpx.OK(c, jsonx.Obj("marked", marked))
}

// ---------------------------------------------------------------------------
// route parameters

// param is the decoded route parameter.
func param(c fiber.Ctx, name string) string {
	raw := c.Params(name)
	if s, err := url.PathUnescape(raw); err == nil {
		return s
	}
	return raw
}

var errModelMissing = errors.New("home: model not found")

// bindModel is implicit route-model binding (`where id = ?` with the raw string): MariaDB
// compares the integer key with the string's numeric value, so only a string whose numeric
// prefix is a positive integer can match. find returns sql.ErrNoRows when the row is missing.
func bindModel(raw string, find func(uint64) (uint64, error)) (uint64, error) {
	id, ok := mysqlKey(raw)
	if !ok {
		return 0, errModelMissing
	}
	got, err := find(id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errModelMissing
	}
	return got, err
}

// mysqlKey is the value MariaDB compares an integer column with for a string operand: the
// longest numeric prefix (leading spaces skipped), which must be a whole positive number.
func mysqlKey(raw string) (uint64, bool) {
	s := strings.TrimLeft(raw, " \t\n\r")
	end := 0
	if end < len(s) && (s[end] == '+' || s[end] == '-') {
		end++
	}
	digits := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end, digits = end+1, digits+1
	}
	if end < len(s) && s[end] == '.' {
		end++
		for end < len(s) && s[end] >= '0' && s[end] <= '9' {
			end, digits = end+1, digits+1
		}
	}
	if digits == 0 {
		return 0, false
	}
	if end < len(s) && (s[end] == 'e' || s[end] == 'E') {
		e := end + 1
		if e < len(s) && (s[e] == '+' || s[e] == '-') {
			e++
		}
		if e < len(s) && s[e] >= '0' && s[e] <= '9' {
			for e < len(s) && s[e] >= '0' && s[e] <= '9' {
				e++
			}
			end = e
		}
	}
	f, err := strconv.ParseFloat(s[:end], 64)
	if err != nil || f < 1 || f != math.Trunc(f) || f > math.MaxInt64 {
		return 0, false
	}
	return uint64(f), true
}

// phpIntCast is PHP's (int) cast of a query value.
func phpIntCast(v any) int64 {
	switch x := v.(type) {
	case nil:
		return 0
	case bool:
		if x {
			return 1
		}
		return 0
	case string:
		if phpval.IsNumericString(x) {
			f := phpval.ToFloat(x)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return 0
			}
			return int64(f)
		}
		s := strings.TrimLeft(x, " \t\n\r\v\f")
		end := 0
		if end < len(s) && (s[end] == '+' || s[end] == '-') {
			end++
		}
		for end < len(s) && s[end] >= '0' && s[end] <= '9' {
			end++
		}
		n, _ := strconv.ParseInt(s[:end], 10, 64)
		return n
	}
	if phpval.IsArray(v) {
		if phpval.Count(v) > 0 {
			return 1
		}
		return 0
	}
	return int64(phpval.ToFloat(v))
}
