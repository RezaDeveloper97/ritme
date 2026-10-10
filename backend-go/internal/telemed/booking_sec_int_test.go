package telemed_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/telemed"
)

// B-N7-03b: booking security fixes.

func (be *benv) verify(t *testing.T, tok string, r response, paid bool) response {
	t.Helper()
	_, err := be.fake.Decide(context.Background(), r.obj("payment")["authority"].(string), paid)
	require.NoError(t, err)
	return be.do(t, http.MethodPost, "/api/v1/telemed/bookings/verify", tok, "fa",
		map[string]any{"reference": r.obj("booking")["reference"], "authority": r.obj("payment")["authority"]})
}

func TestBookingSec_HoldCaps(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")

	// One active hold per user: a new hold lapses the older one and frees its slot.
	first := be.book(t, tok, "16:00", nil)
	require.Equal(t, http.StatusCreated, first.status, first.body)
	require.Equal(t, http.StatusCreated, be.book(t, tok, "16:30", nil).status)
	assert.Equal(t, "expired", be.String(`SELECT status FROM telemed_bookings WHERE id = ?`, id(first.obj("booking")["id"])))
	assert.Equal(t, 1, be.Int(`SELECT COUNT(*) FROM telemed_bookings WHERE status = 'held'`))

	// ≤ 2 upcoming with one doctor.
	be.pay(t, tok, "16:00", nil)
	be.pay(t, tok, "16:30", nil)
	r := be.book(t, tok, "18:00", nil)
	assert.Equal(t, http.StatusConflict, r.status, r.body)
	assert.Equal(t, "too_many_bookings", r.body["error_code"])

	// ≤ 5 upcoming overall: park the paid visits on other doctors, fill up to 5.
	be.Exec(`UPDATE telemed_bookings SET doctor_id = ? WHERE status = 'confirmed'`, be.other)
	be.pay(t, tok, "18:00", nil)
	be.pay(t, tok, "18:30", nil)
	be.Exec(`UPDATE telemed_bookings SET doctor_id = ? WHERE status = 'confirmed' AND doctor_id = ?`, be.other+2, be.a)
	be.pay(t, tok, "19:00", nil)
	r = be.book(t, tok, "19:30", nil)
	assert.Equal(t, http.StatusConflict, r.status, r.body)
	assert.Equal(t, "too_many_bookings", r.body["error_code"])

	// ≤ 10 holds per rolling hour.
	_, tok2 := be.user(t, "Sara", "09120000002")
	for i := range 10 {
		require.Equal(t, http.StatusCreated, be.book(t, tok2, "19:30", nil).status, i)
	}
	r = be.book(t, tok2, "19:30", nil)
	assert.Equal(t, http.StatusTooManyRequests, r.status, r.body)
	assert.Equal(t, "too_many_holds", r.body["error_code"])
	be.clk.set(fixed.Add(61 * time.Minute))
	assert.Equal(t, http.StatusCreated, be.book(t, tok2, "19:30", nil).status, "the hour rolled over")
}

func TestBookingSec_LateVerify(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")

	// Lapsed (not swept) hold whose slot is still free: confirmed after the slot recheck.
	r := be.book(t, tok, "16:00", nil)
	require.Equal(t, http.StatusCreated, r.status)
	be.clk.set(fixed.Add(30 * time.Minute))
	v := be.verify(t, tok, r, true)
	require.Equal(t, http.StatusOK, v.status, v.body)
	assert.Equal(t, "confirmed", v.obj("booking")["status"])

	// Lapsed hold paid after the visit started: refunded.
	be.clk.set(fixed)
	r = be.book(t, tok, "18:00", nil)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	be.clk.set(time.Date(2030, 1, 8, 18, 5, 0, 0, fixed.Location()))
	v = be.verify(t, tok, r, true)
	assert.Equal(t, http.StatusConflict, v.status, v.body)
	assert.Equal(t, "booking_refunded", v.body["error_code"])
	assert.Equal(t, "refunded", be.String(`SELECT payment_status FROM telemed_bookings WHERE id = ?`, id(r.obj("booking")["id"])))

	// Doctor delisted before the payment came back: refunded (L3).
	be.clk.set(fixed)
	_, tok2 := be.user(t, "Sara", "09120000002")
	r = be.book(t, tok2, "19:00", nil)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	be.Exec(`UPDATE telemed_doctors SET is_active = 0 WHERE id = ?`, be.a)
	v = be.verify(t, tok2, r, true)
	assert.Equal(t, "booking_refunded", v.body["error_code"], v.body)
	assert.Equal(t, "failed", be.String(`SELECT status FROM telemed_bookings WHERE id = ?`, id(r.obj("booking")["id"])))
}

func TestBookingSec_DeclinedAfterCancelIsTerminal(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	r := be.book(t, tok, "16:00", nil)
	require.Equal(t, http.StatusCreated, r.status)
	c := be.do(t, http.MethodPost, fmt.Sprintf("/api/v1/telemed/bookings/%d/cancel", id(r.obj("booking")["id"])), tok, "fa", nil)
	require.Equal(t, http.StatusOK, c.status, c.body)
	v := be.verify(t, tok, r, false)
	assert.Equal(t, "payment_failed", v.body["error_code"])
	assert.Equal(t, "declined", be.String(`SELECT payment_status FROM telemed_bookings WHERE id = ?`, id(r.obj("booking")["id"])))
	v = be.verify(t, tok, r, false)
	assert.Equal(t, "booking_closed", v.body["error_code"])
}

func TestBookingSec_RefundOutboxOnce(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	bid := be.pay(t, tok, "16:00", nil)
	be.Exec(`UPDATE telemed_bookings SET ref_id = 'WRONG' WHERE id = ?`, bid) // the provider rejects this refund
	r := be.do(t, http.MethodPost, fmt.Sprintf("/api/v1/telemed/bookings/%d/cancel", bid), tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "refund_pending", r.obj("booking")["payment_status"])
	rows, err := be.bk.RefundPending(context.Background(), 20, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, bid, rows[0].ID)
	r = be.do(t, http.MethodPost, fmt.Sprintf("/api/v1/telemed/bookings/%d/cancel", bid), tok, "fa", nil)
	assert.Equal(t, "booking_closed", r.body["error_code"], "no second refund")
}

func TestBookingSec_RescheduleCap(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	p := fmt.Sprintf("/api/v1/telemed/bookings/%d/reschedule", be.pay(t, tok, "16:00", nil))
	for _, at := range []string{"16:30", "18:00"} {
		r := be.do(t, http.MethodPost, p, tok, "fa", map[string]any{"starts_at": "2030-01-08 " + at})
		require.Equal(t, http.StatusOK, r.status, r.body)
	}
	r := be.do(t, http.MethodPost, p, tok, "fa", map[string]any{"starts_at": "2030-01-08 18:30"})
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "reschedule_limit", r.body["error_code"])
}

func TestBookingSec_ConsentStartBound(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	bid := be.pay(t, tok, "16:00", map[string]any{"share": []any{"cycle_summary"}})
	ctx := context.Background()
	_, err := be.bk.AuthorizeShare(ctx, be.a, bid, telemed.ScopeCycleSummary, time.Date(2030, 1, 7, 15, 59, 59, 0, fixed.Location()))
	require.ErrorIs(t, err, telemed.ErrConsentRequired, "earlier than 24 h before the visit")
	_, err = be.bk.AuthorizeShare(ctx, be.a, bid, telemed.ScopeCycleSummary, time.Date(2030, 1, 7, 16, 0, 0, 0, fixed.Location()))
	require.NoError(t, err)
}

func TestBookingSec_CompleteConcurrentWithHolds(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	bid := be.pay(t, tok, "16:00", nil)
	users := make([]uint64, 6)
	for i := range users {
		users[i], _ = be.user(t, fmt.Sprintf("U%d", i), fmt.Sprintf("0912200%04d", i))
	}
	end := time.Date(2030, 1, 8, 16, 21, 0, 0, fixed.Location())
	start := time.Date(2030, 1, 8, 19, 0, 0, 0, fixed.Location())
	var wg sync.WaitGroup
	errs := make(chan error, len(users)+1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := be.bk.Complete(context.Background(), bid, end)
		errs <- err
	}()
	for _, u := range users {
		wg.Add(1)
		go func(u uint64) {
			defer wg.Done()
			_, err := be.bk.Create(context.Background(), u, telemed.BookInput{
				DoctorID: be.a, Mode: telemed.ModeVideo, StartsAt: start, ForWhom: telemed.ForSelf,
			}, catalog.Localizer{Locale: "fa", Default: "fa"})
			if err != nil && !assert.ErrorIs(t, err, telemed.ErrSlotUnavailable) {
				return
			}
		}(u)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, "completed", be.String(`SELECT status FROM telemed_bookings WHERE id = ?`, bid))
	assert.Equal(t, 1, be.Int(`SELECT visits_count FROM telemed_doctors WHERE id = ?`, be.a))
}
