package partners

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/partners/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Route registers one admin route (httpadmin.Handle with the prefix applied by the caller).
type Route func(method, path string, chain httpadmin.Chain)

// Admin serves /api/admin/v1/partners/* (docs/go-migration/admin-api.md §22): the service catalog CRUD and the request
// queue. The queue shows no name, address, note or account mobile in full: the personal blob is never opened here, the
// user's mobile is masked, and a result is never read.
type Admin struct {
	svc    *Service
	logger *slog.Logger
}

// NewAdmin wires the handlers.
func NewAdmin(svc *Service, logger *slog.Logger) *Admin {
	if logger == nil {
		logger = slog.Default()
	}
	return &Admin{svc: svc, logger: logger}
}

// Routes mounts the module.
func (h *Admin) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/partners/lab-services", kit.Admin(h.Services))
	route(fiber.MethodPost, "/partners/lab-services", kit.Admin(h.CreateService))
	route(fiber.MethodPut, "/partners/lab-services/:id", kit.Admin(h.UpdateService))
	route(fiber.MethodGet, "/partners/lab-requests", kit.Admin(h.Requests))
	route(fiber.MethodPost, "/partners/lab-requests/:id/status", kit.Admin(h.SetStatus))
}

func rawOrNil(s string) any {
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}

func adminServiceJSON(s store.PartnerLabService) *jsonx.OrderedMap {
	desc := any(nil)
	if s.Description.Valid {
		desc = rawOrNil(s.Description.String)
	}
	return jsonx.Obj("id", s.ID, "code", s.Code, "kind", s.Kind, "title", rawOrNil(s.Title), "description", desc,
		"partner_name", s.PartnerName, "partner_code", s.PartnerCode, "is_active", s.IsActive, "sort_order", s.SortOrder)
}

// Services is GET /partners/lab-services (all, active or not).
func (h *Admin) Services(c fiber.Ctx) error {
	rows, err := h.svc.q.ListServices(c.Context())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, s := range rows {
		items = append(items, adminServiceJSON(s))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items, "kinds", jsonx.List(Kinds), "partners", h.partnerCodes()))
}

func (h *Admin) partnerCodes() *jsonx.OrderedMap {
	codes := make([]string, 0, len(h.svc.adapters))
	for k := range h.svc.adapters {
		codes = append(codes, k)
	}
	return jsonx.Obj("codes", jsonx.List(sortedStrings(codes)))
}

func sortedStrings(in []string) []string {
	for i := 1; i < len(in); i++ {
		for j := i; j > 0 && in[j] < in[j-1]; j-- {
			in[j], in[j-1] = in[j-1], in[j]
		}
	}
	return in
}

func serviceRules() validation.Rules {
	return validation.Rules{
		validation.F("code", "required", "string", "max:40", "regex:/^[a-z0-9_]+$/"),
		validation.F("kind", "required", "in:"+strings.Join(Kinds, ",")),
		validation.F("title", "required", "array"),
		validation.F("title.fa", "required", "string", "max:120"),
		validation.F("title.en", "nullable", "string", "max:120"),
		validation.F("description", "nullable", "array"),
		validation.F("description.fa", "nullable", "string", "max:500"),
		validation.F("description.en", "nullable", "string", "max:500"),
		validation.F("partner_name", "required", "string", "max:120"),
		validation.F("partner_code", "nullable", "string", "max:40"),
		validation.F("is_active", "nullable", "boolean"),
		validation.F("sort_order", "nullable", "integer", "min:0", "max:1000"),
	}
}

func localizedJSON(m any) string {
	out := map[string]string{}
	if pm, ok := m.(interface {
		Keys() []string
		Get(string) (any, bool)
	}); ok {
		for _, k := range pm.Keys() {
			if v, ok := pm.Get(k); ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					out[k] = strings.TrimSpace(s)
				}
			}
		}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func (h *Admin) saveService(c fiber.Ctx, id uint64) error {
	data, err := httpadmin.Validate(c, serviceRules())
	if err != nil {
		return err
	}
	code := httpadmin.String(data, "code")
	partnerCode := httpadmin.String(data, "partner_code")
	if partnerCode == "" {
		partnerCode = DefaultPartner
	}
	if _, ok := h.svc.adapters[partnerCode]; !ok {
		return httpadmin.FieldError("partner_code", T("messages.service_unavailable", "en"))
	}
	taken, err := h.svc.q.ServiceCodeExists(c.Context(), store.ServiceCodeExistsParams{Code: code, ExceptID: id})
	if err != nil {
		return err
	}
	if taken {
		return httpadmin.FieldError("code", "The code has already been taken.")
	}
	title, _ := data.Get("title")
	desc := sql.NullString{}
	if d, ok := data.Get("description"); ok && d != nil {
		if j := localizedJSON(d); j != "{}" {
			desc = sql.NullString{String: j, Valid: true}
		}
	}
	now := httpadmin.DBTime(httpadmin.Now(c))
	order := int32(0)
	if v, ok := data.Get("sort_order"); ok && v != nil {
		order = int32(min(max(toInt(v), 0), 1000)) //nolint:gosec // G115: validated 0..1000
	}
	active := true
	if v, ok := data.Get("is_active"); ok && v != nil {
		active = httpadmin.Bool(data, "is_active")
	}
	if id == 0 {
		nid, err := h.svc.q.CreateService(c.Context(), store.CreateServiceParams{Code: code, Kind: httpadmin.String(data, "kind"),
			Title: localizedJSON(title), Description: desc, PartnerName: httpadmin.String(data, "partner_name"),
			PartnerCode: partnerCode, IsActive: active, SortOrder: order, CreatedAt: now, UpdatedAt: now})
		if err != nil {
			return err
		}
		id = uint64(nid) //nolint:gosec // G115
		httpadmin.Audit(c, h.logger, "partner_lab_service.create", "partner_lab_service", id)
		return h.respondService(c, id, true)
	}
	if err := h.svc.q.UpdateService(c.Context(), store.UpdateServiceParams{Code: code, Kind: httpadmin.String(data, "kind"),
		Title: localizedJSON(title), Description: desc, PartnerName: httpadmin.String(data, "partner_name"),
		PartnerCode: partnerCode, IsActive: active, SortOrder: order, UpdatedAt: now, ID: id}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "partner_lab_service.update", "partner_lab_service", id)
	return h.respondService(c, id, false)
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		x := 0
		for _, ch := range n {
			if ch < '0' || ch > '9' {
				return 0
			}
			x = x*10 + int(ch-'0')
		}
		return x
	}
	return 0
}

func (h *Admin) respondService(c fiber.Ctx, id uint64, created bool) error {
	s, err := h.svc.q.GetService(c.Context(), id)
	if err != nil {
		return err
	}
	msg := T("messages.service_saved", "en")
	if created {
		return httpadmin.Created(c, adminServiceJSON(s), msg)
	}
	return httpadmin.OK(c, adminServiceJSON(s), msg)
}

// CreateService is POST /partners/lab-services.
func (h *Admin) CreateService(c fiber.Ctx) error { return h.saveService(c, 0) }

// UpdateService is PUT /partners/lab-services/:id (full replace; is_active false hides the Today cards).
func (h *Admin) UpdateService(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Service")
	}
	if _, err := h.svc.q.GetService(c.Context(), id); errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Service")
	} else if err != nil {
		return err
	}
	return h.saveService(c, id)
}

func maskMobile(s sql.NullString) any {
	m := s.String
	if !s.Valid || len(m) < 8 {
		return nil
	}
	return m[:4] + "•••" + m[len(m)-4:]
}

// Requests is GET /partners/lab-requests?status=&q=<reference>&page= : the queue, newest first. Unknown status = all.
func (h *Admin) Requests(c fiber.Ctx) error {
	status := c.Query("status")
	if !contains(Statuses, status) {
		status = ""
	}
	ref := strings.ToUpper(strings.TrimSpace(c.Query("q")))
	total, err := h.svc.q.CountAdminRequests(c.Context(), store.CountAdminRequestsParams{Status: status, Reference: ref})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.svc.q.ListAdminRequests(c.Context(), store.ListAdminRequestsParams{Status: status, Reference: ref,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30))}) //nolint:gosec // G115: bounded by PageOf
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		manual := false
		if a, ok := h.svc.adapters[r.PartnerCode]; ok {
			if m, ok := a.(ManualStatus); ok {
				manual = m.ManualStatus()
			}
		}
		items = append(items, jsonx.Obj("id", r.ID, "reference", r.Reference, "user_id", r.UserID,
			"user_mobile", maskMobile(r.UserMobile), "status", r.Status, "service", PickText(r.ServiceTitle, "fa", "fa"),
			"service_code", r.ServiceCode, "partner_name", r.PartnerName, "preferred_date", r.PreferredDate.String(),
			"preferred_window", r.PreferredWindow, "consent_import", r.ConsentImport, "imported", r.ImportedLabID.Valid,
			"purged", r.PurgedAt.Valid, "manual_status", manual, "created_at", httpadmin.Time(r.CreatedAt)))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("status", status, "q", ref)).Set("statuses", jsonx.List(Statuses))
	return httpadmin.OK(c, page)
}

// SetStatus is POST /partners/lab-requests/:id/status {status}: the console plays the fake partner (409 for partners
// that update their own statuses, 409 for a move the machine does not allow).
func (h *Admin) SetStatus(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Request")
	}
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("status", "required", "in:"+strings.Join([]string{StatusAccepted, StatusSampled, StatusResultReady, StatusRejected}, ",")),
	})
	if err != nil {
		return err
	}
	to := httpadmin.String(data, "status")
	switch err := h.svc.Advance(c.Context(), id, to, httpadmin.Now(c)); {
	case errors.Is(err, ErrNotFound):
		return httpadmin.NotFound("Request")
	case errors.Is(err, ErrBadTransition):
		return httpadmin.Fail(fiber.StatusConflict, CodeBadTransition, T("messages.bad_transition", "en"))
	case errors.Is(err, ErrManualUnsupported), errors.Is(err, ErrPartnerUnavail):
		return httpadmin.Fail(fiber.StatusConflict, CodeManualUnsupported, T("messages.manual_unsupported", "en"))
	case err != nil:
		return err
	}
	httpadmin.Audit(c, h.logger, "partner_lab_request.status", "partner_lab_request", id, slog.String("status", to))
	return httpadmin.OK(c, jsonx.Obj("id", id, "status", to), T("messages.status_updated", "en"))
}
