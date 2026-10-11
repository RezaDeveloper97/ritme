package cityservices

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestCancelStateAt(t *testing.T) {
	slotBooking := func(status string, hours int64, hasPolicy bool) Booking {
		b := Booking{Status: status}
		b.SlotID = sql.NullInt64{Int64: 1, Valid: true}
		b.SlotStartsAt = sql.NullTime{Time: tehran("2026-09-25 17:00"), Valid: true}
		b.CancelFreeHours = sql.NullInt64{Int64: hours, Valid: hasPolicy}
		return b
	}
	cases := []struct {
		name string
		b    Booking
		now  string
		want CancelState
	}{
		{"bloom request without slot", Booking{Status: StatusRequested, PreferredDate: civildate.MustParse("2020-01-01")}, "2026-09-23 10:00", CancelAllowed},
		{"decided", slotBooking(StatusCompleted, 24, true), "2026-09-23 10:00", CancelClosed},
		{"confirmed before the window", slotBooking(StatusConfirmed, 24, true), "2026-09-24 16:59", CancelAllowed},
		{"confirmed inside the window", slotBooking(StatusConfirmed, 24, true), "2026-09-24 17:00", CancelWindowPassed},
		{"requested inside the window", slotBooking(StatusRequested, 24, true), "2026-09-25 16:00", CancelAllowed},
		{"confirmed without a policy", slotBooking(StatusConfirmed, 0, false), "2026-09-25 16:59", CancelAllowed},
		{"started", slotBooking(StatusRequested, 24, true), "2026-09-25 17:00", CancelClosed},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, c.b.CancelStateAt(tehran(c.now)), c.name)
	}
	b := slotBooking(StatusConfirmed, 24, true)
	until, ok := b.CancelFreeUntil()
	require.True(t, ok)
	assert.Equal(t, tehran("2026-09-24 17:00"), until)
	assert.True(t, b.IsUpcoming(tehran("2026-09-25 16:00")))
	assert.False(t, b.IsUpcoming(tehran("2026-09-25 18:00")))
	assert.False(t, b.CanReschedule(tehran("2026-09-24 18:00")))
}

func TestAgeMonthsOnAndFit(t *testing.T) {
	d := civildate.MustParse
	assert.Equal(t, 14, AgeMonthsOn(d("2025-07-20"), d("2026-09-24")))
	assert.Equal(t, 13, AgeMonthsOn(d("2025-07-25"), d("2026-09-24")))
	assert.Equal(t, 0, AgeMonthsOn(d("2026-10-01"), d("2026-09-24")))
	v := Service{AgeMinMonths: sql.NullInt64{Int64: 6, Valid: true}, AgeMaxMonths: sql.NullInt64{Int64: 18, Valid: true}}
	assert.True(t, AgeFits(v, 6))
	assert.True(t, AgeFits(v, 18))
	assert.False(t, AgeFits(v, 5))
	assert.False(t, AgeFits(v, 19))
	assert.True(t, AgeFits(Service{}, 100), "no bounds")
}

func TestNewBookingCode(t *testing.T) {
	re := regexp.MustCompile(`^[2-9A-HJ-NP-Z]{8}$`)
	seen := map[string]bool{}
	for range 200 {
		c, err := NewBookingCode()
		require.NoError(t, err)
		assert.Regexp(t, re, c)
		seen[c] = true
	}
	assert.Len(t, seen, 200)
}
