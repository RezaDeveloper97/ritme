package aicredits

import (
	"database/sql"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// maxNote bounds the note of a manual adjustment (ai_credit_grants.note is varchar(255)).
const maxNote = 255

func (h *Handlers) userID(c fiber.Ctx) (uint64, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return 0, httpadmin.NotFound("User")
	}
	found, err := h.q.AdminAiCreditUserExists(c.Context(), id)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, httpadmin.NotFound("User")
	}
	return id, nil
}

func balanceJSON(b credits.Balance) *jsonx.OrderedMap {
	var period, endsAt any
	if b.Active {
		period = b.Period
		endsAt = httpadmin.Time(sql.NullTime{Time: b.PeriodEndsAt, Valid: true})
	}
	return jsonx.Obj(
		"active", b.Active,
		"period", period,
		"period_ends_at", endsAt,
		"plan_total", b.PlanTotal,
		"plan_remaining", b.PlanRemaining,
		"extra", b.Extra,
		"extra_locked", !b.Active && b.Extra > 0,
		"spendable", b.Spendable(),
	)
}

// grantJSON is one bucket. `current` says whether it counts now: a plan / trial bucket while its period runs, an
// extra / admin bucket always. `valid_until` is the end of the bucket's period (null for extra / admin).
func grantJSON(g *store.AdminUserAiCreditGrantsRow, at time.Time) *jsonx.OrderedMap {
	var until any
	current := true
	switch g.Kind {
	case credits.KindPlan:
		current = g.SubscriptionStatus.Valid && (g.SubscriptionStatus.String == "active" || g.SubscriptionStatus.String == "canceled") &&
			g.SubscriptionStartsAt.Valid && !g.SubscriptionStartsAt.Time.After(at) &&
			g.SubscriptionEndsAt.Valid && g.SubscriptionEndsAt.Time.After(at)
		until = httpadmin.Time(g.SubscriptionEndsAt)
	case credits.KindTrial:
		current = g.TrialStartedAt.Valid && !g.TrialStartedAt.Time.After(at) && g.TrialEndsAt.Valid && g.TrialEndsAt.Time.After(at)
		until = httpadmin.Time(g.TrialEndsAt)
	}
	return jsonx.Obj(
		"id", g.ID,
		"kind", g.Kind,
		"amount", g.Amount,
		"remaining", g.Remaining,
		"current", current,
		"valid_until", until,
		"subscription_id", nullID(g.SubscriptionID),
		"trial_id", nullID(g.TrialID),
		"invoice_id", nullID(g.InvoiceID),
		"note", httpadmin.NullString(g.Note),
		"created_at", httpadmin.Time(g.CreatedAt),
	)
}

func (h *Handlers) userCreditsJSON(c fiber.Ctx, userID uint64) (*jsonx.OrderedMap, error) {
	at := now(c)
	bal, err := h.credits.Balance(c.Context(), userID, at)
	if err != nil {
		return nil, err
	}
	grants, err := h.q.AdminUserAiCreditGrants(c.Context(), userID)
	if err != nil {
		return nil, err
	}
	spends, err := h.q.AdminUserAiCreditSpends(c.Context(), userID)
	if err != nil {
		return nil, err
	}
	adjs, err := h.q.AdminUserAiCreditAdjustments(c.Context(), nullU64(userID))
	if err != nil {
		return nil, err
	}
	gj := make([]*jsonx.OrderedMap, 0, len(grants))
	for i := range grants {
		gj = append(gj, grantJSON(&grants[i], at))
	}
	sj := make([]*jsonx.OrderedMap, 0, len(spends))
	for _, s := range spends {
		sj = append(sj, jsonx.Obj("id", s.ID, "feature", s.Feature, "credits", s.Credits, "plan_credits", s.PlanCredits,
			"extra_credits", s.ExtraCredits, "status", s.Status, "created_at", httpadmin.Time(s.CreatedAt)))
	}
	aj := make([]*jsonx.OrderedMap, 0, len(adjs))
	for _, a := range adjs {
		var details any
		if a.Details.Valid {
			details = form.Raw(a.Details.V)
		}
		aj = append(aj, jsonx.Obj("id", a.ID, "admin", jsonx.Obj("id", nullID(a.AdminID), "name", httpadmin.NullString(a.AdminName)),
			"note", httpadmin.NullString(a.Note), "details", details, "created_at", httpadmin.Time(a.CreatedAt)))
	}
	return jsonx.Obj(
		"user_id", userID,
		"balance", balanceJSON(bal),
		"grants", jsonx.List(gj),
		"spends", jsonx.List(sj),
		"adjustments", jsonx.List(aj),
		"limits", jsonx.Obj("max_delta", credits.MaxAdminGrant, "max_note", maxNote),
	), nil
}

// ShowUserCredits is GET /users/:id/ai-credits: the balance (credits.Service.Balance — it creates the running
// period's bucket when missing, like every balance read), the user's buckets (newest first, ≤ 100), the last 30 uses
// and the last 20 manual adjustments.
func (h *Handlers) ShowUserCredits(c fiber.Ctx) error {
	uid, err := h.userID(c)
	if err != nil {
		return err
	}
	body, err := h.userCreditsJSON(c, uid)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, body)
}

// AdjustUserCredits is POST /users/:id/ai-credits/adjust {delta, note} (super). A positive delta gifts extra credits
// (credits.Service.AdminGrant: a never-expiring `admin` bucket carrying the note); a negative one takes up to |delta|
// from the user's extra credits only (AdminDeduct; never the period's) — `taken` says how many were actually taken.
// Writes a plus_admin_actions row `credits.adjust` (details {delta, taken, grant_id}) and the audit line.
func (h *Handlers) AdjustUserCredits(c fiber.Ctx) error {
	uid, err := h.userID(c)
	if err != nil {
		return err
	}
	limit := strconv.Itoa(credits.MaxAdminGrant)
	data, err := form.Validate(c, validation.Rules{
		validation.F("delta", "required|integer|min:-"+limit+"|max:"+limit, validation.ParseRule("not_in:0")),
		validation.F("note", "required|string|max:"+strconv.Itoa(maxNote)),
	})
	if err != nil {
		return err
	}
	delta := form.Int(data, "delta", 0)
	note := strings.TrimSpace(httpadmin.String(data, "note"))
	if note == "" {
		return httpadmin.FieldError("note", form.Msg(c, "validation.required", "note"))
	}
	at := now(c)
	var grantID uint64
	var taken uint32
	if delta > 0 {
		grantID, err = h.credits.AdminGrant(c.Context(), uid, uint32(delta), note, at) //nolint:gosec // G115: validated ≤ MaxAdminGrant
	} else {
		taken, err = h.credits.AdminDeduct(c.Context(), uid, uint32(-delta), at) //nolint:gosec // G115: validated ≥ -MaxAdminGrant
	}
	if err != nil {
		return err
	}
	granted := uint32(0)
	if delta > 0 {
		granted = uint32(delta) //nolint:gosec // G115: validated
	}
	details := jsonx.Obj("delta", delta, "granted", granted, "taken", taken)
	if grantID > 0 {
		details.Set("grant_id", grantID)
	}
	if err := h.record(c, h.q, ledger{name: "credits.adjust", targetType: "user", targetID: uid, userID: uid,
		note: note, details: details}, slog.Int64("delta", delta), slog.Uint64("taken", uint64(taken))); err != nil {
		return err
	}
	body, err := h.userCreditsJSON(c, uid)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("delta", delta, "granted", granted, "taken", taken, "grant_id", nullGrant(grantID),
		"credits", body), "Credits adjusted.")
}

func nullGrant(id uint64) any {
	if id == 0 {
		return nil
	}
	return id
}
