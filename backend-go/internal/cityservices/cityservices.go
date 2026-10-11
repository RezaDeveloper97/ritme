// Package cityservices is the city-services directory of bloom B-N10-05 (D-92, docs/night-bloom/extras.md §4):
// classes, pools and playhouses near the user, plus simple booking requests. Go only, no Laravel counterpart.
//
//   - Admin content: cities, categories (class / pool / playhouse, editable) and listings (name, district, address,
//     phone, description, price and hours text, photos on the public disk, booking mode link | request | phone,
//     booking URL). Admin API under /api/admin/v1/city-services/* (capability content) and the request queue under
//     /api/admin/v1/city-service-bookings/* (capability support: the queue holds contact details).
//   - User API /api/v1/city-services/*: options, listing list (city + category filter) and detail, a booking REQUEST
//     for a `request`-mode listing and «رزروها» — the user's own requests with a user-visible status, cancel.
//
// Privacy (extras.md §4): a booking carries contact name + phone, a preferred day, an optional child AGE RANGE and a
// short note — never a health or medical field, never cycle / pregnancy state, never an account id toward a venue.
// BookingNotifier (notifier.go) is the only door to a venue and is typed with exactly those allow-listed fields; its
// `fake` implementation logs ids only. Listings and bookings live in their own tables and join no health table.
//
// Thin on purpose: CB-DIR-01 extends the listing table (map, services, slots, reviews) and CB-DIR-09 replaces the
// simple bookings list; both keep these rows rendering.
package cityservices

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Booking modes of a listing.
const (
	ModeLink    = "link"    // book on the venue's own page (booking_url)
	ModeRequest = "request" // a request lands in the admin queue
	ModePhone   = "phone"   // call the venue
)

// Modes lists the valid booking modes.
var Modes = []string{ModeLink, ModeRequest, ModePhone}

// Booking statuses.
const (
	StatusRequested = "requested"
	StatusConfirmed = "confirmed"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
	StatusCompleted = "completed"
)

// Statuses lists the valid booking statuses.
var Statuses = []string{StatusRequested, StatusConfirmed, StatusRejected, StatusCancelled, StatusCompleted}

// transitions is the status machine (the admin team moves requests; the owner may only cancel).
var transitions = map[string][]string{
	StatusRequested: {StatusConfirmed, StatusRejected, StatusCancelled},
	StatusConfirmed: {StatusCompleted, StatusCancelled},
}

// CanMove reports whether a booking may go from one status to another.
func CanMove(from, to string) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// AgeRanges are the child age ranges a request may carry (the only child detail that is ever sent).
var AgeRanges = []string{"0-1", "1-3", "3-6", "6-12", "12+"}

// Limits.
const (
	MaxPhotos        = 8
	MaxOpenBookings  = 10 // open requests (requested | confirmed, not past) per user
	MaxAdvanceDays   = 365
	PhotoDir         = "city-services"
	DefaultPerPage   = 20
	MaxPerPage       = 50
	maxBookingsShown = 100
)

// Store reads and writes the city-services tables.
type Store struct{ db *sql.DB }

// NewStore wires the store.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// ErrNotFound is a missing (or not visible) row.
var ErrNotFound = errors.New("cityservices: not found")

// ---------------------------------------------------------------------------
// Rows

// Taxon is a city or a category.
type Taxon struct {
	ID        uint64
	Slug      string
	Name      json.RawMessage
	Icon      sql.NullString
	SortOrder int
	IsActive  bool
	Listings  int64
}

// Listing is one venue joined with its city and category.
type Listing struct {
	ID          uint64
	CategoryID  uint64
	CityID      uint64
	Name        string
	District    sql.NullString
	Address     sql.NullString
	Phone       sql.NullString
	Description sql.NullString
	PriceText   sql.NullString
	HoursText   sql.NullString
	Photos      []string
	BookingMode string
	BookingURL  sql.NullString
	SortOrder   int
	IsActive    bool
	CreatedAt   sql.NullTime
	UpdatedAt   sql.NullTime
	CatSlug     string
	CatName     json.RawMessage
	CatIcon     sql.NullString
	CityName    json.RawMessage
	CitySlug    string
	Dir         // CB-DIR-01 directory columns (directory.go), additive
}

// Booking is one booking request joined with its listing.
type Booking struct {
	ID            uint64
	UserID        uint64
	ListingID     uint64
	ContactName   string
	ContactPhone  string
	PreferredDate civildate.Date
	ChildAgeRange sql.NullString
	Note          sql.NullString
	Status        string
	StatusNote    sql.NullString
	CreatedAt     sql.NullTime
	UpdatedAt     sql.NullTime
	Listing       Listing
	BookingDir    // CB-DIR-03: code, service, slot, child, status history, review (booking.go), additive
}

const listingSelect = `SELECT l.id, l.category_id, l.city_id, l.name, l.district, l.address, l.phone, l.description,
	l.price_text, l.hours_text, l.photos, l.booking_mode, l.booking_url, l.sort_order, l.is_active, l.created_at,
	l.updated_at, c.slug, c.name, c.icon, t.name, t.slug` + dirColumns + `
	FROM city_service_listings l
	JOIN city_service_categories c ON c.id = l.category_id
	JOIN city_service_cities t ON t.id = l.city_id`

type scanner interface{ Scan(dest ...any) error }

func scanListing(s scanner, l *Listing) error {
	var photos sql.NullString
	var dir dirScan
	if err := s.Scan(append([]any{&l.ID, &l.CategoryID, &l.CityID, &l.Name, &l.District, &l.Address, &l.Phone, &l.Description,
		&l.PriceText, &l.HoursText, &photos, &l.BookingMode, &l.BookingURL, &l.SortOrder, &l.IsActive, &l.CreatedAt,
		&l.UpdatedAt, &l.CatSlug, &l.CatName, &l.CatIcon, &l.CityName, &l.CitySlug}, l.targets(&dir)...)...); err != nil {
		return err
	}
	l.apply(dir)
	l.Photos = nil
	if photos.Valid && photos.String != "" {
		_ = json.Unmarshal([]byte(photos.String), &l.Photos) // a malformed column reads as no photos
	}
	return nil
}

// ---------------------------------------------------------------------------
// Public reads

// ActiveTaxa returns the active cities and categories that have a visible (active + published, CB-DIR-02) listing.
// CB-DIR-02: a category with no visible listing is hidden too (bloom listed every active category).
//
// CB-DIR-13: allCities (`?scope=join`, the business application) returns every active city instead — the first business
// in a new city would otherwise have no city chip to pick (a city only shows up once it has a published listing).
func (s *Store) ActiveTaxa(ctx context.Context, allCities bool) (cities, categories []Taxon, err error) {
	visible := ` AND EXISTS (SELECT 1 FROM city_service_listings l JOIN city_service_categories c ON c.id = l.category_id
			WHERE l.city_id = t.id AND l.is_active = 1 AND l.status = 'published' AND c.is_active = 1)`
	if allCities {
		visible = ""
	}
	cities, err = s.taxa(ctx, `SELECT t.id, t.slug, t.name, NULL, t.sort_order, t.is_active, 0 FROM city_service_cities t
		WHERE t.is_active = 1`+visible+`
		ORDER BY t.sort_order, t.id`)
	if err != nil {
		return nil, nil, err
	}
	categories, err = s.taxa(ctx, `SELECT c.id, c.slug, c.name, c.icon, c.sort_order, c.is_active, 0 FROM city_service_categories c
		WHERE c.is_active = 1 AND EXISTS (SELECT 1 FROM city_service_listings l JOIN city_service_cities t ON t.id = l.city_id
			WHERE l.category_id = c.id AND l.is_active = 1 AND l.status = 'published' AND t.is_active = 1)
		ORDER BY c.sort_order, c.id`)
	return cities, categories, err
}

func (s *Store) taxa(ctx context.Context, q string, args ...any) ([]Taxon, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("cityservices: taxa: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []Taxon{}
	for rows.Next() {
		var t Taxon
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Icon, &t.SortOrder, &t.IsActive, &t.Listings); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListFilter filters the public listing list.
type ListFilter struct {
	CityID   uint64
	Category string // slug
}

// PublicListing is one active listing (ErrNotFound otherwise, also when its city or category is inactive).
func (s *Store) PublicListing(ctx context.Context, id uint64) (Listing, error) {
	var l Listing
	err := scanListing(s.db.QueryRowContext(ctx, listingSelect+
		publicWhere+" AND l.id = ?", id), &l) // CB-DIR-02: + status = published
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

// HasActive reports whether at least one listing is visible to users (the «برای مادر و کودک» tile switch).
func (s *Store) HasActive(ctx context.Context) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM city_service_listings l
		JOIN city_service_categories c ON c.id = l.category_id JOIN city_service_cities t ON t.id = l.city_id
		WHERE l.is_active = 1 AND l.status = 'published' AND c.is_active = 1 AND t.is_active = 1 LIMIT 1`).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// ---------------------------------------------------------------------------
// Bookings

// NewBooking is a validated request.
type NewBooking struct {
	ListingID     uint64
	ContactName   string
	ContactPhone  string
	PreferredDate civildate.Date
	ChildAgeRange sql.NullString
	Note          sql.NullString
}

const bookingSelect = `SELECT b.id, b.user_id, b.listing_id, b.contact_name, b.contact_phone, b.preferred_date,
	b.child_age_range, b.note, b.status, b.status_note, b.created_at, b.updated_at,
	l.id, l.category_id, l.city_id, l.name, l.district, l.address, l.phone, l.description, l.price_text, l.hours_text,
	l.photos, l.booking_mode, l.booking_url, l.sort_order, l.is_active, l.created_at, l.updated_at,
	c.slug, c.name, c.icon, t.name, t.slug` + bookingDirColumns + `
	FROM city_service_bookings b
	JOIN city_service_listings l ON l.id = b.listing_id
	JOIN city_service_categories c ON c.id = l.category_id
	JOIN city_service_cities t ON t.id = l.city_id` + bookingDirJoins

func scanBooking(s scanner, b *Booking) error {
	var photos sql.NullString
	l := &b.Listing
	if err := s.Scan(append([]any{&b.ID, &b.UserID, &b.ListingID, &b.ContactName, &b.ContactPhone, &b.PreferredDate,
		&b.ChildAgeRange, &b.Note, &b.Status, &b.StatusNote, &b.CreatedAt, &b.UpdatedAt,
		&l.ID, &l.CategoryID, &l.CityID, &l.Name, &l.District, &l.Address, &l.Phone, &l.Description, &l.PriceText,
		&l.HoursText, &photos, &l.BookingMode, &l.BookingURL, &l.SortOrder, &l.IsActive, &l.CreatedAt, &l.UpdatedAt,
		&l.CatSlug, &l.CatName, &l.CatIcon, &l.CityName, &l.CitySlug}, b.targets()...)...); err != nil {
		return err
	}
	if photos.Valid && photos.String != "" {
		_ = json.Unmarshal([]byte(photos.String), &l.Photos)
	}
	return nil
}

// CreateBooking stores a request in status requested and returns it.
func (s *Store) CreateBooking(ctx context.Context, userID uint64, in NewBooking, now time.Time) (Booking, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO city_service_bookings
		(user_id, listing_id, contact_name, contact_phone, preferred_date, child_age_range, note, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'requested', ?, ?)`,
		userID, in.ListingID, in.ContactName, in.ContactPhone, in.PreferredDate, in.ChildAgeRange, in.Note, now, now)
	if err != nil {
		return Booking{}, fmt.Errorf("cityservices: create booking: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Booking{}, err
	}
	return s.UserBooking(ctx, userID, uint64(id)) //nolint:gosec // G115: auto-increment ids are positive
}

// UserBookings lists the user's own bookings, newest first.
func (s *Store) UserBookings(ctx context.Context, userID uint64) ([]Booking, error) {
	return s.bookings(ctx, " WHERE b.user_id = ? ORDER BY b.created_at DESC, b.id DESC LIMIT ?", userID, maxBookingsShown)
}

// UserBooking is one booking of the user (ErrNotFound for anyone else's: owner-only).
func (s *Store) UserBooking(ctx context.Context, userID, id uint64) (Booking, error) {
	var b Booking
	err := scanBooking(s.db.QueryRowContext(ctx, bookingSelect+" WHERE b.id = ? AND b.user_id = ?", id, userID), &b)
	if errors.Is(err, sql.ErrNoRows) {
		return b, ErrNotFound
	}
	return b, err
}

func (s *Store) bookings(ctx context.Context, tail string, args ...any) ([]Booking, error) {
	rows, err := s.db.QueryContext(ctx, bookingSelect+tail, args...)
	if err != nil {
		return nil, fmt.Errorf("cityservices: bookings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []Booking{}
	for rows.Next() {
		var b Booking
		if err := scanBooking(rows, &b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// JSON

// pick returns the translation of a {"fa": …, "en": …} column for locale (default language, then any).
func pick(raw json.RawMessage, locale, def string) string {
	var m map[string]string
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	for _, k := range []string{locale, def} {
		if v := strings.TrimSpace(m[k]); v != "" {
			return v
		}
	}
	for _, v := range m {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func nullStr(s sql.NullString) any {
	if !s.Valid || s.String == "" {
		return nil
	}
	return s.String
}

// Labeler resolves translatable columns and photo URLs for one request.
type Labeler struct {
	Locale  string
	Default string
	AppURL  string
	URL     func(appURL, path string) string
	Now     time.Time // CB-DIR-03: the request time (cancel / reschedule flags); zero = time.Now()
}

func (l Labeler) name(raw json.RawMessage) string { return pick(raw, l.Locale, l.Default) }

func (l Labeler) photos(ps []string) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, l.URL(l.AppURL, p))
	}
	return out
}

// TaxonJSON is a city or category for the user API.
func (l Labeler) TaxonJSON(t Taxon, withIcon bool) *jsonx.OrderedMap {
	o := jsonx.Obj("id", t.ID, "slug", t.Slug, "name", l.name(t.Name))
	if withIcon {
		o.Set("icon", nullStr(t.Icon))
	}
	return o
}

func (l Labeler) place(x Listing) (cat, city *jsonx.OrderedMap) {
	cat = jsonx.Obj("slug", x.CatSlug, "name", l.name(x.CatName), "icon", nullStr(x.CatIcon))
	city = jsonx.Obj("id", x.CityID, "slug", x.CitySlug, "name", l.name(x.CityName))
	return cat, city
}

// CardJSON is a listing in the list. CB-DIR-02 appends the directory fields after bloom's (shape kept): coordinates,
// distance, verified badge, rating, age ranges, amenities, «از … تومان», open-now, next free slot and age fit; the
// photo prefers the structured place photos over bloom's public-disk ones.
func (l Labeler) CardJSON(x Listing, ci CardInfo) *jsonx.OrderedMap {
	cat, city := l.place(x)
	o := jsonx.Obj("id", x.ID, "name", x.Name, "category", cat, "city", city, "district", nullStr(x.District),
		"price_text", nullStr(x.PriceText), "booking_mode", x.BookingMode, "photo_url", firstOrNil(l.listingPhotos(x)))
	l.directoryCard(o, x, ci)
	return o
}

// DetailJSON is the listing detail. booking_url is only present in `link` mode. CB-DIR-02 appends the place page
// (about, weekly hours, rules, cancel policy, services, next slots, reviews summary).
func (l Labeler) DetailJSON(x Listing, ci CardInfo, d PlaceDetail) *jsonx.OrderedMap {
	o := l.CardJSON(x, ci)
	o.Set("address", nullStr(x.Address))
	o.Set("phone", nullStr(x.Phone))
	o.Set("description", nullStr(x.Description))
	o.Set("hours_text", nullStr(x.HoursText))
	o.Set("photos", l.listingPhotos(x))
	var link any
	if x.BookingMode == ModeLink {
		link = nullStr(x.BookingURL)
	}
	o.Set("booking_url", link)
	l.directoryDetail(o, x, d)
	return o
}

// BookingJSON is a booking for its owner. CB-DIR-03 appends the directory fields after bloom's (shape kept) and
// refines can_cancel with the slot and the place's cancel window (booking_json.go).
func (l Labeler) BookingJSON(b Booking) *jsonx.OrderedMap {
	cat, city := l.place(b.Listing)
	now := l.now()
	o := jsonx.Obj(
		"id", b.ID,
		"status", b.Status,
		"status_note", nullStr(b.StatusNote),
		"preferred_date", b.PreferredDate.String(),
		"child_age_range", nullStr(b.ChildAgeRange),
		"note", nullStr(b.Note),
		"contact_name", b.ContactName,
		"contact_phone", b.ContactPhone,
		"can_cancel", b.CancelStateAt(now) == CancelAllowed,
		"created_at", timeJSON(b.CreatedAt),
		"listing", jsonx.Obj("id", b.Listing.ID, "name", b.Listing.Name, "category", cat, "city", city,
			"district", nullStr(b.Listing.District), "address", nullStr(b.Listing.Address),
			"phone", nullStr(b.Listing.Phone), "photo_url", firstOrNil(l.photos(b.Listing.Photos))),
	)
	l.directoryBooking(o, b, now)
	return o
}

func firstOrNil(ps []string) any {
	if len(ps) == 0 {
		return nil
	}
	return ps[0]
}

func timeJSON(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return t.Time.In(civildate.Tehran).Format("2006-01-02T15:04:05-07:00")
}
