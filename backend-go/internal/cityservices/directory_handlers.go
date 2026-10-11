package cityservices

// CB-DIR-02 handlers: bloom's options / listings / listing routes become public (browse without login, web SEO),
// grow the directory params and fields, and gain GET /city-services/listings/{id}/slots. Responses carry a short
// Cache-Control (the routes are also IP-throttled, routes_city_services.go).

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// PublicMaxAge is the Cache-Control max-age of the public directory reads (seconds).
const PublicMaxAge = 60

// WithCatalog wires the catalog reader for the amenity / age-range options and the age fit (nil = none).
func (h *Handlers) WithCatalog(r *catalog.Reader) *Handlers {
	h.catalog = r
	return h
}

func (h *Handlers) catalogItems(c fiber.Ctx, group string) ([]catalog.Item, error) {
	if h.catalog == nil {
		return nil, nil
	}
	return h.catalog.Items(c, group)
}

// cacheable marks a public read: shared caches may keep an anonymous answer, a signed-in one stays private.
func cacheable(c fiber.Ctx) {
	scope := "public"
	if len(c.Get(fiber.HeaderAuthorization)) > 0 {
		scope = "private"
	}
	c.Set(fiber.HeaderCacheControl, scope+", max-age="+strconv.Itoa(PublicMaxAge))
	c.Set(fiber.HeaderVary, "Accept-Language, Authorization")
}

func (h *Handlers) localizer(c fiber.Ctx) catalog.Localizer {
	langs := i18n.LanguagesOf(c)
	return catalog.Localizer{Locale: i18n.ResolveLocale(c, ""), Default: langs.DefaultCode(), Langs: langs}
}

// Options is GET /city-services/options → {cities, categories, age_ranges, max_advance_days} (bloom) + CB-DIR-02's
// {amenities, place_age_ranges, sorts}. Public. Cities and categories are the active ones with at least one visible
// (active + published) listing — CB-DIR-02 hides empty categories too.
func (h *Handlers) Options(c fiber.Ctx) error {
	cities, cats, err := h.store.ActiveTaxa(c)
	if err != nil {
		return fmt.Errorf("cityservices: options: %w", err)
	}
	lb := h.labeler(c)
	cj, kj := make([]*jsonx.OrderedMap, 0, len(cities)), make([]*jsonx.OrderedMap, 0, len(cats))
	for _, t := range cities {
		cj = append(cj, lb.TaxonJSON(t, false))
	}
	for _, t := range cats {
		kj = append(kj, lb.TaxonJSON(t, true))
	}
	loc := h.localizer(c)
	opts := func(group string) ([]*jsonx.OrderedMap, error) {
		items, err := h.catalogItems(c, group)
		if err != nil {
			return nil, fmt.Errorf("cityservices: catalog %s: %w", group, err)
		}
		out := make([]*jsonx.OrderedMap, 0, len(items))
		for _, it := range items {
			out = append(out, CatalogOptionJSON(loc, it))
		}
		return out, nil
	}
	amen, err := opts(catalogAmenity)
	if err != nil {
		return err
	}
	ages, err := opts(catalogAgeRange)
	if err != nil {
		return err
	}
	cacheable(c)
	return httpx.OK(c, jsonx.Obj("cities", cj, "categories", kj, "age_ranges", AgeRanges, "max_advance_days", MaxAdvanceDays,
		"amenities", amen, "place_age_ranges", ages, "sorts", Sorts))
}

// parseFloat reads an optional float query param within [lo, hi].
func parseFloat(s string, lo, hi float64) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f, err == nil && f >= lo && f <= hi
}

// parsePoint reads lat + lng (both or neither).
func parsePoint(c fiber.Ctx) (*Point, string) {
	la, ln := strings.TrimSpace(c.Query("lat")), strings.TrimSpace(c.Query("lng"))
	if la == "" && ln == "" {
		return nil, ""
	}
	lat, ok := parseFloat(la, -90, 90)
	if !ok {
		return nil, "lat"
	}
	lng, ok := parseFloat(ln, -180, 180)
	if !ok {
		return nil, "lng"
	}
	return &Point{Lat: lat, Lng: lng}, ""
}

// ParseBBox reads "min_lng,min_lat,max_lng,max_lat" (GeoJSON order).
func ParseBBox(s string) (*BBox, bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return nil, false
	}
	var v [4]float64
	for i, p := range parts {
		lim := 180.0
		if i%2 == 1 {
			lim = 90
		}
		f, ok := parseFloat(p, -lim, lim)
		if !ok {
			return nil, false
		}
		v[i] = f
	}
	if v[0] > v[2] || v[1] > v[3] {
		return nil, false
	}
	return &BBox{MinLng: v[0], MinLat: v[1], MaxLng: v[2], MaxLat: v[3]}, true
}

// parseSearch reads the list query; the second value is the invalid field ("" = valid).
func (h *Handlers) parseSearch(c fiber.Ctx) (Search, string, error) {
	q := Search{ListFilter: ListFilter{Category: strings.TrimSpace(c.Query("category"))}, AgeMonths: -1, Sort: SortRank}
	if s := strings.TrimSpace(c.Query("city")); s != "" {
		id, ok := parseID(s)
		if !ok {
			return q, "city", nil
		}
		q.CityID = id
	}
	q.Q = strings.Join(strings.Fields(c.Query("q")), " ")
	if utf8.RuneCountInString(q.Q) > MaxQueryLen {
		return q, "q", nil
	}
	if s := strings.TrimSpace(c.Query("age_months")); s != "" {
		n, convErr := strconv.Atoi(s)
		if convErr != nil || n < 0 || n > MaxAgeMonths {
			return q, "age_months", nil //nolint:nilerr // a bad param is a field error, not a server error
		}
		q.AgeMonths = n
	}
	switch strings.TrimSpace(c.Query("open_now")) {
	case "", "0", "false":
	case "1", "true":
		q.OpenNow = true
	default:
		return q, "open_now", nil
	}
	if s := strings.TrimSpace(c.Query("amenities")); s != "" {
		items, err := h.catalogItems(c, catalogAmenity)
		if err != nil {
			return q, "", err
		}
		known := map[string]bool{}
		for _, it := range items {
			known[it.Code] = true
		}
		for _, a := range strings.Split(s, ",") {
			if a = strings.TrimSpace(a); a == "" {
				continue
			}
			if !known[a] || len(q.Amenities) >= MaxAmenities {
				return q, "amenities", nil
			}
			q.Amenities = append(q.Amenities, a)
		}
	}
	at, bad := parsePoint(c)
	if bad != "" {
		return q, bad, nil
	}
	q.At = at
	if s := strings.TrimSpace(c.Query("radius_km")); s != "" {
		r, ok := parseFloat(s, 0.1, MaxRadiusKm)
		if !ok || at == nil {
			return q, "radius_km", nil
		}
		q.RadiusKm = r
	}
	if s := strings.TrimSpace(c.Query("bbox")); s != "" {
		b, ok := ParseBBox(s)
		if !ok {
			return q, "bbox", nil
		}
		q.BBox = b
	}
	if s := strings.TrimSpace(c.Query("sort")); s != "" {
		ok := false
		for _, v := range Sorts {
			ok = ok || v == s
		}
		if !ok {
			return q, "sort", nil
		}
		q.Sort = s
	}
	return q, "", nil
}

func (h *Handlers) ageRanges(c fiber.Ctx) (map[string]AgeRange, error) {
	items, err := h.catalogItems(c, catalogAgeRange)
	if err != nil {
		return nil, fmt.Errorf("cityservices: age ranges: %w", err)
	}
	return AgeRangesFrom(items), nil
}

// resolvePhotos fills FilePhotoURLs of the given listings.
func (h *Handlers) resolvePhotos(c fiber.Ctx, ls []*Listing) error {
	var ids []uint64
	for _, l := range ls {
		ids = append(ids, l.PhotoFileIDs...)
	}
	paths, err := h.store.PhotoPaths(c, ids)
	if err != nil {
		return err
	}
	for _, l := range ls {
		l.FilePhotoURLs = FilePhotoURLs(h.appURL, l.PhotoFileIDs, paths)
	}
	return nil
}

// Listings is GET /city-services/listings: bloom's ?city=<id>&category=<slug>&page=&per_page= plus CB-DIR-02's
// q, age_months, open_now, amenities (comma list), lat + lng [+ radius_km], bbox (min_lng,min_lat,max_lng,max_lat)
// and sort (rank | distance | rating). Public. Ranking never uses money (directory.go).
func (h *Handlers) Listings(c fiber.Ctx) error {
	q, bad, err := h.parseSearch(c)
	if err != nil {
		return err
	}
	if bad != "" {
		return fieldFail(c, bad, msg("validation_failed", i18n.Locale(c)))
	}
	page, per := 1, DefaultPerPage
	if n, err := strconv.Atoi(c.Query("page")); err == nil && n >= 1 && n <= 100000 {
		page = n
	}
	if n, err := strconv.Atoi(c.Query("per_page")); err == nil && n >= 1 {
		per = min(n, MaxPerPage)
	}
	cands, err := h.store.SearchCandidates(c, q)
	if err != nil {
		return fmt.Errorf("cityservices: listings: %w", err)
	}
	var ranges map[string]AgeRange
	serviceFit := map[uint64]bool{}
	if q.AgeMonths >= 0 {
		if ranges, err = h.ageRanges(c); err != nil {
			return err
		}
		ids := make([]uint64, 0, len(cands))
		for _, l := range cands {
			ids = append(ids, l.ID)
		}
		if serviceFit, err = h.store.ServiceAgeFit(c, ids, q.AgeMonths); err != nil {
			return err
		}
	}
	now := h.now(c)
	hits := Rank(cands, q, ranges, serviceFit, now)
	total := len(hits)
	from := min((page-1)*per, total)
	pageHits := hits[from:min(from+per, total)]
	ids, ptrs := make([]uint64, 0, len(pageHits)), make([]*Listing, 0, len(pageHits))
	for i := range pageHits {
		ids = append(ids, pageHits[i].ID)
		ptrs = append(ptrs, &pageHits[i].Listing)
	}
	if err := h.resolvePhotos(c, ptrs); err != nil {
		return err
	}
	ex, err := h.store.Extras(c, ids, now)
	if err != nil {
		return err
	}
	lb := h.labeler(c)
	items := make([]*jsonx.OrderedMap, 0, len(pageHits))
	for _, r := range pageHits {
		items = append(items, lb.CardJSON(r.Listing, InfoOf(r, q.AgeMonths >= 0, ex)))
	}
	sortBy := q.Sort
	if sortBy == SortDistance && q.At == nil {
		sortBy = SortRank
	}
	pg := httpx.NewPage(items, total, per, page, "")
	cacheable(c)
	return httpx.OK(c, jsonx.Obj("listings", items, "meta", pg.Meta(), "sort", sortBy))
}

// Listing is GET /city-services/listings/{id}[?lat=&lng=]: the place page. Public; 404 unless published + active.
func (h *Handlers) Listing(c fiber.Ctx) error {
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c)
	}
	at, bad := parsePoint(c)
	if bad != "" {
		return fieldFail(c, bad, msg("validation_failed", i18n.Locale(c)))
	}
	l, err := h.store.PublicListing(c, id)
	if errors.Is(err, ErrNotFound) {
		return notFound(c)
	}
	if err != nil {
		return fmt.Errorf("cityservices: listing: %w", err)
	}
	now := h.now(c)
	if err := h.resolvePhotos(c, []*Listing{&l}); err != nil {
		return err
	}
	hits := Rank([]Listing{l}, Search{AgeMonths: -1, At: at}, nil, nil, now)
	ex, err := h.store.Extras(c, []uint64{id}, now)
	if err != nil {
		return err
	}
	var d PlaceDetail
	if d.Services, err = h.store.Services(c, id); err != nil {
		return err
	}
	if d.NextSlots, err = h.store.Slots(c, id, 0, now, now.AddDate(0, 0, MaxSlotDays), now, NextSlotsShown); err != nil {
		return err
	}
	if d.Reviews, err = h.store.Reviews(c, id); err != nil {
		return err
	}
	cacheable(c)
	return httpx.OK(c, jsonx.Obj("listing", h.labeler(c).DetailJSON(l, InfoOf(hits[0], false, ex), d)))
}

// Slots is GET /city-services/listings/{id}/slots?service_id=&from=Y-m-d&days=N: the place's open times (full ones
// too, flagged is_full) from `from` (default today) for `days` days (default 14, max 31). Public.
func (h *Handlers) Slots(c fiber.Ctx) error {
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c)
	}
	locale, now := i18n.Locale(c), h.now(c)
	from := civildate.FromTime(now)
	if s := strings.TrimSpace(c.Query("from")); s != "" {
		d, err := civildate.Parse(s)
		if err != nil || d.DiffDays(from) > MaxAdvanceDays || from.DiffDays(d) > MaxAdvanceDays {
			return fieldFail(c, "from", msg("validation_failed", locale))
		}
		from = d
	}
	days := DefaultSlotDays
	if s := strings.TrimSpace(c.Query("days")); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > MaxSlotDays {
			return fieldFail(c, "days", msg("validation_failed", locale))
		}
		days = n
	}
	var serviceID uint64
	if s := strings.TrimSpace(c.Query("service_id")); s != "" {
		if serviceID, ok = parseID(s); !ok {
			return fieldFail(c, "service_id", msg("validation_failed", locale))
		}
	}
	if _, err := h.store.PublicListing(c, id); errors.Is(err, ErrNotFound) {
		return notFound(c)
	} else if err != nil {
		return fmt.Errorf("cityservices: listing: %w", err)
	}
	if serviceID > 0 {
		active, err := h.store.ActiveService(c, id, serviceID)
		if err != nil {
			return err
		}
		if !active {
			return fieldFail(c, "service_id", msg("not_found", locale))
		}
	}
	start := from.TehranMidnight()
	rows, err := h.store.Slots(c, id, serviceID, start, start.AddDate(0, 0, days), now, MaxSlotsShown)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, s := range rows {
		items = append(items, SlotJSON(s))
	}
	cacheable(c)
	return httpx.OK(c, jsonx.Obj("slots", items, "from", from.String(), "days", days))
}
