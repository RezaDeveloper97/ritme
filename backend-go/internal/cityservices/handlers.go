package cityservices

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Handlers serve /api/v1/city-services/* (localized). The bookings routes are auth:api and every booking query is
// scoped to the caller; options / listings / listing / slots are public since CB-DIR-02 (directory_handlers.go).
type Handlers struct {
	store    *Store
	notifier BookingNotifier
	appURL   string
	clock    clock.Clock
	logger   *slog.Logger
	catalog  *catalog.Reader // CB-DIR-02: dir_amenities / dir_age_ranges (WithCatalog), nil = none
}

// NewHandlers wires the handlers; notifier may be nil (fake), clock nil (real).
func NewHandlers(store *Store, notifier BookingNotifier, appURL string, base clock.Clock, logger *slog.Logger) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	if notifier == nil {
		notifier = FakeNotifier{Logger: logger}
	}
	return &Handlers{store: store, notifier: notifier, appURL: appURL, clock: base, logger: logger}
}

// userMessages are the user-API lines (package-local on purpose: three languages, a handful of keys).
var userMessages = map[string]map[string]string{
	"validation_failed": {"fa": "اطلاعات واردشده معتبر نیست.", "en": "The given data was invalid.", "ar": "البيانات المدخلة غير صالحة."},
	"not_found":         {"fa": "پیدا نشد.", "en": "Not found.", "ar": "غير موجود."},
	"booking_created": {"fa": "درخواست شما ثبت شد؛ تیم ریتمی با شما تماس می\u200cگیرد.",
		"en": "Your request was sent; the Ritme team will contact you.", "ar": "تم إرسال طلبك وسيتواصل معك فريق ريتمي."},
	"booking_cancelled": {"fa": "درخواست لغو شد.", "en": "Request cancelled.", "ar": "تم إلغاء الطلب."},
	"not_cancellable": {"fa": "این درخواست دیگر قابل لغو نیست.", "en": "This request can no longer be cancelled.",
		"ar": "لا يمكن إلغاء هذا الطلب بعد الآن."},
	"not_requestable": {"fa": "این مرکز درخواست آنلاین نمی\u200cپذیرد.", "en": "This venue does not take online requests.",
		"ar": "هذا المكان لا يقبل الطلبات عبر الإنترنت."},
	"too_many_open": {"fa": "تعداد درخواست\u200cهای باز شما به حد مجاز رسیده است.", "en": "You have too many open requests.",
		"ar": "لديك عدد كبير من الطلبات المفتوحة."},
	"already_open": {"fa": "برای این مرکز یک درخواست باز دارید.", "en": "You already have an open request for this venue.",
		"ar": "لديك طلب مفتوح لهذا المكان."},
	"date_past": {"fa": "روز دلخواه نمی\u200cتواند گذشته باشد.", "en": "The preferred day cannot be in the past.",
		"ar": "لا يمكن أن يكون اليوم المفضل في الماضي."},
	"phone_format": {"fa": "شمارهٔ تماس معتبر نیست.", "en": "The phone number is not valid.", "ar": "رقم الهاتف غير صالح."},
}

// attributes are the field names in validation messages.
var attributes = map[string][]string{
	"fa": {"listing_id", "مرکز", "contact_name", "نام", "contact_phone", "شمارهٔ تماس", "preferred_date", "روز دلخواه", "child_age_range", "بازهٔ سنی کودک", "note", "توضیح"},
	"en": {"listing_id", "venue", "contact_name", "name", "contact_phone", "phone number", "preferred_date", "preferred day", "child_age_range", "child age range", "note", "note"},
	"ar": {"listing_id", "المكان", "contact_name", "الاسم", "contact_phone", "رقم الهاتف", "preferred_date", "اليوم المفضل", "child_age_range", "الفئة العمرية للطفل", "note", "ملاحظة"},
}

func msg(key, locale string) string {
	m := userMessages[key]
	if s, ok := m[locale]; ok {
		return s
	}
	return m["en"]
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

func (h *Handlers) labeler(c fiber.Ctx) Labeler {
	return Labeler{Locale: i18n.ResolveLocale(c, ""), Default: i18n.LanguagesOf(c).DefaultCode(), AppURL: h.appURL, URL: content.PublicURL}
}

func notFound(c fiber.Ctx) error {
	return httpx.Fail(fiber.StatusNotFound, msg("not_found", i18n.Locale(c)))
}

func fieldFail(c fiber.Ctx, field, text string) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, msg("validation_failed", i18n.Locale(c)), "errors", jsonx.Obj(field, []string{text}))
}

func parseID(s string) (uint64, bool) {
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil && n > 0 && len(s) < 20
}

// persian digits → ASCII, then separators out; the phone is validated on the normalised value.
var digitFold = strings.NewReplacer("۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	" ", "", "-", "", "(", "", ")", "")

func bookingRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("listing_id", "required", "integer", "min:1"),
		F("contact_name", "required", "string", "max:120"),
		F("contact_phone", "required", "string", "regex:/^\\+?[0-9]{7,15}$/"),
		F("preferred_date", "required", "date_format:Y-m-d"),
		F("child_age_range", "nullable", validation.In(AgeRanges...)),
		F("note", "nullable", "string", "max:500"),
	}
}

// StoreBooking is POST /city-services/bookings: a request for a `request`-mode listing. 201 with the booking.
// Only the allow-listed fields are read from the body; nothing about the user's health is accepted or stored.
func (h *Handlers) StoreBooking(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in := validation.Input(c)
	data := phpval.NewMap()
	for _, k := range []string{"listing_id", "contact_name", "contact_phone", "preferred_date", "child_age_range", "note"} {
		if v, ok := in.Get(k); ok {
			if s, isStr := v.(string); isStr && k == "contact_phone" {
				v = digitFold.Replace(s)
			}
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, bookingRules(), validation.Now(now),
		validation.Attributes(attributes[locale]...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("validation_failed", locale), "errors", v.ErrorBag())
	}
	get := func(k string) string { x, _ := data.Get(k); return strings.TrimSpace(phpval.ToString(x)) }
	listingID, _ := parseID(get("listing_id"))
	date, err := civildate.Parse(get("preferred_date"))
	if err != nil {
		return fieldFail(c, "preferred_date", msg("validation_failed", locale))
	}
	today := civildate.FromTime(now)
	if date.Before(today) {
		return fieldFail(c, "preferred_date", msg("date_past", locale))
	}
	if date.DiffDays(today) > MaxAdvanceDays || today.DiffDays(date) > MaxAdvanceDays {
		return fieldFail(c, "preferred_date", msg("validation_failed", locale))
	}
	l, err := h.store.PublicListing(c, listingID)
	if errors.Is(err, ErrNotFound) {
		return fieldFail(c, "listing_id", msg("not_found", locale))
	}
	if err != nil {
		return fmt.Errorf("cityservices: listing: %w", err)
	}
	if l.BookingMode != ModeRequest {
		return fieldFail(c, "listing_id", msg("not_requestable", locale))
	}
	total, forListing, err := h.store.OpenBookings(c, userID, listingID, today)
	if err != nil {
		return fmt.Errorf("cityservices: open bookings: %w", err)
	}
	if forListing > 0 {
		return fieldFail(c, "listing_id", msg("already_open", locale))
	}
	if total >= MaxOpenBookings {
		return fieldFail(c, "listing_id", msg("too_many_open", locale))
	}
	optional := func(k string) (n sql.NullString) {
		if s := get(k); s != "" {
			n.Valid, n.String = true, s
		}
		return n
	}
	b, err := h.store.CreateBooking(c, userID, NewBooking{
		ListingID: listingID, ContactName: get("contact_name"), ContactPhone: get("contact_phone"), PreferredDate: date,
		ChildAgeRange: optional("child_age_range"), Note: optional("note"),
	}, now)
	if err != nil {
		return err
	}
	if err := h.notifier.BookingRequested(c, BookingNotice{
		BookingID: b.ID, ListingID: listingID, ContactName: b.ContactName, ContactPhone: b.ContactPhone,
		PreferredDate: b.PreferredDate, ChildAgeRange: b.ChildAgeRange.String, Note: b.Note.String,
	}); err != nil {
		h.logger.WarnContext(c, "cityservices: notifier failed", slog.Uint64("booking_id", b.ID)) // the queue is the truth
	}
	return httpx.Created(c, jsonx.Obj("booking", h.labeler(c).BookingJSON(b)), msg("booking_created", locale))
}

// Bookings is GET /city-services/bookings → the caller's own requests, newest first («رزروها»).
func (h *Handlers) Bookings(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	rows, err := h.store.UserBookings(c, userID)
	if err != nil {
		return fmt.Errorf("cityservices: bookings: %w", err)
	}
	lb := h.labeler(c)
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, b := range rows {
		items = append(items, lb.BookingJSON(b))
	}
	return httpx.OK(c, jsonx.Obj("bookings", items))
}

// CancelBooking is POST /city-services/bookings/{id}/cancel: owner only (404 for anyone else's), 422 once decided.
func (h *Handlers) CancelBooking(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c)
	}
	if _, err := h.store.UserBooking(c, userID, id); errors.Is(err, ErrNotFound) {
		return notFound(c)
	} else if err != nil {
		return fmt.Errorf("cityservices: booking: %w", err)
	}
	done, err := h.store.CancelBooking(c, userID, id, h.now(c))
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if !done {
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg("not_cancellable", locale))
	}
	b, err := h.store.UserBooking(c, userID, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("booking", h.labeler(c).BookingJSON(b)), msg("booking_cancelled", locale))
}
