package analysis

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/plus"
)

// Handlers serve /api/v1/analysis/*. Every report reads only the authenticated user's own rows (no id
// parameter), and none of them logs anything: no health value reaches a log line.
type Handlers struct {
	cycle     *cycleservice.Service
	logs      *healthlog.Service
	plus      *plus.Service
	bundles   *i18n.TranslationStore
	languages *i18n.Registry
	clock     clock.Clock
	labs      LabsHub
}

// NewHandlers wires the handlers; base is the fallback clock (clock.Middleware's request clock wins).
func NewHandlers(cycle *cycleservice.Service, logs *healthlog.Service, plusSvc *plus.Service,
	bundles *i18n.TranslationStore, languages *i18n.Registry, base clock.Clock,
) *Handlers {
	return &Handlers{cycle: cycle, logs: logs, plus: plusSvc, bundles: bundles, languages: languages, clock: base}
}

func fieldFail(c fiber.Ctx, field, key string, params map[string]string) error {
	all := map[string]string{"attribute": field}
	for k, v := range params {
		all[k] = v
	}
	e := httpx.NewValidationError()
	e.Add(field, lang.Default().Trans(key, all, i18n.Locale(c)))
	return e
}

func (h *Handlers) copy(c fiber.Ctx) *Copy {
	locale, def := i18n.Locale(c), h.languages.DefaultCode(c.Context())
	return NewCopy(h.bundles.NamespaceMessages(locale, Namespace, def),
		h.bundles.NamespaceMessages(locale, healthlog.TaxonomyNamespace, def))
}

// request is the loaded user context of one report.
type request struct {
	userID uint64
	now    time.Time
	in     *Input
}

// load reads the snapshot and the day logs from logsFrom(range) to today. plusOnly skips the day logs
// for a user without plus.deep_analysis: a locked report computes nothing.
func (h *Handlers) load(c fiber.Ctx, rng func(today civildate.Date) Range, logsFrom func(Range) civildate.Date, plusOnly bool) (*request, error) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return nil, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	ctx := c.Context()
	now := clock.FromContext(c, h.clock).Now()
	today := civildate.InTehran(now)
	r := rng(today)
	ent, err := h.plus.Entitlement(ctx, uid, plus.DeepAnalysis, now)
	if err != nil {
		return nil, err
	}
	in := &Input{Today: today, Range: r, DeepAnalysis: ent.Allowed, Copy: h.copy(c), Days: map[civildate.Date]*Day{}}
	sn, err := h.cycle.Load(ctx, uid, today, today, today)
	if err != nil {
		return nil, err
	}
	in.Histories, in.Profile = sn.Histories, sn.EngineProfile()
	in.NoCycleNudge = sn.LifeMode == enums.LifeModeMenopause
	if p := sn.Profile; p != nil {
		if p.Height.Valid {
			in.HeightCM = int(p.Height.Int16)
		}
		if p.Birthday.Valid {
			in.Birthday = p.Birthday.Date
		}
	}
	if !plusOnly || in.DeepAnalysis {
		if in.Days, err = h.days(ctx, uid, logsFrom(r), today); err != nil {
			return nil, err
		}
	}
	return &request{userID: uid, now: now, in: in}, nil
}

func (h *Handlers) days(ctx context.Context, uid uint64, from, to civildate.Date) (map[civildate.Date]*Day, error) {
	rows, err := h.logs.Range(ctx, uid, from, to)
	if err != nil {
		return nil, err
	}
	in := make([]DayEntries, len(rows))
	for i, r := range rows {
		in[i] = DayEntries{Date: r.Date, Entries: r.Entries}
	}
	return BuildDays(in), nil
}

// rangeQuery validates ?range= (nullable, one of RangeKeys).
func rangeQuery(c fiber.Ctx) (string, error) {
	q := validation.Query(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), q, validation.Rules{
		validation.F("range", "nullable", "string", validation.In(RangeKeys...)),
	})
	if v.Fails() {
		return "", v.Errors()
	}
	if raw, ok := q.Get("range"); ok {
		if s, isStr := raw.(string); isStr && s != "" {
			return s, nil
		}
	}
	return DefaultRange, nil
}

func lookback(r Range) civildate.Date { return r.From.AddDays(-LookbackDays) }

// report loads the range's Input and renders build's body under {range, …}.
func (h *Handlers) report(c fiber.Ctx, build func(in *Input) *jsonx.OrderedMap) error {
	key, err := rangeQuery(c)
	if err != nil {
		return err
	}
	req, err := h.load(c, func(today civildate.Date) Range { return NewRange(key, today) }, lookback, false)
	if err != nil {
		return err
	}
	return httpx.OK(c, withRange(req.in.Range, build(req.in)))
}

func withRange(r Range, body *jsonx.OrderedMap) *jsonx.OrderedMap {
	out := jsonx.Obj("range", r.JSON())
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		out.Set(k, v)
	}
	return out
}

// LabsHub fills the hub's `labs` card (bloom B-N6-06, internal/labs.Service.HubLabs): ready when the user has a
// verified lab; data = counts, date span and the highlighted marker's last values.
type LabsHub interface {
	HubLabs(ctx context.Context, userID uint64, locale, defaultLocale string) (bool, any, error)
}

// WithLabs wires the labs card (nil = the card stays empty, as before B-N6-06).
func (h *Handlers) WithLabs(l LabsHub) *Handlers {
	h.labs = l
	return h
}

// Summary is GET /analysis/summary: the hub (free users get every free card; Plus cards are locked).
func (h *Handlers) Summary(c fiber.Ctx) error {
	key, err := rangeQuery(c)
	if err != nil {
		return err
	}
	req, err := h.load(c, func(today civildate.Date) Range { return NewRange(key, today) }, lookback, false)
	if err != nil {
		return err
	}
	body := BuildSummary(req.in).JSON(req.in.Copy)
	if h.labs != nil && req.in.DeepAnalysis {
		ready, data, err := h.labs.HubLabs(c, req.userID, i18n.Locale(c), h.languages.DefaultCode(c.Context()))
		if err != nil {
			return err
		}
		if sections, ok := body.Get("sections"); ok {
			if m, ok := sections.(*jsonx.OrderedMap); ok {
				m.Set("labs", plusSection(true, func() (bool, any) { return ready, data }).JSON())
			}
		}
	}
	return httpx.OK(c, withRange(req.in.Range, body))
}

// Cycle is GET /analysis/cycle.
func (h *Handlers) Cycle(c fiber.Ctx) error {
	return h.report(c, func(in *Input) *jsonx.OrderedMap { return BuildCycle(in).JSON() })
}

// Period is GET /analysis/period.
func (h *Handlers) Period(c fiber.Ctx) error {
	return h.report(c, func(in *Input) *jsonx.OrderedMap { return BuildPeriod(in).JSON(in.Copy) })
}

// Symptoms is GET /analysis/symptoms.
func (h *Handlers) Symptoms(c fiber.Ctx) error {
	return h.report(c, func(in *Input) *jsonx.OrderedMap { return BuildSymptoms(in).JSON(in.Copy) })
}

// Body is GET /analysis/body.
func (h *Handlers) Body(c fiber.Ctx) error {
	return h.report(c, func(in *Input) *jsonx.OrderedMap { return BuildBody(in).JSON() })
}

// Correlations is GET /analysis/correlations: Plus (plus.deep_analysis). A free user gets 200 with
// `locked: true` and no data — the client shows the Plus lock, nothing is computed.
func (h *Handlers) Correlations(c fiber.Ctx) error {
	key, err := rangeQuery(c)
	if err != nil {
		return err
	}
	req, err := h.load(c, func(today civildate.Date) Range { return NewRange(key, today) }, lookback, true)
	if err != nil {
		return err
	}
	in := req.in
	sec := plusSection(in.DeepAnalysis, func() (bool, any) {
		r := BuildCorrelations(in)
		ready := false
		for _, it := range r.Items {
			ready = ready || it.Status == CorrReady
		}
		return ready, r.JSON(in.Copy)
	})
	return httpx.OK(c, jsonx.Obj(
		"range", in.Range.JSON(),
		"not_causal", true,
		"plus", sec.Plus,
		"locked", sec.Locked,
		"ready", sec.Ready,
		"data", sec.Data,
	))
}

var ymPattern = regexp.MustCompile(`^(\d{4})-(0[1-9]|1[0-2])$`)

// Monthly is GET /analysis/monthly/{ym}[?calendar=gregorian|jalali]: ym is YYYY-MM in that calendar
// (default gregorian); the running month is clipped to today, a future month is refused.
func (h *Handlers) Monthly(c fiber.Ctx) error {
	q := validation.Query(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), q, validation.Rules{
		validation.F("calendar", "nullable", "string", validation.In(CalendarGregorian, CalendarJalali)),
	})
	if v.Fails() {
		return v.Errors()
	}
	calendar := CalendarGregorian
	if raw, ok := q.Get("calendar"); ok {
		if s, isStr := raw.(string); isStr && s != "" {
			calendar = s
		}
	}
	m := ymPattern.FindStringSubmatch(c.Params("ym"))
	if m == nil {
		return fieldFail(c, "ym", "validation.date_format", map[string]string{"format": "Y-m"})
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])

	var cur, prev Month
	req, err := h.load(c, func(today civildate.Date) Range {
		cur, prev = MonthOf(year, month, calendar, today)
		return Range{Key: "month", From: cur.From, To: cur.To}
	}, func(Range) civildate.Date { return prev.From.AddDays(-6) }, false)
	if err != nil {
		return err
	}
	if cur.From.After(req.in.Today) {
		return fieldFail(c, "ym", "validation.before_or_equal", map[string]string{"date": req.in.Today.String()})
	}
	pdf, err := h.plus.Entitlement(c.Context(), req.userID, plus.PDFShare, req.now)
	if err != nil {
		return err
	}
	return httpx.OK(c, BuildMonthly(req.in, cur, prev, pdf.Allowed).JSON(req.in.Copy))
}
