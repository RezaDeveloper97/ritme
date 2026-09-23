package service

import (
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Handlers are the CycleCalculationController actions. Mount them behind auth RequireUser and
// the locale middleware.
type Handlers struct {
	svc   *Service
	db    store.DBTX
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (clock.Middleware's request clock
// wins).
func NewHandlers(svc *Service, db store.DBTX, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, db: db, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time { return clock.FromContext(c, h.clock).Now() }

func userID(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// Today is GET /cycle/today.
func (h *Handlers) Today(c fiber.Ctx) error {
	now := h.now(c)
	today := civildate.InTehran(now)
	return h.day(c, today, today)
}

// ForDate is GET /cycle/date/{date}: Carbon::parse, 422 when it fails.
func (h *Handlers) ForDate(c fiber.Ctx) error {
	now := h.now(c)
	raw := c.Params("date")
	if s, err := url.PathUnescape(raw); err == nil {
		raw = s
	}
	t, err := civildate.ParseLenient(raw, now, civildate.Tehran)
	if err != nil {
		return httpx.Fail(fiber.StatusUnprocessableEntity, "Invalid date format. Use YYYY-MM-DD.")
	}
	return h.day(c, civildate.FromTime(t), civildate.InTehran(now))
}

func (h *Handlers) day(c fiber.Ctx, date, today civildate.Date) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	p, err := h.svc.DayJSON(c.Context(), uid, date, today, i18n.ResolveLocale(c, ""))
	if err != nil {
		return err
	}
	var parts struct {
		Calculation json.RawMessage `json:"calculation"`
		CycleView   json.RawMessage `json:"cycle_view"`
	}
	if err := json.Unmarshal(p.JSON, &parts); err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj(
		"calculation", parts.Calculation,
		"cycle_view", parts.CycleView,
		"calculation_status", p.Status,
		"is_recalculating", p.IsRecalculating,
	))
}

// monthViews are CycleCalculationController::MONTH_VIEWS.
var monthViews = []string{"full", "calendar"}

// Month is GET /cycle/month/{year}/{month}?view=full|calendar (JSON_UNESCAPED_UNICODE).
func (h *Handlers) Month(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	// int route parameters: a non-numeric segment is a 404 in Go (D-02; Laravel 500s).
	year, err1 := strconv.Atoi(c.Params("year"))
	month, err2 := strconv.Atoi(c.Params("month"))
	if err1 != nil || err2 != nil {
		return httpx.NotFound()
	}
	if month < 1 || month > 12 {
		return httpx.Fail(fiber.StatusUnprocessableEntity, "Invalid month. Must be between 1 and 12.")
	}
	// $request->query('view', 'full'): present-but-empty (null) or an array is not a view either.
	viewName := "full"
	if v, present := validation.Query(c).Get("view"); present {
		s, _ := v.(string)
		viewName = s
	}
	if !slices.Contains(monthViews, viewName) {
		return httpx.Fail(fiber.StatusUnprocessableEntity, "Invalid view. Must be one of: full, calendar.")
	}
	now := h.now(c)
	p, err := h.svc.MonthJSON(c.Context(), uid, year, month, viewName, civildate.InTehran(now), i18n.ResolveLocale(c, ""))
	if err != nil {
		return err
	}
	var parts struct {
		Calculations json.RawMessage `json:"calculations"`
		MonthSummary json.RawMessage `json:"month_summary"`
	}
	if err := json.Unmarshal(p.JSON, &parts); err != nil {
		return err
	}
	body := httpx.Envelope(jsonx.Obj(
		"calculations", parts.Calculations,
		"calculation_status", p.Status,
		"is_recalculating", p.IsRecalculating,
		"month_summary", parts.MonthSummary,
	))
	return httpx.Send(c, fiber.StatusOK, body, jsonx.UnescapedUnicode)
}

// Status is GET /cycle/status.
func (h *Handlers) Status(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	locale := i18n.ResolveLocale(c, "")
	p, err := h.svc.Profile(c.Context(), uid)
	if err != nil {
		return err
	}
	if p == nil {
		return httpx.OK(c, jsonx.Obj(
			"status", string(enums.CalculationStatusPending),
			"status_label", enums.CalculationStatusPending.Label(locale),
			"is_processing", false,
			"version", 0,
			"started_at", nil,
			"completed_at", nil,
		))
	}
	status := enums.CalculationStatus(p.CalculationStatus)
	if !status.IsValid() {
		status = enums.CalculationStatusPending
	}
	return httpx.OK(c, jsonx.Obj(
		"status", string(status),
		"status_label", status.Label(locale),
		"is_processing", status == enums.CalculationStatusProcessing,
		"version", p.CalculationVersion,
		"started_at", nullDateTime(p.CalculationStartedAt.Time, p.CalculationStartedAt.Valid),
		"completed_at", nullDateTime(p.CalculationCompletedAt.Time, p.CalculationCompletedAt.Valid),
	))
}

func nullDateTime(t time.Time, valid bool) any {
	if !valid {
		return nil
	}
	return jsonx.DateTime(t)
}

// Recalculate is POST /cycle/recalculate: markRecalculated, 400 without a profile.
func (h *Handlers) Recalculate(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	locale := i18n.ResolveLocale(c, "")
	p, err := h.svc.Profile(c.Context(), uid)
	if err != nil {
		return err
	}
	if p == nil {
		msg := "Please complete your profile first"
		if locale == "fa" {
			msg = "لطفاً اول پروفایل خود را تکمیل کنید"
		}
		return httpx.Fail(fiber.StatusBadRequest, msg)
	}
	if err := markRecalculated(c.Context(), h.db, p.ID, h.now(c)); err != nil {
		return err
	}
	msg := "Recalculation completed"
	if locale == "fa" {
		msg = "محاسبه مجدد انجام شد"
	}
	// markRecalculated() returns the in-memory attribute after increment(): loaded + 1.
	return httpx.OK(c, jsonx.Obj(
		"version", int64(p.CalculationVersion)+1,
		"status", string(enums.CalculationStatusCompleted),
	), msg)
}

// markRecalculated is UserProfile::markRecalculated() (profile.MarkRecalculated).
func markRecalculated(ctx context.Context, db store.DBTX, profileID uint64, now time.Time) error {
	return profile.MarkRecalculated(ctx, profilestore.New(db), profileID, now)
}
