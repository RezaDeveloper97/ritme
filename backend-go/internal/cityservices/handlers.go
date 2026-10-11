package cityservices

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
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
	// CB-DIR-13: the public directory reads' throttle (was the framework's English «Too Many Attempts.»).
	"too_many_reads": {"fa": "درخواست‌ها زیاد شد؛ چند ثانیه صبر کن و دوباره امتحان کن.",
		"en": "Too many requests; wait a few seconds and try again.", "ar": "طلبات كثيرة؛ انتظر بضع ثوانٍ ثم حاول مرة أخرى."},
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
	// CB-DIR-03 (directory bookings, reviews, reports).
	"slot_required": {"fa": "یک زمان را انتخاب کن.", "en": "Pick a time.", "ar": "اختر وقتًا."},
	"slot_unavailable": {"fa": "این زمان دیگر در دسترس نیست.", "en": "This time is no longer available.",
		"ar": "هذا الوقت لم يعد متاحًا."},
	"slot_full": {"fa": "ظرفیت این زمان پر شد؛ زمان دیگری انتخاب کن.", "en": "This time is full; pick another one.",
		"ar": "هذا الوقت ممتلئ؛ اختر وقتًا آخر."},
	"slot_taken": {"fa": "برای این زمان یک رزرو باز داری.", "en": "You already have an open booking for this time.",
		"ar": "لديك حجز مفتوح لهذا الوقت."},
	"service_mismatch": {"fa": "این زمان برای خدمت انتخاب\u200cشده نیست.", "en": "This time is not for the chosen service.",
		"ar": "هذا الوقت ليس للخدمة المختارة."},
	"age_mismatch": {"fa": "سن کودک با این خدمت جور نیست.", "en": "The child's age does not fit this service.",
		"ar": "عمر الطفل لا يناسب هذه الخدمة."},
	"cancel_window_passed": {"fa": "مهلت لغو رایگان گذشته است؛ با مجموعه تماس بگیر.",
		"en": "The free cancellation window has passed; please contact the venue.",
		"ar": "انتهت مهلة الإلغاء المجاني؛ تواصل مع المكان."},
	"not_reschedulable": {"fa": "زمان این رزرو قابل تغییر نیست.", "en": "This booking can no longer be rescheduled.",
		"ar": "لا يمكن تغيير موعد هذا الحجز."},
	"rescheduled": {"fa": "درخواست تغییر زمان ثبت شد؛ تیم ریتمی تأیید می\u200cکند.",
		"en": "Your new time was requested; the Ritme team will confirm it.", "ar": "تم طلب الموعد الجديد وسيؤكده فريق ريتمي."},
	"not_reviewable": {"fa": "بعد از انجام رزرو می\u200cتوانی نظر بدهی.", "en": "You can review a booking once it is completed.",
		"ar": "يمكنك التقييم بعد إتمام الحجز."},
	"already_reviewed": {"fa": "برای این رزرو قبلاً نظر داده\u200cای.", "en": "You have already reviewed this booking.",
		"ar": "لقد قيّمت هذا الحجز من قبل."},
	"review_received": {"fa": "ممنون! نظرت بعد از بررسی نمایش داده می\u200cشود.",
		"en": "Thank you! Your review will appear after moderation.", "ar": "شكرًا! سيظهر تقييمك بعد المراجعة."},
	"already_reported": {"fa": "گزارش قبلی تو دربارهٔ این مجموعه در حال بررسی است.",
		"en": "Your earlier report about this venue is being reviewed.", "ar": "بلاغك السابق عن هذا المكان قيد المراجعة."},
	"report_received": {"fa": "ممنون! گزارشت به تیم ریتمی رسید.", "en": "Thank you! Your report reached the Ritme team.",
		"ar": "شكرًا! وصل بلاغك إلى فريق ريتمي."},
}

// attributes are the field names in validation messages.
var attributes = map[string][]string{
	"fa": {"listing_id", "مرکز", "contact_name", "نام", "contact_phone", "شمارهٔ تماس", "preferred_date", "روز دلخواه", "child_age_range", "بازهٔ سنی کودک", "note", "توضیح",
		"service_id", "خدمت", "slot_id", "زمان", "child_id", "کودک", "rating", "امتیاز", "score_cleanliness", "امتیاز تمیزی", "score_instructor", "امتیاز مربی", "score_amenities", "امتیاز امکانات", "body", "متن نظر", "reason", "دلیل", "scope", "بازه"},
	"en": {"listing_id", "venue", "contact_name", "name", "contact_phone", "phone number", "preferred_date", "preferred day", "child_age_range", "child age range", "note", "note",
		"service_id", "service", "slot_id", "time", "child_id", "child", "rating", "rating", "score_cleanliness", "cleanliness score", "score_instructor", "instructor score", "score_amenities", "amenities score", "body", "review", "reason", "reason", "scope", "scope"},
	"ar": {"listing_id", "المكان", "contact_name", "الاسم", "contact_phone", "رقم الهاتف", "preferred_date", "اليوم المفضل", "child_age_range", "الفئة العمرية للطفل", "note", "ملاحظة",
		"service_id", "الخدمة", "slot_id", "الوقت", "child_id", "الطفل", "rating", "التقييم", "score_cleanliness", "تقييم النظافة", "score_instructor", "تقييم المدرب", "score_amenities", "تقييم المرافق", "body", "نص التقييم", "reason", "السبب", "scope", "النطاق"},
}

// TooManyReads is the localized 429 of the public directory reads' IP throttle (CB-DIR-13): the controller envelope
// {success:false, message, error_code:"too_many_requests", retry_after} plus the Retry-After / X-RateLimit-* headers.
func TooManyReads(locale string, retryAfter int, headers map[string]string) *httpx.FailError {
	e := httpx.Fail(fiber.StatusTooManyRequests, msg("too_many_reads", locale),
		"error_code", "too_many_requests", "retry_after", retryAfter)
	for k, v := range headers {
		e = e.WithHeader(k, v)
	}
	return e
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
	return Labeler{Locale: i18n.ResolveLocale(c, ""), Default: i18n.LanguagesOf(c).DefaultCode(), AppURL: h.appURL, URL: content.PublicURL,
		Now: h.now(c)} // CB-DIR-03
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
