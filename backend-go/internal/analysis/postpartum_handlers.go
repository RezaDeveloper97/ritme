package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/babylog"
	"github.com/ritme/backend-go/internal/children"
	childrenstore "github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/postpartum"
	postpartumstore "github.com/ritme/backend-go/internal/postpartum/store"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
	"github.com/ritme/backend-go/seeds/who"
)

// ErrorCodeNotPostpartum is the error_code of the 409 GET /analysis/postpartum answers outside postpartum mode.
const ErrorCodeNotPostpartum = "not_postpartum"

// PostpartumDeps are the reads of the postpartum hub (every one scoped to the authenticated user or to a child she
// owns).
type PostpartumDeps struct {
	Postpartum      *postpartum.Service
	PostpartumStore postpartumstore.Querier
	Pregnancy       pregstore.Querier
	Logs            *healthlog.Service
	Plus            *plus.Service
	Modules         *modules.Service
	ChildrenStore   childrenstore.Querier
	Children        *children.Service
	BabyLog         *babylog.Service
}

// PostpartumHandlers serve GET /api/v1/analysis/postpartum. No id parameter, nothing logged.
type PostpartumHandlers struct {
	d     PostpartumDeps
	clock clock.Clock
}

// NewPostpartumHandlers wires the handlers; base is the fallback clock (clock.Middleware's request clock wins).
func NewPostpartumHandlers(d PostpartumDeps, base clock.Clock) *PostpartumHandlers {
	return &PostpartumHandlers{d: d, clock: base}
}

// moduleOff is the modules.Gate answer (403 module_disabled, never 401).
func moduleOff(module string) error {
	return httpx.Fail(fiber.StatusForbidden, "This section is not available.", "error_code", modules.CodeDisabled, "module", module)
}

// Postpartum is GET /analysis/postpartum: 403 module_disabled when the postpartum module is off, 409 not_postpartum
// outside postpartum mode (or without a birth date, or a companion account).
func (h *PostpartumHandlers) Postpartum(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	ctx := c.Context()
	if !h.d.Modules.Enabled(ctx, modules.Postpartum) {
		return moduleOff(modules.Postpartum)
	}
	now := clock.FromContext(c, h.clock).Now()
	today := civildate.InTehran(now)
	st, err := h.d.Postpartum.State(ctx, uid)
	if err != nil {
		return err
	}
	if !st.Active() || st.Profile == nil {
		return httpx.Fail(fiber.StatusConflict, postpartum.T("messages.not_active", i18n.Locale(c)), "error_code", ErrorCodeNotPostpartum)
	}
	ent, err := h.d.Plus.Entitlement(ctx, uid, plus.DeepAnalysis, now)
	if err != nil {
		return err
	}
	in := &PostpartumInput{
		Today: today, Birth: st.Profile.BirthDate, DeepAnalysis: ent.Allowed,
		Recovery: map[civildate.Date]postpartum.Recovery{}, Weights: map[civildate.Date]float64{},
		ChildrenEnabled: h.d.Modules.Enabled(ctx, modules.Children),
	}
	if st.Profile.Breastfeeding.Valid {
		b := st.Profile.Breastfeeding.Bool
		in.Breastfeeding = &b
	}
	if err := h.load(ctx, uid, st.Profile, in, now, i18n.Locale(c)); err != nil {
		return err
	}
	return httpx.OK(c, BuildPostpartum(in))
}

func (h *PostpartumHandlers) load(ctx context.Context, uid uint64, p *postpartumstore.PostpartumProfile,
	in *PostpartumInput, now time.Time, locale string,
) error {
	in.PregnancyStart = h.pregnancyStart(ctx, uid, p)

	// EPDS: totals only (D-54), the checks since the birth.
	checks, err := h.d.Postpartum.History(ctx, uid, postpartum.HistoryLimit)
	if err != nil {
		return err
	}
	for _, ch := range checks {
		if !ch.TakenOn.Before(in.Birth) && !ch.TakenOn.After(in.Today) {
			in.Checks = append(in.Checks, ch)
		}
	}

	// Day log: the recovery slots (birth … today) and the weights (from before the pregnancy, for the baseline).
	from := in.PregnancyStart.AddDays(-BaselineLookbackDays)
	if in.Birth.Before(from) {
		from = in.Birth
	}
	weekly, err := h.d.Pregnancy.ListWeeklyLogs(ctx, uid)
	if err != nil {
		return fmt.Errorf("analysis postpartum: weekly logs: %w", err)
	}
	for i := range weekly {
		w := &weekly[i]
		if v, ok := decimal(w.Weight); ok && !w.LogDate.Before(from) && !w.LogDate.After(in.Today) {
			in.Weights[w.LogDate] = v
		}
	}
	rows, err := h.d.Logs.Range(ctx, uid, from, in.Today)
	if err != nil {
		return err
	}
	dayRows := make([]DayEntries, len(rows))
	for i, r := range rows {
		dayRows[i] = DayEntries{Date: r.Date, Entries: r.Entries}
		if !r.Date.Before(in.Birth) {
			in.Recovery[r.Date] = postpartum.RecoveryOf(r.Entries)
		}
	}
	for date, day := range BuildDays(dayRows) {
		if day.Weight != nil && *day.Weight > 0 {
			in.Weights[date] = *day.Weight // the day log wins over the weekly log
		}
	}
	// The red-flag rows (temperature, pad soaked within an hour, …) of the recovery sheet.
	logs, err := h.d.PostpartumStore.ExportPostpartumRecoveryLogs(ctx, uid)
	if err != nil {
		return fmt.Errorf("analysis postpartum: recovery logs: %w", err)
	}
	for _, row := range logs {
		if !row.LogDate.Before(in.Birth) && !row.LogDate.After(in.Today) {
			in.Recovery[row.LogDate] = in.Recovery[row.LogDate].WithLog(row)
		}
	}

	if in.ChildrenEnabled {
		return h.loadChild(ctx, uid, in, now, locale)
	}
	return nil
}

// pregnancyStart is day 0 of the pregnancy that ended in the birth: the linked pregnancy profile dated on the birth
// day, else the birth minus PostpartumPregnancyDays.
func (h *PostpartumHandlers) pregnancyStart(ctx context.Context, uid uint64, p *postpartumstore.PostpartumProfile) civildate.Date {
	estimate := p.BirthDate.AddDays(-PostpartumPregnancyDays)
	if !p.PregnancyProfileID.Valid {
		return estimate
	}
	preg, err := h.d.Pregnancy.GetProfileByUser(ctx, uid)
	if err != nil || int64(preg.ID) != p.PregnancyProfileID.Int64 { //nolint:gosec // G115: auto-increment id
		return estimate
	}
	if d, ok := v2.Resolve(&preg, p.BirthDate); ok && d.Start().Before(p.BirthDate) {
		return d.Start()
	}
	return estimate
}

// hubChild is the own child born closest to the birth (the first listed on a tie: twins), nil without one.
func hubChild(kids []childrenstore.Child, birth civildate.Date) *childrenstore.Child {
	var best *childrenstore.Child
	gap := func(c *childrenstore.Child) int {
		d := c.BirthDate.DiffDays(birth)
		if d < 0 {
			return -d
		}
		return d
	}
	for i := range kids {
		if best == nil || gap(&kids[i]) < gap(best) {
			best = &kids[i]
		}
	}
	return best
}

// loadChild reads the hub baby (an own child only — never a shared one): WHO placement, verdict, crossing hints and
// the last week of the baby log.
func (h *PostpartumHandlers) loadChild(ctx context.Context, uid uint64, in *PostpartumInput, now time.Time, locale string) error {
	kids, err := h.d.ChildrenStore.ListOwnedChildren(ctx, uid)
	if err != nil {
		return fmt.Errorf("analysis postpartum: children: %w", err)
	}
	c := hubChild(kids, in.Birth)
	if c == nil {
		return nil
	}
	ms, err := h.d.Children.Measurements(ctx, c.ID)
	if err != nil {
		return err
	}
	pts := children.Points(*c, ms)
	verdict := children.Verdict(*c, pts)
	child := &PostpartumChild{
		ID: c.ID, Name: c.Name, Verdict: children.VerdictJSON(verdict, locale),
		Placed: verdict != children.GrowthUnknown, Crossings: []any{},
	}
	if len(pts) > 0 {
		child.Latest = children.PointJSON(*c, pts[0], locale)
		for _, ind := range who.Indicators {
			if x, ok := children.SeriesJSON(*c, ind, pts, in.Today, locale).Get("crossing"); ok && x != nil {
				child.Crossings = append(child.Crossings, jsonx.Obj("indicator", string(ind), "hint", x))
			}
		}
	}
	in.Child = child
	days := in.weekDays()
	baby, err := h.d.BabyLog.Days(ctx, c.ID, days[0], len(days), now)
	if err != nil {
		return err
	}
	in.BabyDays = baby
	return nil
}
