package telemed_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/telemed"
)

func TestBookingsAdmin_ListRefundQueueAndGuard(t *testing.T) {
	be := setupBooking(t, true)
	var logs bytes.Buffer
	telemed.NewBookingsAdmin(be.bk.Directory(), slog.New(slog.NewJSONHandler(&logs, nil))).Routes(be.Route(), be.Kit)
	_, tok := be.user(t, "A", "09120000001")
	bid := be.pay(t, tok, "16:00", nil)

	// 401 without a session; the list needs any admin and never leaks notes or the full mobile.
	for _, p := range []string{"/telemed/bookings", "/telemed/bookings/refunds"} {
		assert.Equal(t, http.StatusUnauthorized, be.Anonymous().Get(p).Status, p)
	}
	r := be.As(admintest.EditorID).Get("/telemed/bookings?status=confirmed")
	require.Equal(t, http.StatusOK, r.Status, r.Body)
	require.Len(t, r.Items(), 1)
	it := r.Items()[0].(map[string]any)
	assert.Equal(t, "0912•••0001", it["user_mobile"])
	assert.Equal(t, "paid", it["payment_status"])
	assert.Empty(t, be.As(admintest.EditorID).Get("/telemed/bookings/refunds").Items())
	assert.Empty(t, be.As(admintest.EditorID).Get("/telemed/bookings?status=cancelled").Items())

	// The doctor cannot be deleted while the booking exists (409), even by a super admin.
	d := be.As(admintest.SuperID).JSON(http.MethodDelete, fmt.Sprintf("/telemed/doctors/%d", be.a), nil)
	assert.Equal(t, http.StatusConflict, d.Status)
	assert.Equal(t, "doctor_has_bookings", d.Code())
	assert.Equal(t, 1, be.Int(`SELECT COUNT(*) FROM telemed_doctors WHERE id = ?`, be.a))

	// Manual refund: only super; only from refund_pending; audit-logged; once.
	be.Exec(`UPDATE telemed_bookings SET payment_status = 'refund_pending' WHERE id = ?`, bid)
	q := be.As(admintest.EditorID).Get("/telemed/bookings/refunds")
	require.Len(t, q.Items(), 1)
	assert.EqualValues(t, 1, q.Data()["counts"].(map[string]any)["refund_pending"])
	path := fmt.Sprintf("/telemed/bookings/%d/refunded", bid)
	assert.Equal(t, http.StatusForbidden, be.As(admintest.EditorID).JSON(http.MethodPost, path, map[string]any{}).Status)
	assert.Equal(t, http.StatusNotFound, be.As(admintest.SuperID).JSON(http.MethodPost, "/telemed/bookings/999999/refunded", nil).Status)
	ok := be.As(admintest.SuperID).JSON(http.MethodPost, path, map[string]any{"refund_ref": "BANK-1"})
	require.Equal(t, http.StatusOK, ok.Status, ok.Body)
	assert.Equal(t, "refunded", be.String(`SELECT payment_status FROM telemed_bookings WHERE id = ?`, bid))
	assert.Equal(t, "BANK-1", be.String(`SELECT refund_id FROM telemed_bookings WHERE id = ?`, bid))
	assert.Contains(t, logs.String(), `"audit":"telemed_booking.mark_refunded"`)
	again := be.As(admintest.SuperID).JSON(http.MethodPost, path, map[string]any{})
	assert.Equal(t, http.StatusConflict, again.Status)
	assert.Equal(t, "not_refund_pending", again.Code())
	assert.Empty(t, be.As(admintest.EditorID).Get("/telemed/bookings/refunds").Items())
}
