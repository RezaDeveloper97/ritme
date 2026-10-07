package telemed

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// Admin limits.
const (
	MaxNameLen       = 120
	MaxHeadlineLen   = 160
	MaxBioLen        = 2000
	MaxLicenceLen    = 32
	MaxInsurers      = 30
	MaxExperience    = 80
	MaxResponseMins  = 10080 // one week
	MinDuration      = 5
	MaxDuration      = 240
	MaxPriceRials    = 10_000_000_000 // 1 billion toman
	MaxNoteLen       = 200
	MaxRules         = 50
	MaxTimeOffDays   = 366
	MaxTimeOffNote   = 190
	PhotoDir         = "doctors"
	PhotoMaxKB       = 4096
	PhotoMinSide     = 200
	adminTimeLayout  = "2006-01-02 15:04"
	adminTimeFormat  = "Y-m-d H:i"
	timeOfDayPattern = `/^(([01][0-9]|2[0-3]):[0-5][0-9]|24:00)$/`
)

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Admin serves /api/admin/v1/telemed/* (docs/go-migration/admin-api.md §19): the minimal doctors CRUD the directory
// needs (B-N7-08 builds the console on it). Editors and super admins edit; deleting a doctor is super only. Writes are
// audit-logged without personal data.
type Admin struct {
	svc       *Service
	q         *store.Queries
	disk      *media.Disk
	optimizer media.Optimizer
	appURL    string
	logger    *slog.Logger
}

// NewAdmin wires the admin handlers.
func NewAdmin(svc *Service, disk *media.Disk, optimizer media.Optimizer, appURL string, logger *slog.Logger) *Admin {
	if logger == nil {
		logger = slog.Default()
	}
	return &Admin{svc: svc, q: svc.Queries(), disk: disk, optimizer: optimizer, appURL: appURL, logger: logger}
}

// Routes mounts the module; static paths before :id.
func (h *Admin) Routes(route Route, kit *httpadmin.Kit) {
	a := kit.Admin
	p := "/telemed/doctors"
	route(fiber.MethodGet, p, a(h.List))
	route(fiber.MethodGet, p+"/options", a(h.Options))
	route(fiber.MethodPost, p, a(h.Store))
	route(fiber.MethodGet, p+"/:id", a(h.Show))
	route(fiber.MethodPut, p+"/:id", a(h.Update))
	route(fiber.MethodDelete, p+"/:id", kit.Super(h.Destroy))
	route(fiber.MethodPost, p+"/:id/photo", a(h.StorePhoto))
	route(fiber.MethodDelete, p+"/:id/photo", a(h.DestroyPhoto))
	route(fiber.MethodPut, p+"/:id/visit-types", a(h.PutVisitTypes))
	route(fiber.MethodPut, p+"/:id/availability", a(h.PutAvailability))
	route(fiber.MethodPost, p+"/:id/time-off", a(h.StoreTimeOff))
	route(fiber.MethodDelete, p+"/:id/time-off/:off", a(h.DestroyTimeOff))
	route(fiber.MethodGet, p+"/:id/slots", a(h.PreviewSlots))
	route(fiber.MethodGet, "/telemed/reviews", a(h.Reviews))
	route(fiber.MethodPut, "/telemed/reviews/:id", a(h.UpdateReview))
}

// ---------------------------------------------------------------------------
// Output

func (h *Admin) photoURL(p sql.NullString) any {
	if !p.Valid || p.String == "" {
		return nil
	}
	return content.PublicURL(h.appURL, p.String)
}

func nullInt64(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func modesOf(col db.NullRawJSON) any {
	if !col.Valid {
		return nil
	}
	return form.Raw(col.V)
}

// visitTypeAdminJSON is a visit type with raw translations.
func visitTypeAdminJSON(v store.TelemedVisitType) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", v.ID, "mode", v.Mode, "duration_minutes", v.DurationMinutes, "price_rials", v.PriceRials,
		"note", form.NullRaw(v.Note), "address", form.NullRaw(v.Address), "is_active", v.IsActive,
	)
}

func ruleAdminJSON(r store.TelemedAvailabilityRule) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID, "weekday", r.Weekday, "start_time", Minutes(int(r.StartMinute)), "end_time", Minutes(int(r.EndMinute)),
		"slot_minutes", r.SlotMinutes, "modes", modesOf(r.Modes),
	)
}

func timeOffJSON(o store.TelemedTimeOff) *jsonx.OrderedMap {
	return jsonx.Obj("id", o.ID, "starts_at", jsonx.ISO8601(wall(o.StartsAt)), "ends_at", jsonx.ISO8601(wall(o.EndsAt)),
		"note", httpadmin.NullString(o.Note))
}

// doctorJSON is the admin record (all languages); insurers and visit types are attached by the caller.
func (h *Admin) doctorJSON(d store.TelemedDoctor) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", d.ID,
		"kind", d.Kind,
		"name", form.Raw(d.Name),
		"headline", form.NullRaw(d.Headline),
		"bio", form.NullRaw(d.Bio),
		"specialty", d.Specialty,
		"city", httpadmin.NullString(d.City),
		"licence_no", d.LicenceNo,
		"experience_years", nullInt16(d.ExperienceYears),
		"photo_path", httpadmin.NullString(d.PhotoPath),
		"photo_url", h.photoURL(d.PhotoPath),
		"response_minutes", nullInt16(d.ResponseMinutes),
		"visits_count", d.VisitsCount,
		"rating", Rating(d.RatingSum, d.RatingCount),
		"reviews_count", d.RatingCount,
		"satisfaction_percent", Satisfaction(d.PositiveCount, d.RatingCount),
		"is_active", d.IsActive,
		"sort_order", d.SortOrder,
		"admin_id", nullInt64(d.AdminID),
		"created_at", httpadmin.Time(d.CreatedAt),
		"updated_at", httpadmin.Time(d.UpdatedAt),
	)
}

// ---------------------------------------------------------------------------
// Read endpoints

// List is GET /telemed/doctors?q=&status=all|active|inactive&page=&per_page= (sort_order, id); q matches the name
// in any language and the licence number.
func (h *Admin) List(c fiber.Ctx) error {
	search := c.Query("q")
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
	total, err := h.q.CountAdminDoctors(c.Context(), store.CountAdminDoctorsParams{
		Pattern: pattern, JsonPattern: jsonPattern, ActiveMin: lo, ActiveMax: hi,
	})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminDoctors(c.Context(), store.ListAdminDoctorsParams{
		Pattern: pattern, JsonPattern: jsonPattern, ActiveMin: lo, ActiveMax: hi,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	modes := map[uint64][]string{}
	if len(ids) > 0 {
		vts, err := h.q.ListActiveVisitTypesByDoctors(c.Context(), ids)
		if err != nil {
			return err
		}
		for _, v := range vts {
			modes[v.DoctorID] = append(modes[v.DoctorID], v.Mode)
		}
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		o := h.doctorJSON(r)
		o.Set("modes", jsonx.List(modes[r.ID]))
		items = append(items, o)
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("q", search, "status", status))
	return httpadmin.OK(c, page)
}

// Options is GET /telemed/doctors/options: enums, limits and the catalog groups the form picks codes from.
func (h *Admin) Options(c fiber.Ctx) error {
	next, err := h.q.NextDoctorSortOrder(c.Context())
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj(
		"kinds", jsonx.List(Kinds),
		"modes", jsonx.List(Modes),
		"weekdays", jsonx.Obj("first", 0, "last", 6, "note", "Go time.Weekday: 0 = Sunday … 6 = Saturday"),
		"catalog_groups", jsonx.Obj("specialties", GroupSpecialties, "cities", GroupCities, "insurers", GroupInsurers),
		"limits", jsonx.Obj(
			"name", MaxNameLen, "headline", MaxHeadlineLen, "bio", MaxBioLen, "licence_no", MaxLicenceLen,
			"insurers", MaxInsurers, "experience_years", MaxExperience, "response_minutes", MaxResponseMins,
			"duration_min", MinDuration, "duration_max", MaxDuration, "price_rials_max", int64(MaxPriceRials),
			"note", MaxNoteLen, "rules", MaxRules, "time_off_days", MaxTimeOffDays,
			"photo_max_kb", PhotoMaxKB, "photo_min_side", PhotoMinSide,
		),
		"currency", "IRR",
		"next_sort_order", next,
	))
}

// find loads the doctor named by :id (404 otherwise).
func (h *Admin) find(c fiber.Ctx) (store.TelemedDoctor, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.TelemedDoctor{}, httpadmin.NotFound("Doctor")
	}
	d, err := h.q.GetDoctor(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return store.TelemedDoctor{}, httpadmin.NotFound("Doctor")
	}
	return d, err
}

// respond answers {doctor} with insurers, visit types, rules and upcoming time off.
func (h *Admin) respond(c fiber.Ctx, id uint64, created bool, msg string) error {
	d, err := h.q.GetDoctor(c.Context(), id)
	if err != nil {
		return err
	}
	o := h.doctorJSON(d)
	insurers, err := h.q.ListDoctorInsurers(c.Context(), id)
	if err != nil {
		return err
	}
	o.Set("insurers", jsonx.List(insurers))
	vts, err := h.q.ListDoctorVisitTypes(c.Context(), id)
	if err != nil {
		return err
	}
	visits := make([]*jsonx.OrderedMap, 0, len(vts))
	for _, v := range vts {
		visits = append(visits, visitTypeAdminJSON(v))
	}
	o.Set("visit_types", jsonx.List(visits))
	rules, err := h.q.ListDoctorRules(c.Context(), id)
	if err != nil {
		return err
	}
	rs := make([]*jsonx.OrderedMap, 0, len(rules))
	for _, r := range rules {
		rs = append(rs, ruleAdminJSON(r))
	}
	o.Set("availability", jsonx.List(rs))
	offs, err := h.q.ListDoctorTimeOff(c.Context(), store.ListDoctorTimeOffParams{DoctorID: id, Since: httpadmin.Now(c).In(civildate.Tehran)})
	if err != nil {
		return err
	}
	ts := make([]*jsonx.OrderedMap, 0, len(offs))
	for _, t := range offs {
		ts = append(ts, timeOffJSON(t))
	}
	o.Set("time_off", jsonx.List(ts))
	body := jsonx.Obj("doctor", o)
	var msgs []string
	if msg != "" {
		msgs = []string{msg}
	}
	if created {
		return httpadmin.Created(c, body, msgs...)
	}
	return httpadmin.OK(c, body, msgs...)
}

// Show is GET /telemed/doctors/:id.
func (h *Admin) Show(c fiber.Ctx) error {
	d, err := h.find(c)
	if err != nil {
		return err
	}
	return h.respond(c, d.ID, false, "")
}

// ---------------------------------------------------------------------------
// Doctor writes

func (h *Admin) doctorRules(c fiber.Ctx) validation.Rules {
	r := validation.Rules{validation.F("kind", "required|string", validation.In(Kinds...))}
	r = append(r, form.Translatable(c, "name", true, "max:"+strconv.Itoa(MaxNameLen))...)
	r = append(r, form.Translatable(c, "headline", false, "max:"+strconv.Itoa(MaxHeadlineLen))...)
	r = append(r, form.Translatable(c, "bio", false, "max:"+strconv.Itoa(MaxBioLen))...)
	return append(r,
		validation.F("specialty", "required|string|max:"+strconv.Itoa(MaxCodeLen)),
		validation.F("city", "nullable|string|max:"+strconv.Itoa(MaxCodeLen)),
		validation.F("licence_no", "required|string|max:"+strconv.Itoa(MaxLicenceLen)),
		validation.F("experience_years", "nullable|integer|between:0,"+strconv.Itoa(MaxExperience)),
		validation.F("response_minutes", "nullable|integer|between:1,"+strconv.Itoa(MaxResponseMins)),
		validation.F("insurers", "nullable|array|max:"+strconv.Itoa(MaxInsurers)),
		validation.F("insurers.*", "required|string|max:"+strconv.Itoa(MaxCodeLen)),
		validation.F("is_active", "nullable|boolean"),
		validation.F("sort_order", "nullable|integer|min:0"),
	)
}

// catalogCheck reports codes that are not active items of their catalog group.
func (h *Admin) catalogCheck(c fiber.Ctx) form.Check {
	return func(in phpval.Map, add form.Add) error {
		known := func(group string) (map[string]bool, error) {
			items, err := h.svc.Catalog().Items(c.Context(), group)
			if err != nil {
				return nil, err
			}
			m := make(map[string]bool, len(items))
			for _, it := range items {
				m[it.Code] = true
			}
			return m, nil
		}
		one := func(field, group string) error {
			v, ok := phpval.Get(in, field)
			s, isStr := v.(string)
			if !ok || !isStr || s == "" {
				return nil
			}
			m, err := known(group)
			if err != nil {
				return err
			}
			if !m[s] {
				add(field, form.Msg(c, "validation.exists", field))
			}
			return nil
		}
		if err := one("specialty", GroupSpecialties); err != nil {
			return err
		}
		if err := one("city", GroupCities); err != nil {
			return err
		}
		v, _ := phpval.Get(in, "insurers")
		if !phpval.IsArray(v) {
			return nil
		}
		m, err := known(GroupInsurers)
		if err != nil {
			return err
		}
		keys, vals := phpval.Entries(v)
		for i, x := range vals {
			if s, isStr := x.(string); isStr && s != "" && !m[s] {
				add("insurers."+phpval.ToString(keys[i]), form.Msg(c, "validation.exists", "insurers"))
			}
		}
		return nil
	}
}

func nullStr(data phpval.Map, key string) sql.NullString {
	s := httpadmin.String(data, key)
	return sql.NullString{String: s, Valid: s != ""}
}

func nullSmall(data phpval.Map, key string) sql.NullInt16 { return form.NullInt16(data, key) }

// insurersOf is the distinct insurer codes in order.
func insurersOf(data phpval.Map) []string {
	v, _ := data.Get("insurers")
	_, vals := phpval.Entries(v)
	seen := map[string]bool{}
	out := []string{}
	for _, x := range vals {
		if s := phpval.ToString(x); s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func (h *Admin) saveInsurers(c fiber.Ctx, q *store.Queries, id uint64, data phpval.Map) error {
	if _, sent := data.Get("insurers"); !sent {
		return nil
	}
	if err := q.DeleteDoctorInsurers(c.Context(), id); err != nil {
		return err
	}
	for _, code := range insurersOf(data) {
		if err := q.InsertDoctorInsurer(c.Context(), store.InsertDoctorInsurerParams{DoctorID: id, Insurer: code}); err != nil {
			return err
		}
	}
	return nil
}

func translatable(data phpval.Map, key string) json.RawMessage {
	col := form.Clean(data, key)
	if !col.Valid {
		return nil
	}
	return col.V
}

// Store is POST /telemed/doctors (201 {doctor}). The doctor is listed once it has an active visit type.
func (h *Admin) Store(c fiber.Ctx) error {
	data, err := form.Validate(c, h.doctorRules(c), h.catalogCheck(c))
	if err != nil {
		return err
	}
	sortOrder := int64(0)
	if form.Has(data, "sort_order") {
		sortOrder = form.Int(data, "sort_order", 0)
	} else if sortOrder, err = h.q.NextDoctorSortOrder(c.Context()); err != nil {
		return err
	}
	active := true
	if form.Has(data, "is_active") {
		active = httpadmin.Bool(data, "is_active")
	}
	tx, err := h.svc.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	id, err := q.InsertDoctor(c.Context(), store.InsertDoctorParams{
		Kind: httpadmin.String(data, "kind"), Name: translatable(data, "name"), Headline: form.Clean(data, "headline"),
		Bio: form.Clean(data, "bio"), Specialty: httpadmin.String(data, "specialty"), City: nullStr(data, "city"),
		LicenceNo: httpadmin.String(data, "licence_no"), ExperienceYears: nullSmall(data, "experience_years"),
		ResponseMinutes: nullSmall(data, "response_minutes"), IsActive: active,
		SortOrder: int32(sortOrder), //nolint:gosec // G115: clamped by form.Int / MAX()+1
		Now:       httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	uid := uint64(id) //nolint:gosec // G115: auto-increment ids are positive
	if err := h.saveInsurers(c, q, uid, data); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.create", "telemed_doctor", uid)
	return h.respond(c, uid, true, "Doctor created.")
}

// Update is PUT /telemed/doctors/:id: the full form (kind, name, specialty and licence_no are required again);
// optional fields that are absent keep their value, null clears them; `insurers` replaces the set when sent.
func (h *Admin) Update(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, h.doctorRules(c), h.catalogCheck(c))
	if err != nil {
		return err
	}
	sent := func(k string) bool { _, ok := data.Get(k); return ok }
	p := store.UpdateDoctorParams{
		Kind: httpadmin.String(data, "kind"), Name: translatable(data, "name"), Headline: cur.Headline, Bio: cur.Bio,
		Specialty: httpadmin.String(data, "specialty"), City: cur.City, LicenceNo: httpadmin.String(data, "licence_no"),
		ExperienceYears: cur.ExperienceYears, ResponseMinutes: cur.ResponseMinutes, IsActive: cur.IsActive,
		SortOrder: cur.SortOrder, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}
	if sent("headline") {
		p.Headline = form.Clean(data, "headline")
	}
	if sent("bio") {
		p.Bio = form.Clean(data, "bio")
	}
	if sent("city") {
		p.City = nullStr(data, "city")
	}
	if sent("experience_years") {
		p.ExperienceYears = nullSmall(data, "experience_years")
	}
	if sent("response_minutes") {
		p.ResponseMinutes = nullSmall(data, "response_minutes")
	}
	if form.Has(data, "is_active") {
		p.IsActive = httpadmin.Bool(data, "is_active")
	}
	if form.Has(data, "sort_order") {
		p.SortOrder = form.Int32(data, "sort_order", 0)
	}
	tx, err := h.svc.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	if err := q.UpdateDoctor(c.Context(), p); err != nil {
		return err
	}
	if err := h.saveInsurers(c, q, cur.ID, data); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.update", "telemed_doctor", cur.ID)
	return h.respond(c, cur.ID, false, "Doctor updated.")
}

// Destroy is DELETE /telemed/doctors/:id (super admins): the doctor with its visit types, availability, time off and
// reviews, and the photo file. Prefer is_active=false for a doctor users have seen.
func (h *Admin) Destroy(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	n, err := h.q.DeleteDoctor(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpadmin.NotFound("Doctor")
	}
	if cur.PhotoPath.Valid {
		h.deleteFile(c, cur.PhotoPath.String)
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.delete", "telemed_doctor", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Doctor deleted.")
}

func (h *Admin) deleteFile(c fiber.Ctx, rel string) {
	if err := h.disk.Delete(rel); err != nil {
		h.logger.WarnContext(c.Context(), "telemed admin: could not delete old photo",
			slog.String("path", rel), slog.String("error", err.Error()))
	}
}

// StorePhoto is POST /telemed/doctors/:id/photo (multipart `photo`: jpeg/png/webp ≤ 4 MB, ≥ 200×200), stored as WebP
// fitted to 1080×1080 under doctors/ on the public disk; the old file is removed.
func (h *Admin) StorePhoto(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	var img *media.Image
	if _, err := form.Validate(c, validation.Rules{}, form.ImageRule{Field: "photo", Required: true, MaxKB: PhotoMaxKB,
		MinWidth: PhotoMinSide, MinHeight: PhotoMinSide}.Check(c, &img)); err != nil {
		return err
	}
	rel, err := h.disk.Store(h.optimizer, PhotoDir, *img)
	if err != nil {
		return err
	}
	if err := h.q.SetDoctorPhoto(c.Context(), store.SetDoctorPhotoParams{
		PhotoPath: sql.NullString{String: rel, Valid: true}, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}); err != nil {
		h.deleteFile(c, rel)
		return err
	}
	if cur.PhotoPath.Valid {
		h.deleteFile(c, cur.PhotoPath.String)
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.photo", "telemed_doctor", cur.ID)
	return h.respond(c, cur.ID, false, "Photo saved.")
}

// DestroyPhoto is DELETE /telemed/doctors/:id/photo.
func (h *Admin) DestroyPhoto(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	if err := h.q.SetDoctorPhoto(c.Context(), store.SetDoctorPhotoParams{Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID}); err != nil {
		return err
	}
	if cur.PhotoPath.Valid {
		h.deleteFile(c, cur.PhotoPath.String)
	}
	httpadmin.Audit(c, h.logger, "telemed_doctor.photo_delete", "telemed_doctor", cur.ID)
	return h.respond(c, cur.ID, false, "Photo removed.")
}
