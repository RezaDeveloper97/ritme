package cityservices

// CB-DIR-05 admin store: a place's directory fields, services and slots, review moderation (with the rating
// recompute), the wrong-information report queue and publishing an approved application as a listing. Plain SQL like
// bloom's adminstore.go; every child row (service, slot) is read and written WITH its listing id, so an id of another
// place is a 404, never a cross-place write.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/cityservices/join"
	joinstore "github.com/ritme/backend-go/internal/cityservices/join/store"
)

// Admin errors of CB-DIR-05.
var (
	ErrBelowBooked = errors.New("cityservices: capacity below booked seats")
	ErrMoved       = errors.New("cityservices: row changed meanwhile")
)

// Report statuses (the user side creates `open`).
const (
	ReportOpen      = "open"
	ReportResolved  = "resolved"
	ReportDismissed = "dismissed"
)

// Slot statuses.
const (
	SlotOpen   = "open"
	SlotClosed = "closed"
)

// Value lists of the admin forms.
var (
	ListingStatuses = []string{ListingDraft, ListingPending, ListingPublished, ListingSuspended}
	ReviewStatuses  = []string{ReviewPending, ReviewPublished, ReviewHidden}
	ReportStatuses  = []string{ReportOpen, ReportResolved, ReportDismissed}
	SlotStatuses    = []string{SlotOpen, SlotClosed}
	// AdminModes are the booking modes the team may set: bloom's three plus CB-DIR-01's `online`.
	AdminModes = []string{ModeLink, ModeRequest, ModePhone, ModeOnline}
)

// DirInput is the writable directory part of a listing (PUT …/listings/{id}/directory).
type DirInput struct {
	Status          string
	Verified        bool
	Lat, Lng        sql.NullFloat64
	About           sql.NullString
	AgeRanges       []string
	Amenities       []string
	WeeklyHours     []DayHours
	WebsiteURL      sql.NullString
	CancelFreeHours sql.NullInt64
	CancelPolicy    sql.NullString
	Rules           sql.NullString
}

func jsonList[T any](v []T) sql.NullString {
	if len(v) == 0 {
		return sql.NullString{}
	}
	b, _ := json.Marshal(v) // plain slices of strings / structs of strings and ints
	return sql.NullString{String: string(b), Valid: true}
}

// SaveListingDir writes the directory columns; verified_at keeps its first time while verified stays on.
func (s *Store) SaveListingDir(ctx context.Context, id uint64, in DirInput, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE city_service_listings SET status = ?,
		verified_at = IF(?, COALESCE(verified_at, ?), NULL), lat = ?, lng = ?, about = ?, age_ranges = ?, amenities = ?,
		weekly_hours = ?, website_url = ?, cancel_free_hours = ?, cancel_policy = ?, rules = ?, updated_at = ? WHERE id = ?`,
		in.Status, in.Verified, now, in.Lat, in.Lng, in.About, jsonList(in.AgeRanges), jsonList(in.Amenities),
		jsonList(in.WeeklyHours), in.WebsiteURL, in.CancelFreeHours, in.CancelPolicy, in.Rules, now, id)
	return err
}

// SetFilePhotos replaces a listing's photo_file_ids (the files themselves stay with their owner).
func (s *Store) SetFilePhotos(ctx context.Context, id uint64, ids []uint64, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE city_service_listings SET photo_file_ids = ?, updated_at = ? WHERE id = ?",
		jsonList(ids), now, id)
	return err
}

// CatalogOption is a dir_age_ranges / dir_amenities chip for the admin forms (raw translations).
type CatalogOption struct {
	Code  string
	Title json.RawMessage
}

// CatalogOptions lists a catalog group's active items.
func (s *Store) CatalogOptions(ctx context.Context, group string) ([]CatalogOption, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT code, title FROM catalog_items WHERE `group` = ? AND is_active = 1 ORDER BY sort_order, id", group)
	if err != nil {
		return nil, fmt.Errorf("cityservices: catalog: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []CatalogOption{}
	for rows.Next() {
		var o CatalogOption
		var t sql.NullString
		if err := rows.Scan(&o.Code, &t); err != nil {
			return nil, err
		}
		o.Title = json.RawMessage(t.String)
		if !t.Valid || t.String == "" {
			o.Title = json.RawMessage("{}")
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Services

// ServiceRow is a service of any state with its usage.
type ServiceRow struct {
	Service
	SortOrder int
	IsActive  bool
	Slots     int // upcoming-or-past slot rows
	Bookings  int // bookings that name it
}

// ServiceInput is the writable part of a service.
type ServiceInput struct {
	Kind, Name                                            string
	Description, Format                                   sql.NullString
	DurationMinutes, AgeMinMonths, AgeMaxMonths, Capacity sql.NullInt64
	PriceToman, PackageSessions, PackageValidityDays      sql.NullInt64
	SortOrder                                             int32
	IsActive                                              bool
}

const serviceRowSelect = `SELECT v.id, v.kind, v.name, v.description, v.format, v.duration_minutes, v.age_min_months,
	v.age_max_months, v.capacity, v.price_toman, v.package_sessions, v.package_validity_days, v.sort_order, v.is_active,
	(SELECT COUNT(*) FROM city_service_slots s WHERE s.service_id = v.id),
	(SELECT COUNT(*) FROM city_service_bookings b WHERE b.service_id = v.id)
	FROM city_service_services v`

func scanServiceRow(sc scanner, r *ServiceRow) error {
	return sc.Scan(&r.ID, &r.Kind, &r.Name, &r.Description, &r.Format, &r.DurationMinutes, &r.AgeMinMonths,
		&r.AgeMaxMonths, &r.Capacity, &r.PriceToman, &r.PackageSessions, &r.PackageValidityDays, &r.SortOrder, &r.IsActive,
		&r.Slots, &r.Bookings)
}

// AdminServices lists every service of a listing in display order.
func (s *Store) AdminServices(ctx context.Context, listingID uint64) ([]ServiceRow, error) {
	rows, err := s.db.QueryContext(ctx, serviceRowSelect+" WHERE v.listing_id = ? ORDER BY v.sort_order, v.id", listingID)
	if err != nil {
		return nil, fmt.Errorf("cityservices: services: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []ServiceRow{}
	for rows.Next() {
		var r ServiceRow
		if err := scanServiceRow(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AdminService is one service of listingID (ErrNotFound for another place's).
func (s *Store) AdminService(ctx context.Context, listingID, id uint64) (ServiceRow, error) {
	var r ServiceRow
	err := scanServiceRow(s.db.QueryRowContext(ctx, serviceRowSelect+" WHERE v.id = ? AND v.listing_id = ?", id, listingID), &r)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertService(ctx context.Context, db execer, listingID uint64, in ServiceInput, now time.Time) (uint64, error) {
	res, err := db.ExecContext(ctx, `INSERT INTO city_service_services (listing_id, kind, name, description, format,
		duration_minutes, age_min_months, age_max_months, capacity, price_toman, package_sessions, package_validity_days,
		sort_order, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		listingID, in.Kind, in.Name, in.Description, in.Format, in.DurationMinutes, in.AgeMinMonths, in.AgeMaxMonths,
		in.Capacity, in.PriceToman, in.PackageSessions, in.PackageValidityDays, in.SortOrder, in.IsActive, now, now)
	if err != nil {
		return 0, fmt.Errorf("cityservices: insert service: %w", err)
	}
	n, err := res.LastInsertId()
	return uint64(n), err //nolint:gosec // G115: auto-increment ids are positive
}

// SaveService inserts (id 0) or updates a service of listingID and returns its id.
func (s *Store) SaveService(ctx context.Context, listingID, id uint64, in ServiceInput, now time.Time) (uint64, error) {
	if id == 0 {
		return insertService(ctx, s.db, listingID, in, now)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE city_service_services SET kind = ?, name = ?, description = ?, format = ?,
		duration_minutes = ?, age_min_months = ?, age_max_months = ?, capacity = ?, price_toman = ?, package_sessions = ?,
		package_validity_days = ?, sort_order = ?, is_active = ?, updated_at = ? WHERE id = ? AND listing_id = ?`,
		in.Kind, in.Name, in.Description, in.Format, in.DurationMinutes, in.AgeMinMonths, in.AgeMaxMonths, in.Capacity,
		in.PriceToman, in.PackageSessions, in.PackageValidityDays, in.SortOrder, in.IsActive, now, id, listingID)
	return id, err
}

// DeleteService removes a service and its slots; ErrInUse once a booking names it (deactivate it instead).
func (s *Store) DeleteService(ctx context.Context, listingID, id uint64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM city_service_services WHERE id = ? AND listing_id = ?
		AND NOT EXISTS (SELECT 1 FROM city_service_bookings b WHERE b.service_id = ?)`, id, listingID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrInUse
	}
	return nil
}

// ---------------------------------------------------------------------------
// Slots

// SlotRow is a slot of any status.
type SlotRow struct {
	Slot
	Status string
}

const slotRowSelect = `SELECT id, service_id, starts_at, ends_at, capacity, booked_count, status FROM city_service_slots`

func scanSlotRow(sc scanner, r *SlotRow) error {
	return sc.Scan(&r.ID, &r.ServiceID, &r.StartsAt, &r.EndsAt, &r.Capacity, &r.BookedCount, &r.Status)
}

// AdminSlots lists a listing's slots in [from, to), earliest first; serviceID 0 = all.
func (s *Store) AdminSlots(ctx context.Context, listingID, serviceID uint64, from, to time.Time, limit int) ([]SlotRow, error) {
	q, args := slotRowSelect+" WHERE listing_id = ? AND starts_at >= ? AND starts_at < ?", []any{listingID, from, to}
	if serviceID > 0 {
		q += " AND service_id = ?"
		args = append(args, serviceID)
	}
	rows, err := s.db.QueryContext(ctx, q+" ORDER BY starts_at, id LIMIT ?", append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("cityservices: admin slots: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []SlotRow{}
	for rows.Next() {
		var r SlotRow
		if err := scanSlotRow(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AdminSlot is one slot of listingID (ErrNotFound for another place's).
func (s *Store) AdminSlot(ctx context.Context, listingID, id uint64) (SlotRow, error) {
	var r SlotRow
	err := scanSlotRow(s.db.QueryRowContext(ctx, slotRowSelect+" WHERE id = ? AND listing_id = ?", id, listingID), &r)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// NewSlot is one slot to create (Tehran wall clock).
type NewSlot struct {
	StartsAt time.Time
	EndsAt   sql.NullTime
}

// CreateSlots inserts slots of listingID (optionally of one service) with capacity; a slot that already exists at the
// same start for the same service is skipped. Returns how many were created.
func (s *Store) CreateSlots(ctx context.Context, listingID uint64, serviceID sql.NullInt64, slots []NewSlot, capacity int, now time.Time) (int, error) {
	created := 0
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		for _, sl := range slots {
			res, err := tx.ExecContext(ctx, `INSERT INTO city_service_slots (listing_id, service_id, starts_at, ends_at, capacity,
				booked_count, status, created_at, updated_at)
				SELECT ?, ?, ?, ?, ?, 0, 'open', ?, ? FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM city_service_slots
				WHERE listing_id = ? AND starts_at = ? AND service_id <=> ?)`,
				listingID, serviceID, sl.StartsAt, sl.EndsAt, capacity, now, now, listingID, sl.StartsAt, serviceID)
			if err != nil {
				return fmt.Errorf("cityservices: create slot: %w", err)
			}
			n, _ := res.RowsAffected()
			created += int(n)
		}
		return nil
	})
	return created, err
}

// UpdateSlot sets a slot's capacity and status. The capacity guard is in the UPDATE itself (booked_count <= the new
// capacity), so it cannot race a booking that takes a seat; ErrBelowBooked when it would cut a booked seat.
func (s *Store) UpdateSlot(ctx context.Context, listingID, id uint64, capacity int, status string, now time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE city_service_slots SET capacity = ?, status = ?, updated_at = ?
		WHERE id = ? AND listing_id = ? AND booked_count <= ?`, capacity, status, now, id, listingID, capacity)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := s.AdminSlot(ctx, listingID, id); err != nil {
			return err
		}
		return ErrBelowBooked
	}
	return nil
}

// DeleteSlot removes an unbooked slot; ErrInUse while it holds a seat or a booking names it (close it instead).
func (s *Store) DeleteSlot(ctx context.Context, listingID, id uint64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM city_service_slots WHERE id = ? AND listing_id = ? AND booked_count = 0
		AND NOT EXISTS (SELECT 1 FROM city_service_bookings b WHERE b.slot_id = ?)`, id, listingID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrInUse
	}
	return nil
}

// ---------------------------------------------------------------------------
// Reviews

// AdminReview is a review of any status with its place (the author as a first name only).
type AdminReview struct {
	ID, ListingID, BookingID           uint64
	ListingName                        string
	Rating                             int
	Cleanliness, Instructor, Amenities sql.NullInt64
	Body                               sql.NullString
	Status                             string
	Author                             sql.NullString
	CreatedAt                          sql.NullTime
}

const adminReviewSelect = `SELECT r.id, r.listing_id, r.booking_id, l.name, r.rating, r.score_cleanliness,
	r.score_instructor, r.score_amenities, r.body, r.status, u.name, r.created_at
	FROM city_service_reviews r JOIN city_service_listings l ON l.id = r.listing_id LEFT JOIN users u ON u.id = r.user_id`

func scanAdminReview(sc scanner, r *AdminReview) error {
	err := sc.Scan(&r.ID, &r.ListingID, &r.BookingID, &r.ListingName, &r.Rating, &r.Cleanliness, &r.Instructor,
		&r.Amenities, &r.Body, &r.Status, &r.Author, &r.CreatedAt)
	r.Author = firstName(r.Author)
	return err
}

// AdminReviews is one page of reviews (pending first, oldest first; then newest) and the total.
func (s *Store) AdminReviews(ctx context.Context, status string, listingID uint64, limit, offset int) ([]AdminReview, int, error) {
	where, args := " WHERE 1=1", []any{}
	if status != "" {
		where += " AND r.status = ?"
		args = append(args, status)
	}
	if listingID > 0 {
		where += " AND r.listing_id = ?"
		args = append(args, listingID)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM city_service_reviews r"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, adminReviewSelect+where+` ORDER BY (r.status = 'pending') DESC,
		CASE WHEN r.status = 'pending' THEN r.created_at END ASC, r.created_at DESC, r.id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("cityservices: admin reviews: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []AdminReview{}
	for rows.Next() {
		var r AdminReview
		if err := scanAdminReview(rows, &r); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// AdminReviewByID is one review.
func (s *Store) AdminReviewByID(ctx context.Context, id uint64) (AdminReview, error) {
	var r AdminReview
	err := scanAdminReview(s.db.QueryRowContext(ctx, adminReviewSelect+" WHERE r.id = ?", id), &r)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// recomputeRating refreshes a listing's rating_avg / rating_count from its published reviews.
func recomputeRating(ctx context.Context, db execer, listingID uint64) error {
	_, err := db.ExecContext(ctx, `UPDATE city_service_listings SET
		rating_avg = (SELECT ROUND(AVG(r.rating), 2) FROM city_service_reviews r WHERE r.listing_id = ? AND r.status = 'published'),
		rating_count = (SELECT COUNT(*) FROM city_service_reviews r WHERE r.listing_id = ? AND r.status = 'published')
		WHERE id = ?`, listingID, listingID, listingID)
	if err != nil {
		return fmt.Errorf("cityservices: rating: %w", err)
	}
	return nil
}

// ModerateReview moves a review from `from` to `to` (compare-and-set) and recomputes its place's rating in the same
// transaction; ErrMoved when someone else moved it first.
func (s *Store) ModerateReview(ctx context.Context, r AdminReview, to string, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE city_service_reviews SET status = ?, updated_at = ? WHERE id = ? AND status = ?`,
			to, now, r.ID, r.Status)
		if err != nil {
			return fmt.Errorf("cityservices: moderate: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrMoved
		}
		return recomputeRating(ctx, tx, r.ListingID)
	})
}

// ---------------------------------------------------------------------------
// Reports

// AdminReport is a wrong-information report with its place (the reporter is never shown).
type AdminReport struct {
	ID, ListingID         uint64
	ListingName           string
	Reason                string
	Note                  sql.NullString
	Status                string
	ResolvedAt, CreatedAt sql.NullTime
}

const adminReportSelect = `SELECT p.id, p.listing_id, l.name, p.reason, p.note, p.status, p.resolved_at, p.created_at
	FROM city_service_reports p JOIN city_service_listings l ON l.id = p.listing_id`

func scanAdminReport(sc scanner, r *AdminReport) error {
	return sc.Scan(&r.ID, &r.ListingID, &r.ListingName, &r.Reason, &r.Note, &r.Status, &r.ResolvedAt, &r.CreatedAt)
}

// AdminReports is one page of reports (open first, oldest first; then newest) and the total.
func (s *Store) AdminReports(ctx context.Context, status string, limit, offset int) ([]AdminReport, int, error) {
	where, args := " WHERE 1=1", []any{}
	if status != "" {
		where += " AND p.status = ?"
		args = append(args, status)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM city_service_reports p"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, adminReportSelect+where+` ORDER BY (p.status = 'open') DESC,
		CASE WHEN p.status = 'open' THEN p.created_at END ASC, p.created_at DESC, p.id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("cityservices: admin reports: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []AdminReport{}
	for rows.Next() {
		var r AdminReport
		if err := scanAdminReport(rows, &r); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// AdminReportByID is one report.
func (s *Store) AdminReportByID(ctx context.Context, id uint64) (AdminReport, error) {
	var r AdminReport
	err := scanAdminReport(s.db.QueryRowContext(ctx, adminReportSelect+" WHERE p.id = ?", id), &r)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// CloseReport resolves or dismisses an open report; ErrMoved when it is no longer open.
func (s *Store) CloseReport(ctx context.Context, id uint64, to string, now time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE city_service_reports SET status = ?, resolved_at = ?, updated_at = ?
		WHERE id = ? AND status = 'open'`, to, now, now, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrMoved
	}
	return nil
}

// ---------------------------------------------------------------------------
// Applications → listing

// Approval is what the team decides on top of the application when publishing it.
type Approval struct {
	CategoryID uint64
	Phone      sql.NullString // the venue's public phone (never copied from the applicant's contact phone)
	Note       sql.NullString
}

func offerInput(o join.Offer, i int) ServiceInput {
	n := func(p *int64) sql.NullInt64 {
		if p == nil {
			return sql.NullInt64{}
		}
		return sql.NullInt64{Int64: *p, Valid: true}
	}
	sp := func(p *string) sql.NullString {
		if p == nil || *p == "" {
			return sql.NullString{}
		}
		return sql.NullString{String: *p, Valid: true}
	}
	kind := o.Kind
	if kind == "" {
		kind = "session"
	}
	return ServiceInput{Kind: kind, Name: o.Name, Description: sp(o.Description), Format: sp(o.Format),
		DurationMinutes: n(o.DurationMinutes), AgeMinMonths: n(o.AgeMinMonths), AgeMaxMonths: n(o.AgeMaxMonths),
		Capacity: n(o.Capacity), PriceToman: n(o.PriceToman), PackageSessions: n(o.PackageSessions),
		PackageValidityDays: n(o.PackageValidityDays), SortOrder: int32(i), IsActive: true} //nolint:gosec // G115: ≤ join.MaxServices
}

// ApproveApplication publishes a submitted / in-review application as a listing in one transaction: the row is locked,
// a new published + verified listing is created (or the one it already points at is updated, its old services
// deactivated) with the place, photos, weekly hours and services of the application, and the application moves to
// approved. ErrMoved when it is no longer approvable. Returns the listing id.
func (s *Store) ApproveApplication(ctx context.Context, id uint64, in Approval, now time.Time) (uint64, error) {
	var listingID uint64
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		q := joinstore.New(tx)
		row, err := q.AdminLockApplication(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("cityservices: lock application: %w", err)
		}
		if !join.CanApprove(row.Status) {
			return ErrMoved
		}
		a := join.Decode(row)
		mode := row.BookingMode.String
		if mode == "" {
			mode = ModeRequest
		}
		args := []any{in.CategoryID, a.Draft.CityID, row.BusinessName.String, row.District, row.Address, in.Phone, mode,
			row.Lat, row.Lng, row.About, row.AgeRanges, row.Amenities, row.WeeklyHours, jsonList(a.Draft.PhotoIDs), now}
		if row.ListingID.Valid {
			listingID = uint64(row.ListingID.Int64) //nolint:gosec // G115: ids are positive
			if _, err := tx.ExecContext(ctx, `UPDATE city_service_listings SET category_id = ?, city_id = ?, name = ?,
				district = ?, address = ?, phone = ?, booking_mode = ?, booking_url = NULL, lat = ?, lng = ?, about = ?,
				age_ranges = ?, amenities = ?, weekly_hours = ?, photo_file_ids = ?, status = 'published', is_active = 1,
				verified_at = COALESCE(verified_at, ?), owner_user_id = ?, updated_at = ? WHERE id = ?`,
				append(args, row.UserID, now, listingID)...); err != nil {
				return fmt.Errorf("cityservices: update listing: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE city_service_services SET is_active = 0, updated_at = ? WHERE listing_id = ?`,
				now, listingID); err != nil {
				return err
			}
		} else {
			res, err := tx.ExecContext(ctx, `INSERT INTO city_service_listings (category_id, city_id, name, district, address,
				phone, booking_mode, lat, lng, about, age_ranges, amenities, weekly_hours, photo_file_ids, verified_at,
				owner_user_id, status, is_active, sort_order, rating_count, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', 1, 0, 0, ?, ?)`,
				append(args, row.UserID, now, now)...)
			if err != nil {
				return fmt.Errorf("cityservices: insert listing: %w", err)
			}
			n, err := res.LastInsertId()
			if err != nil {
				return err
			}
			listingID = uint64(n) //nolint:gosec // G115: auto-increment ids are positive
		}
		for i, o := range a.Draft.Services {
			if _, err := insertService(ctx, tx, listingID, offerInput(o, i), now); err != nil {
				return err
			}
		}
		n, err := q.AdminApproveApplication(ctx, joinstore.AdminApproveApplicationParams{
			ListingID:  sql.NullInt64{Int64: int64(listingID), Valid: true}, //nolint:gosec // G115: ids are positive
			ReviewNote: in.Note, ReviewedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true},
			ID: id, Status: row.Status})
		if err != nil {
			return fmt.Errorf("cityservices: approve: %w", err)
		}
		if n == 0 {
			return ErrMoved
		}
		return nil
	})
	return listingID, err
}

// CategoryIDBySlug resolves a category slug (0 when unknown).
func (s *Store) CategoryIDBySlug(ctx context.Context, slug string) (uint64, error) {
	var id uint64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM city_service_categories WHERE slug = ?", slug).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// ReviewedApplication is what ReviewApplication changed: the owner and, on a rejection, the photo file ids it
// detached (read from the locked row, so a concurrent edit cannot slip a photo past the delete).
type ReviewedApplication struct {
	UserID   uint64
	PhotoIDs []uint64
}

// ReviewApplication moves an application to in_review / needs_info / rejected in one transaction: the row is locked,
// the move re-checked against its current status (ErrMoved when no longer allowed) and, on a rejection, the photo ids
// cleared from the row. An in_review move without a note keeps the current note. The caller deletes the returned
// photo files after the commit.
func (s *Store) ReviewApplication(ctx context.Context, id uint64, to string, note sql.NullString, now time.Time) (ReviewedApplication, error) {
	var out ReviewedApplication
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		q := joinstore.New(tx)
		row, err := q.AdminLockApplication(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("cityservices: lock application: %w", err)
		}
		if !join.CanReview(row.Status, to) {
			return ErrMoved
		}
		if to == join.StatusInReview && !note.Valid {
			note = row.ReviewNote
		}
		at := sql.NullTime{Time: now, Valid: true}
		n, err := q.AdminReviewApplication(ctx, joinstore.AdminReviewApplicationParams{Status: to, ReviewNote: note,
			ReviewedAt: at, UpdatedAt: at, ID: id, Status_2: row.Status})
		if err != nil {
			return fmt.Errorf("cityservices: review: %w", err)
		}
		if n == 0 {
			return ErrMoved
		}
		out.UserID = row.UserID
		if to == join.StatusRejected {
			if out.PhotoIDs = join.Decode(row).Draft.PhotoIDs; len(out.PhotoIDs) > 0 {
				if err := q.AdminClearApplicationPhotos(ctx, joinstore.AdminClearApplicationPhotosParams{UpdatedAt: at, ID: id}); err != nil {
					return fmt.Errorf("cityservices: clear photos: %w", err)
				}
			}
		}
		return nil
	})
	return out, err
}
