package telemed_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The scheduled telemed:booking_sweep handler run twice leaves the same state: one expired hold, one completed visit,
// the doctor's visit count bumped once (T-M9-11).
func TestJobs_BookingSweepIsIdempotent(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	_, tok2 := be.user(t, "Sara", "09120000002")
	require.Equal(t, http.StatusCreated, be.book(t, tok2, "18:00", nil).status) // a hold that will lapse
	be.pay(t, tok, "16:00", nil)                                                // a visit that will end

	end := time.Date(2030, 1, 8, 16, 21, 0, 0, fixed.Location())
	sweep := be.bk.SweepJob(func() time.Time { return end })
	state := func() string {
		return be.String("SELECT GROUP_CONCAT(CONCAT_WS(':', id, status, payment_status) ORDER BY id) FROM telemed_bookings")
	}
	ctx := context.Background()
	require.NoError(t, sweep(ctx, []byte("{}")))
	first := state()
	visits := be.Int(`SELECT visits_count FROM telemed_doctors WHERE id = ?`, be.a)
	require.NoError(t, sweep(ctx, []byte("{}")))
	assert.Equal(t, first, state())
	assert.Equal(t, visits, be.Int(`SELECT visits_count FROM telemed_doctors WHERE id = ?`, be.a))
	assert.Equal(t, 1, visits)
	assert.Contains(t, first, "completed")
	assert.Contains(t, first, "expired")
}
