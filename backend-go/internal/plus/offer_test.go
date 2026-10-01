package plus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOfferPrice(t *testing.T) {
	assert.Equal(t, uint64(1185000), OfferPrice(2370000, 50)) // nbl_Prem_TrialSheet: ۲۳۷ → ۱۱۸٫۵ هزار تومان
	assert.Equal(t, uint64(395000), OfferPrice(790000, 50))   // «ماهانه حدود ۳۹٫۵ هزار»
	assert.Equal(t, uint64(990000), OfferPrice(990000, 0))    // offer off
	assert.Equal(t, uint64(0), OfferPrice(990000, 100))       // free
	assert.Equal(t, uint64(670), OfferPrice(999, 33))         // discount floored like checkout: 999 - 329
	assert.Equal(t, uint64(990000), OfferPrice(990000, -5))   // clamped
	assert.Equal(t, uint64(0), OfferPrice(990000, 150))       // clamped
	a := Price(999, DiscountPercent, 33, 0)
	assert.Equal(t, a.Subtotal-a.Discount, OfferPrice(999, 33), "same rounding as the invoice")
}

func TestParsePercent(t *testing.T) {
	for _, c := range []struct {
		raw  string
		want int
	}{{"50", 50}, {" 0 ", 0}, {"100", 100}} {
		n, ok := parsePercent(c.raw)
		assert.True(t, ok, c.raw)
		assert.Equal(t, c.want, n)
	}
	for _, raw := range []string{"", "half", "101", "-1", "12.5"} {
		_, ok := parsePercent(raw)
		assert.False(t, ok, raw)
	}
}

func TestCountdown(t *testing.T) {
	now := time.Date(2026, 9, 29, 19, 38, 0, 0, time.UTC)
	end := now.Add(5*24*time.Hour + 14*time.Hour + 22*time.Minute + 59*time.Second)
	secs := secondsLeft(now, end)
	assert.Equal(t, int64(5*86400+14*3600+22*60+59), secs)
	b, err := countdownJSON(secs).MarshalJSON()
	assert.NoError(t, err)
	assert.JSONEq(t, `{"days":5,"hours":14,"minutes":22}`, string(b))
	assert.Equal(t, int64(0), secondsLeft(end, now))
}
