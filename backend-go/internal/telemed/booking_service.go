package telemed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// BookInput is a validated POST /telemed/bookings body.
type BookInput struct {
	DoctorID    uint64
	Mode        string
	StartsAt    time.Time // Tehran wall-clock, minute precision
	ForWhom     string
	ChildID     uint64 // ForChild only
	PatientName string // ForOther only
	Reason      string // catalog code or ""
	Note        string
	Share       []string // consent scopes granted now (ForSelf only)
}

// Created is a new booking and, when something is to be paid, where to send the user.
type Created struct {
	Booking store.TelemedBooking
	Payment *payments.Session
}

// Quote is the review screen's price of one visit type for the user.
type Quote struct {
	Doctor store.TelemedDoctor
	Visit  store.TelemedVisitType
	Price  Price
	Plus   bool // the user has the Plus visit discount (even when the percent is 0)
}

// Quote prices the doctor's visit of mode for the user (ErrDoctorNotFound / ErrModeNotOffered).
func (b *Bookings) Quote(ctx context.Context, userID, doctorID uint64, mode string) (Quote, error) {
	now := b.now(ctx)
	doctor, err := b.listed(ctx, doctorID)
	if err != nil {
		return Quote{}, err
	}
	vt, err := b.visitType(ctx, doctorID, mode)
	if err != nil {
		return Quote{}, err
	}
	pr, plus, err := b.price(ctx, userID, vt, now)
	if err != nil {
		return Quote{}, err
	}
	return Quote{Doctor: doctor, Visit: vt, Price: pr, Plus: plus}, nil
}

func (b *Bookings) listed(ctx context.Context, doctorID uint64) (store.TelemedDoctor, error) {
	d, err := b.q.GetListedDoctor(ctx, doctorID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.TelemedDoctor{}, ErrDoctorNotFound
	}
	if err != nil {
		return store.TelemedDoctor{}, fmt.Errorf("telemed: doctor: %w", err)
	}
	return d, nil
}

// visitType is the doctor's active visit type of mode ("" = the first offered).
func (b *Bookings) visitType(ctx context.Context, doctorID uint64, mode string) (store.TelemedVisitType, error) {
	vts, err := b.q.ListActiveVisitTypesByDoctors(ctx, []uint64{doctorID})
	if err != nil {
		return store.TelemedVisitType{}, fmt.Errorf("telemed: visit types: %w", err)
	}
	for _, v := range vts {
		if mode == "" || v.Mode == mode {
			return v, nil
		}
	}
	return store.TelemedVisitType{}, ErrModeNotOffered
}

// slotOffered checks that start is a free slot of the doctor's mode now (time off, lead time, horizon and the busy
// intervals of other bookings considered) and returns the visit type. Call it while holding the doctor lock.
func (b *Bookings) slotOffered(ctx context.Context, doctor store.TelemedDoctor, mode string, start, now time.Time) (store.TelemedVisitType, error) {
	day := civildate.InTehran(start)
	details, err := b.dir.LoadDetails(ctx, []store.TelemedDoctor{doctor}, day, 1)
	if err != nil {
		return store.TelemedVisitType{}, err
	}
	d := details[doctor.ID]
	vt, ok := d.Visit(mode)
	if !ok {
		return store.TelemedVisitType{}, ErrModeNotOffered
	}
	days := GenerateDays(SlotQuery{
		Rules: d.Rules, Mode: mode, Duration: int(vt.DurationMinutes), From: day, Days: 1, Now: now, Blocked: d.Blocked,
	})
	for _, dd := range days {
		for _, s := range dd.Slots {
			if s.Start.Equal(start) {
				return vt, nil
			}
		}
	}
	return store.TelemedVisitType{}, ErrSlotUnavailable
}

// Create holds the slot for the user and starts the payment. Free visits (nothing to charge) are confirmed at once.
// The doctor row lock serialises concurrent holds; a second request for an overlapping slot gets ErrSlotUnavailable.
func (b *Bookings) Create(ctx context.Context, userID uint64, in BookInput, loc catalog.Localizer) (Created, error) {
	now := b.now(ctx)
	doctor, err := b.listed(ctx, in.DoctorID)
	if err != nil {
		return Created{}, err
	}
	switch {
	case in.ForWhom != ForSelf && len(in.Share) > 0:
		return Created{}, ErrConsentNotAllowed
	case in.ForWhom == ForChild:
		if _, err := b.q.GetOwnedChild(ctx, store.GetOwnedChildParams{ID: in.ChildID, OwnerID: userID}); errors.Is(err, sql.ErrNoRows) {
			return Created{}, ErrChildNotFound
		} else if err != nil {
			return Created{}, fmt.Errorf("telemed: child: %w", err)
		}
	}
	ref, err := newReference()
	if err != nil {
		return Created{}, err
	}
	var id uint64
	var pr Price
	err = b.inTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		if _, err := q.LockDoctor(ctx, doctor.ID); err != nil {
			return fmt.Errorf("telemed: lock doctor: %w", err)
		}
		if err := q.ExpireDoctorHolds(ctx, store.ExpireDoctorHoldsParams{Stamp: nt(now), DoctorID: doctor.ID, Now: nt(now)}); err != nil {
			return fmt.Errorf("telemed: expire holds: %w", err)
		}
		closed, err := b.holdCaps(ctx, q, userID, doctor.ID, now)
		if err != nil {
			return err
		}
		// Busy reads outside this transaction: ignore the user's own hold that holdCaps just closed.
		vt, err := b.slotOffered(withoutBooking(ctx, closed), doctor, in.Mode, in.StartsAt, now)
		if err != nil {
			return err
		}
		if pr, _, err = b.price(ctx, userID, vt, now); err != nil {
			return err
		}
		if pr.Total > 0 && b.opts.Gateway == nil {
			return ErrPaymentUnavailable
		}
		end := in.StartsAt.Add(time.Duration(vt.DurationMinutes) * time.Minute)
		n, err := q.CountOverlappingBookings(ctx, store.CountOverlappingBookingsParams{
			DoctorID: doctor.ID, EndsAt: end, StartsAt: in.StartsAt, Now: nt(now),
		})
		if err != nil {
			return fmt.Errorf("telemed: overlaps: %w", err)
		}
		if n > 0 {
			return ErrSlotUnavailable
		}
		var gateway sql.NullString
		if b.opts.Gateway != nil {
			gateway = ns(b.opts.Gateway.Name())
		}
		var child sql.NullInt64
		if in.ForWhom == ForChild {
			child = sql.NullInt64{Int64: int64(in.ChildID), Valid: true} //nolint:gosec // G115: auto-increment id
		}
		newID, err := q.InsertBooking(ctx, store.InsertBookingParams{
			Reference: ref, UserID: userID, DoctorID: doctor.ID, Mode: vt.Mode, DurationMinutes: vt.DurationMinutes,
			StartsAt: in.StartsAt, EndsAt: end, SlotKey: ns(SlotKey(doctor.ID, in.StartsAt)),
			HoldExpiresAt: nt(now.Add(b.opts.Config.HoldTTL)), ForWhom: in.ForWhom, ChildID: child,
			PatientName: ns(in.PatientName), Reason: ns(in.Reason), Note: ns(in.Note),
			PriceRials: pr.Price, DiscountRials: pr.Discount, DiscountSource: ns(pr.Source), TotalRials: pr.Total,
			PaymentStatus: PaymentNone, Gateway: gateway, Now: nt(now),
		})
		if isDuplicate(err) {
			return ErrSlotUnavailable
		}
		if err != nil {
			return fmt.Errorf("telemed: insert booking: %w", err)
		}
		id = uint64(newID) //nolint:gosec // G115: auto-increment id
		for _, scope := range in.Share {
			if err := q.UpsertConsent(ctx, store.UpsertConsentParams{BookingID: id, Scope: scope, GrantedAt: now, Stamp: nt(now)}); err != nil {
				return fmt.Errorf("telemed: consent: %w", err)
			}
		}
		if err := bookingShareEvidence(ctx, tx, id, userID, doctor.ID, in.Share, now); err != nil { // T-M10-23 ledger
			return err
		}
		if pr.Total > 0 {
			return nil
		}
		bk, err := q.GetBooking(ctx, id)
		if err != nil {
			return fmt.Errorf("telemed: booking: %w", err)
		}
		return b.confirm(ctx, tx, q, bk, PaymentFree, nil, loc, now)
	})
	if err != nil {
		return Created{}, err
	}
	out := Created{}
	if pr.Total > 0 {
		sess, err := b.opts.Gateway.Create(ctx, payments.Request{
			Reference: ref, AmountRials: pr.Total, Description: "Ritme visit " + ref, CallbackURL: b.opts.Config.CallbackURL,
		})
		if err != nil {
			b.opts.Logger.WarnContext(ctx, "telemed: gateway create failed", "gateway", b.opts.Gateway.Name(), "reference", ref, "error", err.Error())
			_ = b.q.CloseBooking(ctx, store.CloseBookingParams{Status: StatusFailed, Now: nt(now), ID: id})
			return Created{}, fmt.Errorf("%w: %w", ErrPaymentUnavailable, err)
		}
		if err := b.q.SetBookingAuthority(ctx, store.SetBookingAuthorityParams{Authority: ns(sess.Authority), UpdatedAt: nt(now), ID: id}); err != nil {
			return Created{}, fmt.Errorf("telemed: booking authority: %w", err)
		}
		out.Payment = &sess
	}
	if out.Booking, err = b.q.GetBooking(ctx, id); err != nil {
		return Created{}, fmt.Errorf("telemed: booking: %w", err)
	}
	return out, nil
}

// holdCaps limits holds (doctor lock held): MaxHoldsPerHour per rolling hour (ErrTooManyHolds), one active hold per
// user (her older holds lapse), MaxUpcomingPerDoctor with this doctor and MaxUpcomingTotal overall (ErrTooManyBookings).
// It returns the id of the hold it closed (0 = none).
func (b *Bookings) holdCaps(ctx context.Context, q *store.Queries, userID, doctorID uint64, now time.Time) (uint64, error) {
	n, err := q.CountUserHoldsSince(ctx, store.CountUserHoldsSinceParams{UserID: userID, Since: nt(now.Add(-time.Hour))})
	if err != nil {
		return 0, fmt.Errorf("telemed: recent holds: %w", err)
	}
	if n >= MaxHoldsPerHour {
		return 0, ErrTooManyHolds
	}
	closed, err := q.GetUserActiveHold(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("telemed: active hold: %w", err)
	}
	if err := q.CloseUserHolds(ctx, store.CloseUserHoldsParams{Now: nt(now), UserID: userID}); err != nil {
		return 0, fmt.Errorf("telemed: close holds: %w", err)
	}
	for _, c := range []struct {
		doctor uint64
		max    int64
	}{{doctorID, MaxUpcomingPerDoctor}, {0, MaxUpcomingTotal}} {
		n, err := q.CountUserUpcoming(ctx, store.CountUserUpcomingParams{UserID: userID, DoctorID: c.doctor, Now: now, HoldNow: nt(now)})
		if err != nil {
			return 0, fmt.Errorf("telemed: upcoming bookings: %w", err)
		}
		if n >= c.max {
			return 0, ErrTooManyBookings
		}
	}
	return closed, nil
}

var errDuplicateRef = errors.New("telemed: bank reference already used")

// confirm turns the locked booking into a confirmed one and creates its care appointment (in the same transaction).
// A lost slot is ErrSlotUnavailable, a reused bank reference errDuplicateRef.
func (b *Bookings) confirm(ctx context.Context, tx *sql.Tx, q *store.Queries, bk store.TelemedBooking, paymentStatus string,
	res *payments.Result, loc catalog.Localizer, now time.Time,
) error {
	p := store.ConfirmBookingParams{
		SlotKey: ns(SlotKey(bk.DoctorID, bk.StartsAt)), PaymentStatus: paymentStatus, Now: nt(now), ID: bk.ID,
	}
	if res != nil {
		p.RefID, p.CardPan, p.PaidAt = ns(res.RefID), ns(res.CardPAN), nt(now)
	}
	if err := q.ConfirmBooking(ctx, p); err != nil {
		if isDuplicate(err) {
			if strings.Contains(err.Error(), "slot_key") {
				return ErrSlotUnavailable
			}
			return errDuplicateRef
		}
		return fmt.Errorf("telemed: confirm booking: %w", err)
	}
	visit, err := b.careVisit(ctx, bk, loc)
	if err != nil {
		return err
	}
	apptID, err := care.ScheduleVisit(ctx, carestore.New(tx), bk.UserID, visit, now)
	if err != nil {
		return err
	}
	if err := q.SetBookingAppointment(ctx, store.SetBookingAppointmentParams{
		AppointmentID: sql.NullInt64{Int64: int64(apptID), Valid: true}, UpdatedAt: nt(now), ID: bk.ID, //nolint:gosec // G115: auto-increment id
	}); err != nil {
		return fmt.Errorf("telemed: booking appointment: %w", err)
	}
	return nil
}

// careKinds maps visit modes to care appointment kinds.
var careKinds = map[string]string{ModeVideo: "online", ModePhone: "phone", ModeInPerson: "in_person"}

// careVisit is the care appointment of a booking: localized title, doctor, specialty, clinic address; the reminder
// fires a day ahead for an in-person visit, an hour ahead otherwise.
func (b *Bookings) careVisit(ctx context.Context, bk store.TelemedBooking, loc catalog.Localizer) (care.BookedVisit, error) {
	doctor, err := b.q.GetDoctor(ctx, bk.DoctorID)
	if err != nil {
		return care.BookedVisit{}, fmt.Errorf("telemed: doctor: %w", err)
	}
	labels, err := b.dir.LoadLabels(ctx)
	if err != nil {
		return care.BookedVisit{}, err
	}
	v := care.BookedVisit{
		BookingID: bk.ID, Title: T("appointment_titles."+bk.Mode, loc.Locale), Kind: careKinds[bk.Mode],
		With: textOf(loc, doctor.Name), RemindBefore: "1h", ScheduledAt: bk.StartsAt,
	}
	if raw := labels.Title(GroupSpecialties, doctor.Specialty); raw != nil {
		v.Specialty = textOf(loc, raw)
	}
	if bk.Mode == ModeInPerson {
		v.RemindBefore = "1d"
		if vt, err := b.visitType(ctx, bk.DoctorID, bk.Mode); err == nil && vt.Address.Valid {
			v.Location = textOf(loc, vt.Address.V)
		}
	}
	if v.Kind == "" {
		v.Kind = care.AppointmentKinds[0]
	}
	return v, nil
}

// textOf is the picked translation as a plain string ("" when missing).
func textOf(loc catalog.Localizer, raw []byte) string {
	s, _ := loc.Text(raw).(string)
	return s
}

// Settled is the outcome of a verify.
type Settled struct {
	Booking     store.TelemedBooking
	AlreadyPaid bool
}

// lockedByReference locks the doctor, then the user's booking reference (the lock order of every booking write).
func (b *Bookings) lockedByReference(ctx context.Context, q *store.Queries, userID uint64, reference string) (store.TelemedBooking, error) {
	pre, err := b.q.LockUserBookingByReference(ctx, store.LockUserBookingByReferenceParams{Reference: reference, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return store.TelemedBooking{}, ErrBookingNotFound
	}
	if err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: booking: %w", err)
	}
	if _, err := q.LockDoctor(ctx, pre.DoctorID); err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: lock doctor: %w", err)
	}
	bk, err := q.LockUserBookingByReference(ctx, store.LockUserBookingByReferenceParams{Reference: reference, UserID: userID})
	if err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: booking: %w", err)
	}
	return bk, nil
}

// locked is lockedByReference by id.
func (b *Bookings) locked(ctx context.Context, q *store.Queries, userID, id uint64) (store.TelemedBooking, error) {
	pre, err := b.q.GetUserBooking(ctx, store.GetUserBookingParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return store.TelemedBooking{}, ErrBookingNotFound
	}
	if err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: booking: %w", err)
	}
	if _, err := q.LockDoctor(ctx, pre.DoctorID); err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: lock doctor: %w", err)
	}
	bk, err := q.LockUserBooking(ctx, store.LockUserBookingParams{ID: id, UserID: userID})
	if err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: booking: %w", err)
	}
	return bk, nil
}

// Verify settles the payment of the user's booking reference server to server (the amount is the booking's stored
// total, never a client value). Idempotent: a confirmed booking is returned unchanged (AlreadyPaid) without asking
// the gateway again. The gateway is asked before the doctor lock is taken (a slow bank never blocks the doctor's
// bookings). A payment that arrives for a slot that is gone (the hold lapsed and the slot was taken, left the
// schedule or is in the past), a doctor no longer listed, or a closed booking is recorded and refunded after the
// commit (ErrBookingRefunded). A declined payment closes the hold (ErrPaymentFailed).
func (b *Bookings) Verify(ctx context.Context, userID uint64, reference, authority string, loc catalog.Localizer) (Settled, error) {
	now := b.now(ctx)
	pre, err := b.q.LockUserBookingByReference(ctx, store.LockUserBookingByReferenceParams{Reference: reference, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Settled{}, ErrBookingNotFound
	}
	if err != nil {
		return Settled{}, fmt.Errorf("telemed: booking: %w", err)
	}
	if paid, err := b.verifiable(pre, authority); err != nil || paid {
		return Settled{Booking: pre, AlreadyPaid: paid}, err
	}
	res, err := b.opts.Gateway.Verify(ctx, payments.VerifyRequest{Authority: authority, AmountRials: pre.TotalRials})
	if err != nil {
		b.opts.Logger.WarnContext(ctx, "telemed: gateway verify failed", "gateway", b.opts.Gateway.Name(), "reference", reference, "error", err.Error())
		return Settled{}, fmt.Errorf("%w: %w", ErrPaymentUnavailable, err)
	}
	var out Settled
	var outcome error // business failures that still commit
	var refund *store.TelemedBooking
	err = b.inTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		bk, err := b.lockedByReference(ctx, q, userID, reference)
		if err != nil {
			return err
		}
		out.Booking = bk
		if paid, err := b.verifiable(bk, authority); err != nil || paid {
			out.AlreadyPaid = paid // a concurrent verify settled it meanwhile
			return err
		}
		holdOpen := bk.Status == StatusHeld || bk.Status == StatusExpired
		failHold := func() error {
			if holdOpen {
				return q.MarkPaymentFailed(ctx, store.MarkPaymentFailedParams{Now: nt(now), ID: bk.ID})
			}
			return q.DeclinePendingPayment(ctx, store.DeclinePendingPaymentParams{Now: nt(now), ID: bk.ID})
		}
		switch {
		case !res.Paid:
			outcome = ErrPaymentFailed
			return failHold()
		case res.AmountRials != bk.TotalRials:
			b.opts.Logger.ErrorContext(ctx, "telemed: settled amount differs from booking", "gateway", b.opts.Gateway.Name(),
				"reference", reference, "total", bk.TotalRials, "settled", res.AmountRials)
			outcome = ErrAmountMismatch
			return failHold()
		}
		if holdOpen {
			ok, err := b.confirmable(ctx, q, bk, now)
			if err != nil {
				return err
			}
			if ok {
				err := b.confirm(ctx, tx, q, bk, PaymentPaid, &res, loc, now)
				switch {
				case err == nil:
					return nil
				case errors.Is(err, errDuplicateRef):
					b.opts.Logger.ErrorContext(ctx, "telemed: bank reference already used by another booking", "gateway", b.opts.Gateway.Name(), "reference", reference)
					outcome = ErrPaymentFailed
					return q.MarkPaymentFailed(ctx, store.MarkPaymentFailedParams{Now: nt(now), ID: bk.ID})
				case !errors.Is(err, ErrSlotUnavailable):
					return err
				}
			}
		}
		// Paid, but the booking cannot be confirmed: record the payment, close the booking, refund after the commit.
		outcome = ErrBookingRefunded
		if err := q.SetBookingPaid(ctx, store.SetBookingPaidParams{RefID: ns(res.RefID), CardPan: ns(res.CardPAN), PaidAt: nt(now), Now: nt(now), ID: bk.ID}); err != nil {
			if isDuplicate(err) {
				outcome = ErrPaymentFailed
				return failHold()
			}
			return fmt.Errorf("telemed: booking paid: %w", err)
		}
		if holdOpen {
			if err := q.CloseBooking(ctx, store.CloseBookingParams{Status: StatusFailed, Now: nt(now), ID: bk.ID}); err != nil {
				return fmt.Errorf("telemed: close booking: %w", err)
			}
		}
		bk.RefID = ns(res.RefID)
		refund = &bk
		return b.queueRefund(ctx, q, bk.ID, now)
	})
	if err != nil {
		return Settled{}, err
	}
	if refund != nil {
		b.settleRefund(ctx, *refund)
	}
	if out.Booking, err = b.q.GetBooking(ctx, out.Booking.ID); err != nil {
		return Settled{}, fmt.Errorf("telemed: booking: %w", err)
	}
	return out, outcome
}

// verifiable reports whether bk is already settled (paid) or why its payment cannot be verified now.
func (b *Bookings) verifiable(bk store.TelemedBooking, authority string) (bool, error) {
	switch {
	case bk.PaymentStatus == PaymentFree || (bk.PaymentStatus == PaymentPaid && (bk.Status == StatusConfirmed || bk.Status == StatusCompleted)):
		return true, nil
	case bk.PaymentStatus != PaymentPending:
		return false, ErrBookingClosed
	case !bk.Authority.Valid || bk.Authority.String != authority:
		return false, ErrAuthorityMismatch
	case b.opts.Gateway == nil || bk.Gateway.String != b.opts.Gateway.Name():
		return false, ErrPaymentUnavailable
	}
	return false, nil
}

// confirmable reports whether the paid hold bk can still become a visit at now (doctor lock held): the visit is
// ahead, the doctor is listed, a lapsed hold's slot is still offered, and nothing else overlaps it.
func (b *Bookings) confirmable(ctx context.Context, q *store.Queries, bk store.TelemedBooking, now time.Time) (bool, error) {
	if !wall(bk.StartsAt).After(now) {
		return false, nil
	}
	doctor, err := b.listed(ctx, bk.DoctorID)
	if errors.Is(err, ErrDoctorNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	lapsed := bk.Status == StatusExpired || !bk.HoldExpiresAt.Valid || !wall(bk.HoldExpiresAt.Time).After(now)
	if lapsed {
		_, err := b.slotOffered(withoutBooking(ctx, bk.ID), doctor, bk.Mode, wall(bk.StartsAt), now)
		switch {
		case errors.Is(err, ErrSlotUnavailable), errors.Is(err, ErrModeNotOffered):
			return false, nil
		case err != nil:
			return false, err
		}
	}
	n, err := q.CountOverlappingBookings(ctx, store.CountOverlappingBookingsParams{
		DoctorID: bk.DoctorID, ExceptID: bk.ID, EndsAt: bk.EndsAt, StartsAt: bk.StartsAt, Now: nt(now),
	})
	if err != nil {
		return false, fmt.Errorf("telemed: overlaps: %w", err)
	}
	return n == 0, nil
}

// queueRefund puts a paid booking in the refund outbox (payment_status refund_pending) inside the booking change's
// transaction; settleRefund pays it out after the commit.
func (b *Bookings) queueRefund(ctx context.Context, q *store.Queries, id uint64, now time.Time) error {
	if err := q.SetBookingRefund(ctx, store.SetBookingRefundParams{PaymentStatus: PaymentRefundPending, Now: nt(now), ID: id}); err != nil {
		return fmt.Errorf("telemed: booking refund: %w", err)
	}
	return nil
}

// settleRefund refunds a refund_pending booking through the gateway once (MarkRefunded only moves refund_pending).
// A provider that cannot refund through the adapter (Zarinpal) or fails leaves it refund_pending for the admins
// (ListRefundPendingBookings) and logs an error with the booking and reference. Paid telemed must stay off in
// production until the admin refund list ships (B-N7-08).
func (b *Bookings) settleRefund(ctx context.Context, bk store.TelemedBooking) {
	gw := "none"
	if b.opts.Gateway != nil {
		gw = b.opts.Gateway.Name()
	}
	if b.opts.Gateway == nil || bk.TotalRials == 0 {
		b.opts.Logger.ErrorContext(ctx, "telemed: refund needs an admin", "gateway", gw, "booking", bk.ID, "reference", bk.Reference)
		return
	}
	r, err := b.opts.Gateway.Refund(ctx, payments.RefundRequest{
		Authority: bk.Authority.String, RefID: bk.RefID.String, AmountRials: bk.TotalRials, Reason: "telemed booking",
	})
	if err != nil {
		b.opts.Logger.ErrorContext(ctx, "telemed: refund needs an admin", "gateway", gw, "booking", bk.ID, "reference", bk.Reference, "error", err.Error())
		return
	}
	now := b.now(ctx)
	if _, err := b.q.MarkRefunded(ctx, store.MarkRefundedParams{RefundID: ns(r.RefundID), RefundedRials: r.AmountRials, Now: nt(now), ID: bk.ID}); err != nil {
		b.opts.Logger.ErrorContext(ctx, "telemed: refund done but not recorded", "gateway", gw, "booking", bk.ID, "reference", bk.Reference, "refund_id", r.RefundID, "error", err.Error())
	}
}

// RefundPending lists the paid bookings an admin still has to refund (newest first).
func (b *Bookings) RefundPending(ctx context.Context, limit, offset int32) ([]store.TelemedBooking, error) {
	rows, err := b.q.ListRefundPendingBookings(ctx, store.ListRefundPendingBookingsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("telemed: refund pending: %w", err)
	}
	return rows, nil
}

// CancelDeadline is the last moment a confirmed booking can be cancelled or moved for free.
func CancelDeadline(bk store.TelemedBooking) time.Time { return wall(bk.StartsAt).Add(-CancelWindow) }

// Changeable reports whether a confirmed booking may still be cancelled or moved at now.
func Changeable(bk store.TelemedBooking, now time.Time) bool {
	return bk.Status == StatusConfirmed && now.Before(CancelDeadline(bk))
}

// Cancel cancels the user's booking: a hold is released; a confirmed booking until CancelWindow before the visit
// (ErrCancelWindowClosed after), with a full refund, its care appointment cancelled and every consent revoked.
func (b *Bookings) Cancel(ctx context.Context, userID, id uint64) (store.TelemedBooking, error) {
	now := b.now(ctx)
	var refund *store.TelemedBooking
	err := b.inTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		bk, err := b.locked(ctx, q, userID, id)
		if err != nil {
			return err
		}
		switch {
		case bk.Status == StatusHeld:
		case bk.Status != StatusConfirmed:
			return ErrBookingClosed
		case !Changeable(bk, now):
			return ErrCancelWindowClosed
		}
		if err := q.CloseBooking(ctx, store.CloseBookingParams{Status: StatusCancelled, CancelledAt: nt(now), Now: nt(now), ID: bk.ID}); err != nil {
			return fmt.Errorf("telemed: cancel booking: %w", err)
		}
		if err := q.RevokeAllConsents(ctx, store.RevokeAllConsentsParams{Now: nt(now), BookingID: bk.ID}); err != nil {
			return fmt.Errorf("telemed: revoke consents: %w", err)
		}
		if bk.AppointmentID.Valid {
			if err := care.CancelVisit(ctx, carestore.New(tx), userID, uint64(bk.AppointmentID.Int64), now); err != nil { //nolint:gosec // G115: positive id
				return err
			}
		}
		if bk.PaymentStatus == PaymentPaid {
			refund = &bk
			return b.queueRefund(ctx, q, bk.ID, now)
		}
		return nil
	})
	if err != nil {
		return store.TelemedBooking{}, err
	}
	if refund != nil {
		b.settleRefund(ctx, *refund)
	}
	return b.q.GetBooking(ctx, id)
}

// Reschedule moves the user's confirmed booking to another free slot of the same doctor and mode, until CancelWindow
// before the current time. The care appointment moves along.
func (b *Bookings) Reschedule(ctx context.Context, userID, id uint64, start time.Time) (store.TelemedBooking, error) {
	now := b.now(ctx)
	err := b.inTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		bk, err := b.locked(ctx, q, userID, id)
		if err != nil {
			return err
		}
		switch {
		case bk.Status != StatusConfirmed:
			return ErrBookingClosed
		case !Changeable(bk, now):
			return ErrCancelWindowClosed
		case bk.Reschedules >= MaxReschedules:
			return ErrRescheduleLimit
		case start.Equal(wall(bk.StartsAt)):
			return ErrSlotUnavailable
		}
		doctor, err := b.listed(ctx, bk.DoctorID)
		if err != nil {
			return err
		}
		if _, err := b.slotOffered(withoutBooking(ctx, bk.ID), doctor, bk.Mode, start, now); err != nil {
			return err
		}
		end := start.Add(time.Duration(bk.DurationMinutes) * time.Minute)
		n, err := q.CountOverlappingBookings(ctx, store.CountOverlappingBookingsParams{
			DoctorID: bk.DoctorID, ExceptID: bk.ID, EndsAt: end, StartsAt: start, Now: nt(now),
		})
		if err != nil {
			return fmt.Errorf("telemed: overlaps: %w", err)
		}
		if n > 0 {
			return ErrSlotUnavailable
		}
		err = q.RescheduleBooking(ctx, store.RescheduleBookingParams{
			StartsAt: start, EndsAt: end, SlotKey: ns(SlotKey(bk.DoctorID, start)), Now: nt(now), ID: bk.ID,
		})
		if isDuplicate(err) {
			return ErrSlotUnavailable
		}
		if err != nil {
			return fmt.Errorf("telemed: reschedule: %w", err)
		}
		if bk.AppointmentID.Valid {
			return care.MoveVisit(ctx, carestore.New(tx), userID, uint64(bk.AppointmentID.Int64), start, now) //nolint:gosec // G115: positive id
		}
		return nil
	})
	if err != nil {
		return store.TelemedBooking{}, err
	}
	return b.q.GetBooking(ctx, id)
}

// Booking is the user's booking id (ErrBookingNotFound for anyone else's).
func (b *Bookings) Booking(ctx context.Context, userID, id uint64) (store.TelemedBooking, error) {
	bk, err := b.q.GetUserBooking(ctx, store.GetUserBookingParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return store.TelemedBooking{}, ErrBookingNotFound
	}
	if err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: booking: %w", err)
	}
	return bk, nil
}

// Booking list scopes.
const (
	ScopeUpcoming = "upcoming"
	ScopePast     = "past"
	ScopeAll      = "all"
)

// ListScopes are the list scopes.
var ListScopes = []string{ScopeUpcoming, ScopePast, ScopeAll}

// Upcoming reports whether a booking is still ahead: held (not lapsed) or confirmed, and not ended.
func Upcoming(bk store.TelemedBooking, now time.Time) bool {
	switch bk.Status {
	case StatusConfirmed:
		return wall(bk.EndsAt).After(now)
	case StatusHeld:
		return bk.HoldExpiresAt.Valid && wall(bk.HoldExpiresAt.Time).After(now)
	}
	return false
}

// List is the user's bookings of scope: upcoming soonest first, past newest first.
func (b *Bookings) List(ctx context.Context, userID uint64, scope string) ([]store.TelemedBooking, error) {
	now := b.now(ctx)
	rows, err := b.q.ListUserBookings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("telemed: bookings: %w", err)
	}
	out := make([]store.TelemedBooking, 0, len(rows))
	for _, r := range rows {
		up := Upcoming(r, now)
		if r.Status == StatusHeld && !up {
			continue // a lapsed hold is not a booking
		}
		if scope == ScopeAll || (scope == ScopeUpcoming) == up {
			out = append(out, r)
		}
	}
	if scope == ScopePast {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out, nil
}

// Complete marks a confirmed visit completed and counts it on the doctor (the doctor side, B-N7-08, and the sweeper
// call it). It reports whether the booking changed.
func (b *Bookings) Complete(ctx context.Context, id uint64, now time.Time) (bool, error) {
	changed := false
	pre, err := b.q.GetBooking(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("telemed: booking: %w", err)
	}
	err = b.inTx(ctx, func(_ *sql.Tx, q *store.Queries) error {
		// The doctor lock first, like every other booking write (no lock-order deadlock with holds/settlement).
		if _, err := q.LockDoctor(ctx, pre.DoctorID); err != nil {
			return fmt.Errorf("telemed: lock doctor: %w", err)
		}
		n, err := q.CompleteBooking(ctx, store.CompleteBookingParams{Now: nt(now), Stamp: nt(now), ID: id})
		if err != nil {
			return fmt.Errorf("telemed: complete booking: %w", err)
		}
		if n == 0 {
			return nil
		}
		changed = true
		bk, err := q.GetBooking(ctx, id)
		if err != nil {
			return fmt.Errorf("telemed: booking: %w", err)
		}
		if err := q.IncrementDoctorVisits(ctx, bk.DoctorID); err != nil {
			return fmt.Errorf("telemed: visits count: %w", err)
		}
		return nil
	})
	return changed, err
}

// Sweep expires lapsed holds and completes ended visits. It returns how many of each changed.
func (b *Bookings) Sweep(ctx context.Context, now time.Time) (expired, completed int, err error) {
	holds, err := b.q.ListLapsedHolds(ctx, nt(now))
	if err != nil {
		return 0, 0, fmt.Errorf("telemed: lapsed holds: %w", err)
	}
	for _, h := range holds {
		n, err := b.q.ExpireHold(ctx, store.ExpireHoldParams{Stamp: nt(now), ID: h.ID, Now: nt(now)})
		if err != nil {
			return expired, completed, fmt.Errorf("telemed: expire hold: %w", err)
		}
		expired += int(n)
	}
	ended, err := b.q.ListEndedConfirmed(ctx, now)
	if err != nil {
		return expired, completed, fmt.Errorf("telemed: ended visits: %w", err)
	}
	for _, e := range ended {
		ok, err := b.Complete(ctx, e.ID, now)
		if err != nil {
			return expired, completed, err
		}
		if ok {
			completed++
		}
	}
	return expired, completed, nil
}

// Background job (T-M9-11, docs/engine-v6/jobs.md): the booking sweep every minute on the default queue, also at
// start-up.
const (
	JobSweep  = "telemed:booking_sweep"
	SweepCron = "* * * * *"
)

// RegisterJobs puts the sweep job on q.
func (b *Bookings) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobSweep, MaxTries: 2, Backoff: []time.Duration{10 * time.Second}, Timeout: 50 * time.Second,
	}, b.SweepJob(now), SweepCron, queue.Unique(50*time.Second), queue.RunAtStart())
}

// SweepJob is the telemed:booking_sweep handler. Idempotent: ExpireHold only matches a hold that is still held and
// lapsed, Complete only a booking that is still confirmed.
func (b *Bookings) SweepJob(now func() time.Time) queue.Handler {
	return func(ctx context.Context, _ []byte) error {
		_, _, err := b.Sweep(ctx, now())
		return err
	}
}
