package cityservices

import (
	"database/sql"
	"errors"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Photo upload limits (jpeg / png / webp, like banners and the shop).
const (
	photoMaxKB  = 4096
	photoMinDim = 200
	// CodeInUse is the 422 error_code of deleting a row that others still depend on.
	CodeInUse = "in_use"
	// CodeBadTransition is the 422 error_code of an illegal booking status move.
	CodeBadTransition = "invalid_transition"
	// CodeNotStarted is the 422 error_code of completing a booking before its slot (or preferred day) has started.
	CodeNotStarted = "booking_not_started"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Admin serves /api/admin/v1/city-services/* (capability «content») and /api/admin/v1/city-service-bookings/*
// (capability «support»: the queue holds contact details), see httpadmin.capabilityOf.
type Admin struct {
	store  *Store
	disk   *media.Disk
	opt    media.Optimizer
	appURL string
	logger *slog.Logger
	files  AdminFiles     // CB-DIR-05: licence links + rejected photos (nil = not wired)
	status StatusNotifier // CB-DIR-05: discreet confirm / decline notice to the mother
}

// NewAdmin builds the admin handlers.
func NewAdmin(db *sql.DB, disk *media.Disk, appURL string, logger *slog.Logger) *Admin {
	if logger == nil {
		logger = slog.Default()
	}
	if disk == nil {
		disk = media.NewDisk("")
	}
	return &Admin{store: NewStore(db), disk: disk, opt: media.DefaultOptimizer, appURL: appURL, logger: logger,
		status: FakeStatusNotifier{Logger: logger}} // CB-DIR-05
}

// Routes registers the endpoints; static paths before :id.
func (a *Admin) Routes(route Route, kit *httpadmin.Kit) {
	k := kit.Admin
	route(fiber.MethodGet, "/city-services/options", k(a.Options))
	for _, t := range []struct{ path, table string }{{"cities", TableCities}, {"categories", TableCategories}} {
		table := t.table
		route(fiber.MethodGet, "/city-services/"+t.path, k(func(c fiber.Ctx) error { return a.listTaxa(c, table) }))
		route(fiber.MethodPost, "/city-services/"+t.path, k(func(c fiber.Ctx) error { return a.saveTaxon(c, table, false) }))
		route(fiber.MethodPut, "/city-services/"+t.path+"/:id", k(func(c fiber.Ctx) error { return a.saveTaxon(c, table, true) }))
		route(fiber.MethodDelete, "/city-services/"+t.path+"/:id", k(func(c fiber.Ctx) error { return a.deleteTaxon(c, table) }))
	}
	route(fiber.MethodGet, "/city-services/listings", k(a.ListListings))
	route(fiber.MethodPost, "/city-services/listings", k(a.StoreListing))
	route(fiber.MethodGet, "/city-services/listings/:id", k(a.ShowListing))
	route(fiber.MethodPut, "/city-services/listings/:id", k(a.UpdateListing))
	route(fiber.MethodPost, "/city-services/listings/:id/toggle", k(a.ToggleListing))
	route(fiber.MethodDelete, "/city-services/listings/:id", k(a.DestroyListing))
	route(fiber.MethodPost, "/city-services/listings/:id/photos", k(a.AddPhoto))
	route(fiber.MethodDelete, "/city-services/listings/:id/photos/:index", k(a.RemovePhoto))

	route(fiber.MethodGet, "/city-service-bookings", k(a.ListBookings))
	route(fiber.MethodGet, "/city-service-bookings/:id", k(a.ShowBooking))
	route(fiber.MethodPut, "/city-service-bookings/:id", k(a.UpdateBooking))
	a.dirRoutes(route, kit) // CB-DIR-05: directory fields, services, slots, reviews, reports, applications (admin_dir.go)
}

func (a *Admin) now(c fiber.Ctx) sql.NullTime { return httpadmin.DBTime(httpadmin.Now(c)) }

func nullable(data phpval.Map, key string) sql.NullString {
	s := form.Str(data, key)
	if s.Valid {
		s.String = strings.TrimSpace(s.String)
		if s.String == "" {
			return sql.NullString{}
		}
	}
	return s
}

func parseUint(s string) (uint64, bool) {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n, err == nil && n > 0 && len(s) < 20
}

func (a *Admin) taxonJSON(t Taxon, icon bool) *jsonx.OrderedMap {
	o := jsonx.Obj("id", t.ID, "slug", t.Slug, "name", form.Raw(t.Name))
	if icon {
		o.Set("icon", httpadmin.NullString(t.Icon))
	}
	o.Set("sort_order", t.SortOrder)
	o.Set("is_active", t.IsActive)
	o.Set("listings_count", t.Listings)
	return o
}

func (a *Admin) photoURLs(ps []string) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, content.PublicURL(a.appURL, p))
	}
	return out
}

func (a *Admin) listingJSON(l Listing) *jsonx.OrderedMap {
	photos := l.Photos
	if photos == nil {
		photos = []string{}
	}
	o := jsonx.Obj("id", l.ID, "category_id", l.CategoryID, "city_id", l.CityID,
		"category", jsonx.Obj("slug", l.CatSlug, "name", form.Raw(l.CatName)),
		"city", jsonx.Obj("slug", l.CitySlug, "name", form.Raw(l.CityName)),
		"name", l.Name, "district", httpadmin.NullString(l.District), "address", httpadmin.NullString(l.Address),
		"phone", httpadmin.NullString(l.Phone), "description", httpadmin.NullString(l.Description),
		"price_text", httpadmin.NullString(l.PriceText), "hours_text", httpadmin.NullString(l.HoursText),
		"photos", photos, "photo_urls", a.photoURLs(photos),
		"booking_mode", l.BookingMode, "booking_url", httpadmin.NullString(l.BookingURL),
		"sort_order", l.SortOrder, "is_active", l.IsActive,
		"created_at", httpadmin.Time(l.CreatedAt), "updated_at", httpadmin.Time(l.UpdatedAt))
	a.listingDirJSON(o, l) // CB-DIR-05: directory columns appended (shape kept)
	return o
}

// Options is GET /city-services/options: every city and category (any state) plus the enums and limits.
func (a *Admin) Options(c fiber.Ctx) error {
	cities, err := a.store.AdminTaxa(c, TableCities)
	if err != nil {
		return err
	}
	cats, err := a.store.AdminTaxa(c, TableCategories)
	if err != nil {
		return err
	}
	cj, kj := []*jsonx.OrderedMap{}, []*jsonx.OrderedMap{}
	for _, t := range cities {
		cj = append(cj, a.taxonJSON(t, false))
	}
	for _, t := range cats {
		kj = append(kj, a.taxonJSON(t, true))
	}
	o := jsonx.Obj("cities", cj, "categories", kj, "booking_modes", Modes, "statuses", Statuses,
		"max_photos", MaxPhotos, "image", jsonx.Obj("max_kb", photoMaxKB, "min_width", photoMinDim, "min_height", photoMinDim,
			"types", []string{"jpeg", "png", "webp"}))
	if err := a.directoryOptions(c, o); err != nil { // CB-DIR-05
		return err
	}
	return httpadmin.OK(c, o)
}

// ---------------------------------------------------------------------------
// Cities and categories

func (a *Admin) listTaxa(c fiber.Ctx, table string) error {
	rows, err := a.store.AdminTaxa(c, table)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, t := range rows {
		items = append(items, a.taxonJSON(t, table == TableCategories))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

func (a *Admin) saveTaxon(c fiber.Ctx, table string, update bool) error {
	var id uint64
	if update {
		var ok bool
		if id, ok = httpadmin.ID(c, "id"); !ok {
			return httpadmin.NotFound("Item")
		}
		if _, err := a.store.AdminTaxon(c, table, id); errors.Is(err, ErrNotFound) {
			return httpadmin.NotFound("Item")
		} else if err != nil {
			return err
		}
	}
	rules := form.Translatable(c, "name", true, "max:120")
	rules = append(rules, validation.F("slug", "required|string|max:64"), validation.F("icon", "nullable|string|max:32"),
		validation.F("sort_order", "nullable|integer"), validation.F("is_active", "nullable"))
	data, err := form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		slug := strings.TrimSpace(httpadmin.String(in, "slug"))
		if slug == "" {
			return nil
		}
		if !slugRe.MatchString(slug) {
			add("slug", form.Msg(c, "validation.regex", "slug"))
			return nil
		}
		taken, err := a.store.SlugTaken(c, table, slug, id)
		if err != nil {
			return err
		}
		if taken {
			add("slug", form.Msg(c, "validation.unique", "slug"))
		}
		return nil
	})
	if err != nil {
		return err
	}
	in := TaxonInput{Slug: strings.TrimSpace(httpadmin.String(data, "slug")), Name: form.ReqJSON(data, "name"),
		SortOrder: form.Int32(data, "sort_order", 0), IsActive: httpadmin.Bool(data, "is_active")}
	if table == TableCategories {
		in.Icon = nullable(data, "icon")
	}
	id, err = a.store.SaveTaxon(c, table, id, in, a.now(c).Time)
	if err != nil {
		return err
	}
	kind := "city_service_city"
	if table == TableCategories {
		kind = "city_service_category"
	}
	httpadmin.Audit(c, a.logger, kind+".save", kind, id)
	t, err := a.store.AdminTaxon(c, table, id)
	if err != nil {
		return err
	}
	if update {
		return httpadmin.OK(c, jsonx.Obj("item", a.taxonJSON(t, table == TableCategories)), "Saved.")
	}
	return httpadmin.Created(c, jsonx.Obj("item", a.taxonJSON(t, table == TableCategories)), "Created.")
}

func (a *Admin) deleteTaxon(c fiber.Ctx, table string) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Item")
	}
	switch err := a.store.DeleteTaxon(c, table, id); {
	case errors.Is(err, ErrInUse):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse, "Listings still use this item. Move or delete them first.")
	case errors.Is(err, ErrNotFound):
		return httpadmin.NotFound("Item")
	case err != nil:
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_taxon.delete", "city_service_taxon", id)
	return httpadmin.OK(c, jsonx.Obj("id", id), "Deleted.")
}

// ---------------------------------------------------------------------------
// Listings

// ListListings is GET /city-services/listings?city_id=&category_id=&status=&q=&page=&per_page=.
func (a *Admin) ListListings(c fiber.Ctx) error {
	f := AdminListingFilter{Query: c.Query("q"), Status: c.Query("status")}
	if s := c.Query("listing_status"); slices.Contains(ListingStatuses, s) { // CB-DIR-05
		f.ListingStatus = s
	}
	f.CityID, _ = parseUint(c.Query("city_id"))
	f.CategoryID, _ = parseUint(c.Query("category_id"))
	p := httpadmin.PageOf(c)
	rows, total, err := a.store.AdminListings(c, f, p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	if err := a.filePhotos(c, rows); err != nil { // CB-DIR-05
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, l := range rows {
		items = append(items, a.listingJSON(l))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, total))
}

func (a *Admin) findListing(c fiber.Ctx) (Listing, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return Listing{}, httpadmin.NotFound("Listing")
	}
	l, err := a.store.AdminListing(c, id)
	if errors.Is(err, ErrNotFound) {
		return l, httpadmin.NotFound("Listing")
	}
	return l, err
}

// ShowListing is GET /city-services/listings/:id.
func (a *Admin) ShowListing(c fiber.Ctx) error {
	l, err := a.findListing(c)
	if err != nil {
		return err
	}
	ls := []Listing{l}
	if err := a.filePhotos(c, ls); err != nil { // CB-DIR-05
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("listing", a.listingJSON(ls[0])))
}

func (a *Admin) validateListing(c fiber.Ctx) (ListingInput, error) {
	rules := validation.Rules{
		validation.F("name", "required|string|max:160"),
		validation.F("category_id", "required|integer"),
		validation.F("city_id", "required|integer"),
		validation.F("district", "nullable|string|max:120"),
		validation.F("address", "nullable|string|max:300"),
		validation.F("phone", "nullable|string|max:32"),
		validation.F("description", "nullable|string|max:5000"),
		validation.F("price_text", "nullable|string|max:120"),
		validation.F("hours_text", "nullable|string|max:200"),
		validation.F("booking_mode", "required", validation.In(AdminModes...)), // CB-DIR-05: + online
		validation.F("booking_url", "nullable|string|max:1000"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
	}
	data, err := form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		mode := httpadmin.String(in, "booking_mode")
		url := strings.TrimSpace(httpadmin.String(in, "booking_url"))
		if url != "" && !isHTTPURL(url) {
			add("booking_url", form.Msg(c, "validation.url", "booking_url"))
		}
		if mode == ModeLink && url == "" {
			add("booking_url", form.Msg(c, "validation.required", "booking_url"))
		}
		if mode == ModePhone && strings.TrimSpace(httpadmin.String(in, "phone")) == "" {
			add("phone", form.Msg(c, "validation.required", "phone"))
		}
		for field, table := range map[string]string{"category_id": TableCategories, "city_id": TableCities} {
			if v, ok := in.Get(field); ok && v != nil {
				n, ok := parseUint(phpval.ToString(v))
				exists := false
				if ok {
					var err error
					if exists, err = a.store.TaxonExists(c, table, n); err != nil {
						return err
					}
				}
				if !exists {
					add(field, form.Msg(c, "validation.exists", field))
				}
			}
		}
		return nil
	})
	if err != nil {
		return ListingInput{}, err
	}
	cat, _ := parseUint(httpadmin.String(data, "category_id"))
	city, _ := parseUint(httpadmin.String(data, "city_id"))
	in := ListingInput{CategoryID: cat, CityID: city, Name: strings.TrimSpace(httpadmin.String(data, "name")),
		District: nullable(data, "district"), Address: nullable(data, "address"), Phone: nullable(data, "phone"),
		Description: nullable(data, "description"), PriceText: nullable(data, "price_text"), Hours: nullable(data, "hours_text"),
		BookingMode: httpadmin.String(data, "booking_mode"), BookingURL: nullable(data, "booking_url"),
		SortOrder: form.Int32(data, "sort_order", 0), IsActive: httpadmin.Bool(data, "is_active")}
	if in.BookingMode != ModeLink { // a stale link must not survive a mode change
		in.BookingURL = sql.NullString{}
	}
	return in, nil
}

// isHTTPURL accepts only absolute http(s) URLs (never javascript: or data:).
func isHTTPURL(s string) bool {
	l := strings.ToLower(s)
	return (strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "http://")) && form.IsURL(s)
}

// StoreListing is POST /city-services/listings.
func (a *Admin) StoreListing(c fiber.Ctx) error {
	in, err := a.validateListing(c)
	if err != nil {
		return err
	}
	id, err := a.store.SaveListing(c, 0, in, a.now(c).Time)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_listing.create", "city_service_listing", id)
	l, err := a.store.AdminListing(c, id)
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("listing", a.listingJSON(l)), "Listing created.")
}

// UpdateListing is PUT /city-services/listings/:id.
func (a *Admin) UpdateListing(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	in, err := a.validateListing(c)
	if err != nil {
		return err
	}
	if _, err := a.store.SaveListing(c, cur.ID, in, a.now(c).Time); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_listing.update", "city_service_listing", cur.ID)
	l, err := a.store.AdminListing(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("listing", a.listingJSON(l)), "Listing updated.")
}

// ToggleListing is POST /city-services/listings/:id/toggle.
func (a *Admin) ToggleListing(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	if err := a.store.ToggleListing(c, cur.ID, a.now(c).Time); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_listing.toggle", "city_service_listing", cur.ID)
	l, err := a.store.AdminListing(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("listing", a.listingJSON(l)), "Status changed.")
}

// DestroyListing is DELETE /city-services/listings/:id (photos go too); 422 in_use once requests reference it.
func (a *Admin) DestroyListing(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	switch err := a.store.DeleteListing(c, cur.ID); {
	case errors.Is(err, ErrInUse):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse, "This listing has booking requests. Deactivate it instead.")
	case err != nil:
		return err
	}
	for _, p := range cur.Photos {
		a.deleteFile(c, p)
	}
	httpadmin.Audit(c, a.logger, "city_service_listing.delete", "city_service_listing", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Listing deleted.")
}

func (a *Admin) deleteFile(c fiber.Ctx, rel string) {
	if rel == "" {
		return
	}
	if err := a.disk.Delete(rel); err != nil {
		a.logger.WarnContext(c.Context(), "cityservices: could not delete photo", slog.String("path", rel), slog.String("error", err.Error()))
	}
}

// AddPhoto is POST /city-services/listings/:id/photos (multipart `photo`), at most MaxPhotos per listing.
func (a *Admin) AddPhoto(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	var img *media.Image
	if _, err := form.Validate(c, validation.Rules{}, form.ImageRule{Field: "photo", Required: true, MaxKB: photoMaxKB,
		MinWidth: photoMinDim, MinHeight: photoMinDim}.Check(c, &img), func(_ phpval.Map, add form.Add) error {
		if len(cur.Photos) >= MaxPhotos {
			add("photo", form.Msg(c, "validation.max.array", "photo", "max", strconv.Itoa(MaxPhotos)))
		}
		return nil
	}); err != nil {
		return err
	}
	rel, err := a.disk.Store(a.opt, PhotoDir, *img)
	if err != nil {
		return err
	}
	if err := a.store.SetPhotos(c, cur.ID, append(slices.Clone(cur.Photos), rel), a.now(c).Time); err != nil {
		a.deleteFile(c, rel)
		return err
	}
	httpadmin.Audit(c, a.logger, "city_service_listing.photo_add", "city_service_listing", cur.ID)
	l, err := a.store.AdminListing(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("listing", a.listingJSON(l)), "Photo added.")
}

// RemovePhoto is DELETE /city-services/listings/:id/photos/:index (0-based position in the photo list).
func (a *Admin) RemovePhoto(c fiber.Ctx) error {
	cur, err := a.findListing(c)
	if err != nil {
		return err
	}
	i, err := strconv.Atoi(c.Params("index"))
	if err != nil || i < 0 || i >= len(cur.Photos) {
		return httpadmin.NotFound("Photo")
	}
	rel := cur.Photos[i]
	next := slices.Delete(slices.Clone(cur.Photos), i, i+1)
	if err := a.store.SetPhotos(c, cur.ID, next, a.now(c).Time); err != nil {
		return err
	}
	a.deleteFile(c, rel)
	httpadmin.Audit(c, a.logger, "city_service_listing.photo_remove", "city_service_listing", cur.ID)
	l, err := a.store.AdminListing(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("listing", a.listingJSON(l)), "Photo removed.")
}

// ---------------------------------------------------------------------------
// Booking queue (capability support)

func (a *Admin) bookingJSON(b Booking) *jsonx.OrderedMap {
	o := jsonx.Obj("id", b.ID, "status", b.Status, "status_note", httpadmin.NullString(b.StatusNote),
		"contact_name", b.ContactName, "contact_phone", b.ContactPhone, "preferred_date", b.PreferredDate.String(),
		"child_age_range", httpadmin.NullString(b.ChildAgeRange), "note", httpadmin.NullString(b.Note),
		"allowed_statuses", transitions[b.Status],
		"listing", jsonx.Obj("id", b.Listing.ID, "name", b.Listing.Name, "phone", httpadmin.NullString(b.Listing.Phone),
			"category", jsonx.Obj("slug", b.Listing.CatSlug, "name", form.Raw(b.Listing.CatName)),
			"city", jsonx.Obj("slug", b.Listing.CitySlug, "name", form.Raw(b.Listing.CityName))),
		"created_at", httpadmin.Time(b.CreatedAt), "updated_at", httpadmin.Time(b.UpdatedAt))
	a.bookingDirJSON(o, b) // CB-DIR-05: code, service, slot, child, price, status history, review (shape kept)
	return o
}

// ListBookings is GET /city-service-bookings?status=&listing_id=&page=&per_page= (open requests first).
func (a *Admin) ListBookings(c fiber.Ctx) error {
	f := AdminBookingFilter{}
	if s := c.Query("status"); slices.Contains(Statuses, s) {
		f.Status = s
	}
	f.ListingID, _ = parseUint(c.Query("listing_id"))
	p := httpadmin.PageOf(c)
	rows, total, err := a.store.AdminBookings(c, f, p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, b := range rows {
		items = append(items, a.bookingJSON(b))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, total))
}

func (a *Admin) findBooking(c fiber.Ctx) (Booking, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return Booking{}, httpadmin.NotFound("Booking")
	}
	b, err := a.store.AdminBooking(c, id)
	if errors.Is(err, ErrNotFound) {
		return b, httpadmin.NotFound("Booking")
	}
	return b, err
}

// ShowBooking is GET /city-service-bookings/:id.
func (a *Admin) ShowBooking(c fiber.Ctx) error {
	b, err := a.findBooking(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("booking", a.bookingJSON(b)))
}

// UpdateBooking is PUT /city-service-bookings/:id {status, status_note}: the status machine decides what is allowed.
func (a *Admin) UpdateBooking(c fiber.Ctx) error {
	cur, err := a.findBooking(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("status", "required", validation.In(Statuses...)),
		validation.F("status_note", "nullable|string|max:300"),
	})
	if err != nil {
		return err
	}
	to := httpadmin.String(data, "status")
	if !CanMove(cur.Status, to) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	}
	if to == StatusCompleted && !bookingStarted(cur, a.now(c).Time) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeNotStarted, "This booking has not started yet, so it cannot be marked completed.")
	}
	ok, err := a.store.MoveBooking(c, cur.ID, cur.Status, to, nullable(data, "status_note"), a.now(c).Time)
	if err != nil {
		return err
	}
	if !ok { // moved by someone else in between
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeBadTransition, "This status change is not allowed.")
	}
	httpadmin.Audit(c, a.logger, "city_service_booking.status", "city_service_booking", cur.ID, slog.String("to", to))
	a.notifyStatus(c, cur, to) // CB-DIR-05: discreet notice on confirm / decline
	b, err := a.store.AdminBooking(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("booking", a.bookingJSON(b)), "Booking updated.")
}

// bookingStarted reports whether a booking may be completed: a slot booking from the slot's start on, a slotless one
// from its preferred day (Tehran calendar) on.
func bookingStarted(b Booking, now time.Time) bool {
	if b.SlotStartsAt.Valid {
		return !now.Before(b.SlotStartsAt.Time)
	}
	return !civildate.InTehran(now).Before(b.PreferredDate)
}
