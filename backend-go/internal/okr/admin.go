package okr

import (
	"database/sql"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Route registers one endpoint.
type Route func(method, path string, chain httpadmin.Chain)

// Admin serves /api/admin/v1/okr* (capability «dashboard», see httpadmin.capabilityOf: view reads, manage writes).
type Admin struct {
	svc    *Service
	logger *slog.Logger
}

// NewAdmin builds the handlers.
func NewAdmin(svc *Service, logger *slog.Logger) *Admin {
	if logger == nil {
		logger = slog.Default()
	}
	return &Admin{svc: svc, logger: logger}
}

// Routes registers the endpoints.
func (a *Admin) Routes(route Route, kit *httpadmin.Kit) {
	k := kit.Admin
	route(fiber.MethodGet, "/okr", k(a.List))
	route(fiber.MethodPost, "/okr/objectives", k(a.StoreObjective))
	route(fiber.MethodPut, "/okr/objectives/:id", k(a.UpdateObjective))
	route(fiber.MethodDelete, "/okr/objectives/:id", k(a.DestroyObjective))
	route(fiber.MethodPost, "/okr/objectives/:id/key-results", k(a.StoreKR))
	route(fiber.MethodPut, "/okr/key-results/:id", k(a.UpdateKR))
	route(fiber.MethodDelete, "/okr/key-results/:id", k(a.DestroyKR))
}

func r2(f float64) float64 { return math.Round(f*100) / 100 }

func pctOf(p *float64) any {
	if p == nil {
		return nil
	}
	return math.Round(*p*1000) / 10 // 0..100, one decimal
}

func val(p *float64) any {
	if p == nil {
		return nil
	}
	return r2(*p)
}

func (a *Admin) krJSON(e Evaluated) *jsonx.OrderedMap {
	var key any
	source := "manual"
	if e.MetricKey.Valid {
		key, source = e.MetricKey.String, "live"
	}
	return jsonx.Obj("id", e.ID, "objective_id", e.ObjectiveID, "title", e.Title, "metric_key", key, "source", source,
		"unit", e.Unit, "baseline", r2(e.Baseline), "target", r2(e.Target), "manual_value", nullFloat(e.ManualValue),
		"current", val(e.Current), "progress_pct", pctOf(e.Progress), "status", e.Status, "owner", httpadmin.NullString(e.Owner))
}

func nullFloat(f sql.NullFloat64) any {
	if !f.Valid {
		return nil
	}
	return r2(f.Float64)
}

func (a *Admin) objectiveJSON(o EvaluatedObjective) *jsonx.OrderedMap {
	krs := make([]*jsonx.OrderedMap, 0, len(o.KRs))
	for _, e := range o.KRs {
		krs = append(krs, a.krJSON(e))
	}
	return jsonx.Obj("id", o.ID, "title", o.Title, "owner", httpadmin.NullString(o.Owner), "period", o.Period,
		"starts_on", o.StartsOn.Format("2006-01-02"), "ends_on", o.EndsOn.Format("2006-01-02"), "sort_order", o.SortOrder,
		"progress_pct", pctOf(o.Progress), "status", o.Status, "key_results", krs)
}

func (a *Admin) today(c fiber.Ctx) time.Time { return httpadmin.Now(c).In(civildate.Tehran) }

// List is GET /okr?period=: every objective (or one period) with live progress, the metric catalogue with the current
// values (for the binding select) and the summary of the objectives active today.
func (a *Admin) List(c fiber.Ctx) error {
	now := a.today(c)
	period := strings.TrimSpace(c.Query("period"))
	all, err := a.svc.List(c.Context(), "")
	if err != nil {
		return err
	}
	evAll, err := a.svc.Evaluate(c.Context(), now, all)
	if err != nil {
		return err
	}
	periods, err := a.svc.Periods(c.Context())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(evAll))
	for _, o := range evAll {
		if period == "" || o.Period == period {
			items = append(items, a.objectiveJSON(o))
		}
	}
	live, err := a.svc.LiveValues(c.Context())
	if err != nil {
		return err
	}
	cat := make([]*jsonx.OrderedMap, 0, len(Metrics))
	for _, m := range Metrics {
		var cur any
		if v, ok := live[m.Key]; ok {
			cur = r2(v)
		}
		cat = append(cat, jsonx.Obj("key", m.Key, "unit", m.Unit, "current", cur))
	}
	return httpadmin.OK(c, jsonx.Obj("objectives", items, "periods", periods, "summary", Summarize(evAll, now),
		"metrics", cat, "units", Units,
		"notes", jsonx.Obj("live", "Live key results read the newest hourly KPI snapshot (retention: the newest weekly cohort with a value). A value is empty while it is suppressed (fewer than 10 users) or not computed yet.")))
}

func (a *Admin) objectiveInput(c fiber.Ctx) (ObjectiveInput, error) {
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("title", "required|string|max:255"),
		validation.F("owner", "nullable|string|max:120"),
		validation.F("period", "required|string|max:40"),
		validation.F("starts_on", "required|date_format:Y-m-d"),
		validation.F("ends_on", "required|date_format:Y-m-d|after_or_equal:starts_on"),
		validation.F("sort_order", "nullable|integer"),
	})
	if err != nil {
		return ObjectiveInput{}, err
	}
	return ObjectiveInput{Title: strings.TrimSpace(httpadmin.String(data, "title")), Owner: nullStr(data, "owner"),
		Period: strings.TrimSpace(httpadmin.String(data, "period")), StartsOn: ParseDay(httpadmin.String(data, "starts_on")),
		EndsOn: ParseDay(httpadmin.String(data, "ends_on")), SortOrder: intOf(data, "sort_order")}, nil
}

func nullStr(data phpval.Map, key string) sql.NullString {
	s := strings.TrimSpace(httpadmin.String(data, key))
	return sql.NullString{String: s, Valid: s != ""}
}

func intOf(data phpval.Map, key string) int {
	n, _ := strconv.Atoi(httpadmin.String(data, key))
	return n
}

func (a *Admin) respondObjective(c fiber.Ctx, id uint64, created bool) error {
	o, err := a.svc.Objective(c.Context(), id)
	if err != nil {
		return err
	}
	objs, err := a.svc.List(c.Context(), o.Period)
	if err != nil {
		return err
	}
	evs, err := a.svc.Evaluate(c.Context(), a.today(c), objs)
	if err != nil {
		return err
	}
	for _, e := range evs {
		if e.ID == id {
			out := jsonx.Obj("objective", a.objectiveJSON(e))
			if created {
				return httpadmin.Created(c, out)
			}
			return httpadmin.OK(c, out)
		}
	}
	return httpadmin.NotFound("Objective")
}

// StoreObjective is POST /okr/objectives.
func (a *Admin) StoreObjective(c fiber.Ctx) error {
	in, err := a.objectiveInput(c)
	if err != nil {
		return err
	}
	id, err := a.svc.CreateObjective(c.Context(), in, a.today(c))
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "okr.objective.create", "okr_objective", id)
	return a.respondObjective(c, id, true)
}

// UpdateObjective is PUT /okr/objectives/:id.
func (a *Admin) UpdateObjective(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Objective")
	}
	in, err := a.objectiveInput(c)
	if err != nil {
		return err
	}
	found, err := a.svc.UpdateObjective(c.Context(), id, in, a.today(c))
	if err != nil {
		return err
	}
	if !found {
		return httpadmin.NotFound("Objective")
	}
	httpadmin.Audit(c, a.logger, "okr.objective.update", "okr_objective", id)
	return a.respondObjective(c, id, false)
}

// DestroyObjective is DELETE /okr/objectives/:id (key results go with it).
func (a *Admin) DestroyObjective(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Objective")
	}
	found, err := a.svc.DeleteObjective(c.Context(), id)
	if err != nil {
		return err
	}
	if !found {
		return httpadmin.NotFound("Objective")
	}
	httpadmin.Audit(c, a.logger, "okr.objective.delete", "okr_objective", id)
	return httpadmin.OK(c, nil)
}

func (a *Admin) krInput(c fiber.Ctx) (KRInput, error) {
	keys := make([]string, 0, len(Metrics))
	for _, m := range Metrics {
		keys = append(keys, m.Key)
	}
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("title", "required|string|max:255"),
		validation.F("metric_key", "nullable|string|in:"+strings.Join(keys, ",")),
		validation.F("unit", "nullable|string|in:"+strings.Join(Units, ",")),
		validation.F("baseline", "required|numeric"),
		validation.F("target", "required|numeric"),
		validation.F("manual_value", "nullable|numeric"),
		validation.F("owner", "nullable|string|max:120"),
		validation.F("sort_order", "nullable|integer"),
	})
	if err != nil {
		return KRInput{}, err
	}
	in := KRInput{Title: strings.TrimSpace(httpadmin.String(data, "title")), MetricKey: nullStr(data, "metric_key"),
		Unit: httpadmin.String(data, "unit"), Owner: nullStr(data, "owner"), SortOrder: intOf(data, "sort_order")}
	in.Baseline, _ = strconv.ParseFloat(httpadmin.String(data, "baseline"), 64)
	in.Target, _ = strconv.ParseFloat(httpadmin.String(data, "target"), 64)
	if in.Baseline == in.Target {
		ve := httpx.NewValidationError()
		ve.Add("target", "The target must be different from the baseline.")
		return KRInput{}, httpadmin.Invalid(ve)
	}
	if s := strings.TrimSpace(httpadmin.String(data, "manual_value")); s != "" {
		f, _ := strconv.ParseFloat(s, 64)
		in.ManualValue = sql.NullFloat64{Float64: f, Valid: true}
	}
	if in.MetricKey.Valid { // bound: the unit follows the metric, a typed value is ignored
		in.Unit, _ = MetricUnit(in.MetricKey.String)
		in.ManualValue = sql.NullFloat64{}
	} else if in.Unit == "" {
		in.Unit = "count"
	}
	return in, nil
}

func (a *Admin) respondKR(c fiber.Ctx, krID uint64, created bool) error {
	kr, err := a.svc.KeyResult(c.Context(), krID)
	if err != nil {
		return err
	}
	return a.respondObjectiveWith(c, kr.ObjectiveID, krID, created)
}

func (a *Admin) respondObjectiveWith(c fiber.Ctx, objID, krID uint64, created bool) error {
	o, err := a.svc.Objective(c.Context(), objID)
	if err != nil {
		return err
	}
	objs, err := a.svc.List(c.Context(), o.Period)
	if err != nil {
		return err
	}
	evs, err := a.svc.Evaluate(c.Context(), a.today(c), objs)
	if err != nil {
		return err
	}
	for _, e := range evs {
		for _, k := range e.KRs {
			if k.ID == krID {
				out := jsonx.Obj("key_result", a.krJSON(k))
				if created {
					return httpadmin.Created(c, out)
				}
				return httpadmin.OK(c, out)
			}
		}
	}
	return httpadmin.NotFound("Key result")
}

// StoreKR is POST /okr/objectives/:id/key-results.
func (a *Admin) StoreKR(c fiber.Ctx) error {
	oid, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Objective")
	}
	if _, err := a.svc.Objective(c.Context(), oid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpadmin.NotFound("Objective")
		}
		return err
	}
	in, err := a.krInput(c)
	if err != nil {
		return err
	}
	id, err := a.svc.CreateKR(c.Context(), oid, in, a.today(c))
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "okr.key_result.create", "okr_key_result", id)
	return a.respondKR(c, id, true)
}

// UpdateKR is PUT /okr/key-results/:id.
func (a *Admin) UpdateKR(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Key result")
	}
	if _, err := a.svc.KeyResult(c.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpadmin.NotFound("Key result")
		}
		return err
	}
	in, err := a.krInput(c)
	if err != nil {
		return err
	}
	if err := a.svc.UpdateKR(c.Context(), id, in, a.today(c)); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "okr.key_result.update", "okr_key_result", id)
	return a.respondKR(c, id, false)
}

// DestroyKR is DELETE /okr/key-results/:id.
func (a *Admin) DestroyKR(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Key result")
	}
	found, err := a.svc.DeleteKR(c.Context(), id)
	if err != nil {
		return err
	}
	if !found {
		return httpadmin.NotFound("Key result")
	}
	httpadmin.Audit(c, a.logger, "okr.key_result.delete", "okr_key_result", id)
	return httpadmin.OK(c, nil)
}
