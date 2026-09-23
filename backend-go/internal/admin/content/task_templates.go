package content

import (
	"context"
	"database/sql"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func taskTemplateJSON(t *store.TaskTemplate) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", t.ID,
		"key", t.Key,
		"title", form.Raw(t.Title),
		"description", form.NullRaw(t.Description),
		"category", t.Category,
		"icon", httpadmin.NullString(t.Icon),
		"cycle_phase", httpadmin.NullString(t.CyclePhase),
		"is_active", t.IsActive,
		"sort_order", t.SortOrder,
		"created_at", httpadmin.Time(t.CreatedAt),
		"updated_at", httpadmin.Time(t.UpdatedAt),
	)
}

func (h *Handlers) taskTemplates() resource[store.TaskTemplate] {
	return resource[store.TaskTemplate]{
		name: "Task template", key: "task_template", get: h.q.GetTaskTemplate, json: taskTemplateJSON,
		id: func(t *store.TaskTemplate) uint64 { return t.ID },
		toggle: func(ctx context.Context, now sql.NullTime, id uint64) error {
			return h.q.ToggleTaskTemplate(ctx, store.ToggleTaskTemplateParams{Now: now, ID: id})
		},
		del: h.q.DeleteTaskTemplate,
	}
}

// ListTaskTemplates is GET /task-templates (sort_order, newest first).
func (h *Handlers) ListTaskTemplates(c fiber.Ctx) error {
	total, err := h.q.CountAdminTaskTemplates(c.Context())
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminTaskTemplates(c.Context(), store.ListAdminTaskTemplatesParams{
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, taskTemplateJSON(&rows[i]))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// TaskTemplateOptions is GET /task-templates/options.
func (h *Handlers) TaskTemplateOptions(c fiber.Ctx) error {
	loc := locale(c)
	return httpadmin.OK(c, jsonx.Obj(
		"phases", options(enums.CyclePhaseOptions(loc)),
		"categories", options(enums.TaskCategoryOptions(loc)),
	))
}

// ShowTaskTemplate is GET /task-templates/:id.
func (h *Handlers) ShowTaskTemplate(c fiber.Ctx) error { return h.taskTemplates().show(c) }

func taskTemplateRules(c fiber.Ctx) validation.Rules {
	rules := validation.Rules{validation.F("key", "required|string|max:255")}
	rules = append(rules, form.Translatable(c, "title", true)...)
	rules = append(rules, form.Translatable(c, "description", false)...)
	return append(rules,
		validation.F("category", "required", validation.In(enums.TaskCategoryValues()...)),
		validation.F("icon", "nullable|string|max:255"),
		validation.F("cycle_phase", "nullable", validation.In(enums.CyclePhaseValues()...)),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
	)
}

// uniqueKey is unique:task_templates,key.
func (h *Handlers) uniqueKey(c fiber.Ctx, exceptID uint64) form.Check {
	return func(in phpval.Map, add form.Add) error {
		key, ok := in.Get("key")
		s, isStr := key.(string)
		if !ok || !isStr {
			return nil
		}
		taken, err := h.q.TaskTemplateKeyTaken(c.Context(), store.TaskTemplateKeyTakenParams{TemplateKey: s, ExceptID: exceptID})
		if err != nil {
			return err
		}
		if taken {
			add("key", form.Msg(c, "validation.unique", "key"))
		}
		return nil
	}
}

// StoreTaskTemplate is POST /task-templates.
func (h *Handlers) StoreTaskTemplate(c fiber.Ctx) error {
	data, err := form.Validate(c, taskTemplateRules(c), h.uniqueKey(c, 0))
	if err != nil {
		return err
	}
	res, err := h.q.CreateTaskTemplate(c.Context(), store.CreateTaskTemplateParams{
		TemplateKey: httpadmin.String(data, "key"), Title: form.ReqJSON(data, "title"),
		Description: form.NullJSON(data, "description"), Category: httpadmin.String(data, "category"),
		Icon: form.Str(data, "icon"), CyclePhase: form.Str(data, "cycle_phase"),
		IsActive: httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0), Now: h.now(c),
	})
	if err != nil {
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "task_template.create", "task_template", id)
	return h.taskTemplates().respond(c, id, true, "Task template created.")
}

// UpdateTaskTemplate is PUT /task-templates/:id.
func (h *Handlers) UpdateTaskTemplate(c fiber.Ctx) error {
	cur, err := find(c, "Task template", h.q.GetTaskTemplate)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, taskTemplateRules(c), h.uniqueKey(c, cur.ID))
	if err != nil {
		return err
	}
	if err := h.q.UpdateTaskTemplate(c.Context(), store.UpdateTaskTemplateParams{
		TemplateKey: httpadmin.String(data, "key"), Title: form.ReqJSON(data, "title"),
		Description: form.KeepJSON(data, "description", cur.Description), Category: httpadmin.String(data, "category"),
		Icon: form.KeepStr(data, "icon", cur.Icon), CyclePhase: form.KeepStr(data, "cycle_phase", cur.CyclePhase),
		IsActive: httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0),
		Now: h.now(c), ID: cur.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "task_template.update", "task_template", cur.ID)
	return h.taskTemplates().respond(c, cur.ID, false, "Task template updated.")
}

// DestroyTaskTemplate is DELETE /task-templates/:id.
func (h *Handlers) DestroyTaskTemplate(c fiber.Ctx) error {
	return h.taskTemplates().destroy(c, h.logger, "Task template deleted.")
}

// ToggleTaskTemplate is POST /task-templates/:id/toggle (is_active).
func (h *Handlers) ToggleTaskTemplate(c fiber.Ctx) error {
	return h.taskTemplates().doToggle(c, h.logger, "Status changed.")
}
