package content

import (
	"context"
	"database/sql"
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	publiccontent "github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

func infoSectionJSON(s *store.InfoSection) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", s.ID,
		"group", s.Group,
		"key", httpadmin.NullString(s.Key),
		"heading", form.Raw(s.Heading),
		"body", form.Raw(s.Body),
		"link_label", form.NullRaw(s.LinkLabel),
		"link_url", httpadmin.NullString(s.LinkUrl),
		"is_active", s.IsActive,
		"sort_order", s.SortOrder,
		"created_at", httpadmin.Time(s.CreatedAt),
		"updated_at", httpadmin.Time(s.UpdatedAt),
	)
}

func (h *Handlers) infoSections() resource[store.InfoSection] {
	return resource[store.InfoSection]{
		name: "Info section", key: "info_section", get: h.q.GetInfoSection, json: infoSectionJSON,
		id: func(s *store.InfoSection) uint64 { return s.ID },
		toggle: func(ctx context.Context, now sql.NullTime, id uint64) error {
			return h.q.ToggleInfoSection(ctx, store.ToggleInfoSectionParams{Now: now, ID: id})
		},
		del: h.q.DeleteInfoSection,
	}
}

// infoGroup is the ?group= being edited; anything unrecognised is the first screen.
func infoGroup(c fiber.Ctx) string {
	if g := c.Query("group"); slices.Contains(publiccontent.InfoGroups, g) {
		return g
	}
	return publiccontent.InfoGroups[0]
}

// ListInfoSections is GET /info-sections?group=help|privacy|terms|about (sort_order, id).
func (h *Handlers) ListInfoSections(c fiber.Ctx) error {
	group := infoGroup(c)
	total, err := h.q.CountAdminInfoSections(c.Context(), group)
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminInfoSections(c.Context(), store.ListAdminInfoSectionsParams{
		Group: group, Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, infoSectionJSON(&rows[i]))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("group", group))
	return httpadmin.OK(c, page)
}

// InfoSectionOptions is GET /info-sections/options?group=: the groups and the sort_order
// a new box in the group should get (max + 10, leaving room to reorder).
func (h *Handlers) InfoSectionOptions(c fiber.Ctx) error {
	group := infoGroup(c)
	maxSort, err := h.q.MaxInfoSectionSortOrder(c.Context(), group)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("groups", publiccontent.InfoGroups, "group", group, "next_sort_order", maxSort+10))
}

// ShowInfoSection is GET /info-sections/:id.
func (h *Handlers) ShowInfoSection(c fiber.Ctx) error { return h.infoSections().show(c) }

func infoSectionRules(c fiber.Ctx) validation.Rules {
	rules := validation.Rules{validation.F("group", "required|string", validation.In(publiccontent.InfoGroups...))}
	rules = append(rules, form.Translatable(c, "heading", true, "max:200")...)
	rules = append(rules, form.Translatable(c, "body", true)...)
	rules = append(rules, form.Translatable(c, "link_label", false, "max:60")...)
	return append(rules,
		// mailto: and tel: matter as much as https here (support boxes).
		validation.F("link_url", "nullable|string|max:500", validation.Regex(`/^(https?:\/\/|mailto:|tel:)/i`)),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
	)
}

// StoreInfoSection is POST /info-sections.
func (h *Handlers) StoreInfoSection(c fiber.Ctx) error {
	data, err := form.Validate(c, infoSectionRules(c))
	if err != nil {
		return err
	}
	res, err := h.q.CreateInfoSection(c.Context(), store.CreateInfoSectionParams{
		SectionGroup: httpadmin.String(data, "group"), Heading: form.ReqJSON(data, "heading"),
		Body: form.ReqJSON(data, "body"), LinkLabel: form.Clean(data, "link_label"), LinkUrl: form.Str(data, "link_url"),
		IsActive: httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0), Now: h.now(c),
	})
	if err != nil {
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "info_section.create", "info_section", id)
	return h.infoSections().respond(c, id, true, "Info section created.")
}

// UpdateInfoSection is PUT /info-sections/:id (a cleared link becomes null).
func (h *Handlers) UpdateInfoSection(c fiber.Ctx) error {
	cur, err := find(c, "Info section", h.q.GetInfoSection)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, infoSectionRules(c))
	if err != nil {
		return err
	}
	if err := h.q.UpdateInfoSection(c.Context(), store.UpdateInfoSectionParams{
		SectionGroup: httpadmin.String(data, "group"), Heading: form.ReqJSON(data, "heading"),
		Body: form.ReqJSON(data, "body"), LinkLabel: form.Clean(data, "link_label"), LinkUrl: form.Str(data, "link_url"),
		IsActive: httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0),
		Now: h.now(c), ID: cur.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "info_section.update", "info_section", cur.ID)
	return h.infoSections().respond(c, cur.ID, false, "Info section updated.")
}

// DestroyInfoSection is DELETE /info-sections/:id.
func (h *Handlers) DestroyInfoSection(c fiber.Ctx) error {
	return h.infoSections().destroy(c, h.logger, "Info section deleted.")
}

// ToggleInfoSection is POST /info-sections/:id/toggle (is_active).
func (h *Handlers) ToggleInfoSection(c fiber.Ctx) error {
	return h.infoSections().doToggle(c, h.logger, "Status changed.")
}
