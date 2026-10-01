package billing_test

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/billing"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/plus"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// stubRefunder is the payment gateway's refund port.
type stubRefunder struct {
	mu    sync.Mutex
	name  string
	err   error
	calls []payments.RefundRequest
}

func (s *stubRefunder) Name() string { return s.name }

func (s *stubRefunder) Refund(_ context.Context, req payments.RefundRequest) (payments.RefundResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, req)
	if s.err != nil {
		return payments.RefundResult{}, s.err
	}
	return payments.RefundResult{RefundID: "RFD-" + strconv.Itoa(len(s.calls)), AmountRials: req.AmountRials}, nil
}

type env struct {
	*admintest.Env
	svc      *plus.Service
	refunder *stubRefunder
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := admintest.New(t)
	svc := plus.NewService(e.DB, config.Plus{VATRateBps: 1000, TrialDays: 7, InvoiceTTL: 30 * time.Minute}, plus.FakeGateway{}, admintest.Quiet)
	ref := &stubRefunder{name: "fake"}
	billing.New(e.DB, svc, ref, admintest.Quiet).Routes(e.Route(), e.Kit)
	return &env{Env: e, svc: svc, refunder: ref}
}

func tehranNow() time.Time { return time.Now().In(civildate.Tehran).Truncate(time.Second) }

func (e *env) user(mobile, name string) uint64 {
	e.T.Helper()
	id, err := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES (?, ?, NOW(), NOW())", mobile, name).LastInsertId()
	require.NoError(e.T, err)
	return uint64(id) //nolint:gosec // test ids
}

func (e *env) planID(code string) uint64 {
	e.T.Helper()
	return uint64(e.Int("SELECT id FROM plus_plans WHERE code = ?", code)) //nolint:gosec // test ids
}

// paidInvoice writes a paid invoice with a receipt (card PAN included, to prove it never leaves the API) and the
// subscription period it bought.
func (e *env) paidInvoice(userID, planID uint64, ref string, total uint64, starts, ends time.Time) (invoiceID, subID uint64) {
	e.T.Helper()
	at := tehranNow()
	res := e.Exec(`INSERT INTO plus_invoices (reference, user_id, plan_id, duration_months, status, currency, subtotal_rials,
		discount_rials, vat_rate_bps, vat_rials, total_rials, gateway, authority, expires_at, paid_at, created_at, updated_at)
		VALUES (?, ?, ?, 1, 'paid', 'IRR', ?, 0, 0, 0, ?, 'fake', ?, ?, ?, ?, ?)`,
		ref, userID, planID, total, total, "FAKE-"+ref, at.Add(30*time.Minute), at, at, at)
	id, err := res.LastInsertId()
	require.NoError(e.T, err)
	e.Exec(`INSERT INTO plus_receipts (invoice_id, user_id, gateway, ref_id, card_pan, amount_rials, paid_at, created_at, updated_at)
		VALUES (?, ?, 'fake', ?, '603799******1234', ?, ?, ?, ?)`, id, userID, "BANK-"+ref, total, at, at, at)
	sres := e.Exec(`INSERT INTO plus_subscriptions (user_id, plan_id, invoice_id, status, source, starts_at, ends_at, auto_renew, created_at, updated_at)
		VALUES (?, ?, ?, 'active', 'purchase', ?, ?, 1, ?, ?)`, userID, planID, id, starts, ends, at, at)
	sid, err := sres.LastInsertId()
	require.NoError(e.T, err)
	return uint64(id), uint64(sid) //nolint:gosec // test ids
}

func (e *env) actions(name string) int {
	return e.Int("SELECT COUNT(*) FROM plus_admin_actions WHERE action = ?", name)
}

func id(v uint64) string { return strconv.FormatUint(v, 10) }

func TestGuards(t *testing.T) {
	e := newEnv(t)
	u := e.user("09121234567", "Sara")
	plan := e.planID("plus_1m")
	inv, sub := e.paidInvoice(u, plan, "RPGUARD1", 990000, tehranNow().Add(-time.Hour), tehranNow().AddDate(0, 1, 0))

	reads := []string{"/plus/plans", "/plus/plans/" + id(plan), "/plus/discount-codes", "/plus/settings",
		"/plus/subscriptions", "/plus/payments", "/plus/payments/" + id(inv)}
	for _, path := range reads {
		r := e.Anonymous().Get(path)
		assert.Equal(t, 401, r.Status, path)
		assert.Equal(t, 200, e.As(admintest.EditorID).Get(path).Status, "any active admin reads "+path)
	}

	editor := e.As(admintest.EditorID)
	writes := []struct{ method, path string }{
		{fiber.MethodPost, "/plus/plans"},
		{fiber.MethodPut, "/plus/plans/" + id(plan)},
		{fiber.MethodDelete, "/plus/plans/" + id(plan)},
		{fiber.MethodPost, "/plus/discount-codes"},
		{fiber.MethodPut, "/plus/settings"},
		{fiber.MethodPost, "/plus/subscriptions/" + id(sub) + "/extend"},
		{fiber.MethodPost, "/plus/payments/" + id(inv) + "/refund"},
	}
	body := map[string]any{"mode": "manual", "note": "refund it", "days": 5, "trial_offer_percent": 10}
	for _, w := range writes {
		r := editor.JSON(w.method, w.path, body)
		assert.Equal(t, 403, r.Status, "money-changing writes need a super admin: "+w.path)
	}
	assert.Equal(t, "paid", e.String("SELECT status FROM plus_invoices WHERE id = ?", inv))
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM plus_admin_actions"))

	noToken := e.As(admintest.SuperID)
	noToken.CSRF = ""
	r := noToken.JSON(fiber.MethodPost, "/plus/payments/"+id(inv)+"/refund", map[string]any{"mode": "manual", "note": "x refund"})
	assert.Equal(t, 419, r.Status, "CSRF")
	assert.Equal(t, "paid", e.String("SELECT status FROM plus_invoices WHERE id = ?", inv))

	super := e.As(admintest.SuperID)
	for _, path := range []string{"/plus/plans/999999", "/plus/plans/abc", "/plus/discount-codes/999999", "/plus/payments/999999"} {
		assert.Equal(t, 404, super.Get(path).Status, path)
	}
	assert.Equal(t, 404, super.JSON(fiber.MethodPost, "/plus/subscriptions/999999/extend", map[string]any{"days": 3, "note": "gift days"}).Status)
	assert.Equal(t, 404, super.JSON(fiber.MethodPost, "/plus/payments/999999/refund", map[string]any{"mode": "manual", "note": "gift"}).Status)
}

func TestPlansCRUD(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)

	r := c.Get("/plus/plans")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 3, "the three seeded plans")
	first := r.Items()[0].(map[string]any)
	assert.Equal(t, "plus_1m", first["code"])
	assert.EqualValues(t, 990000, first["monthly_price_rials"])

	r = c.JSON(fiber.MethodPost, "/plus/plans", map[string]any{"code": "plus_1m", "title": map[string]any{"fa": "یک"},
		"duration_months": 0, "price_rials": 100})
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"code", "duration_months", "price_rials"} {
		assert.Contains(t, r.Errors(), f)
	}

	r = c.JSON(fiber.MethodPost, "/plus/plans", map[string]any{"code": "plus_12m",
		"title": map[string]any{"fa": "۱۲ ماهه", "en": "12 months"}, "badge": map[string]any{"fa": "بصرفه"},
		"duration_months": 12, "price_rials": 6900000, "is_active": false})
	require.Equal(t, 201, r.Status, r.Body)
	p := r.Obj("plan")
	assert.Equal(t, false, p["is_active"])
	assert.EqualValues(t, 575000, p["monthly_price_rials"])
	assert.EqualValues(t, 4, p["sort_order"], "appended after the seeds")
	newID := uint64(p["id"].(float64))
	assert.Equal(t, 1, e.actions("plan.create"))

	r = c.JSON(fiber.MethodPut, "/plus/plans/"+id(newID), map[string]any{"title": map[string]any{"fa": "۱۲ ماهه"},
		"duration_months": 12, "price_rials": 5900000, "is_active": true})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 5900000, r.Obj("plan")["price_rials"])
	assert.Equal(t, map[string]any{"fa": "بصرفه"}, r.Obj("plan")["badge"], "absent badge keeps its value")
	assert.Equal(t, 1, e.actions("plan.update"))
	assert.Contains(t, e.String("SELECT details FROM plus_admin_actions WHERE action = 'plan.update'"), `"price_rials":{"from":6900000,"to":5900000}`)
	assert.Equal(t, int(admintest.SuperID), e.Int("SELECT admin_id FROM plus_admin_actions WHERE action = 'plan.update'"))

	// The public plan list (checkout side) sees the new active plan.
	plans, err := e.svc.Plans(t.Context())
	require.NoError(t, err)
	assert.Len(t, plans, 4)

	r = c.JSON(fiber.MethodDelete, "/plus/plans/"+id(newID), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, true, r.Data()["deleted"], "an unreferenced plan is deleted")
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM plus_plans WHERE id = ?", newID))

	u := e.user("09120000002", "")
	plan := e.planID("plus_3m")
	e.paidInvoice(u, plan, "RPPLAN01", 2370000, tehranNow(), tehranNow().AddDate(0, 3, 0))
	r = c.JSON(fiber.MethodDelete, "/plus/plans/"+id(plan), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, false, r.Data()["deleted"])
	assert.Equal(t, true, r.Data()["deactivated"], "a referenced plan is deactivated instead")
	assert.Equal(t, 0, e.Int("SELECT is_active FROM plus_plans WHERE id = ?", plan))
	assert.Equal(t, 1, e.actions("plan.deactivate"))
}

func TestDiscountCodes(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	plan := e.planID("plus_3m")

	r := c.JSON(fiber.MethodPost, "/plus/discount-codes", map[string]any{"code": "x", "kind": "percent", "value": 150,
		"plan_ids": []any{999999}})
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"code", "value", "plan_ids"} {
		assert.Contains(t, r.Errors(), f)
	}

	r = c.JSON(fiber.MethodPost, "/plus/discount-codes", map[string]any{"code": "yalda30", "kind": "percent", "value": 30,
		"max_redemptions": 100, "plan_ids": []any{plan}, "starts_at": "2026-01-01", "expires_at": "2099-01-01 00:00"})
	require.Equal(t, 201, r.Status, r.Body)
	d := r.Obj("discount_code")
	assert.Equal(t, "YALDA30", d["code"])
	assert.EqualValues(t, 1, d["per_user_limit"], "default one use per user")
	assert.Equal(t, []any{float64(plan)}, d["plan_ids"])
	assert.Equal(t, "active", d["state"])
	did := uint64(d["id"].(float64))

	r = c.JSON(fiber.MethodPost, "/plus/discount-codes", map[string]any{"code": "Yalda30", "kind": "amount", "value": 1000})
	require.Equal(t, 422, r.Status, "codes are unique case-insensitively")
	assert.Contains(t, r.Errors(), "code")

	// Checkout honours the code (and the plan scope).
	u := e.user("09120000003", "")
	q, err := e.svc.Preview(t.Context(), u, plan, "yalda30", tehranNow())
	require.NoError(t, err)
	assert.EqualValues(t, 711000, q.Amounts.Discount)

	r = c.JSON(fiber.MethodPut, "/plus/discount-codes/"+id(did), map[string]any{"kind": "percent", "value": 40, "per_user_limit": nil})
	require.Equal(t, 200, r.Status, r.Body)
	d = r.Obj("discount_code")
	assert.EqualValues(t, 40, d["value"])
	assert.Nil(t, d["per_user_limit"], "null = unlimited per user")
	assert.EqualValues(t, 100, d["max_redemptions"], "absent keeps the stored limit")

	// A used code is deactivated, not deleted.
	inv, _ := e.paidInvoice(u, plan, "RPDISC01", 1000000, tehranNow(), tehranNow().AddDate(0, 3, 0))
	e.Exec("UPDATE plus_invoices SET discount_code_id = ?, discount_code = 'YALDA30' WHERE id = ?", did, inv)
	r = c.Get("/plus/discount-codes/" + id(did))
	assert.EqualValues(t, 1, r.Obj("discount_code")["uses"].(map[string]any)["paid"])
	r = c.JSON(fiber.MethodDelete, "/plus/discount-codes/"+id(did), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, true, r.Data()["deactivated"])
	assert.Equal(t, 0, e.Int("SELECT is_active FROM plus_discount_codes WHERE id = ?", did))

	r = c.Get("/plus/discount-codes?status=inactive&q=yal")
	require.Len(t, r.Items(), 1)
	assert.Equal(t, "inactive", r.Items()[0].(map[string]any)["state"])
	assert.Equal(t, 1, e.actions("discount.create"))
	assert.Equal(t, 1, e.actions("discount.update"))
	assert.Equal(t, 1, e.actions("discount.deactivate"))
}

func TestSettingsDriveCheckout(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)

	r := c.Get("/plus/settings")
	require.Equal(t, 200, r.Status, r.Body)
	s := r.Obj("settings")
	assert.EqualValues(t, 50, s["trial_offer_percent"])
	assert.EqualValues(t, 1000, s["vat_rate_bps"])
	assert.Nil(t, s["vat_override_bps"])
	assert.Equal(t, "env", s["vat_source"])

	r = c.JSON(fiber.MethodPut, "/plus/settings", map[string]any{"trial_offer_percent": 101, "vat_rate_bps": 20000})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "trial_offer_percent")
	assert.Contains(t, r.Errors(), "vat_rate_bps")

	r = c.JSON(fiber.MethodPut, "/plus/settings", map[string]any{"trial_offer_percent": 30, "vat_rate_bps": 900})
	require.Equal(t, 200, r.Status, r.Body)
	s = r.Obj("settings")
	assert.EqualValues(t, 30, s["trial_offer_percent"])
	assert.EqualValues(t, 900, s["vat_rate_bps"])
	assert.Equal(t, "admin", s["vat_source"])
	assert.Equal(t, 1, e.actions("settings.update"))

	u := e.user("09120000004", "")
	q, err := e.svc.Preview(t.Context(), u, e.planID("plus_1m"), "", tehranNow())
	require.NoError(t, err)
	assert.EqualValues(t, 900, q.Amounts.VATRateBps, "checkout reads the admin VAT")
	assert.EqualValues(t, 89100, q.Amounts.VAT)

	pct, err := e.svc.TrialOfferPercent(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 30, pct)

	r = c.JSON(fiber.MethodPut, "/plus/settings", map[string]any{"trial_offer_percent": 30, "vat_rate_bps": nil})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 1000, r.Obj("settings")["vat_rate_bps"], "null clears the override → env")
	assert.Equal(t, 2, e.actions("settings.update"))

	r = c.JSON(fiber.MethodPut, "/plus/settings", map[string]any{"trial_offer_percent": 30})
	require.Equal(t, 200, r.Status)
	assert.Equal(t, 2, e.actions("settings.update"), "no change, no ledger row")
}

func TestSubscriptionsListMasksAndFilters(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	now := tehranNow()
	a := e.user("09121234567", "Sara")
	b := e.user("09359876543", "Mina")
	p1, p3 := e.planID("plus_1m"), e.planID("plus_3m")
	_, active := e.paidInvoice(a, p1, "RPSUB001", 990000, now.AddDate(0, 0, -5), now.AddDate(0, 0, 25))
	_, expired := e.paidInvoice(b, p3, "RPSUB002", 2370000, now.AddDate(0, -4, 0), now.AddDate(0, 0, -2))

	r := c.Get("/plus/subscriptions")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 2)
	row := r.Items()[1].(map[string]any)
	assert.EqualValues(t, active, row["id"])
	assert.Equal(t, map[string]any{"id": float64(a), "name": "Sara", "mobile": "0912•••4567"}, row["user"])
	assert.NotContains(t, string(r.Raw), "09121234567", "lists never carry the full mobile")
	assert.Equal(t, "active", row["effective_status"])
	assert.EqualValues(t, 25, row["days_left"])
	assert.Equal(t, map[string]any{"active": float64(1), "canceled": float64(0), "expired": float64(1), "refunded": float64(0)}, r.Data()["counts"])

	r = c.Get("/plus/subscriptions?status=expired")
	require.Len(t, r.Items(), 1)
	assert.EqualValues(t, expired, r.Items()[0].(map[string]any)["id"])
	assert.Equal(t, "expired", r.Items()[0].(map[string]any)["effective_status"])

	r = c.Get("/plus/subscriptions?q=" + "۹۸۷۶")
	require.Len(t, r.Items(), 1, "mobile search, Persian digits too")
	assert.EqualValues(t, expired, r.Items()[0].(map[string]any)["id"])

	r = c.Get("/plus/subscriptions?plan_id=" + id(p1))
	require.Len(t, r.Items(), 1)
	r = c.Get("/plus/subscriptions?from=" + now.AddDate(0, 0, -10).Format("2006-01-02"))
	require.Len(t, r.Items(), 1, "start date range")
	assert.EqualValues(t, active, r.Items()[0].(map[string]any)["id"])
}

func TestExtendSubscription(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	now := tehranNow()
	u := e.user("09121234567", "")
	plan := e.planID("plus_1m")
	end := now.AddDate(0, 0, 10)
	_, cur := e.paidInvoice(u, plan, "RPEXT001", 990000, now.AddDate(0, 0, -20), end)
	_, queued := e.paidInvoice(u, plan, "RPEXT002", 990000, end, end.AddDate(0, 1, 0))
	_, old := e.paidInvoice(u, plan, "RPEXT003", 990000, now.AddDate(0, -3, 0), now.AddDate(0, -2, 0))

	r := c.JSON(fiber.MethodPost, "/plus/subscriptions/"+id(cur)+"/extend", map[string]any{"days": 0})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "days")
	assert.Contains(t, r.Errors(), "note", "a reason is required")

	r = c.JSON(fiber.MethodPost, "/plus/subscriptions/"+id(cur)+"/extend", map[string]any{"days": 7, "note": "Support ticket #12"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 1, r.Data()["queued_shifted"])
	var got time.Time
	require.NoError(t, e.DB.QueryRowContext(t.Context(), "SELECT ends_at FROM plus_subscriptions WHERE id = ?", cur).Scan(&got))
	assert.True(t, got.Equal(end.AddDate(0, 0, 7)), got)
	require.NoError(t, e.DB.QueryRowContext(t.Context(), "SELECT starts_at FROM plus_subscriptions WHERE id = ?", queued).Scan(&got))
	assert.True(t, got.Equal(end.AddDate(0, 0, 7)), "the queued period moves back, no overlap")
	assert.Equal(t, 1, e.actions("subscription.extend"))
	assert.Equal(t, "Support ticket #12", e.String("SELECT note FROM plus_admin_actions WHERE action = 'subscription.extend'"))
	assert.Equal(t, 7, e.Int("SELECT days FROM plus_admin_actions WHERE action = 'subscription.extend'"))
	assert.Equal(t, int(u), e.Int("SELECT user_id FROM plus_admin_actions WHERE action = 'subscription.extend'"))

	r = c.JSON(fiber.MethodPost, "/plus/subscriptions/"+id(old)+"/extend", map[string]any{"days": 7, "note": "late gift"})
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, billing.CodeSubscriptionNotActive, r.Code())
}

func TestRefundThroughGateway(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	u := e.user("09121234567", "")
	plan := e.planID("plus_1m")
	inv, sub := e.paidInvoice(u, plan, "RPREF001", 1089000, tehranNow().Add(-time.Hour), tehranNow().AddDate(0, 1, 0))

	st, err := e.svc.Status(t.Context(), u, tehranNow())
	require.NoError(t, err)
	require.Equal(t, plus.TierPlus, st.Tier)

	r := c.Get("/plus/payments/" + id(inv))
	require.Equal(t, 200, r.Status, r.Body)
	pay := r.Obj("payment")
	assert.Equal(t, true, pay["refund"].(map[string]any)["gateway_available"])
	assert.NotContains(t, string(r.Raw), "603799", "card data never leaves the API")
	assert.NotContains(t, string(r.Raw), "card_pan")

	r = c.JSON(fiber.MethodPost, "/plus/payments/"+id(inv)+"/refund", map[string]any{"mode": "gateway"})
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, e.refunder.calls, 1)
	call := e.refunder.calls[0]
	assert.Equal(t, "FAKE-RPREF001", call.Authority)
	assert.Equal(t, "BANK-RPREF001", call.RefID)
	assert.EqualValues(t, 1089000, call.AmountRials, "the full receipt amount, never a client value")
	assert.Equal(t, "refunded", r.Obj("payment")["status"])
	assert.Equal(t, "refunded", e.String("SELECT status FROM plus_subscriptions WHERE id = ?", sub))
	assert.Equal(t, 1, e.actions("invoice.refund"))
	assert.Equal(t, "RFD-1", e.String("SELECT gateway_ref FROM plus_admin_actions WHERE action = 'invoice.refund'"))
	assert.Equal(t, 1089000, e.Int("SELECT amount_rials FROM plus_admin_actions WHERE action = 'invoice.refund'"))

	st, err = e.svc.Status(t.Context(), u, tehranNow())
	require.NoError(t, err)
	assert.Equal(t, plus.TierFree, st.Tier, "a refunded period no longer entitles")

	r = c.JSON(fiber.MethodPost, "/plus/payments/"+id(inv)+"/refund", map[string]any{"mode": "gateway"})
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, billing.CodeNotRefundable, r.Code(), "no double refund")
	assert.Len(t, e.refunder.calls, 1)
}

func TestRefundNotSupportedThenManual(t *testing.T) {
	e := newEnv(t)
	e.refunder.err = payments.ErrRefundNotSupported
	c := e.As(admintest.SuperID)
	u := e.user("09121234567", "")
	inv, sub := e.paidInvoice(u, e.planID("plus_1m"), "RPREF002", 1089000, tehranNow(), tehranNow().AddDate(0, 1, 0))

	r := c.JSON(fiber.MethodPost, "/plus/payments/"+id(inv)+"/refund", map[string]any{"mode": "gateway"})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, billing.CodeRefundNotSupported, r.Code())
	assert.Equal(t, "paid", e.String("SELECT status FROM plus_invoices WHERE id = ?", inv), "nothing changed")
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM plus_admin_actions"))

	r = c.JSON(fiber.MethodPost, "/plus/payments/"+id(inv)+"/refund", map[string]any{"mode": "manual"})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "note", "a manual refund needs a note")

	r = c.JSON(fiber.MethodPost, "/plus/payments/"+id(inv)+"/refund", map[string]any{"mode": "manual", "note": "Refunded from the Zarinpal panel"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "refunded", e.String("SELECT status FROM plus_invoices WHERE id = ?", inv))
	assert.Equal(t, "refunded", e.String("SELECT status FROM plus_subscriptions WHERE id = ?", sub))
	assert.Equal(t, "Refunded from the Zarinpal panel", e.String("SELECT note FROM plus_admin_actions WHERE action = 'invoice.refund_manual'"))
	actions := r.Obj("payment")["actions"].([]any)
	require.Len(t, actions, 1)
	assert.Equal(t, "invoice.refund_manual", actions[0].(map[string]any)["action"])

	e.refunder.err = payments.ErrRefundRejected
	inv2, _ := e.paidInvoice(u, e.planID("plus_1m"), "RPREF003", 1089000, tehranNow(), tehranNow().AddDate(0, 1, 0))
	r = c.JSON(fiber.MethodPost, "/plus/payments/"+id(inv2)+"/refund", map[string]any{"mode": "gateway"})
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, billing.CodeRefundRejected, r.Code())
}

func TestPaymentLog(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	a := e.user("09121234567", "Sara")
	b := e.user("09359876543", "")
	p := e.planID("plus_1m")
	e.paidInvoice(a, p, "RPLOG001", 1089000, tehranNow(), tehranNow().AddDate(0, 1, 0))
	inv, _ := e.paidInvoice(b, p, "RPLOG002", 500000, tehranNow(), tehranNow().AddDate(0, 1, 0))
	e.Exec("UPDATE plus_invoices SET status = 'refunded' WHERE id = ?", inv)

	r := c.Get("/plus/payments")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 2)
	assert.NotContains(t, string(r.Raw), "603799")
	assert.NotContains(t, string(r.Raw), "09121234567")
	first := r.Items()[0].(map[string]any)
	assert.Equal(t, "RPLOG002", first["reference"])
	assert.Equal(t, "BANK-RPLOG002", first["receipt"].(map[string]any)["ref_id"])
	assert.Equal(t, map[string]any{"paid_count": float64(1), "paid_rials": float64(1089000), "refunded_rials": float64(500000)}, r.Data()["summary"])
	assert.Equal(t, []any{"fake"}, r.Data()["gateways"])

	r = c.Get("/plus/payments?status=paid")
	require.Len(t, r.Items(), 1)
	r = c.Get("/plus/payments?q=bank-rplog001")
	require.Len(t, r.Items(), 1, "search by the bank reference")
	r = c.Get("/plus/payments?q=" + strings.ToLower("RPLOG002"))
	require.Len(t, r.Items(), 1, "search by the invoice reference")
	r = c.Get("/plus/payments?q=9876543")
	require.Len(t, r.Items(), 1, "search by mobile")
	r = c.Get("/plus/payments?gateway=zarinpal")
	assert.Empty(t, r.Items())
}
