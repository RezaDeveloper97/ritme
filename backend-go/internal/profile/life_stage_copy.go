package profile

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// The calm pregnancy exit of the mode switcher (B-N2-03, nbl_Me_Mode «اگر بارداری‌ات ادامه پیدا نکرد…»).
// Its copy is admin-edited: message_contents group `pregnancy_setup`, item `loss_exit` (registered in
// internal/admin/messages/registry, so admin-web «پیام‌ها» can create and edit it per language). No row is
// seeded — every text is null until an admin writes one, and the app falls back to its translation bundle
// (the same contract as GET /pregnancy/v2/setup-copy). roadmap CB-LOSS-01/02 replace this with the full loss path.

// LossCopyItem is the pregnancy_setup item holding the calm-exit copy.
const LossCopyItem = "loss_exit"

// LossCopyKeys are the texts of the item, in screen order: the confirm step (title, body, confirm, cancel)
// and the closing note shown after the exit (done_title, done_body, done_action).
var LossCopyKeys = []string{"title", "body", "confirm", "cancel", "done_title", "done_body", "done_action"}

// ShowLossCopy is GET /profile/life-stage/loss-copy: the admin-edited calm-exit texts in the request locale
// (else the default language's row); a text is null when no live row has it. Not tied to the user's data —
// the copy is the same for everyone, so it never reads pregnancy rows.
func (h *OnboardingHandlers) ShowLossCopy(c fiber.Ctx) error {
	if _, err := privacyUserID(c); err != nil {
		return err
	}
	l := v2.Lang{Locale: i18n.Locale(c), Default: i18n.LanguagesOf(c).DefaultCode()}
	p, err := v2.MessagePayload(c.Context(), h.pq, v2.SetupGroup, LossCopyItem, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, LossCopyJSON(p))
}

// LossCopyJSON keeps every key of LossCopyKeys: the payload's non-empty string, else null.
func LossCopyJSON(p map[string]any) *jsonx.OrderedMap {
	out := jsonx.Obj()
	for _, k := range LossCopyKeys {
		s, _ := p[k].(string)
		if s == "" {
			out.Set(k, nil)
			continue
		}
		out.Set(k, s)
	}
	return out
}
