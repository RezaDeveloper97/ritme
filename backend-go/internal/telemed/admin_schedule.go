package telemed

import (
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// list is the entries of the array field key of in (nil when absent or not an array).
func list(in phpval.Map, key string) []any {
	v, _ := phpval.Get(in, key)
	if !phpval.IsArray(v) {
		return nil
	}
	_, vals := phpval.Entries(v)
	return vals
}

func field(item any, key string) any {
	v, _ := phpval.Get(item, key)
	return v
}

// ---------------------------------------------------------------------------
// Visit types

// PutVisitTypes is PUT /telemed/doctors/:id/visit-types {visit_types: [{mode, duration_minutes, price_rials, note?,
// address?, is_active?}]}: the full set — each mode at most once; modes left out are removed.
func (h *Admin) PutVisitTypes(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	r := validation.Rules{
		validation.F("visit_types", "present|array|max:"+strconv.Itoa(len(Modes))),
		validation.F("visit_types.*.mode", "required|string", validation.In(Modes...)),
		validation.F("visit_types.*.duration_minutes", "required|integer|between:"+strconv.Itoa(MinDuration)+","+strconv.Itoa(MaxDuration)),
		validation.F("visit_types.*.price_rials", "required|integer|between:0,"+strconv.FormatInt(MaxPriceRials, 10)),
		validation.F("visit_types.*.is_active", "nullable|boolean"),
	}
	r = append(r, form.Translatable(c, "visit_types.*.note", false, "max:"+strconv.Itoa(MaxNoteLen))...)
	r = append(r, form.Translatable(c, "visit_types.*.address", false, "max:"+strconv.Itoa(MaxNoteLen))...)
	dupes := func(in phpval.Map, add form.Add) error {
		seen := map[string]bool{}
		for i, it := range list(in, "visit_types") {
			m := phpval.ToString(field(it, "mode"))
			if m != "" && seen[m] {
				f := "visit_types." + strconv.Itoa(i) + ".mode"
				add(f, form.Msg(c, "validation.distinct", f))
			}
			seen[m] = true
		}
		return nil
	}
	data, err := form.Validate(c, r, dupes)
	if err != nil {
		return err
	}
	codes := i18n.LanguagesOf(c).Codes()
	now := httpadmin.DBTime(httpadmin.Now(c))
	tx, err := h.svc.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	kept := map[string]bool{}
	for _, it := range list(data, "visit_types") {
		mode := phpval.ToString(field(it, "mode"))
		kept[mode] = true
		active := true
		if v := field(it, "is_active"); v != nil {
			active = phpval.Truthy(v)
		}
		if err := q.UpsertVisitType(c.Context(), store.UpsertVisitTypeParams{
			DoctorID: cur.ID, Mode: mode,
			DurationMinutes: uint16(phpval.ToFloat(field(it, "duration_minutes"))), //nolint:gosec // G115: validated 5–240
			PriceRials:      uint64(phpval.ToFloat(field(it, "price_rials"))),      //nolint:gosec // G115: validated ≥ 0
			Note:            translatedCol(field(it, "note"), codes), Address: translatedCol(field(it, "address"), codes),
			IsActive: active, Now: now,
		}); err != nil {
			return err
		}
	}
	for _, m := range Modes {
		if !kept[m] {
			if err := q.DeleteVisitType(c.Context(), store.DeleteVisitTypeParams{DoctorID: cur.ID, Mode: m}); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.visit_types", "telemed_doctor", cur.ID, slog.Int("count", len(kept)))
	return h.respond(c, cur.ID, false, "Visit types saved.")
}

// translatedCol is {code: text} over the active languages with text (SQL NULL when none).
func translatedCol(v any, codes []string) db.NullRawJSON {
	kv := make([]any, 0, 2*len(codes))
	for _, code := range codes {
		t, ok := phpval.Get(v, code)
		if !ok || t == nil {
			continue
		}
		if s := strings.TrimSpace(phpval.ToString(t)); s != "" {
			kv = append(kv, code, s)
		}
	}
	if len(kv) == 0 {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: form.JSON(jsonx.Obj(kv...)), Valid: true}
}

// ---------------------------------------------------------------------------
// Availability

// ParseClock is "HH:MM" (or "24:00") as minutes after midnight.
func ParseClock(s string) (int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	hh, err1 := strconv.Atoi(s[:2])
	mm, err2 := strconv.Atoi(s[3:])
	if err1 != nil || err2 != nil || mm < 0 || mm > 59 || hh < 0 || hh > 24 || (hh == 24 && mm != 0) {
		return 0, false
	}
	return hh*60 + mm, true
}

// PutAvailability is PUT /telemed/doctors/:id/availability {rules: [{weekday 0–6 (0 = Sunday), start_time "HH:MM",
// end_time "HH:MM" (≤ 24:00, after start), slot_minutes 5–240 (≤ the window), modes?: [mode…] (null = all)}]}: the
// full weekly schedule.
func (h *Admin) PutAvailability(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	r := validation.Rules{
		validation.F("rules", "present|array|max:"+strconv.Itoa(MaxRules)),
		validation.F("rules.*.weekday", "required|integer|between:0,6"),
		validation.F("rules.*.start_time", "required|string", validation.Regex(timeOfDayPattern)),
		validation.F("rules.*.end_time", "required|string", validation.Regex(timeOfDayPattern)),
		validation.F("rules.*.slot_minutes", "required|integer|between:"+strconv.Itoa(MinDuration)+","+strconv.Itoa(MaxDuration)),
		validation.F("rules.*.modes", "nullable|array|max:"+strconv.Itoa(len(Modes))),
		validation.F("rules.*.modes.*", "required|string", validation.In(Modes...)),
	}
	order := func(in phpval.Map, add form.Add) error {
		for i, it := range list(in, "rules") {
			start, ok1 := ParseClock(phpval.ToString(field(it, "start_time")))
			end, ok2 := ParseClock(phpval.ToString(field(it, "end_time")))
			if !ok1 || !ok2 {
				continue
			}
			prefix := "rules." + strconv.Itoa(i) + "."
			if end <= start {
				add(prefix+"end_time", form.Msg(c, "validation.after", prefix+"end_time", "date", prefix+"start_time"))
				continue
			}
			if step := int(phpval.ToFloat(field(it, "slot_minutes"))); step > end-start {
				add(prefix+"slot_minutes", form.Msg(c, "validation.max.numeric", prefix+"slot_minutes", "max", strconv.Itoa(end-start)))
			}
		}
		return nil
	}
	data, err := form.Validate(c, r, order)
	if err != nil {
		return err
	}
	now := httpadmin.DBTime(httpadmin.Now(c))
	tx, err := h.svc.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	if err := q.DeleteDoctorRules(c.Context(), cur.ID); err != nil {
		return err
	}
	rules := list(data, "rules")
	for _, it := range rules {
		start, _ := ParseClock(phpval.ToString(field(it, "start_time")))
		end, _ := ParseClock(phpval.ToString(field(it, "end_time")))
		modes := modesCol(field(it, "modes"))
		if err := q.InsertRule(c.Context(), store.InsertRuleParams{
			DoctorID: cur.ID, Weekday: uint8(phpval.ToFloat(field(it, "weekday"))), //nolint:gosec // G115: validated 0–6
			StartMinute: uint16(start), EndMinute: uint16(end), //nolint:gosec // G115: 0–1440
			SlotMinutes: uint16(phpval.ToFloat(field(it, "slot_minutes"))), //nolint:gosec // G115: validated 5–240
			Modes:       modes, Now: now,
		}); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.availability", "telemed_doctor", cur.ID, slog.Int("count", len(rules)))
	return h.respond(c, cur.ID, false, "Availability saved.")
}

// modesCol is the distinct modes of a validated list (SQL NULL = every mode).
func modesCol(v any) db.NullRawJSON {
	if v == nil {
		return db.NullRawJSON{}
	}
	_, vals := phpval.Entries(v)
	seen := map[string]bool{}
	ms := []string{}
	for _, m := range vals {
		if s := phpval.ToString(m); s != "" && !seen[s] {
			seen[s] = true
			ms = append(ms, s)
		}
	}
	if len(ms) == 0 {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: form.JSON(ms), Valid: true}
}

// ---------------------------------------------------------------------------
// Time off

// StoreTimeOff is POST /telemed/doctors/:id/time-off {starts_at, ends_at ("Y-m-d H:i", Tehran), note?} (201): no slot
// overlaps the span; at most a year long.
func (h *Admin) StoreTimeOff(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	parse := func(in phpval.Map, key string) (time.Time, bool) {
		v, _ := phpval.Get(in, key)
		t, err := time.ParseInLocation(adminTimeLayout, phpval.ToString(v), civildate.Tehran)
		return t, err == nil
	}
	span := func(in phpval.Map, add form.Add) error {
		start, ok1 := parse(in, "starts_at")
		end, ok2 := parse(in, "ends_at")
		if !ok1 || !ok2 {
			return nil
		}
		switch {
		case !end.After(start):
			add("ends_at", form.Msg(c, "validation.after", "ends_at", "date", "starts_at"))
		case end.Sub(start) > MaxTimeOffDays*24*time.Hour:
			add("ends_at", form.Msg(c, "validation.before_or_equal", "ends_at", "date",
				start.AddDate(0, 0, MaxTimeOffDays).Format(adminTimeLayout)))
		}
		return nil
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("starts_at", "required|date_format:"+adminTimeFormat),
		validation.F("ends_at", "required|date_format:"+adminTimeFormat),
		validation.F("note", "nullable|string|max:"+strconv.Itoa(MaxTimeOffNote)),
	}, span)
	if err != nil {
		return err
	}
	start, _ := parse(data, "starts_at")
	end, _ := parse(data, "ends_at")
	id, err := h.q.InsertTimeOff(c.Context(), store.InsertTimeOffParams{
		DoctorID: cur.ID, StartsAt: start, EndsAt: end, Note: nullStr(data, "note"), Now: httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.time_off", "telemed_doctor", cur.ID, slog.Int64("time_off_id", id))
	return h.respond(c, cur.ID, true, "Time off added.")
}

// DestroyTimeOff is DELETE /telemed/doctors/:id/time-off/:off.
func (h *Admin) DestroyTimeOff(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	off, ok := httpadmin.ID(c, "off")
	if !ok {
		return httpadmin.NotFound("Time off")
	}
	n, err := h.q.DeleteTimeOff(c.Context(), store.DeleteTimeOffParams{ID: off, DoctorID: cur.ID})
	if err != nil {
		return err
	}
	if n == 0 {
		return httpadmin.NotFound("Time off")
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.time_off_delete", "telemed_doctor", cur.ID)
	return h.respond(c, cur.ID, false, "Time off removed.")
}

// PreviewSlots is GET /telemed/doctors/:id/slots?mode=&from=&days=: the slots users would see (also for an inactive
// doctor), to check a schedule.
func (h *Admin) PreviewSlots(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	now := httpadmin.Now(c).In(civildate.Tehran).Truncate(time.Second)
	in, err := ValidateSlots(validation.Input(c), i18n.Locale(c), now)
	if err != nil {
		return err
	}
	details, err := h.svc.LoadDetails(c.Context(), []store.TelemedDoctor{cur}, in.From, max(in.Days, NextSlotDays))
	if err != nil {
		return err
	}
	res, err := SlotsOf(details[cur.ID], in, now)
	if errors.Is(err, ErrModeNotOffered) {
		return httpadmin.FieldError("mode", form.Msg(c, "validation.in", "mode"))
	}
	if err != nil {
		return err
	}
	labels, err := h.svc.LoadLabels(c.Context())
	if err != nil {
		return err
	}
	langs := i18n.LanguagesOf(c)
	v := View{Labels: labels, AppURL: h.appURL}
	v.Loc.Locale, v.Loc.Default, v.Loc.Langs = i18n.Locale(c), langs.DefaultCode(), langs
	return httpadmin.OK(c, v.SlotsJSON(res, in))
}

// ---------------------------------------------------------------------------
// Reviews moderation

func reviewAdminJSON(r store.TelemedReview) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID, "doctor_id", r.DoctorID, "user_id", r.UserID, "booking_id", nullInt64(r.BookingID),
		"rating", r.Rating, "body", httpadmin.NullString(r.Body), "is_visible", r.IsVisible,
		"created_at", httpadmin.Time(r.CreatedAt), "updated_at", httpadmin.Time(r.UpdatedAt),
	)
}

// Reviews is GET /telemed/reviews?doctor_id=&visibility=all|visible|hidden&page=&per_page= (newest first).
func (h *Admin) Reviews(c fiber.Ctx) error {
	doctorID, _ := strconv.ParseUint(c.Query("doctor_id"), 10, 64)
	visibility := c.Query("visibility")
	var only *bool
	switch visibility {
	case "visible":
		only = new(bool)
		*only = true
	case "hidden":
		only = new(bool)
	default:
		visibility = "all"
	}
	lo, hi := form.BoolRange(only)
	total, err := h.q.CountAdminReviews(c.Context(), store.CountAdminReviewsParams{DoctorID: doctorID, VisibleMin: lo, VisibleMax: hi})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminReviews(c.Context(), store.ListAdminReviewsParams{
		DoctorID: doctorID, VisibleMin: lo, VisibleMax: hi,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, reviewAdminJSON(r))
	}
	page := httpadmin.Page(items, p, int(total))
	var doc any
	if doctorID > 0 {
		doc = doctorID
	}
	page.Set("filters", jsonx.Obj("doctor_id", doc, "visibility", visibility))
	return httpadmin.OK(c, page)
}

// UpdateReview is PUT /telemed/reviews/:id {is_visible}: hide or show a review (the doctor's rating follows).
func (h *Admin) UpdateReview(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Review")
	}
	cur, err := h.q.GetReview(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Review")
	}
	if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{validation.F("is_visible", "required|boolean")})
	if err != nil {
		return err
	}
	tx, err := h.svc.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	if err := q.SetReviewVisibility(c.Context(), store.SetReviewVisibilityParams{
		IsVisible: httpadmin.Bool(data, "is_visible"), Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}); err != nil {
		return err
	}
	if err := q.RefreshDoctorRating(c.Context(), cur.DoctorID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "telemed_review.update", "telemed_review", cur.ID)
	r, err := h.q.GetReview(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("review", reviewAdminJSON(r)), "Review updated.")
}
