package companionhome

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/family/projections"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// CalendarMaxSpan is the widest window one GET /companion/calendar answers (a month grid with its leading and
// trailing weeks fits).
const CalendarMaxSpan = 62

// ForecastReader reads the owner's likely period days for a link's companion (internal/companion/shared).
type ForecastReader interface {
	PeriodForecastFor(ctx context.Context, link companion.Link, from, to civildate.Date, now time.Time) (projections.PeriodForecast, error)
}

// Calendar is GET /companion/calendar?from=Y-m-d&to=Y-m-d — the partner overlay of a male (companion) account's
// calendar: per active partner/spouse link, the days in the window her period is likely to fall on, today onwards.
// period_days is null unless she turned on «روزهای احتمالی پریود» for this link *and* the cycle grant is readable
// (family.Decide), so «not shared» and «nothing predicted in this window» ([]) stay distinct. Each read is audited
// as a cycle read. His own medications and appointments come from /care/* as for any account.
func (h *Handlers) Calendar(c fiber.Ctx) error {
	companion.NoStore(c)
	u := auth.CurrentUser(c)
	if u == nil {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	ctx := c.Context()
	ok, err := h.accounts.IsCompanion(ctx, u.ID)
	if err != nil {
		return err
	}
	if !ok {
		return companion.NonCompanionForbidden(locale)
	}
	clk := clock.FromContext(c, h.base)
	now := clk.Now()
	from, to, err := calendarWindow(validation.Query(c), locale, now)
	if err != nil {
		return err
	}

	svc := companion.NewService(h.conn, clk, companion.Options{})
	all, err := svc.ListForCompanion(ctx, u.ID)
	if err != nil {
		return err
	}
	links := make([]companion.Link, 0, len(all))
	ownerIDs := make([]uint64, 0, len(all))
	for _, l := range all {
		if l.Type != companion.TypeParent { // a teen's parent link is never a partner (CB-TEEN-01)
			links = append(links, l)
			ownerIDs = append(ownerIDs, l.OwnerID)
		}
	}
	names, err := svc.Names(ctx, ownerIDs...)
	if err != nil {
		return err
	}
	partners := make([]*jsonx.OrderedMap, 0, len(links))
	for _, l := range links {
		days, err := h.periodDays(ctx, svc, l, u.ID, from, to, now)
		if err != nil {
			return err
		}
		partners = append(partners, jsonx.Obj(
			"link_id", l.ID,
			"partner_name", nullable(names[l.OwnerID]),
			"period_days", days,
		))
	}
	return httpx.OK(c, jsonx.Obj("from", from.String(), "to", to.String(), "partners", partners))
}

// periodDays is the link's forecast list, or nil (JSON null) when it is not shared. The read is audited only when
// the owner consented and the cycle grant reads, i.e. when data may actually be returned.
func (h *Handlers) periodDays(ctx context.Context, svc *companion.Service, l companion.Link, viewerID uint64,
	from, to civildate.Date, now time.Time) (any, error) {
	if h.forecast == nil || !l.Preferences.PeriodDays || !l.Grants.Of(companion.SectionCycle).CanRead() {
		return nil, nil
	}
	if err := svc.Audit(ctx, l.OwnerID, viewerID, l.ID, companion.SectionCycle, companion.ActionRead); err != nil {
		return nil, err
	}
	f, err := h.forecast.PeriodForecastFor(ctx, l, from, to, now)
	if errors.Is(err, companion.ErrProjectionDenied) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return f.Days, nil
}

// calendarWindow validates ?from / ?to (Y-m-d, both optional: from defaults to today, to to from + 41 days) and
// refuses a reversed or over-wide window with a 422.
func calendarWindow(query any, locale string, now time.Time) (civildate.Date, civildate.Date, error) {
	v := validation.Make(lang.Default(), locale, query, validation.Rules{
		validation.F("from", "nullable", "date_format:Y-m-d"),
		validation.F("to", "nullable", "date_format:Y-m-d"),
	}, validation.Now(now))
	if v.Fails() {
		return civildate.Date{}, civildate.Date{}, httpx.Fail(fiber.StatusUnprocessableEntity,
			companion.T("messages.validation_failed", locale), "errors", v.ErrorBag())
	}
	from := civildate.InTehran(now)
	if d, ok := queryDate(query, "from"); ok {
		from = d
	}
	to := from.AddDays(41)
	if d, ok := queryDate(query, "to"); ok {
		to = d
	}
	if to.Before(from) || from.DiffDays(to) > CalendarMaxSpan {
		return civildate.Date{}, civildate.Date{}, httpx.Fail(fiber.StatusUnprocessableEntity,
			companion.T("messages.validation_failed", locale),
			"errors", jsonx.Obj("to", []string{T("calendar.range_invalid", locale)}))
	}
	return from, to, nil
}

func queryDate(query any, key string) (civildate.Date, bool) {
	m, ok := query.(interface{ Get(string) (any, bool) })
	if !ok {
		return civildate.Date{}, false
	}
	raw, ok := m.Get(key)
	if !ok || raw == nil {
		return civildate.Date{}, false
	}
	s, ok := raw.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return civildate.Date{}, false
	}
	d, err := civildate.Parse(strings.TrimSpace(s))
	return d, err == nil
}
