package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/bannerads"
	publiccontent "github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Banner upload limits (BannerController): jpeg/png/webp, ≤ 4 MB, at least 800×400,
// stored as uploaded under banners/ (no optimisation). 1080×540 is the recommended size.
const (
	BannerImageDir     = "banners"
	bannerImageMaxKB   = 4096
	bannerMinWidth     = 800
	bannerMinHeight    = 400
	bannerRecommendedW = 1080
	bannerRecommendedH = 540
)

func (h *Handlers) bannerJSON(b *store.Banner) *jsonx.OrderedMap {
	var url any
	if b.ImagePath != "" {
		url = publiccontent.PublicURL(h.appURL, b.ImagePath)
	}
	return jsonx.Obj(
		"id", b.ID,
		"title", form.NullRaw(b.Title),
		"image_path", b.ImagePath,
		"image_url", url,
		"position", b.Position,
		"link_url", httpadmin.NullString(b.LinkUrl),
		"link_type", httpadmin.NullString(b.LinkType),
		"starts_at", httpadmin.Time(b.StartsAt),
		"ends_at", httpadmin.Time(b.EndsAt),
		"is_active", b.IsActive,
		"sort_order", b.SortOrder,
		"created_at", httpadmin.Time(b.CreatedAt),
		"updated_at", httpadmin.Time(b.UpdatedAt),
		// B-N10-02 (D-93), additive: sponsored slots.
		"kind", b.Kind,
		"advertiser", httpadmin.NullString(b.Advertiser),
		"cta_label", form.NullRaw(rawOf(b.CtaLabel)),
		"modes", bannerModes(b.Modes),
	)
}

func rawOf(s sql.NullString) db.NullRawJSON {
	if !s.Valid {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: []byte(s.String), Valid: true}
}

// bannerModes is the stored targeting list ([] = every mode).
func bannerModes(s sql.NullString) []string {
	out := []string{}
	if s.Valid {
		_ = json.Unmarshal([]byte(s.String), &out)
	}
	return out
}

func (h *Handlers) banners() resource[store.Banner] {
	return resource[store.Banner]{
		name: "Banner", key: "banner", get: h.q.GetBanner, json: h.bannerJSON,
		id: func(b *store.Banner) uint64 { return b.ID },
		toggle: func(ctx context.Context, now sql.NullTime, id uint64) error {
			return h.q.ToggleBanner(ctx, store.ToggleBannerParams{Now: now, ID: id})
		},
		del: h.q.DeleteBanner,
	}
}

// ListBanners is GET /banners (position, sort_order, newest first).
func (h *Handlers) ListBanners(c fiber.Ctx) error {
	total, err := h.q.CountAdminBanners(c.Context())
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminBanners(c.Context(), store.ListAdminBannersParams{
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, h.bannerJSON(&rows[i]))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// BannerOptions is GET /banners/options.
func (h *Handlers) BannerOptions(c fiber.Ctx) error {
	loc := locale(c)
	return httpadmin.OK(c, jsonx.Obj(
		"positions", options(enums.BannerPositionOptions(loc)),
		"kinds", []string{bannerads.KindBanner, bannerads.KindAd},
		"ad_slots", bannerads.Slots,
		"modes", bannerads.ValidModes(),
		"link_types", options(enums.BannerLinkTypeOptions(loc)),
		"image", jsonx.Obj("max_kb", bannerImageMaxKB, "min_width", bannerMinWidth, "min_height", bannerMinHeight,
			"recommended_width", bannerRecommendedW, "recommended_height", bannerRecommendedH,
			"types", []string{"jpeg", "png", "webp"}),
	))
}

// ShowBanner is GET /banners/:id.
func (h *Handlers) ShowBanner(c fiber.Ctx) error { return h.banners().show(c) }

func bannerRules(c fiber.Ctx) validation.Rules {
	rules := form.Translatable(c, "title", false, "max:255")
	rules = append(rules, form.Translatable(c, "cta_label", false, "max:60")...)
	return append(rules,
		validation.F("position", "required", validation.In(append(enums.BannerPositionValues(), bannerads.Slots...)...)),
		validation.F("kind", "nullable", validation.In(bannerads.KindBanner, bannerads.KindAd)),
		validation.F("advertiser", "nullable|string|max:120"),
		validation.F("modes", "nullable|array"),
		validation.F("modes.*", validation.In(bannerads.ValidModes()...)),
		validation.F("link_type", "nullable", validation.In(enums.BannerLinkTypeValues()...)),
		validation.F("link_url", "nullable|string|max:1000"),
		validation.F("starts_at", "nullable|date"),
		validation.F("ends_at", "nullable|date|after_or_equal:starts_at"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
	)
}

// checkBannerKind keeps kind and slot consistent: an ad sits in an ad_* slot, a plain banner in a home_* slot. On
// update the kind not sent is the stored one.
func checkBannerKind(c fiber.Ctx, cur *store.Banner) form.Check {
	return func(in phpval.Map, add form.Add) error {
		pos, _ := in.Get("position")
		p, ok := pos.(string)
		if !ok {
			return nil
		}
		kind := bannerKind(in, cur)
		if (kind == bannerads.KindAd) != bannerads.IsSlot(p) {
			add("position", form.Msg(c, "validation.in", "position"))
		}
		return nil
	}
}

func bannerKind(in phpval.Map, cur *store.Banner) string {
	if k, ok := in.Get("kind"); ok && k != nil {
		return phpval.ToString(k)
	}
	if cur != nil && cur.Kind != "" {
		return cur.Kind
	}
	return bannerads.KindBanner
}

// bannerModesJSON is the targeting column: NULL for "every mode" (missing or empty list).
func bannerModesJSON(data phpval.Map, cur sql.NullString, keep bool) sql.NullString {
	v, ok := data.Get("modes")
	if !ok {
		if keep {
			return cur
		}
		return sql.NullString{}
	}
	list, _ := phpval.Packed(v).([]any)
	if len(list) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(form.JSON(list)), Valid: true}
}

// checkBannerLink: link_url is required_with:link_type, and must be a `url` for
// external links. Internal links are app paths: they must start with a single `/`
// (D-16; Laravel accepts any string, incl. `javascript:` and protocol-relative
// `//host`). On update the link type not sent is the stored one (cur), as saved.
func checkBannerLink(c fiber.Ctx, cur *store.Banner) form.Check {
	return func(in phpval.Map, add form.Add) error {
		linkType, _ := in.Get("link_type")
		link, _ := in.Get("link_url")
		if linkType != nil && link == nil {
			add("link_url", form.Msg(c, "validation.required_with", "link_url",
				"values", httpadmin.AttributeName(c, "link_type")))
			return nil
		}
		s, isStr := link.(string)
		if !isStr || s == "" {
			return nil
		}
		typ := phpval.ToString(linkType)
		if linkType == nil && cur != nil && cur.LinkType.Valid {
			typ = cur.LinkType.String
		}
		switch {
		case typ == string(enums.BannerLinkTypeExternal) && linkType != nil && !form.IsURL(s):
			add("link_url", form.Msg(c, "validation.url", "link_url"))
		case typ == string(enums.BannerLinkTypeInternal) && !isInternalPath(s):
			add("link_url", form.Msg(c, "validation.regex", "link_url"))
		}
		return nil
	}
}

// isInternalPath reports whether s is a same-origin app path: a leading `/` that is not
// followed by another `/` or `\` (browsers read both as a protocol-relative URL) and no
// control characters (browsers drop tabs/newlines, so "/\t/host" would become "//host")
// (D-16).
func isInternalPath(s string) bool {
	if len(s) == 0 || s[0] != '/' {
		return false
	}
	if len(s) > 1 && (s[1] == '/' || s[1] == '\\') {
		return false
	}
	for _, r := range s {
		if r < ' ' || r == 0x7f {
			return false
		}
	}
	return true
}

// validateBanner validates a create (cur == nil) or an update of cur.
func (h *Handlers) validateBanner(c fiber.Ctx, cur *store.Banner) (phpval.Map, *media.Image, error) {
	var img *media.Image
	data, err := form.Validate(c, bannerRules(c), checkBannerLink(c, cur), checkBannerKind(c, cur),
		form.ImageRule{Field: "image", Required: cur == nil, MaxKB: bannerImageMaxKB,
			MinWidth: bannerMinWidth, MinHeight: bannerMinHeight}.Check(c, &img))
	return data, img, err
}

// bannerLink drops both link fields when no URL was given; otherwise the URL and the
// link type (kept from cur when not sent, as Eloquent's update does).
func bannerLink(data phpval.Map, cur *store.Banner) (url, typ sql.NullString) {
	url = form.Str(data, "link_url")
	if !url.Valid || url.String == "" {
		return sql.NullString{}, sql.NullString{}
	}
	if cur == nil {
		return url, form.Str(data, "link_type")
	}
	return url, form.KeepStr(data, "link_type", cur.LinkType)
}

// StoreBanner is POST /banners (multipart: `image` is required).
func (h *Handlers) StoreBanner(c fiber.Ctx) error {
	data, img, err := h.validateBanner(c, nil)
	if err != nil {
		return err
	}
	rel, err := h.disk.StoreOriginal(BannerImageDir, *img)
	if err != nil {
		return err
	}
	link, linkType := bannerLink(data, nil)
	res, err := h.q.CreateBanner(c.Context(), store.CreateBannerParams{
		Title: form.NullJSON(data, "title"), ImagePath: rel, Position: httpadmin.String(data, "position"),
		LinkUrl: link, LinkType: linkType,
		StartsAt: form.KeepTime(data, "starts_at", sql.NullTime{}), EndsAt: form.KeepTime(data, "ends_at", sql.NullTime{}),
		IsActive: httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0), Now: h.now(c),
		Kind: bannerKind(data, nil), Advertiser: form.Str(data, "advertiser"), CtaLabel: jsonStr(form.NullJSON(data, "cta_label")),
		Modes: bannerModesJSON(data, sql.NullString{}, false),
	})
	if err != nil {
		h.deleteFile(c, rel)
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "banner.create", "banner", id)
	return h.banners().respond(c, id, true, "Banner created.")
}

// UpdateBanner is PUT (or multipart POST) /banners/:id; a new `image` replaces the file.
func (h *Handlers) UpdateBanner(c fiber.Ctx) error {
	cur, err := find(c, "Banner", h.q.GetBanner)
	if err != nil {
		return err
	}
	data, img, err := h.validateBanner(c, &cur)
	if err != nil {
		return err
	}
	imagePath := cur.ImagePath
	if img != nil {
		if imagePath, err = h.disk.StoreOriginal(BannerImageDir, *img); err != nil {
			return err
		}
	}
	link, linkType := bannerLink(data, &cur)
	if err := h.q.UpdateBanner(c.Context(), store.UpdateBannerParams{
		Title: form.KeepJSON(data, "title", cur.Title), ImagePath: imagePath,
		Position: httpadmin.String(data, "position"), LinkUrl: link, LinkType: linkType,
		StartsAt: form.KeepTime(data, "starts_at", cur.StartsAt), EndsAt: form.KeepTime(data, "ends_at", cur.EndsAt),
		IsActive: httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0),
		Kind: bannerKind(data, &cur), Advertiser: form.KeepStr(data, "advertiser", cur.Advertiser),
		CtaLabel: jsonStr(form.KeepJSON(data, "cta_label", rawOf(cur.CtaLabel))),
		Modes:    bannerModesJSON(data, cur.Modes, true),
		Now:      h.now(c), ID: cur.ID,
	}); err != nil {
		if img != nil {
			h.deleteFile(c, imagePath)
		}
		return err
	}
	if img != nil {
		h.deleteFile(c, cur.ImagePath)
	}
	httpadmin.Audit(c, h.logger, "banner.update", "banner", cur.ID, slog.Bool("image", img != nil))
	return h.banners().respond(c, cur.ID, false, "Banner updated.")
}

// DestroyBanner is DELETE /banners/:id (the image file goes too).
func (h *Handlers) DestroyBanner(c fiber.Ctx) error {
	cur, err := find(c, "Banner", h.q.GetBanner)
	if err != nil {
		return err
	}
	res, err := h.q.DeleteBanner(c.Context(), cur.ID)
	if err := deleted(res, err, "Banner"); err != nil {
		return err
	}
	h.deleteFile(c, cur.ImagePath)
	httpadmin.Audit(c, h.logger, "banner.delete", "banner", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Banner deleted.")
}

// ToggleBanner is POST /banners/:id/toggle (is_active).
func (h *Handlers) ToggleBanner(c fiber.Ctx) error {
	return h.banners().doToggle(c, h.logger, "Status changed.")
}

func jsonStr(j db.NullRawJSON) sql.NullString {
	if !j.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: string(j.V), Valid: true}
}

// BannerStats is GET /banners/:id/stats?days=30: the per-day impression / click counters (aggregate only, Tehran
// days, newest first) plus their totals and the click-through rate.
func (h *Handlers) BannerStats(c fiber.Ctx) error {
	cur, err := find(c, "Banner", h.q.GetBanner)
	if err != nil {
		return err
	}
	days := fiber.Query[int](c, "days", 30)
	if days < 1 || days > 366 {
		days = 30
	}
	since := civildate.InTehran(h.now(c).Time.AddDate(0, 0, -(days - 1)))
	rows, err := h.q.ListBannerStats(c.Context(), store.ListBannerStatsParams{BannerID: cur.ID, Since: since})
	if err != nil {
		return err
	}
	var imp, clk int64
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		imp += int64(r.Impressions)
		clk += int64(r.Clicks)
		items = append(items, jsonx.Obj("day", r.Day.String(), "impressions", r.Impressions, "clicks", r.Clicks))
	}
	ctr := 0.0
	if imp > 0 {
		ctr = float64(clk) / float64(imp)
	}
	return httpadmin.OK(c, jsonx.Obj("banner_id", cur.ID, "days", days, "since", since.String(),
		"totals", jsonx.Obj("impressions", imp, "clicks", clk, "ctr", ctr), "items", items))
}
