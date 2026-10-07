package telemed

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// Booking error codes.
const (
	ErrorCodeBookingNotFound    = "booking_not_found"
	ErrorCodeSlotUnavailable    = "slot_unavailable"
	ErrorCodePaymentUnavailable = "payment_unavailable"
	ErrorCodePaymentFailed      = "payment_failed"
	ErrorCodeAmountMismatch     = "amount_mismatch"
	ErrorCodeAuthorityMismatch  = "authority_mismatch"
	ErrorCodeBookingClosed      = "booking_closed"
	ErrorCodeBookingRefunded    = "booking_refunded"
	ErrorCodeCancelWindowClosed = "cancel_window_closed"
	ErrorCodeConsentNotAllowed  = "consent_not_allowed"
)

// BookingHandlers are the /api/v1/telemed/bookings actions. Mount them behind the locale middleware and auth
// RequireUser; every booking is the caller's own (another user's id is a uniform 404).
type BookingHandlers struct {
	b *Bookings
	h *Handlers // directory view helpers
}

// NewBookingHandlers wires the handlers; base is the fallback clock (tests pin it).
func NewBookingHandlers(b *Bookings, base clock.Clock, appURL string) *BookingHandlers {
	return &BookingHandlers{b: b, h: NewHandlers(b.Directory(), base, appURL)}
}

func (bh *BookingHandlers) loc(c fiber.Ctx) catalog.Localizer {
	langs := i18n.LanguagesOf(c)
	return catalog.Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs}
}

func coded(status int, key, code, locale string) error {
	return httpx.Fail(status, T("messages."+key, locale), "error_code", code)
}

// failBooking maps the booking errors.
func failBooking(err error, locale string) error {
	switch {
	case errors.Is(err, ErrBookingNotFound):
		return coded(fiber.StatusNotFound, "booking_not_found", ErrorCodeBookingNotFound, locale)
	case errors.Is(err, ErrSlotUnavailable):
		return coded(fiber.StatusConflict, "slot_unavailable", ErrorCodeSlotUnavailable, locale)
	case errors.Is(err, ErrPaymentUnavailable):
		return coded(fiber.StatusServiceUnavailable, "payment_unavailable", ErrorCodePaymentUnavailable, locale)
	case errors.Is(err, ErrPaymentFailed):
		return coded(fiber.StatusUnprocessableEntity, "payment_failed", ErrorCodePaymentFailed, locale)
	case errors.Is(err, ErrAmountMismatch):
		return coded(fiber.StatusUnprocessableEntity, "amount_mismatch", ErrorCodeAmountMismatch, locale)
	case errors.Is(err, ErrAuthorityMismatch):
		return coded(fiber.StatusUnprocessableEntity, "authority_mismatch", ErrorCodeAuthorityMismatch, locale)
	case errors.Is(err, ErrBookingClosed):
		return coded(fiber.StatusConflict, "booking_closed", ErrorCodeBookingClosed, locale)
	case errors.Is(err, ErrBookingRefunded):
		return coded(fiber.StatusConflict, "booking_refunded", ErrorCodeBookingRefunded, locale)
	case errors.Is(err, ErrCancelWindowClosed):
		return coded(fiber.StatusConflict, "cancel_window_closed", ErrorCodeCancelWindowClosed, locale)
	case errors.Is(err, ErrConsentNotAllowed):
		return coded(fiber.StatusUnprocessableEntity, "consent_not_allowed", ErrorCodeConsentNotAllowed, locale)
	case errors.Is(err, ErrChildNotFound):
		msg := lang.Default().Trans("validation.in", map[string]string{"attribute": T("attributes.child_id", locale)}, locale)
		return failValidation(locale, jsonx.Obj("child_id", []string{msg}))
	}
	return fail(err, locale)
}

// render is one booking with its doctor, child name and consent.
func (bh *BookingHandlers) render(c fiber.Ctx, v View, bk store.TelemedBooking, doctors map[uint64]store.TelemedDoctor, now time.Time) (*jsonx.OrderedMap, error) {
	q := bh.b.q
	d, ok := doctors[bk.DoctorID]
	if !ok {
		var err error
		if d, err = q.GetDoctor(c, bk.DoctorID); err != nil {
			return nil, err
		}
		if doctors != nil {
			doctors[bk.DoctorID] = d
		}
	}
	parts := BookingParts{Doctor: d}
	if bk.ChildID.Valid {
		if ch, err := q.GetOwnedChild(c, store.GetOwnedChildParams{ID: uint64(bk.ChildID.Int64), OwnerID: bk.UserID}); err == nil { //nolint:gosec // G115: positive id
			parts.ChildName = ch.Name
		}
	}
	scopes, err := bh.b.ActiveScopes(c, bk.ID)
	if err != nil {
		return nil, err
	}
	parts.Scopes = scopes
	return v.BookingJSON(bk, parts, now), nil
}

func (bh *BookingHandlers) one(c fiber.Ctx, bk store.TelemedBooking, now time.Time) (*jsonx.OrderedMap, error) {
	v, err := bh.h.view(c)
	if err != nil {
		return nil, err
	}
	return bh.render(c, v, bk, nil, now)
}

// Index is GET /telemed/bookings?scope=upcoming|past|all — {items}.
func (bh *BookingHandlers) Index(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	scope, err := ValidateListScope(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	rows, err := bh.b.List(c, userID, scope)
	if err != nil {
		return err
	}
	v, err := bh.h.view(c)
	if err != nil {
		return err
	}
	doctors := map[uint64]store.TelemedDoctor{}
	items := make([]any, 0, len(rows))
	for _, bk := range rows {
		o, err := bh.render(c, v, bk, doctors, now)
		if err != nil {
			return err
		}
		items = append(items, o)
	}
	return httpx.OK(c, jsonx.Obj("items", jsonx.List(items)))
}

// Quote is GET /telemed/bookings/quote?doctor_id=&mode= (nbl_v17_ReviewBook): the visit type, the user's price with
// the Plus discount, the cancellation rule, the user's children («برای چه کسی؟»), the visit reasons and the consent
// scopes.
func (bh *BookingHandlers) Quote(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	doctorID, mode, err := ValidateQuote(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	qt, err := bh.b.Quote(c, userID, doctorID, mode)
	if err != nil {
		return failBooking(err, locale)
	}
	v, err := bh.h.view(c)
	if err != nil {
		return err
	}
	kids, err := bh.b.q.ListOwnedChildrenForBooking(c, userID)
	if err != nil {
		return err
	}
	children := make([]any, 0, len(kids))
	for _, k := range kids {
		children = append(children, jsonx.Obj("id", k.ID, "name", k.Name, "birth_date", k.BirthDate.String()))
	}
	items, err := bh.b.dir.Catalog().Items(c, GroupVisitReasons)
	if err != nil {
		return err
	}
	reasons := make([]any, 0, len(items))
	for _, it := range items {
		reasons = append(reasons, jsonx.Obj("code", it.Code, "title", v.Loc.Text(it.Title)))
	}
	var percent any
	if qt.Plus {
		percent = bh.b.opts.Config.PlusDiscountPercent
	}
	return httpx.OK(c, jsonx.Obj(
		"doctor", v.doctorBrief(qt.Doctor),
		"visit_type", v.VisitTypeJSON(qt.Visit),
		"price", PriceJSON(qt.Price.Price, qt.Price.Discount, qt.Price.Source, qt.Price.Total),
		"plus", qt.Plus,
		"plus_discount_percent", percent,
		"payments_available", bh.b.Payments() || qt.Price.Total == 0,
		"cancel_window_hours", int(CancelWindow/time.Hour),
		"hold_minutes", int(bh.b.opts.Config.HoldTTL/time.Minute),
		"for_whom", jsonx.List(ForWhom),
		"children", jsonx.List(children),
		"reasons", jsonx.List(reasons),
		"consent_scopes", jsonx.List(Scopes),
	))
}

// Store is POST /telemed/bookings: holds the slot (201 {booking, payment}); payment = {authority, redirect_url} to
// send the user to the gateway, null when nothing is charged (the booking is then confirmed already).
func (bh *BookingHandlers) Store(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	reasons, err := bh.reasonCodes(c)
	if err != nil {
		return err
	}
	in, err := ValidateBook(validation.Input(c), locale, now, reasons)
	if err != nil {
		return err
	}
	res, err := bh.b.Create(c, userID, in, bh.loc(c))
	if err != nil {
		return failBooking(err, locale)
	}
	o, err := bh.one(c, res.Booking, now)
	if err != nil {
		return err
	}
	var pay any
	msg := T("messages.booking_confirmed", locale)
	if res.Payment != nil {
		pay = jsonx.Obj("authority", res.Payment.Authority, "redirect_url", res.Payment.RedirectURL)
		msg = T("messages.booking_held", locale)
	}
	return httpx.Created(c, jsonx.Obj("booking", o, "payment", pay), msg)
}

func (bh *BookingHandlers) reasonCodes(c fiber.Ctx) ([]string, error) {
	items, err := bh.b.dir.Catalog().Items(c, GroupVisitReasons)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(items))
	for _, it := range items {
		codes = append(codes, it.Code)
	}
	return codes, nil
}

// Verify is POST /telemed/bookings/verify {reference, authority}: settles the payment server to server and
// confirms the booking — {booking, already_paid}. Idempotent.
func (bh *BookingHandlers) Verify(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	ref, authority, err := ValidateVerify(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	res, err := bh.b.Verify(c, userID, ref, authority, bh.loc(c))
	if err != nil {
		return failBooking(err, locale)
	}
	o, err := bh.one(c, res.Booking, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", o, "already_paid", res.AlreadyPaid), T("messages.booking_confirmed", locale))
}

func (bh *BookingHandlers) bookingID(c fiber.Ctx) (uint64, error) {
	id, ok := idParam(c, "id")
	if !ok {
		return 0, ErrBookingNotFound
	}
	return id, nil
}

// Show is GET /telemed/bookings/{id} — {booking}.
func (bh *BookingHandlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	id, err := bh.bookingID(c)
	if err != nil {
		return failBooking(err, locale)
	}
	bk, err := bh.b.Booking(c, userID, id)
	if err != nil {
		return failBooking(err, locale)
	}
	o, err := bh.one(c, bk, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", o))
}

// Cancel is POST /telemed/bookings/{id}/cancel — {booking}: free until 2 hours before the visit (409
// cancel_window_closed after), with a full refund.
func (bh *BookingHandlers) Cancel(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	id, err := bh.bookingID(c)
	if err != nil {
		return failBooking(err, locale)
	}
	bk, err := bh.b.Cancel(c, userID, id)
	if err != nil {
		return failBooking(err, locale)
	}
	o, err := bh.one(c, bk, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", o), T("messages.booking_cancelled", locale))
}

// Reschedule is POST /telemed/bookings/{id}/reschedule {starts_at} — {booking}: another free slot of the same doctor
// and mode, until 2 hours before the current time.
func (bh *BookingHandlers) Reschedule(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	id, err := bh.bookingID(c)
	if err != nil {
		return failBooking(err, locale)
	}
	start, err := ValidateReschedule(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	bk, err := bh.b.Reschedule(c, userID, id, start)
	if err != nil {
		return failBooking(err, locale)
	}
	o, err := bh.one(c, bk, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", o), T("messages.booking_rescheduled", locale))
}

// UpdateConsent is PUT /telemed/bookings/{id}/consent {cycle_summary?, bbt_lh?, assistant_summaries?} — {booking}.
func (bh *BookingHandlers) UpdateConsent(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	id, err := bh.bookingID(c)
	if err != nil {
		return failBooking(err, locale)
	}
	changes, err := ValidateConsent(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	bk, err := bh.b.SetConsent(c, userID, id, changes)
	if err != nil {
		return failBooking(err, locale)
	}
	o, err := bh.one(c, bk, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", o), T("messages.consent_saved", locale))
}

// DestroyConsent is DELETE /telemed/bookings/{id}/consent — {booking}: revokes every scope.
func (bh *BookingHandlers) DestroyConsent(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := bh.h.now(c), i18n.Locale(c)
	id, err := bh.bookingID(c)
	if err != nil {
		return failBooking(err, locale)
	}
	bk, err := bh.b.RevokeConsent(c, userID, id)
	if err != nil {
		return failBooking(err, locale)
	}
	o, err := bh.one(c, bk, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", o), T("messages.consent_revoked", locale))
}
