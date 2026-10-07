package telemed

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// Visit booking (bloom B-N7-03, deviation D-80; artboards nbl_v17_ReviewBook / nbl_v17_Booked):
//
//	GET    /api/v1/telemed/bookings                    the user's bookings (scope upcoming | past | all)
//	GET    /api/v1/telemed/bookings/quote              price with the Plus discount, children, reasons (review screen)
//	POST   /api/v1/telemed/bookings                    hold a slot and start the payment (201)
//	POST   /api/v1/telemed/bookings/verify             settle the payment the user came back from (idempotent)
//	GET    /api/v1/telemed/bookings/{id}               one booking
//	POST   /api/v1/telemed/bookings/{id}/cancel        free until CancelWindow before the visit (refund)
//	POST   /api/v1/telemed/bookings/{id}/reschedule    free until CancelWindow before the visit
//	PUT    /api/v1/telemed/bookings/{id}/consent       grant / revoke data-share scopes
//	DELETE /api/v1/telemed/bookings/{id}/consent       revoke every scope
//
// Lifecycle: held (slot blocked for Config.HoldTTL while the user pays) → confirmed (paid, care appointment + reminder
// created) → completed (the visit ended; the doctor's visits_count grows and the user may review), or cancelled /
// expired (hold lapsed) / failed (payment declined). A slot is blocked by held and confirmed bookings; the doctor row
// lock plus the UNIQUE slot_key make double-booking impossible even under concurrent requests.

// Booking statuses (telemed_bookings.status).
const (
	StatusHeld      = "held"
	StatusConfirmed = "confirmed"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"
	StatusFailed    = "failed"
)

// Payment statuses (telemed_bookings.payment_status).
const (
	PaymentNone          = "none"
	PaymentPending       = "pending"
	PaymentPaid          = "paid"
	PaymentFree          = "free"
	PaymentRefunded      = "refunded"
	PaymentRefundPending = "refund_pending" // the provider cannot refund through the adapter: admins refund by hand
)

// For whom the visit is (telemed_bookings.for_whom), in display order.
const (
	ForSelf  = "self"
	ForChild = "child"
	ForOther = "other"
)

// ForWhom lists the for-whom values.
var ForWhom = []string{ForSelf, ForChild, ForOther}

// Data-share consent scopes (telemed_booking_consents.scope), in display order.
const (
	ScopeCycleSummary       = "cycle_summary"       // «خلاصه ۶ سیکل اخیر»: cycle length, period, symptoms
	ScopeBBTLH              = "bbt_lh"              // «دمای پایه و تست LH»: this cycle
	ScopeAssistantSummaries = "assistant_summaries" // «گفت‌وگو با دستیار سلامت»: recent assistant summaries (B-N7-06)
)

// Scopes lists the consent scopes.
var Scopes = []string{ScopeCycleSummary, ScopeBBTLH, ScopeAssistantSummaries}

// GroupVisitReasons holds the «دلیل مراجعه» chips (catalog codes, admin content).
const GroupVisitReasons = "telemed_visit_reasons"

// DiscountSourcePlus marks a Plus «تخفیف ویزیت» on a booking.
const DiscountSourcePlus = "plus"

// Booking limits.
const (
	// CancelWindow: cancel and reschedule are free until this long before the visit, and impossible after.
	CancelWindow = 2 * time.Hour
	// ConsentGrace: a doctor may read what the user shared until this long after the visit ends (follow-up questions).
	ConsentGrace = 72 * time.Hour
	// MaxPatientNameLen / MaxBookingNoteLen: characters.
	MaxPatientNameLen = 64
	MaxBookingNoteLen = 1000
	// SweepEvery: how often lapsed holds are expired and ended visits completed.
	SweepEvery = time.Minute
)

// Booking errors mapped by the handlers.
var (
	ErrBookingNotFound    = errors.New("telemed: booking not found")
	ErrSlotUnavailable    = errors.New("telemed: slot not available")
	ErrPaymentUnavailable = errors.New("telemed: payment unavailable")
	ErrPaymentFailed      = errors.New("telemed: payment failed")
	ErrAmountMismatch     = errors.New("telemed: paid amount differs")
	ErrAuthorityMismatch  = errors.New("telemed: authority mismatch")
	ErrBookingClosed      = errors.New("telemed: booking closed")
	ErrBookingRefunded    = errors.New("telemed: paid but refunded")
	ErrCancelWindowClosed = errors.New("telemed: cancel window closed")
	ErrConsentNotAllowed  = errors.New("telemed: consent only for one's own visit")
	ErrConsentRequired    = errors.New("telemed: no active consent for this share")
	ErrChildNotFound      = errors.New("telemed: child not found")
)

// PaymentGateway is the payments adapter the booking charges through (payments.Gateway implements it).
type PaymentGateway interface {
	Name() string
	Create(ctx context.Context, req payments.Request) (payments.Session, error)
	Verify(ctx context.Context, req payments.VerifyRequest) (payments.Result, error)
	Refund(ctx context.Context, req payments.RefundRequest) (payments.RefundResult, error)
}

// DiscountChecker reports whether the user gets the Plus visit discount now (plus.VisitDiscount entitlement).
type DiscountChecker func(ctx context.Context, userID uint64, now time.Time) (bool, error)

// BookingOptions wire Bookings.
type BookingOptions struct {
	Gateway  PaymentGateway  // nil = payments unavailable (paid bookings answer 503)
	Discount DiscountChecker // nil = nobody gets the discount
	Config   config.Telemed
	Clock    clock.Clock // fallback clock of Busy (requests pin it per context)
	Logger   *slog.Logger
}

// Bookings is the booking service. It is also the directory's Busy and VisitChecker.
type Bookings struct {
	db   *sql.DB
	q    *store.Queries
	dir  *Service
	opts BookingOptions
}

// NewBookings wires the booking service and its directory (Directory) with itself as Busy / VisitChecker.
func NewBookings(conn *sql.DB, reader *catalog.Reader, opts BookingOptions) *Bookings {
	if opts.Clock == nil {
		opts.Clock = clock.Real{}
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Config.HoldTTL <= 0 {
		opts.Config.HoldTTL = 10 * time.Minute
	}
	b := &Bookings{db: conn, q: store.New(conn), opts: opts}
	b.dir = NewService(conn, reader, b, b)
	return b
}

// Directory is the doctors directory backed by these bookings.
func (b *Bookings) Directory() *Service { return b.dir }

// Payments reports whether a gateway is configured.
func (b *Bookings) Payments() bool { return b.opts.Gateway != nil }

func (b *Bookings) now(ctx context.Context) time.Time {
	return clock.FromContext(ctx, b.opts.Clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

type excludeKey struct{}

// withoutBooking makes Busy ignore one booking (a reschedule may move into its own interval).
func withoutBooking(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, excludeKey{}, id)
}

func excluded(ctx context.Context) uint64 {
	id, _ := ctx.Value(excludeKey{}).(uint64)
	return id
}

// Busy implements Busy: confirmed bookings and holds that have not lapsed.
func (b *Bookings) Busy(ctx context.Context, doctorIDs []uint64, from, to time.Time) (map[uint64][]Interval, error) {
	if len(doctorIDs) == 0 {
		return nil, nil
	}
	rows, err := b.q.ListBusyBookings(ctx, store.ListBusyBookingsParams{
		DoctorIds: doctorIDs, ExceptID: excluded(ctx), ToAt: to, FromAt: from, Now: nt(b.now(ctx)),
	})
	if err != nil {
		return nil, fmt.Errorf("telemed: busy bookings: %w", err)
	}
	out := make(map[uint64][]Interval, len(rows))
	for _, r := range rows {
		out[r.DoctorID] = append(out[r.DoctorID], Interval{Start: wall(r.StartsAt), End: wall(r.EndsAt)})
	}
	return out, nil
}

// ReviewableVisit implements VisitChecker: the user's oldest completed, unreviewed visit with the doctor.
func (b *Bookings) ReviewableVisit(ctx context.Context, userID, doctorID uint64) (uint64, bool, error) {
	id, err := b.q.ReviewableBooking(ctx, store.ReviewableBookingParams{UserID: userID, DoctorID: doctorID})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("telemed: reviewable booking: %w", err)
	}
	return id, true, nil
}

// ---------------------------------------------------------------------------
// helpers

func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// SlotKey is the UNIQUE slot_key of an active booking.
func SlotKey(doctorID uint64, start time.Time) string {
	return strconv.FormatUint(doctorID, 10) + ":" + start.In(civildate.Tehran).Format("2006-01-02T15:04")
}

var refEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// newReference is an unguessable public booking reference ("TB" + 16 base32 chars, 80 random bits).
func newReference() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("telemed: reference: %w", err)
	}
	return "TB" + refEncoding.EncodeToString(buf), nil
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// inTx runs fn in a READ COMMITTED transaction (after the doctor row lock is granted, the reads that follow must see
// what the previous lock holder committed).
func (b *Bookings) inTx(ctx context.Context, fn func(tx *sql.Tx, q *store.Queries) error) error {
	tx, err := b.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("telemed: begin: %w", err)
	}
	if err := fn(tx, b.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("telemed: commit: %w", err)
	}
	return nil
}

// Price is a visit price with the Plus discount, in rials.
type Price struct {
	Price, Discount, Total uint64
	Percent                int
	Source                 string // DiscountSourcePlus or ""
}

// PriceOf applies percent to price; the discount is rounded down to whole toman (10 rials).
func PriceOf(price uint64, percent int) Price {
	p := Price{Price: price, Total: price}
	if percent <= 0 || price == 0 {
		return p
	}
	d := price * uint64(percent) / 100 / 10 * 10 //nolint:gosec // G115: percent validated 0–90
	p.Discount, p.Total, p.Percent, p.Source = d, price-d, percent, DiscountSourcePlus
	return p
}

// price is the user's price of a visit type now.
func (b *Bookings) price(ctx context.Context, userID uint64, vt store.TelemedVisitType, now time.Time) (Price, bool, error) {
	plus := false
	if b.opts.Discount != nil {
		ok, err := b.opts.Discount(ctx, userID, now)
		if err != nil {
			return Price{}, false, fmt.Errorf("telemed: plus discount: %w", err)
		}
		plus = ok
	}
	if !plus {
		return PriceOf(vt.PriceRials, 0), false, nil
	}
	return PriceOf(vt.PriceRials, b.opts.Config.PlusDiscountPercent), true, nil
}
