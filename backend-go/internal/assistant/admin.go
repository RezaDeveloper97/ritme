package assistant

import (
	"database/sql"
	"errors"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/assistant/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Department editor limits (admin-api §19c).
const (
	MaxPromptLen     = 4000
	MaxGuardrailsLen = 2000
	MaxListItems     = 10
	MaxListItemLen   = 120
	MaxTextLen       = 600
)

// DepartmentsAdmin serves /api/admin/v1/assistant/departments* — super admins only (audit L1). The emergency floor
// (keyword → 115 card, diagnosis guard, base system guard) lives in code and is not reachable from here. Any change to
// the system prompt or guardrails sets needs_review again; a department can only be activated after `approve`.
// Audit lines carry ids and changed field names, never the prompt text.
type DepartmentsAdmin struct {
	q      *store.Queries
	logger *slog.Logger
}

// NewDepartmentsAdmin wires the handlers.
func NewDepartmentsAdmin(conn *sql.DB, logger *slog.Logger) *DepartmentsAdmin {
	if logger == nil {
		logger = slog.Default()
	}
	return &DepartmentsAdmin{q: store.New(conn), logger: logger}
}

// Route registers one endpoint.
type Route func(method, path string, chain httpadmin.Chain)

// Routes mounts the module; every route is super only.
func (h *DepartmentsAdmin) Routes(route Route, kit *httpadmin.Kit) {
	s := kit.Super
	p := "/assistant/departments"
	route(fiber.MethodGet, p, s(h.List))
	route(fiber.MethodPost, p, s(h.Store))
	route(fiber.MethodGet, p+"/:id", s(h.Show))
	route(fiber.MethodPut, p+"/:id", s(h.Update))
	route(fiber.MethodPost, p+"/:id/approve", s(h.Approve))
}

func departmentAdminJSON(d store.AssistantDepartment) *jsonx.OrderedMap {
	raw := func(s sql.NullString) any {
		if !s.Valid || s.String == "" {
			return nil
		}
		return form.Raw([]byte(s.String))
	}
	return jsonx.Obj(
		"id", d.ID, "code", d.Code, "icon", nullText(d.Icon), "sort_order", d.SortOrder, "is_active", d.IsActive,
		"name", form.Raw([]byte(d.Name)), "tagline", raw(d.Tagline), "intro", raw(d.Intro),
		"does_not_do", raw(d.DoesNotDo), "quick_replies", raw(d.QuickReplies),
		"system_prompt", d.SystemPrompt, "guardrails", nullText(d.Guardrails), "needs_review", d.NeedsReview,
		"updated_at", dt(d.UpdatedAt),
	)
}

func (h *DepartmentsAdmin) find(c fiber.Ctx) (store.AssistantDepartment, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.AssistantDepartment{}, httpadmin.NotFound("Department")
	}
	d, err := h.q.GetAdminDepartment(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return d, httpadmin.NotFound("Department")
	}
	return d, err
}

// List is GET /assistant/departments — {items}.
func (h *DepartmentsAdmin) List(c fiber.Ctx) error {
	rows, err := h.q.ListAdminDepartments(c.Context())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, departmentAdminJSON(r))
	}
	return httpadmin.OK(c, jsonx.Obj("items", jsonx.List(items), "limits", jsonx.Obj(
		"system_prompt", MaxPromptLen, "guardrails", MaxGuardrailsLen, "list_items", MaxListItems,
		"list_item", MaxListItemLen, "text", MaxTextLen)))
}

// Show is GET /assistant/departments/:id — {department}.
func (h *DepartmentsAdmin) Show(c fiber.Ctx) error {
	d, err := h.find(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("department", departmentAdminJSON(d)))
}

func (h *DepartmentsAdmin) rules(c fiber.Ctx, create bool) validation.Rules {
	r := validation.Rules{}
	if create {
		r = append(r, validation.F("code", "required|string|max:32|regex:/^[a-z][a-z0-9_]*$/"))
	}
	r = append(r, form.Translatable(c, "name", true, "max:120")...)
	r = append(r, form.Translatable(c, "tagline", false, "max:"+strconv.Itoa(MaxTextLen))...)
	r = append(r, form.Translatable(c, "intro", false, "max:"+strconv.Itoa(MaxTextLen))...)
	return append(r,
		validation.F("icon", "nullable|string|max:32"),
		validation.F("sort_order", "nullable|integer|min:0"),
		validation.F("is_active", "nullable|boolean"),
		validation.F("system_prompt", "required|string|max:"+strconv.Itoa(MaxPromptLen)),
		validation.F("guardrails", "nullable|string|max:"+strconv.Itoa(MaxGuardrailsLen)),
		validation.F("does_not_do", "nullable|array"),
		validation.F("does_not_do.*", "array|max:"+strconv.Itoa(MaxListItems)),
		validation.F("does_not_do.*.*", "required|string|max:"+strconv.Itoa(MaxListItemLen)),
		validation.F("quick_replies", "nullable|array"),
		validation.F("quick_replies.*", "array|max:"+strconv.Itoa(MaxListItems)),
		validation.F("quick_replies.*.*", "required|string|max:"+strconv.Itoa(MaxListItemLen)),
	)
}

func nullJSONStr(data phpval.Map, key string) sql.NullString {
	col := form.NullJSON(data, key)
	if !col.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: string(col.V), Valid: true}
}

func cleanStr(data phpval.Map, key string) sql.NullString {
	col := form.Clean(data, key)
	if !col.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: string(col.V), Valid: true}
}

// Store is POST /assistant/departments (201 {department}): always created inactive and needs_review.
func (h *DepartmentsAdmin) Store(c fiber.Ctx) error {
	data, err := form.Validate(c, h.rules(c, true), func(in phpval.Map, add form.Add) error {
		code := httpadmin.String(in, "code")
		if code == "" {
			return nil
		}
		n, err := h.q.CountDepartmentCode(c.Context(), code)
		if err != nil {
			return err
		}
		if n > 0 {
			add("code", form.Msg(c, "validation.unique", "code"))
		}
		return nil
	})
	if err != nil {
		return err
	}
	id, err := h.q.InsertDepartment(c.Context(), store.InsertDepartmentParams{
		Code: httpadmin.String(data, "code"), Icon: form.Str(data, "icon"), SortOrder: form.Int32(data, "sort_order", 0),
		Name: string(form.ReqJSON(data, "name")), Tagline: cleanStr(data, "tagline"), Intro: cleanStr(data, "intro"),
		DoesNotDo: nullJSONStr(data, "does_not_do"), QuickReplies: nullJSONStr(data, "quick_replies"),
		SystemPrompt: httpadmin.String(data, "system_prompt"), Guardrails: form.Str(data, "guardrails"),
		Now: httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	uid := uint64(id) //nolint:gosec // G115: auto-increment ids are positive
	httpadmin.Audit(c, h.logger, "assistant_department.create", "assistant_department", uid)
	d, err := h.q.GetAdminDepartment(c.Context(), uid)
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("department", departmentAdminJSON(d)), "Department created.")
}

// Update is PUT /assistant/departments/:id: the full form. Changing system_prompt or guardrails sets needs_review
// (the department keeps its current is_active); switching an unreviewed department on is a 422 until `approve`.
func (h *DepartmentsAdmin) Update(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, h.rules(c, false))
	if err != nil {
		return err
	}
	active := cur.IsActive
	if form.Has(data, "is_active") {
		active = httpadmin.Bool(data, "is_active")
	}
	prompt, guard := httpadmin.String(data, "system_prompt"), form.Str(data, "guardrails")
	promptChanged := prompt != cur.SystemPrompt
	guardChanged := guard.String != cur.Guardrails.String
	review := cur.NeedsReview || promptChanged || guardChanged
	if active && !cur.IsActive && review {
		return httpadmin.FieldError("is_active", "The department needs a medical review (approve it) before it can be activated.")
	}
	icon := cur.Icon
	if form.Has(data, "icon") {
		icon = form.Str(data, "icon")
	}
	if err := h.q.UpdateDepartment(c.Context(), store.UpdateDepartmentParams{
		Icon: icon, SortOrder: form.Int32(data, "sort_order", int64(cur.SortOrder)), IsActive: active,
		Name: string(form.ReqJSON(data, "name")), Tagline: cleanStr(data, "tagline"), Intro: cleanStr(data, "intro"),
		DoesNotDo: nullJSONStr(data, "does_not_do"), QuickReplies: nullJSONStr(data, "quick_replies"),
		SystemPrompt: prompt, Guardrails: guard, NeedsReview: review, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "assistant_department.update", "assistant_department", cur.ID,
		slog.Bool("prompt_changed", promptChanged), slog.Bool("guardrails_changed", guardChanged),
		slog.Bool("is_active", active), slog.Bool("needs_review", review))
	d, err := h.q.GetAdminDepartment(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("department", departmentAdminJSON(d)), "Department updated.")
}

// Approve is POST /assistant/departments/:id/approve: the medical-review sign-off (needs_review → false).
func (h *DepartmentsAdmin) Approve(c fiber.Ctx) error {
	d, err := h.find(c)
	if err != nil {
		return err
	}
	if _, err := h.q.ApproveDepartment(c.Context(), store.ApproveDepartmentParams{Now: httpadmin.DBTime(httpadmin.Now(c)), ID: d.ID}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "assistant_department.approve", "assistant_department", d.ID)
	d, err = h.q.GetAdminDepartment(c.Context(), d.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("department", departmentAdminJSON(d)), "Department approved.")
}
