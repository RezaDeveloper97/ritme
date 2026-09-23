package content

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Banners is GET /banners (BannerController::index): the active, in-window banners of
// every slot (or only ?position= when it is a valid slot), grouped by slot in enum order;
// empty slots come back as [].
func (h *Handlers) Banners(c fiber.Ctx) error {
	locale := i18n.ResolveLocale(c, "")
	def := i18n.LanguagesOf(c).DefaultCode()

	positions := enums.BannerPositionCases()
	if only, _ := validation.Query(c).Get("position"); only != nil {
		if s, ok := only.(string); ok && phpTruthy(s) {
			if p, valid := enums.BannerPositionFrom(s); valid {
				positions = []enums.BannerPosition{p}
			}
		}
	}

	// Banner::active() binds now() as 'Y-m-d H:i:s' (whole seconds, Tehran wall-clock).
	now := h.now(c).Truncate(time.Second)
	rows, err := h.q.ListActiveBanners(c, store.ListActiveBannersParams{Now: sql.NullTime{Time: now, Valid: true}})
	if err != nil {
		return fmt.Errorf("content: banners: %w", err)
	}

	grouped := jsonx.NewObject()
	for _, p := range positions {
		items := []*jsonx.OrderedMap{}
		for _, r := range rows {
			if r.Position != string(p) {
				continue
			}
			var imageURL any
			if phpTruthy(r.ImagePath) {
				imageURL = PublicURL(h.appURL, r.ImagePath)
			}
			items = append(items, jsonx.Obj(
				"id", r.ID,
				"title", phpValue(i18n.Pick(nullJSON(r.Title), locale, def)),
				"image_url", imageURL,
				"position", r.Position,
				"link_url", nullString(r.LinkUrl),
				"link_type", nullString(r.LinkType),
			))
		}
		grouped.Set(string(p), jsonx.List(items))
	}
	return httpx.OK(c, jsonx.Obj("positions", grouped))
}

func nullString(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}
