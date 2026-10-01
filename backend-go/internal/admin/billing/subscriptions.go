package billing

import (
	"database/sql"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Subscription statuses as the admin sees them (stored active | canceled | refunded; expired is derived).
const (
	SubActive   = "active"
	SubCanceled = "canceled"
	SubExpired  = "expired"
	SubRefunded = "refunded"
	SubAll      = "all"
)

// MaxExtendDays caps one extension.
const MaxExtendDays = 365

// CodeSubscriptionNotActive is the 422 error_code of extending a period that is not running (expired / refunded).
const CodeSubscriptionNotActive = "subscription_not_active"

var (
	epoch   = time.Date(1970, 1, 2, 0, 0, 0, 0, civildate.Tehran)
	farAway = time.Date(9999, 1, 1, 0, 0, 0, 0, civildate.Tehran)
)

func effectiveStatus(status string, endsAt, at time.Time) string {
	if (status == SubActive || status == SubCanceled) && !endsAt.After(at) {
		return SubExpired
	}
	return status
}

// digits maps Persian / Arabic-Indic digits to ASCII (admins paste numbers in either).
func digits(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		}
		return r
	}, s)
}

// ListSubscriptions is GET /plus/subscriptions?status=all|active|canceled|expired|refunded&plan_id=&from=&to=&q=
// (q = part of the mobile number; from/to = start date range, Tehran days) — newest first. Mobiles are masked.
func (h *Handlers) ListSubscriptions(c fiber.Ctx) error {
	at := now(c)
	status := c.Query("status")
	p := store.AdminListPlusSubscriptionsParams{StatusA: "%", StatusB: "%", EndsAfter: epoch.Add(-time.Hour), EndsUntil: farAway}
	switch status {
	case SubActive, SubCanceled:
		p.StatusA, p.StatusB, p.EndsAfter = status, status, at
	case SubExpired:
		p.StatusA, p.StatusB, p.EndsUntil = SubActive, SubCanceled, at
	case SubRefunded:
		p.StatusA, p.StatusB = SubRefunded, SubRefunded
	default:
		status = SubAll
	}
	planID, _ := strconv.ParseUint(c.Query("plan_id"), 10, 64)
	p.PlanMin, p.PlanMax = sql.NullInt64{Int64: 0, Valid: true}, sql.NullInt64{Int64: math.MaxInt64, Valid: true}
	if planID > 0 {
		p.PlanMin, p.PlanMax = nullU64(planID), nullU64(planID)
	}
	lo, hi, from, to := dateRange(c)
	p.StartsFrom, p.StartsTo = lo, hi
	search := strings.TrimSpace(digits(c.Query("q")))
	p.Mobile = sql.NullString{String: form.Contains(search), Valid: true}

	total, err := h.q.AdminCountPlusSubscriptions(c.Context(), store.AdminCountPlusSubscriptionsParams{
		StatusA: p.StatusA, StatusB: p.StatusB, EndsAfter: p.EndsAfter, EndsUntil: p.EndsUntil,
		PlanMin: p.PlanMin, PlanMax: p.PlanMax, StartsFrom: p.StartsFrom, StartsTo: p.StartsTo, Mobile: p.Mobile,
	})
	if err != nil {
		return err
	}
	pg := httpadmin.PageOf(c)
	p.Limit, p.Offset = int32(pg.PerPage), int32(min(pg.Offset(), 1<<30)) //nolint:gosec // G115: bounded by PageOf
	rows, err := h.q.AdminListPlusSubscriptions(c.Context(), p)
	if err != nil {
		return err
	}
	counts, err := h.q.AdminPlusSubscriptionCounts(c.Context(), store.AdminPlusSubscriptionCountsParams{Now: at})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		daysLeft := 0
		if r.EndsAt.After(at) && r.Status != SubRefunded {
			daysLeft = int(math.Ceil(r.EndsAt.Sub(at).Hours() / 24))
		}
		items = append(items, jsonx.Obj(
			"id", r.ID,
			"user", userJSON(r.UserID, r.UserName, r.UserMobile),
			"plan", planRef(r.PlanID, r.PlanCode, r.PlanTitle),
			"invoice_reference", httpadmin.NullString(r.InvoiceReference),
			"status", r.Status,
			"effective_status", effectiveStatus(r.Status, r.EndsAt, at),
			"source", r.Source,
			"starts_at", iso(r.StartsAt),
			"ends_at", iso(r.EndsAt),
			"days_left", daysLeft,
			"auto_renew", r.AutoRenew,
			"canceled_at", httpadmin.Time(r.CanceledAt),
			"created_at", httpadmin.Time(r.CreatedAt),
		))
	}
	page := httpadmin.Page(items, pg, int(total))
	filters := jsonx.Obj("status", status, "plan_id", nil, "from", from, "to", to, "q", search)
	if planID > 0 {
		filters.Set("plan_id", planID)
	}
	page.Set("filters", filters)
	page.Set("counts", jsonx.Obj("active", counts.Active, "canceled", counts.Canceled, "expired", counts.Expired,
		"refunded", counts.Refunded))
	return httpadmin.OK(c, page)
}

// ExtendSubscription is POST /plus/subscriptions/:id/extend {days 1–365, note} (super): adds days to a running
// period (active or canceled, not ended). The user's queued periods that start at or after its old end move back by
// the same days, so periods never overlap. One ledger row (subscription.extend) with the note.
func (h *Handlers) ExtendSubscription(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Subscription")
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("days", "required|integer|min:1|max:"+strconv.Itoa(MaxExtendDays)),
		validation.F("note", "required|string|min:3|max:500"),
	})
	if err != nil {
		return err
	}
	days := int(u64(data, "days")) //nolint:gosec // G115: validated 1–365
	note := strings.TrimSpace(httpadmin.String(data, "note"))
	at := now(c)
	var newEnd time.Time
	var shifted int64
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		sub, err := q.AdminLockPlusSubscription(c.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			return httpadmin.NotFound("Subscription")
		}
		if err != nil {
			return err
		}
		if effectiveStatus(sub.Status, sub.EndsAt, at) == SubExpired || sub.Status == SubRefunded {
			return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeSubscriptionNotActive,
				"Only a running subscription can be extended.")
		}
		stamp := httpadmin.DBTime(httpadmin.Now(c))
		newEnd = sub.EndsAt.AddDate(0, 0, days)
		if shifted, err = q.AdminShiftQueuedPlusSubscriptions(c.Context(), store.AdminShiftQueuedPlusSubscriptionsParams{
			Days: days, Now: stamp, UserID: sub.UserID, ID: sub.ID, OldEnd: sub.EndsAt,
		}); err != nil {
			return err
		}
		if err := q.AdminSetPlusSubscriptionEnd(c.Context(), store.AdminSetPlusSubscriptionEndParams{EndsAt: newEnd, Now: stamp, ID: sub.ID}); err != nil {
			return err
		}
		return h.record(c, q, action{name: "subscription.extend", targetType: "subscription", targetID: sub.ID,
			userID: sub.UserID, days: days, note: note,
			details: jsonx.Obj("ends_at", jsonx.Obj("from", iso(sub.EndsAt), "to", iso(newEnd)), "queued_shifted", shifted)})
	})
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("id", id, "ends_at", iso(newEnd), "days", days, "queued_shifted", shifted),
		"Subscription extended.")
}
