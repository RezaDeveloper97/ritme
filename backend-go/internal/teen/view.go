package teen

import (
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func isoTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.In(civildate.Tehran).Format(time.RFC3339)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ProfileJSON is the stored answers (null before onboarding).
func ProfileJSON(p *Profile) any {
	if p == nil {
		return nil
	}
	return jsonx.Obj(
		"age_band", string(p.AgeBand),
		"menarche", string(p.Menarche),
		"parent_note", nullable(p.ParentNote),
		"updated_at", isoTime(p.UpdatedAt),
	)
}

// AllowsJSON is the commercial flag set (false = hidden for this account).
func AllowsJSON(a Allows) *jsonx.OrderedMap {
	return jsonx.Obj(
		"shop", a.Shop,
		"banners", a.Banners,
		"ads", a.Ads,
		"plus_upsell", a.PlusUpsell,
		"commercial_recommendations", a.CommercialRecommendations,
	)
}

// ProfileEnvelope is GET|PUT /teen/profile.
func ProfileEnvelope(p *Profile, a Allows, teenMode bool) *jsonx.OrderedMap {
	return jsonx.Obj(
		"profile", ProfileJSON(p),
		"needs_onboarding", p == nil,
		"is_teen_mode", teenMode,
		"allows", AllowsJSON(a),
	)
}

func itemJSON(it *catalog.Item, l catalog.Localizer) any {
	if it == nil {
		return nil
	}
	return jsonx.Obj("code", it.Code, "title", l.Text(it.Title), "body", l.Text(it.Body), "meta", l.Meta(it.Meta),
		"needs_review", it.NeedsReview)
}

func itemsJSON(items []catalog.Item, l catalog.Localizer) []any {
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, itemJSON(&items[i], l))
	}
	return out
}

// KitJSON is the checklist.
func KitJSON(k Kit, l catalog.Localizer) *jsonx.OrderedMap {
	items := make([]*jsonx.OrderedMap, 0, len(k.Items))
	for _, it := range k.Items {
		items = append(items, jsonx.Obj("code", it.Item.Code, "title", l.Text(it.Item.Title), "body", l.Text(it.Item.Body),
			"needs_review", it.Item.NeedsReview, "checked", it.Checked))
	}
	return jsonx.Obj("items", items, "checked_count", k.Checked, "total", len(k.Items), "ready", k.Ready())
}

func grantsJSON(g companion.Grants) *jsonx.OrderedMap {
	out := jsonx.NewObject()
	for _, s := range companion.TeenSections {
		out.Set(string(s), string(g.Of(s)))
	}
	return out
}

func viewJSON(v ParentView) *jsonx.OrderedMap {
	var week, kit, note any
	if v.PeriodWeek != nil {
		week = string(*v.PeriodWeek)
	}
	if v.KitReady != nil {
		kit = *v.KitReady
	}
	if v.Note != nil {
		note = *v.Note
	}
	return jsonx.Obj("next_period_week", week, "kit_ready", kit, "note", note)
}

// TodayJSON is GET /teen/today.
func TodayJSON(t Today, l catalog.Localizer) *jsonx.OrderedMap {
	links := make([]*jsonx.OrderedMap, 0, len(t.ParentLinks))
	for _, pl := range t.ParentLinks {
		links = append(links, jsonx.Obj("id", pl.ID, "status", string(pl.Status), "display_name", nullable(pl.DisplayName),
			"accepted_at", isoTime(pl.AcceptedAt), "grants", grantsJSON(pl.Grants)))
	}
	return jsonx.Obj(
		"profile", ProfileJSON(t.Profile),
		"needs_onboarding", t.Profile == nil,
		"is_teen_mode", t.TeenMode,
		"allows", AllowsJSON(t.Allows),
		"readiness", itemJSON(t.Content.Readiness, l),
		"signs", itemsJSON(t.Content.Signs, l),
		"talk_note", itemJSON(t.Content.Talk, l),
		"kit", KitJSON(t.Kit, l),
		"faq", itemsJSON(t.Content.FAQ, l),
		"parent_preview", viewJSON(t.Preview),
		"parent_links", links,
	)
}

// CardJSON is one GET /teen/linked item: only the granted parts, everything else null; read only.
func CardJSON(c ParentCard) *jsonx.OrderedMap {
	out := jsonx.Obj(
		"link_id", c.Link.ID,
		"teen", jsonx.Obj("id", c.Link.OwnerID, "name", nullable(c.TeenName)),
		"accepted_at", isoTime(c.Link.AcceptedAt),
		"grants", grantsJSON(c.Link.Grants),
	)
	v := viewJSON(c.View)
	for _, k := range v.Keys() {
		val, _ := v.Get(k)
		out.Set(k, val)
	}
	return out.Set("read_only", true)
}
