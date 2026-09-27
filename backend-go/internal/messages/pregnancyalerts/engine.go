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
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Group is the message_contents group of the rules and the legend.
const Group = "pregnancy_alert"

// LegendKey is the legend row of Group.
const LegendKey = "legend"

// ListWindowDays is the window of GET /pregnancy/v2/alerts.
const ListWindowDays = 7

const typePrefix = "v2:"

//go:embed lang/*/*.json
var langFS embed.FS

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

func (e *Engine) rows(ctx context.Context) (rows, error) {
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
	return out, nil
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
func (e *Engine) facts(ctx context.Context, userID uint64, d v2.Dating, window int) (Facts, error) {
	today := d.Today
	from := today.AddDays(-(max(window, ListWindowDays) - 1))
	f := Facts{Today: today, Week: d.CurrentWeek(), WeekDay: d.Days, Source: d.Source, Days: map[civildate.Date]Day{}}

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
	return f, nil
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
}

// Evaluate runs the log-driven rules after a log save (see EvaluateAll).
func (e *Engine) Evaluate(ctx context.Context, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error) {
	return e.evaluate(ctx, userID, now, l, false)
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
	rs, err := e.rows(ctx)
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
	f, err := e.facts(ctx, userID, d, window)
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

func (e *Engine) persist(ctx context.Context, userID uint64, h Hit, c Config, f Facts, now time.Time, rs rows, l v2.Lang,
) (*jsonx.OrderedMap, error) {
	since := f.windowStart(c.WindowDays).TehranMidnight()
	n, err := e.q.CountV2AlertDedupe(ctx, store.CountV2AlertDedupeParams{
		UserID: userID, AlertType: typePrefix + h.Rule, Since: sql.NullTime{Time: since, Valid: true}, Dedupe: h.Dedupe,
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: dedupe: %w", err)
	}
	if n > 0 {
		return nil, nil
	}
	m := meta{Rule: h.Rule, Level4: c.Level, Dedupe: h.Dedupe, Vars: map[string]string{}}
	for _, kv := range h.Vars {
		m.Vars[kv[0]] = kv[1]
		m.Order = append(m.Order, kv[0])
	}
	trigger, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	txt := render(rs.texts(h.Rule, l), m, l.Locale)
	labels := make([]string, 0, len(txt.actions))
	for _, a := range txt.actions {
		labels = append(labels, a.label)
	}
	actions, err := json.Marshal(labels)
	if err != nil {
		return nil, err
	}
	title := txt.title
	if title == "" {
		title = h.Rule
	}
	ts := sql.NullTime{Time: now.In(civildate.Tehran), Valid: true}
	id, err := e.q.InsertAlert(ctx, store.InsertAlertParams{
		UserID: userID, AlertLevel: V1Level(c.Level), AlertType: typePrefix + h.Rule,
		Title: truncate(title, 255), Message: txt.advice,
		PregnancyWeek:      sql.NullInt32{Int32: int32(f.Week), Valid: true}, //nolint:gosec // 1..42
		TriggerSymptoms:    db.NullRawJSON{V: trigger, Valid: true},
		RecommendedActions: db.NullRawJSON{V: actions, Valid: true},
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

func render(p map[string]any, m meta, locale string) texts {
	fill := func(k string) string {
		s, _ := p[k].(string)
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

// alertJSON renders a v2 row with the current texts of its rule (the stored title / message
// when the rule has no row any more).
func alertJSON(r *store.PregnancyAlert, rs rows, l v2.Lang) *jsonx.OrderedMap {
	m := decodeMeta(r)
	p := rs.texts(m.Rule, l)
	var t texts
	if p != nil {
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
	var contact, created, dateLabel any
	if m.Level4 == "urgent" && t.contact != "" {
		contact = t.contact
	}
	if r.CreatedAt.Valid {
		c := r.CreatedAt.Time.In(civildate.Tehran)
		created = c.Format(time.RFC3339)
		dateLabel = v2.FullDate(civildate.FromTime(c), l.Locale)
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
// window), then the last ListWindowDays days of v2 alerts and the level legend.
func (e *Engine) List(ctx context.Context, userID uint64, now time.Time, l v2.Lang) (*jsonx.OrderedMap, error) {
	today := civildate.InTehran(now)
	if _, err := e.dating(ctx, userID, today); err != nil {
		return nil, err
	}
	if _, err := e.EvaluateAll(ctx, userID, now, l); err != nil {
		return nil, err
	}
	rs, err := e.rows(ctx)
	if err != nil {
		return nil, err
	}
	since := today.AddDays(-(ListWindowDays - 1)).TehranMidnight()
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
	rs, err := e.rows(ctx)
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
