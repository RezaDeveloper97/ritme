package insurance

import (
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/insurance/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Route registers one admin route (httpadmin.Handle with the prefix applied by the caller).
type Route func(method, path string, chain httpadmin.Chain)

// Admin serves /api/admin/v1/insurance/* (content capability): the insurer catalog and the contracted centres. No
// user policy, claim or questionnaire is ever reachable from the admin side.
type Admin struct {
	q      *store.Queries
	logger *slog.Logger
}

// NewAdmin wires the handlers.
func NewAdmin(svc *Service, logger *slog.Logger) *Admin {
	if logger == nil {
		logger = slog.Default()
	}
	return &Admin{q: svc.q, logger: logger}
}

// Routes mounts the module.
func (a *Admin) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/insurance/insurers", kit.Admin(a.Insurers))
	route(fiber.MethodPost, "/insurance/insurers", kit.Admin(a.StoreInsurer))
	route(fiber.MethodPut, "/insurance/insurers/:id", kit.Admin(a.UpdateInsurer))
	route(fiber.MethodDelete, "/insurance/insurers/:id", kit.Admin(a.DestroyInsurer))
	route(fiber.MethodGet, "/insurance/centres", kit.Admin(a.Centres))
	route(fiber.MethodPost, "/insurance/centres", kit.Admin(a.StoreCentre))
	route(fiber.MethodPut, "/insurance/centres/:id", kit.Admin(a.UpdateCentre))
	route(fiber.MethodDelete, "/insurance/centres/:id", kit.Admin(a.DestroyCentre))
}

func adminNullable(data phpval.Map, key string) sql.NullString {
	s := strings.TrimSpace(httpadmin.String(data, key))
	return sql.NullString{String: s, Valid: s != ""}
}

func isHTTPURL(s string) bool {
	l := strings.ToLower(s)
	return (strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "http://")) && form.IsURL(s)
}

// ---------------------------------------------------------------------------------------------------------------
// Insurers

func insurerJSON(i store.InsuranceInsurer) *jsonx.OrderedMap {
	return jsonx.Obj("id", i.ID, "code", i.Code, "name", i.Name, "website", strOrNil(i.Website), "phone", strOrNil(i.Phone),
		"is_active", i.IsActive, "sort_order", i.SortOrder, "updated_at", httpadmin.Time(i.UpdatedAt))
}

// Insurers is GET /insurance/insurers (all, active or not).
func (a *Admin) Insurers(c fiber.Ctx) error {
	rows, err := a.q.ListInsurers(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, i := range rows {
		items = append(items, insurerJSON(i))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

func (a *Admin) saveInsurer(c fiber.Ctx, id uint64) error {
	data, err := form.Validate(c, validation.Rules{
		validation.F("code", "required|string|max:40|regex:/^[a-z0-9_]+$/"),
		validation.F("name", "required|string|max:120"),
		validation.F("website", "nullable|string|max:255"),
		validation.F("phone", "nullable|string|max:32"),
		validation.F("is_active", "nullable|boolean"),
		validation.F("sort_order", "nullable|integer|min:0|max:1000"),
	}, func(in phpval.Map, add form.Add) error {
		if w := strings.TrimSpace(httpadmin.String(in, "website")); w != "" && !isHTTPURL(w) {
			add("website", form.Msg(c, "validation.url", "website"))
		}
		code := strings.TrimSpace(httpadmin.String(in, "code"))
		if code == "" {
			return nil
		}
		taken, err := a.q.InsurerCodeExists(c, store.InsurerCodeExistsParams{Code: code, ExceptID: id})
		if err != nil {
			return err
		}
		if taken {
			add("code", form.Msg(c, "validation.unique", "code"))
		}
		return nil
	})
	if err != nil {
		return err
	}
	now := httpadmin.DBTime(httpadmin.Now(c))
	active := !form.Has(data, "is_active") || httpadmin.Bool(data, "is_active")
	code, name := strings.TrimSpace(httpadmin.String(data, "code")), strings.TrimSpace(httpadmin.String(data, "name"))
	order := form.Int32(data, "sort_order", 0)
	created := id == 0
	if created {
		nid, err := a.q.CreateInsurer(c, store.CreateInsurerParams{Code: code, Name: name, Website: adminNullable(data, "website"),
			Phone: adminNullable(data, "phone"), IsActive: active, SortOrder: order, CreatedAt: now, UpdatedAt: now})
		if err != nil {
			return err
		}
		id = uint64(nid) //nolint:gosec // G115: DB id
	} else if err := a.q.UpdateInsurer(c, store.UpdateInsurerParams{Code: code, Name: name, Website: adminNullable(data, "website"),
		Phone: adminNullable(data, "phone"), IsActive: active, SortOrder: order, UpdatedAt: now, ID: id}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "insurance_insurer.save", "insurance_insurer", id)
	i, err := a.q.GetInsurer(c, id)
	if err != nil {
		return err
	}
	if created {
		return httpadmin.Created(c, jsonx.Obj("item", insurerJSON(i)), "Created.")
	}
	return httpadmin.OK(c, jsonx.Obj("item", insurerJSON(i)), "Saved.")
}

// StoreInsurer is POST /insurance/insurers.
func (a *Admin) StoreInsurer(c fiber.Ctx) error { return a.saveInsurer(c, 0) }

// UpdateInsurer is PUT /insurance/insurers/:id (full replace).
func (a *Admin) UpdateInsurer(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Insurer")
	}
	if _, err := a.q.GetInsurer(c, id); errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Insurer")
	} else if err != nil {
		return err
	}
	return a.saveInsurer(c, id)
}

// DestroyInsurer is DELETE /insurance/insurers/:id (policies and centres keep their rows; the link becomes empty, the
// user's typed name is not restored — deactivate instead to keep it visible).
func (a *Admin) DestroyInsurer(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Insurer")
	}
	n, err := a.q.DeleteInsurer(c, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpadmin.NotFound("Insurer")
	}
	httpadmin.Audit(c, a.logger, "insurance_insurer.delete", "insurance_insurer", id)
	return httpadmin.OK(c, jsonx.Obj("id", id), "Deleted.")
}

// ---------------------------------------------------------------------------------------------------------------
// Centres

func centreJSON(r store.GetCentreRow) *jsonx.OrderedMap {
	var insurer any
	if r.InsurerID.Valid {
		insurer = r.InsurerID.Int64
	}
	return jsonx.Obj("id", r.ID, "insurer_id", insurer, "insurer_name", strOrNil(r.InsurerName), "name", r.Name, "type", r.Type,
		"services", strOrNil(r.Services), "city", strOrNil(r.City), "address", strOrNil(r.Address), "phone", strOrNil(r.Phone),
		"lat", strOrNil(r.Lat), "lng", strOrNil(r.Lng), "is_direct", r.IsDirect, "is_active", r.IsActive,
		"sort_order", r.SortOrder, "updated_at", httpadmin.Time(r.UpdatedAt))
}

// Centres is GET /insurance/centres (all).
func (a *Admin) Centres(c fiber.Ctx) error {
	rows, err := a.q.ListCentres(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, centreJSON(store.GetCentreRow(r)))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items, "types", jsonx.List(CentreTypes)))
}

func coord(data phpval.Map, key string) sql.NullString {
	if !form.Has(data, key) {
		return sql.NullString{}
	}
	v, _ := data.Get(key)
	return sql.NullString{String: strconv.FormatFloat(phpval.ToFloat(v), 'f', 7, 64), Valid: true}
}

func (a *Admin) saveCentre(c fiber.Ctx, id uint64) error {
	data, err := form.Validate(c, validation.Rules{
		validation.F("insurer_id", "nullable|integer|min:1"),
		validation.F("name", "required|string|max:160"),
		validation.F("type", "required", validation.In(CentreTypes...)),
		validation.F("services", "nullable|string|max:160"),
		validation.F("city", "nullable|string|max:80"),
		validation.F("address", "nullable|string|max:300"),
		validation.F("phone", "nullable|string|max:32"),
		validation.F("lat", "nullable|numeric|between:-90,90"),
		validation.F("lng", "nullable|numeric|between:-180,180"),
		validation.F("is_direct", "nullable|boolean"),
		validation.F("is_active", "nullable|boolean"),
		validation.F("sort_order", "nullable|integer|min:0|max:1000"),
	}, func(in phpval.Map, add form.Add) error {
		if form.Has(in, "lat") != form.Has(in, "lng") {
			add("lng", form.Msg(c, "validation.required", "lng"))
		}
		if !form.Has(in, "insurer_id") {
			return nil
		}
		v, _ := in.Get("insurer_id")
		n, err := strconv.ParseUint(phpval.ToString(v), 10, 64)
		if err != nil {
			return nil //nolint:nilerr // the integer rule reports it
		}
		if _, err := a.q.GetInsurer(c, n); errors.Is(err, sql.ErrNoRows) {
			add("insurer_id", form.Msg(c, "validation.exists", "insurer_id"))
		} else if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	now := httpadmin.DBTime(httpadmin.Now(c))
	var insurer sql.NullInt64
	if form.Has(data, "insurer_id") {
		n, _ := strconv.ParseInt(httpadmin.String(data, "insurer_id"), 10, 64)
		insurer = sql.NullInt64{Int64: n, Valid: true}
	}
	p := store.CreateCentreParams{InsurerID: insurer, Name: strings.TrimSpace(httpadmin.String(data, "name")),
		Type: httpadmin.String(data, "type"), Services: adminNullable(data, "services"), City: adminNullable(data, "city"),
		Address: adminNullable(data, "address"), Phone: adminNullable(data, "phone"), Lat: coord(data, "lat"), Lng: coord(data, "lng"),
		IsDirect: httpadmin.Bool(data, "is_direct"), IsActive: !form.Has(data, "is_active") || httpadmin.Bool(data, "is_active"),
		SortOrder: form.Int32(data, "sort_order", 0), CreatedAt: now, UpdatedAt: now}
	created := id == 0
	if created {
		nid, err := a.q.CreateCentre(c, p)
		if err != nil {
			return err
		}
		id = uint64(nid) //nolint:gosec // G115: DB id
	} else if err := a.q.UpdateCentre(c, store.UpdateCentreParams{InsurerID: p.InsurerID, Name: p.Name, Type: p.Type,
		Services: p.Services, City: p.City, Address: p.Address, Phone: p.Phone, Lat: p.Lat, Lng: p.Lng, IsDirect: p.IsDirect,
		IsActive: p.IsActive, SortOrder: p.SortOrder, UpdatedAt: now, ID: id}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "insurance_centre.save", "insurance_centre", id)
	r, err := a.q.GetCentre(c, id)
	if err != nil {
		return err
	}
	if created {
		return httpadmin.Created(c, jsonx.Obj("item", centreJSON(r)), "Created.")
	}
	return httpadmin.OK(c, jsonx.Obj("item", centreJSON(r)), "Saved.")
}

// StoreCentre is POST /insurance/centres.
func (a *Admin) StoreCentre(c fiber.Ctx) error { return a.saveCentre(c, 0) }

// UpdateCentre is PUT /insurance/centres/:id (full replace).
func (a *Admin) UpdateCentre(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Centre")
	}
	if _, err := a.q.GetCentre(c, id); errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Centre")
	} else if err != nil {
		return err
	}
	return a.saveCentre(c, id)
}

// DestroyCentre is DELETE /insurance/centres/:id (claims keep their centre name).
func (a *Admin) DestroyCentre(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Centre")
	}
	n, err := a.q.DeleteCentre(c, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpadmin.NotFound("Centre")
	}
	httpadmin.Audit(c, a.logger, "insurance_centre.delete", "insurance_centre", id)
	return httpadmin.OK(c, jsonx.Obj("id", id), "Deleted.")
}
