package care

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Handlers are the /api/v1/care actions. Mount them behind the locale middleware and auth
// RequireUser.
type Handlers struct {
	q          *store.Queries
	clock      clock.Clock
	delegation Delegation // «ثبت برای …» (B-N4-02); nil = own records only
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(db store.DBTX, base clock.Clock) *Handlers {
	return &Handlers{q: store.New(db), clock: base}
}

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// now is the request time, Tehran wall-clock, whole seconds.
func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// Enums is GET /care/enums.
func (h *Handlers) Enums(c fiber.Ctx) error { return Enums(c) }

// ListMedications is GET /care/medications (?active=1: active only), newest first.
func (h *Handlers) ListMedications(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	var rows []store.Reminder
	if active, _ := validation.Query(c).Get("active"); phpval.Truthy(active) {
		rows, err = h.q.ListActiveMedications(c, userID)
	} else {
		rows, err = h.q.ListMedications(c, userID)
	}
	if err != nil {
		return fmt.Errorf("care: list medications: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		out = append(out, ParseMedication(r).JSON(i18n.Locale(c)))
	}
	return httpx.OK(c, out)
}

// ShowMedication is GET /care/medications/{id} (?for_user_id=: a companion with view on the owner's meds).
func (h *Handlers) ShowMedication(c fiber.Ctx) error {
	userID, actorID, err := h.subject(c, companion.SectionMeds, false)
	if err != nil {
		return err
	}
	m, err := h.find(c, userID)
	if err != nil {
		return err
	}
	if err := h.delegated(c, userID, actorID, companion.SectionMeds, false, false); err != nil {
		return err
	}
	return httpx.OK(c, m.JSON(i18n.Locale(c)))
}

// StoreMedication is POST /care/medications: 201 with the medication ({for_user_id}: a companion with edit on the
// owner's meds records it in the owner's list).
func (h *Handlers) StoreMedication(c fiber.Ctx) error {
	userID, actorID, err := h.subject(c, companion.SectionMeds, true)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	data := pickMedication(phpval.NewMap(), validation.Input(c))
	if v, _ := data.Get("starts_on"); v == nil {
		data.Set("starts_on", civildate.FromTime(now).String())
	}
	in, err := validateMedication(locale, data, now)
	if err != nil {
		return err
	}
	if err := h.checkMedicationCap(c, userID, locale); err != nil {
		return err
	}
	p, err := h.columns(c, userID, in, i18n.LanguagesOf(c).DefaultCode())
	if err != nil {
		return err
	}
	ts := sql.NullTime{Time: now, Valid: true}
	id, err := h.q.InsertMedication(c, store.InsertMedicationParams{
		UserID: userID, Title: p.Title, Subtitle: p.Subtitle, Notes: p.Notes,
		Recurrence: p.Recurrence, RecurrenceTime: p.RecurrenceTime, StartsOn: p.StartsOn, EndsOn: p.EndsOn,
		IsActive: p.IsActive, Meta: p.Meta, CreatedAt: ts, UpdatedAt: ts,
	})
	if err != nil || id <= 0 {
		return fmt.Errorf("care: insert medication (id %d): %w", id, err)
	}
	m, err := h.load(c, uint64(id), userID)
	if err != nil {
		return err
	}
	if err := h.delegated(c, userID, actorID, companion.SectionMeds, true, true); err != nil {
		return err
	}
	return httpx.Created(c, m.JSON(locale), T("messages.medication_created", locale))
}

// UpdateMedication is PUT /care/medications/{id}: a partial update (only the keys sent
// change; `{"is_active": false}` is the list switch). The merged medication is validated
// as a whole. {for_user_id}: a companion with edit on the owner's meds.
func (h *Handlers) UpdateMedication(c fiber.Ctx) error {
	userID, actorID, err := h.subject(c, companion.SectionMeds, true)
	if err != nil {
		return err
	}
	m, err := h.find(c, userID)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	body := validation.Input(c)
	if switchesOnly(body) {
		if err := h.updateSwitches(c, m, body, locale, now); err != nil {
			return err
		}
		return h.delegated(c, userID, actorID, companion.SectionMeds, true, false)
	}
	data := pickMedication(storedMedication(m), body)
	in, err := validateMedication(locale, data, now)
	if err != nil {
		return err
	}
	p, err := h.columns(c, userID, in, i18n.LanguagesOf(c).DefaultCode())
	if err != nil {
		return err
	}
	if err := h.q.UpdateMedication(c, store.UpdateMedicationParams{
		Title: p.Title, Subtitle: p.Subtitle, Notes: p.Notes, Recurrence: p.Recurrence,
		RecurrenceTime: p.RecurrenceTime, StartsOn: p.StartsOn, EndsOn: p.EndsOn, IsActive: p.IsActive,
		Meta: p.Meta, UpdatedAt: sql.NullTime{Time: now, Valid: true}, ID: m.Row.ID, UserID: userID,
	}); err != nil {
		return fmt.Errorf("care: update medication: %w", err)
	}
	fresh, err := h.load(c, m.Row.ID, userID)
	if err != nil {
		return err
	}
	if err := h.delegated(c, userID, actorID, companion.SectionMeds, true, false); err != nil {
		return err
	}
	return httpx.OK(c, fresh.JSON(locale), T("messages.medication_updated", locale))
}

// updateSwitches is a PUT that only flips is_active and/or notify: only those change, the
// rest of the row (including a legacy row's missing starts_on / times) is left as stored.
func (h *Handlers) updateSwitches(c fiber.Ctx, m Medication, body phpval.Map, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, body, switchRules(),
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	active, meta := m.Row.IsActive, m.Row.Meta
	if x, ok := body.Get("is_active"); ok {
		active = phpval.Truthy(x)
	}
	if x, ok := body.Get("notify"); ok {
		mm := m.Meta
		mm.Notify = phpval.Truthy(x)
		raw, err := json.Marshal(mm)
		if err != nil {
			return fmt.Errorf("care: encode meta: %w", err)
		}
		meta = rootdb.NullRawJSON{V: raw, Valid: true}
	}
	if err := h.q.UpdateMedicationSwitches(c, store.UpdateMedicationSwitchesParams{
		IsActive: active, Meta: meta, UpdatedAt: sql.NullTime{Time: now, Valid: true}, ID: m.Row.ID, UserID: m.Row.UserID,
	}); err != nil {
		return fmt.Errorf("care: update medication switches: %w", err)
	}
	fresh, err := h.load(c, m.Row.ID, m.Row.UserID)
	if err != nil {
		return err
	}
	return httpx.OK(c, fresh.JSON(locale), T("messages.medication_updated", locale))
}

// DestroyMedication is DELETE /care/medications/{id} (its intakes cascade).
func (h *Handlers) DestroyMedication(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c)
	}
	n, err := h.q.DeleteMedication(c, store.DeleteMedicationParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("care: delete medication: %w", err)
	}
	if n == 0 {
		return notFound(c)
	}
	return httpx.OK(c, nil, T("messages.medication_deleted", i18n.Locale(c)))
}

// TakeIntake is POST /care/medications/{id}/intakes {date, slot}: marks the dose taken.
// Idempotent: a repeat keeps the first taken_at. The slot must be one of the medication's
// times and the date inside its weekday/start/end window, and not in the future.
func (h *Handlers) TakeIntake(c fiber.Ctx) error {
	return h.intake(c, true)
}

// UntakeIntake is DELETE /care/medications/{id}/intakes {date, slot}: unticks the dose
// (idempotent; nothing to delete is fine).
func (h *Handlers) UntakeIntake(c fiber.Ctx) error {
	return h.intake(c, false)
}

func (h *Handlers) intake(c fiber.Ctx, tick bool) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	m, err := h.find(c, userID)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	v := validation.Make(lang.Default(), locale, validation.Input(c), intakeRules(tick),
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	attrs := v.Validated()
	rawDate, _ := attrs.Get("date")
	rawSlot, _ := attrs.Get("slot")
	day, err := civildate.Parse(phpval.ToString(rawDate))
	if err != nil {
		return fmt.Errorf("care: intake date: %w", err)
	}
	slot := phpval.ToString(rawSlot)
	key := store.GetIntakeParams{ReminderID: m.Row.ID, UserID: userID, IntakeDate: day, Slot: slot}

	if !tick {
		if _, err := h.q.DeleteIntake(c, store.DeleteIntakeParams(key)); err != nil {
			return fmt.Errorf("care: delete intake: %w", err)
		}
		return httpx.OK(c, intakeJSON(m.Row.ID, day, slot, sql.NullTime{}), T("messages.intake_untaken", locale))
	}

	if !slices.Contains(m.Meta.Times, slot) {
		return fieldError(locale, "slot", T("validation.slot_unknown", locale))
	}
	if !m.Covers(day) {
		return fieldError(locale, "date", T("validation.date_not_scheduled", locale))
	}
	ts := sql.NullTime{Time: now, Valid: true}
	if _, err := h.q.InsertIntake(c, store.InsertIntakeParams{
		UserID: userID, ReminderID: m.Row.ID, IntakeDate: day, Slot: slot, TakenAt: now,
		CreatedAt: ts, UpdatedAt: ts,
	}); err != nil {
		return fmt.Errorf("care: insert intake: %w", err)
	}
	row, err := h.q.GetIntake(c, key)
	if err != nil {
		return fmt.Errorf("care: reload intake: %w", err)
	}
	return httpx.OK(c, intakeJSON(m.Row.ID, day, slot, sql.NullTime{Time: row.TakenAt, Valid: true}),
		T("messages.intake_taken", locale))
}

func intakeJSON(reminderID uint64, day civildate.Date, slot string, takenAt sql.NullTime) *jsonx.OrderedMap {
	return jsonx.Obj(
		"reminder_id", reminderID,
		"date", day.String(),
		"slot", slot,
		"taken", takenAt.Valid,
		"taken_at", nullDateTime(takenAt),
	)
}

// find loads the user's medication {id}; another user's id, a non-medication reminder
// and a malformed id are all the same 404.
func (h *Handlers) find(c fiber.Ctx, userID uint64) (Medication, error) {
	id, ok := parseID(c.Params("id"))
	if !ok {
		return Medication{}, notFound(c)
	}
	return h.load(c, id, userID)
}

func (h *Handlers) load(c fiber.Ctx, id, userID uint64) (Medication, error) {
	row, err := h.q.GetMedication(c, store.GetMedicationParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Medication{}, notFound(c)
	}
	if err != nil {
		return Medication{}, fmt.Errorf("care: find medication: %w", err)
	}
	return ParseMedication(row), nil
}

func notFound(c fiber.Ctx) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.medication_not_found", i18n.Locale(c)))
}

// medicationColumns are the reminders columns of a validated medication.
type medicationColumns struct {
	Title          string
	Subtitle       sql.NullString
	Notes          sql.NullString
	Recurrence     string
	RecurrenceTime sql.NullString
	StartsOn       civildate.NullDate
	EndsOn         civildate.NullDate
	IsActive       bool
	Meta           rootdb.NullRawJSON
}

// columns derives the stored row: legacy subtitle (in subtitleLocale, the default language —
// /care renders its own subtitle from the meta at read time)/recurrence/recurrence_time from the meta,
// and ends_on from the duration (until_date: as sent; pregnancy_end: the active pregnancy's
// due date, else open-ended like ongoing; ongoing: NULL).
func (h *Handlers) columns(ctx context.Context, userID uint64, in medicationInput, subtitleLocale string) (medicationColumns, error) {
	meta, err := json.Marshal(in.Meta)
	if err != nil {
		return medicationColumns{}, fmt.Errorf("care: encode meta: %w", err)
	}
	p := medicationColumns{
		Title: in.Title, Subtitle: in.Meta.Subtitle(subtitleLocale), Recurrence: in.Meta.Recurrence(),
		RecurrenceTime: in.Meta.RecurrenceTime(), StartsOn: civildate.NullDate{Date: in.StartsOn, Valid: true},
		IsActive: in.IsActive, Meta: rootdb.NullRawJSON{V: meta, Valid: true},
	}
	if in.Notes != nil {
		p.Notes = sql.NullString{String: *in.Notes, Valid: true}
	}
	switch in.Meta.Duration {
	case DurationUntilDate:
		if in.EndsOn != nil {
			p.EndsOn = civildate.NullDate{Date: *in.EndsOn, Valid: true}
		}
	case DurationPregnancyEnd:
		due, err := h.q.ActivePregnancyDueDate(ctx, userID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return medicationColumns{}, fmt.Errorf("care: pregnancy due date: %w", err)
		}
		if err == nil && due.Valid {
			p.EndsOn = due
		}
	}
	return p, nil
}
