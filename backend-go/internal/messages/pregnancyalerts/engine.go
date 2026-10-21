package pregnancyalerts

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Group is the message_contents group of the rules and the legend.
const Group = "pregnancy_alert"

// LegendKey is the legend row of Group.
const LegendKey = "legend"

// ListWindowDays is the window of GET /pregnancy/v2/alerts (the Today badge counts the same window).
const ListWindowDays = v2.AlertsWindowDays

const typePrefix = "v2:"

//go:embed lang/*/*.json
var langFS embed.FS

// emergencyFS is the embedded copy of the EmergencyRules (one payload per rule in the row shape, plus `variants`),
// used when a rule's message_contents row is missing or disabled and for copy variants a row does not carry.
//
//go:embed emergency/*.json
var emergencyFS embed.FS

// emergencyCopy is locale → rule → payload of the embedded emergency copy.
var emergencyCopy = sync.OnceValue(func() map[string]map[string]map[string]any {
	out := map[string]map[string]map[string]any{}
	entries, err := emergencyFS.ReadDir("emergency")
	if err != nil {
		panic(err)
	}
	for _, en := range entries {
		b, err := emergencyFS.ReadFile("emergency/" + en.Name())
		if err != nil {
			panic(err)
		}
		var m map[string]map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			panic(fmt.Errorf("pregnancy alerts: emergency copy %s: %w", en.Name(), err))
		}
		out[strings.TrimSuffix(en.Name(), ".json")] = m
	}
	return out
})

// embeddedRule is the embedded payload of rule in locale (the fallback locale when the locale has none; nil when
// the rule has no embedded copy).
func embeddedRule(rule, locale string) map[string]any {
	c := emergencyCopy()
	if m, ok := c[locale][rule]; ok {
		return m
	}
	return c[lang.FallbackLocale][rule]
}

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err)
	}
	return t
})

// T is the pregnancy_alerts line for key in locale.
func T(key, locale string) string { return translator().Trans("pregnancy_alerts."+key, nil, locale) }

// Engine evaluates, lists and acts on the v2 alerts of one user.
type Engine struct{ q store.Querier }

// New wires the engine.
func New(q store.Querier) *Engine { return &Engine{q: q} }

// rows is the live pregnancy_alert group: item_key → locale → decoded payload.
type rows map[string]map[string]map[string]any

func (e *Engine) rows(ctx context.Context, l v2.Lang) (rows, error) {
	list, err := e.q.ListLiveMessageGroup(ctx, Group)
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: rules: %w", err)
	}
	out := rows{}
	for _, r := range list {
		var m map[string]any
		if json.Unmarshal(r.Payload, &m) != nil || m == nil {
			continue
		}
		if out[r.ItemKey] == nil {
			out[r.ItemKey] = map[string]map[string]any{}
		}
		out[r.ItemKey][r.Locale] = m
	}
	out.withEmergency(l)
	return out, nil
}

// withEmergency replaces the row of every EmergencyRules rule that is missing or disabled (in the default language)
// by its embedded copy, marked forced, so the path still fires (T-M9-05).
func (r rows) withEmergency(l v2.Lang) {
	for _, rule := range EmergencyRules {
		if m := r.behaviour(rule, l); m != nil {
			if on, _ := m["enabled"].(bool); on {
				continue
			}
		}
		byLoc := map[string]map[string]any{}
		for loc, rules := range emergencyCopy() {
			p, ok := rules[rule]
			if !ok {
				continue
			}
			cp := make(map[string]any, len(p)+1)
			for k, v := range p {
				cp[k] = v
			}
			cp["forced"] = true
			byLoc[loc] = cp
		}
		r[rule] = byLoc
	}
}

// behaviour is the default-language row of an item (any row when it is missing).
func (r rows) behaviour(item string, l v2.Lang) map[string]any {
	byLoc := r[item]
	if m, ok := byLoc[l.Default]; ok {
		return m
	}
	if m, ok := byLoc[l.Locale]; ok {
		return m
	}
	for _, m := range byLoc {
		return m
	}
	return nil
}

// texts is the request-locale row, else the default-language one.
func (r rows) texts(item string, l v2.Lang) map[string]any {
	byLoc := r[item]
	if m, ok := byLoc[l.Locale]; ok {
		return m
	}
	return r.behaviour(item, l)
}

func (r rows) config(rule string, l v2.Lang) (Config, bool) {
	m := r.behaviour(rule, l)
	if m == nil {
		return Config{}, false
	}
	c := Config{Params: map[string]any{}}
	c.Enabled, _ = m["enabled"].(bool)
	c.Forced, _ = m["forced"].(bool)
	c.Level, _ = m["level"].(string)
	if n, ok := m["window_days"].(float64); ok {
		c.WindowDays = int(n)
	}
	c.WindowDays = max(1, c.WindowDays)
	if p, ok := m["params"].(map[string]any); ok {
		c.Params = p
	}
	if !contains(Levels, c.Level) {
		c.Level = "info"
	}
	return c, true
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// ErrNotActive: the user has no active, dated pregnancy.
var ErrNotActive = errors.New("pregnancy alerts: pregnancy not active")

func (e *Engine) dating(ctx context.Context, userID uint64, today civildate.Date) (v2.Dating, error) {
	p, err := e.q.GetProfileByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return v2.Dating{}, ErrNotActive
	}
	if err != nil {
		return v2.Dating{}, fmt.Errorf("pregnancy alerts: profile: %w", err)
	}
	if !p.PregnancyMode {
		return v2.Dating{}, ErrNotActive
	}
	d, ok := v2.Resolve(&p, today)
	if !ok {
		return v2.Dating{}, ErrNotActive
	}
	return d, nil
}

// facts loads the user's logs of the longest window (at least ListWindowDays).
func (e *Engine) facts(ctx context.Context, userID uint64, d v2.Dating, window int, now time.Time) (Facts, error) {
	today := d.Today
	from := today.AddDays(-(max(window, ListWindowDays) - 1))
	f := Facts{Today: today, Week: d.CurrentWeek(), WeekDay: d.Days, Source: d.Source, Days: map[civildate.Date]Day{},
		Now: now, TotalDays: d.TotalDays}

	syms, err := e.q.ListV2SymptomLogsRange(ctx, store.ListV2SymptomLogsRangeParams{UserID: userID, DateFrom: from, DateTo: today})
	if err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: symptoms: %w", err)
	}
	for i := range syms {
		f.Days[syms[i].LogDate] = SymptomDay(&syms[i], nil)
	}
	extras, err := e.q.ListDailyExtrasRange(ctx, store.ListDailyExtrasRangeParams{UserID: userID, DateFrom: from, DateTo: today})
	if err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: extras: %w", err)
	}
	for i := range extras {
		day := f.Days[extras[i].LogDate]
		if day == nil {
			day = Day{}
		}
		for k, v := range SymptomDay(nil, &extras[i]) {
			day[k] = v
		}
		f.Days[extras[i].LogDate] = day
	}
	if f.Weekly, err = e.q.ListV2WeeklyLogsSince(ctx, store.ListV2WeeklyLogsSinceParams{UserID: userID, DateFrom: from}); err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: weekly: %w", err)
	}
	wl, err := e.q.GetWeeklyLog(ctx, store.GetWeeklyLogParams{UserID: userID, PregnancyWeek: int32(f.Week)}) //nolint:gosec // 1..42
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return Facts{}, fmt.Errorf("pregnancy alerts: week log: %w", err)
	default:
		f.HasWeight = wl.Weight.Valid
	}
	if f.Fetal, err = e.q.ListFetalMovements(ctx, store.ListFetalMovementsParams{
		UserID: userID, HasFrom: 1, FromValue: from.String(), HasTo: 1, ToValue: today.String(),
	}); err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: fetal: %w", err)
	}
	cs, err := e.q.ListActiveSessionContractions(ctx, store.ListActiveSessionContractionsParams{UserID: userID})
	if err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: contractions: %w", err)
	}
	for _, c := range cs {
		f.ContractionSession = c.SessionID
		lc := labor.Contraction{Start: c.StartedAt}
		if c.EndedAt.Valid {
			lc.End = c.EndedAt.Time
		}
		f.Contractions = append(f.Contractions, lc)
	}
	if f.Kicks, err = e.kicks(ctx, userID, today); err != nil {
		return Facts{}, err
	}
	return f, nil
}

// kicks are the ended kick sessions started in the last KickLookbackDays (plus one day for a session that expired
// after midnight) and the running session, whenever it started.
func (e *Engine) kicks(ctx context.Context, userID uint64, today civildate.Date) ([]Kick, error) {
	ended, err := e.q.ListKickSessionsBetween(ctx, store.ListKickSessionsBetweenParams{
		UserID: userID, DateFrom: today.AddDays(-KickLookbackDays).TehranMidnight(), DateTo: today.AddDays(1).TehranMidnight(),
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: kicks: %w", err)
	}
	active, err := e.q.GetActiveKickSession(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("pregnancy alerts: active kicks: %w", err)
	default:
		ended = append(ended, active)
	}
	out := make([]Kick, 0, len(ended))
	for _, k := range ended {
		kk := Kick{ID: k.ID, Start: k.StartedAt, Count: int(k.Kicks)}
		if k.EndedAt.Valid && !k.ActiveLock.Valid {
			kk.End = k.EndedAt.Time
		}
		if k.TenthKickAt.Valid {
			kk.TargetAt = k.TenthKickAt.Time
		}
		out = append(out, kk)
	}
	return out, nil
}

// SymptomDay reads the v2 symptoms of a v1 symptom log and / or an extras row (has_X → severity,
// "mild" when unset), including the critical symptoms.
func SymptomDay(r *store.PregnancySymptomLog, ex *store.PregnancyDailyExtra) Day {
	out := Day{}
	add := func(name string, has sql.NullBool, sev sql.NullString) {
		if has.Valid && has.Bool {
			if sev.Valid && sev.String != "" {
				out[name] = sev.String
			} else {
				out[name] = "mild"
			}
		}
	}
	if r != nil {
		add("nausea", r.HasNausea, r.NauseaSeverity)
		add("vomiting", r.HasVomiting, r.VomitingSeverity)
		add("fatigue", r.HasFatigue, r.FatigueSeverity)
		add("headache", r.HasHeadache, r.HeadacheSeverity)
		add("back_pain", r.HasBackPain, r.BackPainSeverity)
		add("breast_pain", r.HasBreastPain, r.BreastPainSeverity)
		add("spotting", r.HasSpotting, r.SpottingSeverity)
		add("bleeding", r.HasBleeding, r.BleedingSeverity)
		add("fluid_leakage", r.HasFluidLeakage, r.FluidLeakageSeverity)
		add("severe_sudden_pain", r.HasSevereSuddenPain, r.SevereSuddenPainSeverity)
	}
	if ex != nil {
		if ex.HeartburnSeverity.Valid {
			out["heartburn"] = ex.HeartburnSeverity.String
		}
		if ex.ConstipationSeverity.Valid {
			out["constipation"] = ex.ConstipationSeverity.String
		}
	}
	return out
}

// meta is the v2 part of a pregnancy_alerts row (stored in trigger_symptoms).
type meta struct {
	Rule   string            `json:"rule"`
	Level4 string            `json:"level4"`
	Dedupe string            `json:"dedupe"`
	Vars   map[string]string `json:"vars"`
	Order  []string          `json:"order,omitempty"`
	// Variant is the copy variant of the hit (Hit.Variant).
	Variant string `json:"variant,omitempty"`
	// On is the fact's day (YYYY-MM-DD) when it differs from the evaluation day (Hit.On).
	On string `json:"on,omitempty"`
}

// Evaluate runs the log-driven rules after a log save (see EvaluateAll).
func (e *Engine) Evaluate(ctx context.Context, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error) {
	return e.evaluate(ctx, userID, now, l, false)
}

// EvaluateOn is Evaluate on q (the caller's transaction, e.g. the day-log save), so the rules
// see the caller's uncommitted writes and their alerts commit or roll back with them.
func (e *Engine) EvaluateOn(ctx context.Context, q store.Querier, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error) {
	return (&Engine{q: q}).evaluate(ctx, userID, now, l, false)
}

// EvaluateAll is the daily evaluation: every rule, including the calendar ones (TimeRules).
func (e *Engine) EvaluateAll(ctx context.Context, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error) {
	return e.evaluate(ctx, userID, now, l, true)
}

// TimeRules depend on the calendar, not on a save: they run in the daily evaluation only.
var TimeRules = []string{"weight_missing_week", "week_entered"}

// evaluate runs the enabled rules for the window ending today and persists the new hits
// (deduplicated per rule + key per window). It returns the created alerts rendered in l. A user
// without an active pregnancy gets nothing (no error).
func (e *Engine) evaluate(ctx context.Context, userID uint64, now time.Time, l v2.Lang, daily bool) ([]*jsonx.OrderedMap, error) {
	d, err := e.dating(ctx, userID, civildate.InTehran(now))
	if errors.Is(err, ErrNotActive) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rs, err := e.rows(ctx, l)
	if err != nil {
		return nil, err
	}
	configs := map[string]Config{}
	window := 1
	for _, k := range RuleKeys {
		if !daily && contains(TimeRules, k) {
			continue
		}
		if c, ok := rs.config(k, l); ok && c.Enabled {
			configs[k] = c
			window = max(window, c.WindowDays)
		}
	}
	if len(configs) == 0 {
		return nil, nil
	}
	f, err := e.facts(ctx, userID, d, window, now)
	if err != nil {
		return nil, err
	}
	out := []*jsonx.OrderedMap{}
	for _, k := range RuleKeys {
		c, ok := configs[k]
		if !ok {
			continue
		}
		for _, h := range Detect(k, c, f) {
			a, err := e.persist(ctx, userID, h, c, f, now, rs, l)
			if err != nil {
				return nil, err
			}
			if a != nil {
				out = append(out, a)
			}
		}
	}
	return out, nil
}

// RefreshRules take their facts from the dating alone (week, basis, week start), so a deduplicated
// hit rewrites the existing alert instead of being dropped: re-dating in Setup must not leave the
// alert with the old basis and fact date (QA 2026-09-29-c L5). Read / ack state is kept.
var RefreshRules = []string{"week_entered"}

// hitRow is what a hit stores: the v2 metadata and the locale-rendered fallback title / message.
type hitRow struct {
	trigger, actions []byte
	title, advice    string
}

func buildHit(h Hit, c Config, f Facts, rs rows, l v2.Lang, pinOn bool) (hitRow, error) {
	m := meta{Rule: h.Rule, Level4: hitLevel(h, c), Dedupe: h.Dedupe, Vars: map[string]string{}, Variant: h.Variant}
	// A refreshed row was created on another day, so its fact day is always pinned.
	if !h.On.IsZero() && (pinOn || h.On != f.Today) {
		m.On = h.On.String()
	}
	for _, kv := range h.Vars {
		m.Vars[kv[0]] = kv[1]
		m.Order = append(m.Order, kv[0])
	}
	trigger, err := json.Marshal(m)
	if err != nil {
		return hitRow{}, err
	}
	txt := render(rs.texts(h.Rule, l), m, l.Locale)
	labels := make([]string, 0, len(txt.actions))
	for _, a := range txt.actions {
		labels = append(labels, a.label)
	}
	actions, err := json.Marshal(labels)
	if err != nil {
		return hitRow{}, err
	}
	title := txt.title
	if title == "" {
		title = h.Rule
	}
	return hitRow{trigger: trigger, actions: actions, title: truncate(title, 255), advice: txt.advice}, nil
}

// hitLevel is the hit's own level (the emergency tiers) or the row's.
func hitLevel(h Hit, c Config) string {
	if contains(Levels, h.Level) {
		return h.Level
	}
	return c.Level
}

func (e *Engine) persist(ctx context.Context, userID uint64, h Hit, c Config, f Facts, now time.Time, rs rows, l v2.Lang,
) (*jsonx.OrderedMap, error) {
	since := sql.NullTime{Time: f.windowStart(c.WindowDays).TehranMidnight(), Valid: true}
	ts := sql.NullTime{Time: now.In(civildate.Tehran), Valid: true}
	if contains(RefreshRules, h.Rule) {
		return e.refresh(ctx, userID, h, c, f, since, ts, rs, l)
	}
	// A fact older than the window (a kick session that expired yesterday) is deduplicated back to its own day.
	if !h.On.IsZero() && h.On.TehranMidnight().Before(since.Time) {
		since.Time = h.On.TehranMidnight()
	}
	n, err := e.q.CountV2AlertDedupe(ctx, store.CountV2AlertDedupeParams{
		UserID: userID, AlertType: typePrefix + h.Rule, Since: since, Dedupe: h.Dedupe,
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: dedupe: %w", err)
	}
	if n > 0 {
		return nil, nil
	}
	return e.insert(ctx, userID, h, c, f, ts, rs, l)
}

func (e *Engine) insert(ctx context.Context, userID uint64, h Hit, c Config, f Facts, ts sql.NullTime, rs rows, l v2.Lang,
) (*jsonx.OrderedMap, error) {
	hr, err := buildHit(h, c, f, rs, l, false)
	if err != nil {
		return nil, err
	}
	id, err := e.q.InsertAlert(ctx, store.InsertAlertParams{
		UserID: userID, AlertLevel: V1Level(hitLevel(h, c)), AlertType: typePrefix + h.Rule,
		Title: hr.title, Message: hr.advice,
		PregnancyWeek:      sql.NullInt32{Int32: int32(f.Week), Valid: true}, //nolint:gosec // 1..42
		TriggerSymptoms:    db.NullRawJSON{V: hr.trigger, Valid: true},
		RecommendedActions: db.NullRawJSON{V: hr.actions, Valid: true},
		CreatedAt:          ts, UpdatedAt: ts,
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: insert: %w", err)
	}
	row, err := e.q.GetV2Alert(ctx, store.GetV2AlertParams{UserID: userID, ID: uint64(id)}) //nolint:gosec // auto-increment id
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: reload: %w", err)
	}
	return alertJSON(&row, rs, l), nil
}

// refresh is persist for a RefreshRules hit: a new alert when the window has none (returned as
// created), else the existing ones get the hit's current facts when they differ (not "created").
func (e *Engine) refresh(ctx context.Context, userID uint64, h Hit, c Config, f Facts, since, ts sql.NullTime, rs rows, l v2.Lang,
) (*jsonx.OrderedMap, error) {
	list, err := e.q.ListV2AlertsByDedupe(ctx, store.ListV2AlertsByDedupeParams{
		UserID: userID, AlertType: typePrefix + h.Rule, Since: since, Dedupe: h.Dedupe,
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: dedupe: %w", err)
	}
	if len(list) == 0 {
		return e.insert(ctx, userID, h, c, f, ts, rs, l)
	}
	hr, err := buildHit(h, c, f, rs, l, true)
	if err != nil {
		return nil, err
	}
	cur := decodeMeta(&store.PregnancyAlert{TriggerSymptoms: db.NullRawJSON{V: hr.trigger, Valid: true}})
	for i := range list {
		if sameFacts(&list[i], cur) {
			continue
		}
		if err := e.q.RefreshV2AlertFacts(ctx, store.RefreshV2AlertFactsParams{
			Title: hr.title, Message: hr.advice, TriggerSymptoms: db.NullRawJSON{V: hr.trigger, Valid: true},
			Now: ts, UserID: userID, ID: list[i].ID,
		}); err != nil {
			return nil, fmt.Errorf("pregnancy alerts: refresh: %w", err)
		}
	}
	return nil, nil
}

// sameFacts: the row's placeholders and fact day (`on`, else its creation day) match the hit's.
func sameFacts(r *store.PregnancyAlert, hit meta) bool {
	m := decodeMeta(r)
	on := m.On
	if on == "" && r.CreatedAt.Valid {
		on = civildate.FromTime(r.CreatedAt.Time.In(civildate.Tehran)).String()
	}
	if on != hit.On || len(m.Vars) != len(hit.Vars) {
		return false
	}
	for k, v := range m.Vars {
		if hit.Vars[k] != v {
			return false
		}
	}
	return true
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

type action struct{ key, label string }

type texts struct {
	title, whatWeSaw, howSure, advice, contact string
	actions                                    []action
}

// value localizes a placeholder value (symptom, status, basis, missing reading); anything else
// is a number (week, days, count, readings) and gets the locale's digits.
func value(name, v, locale string) string {
	switch {
	case v == none:
		return T("none", locale)
	case name == "symptom":
		return T("symptoms."+v, locale)
	case name == "status":
		return enums.FetalMovementStatus(v).Label(locale)
	case name == "basis":
		return enums.PregnancyAgeSource(v).Label(locale)
	}
	return v2.Digits(v, locale)
}

// render fills the rule's texts. A copy variant (meta.Variant) overrides the base texts with the row's
// `variants.<name>`, else the embedded variant. An emergency rule falls back to its embedded copy for any text, action
// or (urgent) contact its row lacks, so the urgent path never renders empty.
func render(p map[string]any, m meta, locale string) texts {
	emb := embeddedRule(m.Rule, locale)
	var variant map[string]any
	if m.Variant != "" {
		if vs, ok := p["variants"].(map[string]any); ok {
			variant, _ = vs[m.Variant].(map[string]any)
		}
		if variant == nil {
			if vs, ok := emb["variants"].(map[string]any); ok {
				variant, _ = vs[m.Variant].(map[string]any)
			}
		}
	}
	emergency := contains(EmergencyRules, m.Rule)
	fill := func(k string) string {
		s, _ := variant[k].(string)
		if s == "" {
			s, _ = p[k].(string)
		}
		if s == "" && emergency && (k != "contact" || m.Level4 == "urgent") {
			s, _ = emb[k].(string)
		}
		for name, v := range m.Vars {
			s = strings.ReplaceAll(s, "{"+name+"}", value(name, v, locale))
		}
		return s
	}
	t := texts{
		title: fill("title"), whatWeSaw: fill("what_we_saw"), howSure: fill("how_sure"),
		advice: fill("advice"), contact: fill("contact"),
	}
	list, _ := p["actions"].([]any)
	if len(list) == 0 && emergency {
		list, _ = emb["actions"].([]any)
	}
	for _, raw := range list {
		a, _ := raw.(map[string]any)
		key, _ := a["key"].(string)
		label, _ := a["label"].(string)
		if key != "" {
			t.actions = append(t.actions, action{key: key, label: label})
		}
	}
	return t
}

func decodeMeta(r *store.PregnancyAlert) meta {
	var m meta
	if r.TriggerSymptoms.Valid {
		_ = json.Unmarshal(r.TriggerSymptoms.V, &m)
	}
	if m.Rule == "" {
		m.Rule = strings.TrimPrefix(r.AlertType, typePrefix)
	}
	if !contains(Levels, m.Level4) {
		m.Level4 = "info"
	}
	return m
}

// ContactPhoneParam is the rule param with the number an urgent alert's call action dials (admin registry
// ContactPhoneParam).
const ContactPhoneParam = "contact_phone"

// alertJSON renders a v2 row with the current texts of its rule (the stored title / message
// when the rule has no row any more).
func alertJSON(r *store.PregnancyAlert, rs rows, l v2.Lang) *jsonx.OrderedMap {
	m := decodeMeta(r)
	p := rs.texts(m.Rule, l)
	var t texts
	if p != nil || contains(EmergencyRules, m.Rule) {
		t = render(p, m, l.Locale)
	}
	if t.title == "" {
		t.title = r.Title
	}
	if t.advice == "" {
		t.advice = r.Message
	}
	actions := []any{}
	for _, a := range t.actions {
		actions = append(actions, jsonx.Obj("key", a.key, "label", a.label))
	}
	var contact, created, factDate, dateLabel any
	if m.Level4 == "urgent" && t.contact != "" {
		contact = t.contact
		// A rule with a contact phone (contractions_511 param contact_phone, B-N5-09) answers {text, phone} so the
		// app's «تماس» dials it; without one the contact stays the plain text (unchanged shape).
		if c, ok := rs.config(m.Rule, l); ok {
			if phone, isStr := c.Params[ContactPhoneParam].(string); isStr && strings.TrimSpace(phone) != "" {
				contact = jsonx.Obj("text", t.contact, "phone", strings.TrimSpace(phone))
			}
		}
	}
	if r.CreatedAt.Valid {
		c := r.CreatedAt.Time.In(civildate.Tehran)
		created = c.Format(time.RFC3339)
		day := civildate.FromTime(c)
		if on, err := civildate.Parse(m.On); m.On != "" && err == nil {
			day = on
		}
		factDate, dateLabel = day.String(), v2.FullDate(day, l.Locale)
	}
	var week any
	if r.PregnancyWeek.Valid {
		week = int(r.PregnancyWeek.Int32)
	}
	return jsonx.Obj(
		"id", r.ID,
		"rule_key", m.Rule,
		"level", m.Level4,
		"alert_level", r.AlertLevel,
		"title", t.title,
		"what_we_saw", nilIfEmpty(t.whatWeSaw),
		"how_sure", nilIfEmpty(t.howSure),
		"advice", nilIfEmpty(t.advice),
		"actions", actions,
		"contact", contact,
		"pregnancy_week", week,
		"created_at", created,
		"fact_date", factDate,
		"date_label", dateLabel,
		"is_read", r.IsRead,
		"is_acked", r.IsDismissed,
	)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// List is GET /pregnancy/v2/alerts: the daily evaluation (deduplicated, so at most once per
// window) when evaluate is set, then the last ListWindowDays days of v2 alerts and the level
// legend. HEAD passes evaluate=false so it never writes (review #10, T-M2-34).
func (e *Engine) List(ctx context.Context, userID uint64, now time.Time, l v2.Lang, evaluate bool) (*jsonx.OrderedMap, error) {
	today := civildate.InTehran(now)
	if _, err := e.dating(ctx, userID, today); err != nil {
		return nil, err
	}
	if evaluate {
		if _, err := e.EvaluateAll(ctx, userID, now, l); err != nil {
			return nil, err
		}
	}
	rs, err := e.rows(ctx, l)
	if err != nil {
		return nil, err
	}
	since := v2.AlertsSince(today)
	list, err := e.q.ListV2AlertsSince(ctx, store.ListV2AlertsSinceParams{UserID: userID, Since: sql.NullTime{Time: since, Valid: true}})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: list: %w", err)
	}
	alerts := make([]any, 0, len(list))
	for i := range list {
		alerts = append(alerts, alertJSON(&list[i], rs, l))
	}

	lg := rs.texts(LegendKey, l)
	days := fmt.Sprint(ListWindowDays)
	str := func(k string) any {
		s, _ := lg[k].(string)
		return nilIfEmpty(strings.ReplaceAll(s, "{days}", days))
	}
	legend := []any{}
	levels, _ := lg["levels"].(map[string]any)
	for _, lv := range Levels {
		row, _ := levels[lv].(map[string]any)
		label, _ := row["label"].(string)
		desc, _ := row["description"].(string)
		legend = append(legend, jsonx.Obj("level", lv, "label", nilIfEmpty(label), "description", nilIfEmpty(desc)))
	}
	return jsonx.Obj(
		"window_days", ListWindowDays,
		"window_note", str("window_note"),
		"title", str("title"),
		"alerts", alerts,
		"legend", legend,
		"disclaimer", str("disclaimer"),
	), nil
}

// ErrNotFound: no v2 alert with that id for the user.
var ErrNotFound = errors.New("pregnancy alerts: not found")

// Actions are the server-side alert actions.
var Actions = []string{"ack", "add_to_visit_note"}

// Act runs POST /pregnancy/v2/alerts/{id}/actions/{action}. ack marks the alert read and
// dismissed; add_to_visit_note appends its "what we saw" line (title when empty) to today's
// visit_note (once) and marks it read.
func (e *Engine) Act(ctx context.Context, userID, id uint64, act string, now time.Time, l v2.Lang) (*jsonx.OrderedMap, error) {
	row, err := e.q.GetV2Alert(ctx, store.GetV2AlertParams{UserID: userID, ID: id})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: get: %w", err)
	}
	rs, err := e.rows(ctx, l)
	if err != nil {
		return nil, err
	}
	ts := sql.NullTime{Time: now.In(civildate.Tehran), Valid: true}
	var note any
	switch act {
	case "ack":
		if err := e.q.AckV2Alert(ctx, store.AckV2AlertParams{Now: ts, UserID: userID, ID: id}); err != nil {
			return nil, fmt.Errorf("pregnancy alerts: ack: %w", err)
		}
	case "add_to_visit_note":
		cur := alertJSON(&row, rs, l)
		line, _ := cur.Get("what_we_saw")
		if line == nil {
			line, _ = cur.Get("title")
		}
		if note, err = e.appendNote(ctx, userID, civildate.InTehran(now), fmt.Sprint(line), ts); err != nil {
			return nil, err
		}
		if err := e.q.MarkV2AlertRead(ctx, store.MarkV2AlertReadParams{Now: ts, UserID: userID, ID: id}); err != nil {
			return nil, fmt.Errorf("pregnancy alerts: read: %w", err)
		}
	default:
		return nil, ErrNotFound
	}
	if row, err = e.q.GetV2Alert(ctx, store.GetV2AlertParams{UserID: userID, ID: id}); err != nil {
		return nil, fmt.Errorf("pregnancy alerts: reload: %w", err)
	}
	return jsonx.Obj("alert", alertJSON(&row, rs, l), "visit_note", note), nil
}

// maxVisitNote is the day log's visit_note limit.
const maxVisitNote = 2000

func (e *Engine) appendNote(ctx context.Context, userID uint64, today civildate.Date, line string, ts sql.NullTime) (string, error) {
	ex, err := e.q.GetDailyExtras(ctx, store.GetDailyExtrasParams{UserID: userID, LogDate: today})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("pregnancy alerts: extras: %w", err)
	}
	cur := strings.TrimSpace(ex.VisitNote.String)
	line = strings.TrimSpace(line)
	if line == "" || strings.Contains(cur, line) {
		return cur, nil
	}
	next := line
	if cur != "" {
		next = cur + "\n" + line
	}
	next = truncate(next, maxVisitNote)
	if err := e.q.SetDailyVisitNote(ctx, store.SetDailyVisitNoteParams{
		UserID: userID, LogDate: today, VisitNote: sql.NullString{String: next, Valid: true}, Now: ts,
	}); err != nil {
		return "", fmt.Errorf("pregnancy alerts: visit note: %w", err)
	}
	return next, nil
}

// ReadingRules are the rules a handed-over vitals reading runs through.
var ReadingRules = []string{"bp_high", "sugar_high"}

// EvaluateReadings hands vitals readings of a pregnant user to the BP and GDM rules (T-M9-05): the hits are persisted
// like any other (deduplicated with the weekly-log hits of the same day and values, so one fact makes one row) and
// the alerts of the readings are returned rendered in l: the ones created now and the ones that already existed for
// the same fact. A user without an active pregnancy gets nothing.
func (e *Engine) EvaluateReadings(ctx context.Context, userID uint64, readings []Reading, now time.Time, l v2.Lang,
) ([]*jsonx.OrderedMap, error) {
	d, err := e.dating(ctx, userID, civildate.InTehran(now))
	if errors.Is(err, ErrNotActive) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rs, err := e.rows(ctx, l)
	if err != nil {
		return nil, err
	}
	f := Facts{Today: d.Today, Week: d.CurrentWeek(), WeekDay: d.Days, Source: d.Source, Days: map[civildate.Date]Day{},
		Now: now, TotalDays: d.TotalDays, Readings: readings}
	out := []*jsonx.OrderedMap{}
	for _, rule := range ReadingRules {
		c, ok := rs.config(rule, l)
		if !ok || !c.Enabled {
			continue
		}
		for _, h := range Detect(rule, c, f) {
			a, err := e.persist(ctx, userID, h, c, f, now, rs, l)
			if err != nil {
				return nil, err
			}
			if a == nil {
				if a, err = e.existing(ctx, userID, h, c, f, rs, l); err != nil {
					return nil, err
				}
			}
			if a != nil {
				out = append(out, a)
			}
		}
	}
	return out, nil
}

// existing is the alert a deduplicated hit already has in its window (nil when none).
func (e *Engine) existing(ctx context.Context, userID uint64, h Hit, c Config, f Facts, rs rows, l v2.Lang,
) (*jsonx.OrderedMap, error) {
	list, err := e.q.ListV2AlertsByDedupe(ctx, store.ListV2AlertsByDedupeParams{
		UserID: userID, AlertType: typePrefix + h.Rule,
		Since: sql.NullTime{Time: f.windowStart(c.WindowDays).TehranMidnight(), Valid: true}, Dedupe: h.Dedupe,
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: dedupe: %w", err)
	}
	if len(list) == 0 {
		return nil, nil
	}
	return alertJSON(&list[0], rs, l), nil
}

// FiresWithoutCopy runs rule on f with no message_contents rows at all (every row deleted) and reports the level and
// rendered title of its first hit in locale ("" , "" when it does not fire). The nightly safety scenario
// emergency_rule_without_copy calls it.
func FiresWithoutCopy(rule string, f Facts, l v2.Lang) (level, title, contact string) {
	rs := rows{}
	rs.withEmergency(l)
	c, ok := rs.config(rule, l)
	if !ok {
		return "", "", ""
	}
	hits := Detect(rule, c, f)
	if len(hits) == 0 {
		return "", "", ""
	}
	m := meta{Rule: rule, Level4: hitLevel(hits[0], c), Vars: map[string]string{}, Variant: hits[0].Variant}
	for _, kv := range hits[0].Vars {
		m.Vars[kv[0]] = kv[1]
	}
	t := render(rs.texts(rule, l), m, l.Locale)
	return m.Level4, t.title, t.contact
}
