package payments

import (
	"bytes"
	"errors"
	"html/template"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/httpx"
)

const defaultTimeout = 15 * time.Second

// noStore are the headers of every payment redirect and the TEST page: never cached, never sent as a Referer (the
// URL carries the payment id), never framed or indexed.
func noStore(c fiber.Ctx) {
	c.Set("Cache-Control", "no-store")
	c.Set("Referrer-Policy", "no-referrer")
	c.Set("X-Robots-Tag", "noindex, nofollow")
	c.Set("X-Content-Type-Options", "nosniff")
}

func seeOther(c fiber.Ctx, location string) error {
	noStore(c)
	c.Set("Location", location)
	return c.SendStatus(fiber.StatusSeeOther)
}

// Return is GET /api/v1/payments/{provider}/return: the browser lands here from the gateway. It changes nothing
// (CSRF-safe) — it validates the parameters and 303-redirects to an allow-listed web-app page with
// ?reference&authority&status for the web app to call the domain's authenticated verify. `next` is honored only
// when it is on the allow-list (scheme, host and path), otherwise the first allowed page is used: no open redirect.
func (g *Gateway) Return(c fiber.Ctx) error {
	if c.Params("provider") != g.Name() {
		return httpx.NotFound()
	}
	q := map[string]string{}
	for k, v := range c.Queries() {
		q[strings.ToLower(k)] = v
	}
	target := g.returns.Resolve(q["next"])
	if target == "" {
		return httpx.NotFound()
	}
	authority, status := g.p.ReturnParams(q)
	out := url.Values{}
	if ref := q["reference"]; referencePattern.MatchString(ref) {
		out.Set("reference", ref)
	}
	if authorityPattern.MatchString(authority) {
		out.Set("authority", authority)
	}
	if strings.EqualFold(strings.TrimSpace(status), "OK") {
		out.Set("status", "OK")
	} else {
		out.Set("status", "NOK")
	}
	return seeOther(c, target+"?"+out.Encode())
}

// FakeHandlers serve the fake provider's TEST page. Mounted only when PAYMENT_PROVIDER=fake (never in production).
type FakeHandlers struct{ fake *Fake }

// FakeHandlersFor returns the TEST page handlers when g runs the fake provider, nil otherwise.
func FakeHandlersFor(g *Gateway) *FakeHandlers {
	if g == nil {
		return nil
	}
	f, ok := g.p.(*Fake)
	if !ok {
		return nil
	}
	return &FakeHandlers{fake: f}
}

// pageHeaders lock the TEST page down: no scripts, no external resources, never framed.
func pageHeaders(c fiber.Ctx) {
	noStore(c)
	c.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'")
	c.Set("X-Frame-Options", "DENY")
	c.Set("Content-Type", "text/html; charset=utf-8")
}

// Page is GET /api/v1/payments/fake/pay/{authority}: the TEST gateway page (amount, reference, success / fail).
func (h *FakeHandlers) Page(c fiber.Ctx) error {
	authority := c.Params("authority")
	pageHeaders(c)
	if !authorityPattern.MatchString(authority) {
		return c.Status(fiber.StatusNotFound).SendString(renderPage(pageData{NotFound: true}))
	}
	p, ok, err := h.fake.Payment(c, authority)
	if err != nil {
		return err
	}
	if !ok {
		return c.Status(fiber.StatusNotFound).SendString(renderPage(pageData{NotFound: true}))
	}
	return c.SendString(renderPage(pageData{Payment: p, Amount: groupDigits(p.AmountRials), Pending: p.Status == FakePending}))
}

// Decide is POST /api/v1/payments/fake/pay/{authority} (form decision=success|fail): records the tester's choice
// and 303-redirects to the API's return URL. Only the fake's own state changes; no money and no domain row.
func (h *FakeHandlers) Decide(c fiber.Ctx) error {
	authority := c.Params("authority")
	pageHeaders(c)
	decision := c.FormValue("decision")
	if !authorityPattern.MatchString(authority) || (decision != "success" && decision != "fail") {
		return c.Status(fiber.StatusUnprocessableEntity).SendString(renderPage(pageData{NotFound: true}))
	}
	next, err := h.fake.Decide(c, authority, decision == "success")
	if errors.Is(err, ErrInvalidRequest) {
		return c.Status(fiber.StatusNotFound).SendString(renderPage(pageData{NotFound: true}))
	}
	if err != nil {
		return err
	}
	return seeOther(c, next)
}

type pageData struct {
	Payment  FakePayment
	Amount   string
	Pending  bool
	NotFound bool
}

func renderPage(d pageData) string {
	var b bytes.Buffer
	if err := pageTmpl.Execute(&b, d); err != nil {
		return "TEST payment page unavailable"
	}
	return b.String()
}

// groupDigits formats rials as 1,234,000.
func groupDigits(n uint64) string {
	s := strconv.FormatUint(n, 10)
	var out []byte
	for i := range len(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}

// pageTmpl is the TEST gateway page. Static test copy (fake provider only, never shown in production).
var pageTmpl = template.Must(template.New("fake").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex"><title>TEST payment — Ritme</title>
<style>
:root{color-scheme:light dark;--bg:#f6f3fb;--card:#fff;--fg:#1f1a2b;--muted:#6b6478;--ok:#0f8a6c;--bad:#c2334d;--warn:#b45309}
@media (prefers-color-scheme:dark){:root{--bg:#15121c;--card:#211c2b;--fg:#f1edf7;--muted:#a59fb3}}
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:var(--bg);color:var(--fg);font:16px/1.5 system-ui,sans-serif;padding:16px;box-sizing:border-box}
main{width:100%;max-width:420px;background:var(--card);border-radius:20px;padding:24px;box-shadow:0 8px 30px rgba(0,0,0,.08)}
.badge{display:inline-block;background:var(--warn);color:#fff;font-weight:700;letter-spacing:.08em;border-radius:999px;padding:4px 12px;font-size:13px}
h1{font-size:20px;margin:16px 0 4px}p{margin:4px 0;color:var(--muted)}dl{margin:20px 0;display:grid;grid-template-columns:auto 1fr;gap:8px 16px}
dt{color:var(--muted)}dd{margin:0;font-weight:600;word-break:break-all}form{display:flex;gap:12px;margin-top:20px}
button{flex:1;border:0;border-radius:14px;padding:14px;font:inherit;font-weight:700;color:#fff;cursor:pointer}
.ok{background:var(--ok)}.bad{background:var(--bad)}
</style></head><body><main>
<span class="badge">TEST — FAKE GATEWAY</span>
{{if .NotFound}}<h1>Payment not found</h1><p>This test payment does not exist or has expired.</p>
{{else}}<h1>Test payment</h1><p>No real money moves. Choose the outcome to simulate.</p>
<dl><dt>Amount</dt><dd>{{.Amount}} IRR</dd><dt>Reference</dt><dd>{{.Payment.Reference}}</dd><dt>Item</dt><dd>{{.Payment.Description}}</dd><dt>Status</dt><dd>{{.Payment.Status}}</dd></dl>
{{if .Pending}}<form method="post"><button class="ok" name="decision" value="success" type="submit">Simulate success</button><button class="bad" name="decision" value="fail" type="submit">Simulate failure</button></form>
{{else}}<form method="post"><button class="ok" name="decision" value="success" type="submit">Back to Ritme</button></form>{{end}}
{{end}}</main></body></html>`))
