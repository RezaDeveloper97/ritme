package cityservices

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInUse is a delete refused because rows still depend on the row.
var ErrInUse = errors.New("cityservices: in use")

// Admin-side reads and writes (plain SQL, like the public half).

// Table names of the two taxonomies (a closed set: never user input).
const (
	TableCities     = "city_service_cities"
	TableCategories = "city_service_categories"
)

func listingFK(table string) string {
	if table == TableCities {
		return "city_id"
	}
	return "category_id"
}

// TaxonInput is the writable part of a city / category.
type TaxonInput struct {
	Slug      string
	Name      json.RawMessage
	Icon      sql.NullString // categories only
	SortOrder int32
	IsActive  bool
}

// AdminTaxa lists every city or category with its listing count.
func (s *Store) AdminTaxa(ctx context.Context, table string) ([]Taxon, error) {
	icon := "NULL"
	if table == TableCategories {
		icon = "t.icon"
	}
	return s.taxa(ctx, fmt.Sprintf(`SELECT t.id, t.slug, t.name, %s, t.sort_order, t.is_active,
		(SELECT COUNT(*) FROM city_service_listings l WHERE l.%s = t.id) FROM %s t ORDER BY t.sort_order, t.id`,
		icon, listingFK(table), table)) //nolint:gosec // G201: table / column are package constants
}

// AdminTaxon is one city / category by id.
func (s *Store) AdminTaxon(ctx context.Context, table string, id uint64) (Taxon, error) {
	icon := "NULL"
	if table == TableCategories {
		icon = "t.icon"
	}
	rows, err := s.taxa(ctx, fmt.Sprintf(`SELECT t.id, t.slug, t.name, %s, t.sort_order, t.is_active,
		(SELECT COUNT(*) FROM city_service_listings l WHERE l.%s = t.id) FROM %s t WHERE t.id = ?`,
		icon, listingFK(table), table), id) //nolint:gosec // G201: constants
	if err != nil {
		return Taxon{}, err
	}
	if len(rows) == 0 {
		return Taxon{}, ErrNotFound
	}
	return rows[0], nil
}

// SlugTaken reports whether another row of table uses slug.
func (s *Store) SlugTaken(ctx context.Context, table, slug string, exceptID uint64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE slug = ? AND id <> ?", slug, exceptID).Scan(&n) //nolint:gosec // G202: constant
	return n > 0, err
}

// SaveTaxon inserts (id 0) or updates a city / category and returns its id.
func (s *Store) SaveTaxon(ctx context.Context, table string, id uint64, in TaxonInput, now time.Time) (uint64, error) {
	if id == 0 {
		var res sql.Result
		var err error
		if table == TableCategories {
			res, err = s.db.ExecContext(ctx, `INSERT INTO city_service_categories (slug, name, icon, sort_order, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)`, in.Slug, string(in.Name), in.Icon, in.SortOrder, in.IsActive, now, now)
		} else {
			res, err = s.db.ExecContext(ctx, `INSERT INTO city_service_cities (slug, name, sort_order, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)`, in.Slug, string(in.Name), in.SortOrder, in.IsActive, now, now)
		}
		if err != nil {
			return 0, err
		}
		n, err := res.LastInsertId()
		return uint64(n), err //nolint:gosec // G115: auto-increment ids are positive
	}
	var err error
	if table == TableCategories {
		_, err = s.db.ExecContext(ctx, `UPDATE city_service_categories SET slug = ?, name = ?, icon = ?, sort_order = ?, is_active = ?, updated_at = ? WHERE id = ?`,
			in.Slug, string(in.Name), in.Icon, in.SortOrder, in.IsActive, now, id)
	} else {
		_, err = s.db.ExecContext(ctx, `UPDATE city_service_cities SET slug = ?, name = ?, sort_order = ?, is_active = ?, updated_at = ? WHERE id = ?`,
			in.Slug, string(in.Name), in.SortOrder, in.IsActive, now, id)
	}
	return id, err
}

// DeleteTaxon removes a city / category; ErrInUse while listings use it.
func (s *Store) DeleteTaxon(ctx context.Context, table string, id uint64) error {
	t, err := s.AdminTaxon(ctx, table, id)
	if err != nil {
		return err
	}
	if t.Listings > 0 {
		return ErrInUse
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM "+table+" WHERE id = ?", id) //nolint:gosec // G202: constant
	return err
}

// TaxonExists reports whether a city / category row exists.
func (s *Store) TaxonExists(ctx context.Context, table string, id uint64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE id = ?", id).Scan(&n) //nolint:gosec // G202: constant
	return n > 0, err
}

// ListingInput is the writable part of a listing (photos are managed by their own endpoints).
type ListingInput struct {
	CategoryID, CityID                                      uint64
	Name                                                    string
	District, Address, Phone, Description, PriceText, Hours sql.NullString
	BookingMode                                             string
	BookingURL                                              sql.NullString
	SortOrder                                               int32
	IsActive                                                bool
}

// AdminListingFilter filters the admin listing list.
type AdminListingFilter struct {
	CityID, CategoryID uint64
	Status             string // all | active | inactive
	Query              string
}

// AdminListings is one page of listings (any state) and the total.
func (s *Store) AdminListings(ctx context.Context, f AdminListingFilter, limit, offset int) ([]Listing, int, error) {
	where, args := " WHERE 1=1", []any{}
	if f.CityID > 0 {
		where += " AND l.city_id = ?"
		args = append(args, f.CityID)
	}
	if f.CategoryID > 0 {
		where += " AND l.category_id = ?"
		args = append(args, f.CategoryID)
	}
	switch f.Status {
	case "active":
		where += " AND l.is_active = 1"
	case "inactive":
		where += " AND l.is_active = 0"
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
		where += " AND (l.name LIKE ? OR l.district LIKE ?)"
		args = append(args, like, like)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM city_service_listings l"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, listingSelect+where+" ORDER BY l.sort_order, l.id LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []Listing{}
	for rows.Next() {
		var l Listing
		if err := scanListing(rows, &l); err != nil {
			return nil, 0, err
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

// AdminListing is one listing in any state.
func (s *Store) AdminListing(ctx context.Context, id uint64) (Listing, error) {
	var l Listing
	err := scanListing(s.db.QueryRowContext(ctx, listingSelect+" WHERE l.id = ?", id), &l)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

// SaveListing inserts (id 0) or updates a listing and returns its id.
func (s *Store) SaveListing(ctx context.Context, id uint64, in ListingInput, now time.Time) (uint64, error) {
	if id == 0 {
		res, err := s.db.ExecContext(ctx, `INSERT INTO city_service_listings (category_id, city_id, name, district, address, phone,
			description, price_text, hours_text, booking_mode, booking_url, sort_order, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, in.CategoryID, in.CityID, in.Name, in.District, in.Address,
			in.Phone, in.Description, in.PriceText, in.Hours, in.BookingMode, in.BookingURL, in.SortOrder, in.IsActive, now, now)
		if err != nil {
			return 0, err
		}
		n, err := res.LastInsertId()
		return uint64(n), err //nolint:gosec // G115: auto-increment ids are positive
	}
	_, err := s.db.ExecContext(ctx, `UPDATE city_service_listings SET category_id = ?, city_id = ?, name = ?, district = ?, address = ?,
		phone = ?, description = ?, price_text = ?, hours_text = ?, booking_mode = ?, booking_url = ?, sort_order = ?, is_active = ?,
		updated_at = ? WHERE id = ?`, in.CategoryID, in.CityID, in.Name, in.District, in.Address, in.Phone, in.Description,
		in.PriceText, in.Hours, in.BookingMode, in.BookingURL, in.SortOrder, in.IsActive, now, id)
	return id, err
}

// SetPhotos replaces a listing's photo path list.
func (s *Store) SetPhotos(ctx context.Context, id uint64, photos []string, now time.Time) error {
	b, err := json.Marshal(photos)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE city_service_listings SET photos = ?, updated_at = ? WHERE id = ?", string(b), now, id)
	return err
}

// ToggleListing flips is_active.
func (s *Store) ToggleListing(ctx context.Context, id uint64, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE city_service_listings SET is_active = 1 - is_active, updated_at = ? WHERE id = ?", now, id)
	return err
}

// DeleteListing removes a listing; ErrInUse while booking requests reference it (deactivate it instead).
func (s *Store) DeleteListing(ctx context.Context, id uint64) error {
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM city_service_bookings WHERE listing_id = ?", id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrInUse
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM city_service_listings WHERE id = ?", id)
	return err
}

// AdminBookingFilter filters the request queue.
type AdminBookingFilter struct {
	Status    string // "" = all
	ListingID uint64
}

// AdminBookings is one page of the queue (oldest open first, then newest) and the total.
func (s *Store) AdminBookings(ctx context.Context, f AdminBookingFilter, limit, offset int) ([]Booking, int, error) {
	where, args := " WHERE 1=1", []any{}
	if f.Status != "" {
		where += " AND b.status = ?"
		args = append(args, f.Status)
	}
	if f.ListingID > 0 {
		where += " AND b.listing_id = ?"
		args = append(args, f.ListingID)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM city_service_bookings b"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.bookings(ctx, where+` ORDER BY (b.status = 'requested') DESC, b.created_at DESC, b.id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	return rows, total, err
}

// AdminBooking is one request.
func (s *Store) AdminBooking(ctx context.Context, id uint64) (Booking, error) {
	var b Booking
	err := scanBooking(s.db.QueryRowContext(ctx, bookingSelect+" WHERE b.id = ?", id), &b)
	if errors.Is(err, sql.ErrNoRows) {
		return b, ErrNotFound
	}
	return b, err
}

// MoveBooking sets a request's status (compare-and-set on `from`) and the note shown to the user.
func (s *Store) MoveBooking(ctx context.Context, id uint64, from, to string, note sql.NullString, now time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx, "UPDATE city_service_bookings SET status = ?, status_note = ?, updated_at = ? WHERE id = ? AND status = ?",
		to, note, now, id, from)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
