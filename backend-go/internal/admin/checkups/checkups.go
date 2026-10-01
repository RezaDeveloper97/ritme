// Package checkups is the admin API of the periodic-checkup catalog (T-M4-03,
// docs/checkups/README.md): /api/admin/v1/checkup-types — list, options, show, create,
// update, delete (refused while records exist), reorder and per-type stats. Only the shared
// catalog (`user_id IS NULL`) is reachable; users' custom checkups are invisible here (404).
// The user API reads the same rows without a cache, so an edit is live on the next request.
package checkups

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// StatsWindowDays is the "records in the last N days" window of GET /checkup-types/stats.
const StatsWindowDays = 30

// CodeInUse is the 422 error_code of deleting a type that has records.
const CodeInUse = "in_use"

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Handlers serve the checkup-type endpoints.
type Handlers struct {
	db     *sql.DB
	q      *store.Queries
	logger *slog.Logger
}

// New builds the handlers.
func New(db *sql.DB, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{db: db, q: store.New(db), logger: logger}
}

// Routes registers the endpoints (editor and super admins); static paths before :id.
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	a := kit.Admin
	route(fiber.MethodGet, "/checkup-types", a(h.List))
	route(fiber.MethodGet, "/checkup-types/options", a(h.Options))
	route(fiber.MethodGet, "/checkup-types/stats", a(h.Stats))
	route(fiber.MethodPost, "/checkup-types/reorder", a(h.Reorder))
	route(fiber.MethodPost, "/checkup-types", a(h.Store))
	route(fiber.MethodGet, "/checkup-types/:id", a(h.Show))
	route(fiber.MethodPut, "/checkup-types/:id", a(h.Update))
	route(fiber.MethodDelete, "/checkup-types/:id", a(h.Destroy))
}

// ---------------------------------------------------------------------------
// Output

func nullInt(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return n.Int16
}

// typeJSON is the CheckupType record (columns in table order, JSON columns decoded).
func typeJSON(t *store.CheckupType, recordsCount int64) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", t.ID,
		"key", httpadmin.NullString(t.Key),
		"category", t.Category,
		"title", form.Raw(t.Title),
		"subtitle", form.NullRaw(t.Subtitle),
		"why", form.NullRaw(t.Why),
		"performed_by", t.PerformedBy,
		"icon", httpadmin.NullString(t.Icon),
		"tone", t.Tone,
		"interval_months", t.IntervalMonths,
		"interval_months_max", nullInt(t.IntervalMonthsMax),
		"age_min", nullInt(t.AgeMin),
		"age_max", nullInt(t.AgeMax),
		"cycle_day_from", nullInt(t.CycleDayFrom),
		"cycle_day_to", nullInt(t.CycleDayTo),
		"remind_lead_days", t.RemindLeadDays,
		"prep_steps", listOrEmpty(t.PrepSteps),
		"guide_steps", listOrEmpty(t.GuideSteps),
		"finding_options", listOrEmpty(t.FindingOptions),
		"hide_in_pregnancy", t.HideInPregnancy,
		"audiences", form.NullRaw(t.Audiences),
		"is_active", t.IsActive,
		"sort_order", t.SortOrder,
		"source_note", httpadmin.NullString(t.SourceNote),
		"records_count", recordsCount,
		"created_at", httpadmin.Time(t.CreatedAt),
		"updated_at", httpadmin.Time(t.UpdatedAt),
	)
}

// listOrEmpty decodes a JSON array column; NULL is [] so admin-web can always map it.
func listOrEmpty(col db.NullRawJSON) any {
	if !col.Valid {
		return []any{}
	}
	v := form.Raw(col.V)
	if v == nil {
		return []any{}
	}
	return v
}

// ---------------------------------------------------------------------------
// Read endpoints

// List is GET /checkup-types?q=&status=all|active|inactive&page=&per_page= (sort_order, id).
func (h *Handlers) List(c fiber.Ctx) error {
	search := strings.TrimSpace(c.Query("q"))
	status := c.Query("status")
	var only *bool
	switch status {
	case "active":
		only = new(bool)
		*only = true
	case "inactive":
		only = new(bool)
	default:
		status = "all"
	}
	lo, hi := form.BoolRange(only)
	pattern, jsonPattern := form.Contains(search), form.ContainsJSON(search)
	total, err := h.q.CountAdminCheckupTypes(c.Context(), store.CountAdminCheckupTypesParams{
		Pattern: pattern, JsonPattern: jsonPattern, ActiveMin: lo, ActiveMax: hi,
	})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminCheckupTypes(c.Context(), store.ListAdminCheckupTypesParams{
		Pattern: pattern, JsonPattern: jsonPattern, ActiveMin: lo, ActiveMax: hi,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	counts, err := h.recordStats(c, civildate.InTehran(httpadmin.Now(c)))
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, typeJSON(&rows[i], counts[rows[i].ID].RecordsTotal))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("q", search, "status", status))
	return httpadmin.OK(c, page)
}

// Options is GET /checkup-types/options: the enum values and limits the form needs.
func (h *Handlers) Options(c fiber.Ctx) error {
	next, err := h.q.NextAdminCheckupTypeSortOrder(c.Context())
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj(
		"categories", jsonx.List(Categories),
		"performed_by", jsonx.List(Performers),
		"icons", jsonx.List(Icons),
		"tones", jsonx.List(Tones),
		"audiences", jsonx.List(Audiences()),
		"default_tone", DefaultTone,
		"default_remind_lead_days", DefaultRemindLeadDays,
		"max_steps", MaxSteps,
		"max_cycle_day", MaxCycleDay,
		"max_interval_months", MaxIntervalMonths,
		"next_sort_order", next,
	))
}

// Show is GET /checkup-types/:id.
func (h *Handlers) Show(c fiber.Ctx) error {
	t, err := h.find(c)
	if err != nil {
		return err
	}
	return h.respond(c, t.ID, false, "")
}

// Stats is GET /checkup-types/stats: per catalog type (sort order), users with records, records
// done in the last 30 days, and users whose latest record is past its due-by date.
func (h *Handlers) Stats(c fiber.Ctx) error {
	today := civildate.InTehran(httpadmin.Now(c))
	since := today.AddDays(-StatsWindowDays)
	rows, err := h.q.ListAdminCheckupTypes(c.Context(), store.ListAdminCheckupTypesParams{
		Pattern: "%", JsonPattern: "%", ActiveMin: false, ActiveMax: true, Limit: 1 << 30, Offset: 0,
	})
	if err != nil {
		return err
	}
	counts, err := h.recordStats(c, today)
	if err != nil {
		return err
	}
	overdue, err := h.q.CheckupOverdueUsersByType(c.Context(), civildate.NullDate{Date: today, Valid: true})
	if err != nil {
		return err
	}
	late := make(map[uint64]int64, len(overdue))
	for _, o := range overdue {
		late[o.CheckupTypeID] = o.OverdueUsers
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		t := &rows[i]
		s := counts[t.ID]
		items = append(items, jsonx.Obj(
			"id", t.ID,
			"key", httpadmin.NullString(t.Key),
			"title", form.Raw(t.Title),
			"category", t.Category,
			"is_active", t.IsActive,
			"records_total", s.RecordsTotal,
			"users_with_records", s.UsersWithRecords,
			"records_last_30_days", s.RecordsRecent,
			"overdue_users", late[t.ID],
		))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"items", jsonx.List(items),
		"window_days", StatsWindowDays,
		"since", since,
		"today", today,
	))
}

// recordStats is CheckupRecordStatsByType keyed by type id (recent = done on/after today − 30).
func (h *Handlers) recordStats(c fiber.Ctx, today civildate.Date) (map[uint64]store.CheckupRecordStatsByTypeRow, error) {
	rows, err := h.q.CheckupRecordStatsByType(c.Context(), today.AddDays(-StatsWindowDays))
	if err != nil {
		return nil, fmt.Errorf("checkups admin: record stats: %w", err)
	}
	out := make(map[uint64]store.CheckupRecordStatsByTypeRow, len(rows))
	for _, r := range rows {
		out[r.CheckupTypeID] = r
	}
	return out, nil
}

// find loads the catalog row named by :id (404 for a missing, custom or non-numeric id).
func (h *Handlers) find(c fiber.Ctx) (store.CheckupType, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.CheckupType{}, httpadmin.NotFound("Checkup type")
	}
	t, err := h.q.GetAdminCheckupType(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return store.CheckupType{}, httpadmin.NotFound("Checkup type")
	}
	return t, err
}

// respond reloads the row and answers {checkup_type} with 200 or 201.
func (h *Handlers) respond(c fiber.Ctx, id uint64, created bool, msg string) error {
	t, err := h.q.GetAdminCheckupType(c.Context(), id)
	if err != nil {
		return err
	}
	n, err := h.q.CountCheckupRecordsOfType(c.Context(), id)
	if err != nil {
		return err
	}
	body := jsonx.Obj("checkup_type", typeJSON(&t, n))
	var msgs []string
	if msg != "" {
		msgs = []string{msg}
	}
	if created {
		return httpadmin.Created(c, body, msgs...)
	}
	return httpadmin.OK(c, body, msgs...)
}

// ---------------------------------------------------------------------------
// Mutations

// Store is POST /checkup-types.
func (h *Handlers) Store(c fiber.Ctx) error {
	data, err := h.validate(c, nil)
	if err != nil {
		return err
	}
	sortOrder := int64(0)
	if form.Has(data, "sort_order") {
		sortOrder = form.Int(data, "sort_order", 0)
	} else if sortOrder, err = h.q.NextAdminCheckupTypeSortOrder(c.Context()); err != nil {
		return err
	}
	p := buildParams(c, data, nil)
	res, err := h.q.CreateAdminCheckupType(c.Context(), store.CreateAdminCheckupTypeParams{
		TypeKey: form.Str(data, "key"), Category: p.Category, Title: p.Title, Subtitle: p.Subtitle, Why: p.Why,
		PerformedBy: p.PerformedBy, Icon: p.Icon, Tone: p.Tone, IntervalMonths: p.IntervalMonths,
		IntervalMonthsMax: p.IntervalMonthsMax, AgeMin: p.AgeMin, AgeMax: p.AgeMax,
		CycleDayFrom: p.CycleDayFrom, CycleDayTo: p.CycleDayTo, RemindLeadDays: p.RemindLeadDays,
		PrepSteps: p.PrepSteps, GuideSteps: p.GuideSteps, FindingOptions: p.FindingOptions,
		HideInPregnancy: p.HideInPregnancy, Audiences: p.Audiences, IsActive: p.IsActive,
		SortOrder:  int32(sortOrder), //nolint:gosec // G115: clamped to int32 by form.Int / a small MAX()+1
		SourceNote: p.SourceNote, Now: httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	uid := uint64(id) //nolint:gosec // G115: auto-increment ids are positive
	httpadmin.Audit(c, h.logger, "checkup_type.create", "checkup_type", uid)
	return h.respond(c, uid, true, "Checkup type created.")
}

// Update is PUT /checkup-types/:id (the key is fixed at creation; optional fields that are
// absent keep their stored value, booleans follow $request->boolean()).
func (h *Handlers) Update(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	data, err := h.validate(c, &cur)
	if err != nil {
		return err
	}
	p := buildParams(c, data, &cur)
	p.SortOrder = cur.SortOrder
	if form.Has(data, "sort_order") {
		p.SortOrder = form.Int32(data, "sort_order", 0)
	}
	p.Now, p.ID = httpadmin.DBTime(httpadmin.Now(c)), cur.ID
	if err := h.q.UpdateAdminCheckupType(c.Context(), p); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "checkup_type.update", "checkup_type", cur.ID)
	return h.respond(c, cur.ID, false, "Checkup type updated.")
}

// Destroy is DELETE /checkup-types/:id: refused with 422 in_use while any user has a record of
// the type (deleting would cascade to their health records) — deactivate it instead.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	res, err := h.q.DeleteUnusedAdminCheckupType(c.Context(), store.DeleteUnusedAdminCheckupTypeParams{TypeID: cur.ID})
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		count, err := h.q.CountCheckupRecordsOfType(c.Context(), cur.ID)
		if err != nil {
			return err
		}
		if count == 0 {
			return httpadmin.NotFound("Checkup type") // deleted concurrently
		}
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse,
			"This checkup type has "+strconv.FormatInt(count, 10)+" records; deactivate it instead.",
			"records_count", count)
	}
	httpadmin.Audit(c, h.logger, "checkup_type.delete", "checkup_type", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Checkup type deleted.")
}

// Reorder is POST /checkup-types/reorder {ids:[…]}: every catalog id exactly once, in the new
// order; sort_order becomes 1…n.
func (h *Handlers) Reorder(c fiber.Ctx) error {
	all, err := h.q.ListAdminCheckupTypeIDs(c.Context())
	if err != nil {
		return err
	}
	ids, err := validateReorder(c, all)
	if err != nil {
		return err
	}
	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	now := httpadmin.DBTime(httpadmin.Now(c))
	for i, id := range ids {
		if err := q.SetAdminCheckupTypeSortOrder(c.Context(), store.SetAdminCheckupTypeSortOrderParams{
			SortOrder: int32(i + 1), Now: now, ID: id, //nolint:gosec // G115: bounded by the catalog size
		}); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "checkup_type.reorder", "checkup_type", 0,
		slog.Int("count", len(ids)))
	out := make([]*jsonx.OrderedMap, 0, len(ids))
	for i, id := range ids {
		out = append(out, jsonx.Obj("id", id, "sort_order", i+1))
	}
	return httpadmin.OK(c, jsonx.Obj("items", jsonx.List(out)), "Order saved.")
}
