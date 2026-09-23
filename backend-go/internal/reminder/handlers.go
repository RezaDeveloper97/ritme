package reminder

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/reminder/store"
)

// recurrences are the recurrence values with ReminderController::enums' hard-coded labels.
// Like Laravel, a locale without a label (anything but fa / en) fails the request with a
// 500 (D-01 proposes an English fallback; not approved, so the 500 is kept).
var recurrences = []struct {
	value  string
	labels map[string]string
}{
	{"none", map[string]string{"fa": "یک\u200cبار", "en": "One-off"}},
	{"daily", map[string]string{"fa": "روزانه", "en": "Daily"}},
	{"weekly", map[string]string{"fa": "هفتگی", "en": "Weekly"}},
	{"monthly", map[string]string{"fa": "ماهانه", "en": "Monthly"}},
}

func recurrenceValues() []string {
	out := make([]string, 0, len(recurrences))
	for _, r := range recurrences {
		out = append(out, r.value)
	}
	return out
}

// Handlers are the ReminderController actions. Mount them behind auth RequireUser.
type Handlers struct {
	q     *store.Queries
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock.
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

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// Enums is GET /reminders/enums.
func (h *Handlers) Enums(c fiber.Ctx) error {
	locale := i18n.ResolveLocale(c, "")
	types := make([]*jsonx.OrderedMap, 0, 4)
	for _, t := range enums.ReminderTypeCases() {
		types = append(types, jsonx.Obj("value", string(t), "label", t.Label(locale), "icon", t.Icon()))
	}
	recs := make([]*jsonx.OrderedMap, 0, len(recurrences))
	for _, r := range recurrences {
		label, ok := r.labels[locale]
		if !ok { // $recurrenceLabels[$r][$locale]: undefined array key → 500
			return fmt.Errorf("reminder: no %q label for recurrence %q", locale, r.value)
		}
		recs = append(recs, jsonx.Obj("value", r.value, "label", label))
	}
	return httpx.OK(c, jsonx.Obj("types", types, "recurrences", recs))
}

// Index is GET /reminders (?type= filters when truthy).
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	var rows []store.Reminder
	t, _ := validation.Query(c).Get("type")
	if s, ok := t.(string); ok && phpval.Truthy(s) {
		rows, err = h.q.ListRemindersByType(c.Context(), store.ListRemindersByTypeParams{UserID: userID, Type: s})
	} else {
		rows, err = h.q.ListReminders(c.Context(), userID)
	}
	if err != nil {
		return fmt.Errorf("reminder: list: %w", err)
	}
	return httpx.OK(c, List(rows))
}

// Store is POST /reminders: 201 with the created model (only the attributes it was created
// with, plus user_id, timestamps and id).
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	attrs, err := h.validate(c, true)
	if err != nil {
		return err
	}
	if _, ok := attrs.Get("is_active"); !ok { // $validated + ['is_active' => true]
		attrs.Set("is_active", true)
	}
	now := h.now(c)
	row := store.Reminder{UserID: userID, Recurrence: "none"} // recurrence: column default
	data := jsonx.NewObject()
	for _, k := range attrs.Keys() {
		v, _ := attrs.Get(k)
		cast, _, err := assign(&row, k, v, now)
		if err != nil {
			return err
		}
		data.Set(k, cast)
	}
	ts := sql.NullTime{Time: now, Valid: true}
	id, err := h.q.InsertReminder(c.Context(), store.InsertReminderParams{
		UserID: userID, Type: row.Type, Title: row.Title, Subtitle: row.Subtitle, Notes: row.Notes,
		ScheduledAt: row.ScheduledAt, Recurrence: row.Recurrence, RecurrenceTime: row.RecurrenceTime,
		StartsOn: row.StartsOn, EndsOn: row.EndsOn, IsActive: row.IsActive, CreatedAt: ts, UpdatedAt: ts,
	})
	if err != nil {
		return fmt.Errorf("reminder: insert: %w", err)
	}
	data.Set("user_id", userID)
	data.Set("updated_at", dateTimeValue(now))
	data.Set("created_at", dateTimeValue(now))
	data.Set("id", id)
	return httpx.Created(c, data)
}

// Update is PUT /reminders/{id}: `sometimes` rules, then the fresh row.
func (h *Handlers) Update(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	row, err := h.find(c, userID)
	if err != nil {
		return err
	}
	attrs, err := h.validate(c, false)
	if err != nil {
		return err
	}
	now := h.now(c)
	dirty := false
	for _, k := range attrs.Keys() {
		v, _ := attrs.Get(k)
		_, changed, err := assign(&row, k, v, now)
		if err != nil {
			return err
		}
		dirty = dirty || changed
	}
	if dirty { // save() is a no-op (updated_at kept) when nothing changed
		if err := h.q.UpdateReminder(c.Context(), store.UpdateReminderParams{
			Type: row.Type, Title: row.Title, Subtitle: row.Subtitle, Notes: row.Notes,
			ScheduledAt: row.ScheduledAt, Recurrence: row.Recurrence, RecurrenceTime: row.RecurrenceTime,
			StartsOn: row.StartsOn, EndsOn: row.EndsOn, IsActive: row.IsActive,
			UpdatedAt: sql.NullTime{Time: now, Valid: true}, ID: row.ID,
		}); err != nil {
			return fmt.Errorf("reminder: update: %w", err)
		}
	}
	fresh, err := h.q.GetUserReminder(c.Context(), store.GetUserReminderParams{ID: row.ID, UserID: userID})
	if err != nil {
		return fmt.Errorf("reminder: reload: %w", err)
	}
	return httpx.OK(c, JSON(fresh))
}

// Destroy is DELETE /reminders/{id}: {"success":true}.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	row, err := h.find(c, userID)
	if err != nil {
		return err
	}
	if err := h.q.DeleteReminder(c.Context(), row.ID); err != nil {
		return fmt.Errorf("reminder: delete: %w", err)
	}
	return httpx.JSON(c, fiber.StatusOK, jsonx.Obj("success", true))
}

// find is $request->user()->reminders()->find($id) with the `int $id` route parameter.
func (h *Handlers) find(c fiber.Ctx, userID uint64) (store.Reminder, error) {
	id, ok := intParam(c.Params("id"))
	if !ok { // TypeError on the typed controller argument → 500 (D-02 proposes 404; not approved)
		return store.Reminder{}, httpx.ServerError()
	}
	if id <= 0 {
		return store.Reminder{}, h.notFound(c)
	}
	row, err := h.q.GetUserReminder(c.Context(), store.GetUserReminderParams{ID: uint64(id), UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return store.Reminder{}, h.notFound(c)
	}
	if err != nil {
		return store.Reminder{}, fmt.Errorf("reminder: find: %w", err)
	}
	return row, nil
}

func (h *Handlers) notFound(c fiber.Ctx) error {
	msg := "Reminder not found"
	if i18n.ResolveLocale(c, "") == "fa" {
		msg = "یادآور پیدا نشد"
	}
	return httpx.Fail(fiber.StatusNotFound, msg)
}

// intParam is PHP's coercion of a route string to an `int` parameter: numeric strings
// (surrounding whitespace allowed, fractions truncated) convert, anything else is a
// TypeError (ok=false).
func intParam(raw string) (int64, bool) {
	if !phpval.IsNumericString(raw) {
		return 0, false
	}
	s := strings.TrimSpace(raw)
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(f, 0) || f >= math.MaxInt64 || f < math.MinInt64 {
		return 0, false
	}
	return int64(f), true
}

// rules is ReminderController::rules($required).
func rules(required bool) validation.Rules {
	presence := "sometimes"
	if required {
		presence = "required"
	}
	F := validation.F
	return validation.Rules{
		F("type", presence, validation.In(enums.ReminderTypeValues()...)),
		F("title", presence, "string", "max:255"),
		F("subtitle", "nullable", "string", "max:255"),
		F("notes", "nullable", "string", "max:2000"),
		F("scheduled_at", "nullable", "date"),
		F("recurrence", "sometimes", validation.In(recurrenceValues()...)),
		F("recurrence_time", "nullable", "date_format:H:i"),
		F("starts_on", "nullable", "date"),
		F("ends_on", "nullable", "date", "after_or_equal:starts_on"),
		F("is_active", "sometimes", "boolean"),
	}
}

// validate is Validator::make($request->all(), rules) with the controller's own 422 body
// {success:false, message (fa|en), errors}.
func (h *Handlers) validate(c fiber.Ctx, required bool) (phpval.Map, error) {
	v := validation.Make(lang.Default(), i18n.Locale(c), validation.Input(c), rules(required),
		validation.Now(clock.FromContext(c, h.clock).Now()))
	if !v.Fails() {
		return v.Validated(), nil
	}
	errs, _ := v.Errors().Body().Get("errors")
	msg := "Validation failed"
	if i18n.ResolveLocale(c, "") == "fa" {
		msg = "اطلاعات واردشده نامعتبر است"
	}
	return nil, httpx.Fail(fiber.StatusUnprocessableEntity, msg, "errors", errs)
}

// assign is $reminder->setAttribute($key, $value) for a validated attribute: it updates row
// the way the DB will store the value and returns the cast value a fresh model serialises
// plus whether the attribute is dirty against the row's previous value
// (Model::originalIsEquivalent).
func assign(row *store.Reminder, key string, v any, now time.Time) (any, bool, error) {
	switch key {
	case "type", "title", "recurrence":
		s := phpval.ToString(v)
		target := map[string]*string{"type": &row.Type, "title": &row.Title, "recurrence": &row.Recurrence}[key]
		changed := *target != s
		*target = s
		return v, changed, nil
	case "subtitle", "notes", "recurrence_time":
		target := map[string]*sql.NullString{"subtitle": &row.Subtitle, "notes": &row.Notes, "recurrence_time": &row.RecurrenceTime}[key]
		next := sql.NullString{}
		if v != nil {
			next = sql.NullString{String: phpval.ToString(v), Valid: true}
		}
		changed := *target != next
		*target = next
		return v, changed, nil
	case "scheduled_at":
		if v == nil {
			changed := row.ScheduledAt.Valid
			row.ScheduledAt = sql.NullTime{}
			return nil, changed, nil
		}
		t, err := parseDateTime(v, now)
		if err != nil {
			return nil, false, err
		}
		changed := !row.ScheduledAt.Valid || storageFormat(row.ScheduledAt.Time) != storageFormat(t)
		row.ScheduledAt = sql.NullTime{Time: t, Valid: true}
		return jsonx.DateTime(t), changed, nil
	case "starts_on", "ends_on":
		target := map[string]*civildate.NullDate{"starts_on": &row.StartsOn, "ends_on": &row.EndsOn}[key]
		if v == nil {
			changed := target.Valid
			*target = civildate.NullDate{}
			return nil, changed, nil
		}
		t, err := parseDateTime(v, now)
		if err != nil {
			return nil, false, err
		}
		d := civildate.FromTime(t)
		// fromDateTime() keeps the time of day, so "2026-09-23 10:00" differs from the
		// stored 2026-09-23 even though the DATE column ends up equal.
		changed := !target.Valid || storageFormat(target.Date.TehranMidnight()) != storageFormat(t)
		*target = civildate.NullDate{Date: d, Valid: true}
		return jsonx.DateCast(d), changed, nil
	case "is_active":
		b := phpval.Truthy(v)
		changed := row.IsActive != b
		row.IsActive = b
		return b, changed, nil
	}
	return nil, false, fmt.Errorf("reminder: unexpected attribute %q", key)
}

// parseDateTime is Model::asDateTime for a request string (Y-m-d → start of day, otherwise
// Carbon::parse in Asia/Tehran), truncated to whole seconds as fromDateTime() stores it.
func parseDateTime(v any, now time.Time) (time.Time, error) {
	t, err := civildate.ParseLenient(phpval.ToString(v), now, civildate.Tehran)
	if err != nil {
		return time.Time{}, fmt.Errorf("reminder: date %v: %w", v, err)
	}
	return t.In(civildate.Tehran).Truncate(time.Second), nil
}

func storageFormat(t time.Time) string { return t.In(civildate.Tehran).Format(time.DateTime) }

// decodeArray is the `array` cast (json_decode($raw, true)); invalid JSON is null.
func decodeArray(raw []byte, valid bool) any {
	if !valid {
		return nil
	}
	v, err := phpval.Decode(raw)
	if err != nil {
		return nil
	}
	return phpval.Packed(v)
}
