package cityservices

// The user booking routes: bloom's B-N10-05 create / list / cancel, extended in place by CB-DIR-03 (service, slot,
// child, upcoming / past, cancel window) plus one booking, reschedule, review and the place report. All auth:api and
// owner-scoped (anyone else's booking is a 404).

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func bookingRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("listing_id", "required", "integer", "min:1"),
		F("contact_name", "required", "string", "max:120"),
		F("contact_phone", "required", "string", "regex:/^\\+?[0-9]{7,15}$/"),
		// CB-DIR-03: a chosen slot fixes the day.
		F("preferred_date", "required_without:slot_id", "nullable", "date_format:Y-m-d"),
		F("child_age_range", "nullable", validation.In(AgeRanges...)),
		F("note", "nullable", "string", "max:500"),
		F("service_id", "nullable", "integer", "min:1"), // CB-DIR-03
		F("slot_id", "nullable", "integer", "min:1"),    // CB-DIR-03
		F("child_id", "nullable", "integer", "min:1"),   // CB-DIR-03: her own child (bloom `children`)
	}
}

// bookingFields are the only body keys a request ever reads (nothing about her health is accepted or stored).
var bookingFields = []string{"listing_id", "contact_name", "contact_phone", "preferred_date", "child_age_range", "note",
	"service_id", "slot_id", "child_id"}

// StoreBooking is POST /city-services/bookings: a request for a `request`- or (CB-DIR-03) `online`-mode listing.
// 201 with the booking in status requested — the Ritme team confirms it; nothing is ever auto-confirmed or paid.
//
// CB-DIR-03 additions: service_id (an active service of the place, its price snapshotted), slot_id (required for an
// online place; a future open slot of the place and service whose seat is held transactionally — 422 slot_full when
// another mother took the last one), child_id (the caller's own child, whose age must fit the service). A missing
// contact_name / contact_phone defaults to the account's name / mobile.
func (h *Handlers) StoreBooking(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in := validation.Input(c)
	data := phpval.NewMap()
	for _, k := range bookingFields {
		if v, ok := in.Get(k); ok {
			if s, isStr := v.(string); isStr && k == "contact_phone" {
				v = digitFold.Replace(s)
			}
			data.Set(k, v)
		}
	}
	if err := h.defaultContact(c, userID, data); err != nil {
		return err
	}
	v := validation.Make(lang.Default(), locale, data, bookingRules(), validation.Now(now),
		validation.Attributes(attributes[locale]...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("validation_failed", locale), "errors", v.ErrorBag())
	}
	get := func(k string) string { x, _ := data.Get(k); return strings.TrimSpace(phpval.ToString(x)) }
	id := func(k string) uint64 { n, _ := parseID(get(k)); return n }
	listingID, slotID, serviceID, childID := id("listing_id"), id("slot_id"), id("service_id"), id("child_id")
	today := civildate.FromTime(now)

	l, err := h.store.PublicListing(c, listingID)
	if errors.Is(err, ErrNotFound) {
		return fieldFail(c, "listing_id", msg("not_found", locale))
	}
	if err != nil {
		return fmt.Errorf("cityservices: listing: %w", err)
	}
	if l.BookingMode != ModeRequest && l.BookingMode != ModeOnline {
		return fieldFail(c, "listing_id", msg("not_requestable", locale))
	}
	if l.BookingMode == ModeOnline && slotID == 0 {
		return fieldFail(c, "slot_id", msg("slot_required", locale))
	}

	var dir DirBooking
	var service *Service
	if serviceID > 0 {
		sv, err := h.store.ActiveServiceRow(c, listingID, serviceID)
		if errors.Is(err, ErrNotFound) {
			return fieldFail(c, "service_id", msg("not_found", locale))
		}
		if err != nil {
			return err
		}
		service = &sv
	}
	var date civildate.Date
	if slotID > 0 {
		slot, err := h.store.BookableSlot(c, listingID, slotID, now)
		if errors.Is(err, ErrSlotUnavailable) {
			return fieldFail(c, "slot_id", msg("slot_unavailable", locale))
		}
		if err != nil {
			return err
		}
		if slot.ServiceID.Valid {
			slotService := uint64(slot.ServiceID.Int64) //nolint:gosec // G115: ids are positive
			if service != nil && service.ID != slotService {
				return fieldFail(c, "service_id", msg("service_mismatch", locale))
			}
			if service == nil { // the slot belongs to one service: book that one
				sv, err := h.store.ActiveServiceRow(c, listingID, slotService)
				if err != nil {
					return fieldFail(c, "slot_id", msg("slot_unavailable", locale))
				}
				service = &sv
			}
		}
		if slot.BookedCount >= slot.Capacity {
			return fieldFail(c, "slot_id", msg("slot_full", locale))
		}
		date = civildate.FromTime(wall(slot.StartsAt))
		dir.SlotID = sql.NullInt64{Int64: int64(slotID), Valid: true} //nolint:gosec // G115: validated ids
	} else {
		if date, err = civildate.Parse(get("preferred_date")); err != nil {
			return fieldFail(c, "preferred_date", msg("validation_failed", locale))
		}
		if date.Before(today) {
			return fieldFail(c, "preferred_date", msg("date_past", locale))
		}
		if date.DiffDays(today) > MaxAdvanceDays || today.DiffDays(date) > MaxAdvanceDays {
			return fieldFail(c, "preferred_date", msg("validation_failed", locale))
		}
	}
	if service != nil {
		dir.ServiceID = sql.NullInt64{Int64: int64(service.ID), Valid: true} //nolint:gosec // G115: ids are positive
		dir.PriceToman = service.PriceToman
	}
	var child *Child
	if childID > 0 {
		ch, err := h.store.UserChild(c, userID, childID)
		if errors.Is(err, ErrNotFound) { // someone else's child reads exactly like a missing one
			return fieldFail(c, "child_id", msg("not_found", locale))
		}
		if err != nil {
			return err
		}
		if service != nil && !AgeFits(*service, AgeMonthsOn(ch.Birth, date)) {
			return fieldFail(c, "child_id", msg("age_mismatch", locale))
		}
		child = &ch
		dir.ChildID = sql.NullInt64{Int64: int64(childID), Valid: true} //nolint:gosec // G115: validated ids
	}

	optional := func(k string) (n sql.NullString) {
		if s := get(k); s != "" {
			n.Valid, n.String = true, s
		}
		return n
	}
	b, err := h.store.CreateDirBooking(c, userID, NewBooking{
		ListingID: listingID, ContactName: get("contact_name"), ContactPhone: get("contact_phone"), PreferredDate: date,
		ChildAgeRange: optional("child_age_range"), Note: optional("note"),
	}, dir, today, now)
	switch { // the per-user guards (slot taken, one day request per venue, open cap) run in the transaction
	case errors.Is(err, ErrSlotTaken):
		return fieldFail(c, "slot_id", msg("slot_taken", locale))
	case errors.Is(err, ErrAlreadyOpen):
		return fieldFail(c, "listing_id", msg("already_open", locale))
	case errors.Is(err, ErrTooManyOpen):
		return fieldFail(c, "listing_id", msg("too_many_open", locale))
	case errors.Is(err, ErrSlotFull):
		return fieldFail(c, "slot_id", msg("slot_full", locale))
	case errors.Is(err, ErrSlotUnavailable):
		return fieldFail(c, "slot_id", msg("slot_unavailable", locale))
	case err != nil:
		return err
	}
	h.notify(c, b, child)
	return httpx.Created(c, jsonx.Obj("booking", h.labeler(c).BookingJSON(b)), msg("booking_created", locale))
}

// defaultContact fills an absent contact_name / contact_phone from the account (an explicit "" stays a 422).
func (h *Handlers) defaultContact(c fiber.Ctx, userID uint64, data phpval.Map) error {
	_, hasName := data.Get("contact_name")
	_, hasPhone := data.Get("contact_phone")
	if hasName && hasPhone {
		return nil
	}
	name, mobile, err := h.store.UserContact(c, userID)
	if err != nil {
		return fmt.Errorf("cityservices: contact: %w", err)
	}
	if !hasName && strings.TrimSpace(name) != "" {
		data.Set("contact_name", name)
	}
	if !hasPhone && mobile != "" {
		data.Set("contact_phone", digitFold.Replace(mobile))
	}
	return nil
}

// notify tells the venue door about a request: allow-listed fields only (name, phone, child first name + age).
func (h *Handlers) notify(c fiber.Ctx, b Booking, child *Child) {
	n := BookingNotice{
		BookingID: b.ID, ListingID: b.ListingID, ContactName: b.ContactName, ContactPhone: b.ContactPhone,
		PreferredDate: b.PreferredDate, ChildAgeRange: b.ChildAgeRange.String, Note: b.Note.String,
		ServiceName: b.ServiceName.String, ChildAgeMonths: -1,
	}
	if at, ok := b.StartsAt(); ok {
		n.StartsAt = at
	}
	if child != nil {
		n.ChildName, n.ChildAgeMonths = child.Name, AgeMonthsOn(child.Birth, b.PreferredDate)
	}
	if err := h.notifier.BookingRequested(c, n); err != nil {
		h.logger.WarnContext(c, "cityservices: notifier failed", slog.Uint64("booking_id", b.ID)) // the queue is the truth
	}
}

// Bookings is GET /city-services/bookings[?scope=upcoming|past] → the caller's own bookings («رزروهای من»): bloom's
// newest-first list without a scope; upcoming soonest first, past latest first. CB-DIR-03 adds `counts`.
func (h *Handlers) Bookings(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	scope := strings.TrimSpace(c.Query("scope"))
	if scope != "" && scope != ScopeUpcoming && scope != ScopePast {
		return fieldFail(c, "scope", msg("validation_failed", i18n.Locale(c)))
	}
	now := h.now(c)
	rows, err := h.store.UserBookingsScoped(c, userID, scope, now)
	if err != nil {
		return fmt.Errorf("cityservices: bookings: %w", err)
	}
	upcoming, past, err := h.store.BookingCounts(c, userID, now)
	if err != nil {
		return fmt.Errorf("cityservices: booking counts: %w", err)
	}
	lb := h.labeler(c)
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, b := range rows {
		items = append(items, lb.BookingJSON(b))
	}
	return httpx.OK(c, jsonx.Obj("bookings", items, "counts", jsonx.Obj(ScopeUpcoming, upcoming, ScopePast, past)))
}

// ownBooking loads the caller's booking from :id; any other id (anyone else's, missing, malformed) is a 404.
func (h *Handlers) ownBooking(c fiber.Ctx) (uint64, Booking, error) {
	userID, err := h.user(c)
	if err != nil {
		return 0, Booking{}, err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return userID, Booking{}, notFound(c)
	}
	b, err := h.store.UserBooking(c, userID, id)
	if errors.Is(err, ErrNotFound) {
		return userID, b, notFound(c)
	}
	if err != nil {
		return userID, b, fmt.Errorf("cityservices: booking: %w", err)
	}
	return userID, b, nil
}

// Booking is GET /city-services/bookings/{id} (CB-DIR-03, Dir_Booked): one of the caller's bookings.
func (h *Handlers) Booking(c fiber.Ctx) error {
	_, b, err := h.ownBooking(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", h.labeler(c).BookingJSON(b)))
}

// CancelBooking is POST /city-services/bookings/{id}/cancel: owner only (404 for anyone else's), 422 once decided.
// CB-DIR-03: a slot booking can be cancelled until its start, a confirmed one only until the place's
// cancel_free_hours before it (422 cancel_window_passed); the seat goes back to the slot.
func (h *Handlers) CancelBooking(c fiber.Ctx) error {
	userID, b, err := h.ownBooking(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	switch b.CancelStateAt(now) {
	case CancelClosed:
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("not_cancellable", locale))
	case CancelWindowPassed:
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("cancel_window_passed", locale))
	}
	done, err := h.store.CancelOwnBooking(c, userID, b, now)
	if err != nil {
		return err
	}
	if !done {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("not_cancellable", locale))
	}
	if b, err = h.store.UserBooking(c, userID, b.ID); err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", h.labeler(c).BookingJSON(b)), msg("booking_cancelled", locale))
}

// RescheduleBooking is POST /city-services/bookings/{id}/reschedule {slot_id} (CB-DIR-03, «تغییر زمان»): another slot
// of the same place and service while the booking could still be cancelled; it goes back to `requested`.
func (h *Handlers) RescheduleBooking(c fiber.Ctx) error {
	userID, b, err := h.ownBooking(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	v := validation.Make(lang.Default(), locale, validation.Input(c), validation.Rules{
		validation.F("slot_id", "required", "integer", "min:1"),
	}, validation.Now(now), validation.Attributes(attributes[locale]...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("validation_failed", locale), "errors", v.ErrorBag())
	}
	if !b.CanReschedule(now) {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("not_reschedulable", locale))
	}
	x, _ := validation.Input(c).Get("slot_id")
	slotID, _ := parseID(strings.TrimSpace(phpval.ToString(x)))
	if int64(slotID) == b.SlotID.Int64 { //nolint:gosec // G115: validated ids
		return fieldFail(c, "slot_id", msg("slot_taken", locale))
	}
	slot, err := h.store.BookableSlot(c, b.ListingID, slotID, now)
	if errors.Is(err, ErrSlotUnavailable) {
		return fieldFail(c, "slot_id", msg("slot_unavailable", locale))
	}
	if err != nil {
		return err
	}
	// The new slot must be of the very same service (both none, or the same id), and the child must still fit its
	// age bounds on the new date (security audit L2) — a reschedule is not a way around the create checks.
	if slot.ServiceID.Valid != b.ServiceID.Valid || (slot.ServiceID.Valid && slot.ServiceID.Int64 != b.ServiceID.Int64) {
		return fieldFail(c, "slot_id", msg("service_mismatch", locale))
	}
	if b.ServiceID.Valid && b.ChildBirth.Valid {
		sv, err := h.store.ActiveServiceRow(c, b.ListingID, uint64(b.ServiceID.Int64)) //nolint:gosec // G115: ids are positive
		if errors.Is(err, ErrNotFound) {
			return fieldFail(c, "slot_id", msg("slot_unavailable", locale))
		}
		if err != nil {
			return err
		}
		if !AgeFits(sv, AgeMonthsOn(civildate.FromTime(b.ChildBirth.Time), civildate.FromTime(wall(slot.StartsAt)))) {
			return fieldFail(c, "slot_id", msg("age_mismatch", locale))
		}
	}
	switch err := h.store.Reschedule(c, userID, b, slot, now); {
	case errors.Is(err, ErrSlotFull):
		return fieldFail(c, "slot_id", msg("slot_full", locale))
	case errors.Is(err, ErrSlotUnavailable):
		return fieldFail(c, "slot_id", msg("slot_unavailable", locale))
	case errors.Is(err, ErrNotOpen):
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("not_reschedulable", locale))
	case err != nil:
		return err
	}
	if b, err = h.store.UserBooking(c, userID, b.ID); err != nil {
		return err
	}
	h.notify(c, b, nil)
	return httpx.OK(c, jsonx.Obj("booking", h.labeler(c).BookingJSON(b)), msg("rescheduled", locale))
}

func reviewRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("rating", "required", "integer", "min:1", "max:5"),
		F("score_cleanliness", "nullable", "integer", "min:1", "max:5"),
		F("score_instructor", "nullable", "integer", "min:1", "max:5"),
		F("score_amenities", "nullable", "integer", "min:1", "max:5"),
		F("body", "nullable", "string", "max:1000"),
	}
}

// ReviewBooking is POST /city-services/bookings/{id}/review (CB-DIR-03, «نظر بده»): one review per COMPLETED booking
// of the caller, stored `pending` — the Ritme team moderates it before it is public (and before it counts).
func (h *Handlers) ReviewBooking(c fiber.Ctx) error {
	userID, b, err := h.ownBooking(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	data := phpval.NewMap()
	in := validation.Input(c)
	for _, k := range []string{"rating", "score_cleanliness", "score_instructor", "score_amenities", "body"} {
		if v, ok := in.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, reviewRules(), validation.Now(now),
		validation.Attributes(attributes[locale]...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("validation_failed", locale), "errors", v.ErrorBag())
	}
	if b.ReviewID.Valid {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("already_reviewed", locale))
	}
	if b.Status != StatusCompleted {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("not_reviewable", locale))
	}
	get := func(k string) string { x, _ := data.Get(k); return strings.TrimSpace(phpval.ToString(x)) }
	score := func(k string) (n sql.NullInt64) {
		if x, ok := parseID(get(k)); ok {
			n.Valid, n.Int64 = true, int64(x) //nolint:gosec // G115: 1..5
		}
		return n
	}
	r := NewReview{Cleanliness: score("score_cleanliness"), Instructor: score("score_instructor"), Amenities: score("score_amenities")}
	r.Rating = int(score("rating").Int64)
	if body := get("body"); body != "" {
		r.Body = sql.NullString{String: body, Valid: true}
	}
	id, err := h.store.CreateReview(c, userID, b, r, now)
	switch {
	case errors.Is(err, ErrAlreadyReviewed):
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("already_reviewed", locale))
	case errors.Is(err, ErrNotOpen):
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("not_reviewable", locale))
	case err != nil:
		return err
	}
	return httpx.Created(c, jsonx.Obj("review", jsonx.Obj("id", id, "booking_id", b.ID, "rating", r.Rating,
		"score_cleanliness", nullInt(r.Cleanliness), "score_instructor", nullInt(r.Instructor),
		"score_amenities", nullInt(r.Amenities), "body", nullStr(r.Body), "status", ReviewPending)),
		msg("review_received", locale))
}

// ReportListing is POST /city-services/listings/{id}/report {reason, note} (CB-DIR-03, «گزارش اطلاعات نادرست»):
// auth:api; a published place only (404 otherwise); one open report per user and place.
func (h *Handlers) ReportListing(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c)
	}
	locale, now := i18n.Locale(c), h.now(c)
	data := phpval.NewMap()
	in := validation.Input(c)
	for _, k := range []string{"reason", "note"} {
		if v, ok := in.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, validation.Rules{
		validation.F("reason", "required", validation.In(ReportReasons...)),
		validation.F("note", "nullable", "string", "max:500"),
	}, validation.Now(now), validation.Attributes(attributes[locale]...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("validation_failed", locale), "errors", v.ErrorBag())
	}
	if _, err := h.store.PublicListing(c, id); errors.Is(err, ErrNotFound) {
		return notFound(c)
	} else if err != nil {
		return fmt.Errorf("cityservices: listing: %w", err)
	}
	open, err := h.store.OpenReport(c, userID, id)
	if err != nil {
		return err
	}
	if open {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("already_reported", locale))
	}
	get := func(k string) string { x, _ := data.Get(k); return strings.TrimSpace(phpval.ToString(x)) }
	var note sql.NullString
	if s := get("note"); s != "" {
		note = sql.NullString{String: s, Valid: true}
	}
	rid, err := h.store.CreateReport(c, userID, id, get("reason"), note, now)
	if err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("report", jsonx.Obj("id", rid, "listing_id", id, "reason", get("reason"), "status", "open")),
		msg("report_received", locale))
}
