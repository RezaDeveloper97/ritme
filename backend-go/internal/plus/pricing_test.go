package plus

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/plus/store"
)

func TestPrice_CheckoutArtboard(t *testing.T) {
	// nbl_Prem_Checkout: 3 months 237,000 toman, RITME10 −23,700, VAT 10 % on the rest.
	a := Price(2370000, DiscountPercent, 10, 1000)
	assert.Equal(t, Amounts{Subtotal: 2370000, Discount: 237000, VATRateBps: 1000, VAT: 213300, Total: 2346300}, a)
}

func TestPrice(t *testing.T) {
	cases := []struct {
		name             string
		kind             string
		value            uint64
		vat              uint32
		discount, vatAmt uint64
		total            uint64
		subtotal         uint64
	}{
		{"no discount", "", 0, 1000, 0, 99000, 1089000, 990000},
		{"amount", DiscountAmount, 100000, 1000, 100000, 89000, 979000, 990000},
		{"amount capped at price", DiscountAmount, 5000000, 1000, 990000, 0, 0, 990000},
		{"percent 100", DiscountPercent, 100, 1000, 990000, 0, 0, 990000},
		{"percent over 100 is capped", DiscountPercent, 150, 1000, 990000, 0, 0, 990000},
		{"vat rounds down", "", 0, 900, 0, 89, 1088, 999},
		{"zero vat", "", 0, 0, 0, 0, 990000, 990000},
		{"unknown kind ignored", "bogus", 50, 1000, 0, 99000, 1089000, 990000},
	}
	for _, c := range cases {
		a := Price(c.subtotal, c.kind, c.value, c.vat)
		assert.Equal(t, c.discount, a.Discount, c.name)
		assert.Equal(t, c.vatAmt, a.VAT, c.name)
		assert.Equal(t, c.total, a.Total, c.name)
	}
}

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, civildate.Tehran)
	if err != nil {
		panic(err)
	}
	return t
}

func code(mut func(*store.PlusDiscountCode)) store.PlusDiscountCode {
	c := store.PlusDiscountCode{ID: 1, Code: "RITME10", Kind: DiscountPercent, Value: 10, IsActive: true,
		PerUserLimit: sql.NullInt32{Int32: 1, Valid: true}}
	if mut != nil {
		mut(&c)
	}
	return c
}

func reason(err error) string {
	var de *DiscountError
	if errors.As(err, &de) {
		return de.Reason
	}
	return ""
}

func TestCheckDiscount(t *testing.T) {
	now := at("2026-09-23 10:00:00")
	ids := func(v ...uint64) db.NullRawJSON {
		b, _ := json.Marshal(v)
		return db.NullRawJSON{V: b, Valid: true}
	}
	cases := []struct {
		name        string
		c           store.PlusDiscountCode
		plan        uint64
		total, mine int64
		want        string
	}{
		{"ok", code(nil), 2, 0, 0, ""},
		{"inactive", code(func(c *store.PlusDiscountCode) { c.IsActive = false }), 2, 0, 0, DiscountInvalid},
		{"bad kind", code(func(c *store.PlusDiscountCode) { c.Kind = "x" }), 2, 0, 0, DiscountInvalid},
		{"zero value", code(func(c *store.PlusDiscountCode) { c.Value = 0 }), 2, 0, 0, DiscountInvalid},
		{"not started", code(func(c *store.PlusDiscountCode) { c.StartsAt = sql.NullTime{Time: now.Add(time.Second), Valid: true} }), 2, 0, 0, DiscountNotStarted},
		{"started exactly now", code(func(c *store.PlusDiscountCode) { c.StartsAt = sql.NullTime{Time: now, Valid: true} }), 2, 0, 0, ""},
		{"expires exactly now", code(func(c *store.PlusDiscountCode) { c.ExpiresAt = sql.NullTime{Time: now, Valid: true} }), 2, 0, 0, DiscountExpired},
		{"expires later", code(func(c *store.PlusDiscountCode) { c.ExpiresAt = sql.NullTime{Time: now.Add(time.Second), Valid: true} }), 2, 0, 0, ""},
		{"plan restricted", code(func(c *store.PlusDiscountCode) { c.PlanIds = ids(1, 3) }), 2, 0, 0, DiscountPlan},
		{"plan allowed", code(func(c *store.PlusDiscountCode) { c.PlanIds = ids(1, 2) }), 2, 0, 0, ""},
		{"plan list unreadable", code(func(c *store.PlusDiscountCode) { c.PlanIds = db.NullRawJSON{V: []byte(`"x"`), Valid: true} }), 2, 0, 0, DiscountPlan},
		{"global limit reached", code(func(c *store.PlusDiscountCode) { c.MaxRedemptions = sql.NullInt32{Int32: 5, Valid: true} }), 2, 5, 0, DiscountExhausted},
		{"global limit not reached", code(func(c *store.PlusDiscountCode) { c.MaxRedemptions = sql.NullInt32{Int32: 5, Valid: true} }), 2, 4, 0, ""},
		{"per user used", code(nil), 2, 1, 1, DiscountUsed},
		{"per user unlimited", code(func(c *store.PlusDiscountCode) { c.PerUserLimit = sql.NullInt32{} }), 2, 9, 9, ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, reason(CheckDiscount(c.c, c.plan, now, c.total, c.mine)), c.name)
	}
}

func plan(months uint16, price uint64) store.PlusPlan {
	return store.PlusPlan{DurationMonths: months, PriceRials: price}
}

func TestMonthlyAndSavings_PlansArtboard(t *testing.T) {
	// nbl_Prem_Plans: 99,000 / 79,000 (٪۲۰) / 65,000 (٪۳۴) toman per month.
	plans := []store.PlusPlan{plan(1, 990000), plan(3, 2370000), plan(6, 3900000)}
	base := BaseMonthly(plans)
	assert.Equal(t, uint64(990000), base)
	assert.Equal(t, []uint64{990000, 790000, 650000}, []uint64{MonthlyPrice(plans[0]), MonthlyPrice(plans[1]), MonthlyPrice(plans[2])})
	assert.Equal(t, []int{0, 20, 34}, []int{SavingsPercent(plans[0], base), SavingsPercent(plans[1], base), SavingsPercent(plans[2], base)})

	override := plans[1]
	override.MonthlyDisplayRials = sql.NullInt64{Int64: 800000, Valid: true}
	assert.Equal(t, uint64(800000), MonthlyPrice(override))
	assert.Equal(t, 0, SavingsPercent(plan(3, 3000000), base), "a dearer plan saves nothing")
	assert.Zero(t, BaseMonthly(nil))
}

func TestAddMonths(t *testing.T) {
	assert.Equal(t, at("2026-12-23 10:00:00"), AddMonths(at("2026-09-23 10:00:00"), 3))
	assert.Equal(t, at("2027-02-28 08:30:00"), AddMonths(at("2027-01-31 08:30:00"), 1), "no overflow into March")
	assert.Equal(t, at("2028-02-29 08:30:00"), AddMonths(at("2027-08-31 08:30:00"), 6), "leap year")
	assert.Equal(t, at("2027-03-23 10:00:00"), AddMonths(at("2026-09-23 10:00:00"), 6))
}

func TestPeriod(t *testing.T) {
	assert.Equal(t, "2026-09-01", PeriodStart(at("2026-09-30 23:59:59")).String())
	assert.Equal(t, at("2026-10-01 00:00:00"), resetsAt(at("2026-09-30 23:59:59")))
	// 21:00 UTC on Sept 30 is already Oct 1 in Tehran.
	assert.Equal(t, "2026-10-01", PeriodStart(time.Date(2026, 9, 30, 21, 0, 0, 0, time.UTC)).String())
}

func TestDaysLeft(t *testing.T) {
	now := at("2026-09-23 10:00:00")
	assert.Equal(t, 7, daysLeft(now, now.AddDate(0, 0, 7)))
	assert.Equal(t, 1, daysLeft(now, now.Add(time.Minute)))
	assert.Equal(t, 0, daysLeft(now, now))
	assert.Equal(t, 0, daysLeft(now, now.Add(-time.Hour)))
}

func TestFakeGateway(t *testing.T) {
	g := FakeGateway{}
	ctx := context.Background()
	s, err := g.Create(ctx, PaymentRequest{Reference: "RPABC", AmountRials: 1000, CallbackURL: "http://x.test/fa/plus/return?from=checkout"})
	require.NoError(t, err)
	assert.Equal(t, "FAKE-RPABC", s.Authority)
	u, err := url.Parse(s.RedirectURL)
	require.NoError(t, err)
	assert.Equal(t, "/fa/plus/return", u.Path)
	assert.Equal(t, url.Values{"from": {"checkout"}, "reference": {"RPABC"}, "authority": {"FAKE-RPABC"}, "status": {"OK"}}, u.Query())

	ok, err := g.Verify(ctx, VerifyRequest{Authority: s.Authority, AmountRials: 1000, Callback: map[string]string{"status": "ok"}})
	require.NoError(t, err)
	assert.Equal(t, VerifyResult{Paid: true, RefID: "FAKE-REF-RPABC", AmountRials: 1000}, ok)
	for _, cb := range []map[string]string{nil, {"status": "NOK"}, {}} {
		res, err := g.Verify(ctx, VerifyRequest{Authority: s.Authority, AmountRials: 1000, Callback: cb})
		require.NoError(t, err)
		assert.False(t, res.Paid)
	}
	res, _ := g.Verify(ctx, VerifyRequest{Authority: "OTHER-1", Callback: map[string]string{"status": "OK"}})
	assert.False(t, res.Paid, "not a fake authority")
}

func TestNewReference(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		r, err := newReference()
		require.NoError(t, err)
		assert.Regexp(t, `^RP[A-Z2-7]{16}$`, r)
		assert.False(t, seen[r])
		seen[r] = true
	}
}

func TestLangFiles(t *testing.T) {
	for _, locale := range []string{"fa", "en"} {
		for _, k := range []string{"messages.trial_started", "errors.payment_unavailable", "discount.expired", "messages.validation_failed"} {
			got := T(k, locale)
			assert.NotEqual(t, "plus."+k, got, "%s %s", locale, k)
			assert.NotEmpty(t, got)
		}
		assert.NotEmpty(t, attributes(locale))
	}
	assert.Equal(t, T("errors.trial_used", "en"), T("errors.trial_used", "xx"), "unknown locale falls back to English")
}
