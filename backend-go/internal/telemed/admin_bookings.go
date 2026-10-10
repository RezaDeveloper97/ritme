package telemed

import (
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// BookingsAdmin serves /api/admin/v1/telemed/bookings* (admin-api §19c, B-N7-08): the bookings list and the manual
// refund queue. No notes, reasons, children or message data is exposed; the patient's mobile is masked.
type BookingsAdmin struct {
	q      *store.Queries
	logger *slog.Logger
}

// NewBookingsAdmin wires the handlers.
func NewBookingsAdmin(svc *Service, logger *slog.Logger) *BookingsAdmin {
	if logger == nil {
		logger = slog.Default()
	}
	return &BookingsAdmin{q: svc.Queries(), logger: logger}
}

// Routes mounts the module (static paths before :id).
func (h *BookingsAdmin) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/telemed/bookings", kit.Admin(h.List))
	route(fiber.MethodGet, "/telemed/bookings/refunds", kit.Admin(h.Refunds))
	route(fiber.MethodPost, "/telemed/bookings/:id/refunded", kit.Super(h.MarkRefunded))
}

var (
	bookingStatuses = []string{StatusHeld, StatusConfirmed, StatusCompleted, StatusCancelled, StatusExpired, StatusFailed}
	paymentStatuses = []string{PaymentNone, PaymentPending, PaymentPaid, PaymentFree, PaymentRefunded, PaymentRefundPending, PaymentDeclined}
)

func maskMobile(s sql.NullString) any {
	m := s.String
	if !s.Valid || len(m) < 8 {
		return nil
	}
	return m[:4] + "•••" + m[len(m)-4:]
}

func adminBookingJSON(r store.ListAdminBookingsRow) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID, "reference", r.Reference, "user_id", r.UserID, "user_mobile", maskMobile(r.UserMobile),
		"doctor", jsonx.Obj("id", r.DoctorID, "name", form.Raw(r.DoctorName)),
		"mode", r.Mode, "status", r.Status, "payment_status", r.PaymentStatus,
		"starts_at", jsonx.ISO8601(wall(r.StartsAt)), "ends_at", jsonx.ISO8601(wall(r.EndsAt)),
		"total_rials", r.TotalRials, "refunded_rials", r.RefundedRials, "refund_ref", httpadmin.NullString(r.RefundID),
		"refunded_at", isoOrNil(r.RefundedAt), "paid_at", isoOrNil(r.PaidAt), "gateway", httpadmin.NullString(r.Gateway),
		"created_at", isoOrNil(r.CreatedAt),
	)
}

func known(v string, set []string) string {
	for _, s := range set {
		if s == v {
			return v
		}
	}
	return ""
}

func (h *BookingsAdmin) list(c fiber.Ctx, status, payment string) error {
	search := strings.TrimSpace(c.Query("q"))
	var doctor uint64
	if n, ok := parseUint(c.Query("doctor_id")); ok {
		doctor = n
	}
	pattern := form.Contains(search)
	total, err := h.q.CountAdminBookings(c.Context(), store.CountAdminBookingsParams{
		Status: status, PaymentStatus: payment, DoctorID: doctor, Pattern: pattern})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminBookings(c.Context(), store.ListAdminBookingsParams{
		Status: status, PaymentStatus: payment, DoctorID: doctor, Pattern: pattern,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, adminBookingJSON(r))
	}
	pending, err := h.q.CountRefundPendingBookings(c.Context())
	if err != nil {
		return err
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("status", status, "payment_status", payment, "q", search, "doctor_id", doctor))
	page.Set("counts", jsonx.Obj("refund_pending", pending))
	page.Set("statuses", jsonx.List(bookingStatuses)).Set("payment_statuses", jsonx.List(paymentStatuses))
	return httpadmin.OK(c, page)
}

func parseUint(s string) (uint64, bool) {
	var n uint64
	if s == "" || len(s) > 18 {
		return 0, false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		n = n*10 + uint64(ch-'0')
	}
	return n, true
}

// List is GET /telemed/bookings?status=&payment_status=&doctor_id=&q=<reference>&page= (unknown filter values = all).
func (h *BookingsAdmin) List(c fiber.Ctx) error {
	return h.list(c, known(c.Query("status"), bookingStatuses), known(c.Query("payment_status"), paymentStatuses))
}

// Refunds is GET /telemed/bookings/refunds: paid bookings whose refund an admin still has to make (newest first).
func (h *BookingsAdmin) Refunds(c fiber.Ctx) error { return h.list(c, "", PaymentRefundPending) }

// MarkRefunded is POST /telemed/bookings/:id/refunded {refund_ref?} (super): the admin refunded the payment outside the
// gateway; refund_pending → refunded (a conditional update, so a second call is the 409 `not_refund_pending`).
func (h *BookingsAdmin) MarkRefunded(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Booking")
	}
	bk, err := h.q.GetBooking(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Booking")
	}
	if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{validation.F("refund_ref", "nullable|string|max:64")})
	if err != nil {
		return err
	}
	n, err := h.q.MarkRefunded(c.Context(), store.MarkRefundedParams{
		RefundID: form.Str(data, "refund_ref"), RefundedRials: bk.TotalRials, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: bk.ID})
	if err != nil {
		return err
	}
	if n == 0 {
		return httpadmin.Fail(fiber.StatusConflict, "not_refund_pending", "This booking has no pending refund.")
	}
	httpadmin.Audit(c, h.logger, "telemed_booking.mark_refunded", "telemed_booking", bk.ID,
		slog.Uint64("refunded_rials", bk.TotalRials))
	return httpadmin.OK(c, jsonx.Obj("id", bk.ID, "payment_status", PaymentRefunded), "Booking marked as refunded.")
}
