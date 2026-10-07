package care

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Appointment list scopes.
const (
	ScopeUpcoming = "upcoming"
	ScopePast     = "past"
	ScopeAll      = "all"
)

// AppointmentScopes are the ?scope= values of GET /care/appointments.
var AppointmentScopes = []string{ScopeUpcoming, ScopePast, ScopeAll}

// ListAppointments is GET /care/appointments?scope=upcoming|past|all (default upcoming).
// upcoming: scheduled (not cancelled) and at or after now, soonest first; past: everything
// else (earlier, or cancelled), latest first; all: every appointment, soonest first.
func (h *Handlers) ListAppointments(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	scope := ScopeUpcoming
	if s, _ := validation.Query(c).Get("scope"); s != nil && phpval.ToString(s) != "" {
		scope = phpval.ToString(s)
	}
	if !slices.Contains(AppointmentScopes, scope) {
		return fieldError(locale, "scope", T("validation.scope_invalid", locale))
	}
	rows, err := h.q.ListAppointments(c, userID)
	if err != nil {
		return fmt.Errorf("care: list appointments: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		a := ParseAppointment(r)
		switch scope {
		case ScopeUpcoming:
			if !a.Upcoming(now) {
				continue
			}
		case ScopePast:
			if a.Upcoming(now) {
				continue
			}
		}
		out = append(out, a.JSON(now))
	}
	if scope == ScopePast {
		slices.Reverse(out)
	}
	return httpx.OK(c, out)
}

// ShowAppointment is GET /care/appointments/{id} (?for_user_id=: a companion with view on the owner's appointments).
func (h *Handlers) ShowAppointment(c fiber.Ctx) error {
	userID, actorID, err := h.subject(c, companion.SectionAppointments, false)
	if err != nil {
		return err
	}
	a, err := h.findAppointment(c, userID)
	if err != nil {
		return err
	}
	if a.HiddenFrom(userID, actorID) || outsideAppointmentView(a, userID, actorID, h.now(c)) { // CB-LOSS-01, CMP-M2
		return appointmentNotFound(c)
	}
	if err := h.delegated(c, userID, actorID, companion.SectionAppointments, false, false); err != nil {
		return err
	}
	return httpx.OK(c, a.JSON(h.now(c)))
}

// StoreAppointment is POST /care/appointments: 201 with the appointment ({for_user_id}: a companion with edit on
// the owner's appointments records it in the owner's list).
func (h *Handlers) StoreAppointment(c fiber.Ctx) error {
	userID, actorID, err := h.subject(c, companion.SectionAppointments, true)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	data := pickAppointment(phpval.NewMap(), validation.Input(c))
	in, err := validateAppointment(locale, data, now, nil, StatusScheduled)
	if err != nil {
		return err
	}
	if err := h.checkAppointmentCap(c, userID, in.ScheduledAt, now, locale); err != nil {
		return err
	}
	meta, err := json.Marshal(in.Meta)
	if err != nil {
		return fmt.Errorf("care: encode appointment meta: %w", err)
	}
	ts := sql.NullTime{Time: now, Valid: true}
	id, err := h.q.InsertAppointment(c, store.InsertAppointmentParams{
		UserID: userID, Title: in.Title, Subtitle: in.Meta.Subtitle(), Notes: nullStr(in.Notes),
		ScheduledAt: sql.NullTime{Time: in.ScheduledAt, Valid: true}, IsActive: in.IsActive,
		Meta: rootdb.NullRawJSON{V: meta, Valid: true}, CreatedAt: ts, UpdatedAt: ts,
	})
	if err != nil || id <= 0 {
		return fmt.Errorf("care: insert appointment (id %d): %w", id, err)
	}
	a, err := h.loadAppointment(c, uint64(id), userID)
	if err != nil {
		return err
	}
	if err := h.delegated(c, userID, actorID, companion.SectionAppointments, true, true); err != nil {
		return err
	}
	return httpx.Created(c, a.JSON(now), T("messages.appointment_created", locale))
}

// UpdateAppointment is PUT /care/appointments/{id}: a partial update (only the keys sent
// change; `{"is_active": false}` is the reminder switch, `{"prep": […]}` the checklist).
// The merged appointment is validated as a whole; the status is kept. {for_user_id}: a companion with edit on the
// owner's appointments.
func (h *Handlers) UpdateAppointment(c fiber.Ctx) error {
	userID, actorID, err := h.subject(c, companion.SectionAppointments, true)
	if err != nil {
		return err
	}
	a, err := h.findAppointment(c, userID)
	if err != nil {
		return err
	}
	if a.HiddenFrom(userID, actorID) || outsideAppointmentView(a, userID, actorID, h.now(c)) { // CB-LOSS-01, CMP-M2
		return appointmentNotFound(c)
	}
	locale, now := i18n.Locale(c), h.now(c)
	body := validation.Input(c)
	if userID != actorID {
		body.Delete("prep") // the prep checklist stays owner-only (delegation.go)
	}
	data := pickAppointment(storedAppointment(a, locale), body)
	in, err := validateAppointment(locale, data, now, a.Meta.Prep, a.Meta.Status)
	if err != nil {
		return err
	}
	in.Meta.Private = a.Meta.Private // never changed by a request (CB-LOSS-01)
	in.Meta.BookingID = a.Meta.BookingID // set by a telemed booking only (B-N7-03)
	// A cancelled visit keeps its reminder off (D-29); the other fields stay editable.
	if on, sent := body.Get("is_active"); sent && a.Meta.Status == StatusCancelled && phpval.Truthy(on) {
		return fieldError(locale, "is_active", CancelledReminderMessage(locale))
	}
	fresh, err := h.saveAppointment(c, a.Row.ID, userID, in, now)
	if err != nil {
		return err
	}
	if err := h.delegated(c, userID, actorID, companion.SectionAppointments, true, false); err != nil {
		return err
	}
	return httpx.OK(c, fresh.JSON(now), T("messages.appointment_updated", locale))
}

// DestroyAppointment is DELETE /care/appointments/{id}.
func (h *Handlers) DestroyAppointment(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return appointmentNotFound(c)
	}
	n, err := h.q.DeleteAppointment(c, store.DeleteAppointmentParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("care: delete appointment: %w", err)
	}
	if n == 0 {
		return appointmentNotFound(c)
	}
	return httpx.OK(c, nil, T("messages.appointment_deleted", i18n.Locale(c)))
}

// CancelAppointment is POST /care/appointments/{id}/cancel: status=cancelled and the
// reminder off (is_active=false); the row is kept for history. Idempotent.
func (h *Handlers) CancelAppointment(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	a, err := h.findAppointment(c, userID)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in := inputOf(a)
	in.Meta.Status = StatusCancelled
	in.IsActive = false
	fresh, err := h.saveAppointment(c, a.Row.ID, userID, in, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, fresh.JSON(now), T("messages.appointment_cancelled", locale))
}

// TogglePrepItem is PATCH /care/appointments/{id}/prep/{itemId} {done}: ticks or unticks
// one checklist item; an unknown item is a 404.
func (h *Handlers) TogglePrepItem(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	a, err := h.findAppointment(c, userID)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	v := validation.Make(lang.Default(), locale, validation.Input(c), prepRules(),
		validation.Now(now), validation.Attributes(attributesOf("appointment_attributes", locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	done, _ := v.Validated().Get("done")
	in := inputOf(a)
	itemID := c.Params("itemId")
	i := slices.IndexFunc(in.Meta.Prep, func(p PrepItem) bool { return p.ID == itemID })
	if i < 0 {
		return httpx.Fail(fiber.StatusNotFound, T("messages.prep_item_not_found", locale))
	}
	in.Meta.Prep[i].Done = phpval.Truthy(done)
	fresh, err := h.saveAppointment(c, a.Row.ID, userID, in, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, fresh.JSON(now), T("messages.appointment_updated", locale))
}

// inputOf is the stored appointment as an input (cancel and the prep toggle change one field).
func inputOf(a Appointment) appointmentInput {
	in := appointmentInput{Title: a.Row.Title, IsActive: a.Row.IsActive, Meta: a.Meta}
	in.Meta.V = MetaVersion
	in.Meta.Prep = slices.Clone(a.Meta.Prep)
	if a.Row.Notes.Valid {
		n := a.Row.Notes.String
		in.Notes = &n
	}
	if a.Row.ScheduledAt.Valid {
		in.ScheduledAt = a.Row.ScheduledAt.Time
	}
	return in
}

// saveAppointment writes in over the row and reloads it.
func (h *Handlers) saveAppointment(c fiber.Ctx, id, userID uint64, in appointmentInput, now time.Time) (Appointment, error) {
	meta, err := json.Marshal(in.Meta)
	if err != nil {
		return Appointment{}, fmt.Errorf("care: encode appointment meta: %w", err)
	}
	if err := h.q.UpdateAppointment(c, store.UpdateAppointmentParams{
		Title: in.Title, Subtitle: in.Meta.Subtitle(), Notes: nullStr(in.Notes),
		ScheduledAt: sql.NullTime{Time: in.ScheduledAt, Valid: !in.ScheduledAt.IsZero()}, IsActive: in.IsActive,
		Meta: rootdb.NullRawJSON{V: meta, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true},
		ID: id, UserID: userID,
	}); err != nil {
		return Appointment{}, fmt.Errorf("care: update appointment: %w", err)
	}
	return h.loadAppointment(c, id, userID)
}

// findAppointment loads the user's appointment {id}; another user's id, a non-appointment
// reminder and a malformed id are all the same 404.
func (h *Handlers) findAppointment(c fiber.Ctx, userID uint64) (Appointment, error) {
	id, ok := parseID(c.Params("id"))
	if !ok {
		return Appointment{}, appointmentNotFound(c)
	}
	return h.loadAppointment(c, id, userID)
}

func (h *Handlers) loadAppointment(c fiber.Ctx, id, userID uint64) (Appointment, error) {
	row, err := h.q.GetAppointment(c, store.GetAppointmentParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Appointment{}, appointmentNotFound(c)
	}
	if err != nil {
		return Appointment{}, fmt.Errorf("care: find appointment: %w", err)
	}
	return ParseAppointment(row), nil
}

func appointmentNotFound(c fiber.Ctx) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.appointment_not_found", i18n.Locale(c)))
}

func nullStr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
