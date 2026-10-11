package analyticsevents

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Handlers serve POST /analytics/events (auth:api, per-user throttled).
type Handlers struct {
	store    *Store
	consents *consent.Service
	clock    clock.Clock
}

// NewHandlers wires the handlers.
func NewHandlers(store *Store, consents *consent.Service, base clock.Clock) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	return &Handlers{store: store, consents: consents, clock: base}
}

var messages = map[string]map[string]string{
	"en": {
		"unknown_event": "This event is not allowed.",
		"unknown_prop":  "This property is not allowed.",
		"health_prop":   "Health data is never accepted in analytics events.",
		"bad_prop":      "This property value is not allowed.",
		"bad_time":      "The time is invalid, in the future or older than 72 hours.",
		"bad_shape":     "The request shape is invalid.",
		"too_many":      "At most 25 events per request.",
		"empty":         "Send at least one event.",
		"failed":        "The given data was invalid.",
	},
	"fa": {
		"unknown_event": "این رویداد مجاز نیست.",
		"unknown_prop":  "این ویژگی مجاز نیست.",
		"health_prop":   "داده\u200cهای سلامت هرگز در رویدادهای تحلیلی پذیرفته نمی\u200cشود.",
		"bad_prop":      "مقدار این ویژگی مجاز نیست.",
		"bad_time":      "زمان نامعتبر است، در آینده است یا بیش از ۷۲ ساعت پیش است.",
		"bad_shape":     "ساختار درخواست نامعتبر است.",
		"too_many":      "در هر درخواست حداکثر ۲۵ رویداد مجاز است.",
		"empty":         "دست\u200cکم یک رویداد بفرستید.",
		"failed":        "داده\u200cهای ارسالی نامعتبر است.",
	},
}

func msg(locale, code string) string {
	if m, ok := messages[locale]; ok {
		return m[code]
	}
	return messages["en"][code]
}

// Ingest is POST /analytics/events {events:[{name, props?, occurred_at?}]} → {accepted, tracking}. 422 on any unknown
// name/prop (the whole batch is refused, nothing is stored). A user without the `anonymous_stats` consent gets
// {accepted:0, tracking:false} (200): nothing is stored and the client stops sending.
func (h *Handlers) Ingest(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	body := c.Body()
	if len(body) > MaxBodyBytes {
		return httpx.Fail(fiber.StatusRequestEntityTooLarge, msg(locale, "failed"))
	}
	now := clock.FromContext(c, h.clock).Now().In(civildate.Tehran)
	evs, probs := Parse(body, now)
	if len(probs) > 0 {
		ve := httpx.NewValidationError()
		for _, p := range probs {
			ve.Add(p.Field, msg(locale, p.Code))
		}
		return ve
	}
	st, err := h.consents.Get(c, uid, consent.AnonymousStats)
	if err != nil {
		return err
	}
	if !st.Valid() {
		return httpx.OK(c, jsonx.Obj("accepted", 0, "tracking", false))
	}
	if err := h.store.Insert(c, uid, evs, now); err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("accepted", len(evs), "tracking", true))
}
