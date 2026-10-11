package cityservices

// CB-DIR-05 — the mother & child directory in admin, ADDED to bloom's city-services admin (admin.go, B-N10-05): the
// same Admin, the same /api/admin/v1/city-services/* family (capability content) and booking queue
// /api/admin/v1/city-service-bookings/* (capability support), plus the application review queue
// /api/admin/v1/city-service-applications/* (capability content, permissions.go). There is no second admin section.
//
//   - Places: the directory fields (status draft | pending | published | suspended, verified badge, map point, about,
//     age ranges, amenities, weekly hours, website, cancel policy, rules), the structured photos, services / packages
//     and slots (single or a weekly generator; a slot's capacity can never drop below its booked seats).
//   - Review moderation (pending → published | hidden) recomputes the place's rating_avg / rating_count in the same
//     transaction; the wrong-information report queue (open → resolved | dismissed).
//   - Bookings: bloom's queue grows the directory fields; a confirm / decline tells the mother through the
//     StatusNotifier — discreet (booking id + status only, never the place, category or child), fake provider logs ids.
//
// Privacy: the applicant's contact name / phone are only sent to admins who also hold «support» (the capability that
// already sees booking contacts); everyone else gets null + contact_hidden. Licences are private files: the team
// gets a fresh 5-minute signed link per view (audited), never a path. Logs and audit rows carry ids only.

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/cityservices/join"
	joinstore "github.com/ritme/backend-go/internal/cityservices/join/store"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Admin limits of CB-DIR-05.
const (
	// CodeBelowBooked is the 422 error_code of a slot capacity under its booked seats.
	CodeBelowBooked = "capacity_below_booked"
	MaxSlotCapacity = 500
	MaxGenerated    = 500 // slots per generator run
	MaxGenerateDays = 92
	MaxAdminSlots   = 500
	MaxTimesPerDay  = 12
)

// AdminFiles is the slice of internal/files the admin uses (*files.Service): licence links and photo cleanup.
type AdminFiles interface {
	Get(ctx context.Context, owner, id uint64) (files.File, error)
	Link(f files.File, now time.Time, ttl time.Duration) (files.Link, error)
	Delete(ctx context.Context, owner, id uint64) error
}

// WithFiles wires the file service (licence links, deleting a rejected application's photos).
func (a *Admin) WithFiles(fs AdminFiles) *Admin { a.files = fs; return a }

// WithStatusNotifier swaps the booking status notifier (tests).
func (a *Admin) WithStatusNotifier(n StatusNotifier) *Admin { a.status = n; return a }

// StatusNotice is everything the mother is told when the team decides her booking: the ids and the new status. The
// message itself is generic («وضعیت رزروت به‌روز شد») — no place, category, child or time, so a lock-screen preview
// reveals nothing (discreet).
type StatusNotice struct {
	BookingID uint64
	UserID    uint64
	Status    string
}

// StatusNotifier tells the mother about a confirmed / declined booking. A failure never fails the move.
type StatusNotifier interface {
	BookingStatusChanged(ctx context.Context, n StatusNotice) error
}

// FakeStatusNotifier logs ids and the status only.
type FakeStatusNotifier struct{ Logger *slog.Logger }

// BookingStatusChanged logs the ids.
func (f FakeStatusNotifier) BookingStatusChanged(ctx context.Context, n StatusNotice) error {
	l := f.Logger
	if l == nil {
		l = slog.Default()
	}
	l.InfoContext(ctx, "cityservices: booking status notice (fake notifier)", slog.Uint64("booking_id", n.BookingID),
		slog.Uint64("user_id", n.UserID), slog.String("status", n.Status))
	return nil
}

// notifiedStatuses are the moves the mother hears about.
var notifiedStatuses = []string{StatusConfirmed, StatusRejected}

func (a *Admin) notifyStatus(c fiber.Ctx, b Booking, to string) {
	if !slices.Contains(notifiedStatuses, to) || a.status == nil {
		return
	}
	if err := a.status.BookingStatusChanged(c.Context(), StatusNotice{BookingID: b.ID, UserID: b.UserID, Status: to}); err != nil {
		a.logger.WarnContext(c.Context(), "cityservices: status notice failed", slog.Uint64("booking_id", b.ID))
	}
}

// dirRoutes registers the CB-DIR-05 endpoints (called from Routes).
func (a *Admin) dirRoutes(route Route, kit *httpadmin.Kit) {
	k := kit.Admin
	const l = "/city-services/listings/:id"
	route(fiber.MethodPut, l+"/directory", k(a.SaveDirectory))
	route(fiber.MethodDelete, l+"/file-photos/:fileId", k(a.RemoveFilePhoto))
	route(fiber.MethodGet, l+"/services", k(a.ListServices))
	route(fiber.MethodPost, l+"/services", k(func(c fiber.Ctx) error { return a.SaveService(c, false) }))
	route(fiber.MethodPut, l+"/services/:serviceId", k(func(c fiber.Ctx) error { return a.SaveService(c, true) }))
	route(fiber.MethodDelete, l+"/services/:serviceId", k(a.DestroyService))
	route(fiber.MethodGet, l+"/slots", k(a.ListSlots))
	route(fiber.MethodPost, l+"/slots", k(a.StoreSlot))
	route(fiber.MethodPost, l+"/slots/generate", k(a.GenerateSlots))
	route(fiber.MethodPut, l+"/slots/:slotId", k(a.UpdateSlot))
	route(fiber.MethodDelete, l+"/slots/:slotId", k(a.DestroySlot))

	route(fiber.MethodGet, "/city-services/reviews", k(a.ListReviews))
	route(fiber.MethodPut, "/city-services/reviews/:id", k(a.ModerateReview))
	route(fiber.MethodGet, "/city-services/reports", k(a.ListReports))
	route(fiber.MethodPut, "/city-services/reports/:id", k(a.CloseReport))

	const p = "/city-service-applications"
	route(fiber.MethodGet, p, k(a.ListApplications))
	route(fiber.MethodGet, p+"/:id", k(a.ShowApplication))
	route(fiber.MethodGet, p+"/:id/documents/:slot", k(a.DocumentLink))
	route(fiber.MethodPost, p+"/:id/review", k(a.ReviewApplication))
	route(fiber.MethodPost, p+"/:id/approve", k(a.ApproveApplication))
}

// ---------------------------------------------------------------------------
// Shared JSON

func idList(v []uint64) []uint64 {
	if v == nil {
		return []uint64{}
	}
	return v
}

func hours(v []DayHours) []DayHours {
	if v == nil {
		return []DayHours{}
	}
	return v
}

// listingDirJSON appends the directory columns to bloom's admin listing shape.
func (a *Admin) listingDirJSON(o *jsonx.OrderedMap, l Listing) {
	o.Set("status", l.Status)
	o.Set("verified", l.Verified)
	o.Set("lat", nullFloat(l.Lat))
	o.Set("lng", nullFloat(l.Lng))
	o.Set("about", httpadmin.NullString(l.About))
	o.Set("age_ranges", list(l.AgeRanges))
	o.Set("amenities", list(l.Amenities))
	o.Set("weekly_hours", hours(l.WeeklyHours))
	o.Set("photo_file_ids", idList(l.PhotoFileIDs))
	o.Set("file_photo_urls", list(l.FilePhotoURLs))
	o.Set("website_url", httpadmin.NullString(l.WebsiteURL))
	o.Set("cancel_free_hours", nullInt(l.CancelFreeHours))
	o.Set("cancel_policy", httpadmin.NullString(l.CancelPolicy))
	o.Set("rules", httpadmin.NullString(l.Rules))
	o.Set("rating_avg", nullFloat(l.RatingAvg))
	o.Set("rating_count", l.RatingCount)
}

// filePhotos resolves the structured photo URLs of ls in one query.
func (a *Admin) filePhotos(ctx context.Context, ls []Listing) error {
	var all []uint64
	for _, l := range ls {
		all = append(all, l.PhotoFileIDs...)
	}
	if len(all) == 0 {
		return nil
	}
	paths, err := a.store.PhotoPaths(ctx, all)
	if err != nil {
		return err
	}
	for i := range ls {
		ls[i].FilePhotoURLs = FilePhotoURLs(a.appURL, ls[i].PhotoFileIDs, paths)
	}
	return nil
}

// bookingDirJSON appends the directory fields to bloom's admin booking shape. The child's first name and age reach
// only this queue (capability support), never a log.
func (a *Admin) bookingDirJSON(o *jsonx.OrderedMap, b Booking) {
	var service, slot, child, review any
	if b.ServiceID.Valid {
		service = jsonx.Obj("id", b.ServiceID.Int64, "name", nullStr(b.ServiceName), "duration_minutes", nullInt(b.ServiceDuration))
	}
	if b.SlotID.Valid {
		slot = jsonx.Obj("id", b.SlotID.Int64, "starts_at", nullWall(b.SlotStartsAt), "ends_at", nullWall(b.SlotEndsAt))
	}
	if b.ChildID.Valid && b.ChildName.Valid {
		child = jsonx.Obj("name", b.ChildName.String, "age_months", b.ChildAgeMonths())
	}
	if b.ReviewID.Valid {
		review = jsonx.Obj("id", b.ReviewID.Int64, "rating", nullInt(b.ReviewRating), "status", nullStr(b.ReviewStatus))
	}
	o.Set("code", nullStr(b.Code))
	o.Set("service", service)
	o.Set("slot", slot)
	o.Set("child", child)
	o.Set("price_toman", nullInt(b.PriceToman))
	o.Set("confirmed_at", httpadmin.Time(b.ConfirmedAt))
	o.Set("cancelled_at", httpadmin.Time(b.CancelledAt))
	o.Set("cancelled_by", nullStr(b.CancelledBy))
	o.Set("completed_at", httpadmin.Time(b.CompletedAt))
	o.Set("review", review)
}

func strList(data phpval.Map, key string) []string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return nil
	}
	_, vals := phpval.Entries(v)
	out := make([]string, 0, len(vals))
	for _, x := range vals {
		if s := strings.TrimSpace(phpval.ToString(x)); s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func nullInt64(data phpval.Map, key string) sql.NullInt64 {
	if !form.Has(data, key) {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: form.Int(data, key, 0), Valid: true}
}

func nullFloat64(data phpval.Map, key string) sql.NullFloat64 {
	if !form.Has(data, key) {
		return sql.NullFloat64{}
	}
	v, _ := data.Get(key)
	return sql.NullFloat64{Float64: phpval.ToFloat(v), Valid: true}
}

// ---------------------------------------------------------------------------
// Place directory fields

// SaveDirectory is PUT /city-services/listings/:id/directory.
func (a *Admin) SaveDirectory(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	F := validation.F
	rules := validation.Rules{
		F("status", "required", validation.In(ListingStatuses...)),
		F("verified", "nullable", "boolean"),
		F("lat", "nullable", "numeric", "between:-90,90"),
		F("lng", "nullable", "numeric", "between:-180,180"),
		F("about", "nullable", "string", "max:5000"),
		F("age_ranges", "nullable", "list", "max:20"),
		F("age_ranges.*", "required", "string", "max:64"),
		F("amenities", "nullable", "list", "max:30"),
		F("amenities.*", "required", "string", "max:64"),
		F("weekly_hours", "nullable", "list", "max:14"),
		F("weekly_hours.*", "required", "array"),
		F("weekly_hours.*.day", "required", "integer", "min:0", "max:6"),
		F("weekly_hours.*.open", "required", "date_format:H:i"),
		F("weekly_hours.*.close", "required", "date_format:H:i"),
		F("website_url", "nullable", "string", "max:1000"),
		F("cancel_free_hours", "nullable", "integer", "min:0", "max:720"),
		F("cancel_policy", "nullable", "string", "max:300"),
		F("rules", "nullable", "string", "max:5000"),
	}
	data, err := form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		if form.Has(in, "lat") != form.Has(in, "lng") {
			add("lat", form.Msg(c, "validation.required_with", "lat", "values", "lng"))
		}
		if u := strings.TrimSpace(httpadmin.String(in, "website_url")); u != "" && !isHTTPURL(u) {
			add("website_url", form.Msg(c, "validation.url", "website_url"))
		}
		for field, group := range map[string]string{"age_ranges": catalogAgeRange, "amenities": catalogAmenity} {
			codes := strList(in, field)
			if len(codes) == 0 {
				continue
			}
			opts, err := a.store.CatalogOptions(c, group)
			if err != nil {
				return err
			}
			for _, code := range codes {
				if !slices.ContainsFunc(opts, func(o CatalogOption) bool { return o.Code == code }) {
					add(field, form.Msg(c, "validation.in", field))
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	in := DirInput{Status: httpadmin.String(data, "status"), Verified: httpadmin.Bool(data, "verified"),
		Lat: nullFloat64(data, "lat"), Lng: nullFloat64(data, "lng"), About: nullable(data, "about"),
		AgeRanges: strList(data, "age_ranges"), Amenities: strList(data, "amenities"),
		WebsiteURL: nullable(data, "website_url"), CancelFreeHours: nullInt64(data, "cancel_free_hours"),
		CancelPolicy: nullable(data, "cancel_policy"), Rules: nullable(data, "rules")}
	if v, ok := data.Get("weekly_hours"); ok && v != nil {
		_, rows := phpval.Entries(v)
		for _, r := range rows {
			day, _ := phpval.Get(r, "day")
			open, _ := phpval.Get(r, "open")
			closeAt, _ := phpval.Get(r, "close")
			in.WeeklyHours = append(in.WeeklyHours, DayHours{Day: int(phpval.ToFloat(day)), Open: phpval.ToString(open), Close: phpval.ToString(closeAt)})
		}
	}
	if err := a.store.SaveListingDir(c, cur.ID, in, a.now(c).Time); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_listing.directory", "city_service_listing", cur.ID, slog.String("status", in.Status))
	return a.listingReply(c, cur.ID, "Saved.")
}

func (a *Admin) listingReply(c fiber.Ctx, id uint64, msg string) error {
	l, err := a.store.AdminListing(c, id)
	if err != nil {
		return err
	}
	ls := []Listing{l}
	if err := a.filePhotos(c, ls); err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("listing", a.listingJSON(ls[0])), msg)
}

// RemoveFilePhoto is DELETE /city-services/listings/:id/file-photos/:fileId: takes a structured photo off the place
// (the file stays with its owner, who can delete it under /files).
func (a *Admin) RemoveFilePhoto(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	fid, ok := httpadmin.ID(c, "fileId")
	i := slices.Index(cur.PhotoFileIDs, fid)
	if !ok || i < 0 {
		return httpadmin.NotFound("Photo")
	}
	if err := a.store.SetFilePhotos(c, cur.ID, slices.Delete(slices.Clone(cur.PhotoFileIDs), i, i+1), a.now(c).Time); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_listing.file_photo_remove", "city_service_listing", cur.ID)
	return a.listingReply(c, cur.ID, "Photo removed.")
}

// ---------------------------------------------------------------------------
// Services

func (a *Admin) serviceJSON(v ServiceRow) *jsonx.OrderedMap {
	o := ServiceJSON(v.Service)
	o.Set("sort_order", v.SortOrder)
	o.Set("is_active", v.IsActive)
	o.Set("slots_count", v.Slots)
	o.Set("bookings_count", v.Bookings)
	return o
}

// ListServices is GET /city-services/listings/:id/services (every state).
func (a *Admin) ListServices(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	rows, err := a.store.AdminServices(c, cur.ID)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, v := range rows {
		items = append(items, a.serviceJSON(v))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

func (a *Admin) findService(c fiber.Ctx, listingID uint64) (ServiceRow, error) {
	id, ok := httpadmin.ID(c, "serviceId")
	if !ok {
		return ServiceRow{}, httpadmin.NotFound("Service")
	}
	v, err := a.store.AdminService(c, listingID, id)
	if errors.Is(err, ErrNotFound) {
		return v, httpadmin.NotFound("Service")
	}
	return v, err
}

// SaveService is POST /city-services/listings/:id/services and PUT …/services/:serviceId.
func (a *Admin) SaveService(c fiber.Ctx, update bool) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	var id uint64
	if update {
		v, err := a.findService(c, cur.ID)
		if err != nil {
			return err
		}
		id = v.ID
	}
	F := validation.F
	data, err := form.Validate(c, validation.Rules{
		F("kind", "required", validation.In(join.ServiceKinds...)),
		F("name", "required", "string", "max:160"),
		F("description", "nullable", "string", "max:300"),
		F("format", "nullable", validation.In(join.ServiceFormats...)),
		F("duration_minutes", "nullable", "integer", "min:5", "max:600"),
		F("age_min_months", "nullable", "integer", "min:0", "max:240"),
		F("age_max_months", "nullable", "integer", "min:0", "max:240"),
		F("capacity", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxSlotCapacity)),
		F("price_toman", "nullable", "integer", "min:0", "max:"+strconv.Itoa(join.MaxPrice)),
		F("package_sessions", "nullable", "integer", "min:1", "max:200"),
		F("package_validity_days", "nullable", "integer", "min:1", "max:730"),
		F("sort_order", "nullable", "integer"),
		F("is_active", "nullable", "boolean"),
	}, func(in phpval.Map, add form.Add) error {
		lo, hi := nullInt64(in, "age_min_months"), nullInt64(in, "age_max_months")
		if lo.Valid && hi.Valid && lo.Int64 > hi.Int64 {
			add("age_max_months", form.Msg(c, "validation.gte.numeric", "age_max_months", "value", strconv.FormatInt(lo.Int64, 10)))
		}
		return nil
	})
	if err != nil {
		return err
	}
	in := ServiceInput{Kind: httpadmin.String(data, "kind"), Name: strings.TrimSpace(httpadmin.String(data, "name")),
		Description: nullable(data, "description"), Format: nullable(data, "format"),
		DurationMinutes: nullInt64(data, "duration_minutes"), AgeMinMonths: nullInt64(data, "age_min_months"),
		AgeMaxMonths: nullInt64(data, "age_max_months"), Capacity: nullInt64(data, "capacity"),
		PriceToman: nullInt64(data, "price_toman"), PackageSessions: nullInt64(data, "package_sessions"),
		PackageValidityDays: nullInt64(data, "package_validity_days"), SortOrder: form.Int32(data, "sort_order", 0),
		IsActive: !form.Has(data, "is_active") || httpadmin.Bool(data, "is_active")}
	id, err = a.store.SaveService(c, cur.ID, id, in, a.now(c).Time)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_service.save", "city_service_service", id, slog.Uint64("listing_id", cur.ID))
	v, err := a.store.AdminService(c, cur.ID, id)
	if err != nil {
		return err
	}
	if update {
		return httpadmin.OK(c, jsonx.Obj("item", a.serviceJSON(v)), "Saved.")
	}
	return httpadmin.Created(c, jsonx.Obj("item", a.serviceJSON(v)), "Created.")
}

// DestroyService is DELETE …/services/:serviceId; 422 in_use once a booking names it (deactivate it instead).
func (a *Admin) DestroyService(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	v, err := a.findService(c, cur.ID)
	if err != nil {
		return err
	}
	switch err := a.store.DeleteService(c, cur.ID, v.ID); {
	case errors.Is(err, ErrInUse):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse, "Bookings name this service. Deactivate it instead.")
	case err != nil:
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_service.delete", "city_service_service", v.ID, slog.Uint64("listing_id", cur.ID))
	return httpadmin.OK(c, jsonx.Obj("id", v.ID), "Deleted.")
}

// ---------------------------------------------------------------------------
// Slots

func slotRowJSON(s SlotRow) *jsonx.OrderedMap {
	o := SlotJSON(s.Slot)
	o.Set("booked_count", s.BookedCount)
	o.Set("status", s.Status)
	return o
}

func parseWall(layout, s string) (time.Time, bool) {
	t, err := time.ParseInLocation(layout, strings.TrimSpace(s), civildate.Tehran)
	return t, err == nil
}

// ListSlots is GET /city-services/listings/:id/slots?from=Y-m-d&to=Y-m-d&service_id= (any status, default the next
// 31 days from today).
func (a *Admin) ListSlots(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	now := httpadmin.Now(c).In(civildate.Tehran)
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, civildate.Tehran)
	if t, ok := parseWall(time.DateOnly, c.Query("from")); ok {
		from = t
	}
	to := from.AddDate(0, 0, MaxSlotDays)
	if t, ok := parseWall(time.DateOnly, c.Query("to")); ok && t.After(from) {
		to = t.AddDate(0, 0, 1) // inclusive day
	}
	service, _ := parseUint(c.Query("service_id"))
	rows, err := a.store.AdminSlots(c, cur.ID, service, from, to, MaxAdminSlots)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, s := range rows {
		items = append(items, slotRowJSON(s))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items, "from", from.Format(time.DateOnly), "to", to.AddDate(0, 0, -1).Format(time.DateOnly)))
}

// slotService validates the optional service_id of a slot body against the place.
func (a *Admin) slotService(c fiber.Ctx, listingID uint64, in phpval.Map, add form.Add) (ServiceRow, error) {
	if !form.Has(in, "service_id") {
		return ServiceRow{}, nil
	}
	id, ok := parseUint(httpadmin.String(in, "service_id"))
	if !ok {
		add("service_id", form.Msg(c, "validation.exists", "service_id"))
		return ServiceRow{}, nil
	}
	v, err := a.store.AdminService(c, listingID, id)
	if errors.Is(err, ErrNotFound) {
		add("service_id", form.Msg(c, "validation.exists", "service_id"))
		return v, nil
	}
	return v, err
}

func serviceArg(v ServiceRow) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(v.ID), Valid: v.ID > 0} //nolint:gosec // G115: ids are positive
}

// StoreSlot is POST /city-services/listings/:id/slots {service_id?, starts_at "Y-m-d H:i", ends_at?, capacity}.
func (a *Admin) StoreSlot(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	var svc ServiceRow
	F := validation.F
	data, err := form.Validate(c, validation.Rules{
		F("service_id", "nullable", "integer"),
		F("starts_at", "required", "date_format:Y-m-d H:i"),
		F("ends_at", "nullable", "date_format:Y-m-d H:i"),
		F("capacity", "required", "integer", "min:1", "max:"+strconv.Itoa(MaxSlotCapacity)),
	}, func(in phpval.Map, add form.Add) error {
		var err error
		svc, err = a.slotService(c, cur.ID, in, add)
		if err != nil {
			return err
		}
		s, ok1 := parseWall("2006-01-02 15:04", httpadmin.String(in, "starts_at"))
		e, ok2 := parseWall("2006-01-02 15:04", httpadmin.String(in, "ends_at"))
		if ok1 && ok2 && !e.After(s) {
			add("ends_at", form.Msg(c, "validation.after", "ends_at", "date", "starts_at"))
		}
		return nil
	})
	if err != nil {
		return err
	}
	starts, _ := parseWall("2006-01-02 15:04", httpadmin.String(data, "starts_at"))
	sl := NewSlot{StartsAt: starts}
	if e, ok := parseWall("2006-01-02 15:04", httpadmin.String(data, "ends_at")); ok {
		sl.EndsAt = sql.NullTime{Time: e, Valid: true}
	} else if svc.DurationMinutes.Valid {
		sl.EndsAt = sql.NullTime{Time: starts.Add(time.Duration(svc.DurationMinutes.Int64) * time.Minute), Valid: true}
	}
	n, err := a.store.CreateSlots(c, cur.ID, serviceArg(svc), []NewSlot{sl}, int(form.Int(data, "capacity", 1)), a.now(c).Time)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_slot.create", "city_service_listing", cur.ID, slog.Int("created", n))
	return httpadmin.Created(c, jsonx.Obj("created", n), "Created.")
}

// GenerateSlots is POST /city-services/listings/:id/slots/generate — the weekly generator: every chosen weekday
// (0 = Saturday) between from and to (≤ MaxGenerateDays) at each time, capacity seats, duration from the body or the
// service. Existing slots at the same start are skipped; at most MaxGenerated per run.
func (a *Admin) GenerateSlots(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	var svc ServiceRow
	F := validation.F
	data, err := form.Validate(c, validation.Rules{
		F("service_id", "nullable", "integer"),
		F("from", "required", "date_format:Y-m-d"),
		F("to", "required", "date_format:Y-m-d"),
		F("weekdays", "required", "list", "min:1", "max:7"),
		F("weekdays.*", "required", "integer", "min:0", "max:6"),
		F("times", "required", "list", "min:1", "max:"+strconv.Itoa(MaxTimesPerDay)),
		F("times.*", "required", "date_format:H:i"),
		F("duration_minutes", "nullable", "integer", "min:5", "max:600"),
		F("capacity", "required", "integer", "min:1", "max:"+strconv.Itoa(MaxSlotCapacity)),
	}, func(in phpval.Map, add form.Add) error {
		var err error
		svc, err = a.slotService(c, cur.ID, in, add)
		if err != nil {
			return err
		}
		f, ok1 := parseWall(time.DateOnly, httpadmin.String(in, "from"))
		t, ok2 := parseWall(time.DateOnly, httpadmin.String(in, "to"))
		if ok1 && ok2 && (t.Before(f) || t.Sub(f) > MaxGenerateDays*24*time.Hour) {
			add("to", form.Msg(c, "validation.between.numeric", "to", "min", "0", "max", strconv.Itoa(MaxGenerateDays)))
		}
		return nil
	})
	if err != nil {
		return err
	}
	from, _ := parseWall(time.DateOnly, httpadmin.String(data, "from"))
	to, _ := parseWall(time.DateOnly, httpadmin.String(data, "to"))
	days := strList(data, "weekdays")
	times := strList(data, "times")
	dur := time.Duration(form.Int(data, "duration_minutes", svc.DurationMinutes.Int64)) * time.Minute
	now := httpadmin.Now(c)
	var slots []NewSlot
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if !slices.Contains(days, strconv.Itoa(JalaliWeekday(d))) {
			continue
		}
		for _, hm := range times {
			at, ok := parseWall("2006-01-02 15:04", d.Format(time.DateOnly)+" "+hm)
			if !ok || !at.After(now) {
				continue
			}
			sl := NewSlot{StartsAt: at}
			if dur > 0 {
				sl.EndsAt = sql.NullTime{Time: at.Add(dur), Valid: true}
			}
			slots = append(slots, sl)
		}
	}
	if len(slots) > MaxGenerated {
		return httpadmin.FieldError("to", form.Msg(c, "validation.max.array", "to", "max", strconv.Itoa(MaxGenerated)))
	}
	n, err := a.store.CreateSlots(c, cur.ID, serviceArg(svc), slots, int(form.Int(data, "capacity", 1)), a.now(c).Time)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_slot.generate", "city_service_listing", cur.ID, slog.Int("created", n))
	return httpadmin.Created(c, jsonx.Obj("created", n, "skipped", len(slots)-n), "Created.")
}

func (a *Admin) findSlot(c fiber.Ctx, listingID uint64) (SlotRow, error) {
	id, ok := httpadmin.ID(c, "slotId")
	if !ok {
		return SlotRow{}, httpadmin.NotFound("Slot")
	}
	s, err := a.store.AdminSlot(c, listingID, id)
	if errors.Is(err, ErrNotFound) {
		return s, httpadmin.NotFound("Slot")
	}
	return s, err
}

// UpdateSlot is PUT …/slots/:slotId {capacity, status open|closed}; 422 capacity_below_booked under the booked seats.
func (a *Admin) UpdateSlot(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	s, err := a.findSlot(c, cur.ID)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("capacity", "required", "integer", "min:1", "max:"+strconv.Itoa(MaxSlotCapacity)),
		validation.F("status", "required", validation.In(SlotStatuses...)),
	})
	if err != nil {
		return err
	}
	capacity := int(form.Int(data, "capacity", 1))
	switch err := a.store.UpdateSlot(c, cur.ID, s.ID, capacity, httpadmin.String(data, "status"), a.now(c).Time); {
	case errors.Is(err, ErrBelowBooked):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBelowBooked, "Capacity cannot be lower than the seats already booked.",
			"booked_count", s.BookedCount)
	case errors.Is(err, ErrNotFound):
		return httpadmin.NotFound("Slot")
	case err != nil:
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_slot.update", "city_service_slot", s.ID, slog.Uint64("listing_id", cur.ID))
	s, err = a.store.AdminSlot(c, cur.ID, s.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("item", slotRowJSON(s)), "Saved.")
}

// DestroySlot is DELETE …/slots/:slotId; 422 in_use while booked (close it instead).
func (a *Admin) DestroySlot(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	s, err := a.findSlot(c, cur.ID)
	if err != nil {
		return err
	}
	switch err := a.store.DeleteSlot(c, cur.ID, s.ID); {
	case errors.Is(err, ErrInUse):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse, "This slot has bookings. Close it instead.")
	case err != nil:
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_slot.delete", "city_service_slot", s.ID, slog.Uint64("listing_id", cur.ID))
	return httpadmin.OK(c, jsonx.Obj("id", s.ID), "Deleted.")
}

// ---------------------------------------------------------------------------
// Reviews and reports

func reviewJSON(r AdminReview) *jsonx.OrderedMap {
	return jsonx.Obj("id", r.ID, "listing", jsonx.Obj("id", r.ListingID, "name", r.ListingName), "booking_id", r.BookingID,
		"rating", r.Rating, "score_cleanliness", nullInt(r.Cleanliness), "score_instructor", nullInt(r.Instructor),
		"score_amenities", nullInt(r.Amenities), "body", httpadmin.NullString(r.Body), "status", r.Status,
		"author", httpadmin.NullString(r.Author), "created_at", httpadmin.Time(r.CreatedAt))
}

// ListReviews is GET /city-services/reviews?status=&listing_id=&page= (pending first).
func (a *Admin) ListReviews(c fiber.Ctx) error {
	status := c.Query("status")
	if !slices.Contains(ReviewStatuses, status) {
		status = ""
	}
	listing, _ := parseUint(c.Query("listing_id"))
	p := httpadmin.PageOf(c)
	rows, total, err := a.store.AdminReviews(c, status, listing, p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, reviewJSON(r))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, total))
}

// ModerateReview is PUT /city-services/reviews/:id {status published|hidden}; the place's rating follows.
func (a *Admin) ModerateReview(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Review")
	}
	r, err := a.store.AdminReviewByID(c, id)
	if errors.Is(err, ErrNotFound) {
		return httpadmin.NotFound("Review")
	} else if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{validation.F("status", "required", validation.In(ReviewPublished, ReviewHidden))})
	if err != nil {
		return err
	}
	to := httpadmin.String(data, "status")
	if to == r.Status {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	}
	if err := a.store.ModerateReview(c, r, to, a.now(c).Time); errors.Is(err, ErrMoved) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	} else if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_review.moderate", "city_service_review", r.ID, slog.String("to", to))
	r, err = a.store.AdminReviewByID(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("item", reviewJSON(r)), "Saved.")
}

func reportJSON(r AdminReport) *jsonx.OrderedMap {
	return jsonx.Obj("id", r.ID, "listing", jsonx.Obj("id", r.ListingID, "name", r.ListingName), "reason", r.Reason,
		"note", httpadmin.NullString(r.Note), "status", r.Status, "resolved_at", httpadmin.Time(r.ResolvedAt),
		"created_at", httpadmin.Time(r.CreatedAt))
}

// ListReports is GET /city-services/reports?status=&page= (open first). The reporter is never shown.
func (a *Admin) ListReports(c fiber.Ctx) error {
	status := c.Query("status")
	if !slices.Contains(ReportStatuses, status) {
		status = ""
	}
	p := httpadmin.PageOf(c)
	rows, total, err := a.store.AdminReports(c, status, p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, reportJSON(r))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, total))
}

// CloseReport is PUT /city-services/reports/:id {status resolved|dismissed} (only an open report).
func (a *Admin) CloseReport(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Report")
	}
	if _, err := a.store.AdminReportByID(c, id); errors.Is(err, ErrNotFound) {
		return httpadmin.NotFound("Report")
	} else if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{validation.F("status", "required", validation.In(ReportResolved, ReportDismissed))})
	if err != nil {
		return err
	}
	to := httpadmin.String(data, "status")
	if err := a.store.CloseReport(c, id, to, a.now(c).Time); errors.Is(err, ErrMoved) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	} else if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_report.close", "city_service_report", id, slog.String("to", to))
	r, err := a.store.AdminReportByID(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("item", reportJSON(r)), "Saved.")
}

// ---------------------------------------------------------------------------
// Applications

func (a *Admin) jq() *joinstore.Queries { return joinstore.New(a.store.db) }

// seesContacts reports whether the admin may read an applicant's contact name / phone (capability support).
func seesContacts(c fiber.Ctx) bool {
	return httpadmin.Can(c, httpadmin.CapSupport, httpadmin.LevelView)
}

func strOrNil(s sql.NullString) any {
	if !s.Valid || s.String == "" {
		return nil
	}
	return s.String
}

func (a *Admin) applicationJSON(c fiber.Ctx, app join.Application) *jsonx.OrderedMap {
	r, d := app.Row, app.Draft
	cats := d.CategorySlugs
	if cats == nil {
		cats = []string{}
	}
	var city, listing any
	if d.CityID > 0 {
		city = d.CityID
	}
	if r.ListingID.Valid {
		listing = r.ListingID.Int64
	}
	o := jsonx.Obj("id", r.ID, "tracking_code", strOrNil(r.TrackingCode), "status", r.Status,
		"business_name", strOrNil(r.BusinessName), "category_slugs", cats, "city_id", city, "district", strOrNil(r.District))
	if seesContacts(c) {
		o.Set("contact_name", strOrNil(r.ContactName))
		o.Set("contact_phone", strOrNil(r.ContactPhone))
		o.Set("contact_hidden", false)
	} else {
		o.Set("contact_name", nil)
		o.Set("contact_phone", nil)
		o.Set("contact_hidden", true)
	}
	o.Set("photos_count", len(d.PhotoIDs))
	o.Set("booking_mode", strOrNil(r.BookingMode))
	o.Set("submitted_at", httpadmin.Time(r.SubmittedAt))
	o.Set("reviewed_at", httpadmin.Time(r.ReviewedAt))
	o.Set("review_note", strOrNil(r.ReviewNote))
	o.Set("listing_id", listing)
	o.Set("allowed_moves", join.ReviewMoves(r.Status))
	o.Set("can_approve", join.CanApprove(r.Status))
	return o
}

// ListApplications is GET /city-service-applications?status=&q=&page= (open work first; drafts never listed).
func (a *Admin) ListApplications(c fiber.Ctx) error {
	status := c.Query("status")
	if !slices.Contains([]string{join.StatusSubmitted, join.StatusInReview, join.StatusNeedsInfo, join.StatusApproved, join.StatusRejected}, status) {
		status = ""
	}
	q := strings.TrimSpace(c.Query("q"))
	if len(q) > MaxQueryLen {
		q = q[:MaxQueryLen]
	}
	arg := sql.NullString{String: escapeLike(q), Valid: true}
	p := httpadmin.PageOf(c)
	total, err := a.jq().AdminCountApplications(c, joinstore.AdminCountApplicationsParams{Status: status, Q: arg})
	if err != nil {
		return err
	}
	rows, err := a.jq().AdminListApplications(c, joinstore.AdminListApplicationsParams{Status: status, Q: arg,
		Limit: int32(p.PerPage), Offset: int32(p.Offset())}) //nolint:gosec // G115: PageOf clamps both
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, a.applicationJSON(c, join.Decode(r)))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

func (a *Admin) findApplication(c fiber.Ctx) (join.Application, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return join.Application{}, httpadmin.NotFound("Application")
	}
	r, err := a.jq().AdminGetApplication(c, id)
	if errors.Is(err, sql.ErrNoRows) {
		return join.Application{}, httpadmin.NotFound("Application")
	}
	if err != nil {
		return join.Application{}, err
	}
	return join.Decode(r), nil
}

// ShowApplication is GET /city-service-applications/:id: every step, the photos (public URLs) and which document
// slots hold a file — never a document URL (DocumentLink mints one per view).
func (a *Admin) ShowApplication(c fiber.Ctx) error {
	app, err := a.findApplication(c)
	if err != nil {
		return err
	}
	return a.applicationReply(c, app, "")
}

func (a *Admin) applicationReply(c fiber.Ctx, app join.Application, msg string) error {
	d := app.Draft
	o := a.applicationJSON(c, app)
	paths, err := a.store.PhotoPaths(c, d.PhotoIDs)
	if err != nil {
		return err
	}
	photos := []*jsonx.OrderedMap{}
	for _, fid := range d.PhotoIDs {
		if u := FilePhotoURLs(a.appURL, []uint64{fid}, paths); len(u) == 1 {
			photos = append(photos, jsonx.Obj("id", fid, "url", u[0]))
		}
	}
	docs := []*jsonx.OrderedMap{}
	slots := join.Slots
	if !seesContacts(c) { // without support the private documents are not even listed (DocumentLink answers 403)
		slots = nil
	}
	for _, slot := range slots {
		var mime, size any
		present := false
		if fid := d.Documents[slot]; fid > 0 && a.files != nil {
			if f, err := a.files.Get(c, app.Row.UserID, fid); err == nil && f.Purpose == files.PurposePlaceLicence {
				present, mime, size = true, f.MIME, f.Size
			} else if err != nil && !errors.Is(err, files.ErrNotFound) {
				return err
			}
		}
		docs = append(docs, jsonx.Obj("slot", slot, "present", present, "mime", mime, "size", size))
	}
	services := d.Services
	if services == nil {
		services = []join.Offer{}
	}
	wh := d.WeeklyHours
	if wh == nil {
		wh = []join.Hours{}
	}
	o.Set("owner_confirmed", d.OwnerConfirmed)
	o.Set("lat", strOrNil(app.Row.Lat))
	o.Set("lng", strOrNil(app.Row.Lng))
	o.Set("address", strOrNil(app.Row.Address))
	o.Set("about", strOrNil(app.Row.About))
	o.Set("age_ranges", list(d.AgeRanges))
	o.Set("amenities", list(d.Amenities))
	o.Set("weekly_hours", wh)
	o.Set("services", services)
	o.Set("photos", photos)
	o.Set("documents", docs)
	o.Set("terms_accepted_at", httpadmin.Time(app.Row.TermsAcceptedAt))
	if msg == "" {
		return httpadmin.OK(c, jsonx.Obj("application", o))
	}
	return httpadmin.OK(c, jsonx.Obj("application", o), msg)
}

// DocumentLink is GET /city-service-applications/:id/documents/:slot: a fresh 5-minute signed link to a private
// licence (each view is audited; the link names no path). Needs the support capability (403 otherwise).
func (a *Admin) DocumentLink(c fiber.Ctx) error {
	if !seesContacts(c) { // private applicant documents are personal data, like the contact name / phone
		return httpadmin.ForbiddenPermission(httpadmin.CapSupport)
	}
	app, err := a.findApplication(c)
	if err != nil {
		return err
	}
	slot := c.Params("slot")
	fid := app.Draft.Documents[slot]
	if !slices.Contains(join.Slots, slot) || fid == 0 || a.files == nil {
		return httpadmin.NotFound("Document")
	}
	f, err := a.files.Get(c, app.Row.UserID, fid)
	if errors.Is(err, files.ErrNotFound) || (err == nil && f.Purpose != files.PurposePlaceLicence) {
		return httpadmin.NotFound("Document")
	}
	if err != nil {
		return err
	}
	link, err := a.files.Link(f, httpadmin.Now(c), 0)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_application.document_view", "city_service_application", app.Row.ID,
		slog.String("slot", slot))
	var exp any
	if link.ExpiresAt != nil {
		exp = link.ExpiresAt.In(civildate.Tehran).Format("2006-01-02T15:04:05-07:00")
	}
	return httpadmin.OK(c, jsonx.Obj("url", link.URL, "expires_at", exp, "mime", f.MIME))
}

// ReviewApplication is POST /city-service-applications/:id/review {status in_review|needs_info|rejected, note}; a
// note is required for needs_info / rejected (the applicant reads it). Rejecting deletes the application's photos
// (public files of a business that will not be listed); the private documents stay for the record.
func (a *Admin) ReviewApplication(c fiber.Ctx) error {
	app, err := a.findApplication(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("status", "required", validation.In(join.StatusInReview, join.StatusNeedsInfo, join.StatusRejected)),
		validation.F("note", "nullable", "string", "max:500"),
	}, func(in phpval.Map, add form.Add) error {
		if join.NoteRequired(httpadmin.String(in, "status")) && strings.TrimSpace(httpadmin.String(in, "note")) == "" {
			add("note", form.Msg(c, "validation.required", "note"))
		}
		return nil
	})
	if err != nil {
		return err
	}
	to := httpadmin.String(data, "status")
	if !join.CanReview(app.Row.Status, to) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	}
	done, err := a.store.ReviewApplication(c, app.Row.ID, to, nullable(data, "note"), a.now(c).Time)
	switch {
	case errors.Is(err, ErrMoved):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	case errors.Is(err, ErrNotFound):
		return httpadmin.NotFound("Application")
	case err != nil:
		return err
	}
	a.dropApplicationPhotos(c, app.Row.ID, done)
	httpadmin.Audit(c, a.logger, "city_service_application.review", "city_service_application", app.Row.ID, slog.String("to", to))
	app, err = a.findApplication(c)
	if err != nil {
		return err
	}
	return a.applicationReply(c, app, "Saved.")
}

// dropApplicationPhotos deletes the photo files a rejection detached (after the commit; best effort: a failure is
// logged by id — the row no longer points at them either way).
func (a *Admin) dropApplicationPhotos(c fiber.Ctx, appID uint64, done ReviewedApplication) {
	if len(done.PhotoIDs) == 0 {
		return
	}
	if a.files == nil {
		a.logger.WarnContext(c.Context(), "cityservices: no file service, rejected photos kept", slog.Uint64("application_id", appID))
		return
	}
	for _, fid := range done.PhotoIDs {
		if err := a.files.Delete(c, done.UserID, fid); err != nil && !errors.Is(err, files.ErrNotFound) {
			a.logger.WarnContext(c.Context(), "cityservices: photo delete failed", slog.Uint64("application_id", appID),
				slog.Uint64("file_id", fid))
		}
	}
}

// ApproveApplication is POST /city-service-applications/:id/approve {category_id?, phone?, note?}: publishes the
// application as a verified listing (new, or the one it already points at) with its place, photos, hours and services.
// category_id defaults to the first category the applicant chose; phone (the venue's public number) is required in
// phone mode and is never copied from the applicant's contact phone.
func (a *Admin) ApproveApplication(c fiber.Ctx) error {
	app, err := a.findApplication(c)
	if err != nil {
		return err
	}
	if !join.CanApprove(app.Row.Status) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	}
	var category uint64
	data, err := form.Validate(c, validation.Rules{
		validation.F("category_id", "nullable", "integer"),
		validation.F("phone", "nullable", "string", "max:32"),
		validation.F("note", "nullable", "string", "max:500"),
	}, func(in phpval.Map, add form.Add) error {
		if form.Has(in, "category_id") {
			n, ok := parseUint(httpadmin.String(in, "category_id"))
			exists := false
			if ok {
				var err error
				if exists, err = a.store.TaxonExists(c, TableCategories, n); err != nil {
					return err
				}
			}
			if !exists {
				add("category_id", form.Msg(c, "validation.exists", "category_id"))
			}
			category = n
		} else {
			for _, slug := range app.Draft.CategorySlugs {
				id, err := a.store.CategoryIDBySlug(c, slug)
				if err != nil {
					return err
				}
				if id > 0 {
					category = id
					break
				}
			}
			if category == 0 {
				add("category_id", form.Msg(c, "validation.required", "category_id"))
			}
		}
		if app.Draft.CityID == 0 {
			add("city_id", form.Msg(c, "validation.required", "city_id"))
		} else if ok, err := a.store.TaxonExists(c, TableCities, app.Draft.CityID); err != nil {
			return err
		} else if !ok {
			add("city_id", form.Msg(c, "validation.exists", "city_id"))
		}
		if strings.TrimSpace(app.Row.BusinessName.String) == "" {
			add("business_name", form.Msg(c, "validation.required", "business_name"))
		}
		if app.Row.BookingMode.String == ModePhone && strings.TrimSpace(httpadmin.String(in, "phone")) == "" {
			add("phone", form.Msg(c, "validation.required", "phone"))
		}
		return nil
	})
	if err != nil {
		return err
	}
	listingID, err := a.store.ApproveApplication(c, app.Row.ID, Approval{CategoryID: category, Phone: nullable(data, "phone"),
		Note: nullable(data, "note")}, a.now(c).Time)
	switch {
	case errors.Is(err, ErrMoved):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	case errors.Is(err, ErrNotFound):
		return httpadmin.NotFound("Application")
	case err != nil:
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_application.approve", "city_service_application", app.Row.ID,
		slog.Uint64("listing_id", listingID))
	app, err = a.findApplication(c)
	if err != nil {
		return err
	}
	return a.applicationReply(c, app, "Application approved.")
}

// directoryOptions appends the CB-DIR-05 enums and catalog chips to GET /city-services/options.
func (a *Admin) directoryOptions(c fiber.Ctx, o *jsonx.OrderedMap) error {
	chips := func(group string) ([]*jsonx.OrderedMap, error) {
		rows, err := a.store.CatalogOptions(c, group)
		if err != nil {
			return nil, err
		}
		out := make([]*jsonx.OrderedMap, 0, len(rows))
		for _, r := range rows {
			out = append(out, jsonx.Obj("code", r.Code, "name", form.Raw(r.Title)))
		}
		return out, nil
	}
	ages, err := chips(catalogAgeRange)
	if err != nil {
		return err
	}
	amen, err := chips(catalogAmenity)
	if err != nil {
		return err
	}
	o.Set("admin_booking_modes", AdminModes)
	o.Set("listing_statuses", ListingStatuses)
	o.Set("review_statuses", ReviewStatuses)
	o.Set("report_statuses", ReportStatuses)
	o.Set("report_reasons", ReportReasons)
	o.Set("slot_statuses", SlotStatuses)
	o.Set("service_kinds", join.ServiceKinds)
	o.Set("service_formats", join.ServiceFormats)
	o.Set("application_statuses", []string{join.StatusSubmitted, join.StatusInReview, join.StatusNeedsInfo, join.StatusApproved, join.StatusRejected})
	o.Set("document_slots", join.Slots)
	o.Set("age_ranges", ages)
	o.Set("amenities", amen)
	return nil
}
