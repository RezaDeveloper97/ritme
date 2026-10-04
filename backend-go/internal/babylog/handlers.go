package babylog

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/children"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Error codes of the non-validation failures (the child ones are children's).
const (
	ErrorCodeFeedNotFound   = "feed_not_found"
	ErrorCodeSleepNotFound  = "sleep_not_found"
	ErrorCodeDiaperNotFound = "diaper_not_found"
	ErrorCodeFeedActive     = "feed_active"
	ErrorCodeSleepActive    = "sleep_active"
	ErrorCodeEnded          = "session_ended"
	ErrorCodeRunning        = "session_running"
	ErrorCodeNotBreast      = "not_breast_feed"
)

// Handlers are the /api/v1/children/{id}/{feeds,sleeps,diapers,baby-logs} actions. Mount them behind the locale
// middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// fail maps the domain errors: unknown / foreign child or row → uniform 404, shared child write → 403.
func fail(err error, locale string) error {
	f := func(status int, key, code string) error {
		return httpx.Fail(status, T("messages."+key, locale), "error_code", code)
	}
	switch {
	case errors.Is(err, children.ErrNotFound):
		return httpx.Fail(fiber.StatusNotFound, children.T("messages.not_found", locale), "error_code", children.ErrorCodeNotFound)
	case errors.Is(err, children.ErrReadOnly):
		return httpx.Fail(fiber.StatusForbidden, children.T("messages.read_only", locale), "error_code", children.ErrorCodeReadOnly)
	case errors.Is(err, ErrFeedNotFound):
		return f(fiber.StatusNotFound, "feed_not_found", ErrorCodeFeedNotFound)
	case errors.Is(err, ErrSleepNotFound):
		return f(fiber.StatusNotFound, "sleep_not_found", ErrorCodeSleepNotFound)
	case errors.Is(err, ErrDiaperNotFound):
		return f(fiber.StatusNotFound, "diaper_not_found", ErrorCodeDiaperNotFound)
	case errors.Is(err, ErrFeedActive):
		return f(fiber.StatusConflict, "feed_active", ErrorCodeFeedActive)
	case errors.Is(err, ErrSleepActive):
		return f(fiber.StatusConflict, "sleep_active", ErrorCodeSleepActive)
	case errors.Is(err, ErrEnded):
		return f(fiber.StatusConflict, "ended", ErrorCodeEnded)
	case errors.Is(err, ErrRunning):
		return f(fiber.StatusConflict, "running", ErrorCodeRunning)
	case errors.Is(err, ErrNotBreast):
		return f(fiber.StatusUnprocessableEntity, "not_breast", ErrorCodeNotBreast)
	}
	return err
}

func idParam(c fiber.Ctx, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return id, err == nil && id > 0
}

// child resolves :id for reading (edit = for writing; a shared child is 403).
func (h *Handlers) child(c fiber.Ctx, edit bool) (children.Access, error) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		return children.Access{}, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return children.Access{}, fail(children.ErrNotFound, locale)
	}
	var a children.Access
	var err error
	if edit {
		a, err = h.svc.Editable(c, userID, id, h.now(c))
	} else {
		a, err = h.svc.Access(c, userID, id, h.now(c))
	}
	if err != nil {
		return children.Access{}, fail(err, locale)
	}
	return a, nil
}

// row resolves the child for writing and the sub-resource id param (a malformed id is notFound).
func (h *Handlers) row(c fiber.Ctx, param string, notFound error) (children.Access, uint64, error) {
	a, err := h.child(c, true)
	if err != nil {
		return a, 0, err
	}
	id, ok := idParam(c, param)
	if !ok {
		return a, 0, fail(notFound, i18n.Locale(c))
	}
	return a, id, nil
}

// ---------------------------------------------------------------- feeds

// Feeds is GET /children/{id}/feeds?date=: the running feed, the last ended feed, the side to offer next, the day's
// totals and its feeds (newest first).
func (h *Handlers) Feeds(c fiber.Ctx) error {
	a, err := h.child(c, false)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	day, err := validateDay(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	rows, err := h.svc.Feeds(c, a.Child.ID, day)
	if err != nil {
		return err
	}
	act, err := h.svc.ActiveFeed(c, a.Child.ID)
	if err != nil {
		return err
	}
	last, err := h.svc.LastFeed(c, a.Child.ID)
	if err != nil {
		return err
	}
	items := make([]any, 0, len(rows))
	for _, f := range rows {
		items = append(items, FeedJSON(f, now))
	}
	side, err := h.svc.NextSide(c, a.Child.ID)
	if err != nil {
		return err
	}
	var next any
	if side != "" {
		next = side
	}
	sum := Summarize(day, 1, rows, nil, nil, now)[0]
	return httpx.OK(c, jsonx.Obj(
		"date", day.String(),
		"active", feedNull(act, now),
		"last", feedNull(last, now),
		"next_side", next,
		"summary", FeedTotalsJSON(sum.Feeds),
		"items", items,
	))
}

// StartFeed is POST /children/{id}/feeds/start {type, side?} (201; 409 feed_active while one runs).
func (h *Handlers) StartFeed(c fiber.Ctx) error {
	a, err := h.child(c, true)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := validateStartFeed(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	f, err := h.svc.StartFeed(c, a, in.Type, in.Side, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, FeedJSON(f, now))
}

// StoreFeed is POST /children/{id}/feeds: an ended feed entered by hand (201).
func (h *Handlers) StoreFeed(c fiber.Ctx) error {
	a, err := h.child(c, true)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := validateFeed(validation.Input(c), a.Child.BirthDate, locale, now)
	if err != nil {
		return err
	}
	f, err := h.svc.CreateFeed(c, a, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, FeedJSON(f, now), T("messages.saved", locale))
}

// FeedSide is POST /children/{id}/feeds/{fid}/side {side: left|right|null}: switch or pause the breast timer.
func (h *Handlers) FeedSide(c fiber.Ctx) error {
	a, id, err := h.row(c, "fid", ErrFeedNotFound)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	side, err := validateSide(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	f, err := h.svc.SetSide(c, a.Child.ID, id, side, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, FeedJSON(f, now))
}

// StopFeed is POST /children/{id}/feeds/{fid}/stop {amount_ml?, note?} («پایان و ذخیره»).
func (h *Handlers) StopFeed(c fiber.Ctx) error {
	a, id, err := h.row(c, "fid", ErrFeedNotFound)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := validateStopFeed(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	f, err := h.svc.StopFeed(c, a, id, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, FeedJSON(f, now), T("messages.saved", locale))
}

// UpdateFeed is PUT /children/{id}/feeds/{fid}: replaces an ended feed (409 session_running for the running one).
func (h *Handlers) UpdateFeed(c fiber.Ctx) error {
	a, id, err := h.row(c, "fid", ErrFeedNotFound)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	if _, err := h.svc.Feed(c, a.Child.ID, id); err != nil {
		return fail(err, locale)
	}
	in, err := validateFeed(validation.Input(c), a.Child.BirthDate, locale, now)
	if err != nil {
		return err
	}
	f, err := h.svc.UpdateFeed(c, a, id, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, FeedJSON(f, now), T("messages.saved", locale))
}

// DestroyFeed is DELETE /children/{id}/feeds/{fid} (a running feed is discarded).
func (h *Handlers) DestroyFeed(c fiber.Ctx) error {
	a, id, err := h.row(c, "fid", ErrFeedNotFound)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.DeleteFeed(c, a, id, h.now(c)); err != nil {
		return fail(err, locale)
	}
	return httpx.Message(c, T("messages.deleted", locale))
}

// ---------------------------------------------------------------- sleep

// Sleeps is GET /children/{id}/sleeps?date=: the running sleep, the day's totals and the sleeps overlapping it.
func (h *Handlers) Sleeps(c fiber.Ctx) error {
	a, err := h.child(c, false)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	day, err := validateDay(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	rows, err := h.svc.Sleeps(c, a.Child.ID, day)
	if err != nil {
		return err
	}
	act, err := h.svc.ActiveSleep(c, a.Child.ID)
	if err != nil {
		return err
	}
	items := make([]any, 0, len(rows))
	for _, s := range rows {
		items = append(items, SleepJSON(s, now))
	}
	sum := Summarize(day, 1, nil, rows, nil, now)[0]
	return httpx.OK(c, jsonx.Obj(
		"date", day.String(),
		"active", sleepNull(act, now),
		"summary", SleepTotalsJSON(sum.Sleep),
		"items", items,
	))
}

// StartSleep is POST /children/{id}/sleeps/start (201; 409 sleep_active while one runs).
func (h *Handlers) StartSleep(c fiber.Ctx) error {
	a, err := h.child(c, true)
	if err != nil {
		return err
	}
	now := h.now(c)
	s, err := h.svc.StartSleep(c, a.Child.ID, now)
	if err != nil {
		return fail(err, i18n.Locale(c))
	}
	return httpx.Created(c, SleepJSON(s, now))
}

// StoreSleep is POST /children/{id}/sleeps: an ended sleep entered by hand (201).
func (h *Handlers) StoreSleep(c fiber.Ctx) error {
	a, err := h.child(c, true)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := validateSleep(validation.Input(c), a.Child.BirthDate, locale, now)
	if err != nil {
		return err
	}
	s, err := h.svc.CreateSleep(c, a.Child.ID, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, SleepJSON(s, now), T("messages.saved", locale))
}

// StopSleep is POST /children/{id}/sleeps/{sid}/stop {note?}.
func (h *Handlers) StopSleep(c fiber.Ctx) error {
	a, id, err := h.row(c, "sid", ErrSleepNotFound)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	note, err := validateNote(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	s, err := h.svc.StopSleep(c, a.Child.ID, id, note, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, SleepJSON(s, now), T("messages.saved", locale))
}

// UpdateSleep is PUT /children/{id}/sleeps/{sid}: replaces an ended sleep.
func (h *Handlers) UpdateSleep(c fiber.Ctx) error {
	a, id, err := h.row(c, "sid", ErrSleepNotFound)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	if _, err := h.svc.Sleep(c, a.Child.ID, id); err != nil {
		return fail(err, locale)
	}
	in, err := validateSleep(validation.Input(c), a.Child.BirthDate, locale, now)
	if err != nil {
		return err
	}
	s, err := h.svc.UpdateSleep(c, a.Child.ID, id, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, SleepJSON(s, now), T("messages.saved", locale))
}

// DestroySleep is DELETE /children/{id}/sleeps/{sid}.
func (h *Handlers) DestroySleep(c fiber.Ctx) error {
	a, id, err := h.row(c, "sid", ErrSleepNotFound)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.DeleteSleep(c, a.Child.ID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.Message(c, T("messages.deleted", locale))
}

// ---------------------------------------------------------------- diapers

// Diapers is GET /children/{id}/diapers?date=: the day's totals and changes (newest first).
func (h *Handlers) Diapers(c fiber.Ctx) error {
	a, err := h.child(c, false)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	day, err := validateDay(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	rows, err := h.svc.Diapers(c, a.Child.ID, day)
	if err != nil {
		return err
	}
	items := make([]any, 0, len(rows))
	for _, d := range rows {
		items = append(items, DiaperJSON(d))
	}
	sum := Summarize(day, 1, nil, nil, rows, now)[0]
	return httpx.OK(c, jsonx.Obj("date", day.String(), "summary", DiaperTotalsJSON(sum.Diapers), "items", items))
}

// StoreDiaper is POST /children/{id}/diapers {kind, changed_at?, note?} (201).
func (h *Handlers) StoreDiaper(c fiber.Ctx) error {
	a, err := h.child(c, true)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := validateDiaper(validation.Input(c), a.Child.BirthDate, locale, now)
	if err != nil {
		return err
	}
	d, err := h.svc.CreateDiaper(c, a.Child.ID, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, DiaperJSON(d), T("messages.saved", locale))
}

// UpdateDiaper is PUT /children/{id}/diapers/{did}.
func (h *Handlers) UpdateDiaper(c fiber.Ctx) error {
	a, id, err := h.row(c, "did", ErrDiaperNotFound)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	if _, err := h.svc.Diaper(c, a.Child.ID, id); err != nil {
		return fail(err, locale)
	}
	in, err := validateDiaper(validation.Input(c), a.Child.BirthDate, locale, now)
	if err != nil {
		return err
	}
	d, err := h.svc.UpdateDiaper(c, a.Child.ID, id, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, DiaperJSON(d), T("messages.saved", locale))
}

// DestroyDiaper is DELETE /children/{id}/diapers/{did}.
func (h *Handlers) DestroyDiaper(c fiber.Ctx) error {
	a, id, err := h.row(c, "did", ErrDiaperNotFound)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.DeleteDiaper(c, a.Child.ID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.Message(c, T("messages.deleted", locale))
}

// ---------------------------------------------------------------- summary

// Summary is GET /children/{id}/baby-logs?days=: today's card, the last `days` days (oldest first, today included)
// and their per-day averages with the breast side split (nbl_An_Hub_Post).
func (h *Handlers) Summary(c fiber.Ctx) error {
	a, err := h.child(c, false)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	n, err := validateDays(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	today := civildate.InTehran(now)
	from := today.AddDays(1 - n)
	days, err := h.svc.Days(c, a.Child.ID, from, n, now)
	if err != nil {
		return err
	}
	card, err := h.svc.Today(c, a.Child.ID, now)
	if err != nil {
		return err
	}
	list := make([]any, 0, len(days))
	for _, d := range days {
		list = append(list, DayJSON(d))
	}
	return httpx.OK(c, jsonx.Obj(
		"from", from.String(),
		"to", today.String(),
		"today", card,
		"days", list,
		"averages", AveragesJSON(days),
	))
}
