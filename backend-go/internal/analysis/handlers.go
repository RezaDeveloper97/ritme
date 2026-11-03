package analysis

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
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
	labTrends LabTrends
	vitals    VitalsSource
	states    state.UserStates // v6 engine (nil = CYCLE_ENGINE v1, engine.go)
	cache     *Cache           // nil = every report computed (T-M10-28)
}

// WithEngine makes the reports read the v6 engine (CYCLE_ENGINE shadow|v6, T-M9-35, D-129); nil keeps v1 unchanged.
func (h *Handlers) WithEngine(states state.UserStates) *Handlers {
	h.states = states
	return h
}

// engineState is the user's v6 state on her day: the request's engine revision (from the auth query) on the shared
// provider (no query on an L1 hit), else Source.ForUser.
func (h *Handlers) engineState(ri *reqInfo) (*engine.State, error) {
	if src, ok := h.states.(state.Source); ok && src.P != nil && ri.hasRevs {
		st, err := src.P.For(ri.ctx, ri.uid, ri.revs.Engine, state.Params{Engine: engineparams.For(ri.today), FP: engineparams.Fingerprint(ri.today)})
		if err != nil {
			return nil, fmt.Errorf("analysis: engine state: %w", err)
		}
		return st, nil
	}
	st, err := h.states.ForUser(ri.ctx, ri.uid, ri.today)
	if err != nil {
		return nil, fmt.Errorf("analysis: engine state: %w", err)
	}
	return st, nil
}

// NewHandlers wires the handlers; base is the fallback clock (clock.Middleware's request clock wins).
func NewHandlers(cycle *cycleservice.Service, logs *healthlog.Service, plusSvc *plus.Service,
	bundles *i18n.TranslationStore, languages *i18n.Registry, base clock.Clock,
) *Handlers {
	return &Handlers{cycle: cycle, logs: logs, plus: plusSvc, bundles: bundles, languages: languages, clock: base}
}

// WithCache serves the reports through the analysis cache (T-M10-28: Redis cache-aside + ETag / 304); nil computes
// every request, as before.
func (h *Handlers) WithCache(c *Cache) *Handlers {
	h.cache = c
	return h
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

func (h *Handlers) copy(ri *reqInfo) *Copy {
	return NewCopy(h.bundles.NamespaceMessages(ri.locale, Namespace, ri.def),
		h.bundles.NamespaceMessages(ri.locale, healthlog.TaxonomyNamespace, ri.def))
}

// reqInfo is what one report request knows before it loads anything: the user, her clock and day, the locale, her
// revisions (from the auth query) and her Plus entitlements (one plus.Usage read). The warm job builds one without a
// Fiber request.
type reqInfo struct {
	ctx     context.Context
	uid     uint64
	now     time.Time
	today   civildate.Date
	locale  string
	def     string
	revs    auth.Revisions
	hasRevs bool
	ents    entitlements
}

// info reads the request's reqInfo.
func (h *Handlers) info(c fiber.Ctx) (*reqInfo, error) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return nil, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	clk := clock.FromContext(c, h.clock)
	now := clk.Now()
	today := civildate.InTehran(now)
	if h.states != nil {
		today = civildate.LocalToday(c, clk) // D-126
	}
	ri := &reqInfo{ctx: c.Context(), uid: uid, now: now, today: today, locale: i18n.Locale(c),
		def: requestDefaultCode(c, h.languages)}
	ri.revs, ri.hasRevs = auth.CurrentRevisions(c)
	return ri, h.loadEntitlements(ri)
}

// requestDefaultCode is the default language of the list the locale middleware already loaded for this request, so
// a cold config cache costs no second languages read (T-M9-49c); without the middleware (Bootstrap) it asks reg.
func requestDefaultCode(c fiber.Ctx, reg *i18n.Registry) string {
	if l := i18n.LanguagesOf(c); len(l) > 0 && (len(i18n.Bootstrap) == 0 || &l[0] != &i18n.Bootstrap[0]) {
		return l.DefaultCode()
	}
	return reg.DefaultCode(c.Context())
}

func (h *Handlers) loadEntitlements(ri *reqInfo) error {
	u, err := h.plus.Usage(ri.ctx, ri.uid, ri.now)
	if err != nil {
		return err
	}
	ri.ents = entitlementsOf(u)
	return nil
}

// key is the cache key of one report of the request.
func (ri *reqInfo) key(endpoint, rng string) CacheKey {
	return CacheKey{UserID: ri.uid, AnalysisRev: ri.revs.Analysis, EngineRev: ri.revs.Engine, Endpoint: endpoint,
		Range: rng, Plus: ri.ents.plusPart(), Today: ri.today, Locale: ri.locale, ConfigFP: ConfigFingerprint(ri.today)}
}

// serve answers a report through the cache (computed directly when the request carries no revisions).
func serve(c fiber.Ctx, cc *Cache, ri *reqInfo, endpoint, rng string, compute Compute) error {
	if !ri.hasRevs {
		cc = nil
	}
	return cc.Serve(c, ri.key(endpoint, rng), compute)
}

// load reads the snapshot and the day logs from logsFrom to today. plusOnly skips the day logs for a user without
// plus.deep_analysis: a locked report computes nothing.
func (h *Handlers) load(ri *reqInfo, r Range, logsFrom civildate.Date, plusOnly bool) (*Input, error) {
	ctx, uid, today := ri.ctx, ri.uid, ri.today
	in := &Input{Today: today, Range: r, DeepAnalysis: ri.ents.allowed(plus.DeepAnalysis), Copy: h.copy(ri),
		Days: map[civildate.Date]*Day{}}
	sn, err := h.cycle.Load(ctx, uid, today, today, today)
	if err != nil {
		return nil, err
	}
	in.Histories, in.Profile = sn.Histories, sn.EngineProfile()
	if h.states != nil {
		if in.Engine, err = h.engineState(ri); err != nil {
			return nil, err
		}
	}
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
		if in.Days, err = h.days(ctx, uid, logsFrom, today); err != nil {
			return nil, err
		}
		if err := loadVitals(ctx, h.vitals, uid, in, logsFrom, today); err != nil {
			return nil, err
		}
	}
	return in, nil
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

// report answers a ranged report: build's body under {range, …}, through the cache.
func (h *Handlers) report(c fiber.Ctx, endpoint string, build func(in *Input) *jsonx.OrderedMap) error {
	ri, err := h.info(c)
	if err != nil {
		return err
	}
	key, err := h.rangeQueryFor(c, ri)
	if err != nil {
		return err
	}
	return serve(c, h.cache, ri, endpoint, key, func(context.Context) (any, error) {
		r := NewRange(key, ri.today)
		in, err := h.load(ri, r, lookback(r), false)
		if err != nil {
			return nil, err
		}
		return withRange(in.Range, build(in)), nil
	})
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

// WithVitals wires the merged vitals readings (nil = the vitals averages read the log sheet only, as before B-N6-03).
func (h *Handlers) WithVitals(v VitalsSource) *Handlers {
	h.vitals = v
	return h
}

// EndpointSummary is the hub's cache endpoint (the one the warm job fills).
const EndpointSummary = "summary"

// Summary is GET /analysis/summary: the hub (free users get every free card; Plus cards are locked).
func (h *Handlers) Summary(c fiber.Ctx) error {
	ri, err := h.info(c)
	if err != nil {
		return err
	}
	key, err := h.rangeQueryFor(c, ri)
	if err != nil {
		return err
	}
	return serve(c, h.cache, ri, EndpointSummary, key, func(context.Context) (any, error) { return h.summary(ri, key) })
}

// summary is the hub's data for the range key.
func (h *Handlers) summary(ri *reqInfo, key string) (any, error) {
	r := NewRange(key, ri.today)
	in, err := h.load(ri, r, lookback(r), false)
	if err != nil {
		return nil, err
	}
	body := BuildSummary(in).JSON(in.Copy)
	if h.labs != nil && in.DeepAnalysis {
		ready, data, err := h.labs.HubLabs(ri.ctx, ri.uid, ri.locale, ri.def)
		if err != nil {
			return nil, err
		}
		if sections, ok := body.Get("sections"); ok {
			if m, ok := sections.(*jsonx.OrderedMap); ok {
				m.Set("labs", plusSection(true, func() (bool, any) { return ready, data }).JSON())
			}
		}
	}
	return withRange(in.Range, body), nil
}

// Cycle is GET /analysis/cycle.
func (h *Handlers) Cycle(c fiber.Ctx) error {
	return h.report(c, "cycle", func(in *Input) *jsonx.OrderedMap { return BuildCycle(in).JSON() })
}

// Period is GET /analysis/period.
func (h *Handlers) Period(c fiber.Ctx) error {
	return h.report(c, "period", func(in *Input) *jsonx.OrderedMap { return BuildPeriod(in).JSON(in.Copy) })
}

// Symptoms is GET /analysis/symptoms.
func (h *Handlers) Symptoms(c fiber.Ctx) error {
	return h.report(c, "symptoms", func(in *Input) *jsonx.OrderedMap { return BuildSymptoms(in).JSON(in.Copy) })
}

// Body is GET /analysis/body.
func (h *Handlers) Body(c fiber.Ctx) error {
	return h.report(c, "body", func(in *Input) *jsonx.OrderedMap { return BuildBody(in).JSON() })
}

// Correlations is GET /analysis/correlations: Plus (plus.deep_analysis). A free user gets 200 with
// `locked: true` and no data — the client shows the Plus lock, nothing is computed.
func (h *Handlers) Correlations(c fiber.Ctx) error {
	ri, err := h.info(c)
	if err != nil {
		return err
	}
	key, err := h.rangeQueryFor(c, ri)
	if err != nil {
		return err
	}
	return serve(c, h.cache, ri, "correlations", key, func(context.Context) (any, error) {
		r := NewRange(key, ri.today)
		in, err := h.load(ri, r, lookback(r), true)
		if err != nil {
			return nil, err
		}
		sec := plusSection(in.DeepAnalysis, func() (bool, any) {
			r := BuildCorrelations(in)
			ready := false
			for _, it := range r.Items {
				ready = ready || it.Status == CorrReady
			}
			return ready, r.JSON(in.Copy)
		})
		return jsonx.Obj(
			"range", in.Range.JSON(),
			"not_causal", true,
			"plus", sec.Plus,
			"locked", sec.Locked,
			"ready", sec.Ready,
			"data", sec.Data,
		), nil
	})
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

	ri, err := h.info(c)
	if err != nil {
		return err
	}
	cur, prev := MonthOf(year, month, calendar, ri.today)
	if cur.From.After(ri.today) {
		return fieldFail(c, "ym", "validation.before_or_equal", map[string]string{"date": ri.today.String()})
	}
	return serve(c, h.cache, ri, "monthly:"+m[0]+":"+calendar, "-", func(context.Context) (any, error) {
		in, err := h.load(ri, Range{Key: "month", From: cur.From, To: cur.To}, prev.From.AddDays(-6), false)
		if err != nil {
			return nil, err
		}
		return BuildMonthly(in, cur, prev, ri.ents.allowed(plus.PDFShare)).JSON(in.Copy), nil
	})
}
