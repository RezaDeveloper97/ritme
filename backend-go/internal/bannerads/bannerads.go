// Package bannerads serves the sponsored home slots of the banner system (bloom B-N10-02, D-93, Go only):
// GET /banners/ads (kind='ad' banners grouped by ad slot, targeted by the app mode the client sends and nothing else)
// and POST /banners/:id/events (per-day impression / click aggregate counters; no user id, device or health context is
// stored). The existing GET /banners is untouched and never returns ads.
package bannerads

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Slots are the ad slot values stored in banners.position for kind='ad' (342x170, 166x166, 342x76), in output order.
var Slots = []string{"ad_slider", "ad_square", "ad_strip"}

// Kinds of banners.kind.
const (
	KindBanner = "banner"
	KindAd     = "ad"
)

// IsSlot reports whether s is an ad slot.
func IsSlot(s string) bool {
	for _, v := range Slots {
		if v == s {
			return true
		}
	}
	return false
}

// ValidModes are the targetable app modes (the stored life modes plus the male companion mode).
func ValidModes() []string {
	return append(enums.LifeModeValues(), string(enums.LifeModeCompanion))
}

// Querier is the subset of the content store the handlers use.
type Querier interface {
	ListActiveAds(ctx context.Context, arg store.ListActiveAdsParams) ([]store.ListActiveAdsRow, error)
	GetActiveAdID(ctx context.Context, arg store.GetActiveAdIDParams) (uint64, error)
	BumpBannerImpression(ctx context.Context, arg store.BumpBannerImpressionParams) error
	BumpBannerClick(ctx context.Context, arg store.BumpBannerClickParams) error
}

// CommercialPolicy is internal/teen.Policy (no ads for a teen-mode account).
type CommercialPolicy interface {
	AllowsCommercial(ctx context.Context, userID uint64) (bool, error)
}

// Handlers are the ad controllers.
type Handlers struct {
	q      Querier
	appURL string
	policy CommercialPolicy
	clock  clock.Clock
	logger *slog.Logger
}

// New builds the handlers.
func New(q Querier, appURL string, policy CommercialPolicy, clk clock.Clock, logger *slog.Logger) *Handlers {
	if clk == nil {
		clk = clock.Real{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{q: q, appURL: appURL, policy: policy, clock: clk, logger: logger}
}

func (h *Handlers) now(c fiber.Ctx) time.Time { return clock.FromContext(c, h.clock).Now() }

// Targets reports whether an ad with the stored modes JSON (NULL / [] = every mode) is shown in mode.
func Targets(modes sql.NullString, mode string) bool {
	if !modes.Valid || modes.String == "" {
		return true
	}
	var list []string
	if err := json.Unmarshal([]byte(modes.String), &list); err != nil || len(list) == 0 {
		return true
	}
	for _, m := range list {
		if m == mode {
			return true
		}
	}
	return false
}

// List is GET /banners/ads?mode=<life mode>: {positions: {ad_slider: [], ad_square: [], ad_strip: []}}. The server
// answers from the mode the client sends (never from logs or predictions); teen mode (query or stored account mode)
// gets every slot empty. A missing / unknown mode only matches ads that target every mode. The advertiser name is
// never returned; the «تبلیغ» label is rendered by the client.
func (h *Handlers) List(c fiber.Ctx) error {
	locale := i18n.ResolveLocale(c, "")
	def := i18n.LanguagesOf(c).DefaultCode()
	mode := c.Query("mode")
	suppressed := mode == string(enums.LifeModeTeen)
	if !suppressed && h.policy != nil {
		if uid, ok := auth.CurrentUserID(c); ok {
			allowed, err := h.policy.AllowsCommercial(c, uid)
			if err != nil {
				return fmt.Errorf("bannerads: policy: %w", err)
			}
			suppressed = !allowed
		}
	}
	var rows []store.ListActiveAdsRow
	if !suppressed {
		var err error
		rows, err = h.q.ListActiveAds(c, store.ListActiveAdsParams{
			Now: sql.NullTime{Time: h.now(c).Truncate(time.Second), Valid: true}})
		if err != nil {
			return fmt.Errorf("bannerads: list: %w", err)
		}
	}
	grouped := jsonx.NewObject()
	for _, slot := range Slots {
		items := []*jsonx.OrderedMap{}
		for _, r := range rows {
			if r.Position != slot || !Targets(r.Modes, mode) {
				continue
			}
			var imageURL any
			if r.ImagePath != "" {
				imageURL = content.PublicURL(h.appURL, r.ImagePath)
			}
			var link, linkType, cta any
			if r.LinkUrl.Valid {
				link = r.LinkUrl.String
			}
			if r.LinkType.Valid {
				linkType = r.LinkType.String
			}
			if r.CtaLabel.Valid {
				if s := i18n.PickString(json.RawMessage(r.CtaLabel.String), locale, def); s != "" {
					cta = s
				}
			}
			var title any
			if r.Title.Valid {
				if s := i18n.PickString(r.Title.V, locale, def); s != "" {
					title = s
				}
			}
			items = append(items, jsonx.Obj("id", r.ID, "title", title, "image_url", imageURL, "position", r.Position,
				"cta_label", cta, "link_url", link, "link_type", linkType))
		}
		grouped.Set(slot, jsonx.List(items))
	}
	c.Set("Cache-Control", "no-store")
	return httpx.OK(c, jsonx.Obj("positions", grouped))
}

// Event is POST /banners/:id/events {event: impression|click}: bumps the banner's counter for today (Tehran). The
// authenticated identity is used by the throttle only; nothing about the caller reaches the database.
func (h *Handlers) Event(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return httpx.ModelNotFound("Banner", c.Params("id"))
	}
	var body struct {
		Event string `json:"event"`
	}
	if len(c.Body()) > 256 || json.Unmarshal(c.Body(), &body) != nil || (body.Event != "impression" && body.Event != "click") {
		ve := httpx.NewValidationError()
		ve.Add("event", "The selected event is invalid.")
		return ve
	}
	now := h.now(c)
	if _, err := h.q.GetActiveAdID(c, store.GetActiveAdIDParams{ID: id,
		Now: sql.NullTime{Time: now.Truncate(time.Second), Valid: true}}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ModelNotFound("Banner", id)
		}
		return fmt.Errorf("bannerads: event lookup: %w", err)
	}
	day := civildate.InTehran(now)
	if body.Event == "click" {
		err = h.q.BumpBannerClick(c, store.BumpBannerClickParams{BannerID: id, Day: day})
	} else {
		err = h.q.BumpBannerImpression(c, store.BumpBannerImpressionParams{BannerID: id, Day: day})
	}
	if err != nil {
		return fmt.Errorf("bannerads: event: %w", err)
	}
	return httpx.OK(c, jsonx.Obj("recorded", true))
}
