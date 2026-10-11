package cityservices

// CB-DIR-03 (mother & child directory, boards nbl_Dir_Book / Booked / MyBookings, docs/canvas-build/directory.md):
// the user side of a directory booking, built ON bloom's B-N10-05 booking routes and `city_service_bookings` rows —
// there is no parallel /directory/bookings family. A booking is still a REQUEST the Ritme team confirms in admin
// (DECISIONS #13): never auto-confirmed, no payment.
//
//   - Create: bloom's body plus optional service_id / slot_id / child_id. A slot seat is held in the same transaction
//     as the insert (`SELECT … FOR UPDATE` on the slot, then booked_count + 1), so two mothers can never take the last
//     seat; a cancelled / rejected / rescheduled booking gives its seat back. child_id must be the caller's own child
//     (bloom `children`), and must fit the service's age bounds on the slot's day.
//   - Upcoming / past («پیش رو» / «گذشته»), one booking, cancel within the place's cancel_free_hours, a reschedule
//     request (back to `requested`), one moderated review per completed booking, a wrong-information report.
//
// Owner-scoped everywhere: every booking query carries `user_id = ?`, anyone else's id is a 404. Logs carry ids only.

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Booking scopes of GET /city-services/bookings.
const (
	ScopeUpcoming = "upcoming"
	ScopePast     = "past"
)

// Review statuses: a new review waits for moderation (CB-DIR-05 publishes or hides it); only published ones are public.
const (
	ReviewPending   = "pending"
	ReviewPublished = "published"
	ReviewHidden    = "hidden"
)

// ReportReasons are the «گزارش اطلاعات نادرست» reason codes (docs/canvas-build/directory.md).
var ReportReasons = []string{"wrong_info", "closed", "wrong_location", "other"}

// Who cancelled a booking.
const (
	CancelledByUser  = "user"
	CancelledByAdmin = "admin"
)

// Directory booking errors (the handler maps each to a field message).
var (
	ErrSlotUnavailable = errors.New("cityservices: slot unavailable") // closed, past, or of another place / service
	ErrSlotFull        = errors.New("cityservices: slot full")
	ErrNotOpen         = errors.New("cityservices: booking no longer open")
	ErrAlreadyReviewed = errors.New("cityservices: already reviewed")
)

// BookingDir holds a booking's directory columns (CB-DIR-01) and its joined service / slot / child / review.
// Zero values = a bloom-era request.
type BookingDir struct {
	Code            sql.NullString
	ServiceID       sql.NullInt64
	SlotID          sql.NullInt64
	ChildID         sql.NullInt64
	PriceToman      sql.NullInt64
	ConfirmedAt     sql.NullTime
	CancelledAt     sql.NullTime
	CancelledBy     sql.NullString
	CompletedAt     sql.NullTime
	ServiceName     sql.NullString
	ServiceDuration sql.NullInt64
	SlotStartsAt    sql.NullTime
	SlotEndsAt      sql.NullTime
	ChildName       sql.NullString
	ChildBirth      sql.NullTime
	PlaceLat        sql.NullFloat64
	PlaceLng        sql.NullFloat64
	CancelFreeHours sql.NullInt64
	CancelPolicy    sql.NullString
	ReviewID        sql.NullInt64
	ReviewRating    sql.NullInt64
	ReviewStatus    sql.NullString
}

const bookingDirColumns = `, b.code, b.service_id, b.slot_id, b.child_id, b.price_toman, b.confirmed_at, b.cancelled_at,
	b.cancelled_by, b.completed_at, v.name, v.duration_minutes, s.starts_at, s.ends_at, ch.name, ch.birth_date,
	l.lat, l.lng, l.cancel_free_hours, l.cancel_policy, r.id, r.rating, r.status`

// bookingDirJoins never widens the row count: every join is on a unique key; the child join re-checks the owner.
const bookingDirJoins = `
	LEFT JOIN city_service_services v ON v.id = b.service_id
	LEFT JOIN city_service_slots s ON s.id = b.slot_id
	LEFT JOIN children ch ON ch.id = b.child_id AND ch.owner_id = b.user_id
	LEFT JOIN city_service_reviews r ON r.booking_id = b.id`

func (d *BookingDir) targets() []any {
	return []any{&d.Code, &d.ServiceID, &d.SlotID, &d.ChildID, &d.PriceToman, &d.ConfirmedAt, &d.CancelledAt,
		&d.CancelledBy, &d.CompletedAt, &d.ServiceName, &d.ServiceDuration, &d.SlotStartsAt, &d.SlotEndsAt, &d.ChildName,
		&d.ChildBirth, &d.PlaceLat, &d.PlaceLng, &d.CancelFreeHours, &d.CancelPolicy, &d.ReviewID, &d.ReviewRating,
		&d.ReviewStatus}
}

// wall reads a DATETIME (Tehran wall clock) as a Tehran instant whatever location the driver scanned it in.
func wall(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, civildate.Tehran)
}

// StartsAt is the booked slot's start (Tehran), zero for a request without a slot.
func (b Booking) StartsAt() (time.Time, bool) {
	if !b.SlotStartsAt.Valid {
		return time.Time{}, false
	}
	return wall(b.SlotStartsAt.Time), true
}

// IsOpen reports a requested / confirmed booking.
func (b Booking) IsOpen() bool { return b.Status == StatusRequested || b.Status == StatusConfirmed }

// IsUpcoming reports an open booking whose slot (or, without one, preferred day) is not over yet.
func (b Booking) IsUpcoming(now time.Time) bool {
	if !b.IsOpen() {
		return false
	}
	if at, ok := b.StartsAt(); ok {
		return !now.After(at)
	}
	return !b.PreferredDate.Before(civildate.FromTime(now.In(civildate.Tehran)))
}

// CancelFreeUntil is the end of the free-cancellation window: the slot start minus the place's cancel_free_hours.
// ok is false without a slot or without a policy.
func (b Booking) CancelFreeUntil() (time.Time, bool) {
	at, ok := b.StartsAt()
	if !ok || !b.CancelFreeHours.Valid {
		return time.Time{}, false
	}
	return at.Add(-time.Duration(b.CancelFreeHours.Int64) * time.Hour), true
}

// CancelState is why a booking can or cannot be cancelled now.
type CancelState int

// Cancel states.
const (
	CancelAllowed      CancelState = iota
	CancelClosed                   // decided (rejected / cancelled / completed) or its slot already started
	CancelWindowPassed             // confirmed and inside the place's no-free-cancellation window
)

// CancelStateAt applies the cancel rules: an open booking can be cancelled until its slot starts; once the Ritme team
// has confirmed it, only until `cancel_free_hours` before the start (the place's policy — there is no fee to charge,
// so after the window she contacts the place instead). A bloom request without a slot keeps bloom's rule.
func (b Booking) CancelStateAt(now time.Time) CancelState {
	if !b.IsOpen() {
		return CancelClosed
	}
	at, ok := b.StartsAt()
	if !ok {
		return CancelAllowed
	}
	if !now.Before(at) {
		return CancelClosed
	}
	if until, ok := b.CancelFreeUntil(); ok && b.Status == StatusConfirmed && !now.Before(until) {
		return CancelWindowPassed
	}
	return CancelAllowed
}

// CanReschedule: an open slot booking that could still be cancelled may ask for another time.
func (b Booking) CanReschedule(now time.Time) bool {
	return b.SlotID.Valid && b.CancelStateAt(now) == CancelAllowed
}

// CanReview: a completed booking without a review.
func (b Booking) CanReview() bool { return b.Status == StatusCompleted && !b.ReviewID.Valid }

// ChildAgeMonths is the booked child's age in completed months on the booking day (-1 without a child).
func (b Booking) ChildAgeMonths() int {
	if !b.ChildBirth.Valid {
		return -1
	}
	return AgeMonthsOn(civildate.FromTime(b.ChildBirth.Time), b.PreferredDate)
}

// AgeMonthsOn is the age in completed calendar months of someone born on birth, on day on (0 before the birth).
func AgeMonthsOn(birth, on civildate.Date) int {
	if on.Before(birth) {
		return 0
	}
	m := (on.Year-birth.Year)*12 + int(on.Month) - int(birth.Month)
	if on.Day < birth.Day {
		m--
	}
	return max(m, 0)
}

// AgeFits reports whether ageMonths is inside a service's optional bounds.
func AgeFits(v Service, ageMonths int) bool {
	if v.AgeMinMonths.Valid && int64(ageMonths) < v.AgeMinMonths.Int64 {
		return false
	}
	return !v.AgeMaxMonths.Valid || int64(ageMonths) <= v.AgeMaxMonths.Int64
}

// ---------------------------------------------------------------------------
// Reads

// UserBookingsScoped lists the caller's bookings: upcoming soonest first, past latest first, "" = bloom's order.
func (s *Store) UserBookingsScoped(ctx context.Context, userID uint64, scope string, now time.Time) ([]Booking, error) {
	const upcoming = ` AND b.status IN ('requested','confirmed') AND (
		(s.starts_at IS NOT NULL AND s.starts_at >= ?) OR (s.starts_at IS NULL AND b.preferred_date >= ?))`
	today := civildate.FromTime(now.In(civildate.Tehran))
	switch scope {
	case ScopeUpcoming:
		return s.bookings(ctx, " WHERE b.user_id = ?"+upcoming+
			" ORDER BY COALESCE(s.starts_at, b.preferred_date), b.id LIMIT ?", userID, now, today, maxBookingsShown)
	case ScopePast:
		return s.bookings(ctx, " WHERE b.user_id = ? AND NOT (b.status IN ('requested','confirmed') AND COALESCE("+
			"(s.starts_at IS NOT NULL AND s.starts_at >= ?) OR (s.starts_at IS NULL AND b.preferred_date >= ?), 0))"+
			" ORDER BY COALESCE(s.starts_at, b.preferred_date) DESC, b.id DESC LIMIT ?", userID, now, today, maxBookingsShown)
	}
	return s.UserBookings(ctx, userID)
}

// BookingCounts counts the caller's upcoming and past bookings («پیش رو · ۲»).
func (s *Store) BookingCounts(ctx context.Context, userID uint64, now time.Time) (upcoming, past int, err error) {
	today := civildate.FromTime(now.In(civildate.Tehran))
	err = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(u), 0), COALESCE(SUM(1 - u), 0) FROM (
		SELECT (b.status IN ('requested','confirmed') AND COALESCE(
			(s.starts_at IS NOT NULL AND s.starts_at >= ?) OR (s.starts_at IS NULL AND b.preferred_date >= ?), 0)) AS u
		FROM city_service_bookings b LEFT JOIN city_service_slots s ON s.id = b.slot_id WHERE b.user_id = ?) x`,
		now, today, userID).Scan(&upcoming, &past)
	return upcoming, past, err
}

// Child is the caller's own child (bloom `children`), first name and birth day only.
type Child struct {
	ID    uint64
	Name  string
	Birth civildate.Date
}

// UserChild is one child of ownerID (ErrNotFound for anyone else's).
func (s *Store) UserChild(ctx context.Context, ownerID, id uint64) (Child, error) {
	var c Child
	err := s.db.QueryRowContext(ctx, `SELECT id, name, birth_date FROM children WHERE id = ? AND owner_id = ?`, id, ownerID).
		Scan(&c.ID, &c.Name, &c.Birth)
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// UserContact is the account's display name and mobile (the default contact of a request).
func (s *Store) UserContact(ctx context.Context, userID uint64) (name, mobile string, err error) {
	var n, m sql.NullString
	err = s.db.QueryRowContext(ctx, `SELECT name, mobile FROM users WHERE id = ?`, userID).Scan(&n, &m)
	return n.String, m.String, err
}

// ActiveServiceRow is an active service of listingID (ErrNotFound otherwise).
func (s *Store) ActiveServiceRow(ctx context.Context, listingID, id uint64) (Service, error) {
	var v Service
	err := s.db.QueryRowContext(ctx, `SELECT id, kind, name, description, format, duration_minutes, age_min_months,
		age_max_months, capacity, price_toman, package_sessions, package_validity_days FROM city_service_services
		WHERE id = ? AND listing_id = ? AND is_active = 1`, id, listingID).
		Scan(&v.ID, &v.Kind, &v.Name, &v.Description, &v.Format, &v.DurationMinutes, &v.AgeMinMonths,
			&v.AgeMaxMonths, &v.Capacity, &v.PriceToman, &v.PackageSessions, &v.PackageValidityDays)
	if errors.Is(err, sql.ErrNoRows) {
		return v, ErrNotFound
	}
	return v, err
}

// BookableSlot is a future open slot of listingID whose service (if any) is active (ErrSlotUnavailable otherwise).
// Seats are NOT checked here — the transaction does.
func (s *Store) BookableSlot(ctx context.Context, listingID, id uint64, now time.Time) (Slot, error) {
	var v Slot
	err := s.db.QueryRowContext(ctx, `SELECT s.id, s.service_id, s.starts_at, s.ends_at, s.capacity, s.booked_count
		FROM city_service_slots s LEFT JOIN city_service_services v ON v.id = s.service_id
		WHERE s.id = ? AND s.listing_id = ? AND s.status = 'open' AND s.starts_at > ? AND (s.service_id IS NULL OR v.is_active = 1)`,
		id, listingID, now).Scan(&v.ID, &v.ServiceID, &v.StartsAt, &v.EndsAt, &v.Capacity, &v.BookedCount)
	if errors.Is(err, sql.ErrNoRows) {
		return v, ErrSlotUnavailable
	}
	return v, err
}

// OpenOnSlot counts the caller's open bookings of one slot (a seat per mother and slot).
func (s *Store) OpenOnSlot(ctx context.Context, userID, slotID uint64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM city_service_bookings WHERE user_id = ? AND slot_id = ?
		AND status IN ('requested','confirmed')`, userID, slotID).Scan(&n)
	return n, err
}

// ---------------------------------------------------------------------------
// Writes

// DirBooking are the CB-DIR-03 extras of a new request.
type DirBooking struct {
	ServiceID  sql.NullInt64
	SlotID     sql.NullInt64
	ChildID    sql.NullInt64
	PriceToman sql.NullInt64
}

// bookingCodeAlphabet has no 0/O/1/I (read over the phone); 8 symbols = 40 random bits, retried on a clash.
const bookingCodeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// NewBookingCode returns a fresh user-visible booking code.
func NewBookingCode() (string, error) {
	var b strings.Builder
	n := big.NewInt(int64(len(bookingCodeAlphabet)))
	for range 8 {
		x, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		b.WriteByte(bookingCodeAlphabet[x.Int64()])
	}
	return b.String(), nil
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// takeSeat locks the slot row and takes one seat; ErrSlotUnavailable / ErrSlotFull when it cannot.
func takeSeat(ctx context.Context, tx *sql.Tx, listingID, slotID uint64, now time.Time) error {
	var capacity, booked int
	var status string
	var starts time.Time
	err := tx.QueryRowContext(ctx, `SELECT capacity, booked_count, status, starts_at FROM city_service_slots
		WHERE id = ? AND listing_id = ? FOR UPDATE`, slotID, listingID).Scan(&capacity, &booked, &status, &starts)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSlotUnavailable
	}
	if err != nil {
		return fmt.Errorf("cityservices: lock slot: %w", err)
	}
	if status != "open" || !wall(starts).After(now) {
		return ErrSlotUnavailable
	}
	if booked >= capacity {
		return ErrSlotFull
	}
	if _, err := tx.ExecContext(ctx, `UPDATE city_service_slots SET booked_count = booked_count + 1, updated_at = ? WHERE id = ?`,
		now, slotID); err != nil {
		return fmt.Errorf("cityservices: take seat: %w", err)
	}
	return nil
}

// releaseSeat gives one seat back (never below zero).
func releaseSeat(ctx context.Context, tx *sql.Tx, slotID sql.NullInt64, now time.Time) error {
	if !slotID.Valid {
		return nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE city_service_slots SET booked_count = booked_count - 1, updated_at = ?
		WHERE id = ? AND booked_count > 0`, now, slotID.Int64)
	return err
}

func (s *Store) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// CreateDirBooking stores a request in status requested with a fresh code, holding a seat of its slot (if any) in
// the same transaction, and returns it. ErrSlotUnavailable / ErrSlotFull when the seat cannot be taken.
func (s *Store) CreateDirBooking(ctx context.Context, userID uint64, in NewBooking, d DirBooking, now time.Time) (Booking, error) {
	var id int64
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if d.SlotID.Valid {
			if err := takeSeat(ctx, tx, in.ListingID, uint64(d.SlotID.Int64), now); err != nil { //nolint:gosec // G115: ids are positive
				return err
			}
		}
		for range 5 {
			code, err := NewBookingCode()
			if err != nil {
				return err
			}
			res, err := tx.ExecContext(ctx, `INSERT INTO city_service_bookings
				(user_id, listing_id, contact_name, contact_phone, preferred_date, child_age_range, note, status, code,
				 service_id, slot_id, child_id, price_toman, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'requested', ?, ?, ?, ?, ?, ?, ?)`,
				userID, in.ListingID, in.ContactName, in.ContactPhone, in.PreferredDate, in.ChildAgeRange, in.Note, code,
				d.ServiceID, d.SlotID, d.ChildID, d.PriceToman, now, now)
			if isDuplicate(err) {
				continue // a code clash (40 random bits): draw again
			}
			if err != nil {
				return fmt.Errorf("cityservices: create booking: %w", err)
			}
			id, err = res.LastInsertId()
			return err
		}
		return errors.New("cityservices: no free booking code")
	})
	if err != nil {
		return Booking{}, err
	}
	return s.UserBooking(ctx, userID, uint64(id)) //nolint:gosec // G115: auto-increment ids are positive
}

// CancelOwnBooking cancels the caller's open booking (still in status from) and gives its seat back; false when the
// row moved meanwhile.
func (s *Store) CancelOwnBooking(ctx context.Context, userID uint64, b Booking, now time.Time) (bool, error) {
	done := false
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE city_service_bookings SET status = 'cancelled', cancelled_at = ?,
			cancelled_by = 'user', updated_at = ? WHERE id = ? AND user_id = ? AND status = ?`, now, now, b.ID, userID, b.Status)
		if err != nil {
			return fmt.Errorf("cityservices: cancel: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return nil
		}
		done = true
		return releaseSeat(ctx, tx, b.SlotID, now)
	})
	return done, err
}

// Reschedule moves the caller's open booking to another slot of the same place: the new seat is taken and the old one
// given back in one transaction, and the booking goes back to `requested` (the Ritme team confirms the new time).
func (s *Store) Reschedule(ctx context.Context, userID uint64, b Booking, slot Slot, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := takeSeat(ctx, tx, b.ListingID, slot.ID, now); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE city_service_bookings SET slot_id = ?, preferred_date = ?, status = 'requested',
			confirmed_at = NULL, updated_at = ? WHERE id = ? AND user_id = ? AND status = ? AND slot_id = ?`,
			slot.ID, civildate.FromTime(wall(slot.StartsAt)), now, b.ID, userID, b.Status, b.SlotID)
		if err != nil {
			return fmt.Errorf("cityservices: reschedule: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotOpen
		}
		return releaseSeat(ctx, tx, b.SlotID, now)
	})
}

// NewReview is a validated review.
type NewReview struct {
	Rating                             int
	Cleanliness, Instructor, Amenities sql.NullInt64
	Body                               sql.NullString
}

// CreateReview stores the review of a completed booking in status pending (moderated before it is public).
// ErrAlreadyReviewed when the booking has one (booking_id is unique).
func (s *Store) CreateReview(ctx context.Context, userID uint64, b Booking, in NewReview, now time.Time) (uint64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO city_service_reviews (listing_id, booking_id, user_id, rating,
		score_cleanliness, score_instructor, score_amenities, body, status, created_at, updated_at)
		SELECT b.listing_id, b.id, b.user_id, ?, ?, ?, ?, ?, ?, ?, ? FROM city_service_bookings b
		WHERE b.id = ? AND b.user_id = ? AND b.status = 'completed'`,
		in.Rating, in.Cleanliness, in.Instructor, in.Amenities, in.Body, ReviewPending, now, now, b.ID, userID)
	if isDuplicate(err) {
		return 0, ErrAlreadyReviewed
	}
	if err != nil {
		return 0, fmt.Errorf("cityservices: review: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNotOpen
	}
	id, err := res.LastInsertId()
	return uint64(id), err //nolint:gosec // G115: auto-increment ids are positive
}

// OpenReport reports whether the caller already has an open report about the place.
func (s *Store) OpenReport(ctx context.Context, userID, listingID uint64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM city_service_reports WHERE user_id = ? AND listing_id = ?
		AND status = 'open'`, userID, listingID).Scan(&n)
	return n > 0, err
}

// CreateReport stores a wrong-information report (worked in admin).
func (s *Store) CreateReport(ctx context.Context, userID, listingID uint64, reason string, note sql.NullString, now time.Time) (uint64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO city_service_reports (listing_id, user_id, reason, note, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'open', ?, ?)`, listingID, userID, reason, note, now, now)
	if err != nil {
		return 0, fmt.Errorf("cityservices: report: %w", err)
	}
	id, err := res.LastInsertId()
	return uint64(id), err //nolint:gosec // G115: auto-increment ids are positive
}
