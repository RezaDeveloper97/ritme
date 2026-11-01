package inboxalerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/inbox"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/messages/rules"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
	"github.com/ritme/backend-go/internal/vitals"
	vstore "github.com/ritme/backend-go/internal/vitals/store"
)

// Service is the pregnancyalerts.Backend over the alerts inbox.
type Service struct {
	mq     mstore.Querier
	logger *slog.Logger
	load   func(ctx context.Context) (*compiled.Compiled, error)
	memo   atomic.Pointer[snapshot]
	mu     sync.Mutex
}

var _ pregnancyalerts.Backend = (*Service)(nil)

// New is the service on the messages store (dbtx: the database; the pregnancy reads go through the caller's querier).
func New(dbtx mstore.DBTX, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	s := &Service{mq: mstore.New(dbtx), logger: logger}
	s.load = func(ctx context.Context) (*compiled.Compiled, error) { return compiled.Load(ctx, s.mq) }
	return s
}

// Install makes the inbox the pregnancy alert backend (pregnancyalerts.InstallBackend) on the database dbtx.
func Install(dbtx mstore.DBTX, logger *slog.Logger) *Service {
	s := New(dbtx, logger)
	pregnancyalerts.InstallBackend(s)
	return s
}

// snapshot is the served preg.* rules of one compiled message snapshot, with the code-held safety rules standing in
// for a locked rule whose row is missing, inactive or not approved (SPEC §4.2, T-M9-05's guarantee).
type snapshot struct {
	base         *compiled.Compiled
	comp         *compiled.Compiled
	list         []rules.Rule
	byID         map[uint64]*rules.Rule
	codeHeld     map[uint64]bool
	translations map[uint64]map[string]rules.Content
}

// Snapshot returns the current pregnancy rule snapshot (recompiled when the process message snapshot changes).
func (s *Service) snapshot(ctx context.Context) (*snapshot, error) {
	base, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	if m := s.memo.Load(); m != nil && m.base == base {
		return m, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if m := s.memo.Load(); m != nil && m.base == base {
		return m, nil
	}
	rows, err := rules.ActiveRows(ctx, s.mq, rules.SlotAlertsInbox)
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: rules: %w", err)
	}
	snap, err := build(ctx, s.mq, rows)
	if err != nil {
		return nil, err
	}
	snap.base = base
	s.memo.Store(snap)
	return snap, nil
}

// build is the snapshot of the served rows: the preg.* rules that are served, plus the code-held definition of every
// locked rule that is not (keeping the row's id when the row still exists, so its open alert stays one row).
func build(ctx context.Context, q mstore.Querier, rows []mstore.MessageRule) (*snapshot, error) {
	snap := &snapshot{byID: map[uint64]*rules.Rule{}, codeHeld: map[uint64]bool{}, translations: map[uint64]map[string]rules.Content{}}
	served := map[string]bool{}
	for i := range rows {
		row := &rows[i]
		if !strings.HasPrefix(row.RuleKey.String, KeyPrefix) {
			continue
		}
		r := rules.FromRow(row)
		if !r.Served() {
			continue
		}
		snap.list = append(snap.list, r)
		served[r.RuleKey] = true
		var sc storedContent
		if json.Unmarshal([]byte(row.Content), &sc) == nil && len(sc.Translations) > 0 {
			snap.translations[r.ID] = sc.Translations
		}
	}
	for _, d := range Defs {
		if !d.Locked || served[d.Key] {
			continue
		}
		id, err := q.GetMessageRuleIDByKey(ctx, sql.NullString{String: d.Key, Valid: true})
		switch {
		case errors.Is(err, sql.ErrNoRows):
			id = CodeRuleID(d.Key)
		case err != nil:
			return nil, fmt.Errorf("pregnancy alerts: rule id: %w", err)
		}
		snap.list = append(snap.list, d.Rule(id))
		snap.codeHeld[id] = true
	}
	for i := range snap.list {
		snap.byID[snap.list[i].ID] = &snap.list[i]
	}
	snap.comp = compiled.CompileRules(snap.list)
	return snap, nil
}

// ---- facts ----

// source is the pregnancy fact source of one user and evaluation (facts.PregnancySource + PregnancyAlertSource).
type source struct {
	q      pstore.Querier
	mq     mstore.Querier
	userID uint64
	now    time.Time
	extra  []pregnancyalerts.Reading

	loaded bool
	dating *v2.Dating
	input  *facts.PregnancyAlertInput
	err    error
}

func (s *source) load(ctx context.Context) error {
	if s.loaded {
		return s.err
	}
	s.loaded = true
	s.err = s.read(ctx)
	return s.err
}

func (s *source) read(ctx context.Context) error {
	today := civildate.InTehran(s.now)
	d, err := pregnancyalerts.DatingOf(ctx, s.q, s.userID, today)
	if errors.Is(err, pregnancyalerts.ErrNotActive) {
		return nil
	}
	if err != nil {
		return err
	}
	f, ok, err := pregnancyalerts.LoadFacts(ctx, s.q, s.userID, s.now)
	if err != nil || !ok {
		return err
	}
	if f.Readings, err = s.readings(ctx, today, today.AddDays(-d.TotalDays)); err != nil {
		return err
	}
	in := &facts.PregnancyAlertInput{Facts: f, Start: today.AddDays(-d.TotalDays)}
	w, err := s.q.GetV2LastWeight(ctx, s.userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return fmt.Errorf("pregnancy alerts: last weight: %w", err)
	default:
		in.LastWeight = w.LogDate
	}
	ex, err := s.q.GetV2TodayExtras(ctx, pstore.GetV2TodayExtrasParams{UserID: s.userID,
		AlertsSince: sql.NullTime{Time: s.now, Valid: true}, Now: sql.NullTime{Time: s.now.In(civildate.Tehran), Valid: true},
		CurrentWeek: uint8(min(max(d.CurrentWeek(), 0), 255))}) //nolint:gosec // 1..42
	var me *mysql.MySQLError
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case errors.As(err, &me) && me.Number == 1146:
		// The care tables are Go-only (the Laravel contract schema lacks them): no visit, the other rules still run.
	case err != nil:
		return fmt.Errorf("pregnancy alerts: next visit: %w", err)
	default:
		if ex.AppointmentAt.Valid {
			in.NextVisit = ex.AppointmentAt.Time
		}
	}
	s.dating, s.input = &d, in
	return nil
}

// readings are the vitals BP and glucose readings of the last 7 days inside the pregnancy (timed + log sheet, merged as
// the vitals screens merge them), plus the readings handed over by the save.
func (s *source) readings(ctx context.Context, today, start civildate.Date) ([]pregnancyalerts.Reading, error) {
	from := today.AddDays(-(pregnancyalerts.FactsWindowDays - 1))
	if from.Before(start) {
		from = start
	}
	rows, err := s.mq.ListMessageFactVitalReadings(ctx, mstore.ListMessageFactVitalReadingsParams{
		UserID: s.userID, DateFrom: from.TehranMidnight(), DateTo: today.AddDays(1).TehranMidnight()})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: vitals: %w", err)
	}
	logRows, err := s.mq.ListMessageFactLogMeasurements(ctx, mstore.ListMessageFactLogMeasurementsParams{
		UserID: s.userID, DateFrom: from, DateTo: today})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: vitals log values: %w", err)
	}
	timed := make([]vitals.Reading, len(rows))
	for i, r := range rows {
		timed[i] = vitals.FromRow(vstore.VitalReading(r))
	}
	logs := make([]vstore.ListLogMeasurementsRow, len(logRows))
	for i, r := range logRows {
		logs[i] = vstore.ListLogMeasurementsRow(r)
	}
	var out []pregnancyalerts.Reading
	for _, r := range vitals.Merge(timed, logs) {
		if pr, ok := vitals.PregnancyReading(r); ok {
			out = append(out, pr)
		}
	}
	for _, r := range s.extra {
		if !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	return out, nil
}

// PregnancyDating implements facts.PregnancySource.
func (s *source) PregnancyDating(ctx context.Context, _ civildate.Date) (*facts.PregnancyDating, error) {
	if err := s.load(ctx); err != nil || s.dating == nil {
		return nil, err
	}
	return &facts.PregnancyDating{Week: s.dating.CurrentWeek(), Trimester: s.dating.Trimester()}, nil
}

// PregnancyAlertInput implements facts.PregnancyAlertSource.
func (s *source) PregnancyAlertInput(ctx context.Context, _ civildate.Date) (*facts.PregnancyAlertInput, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	return s.input, nil
}

// Input is the pregnancy alert input of userID at now (nil when not pregnant): the pregnancy logs read through q, the
// vitals readings through mq. It is what the facts.PregnancyAlertSource of a fact source returns (the feed's
// messages.StoreSource delegates here, so the feed and the pregnancy paths evaluate the preg.* rules on one input).
func Input(ctx context.Context, q pstore.Querier, mq mstore.Querier, userID uint64, now time.Time) (*facts.PregnancyAlertInput, error) {
	src := &source{q: q, mq: mq, userID: userID, now: now.In(civildate.Tehran)}
	return src.PregnancyAlertInput(ctx, civildate.InTehran(now))
}

// Facts are the pregnancy and pregnancy alert facts of src's user on day (the facts the preg.* rules read).
func Facts(ctx context.Context, src facts.PregnancySource, day civildate.Date) (rules.Facts, error) {
	fs := facts.New(facts.Sources{Pregnancy: src}, day, "")
	if err := fs.Need(ctx, "pregnancy_week", "preg.week_day"); err != nil {
		return nil, err
	}
	return rules.Facts(fs.Values()), nil
}

// box is an inbox service over the snapshot's rules and the user's pregnancy facts read through q.
// week, when set, gets the pregnancy week the evaluation loaded (0 = no dating), for the alerts whose texts carry no
// {preg_week}.
func (s *Service) box(snap *snapshot, q pstore.Querier, extra []pregnancyalerts.Reading, week *int) *inbox.Service {
	return inbox.New(s.mq, nil, s.logger).
		WithLoader(func(context.Context) (*compiled.Compiled, error) { return snap.comp, nil }).
		WithFacts(func(ctx context.Context, userID uint64, at time.Time) (rules.Facts, time.Time, error) {
			local := at.In(civildate.Tehran)
			src := &source{q: q, mq: s.mq, userID: userID, now: local, extra: extra}
			f, err := Facts(ctx, src, civildate.FromTime(local))
			if week != nil && src.dating != nil {
				*week = src.dating.CurrentWeek()
			}
			return f, local, err
		})
}

// ---- Backend ----

// Evaluate implements pregnancyalerts.Backend.
func (s *Service) Evaluate(ctx context.Context, q pstore.Querier, userID uint64, now time.Time, l v2.Lang,
	opt pregnancyalerts.EvalOptions,
) ([]*jsonx.OrderedMap, error) {
	snap, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	week := 0
	open, err := s.box(snap, q, opt.Readings, &week).EvaluateUser(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	changed := map[uint64]bool{}
	for _, o := range open {
		if o.Rule != nil && o.LastAt.IsZero() {
			changed[o.Rule.ID] = true
		}
	}
	if len(changed) == 0 && len(opt.Readings) == 0 {
		return []*jsonx.OrderedMap{}, nil
	}
	rows, err := s.mq.ListUserOpenInbox(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: open rows: %w", err)
	}
	out := []*jsonx.OrderedMap{}
	for i := range rows {
		r := rowOfOpen(&rows[i])
		d, ok := DefOf(r.RuleKey.String)
		if !ok || (d.Time && !opt.Daily) {
			continue
		}
		pick := changed[r.RuleID]
		if !pick && d.Readings && len(opt.Readings) > 0 {
			pick = sameReading(r.vars(), opt.Readings, d)
		}
		if pick {
			a := s.alertJSON(snap, &r, l)
			if v, _ := a.Get("pregnancy_week"); v == nil && week > 0 {
				a.Set("pregnancy_week", week)
			}
			if opt.V1 {
				a.Set("trigger_symptoms", V1Trigger(d.Key, r.vars()))
			}
			out = append(out, a)
		}
	}
	return withContactPhone(ctx, q, l, out)
}

// sameReading reports whether a reading rule's open row holds one of the handed-over readings (the save re-sent a
// reading that already raised it: its alert is the save's answer again).
func sameReading(vars map[string]any, readings []pregnancyalerts.Reading, d Def) bool {
	for _, rd := range readings {
		on := rd.Date.String()
		if d.Key == "preg.gdm" {
			v := strconv.FormatFloat(rd.MgDl, 'f', -1, 64)
			if vars["preg_gdm_on"] == on && (vars["preg_fasting"] == v || vars["preg_post_meal"] == v) {
				return true
			}
			continue
		}
		if vars["preg_bp_on"] == on && vars["preg_bp_sys"] == strconv.Itoa(rd.Systolic) && vars["preg_bp_dia"] == strconv.Itoa(rd.Diastolic) {
			return true
		}
	}
	return false
}

// List implements pregnancyalerts.Backend: GET /pregnancy/v2/alerts over the inbox — the open alerts and the
// alerts of the last ListWindowDays days, newest first, plus the level legend.
func (s *Service) List(ctx context.Context, q pstore.Querier, userID uint64, now time.Time, l v2.Lang, evaluate bool,
) (*jsonx.OrderedMap, error) {
	today := civildate.InTehran(now)
	if _, err := pregnancyalerts.DatingOf(ctx, q, userID, today); err != nil {
		return nil, err
	}
	if evaluate {
		if _, err := s.Evaluate(ctx, q, userID, now, l, pregnancyalerts.EvalOptions{Daily: true}); err != nil {
			return nil, err
		}
	}
	snap, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.rows(ctx, userID, v2.AlertsSince(today))
	if err != nil {
		return nil, err
	}
	list := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		list = append(list, s.alertJSON(snap, &rows[i], l))
	}
	if list, err = withContactPhone(ctx, q, l, list); err != nil {
		return nil, err
	}
	alerts := make([]any, 0, len(list))
	for _, a := range list {
		alerts = append(alerts, a)
	}
	lg, err := legend(ctx, q, l)
	if err != nil {
		return nil, err
	}
	days := strconv.Itoa(pregnancyalerts.ListWindowDays)
	str := func(k string) any {
		v, _ := lg[k].(string)
		return nilIfEmpty(strings.ReplaceAll(v, "{days}", days))
	}
	levels, _ := lg["levels"].(map[string]any)
	leg := []any{}
	for _, lv := range pregnancyalerts.Levels {
		row, _ := levels[lv].(map[string]any)
		label, _ := row["label"].(string)
		desc, _ := row["description"].(string)
		leg = append(leg, jsonx.Obj("level", lv, "label", nilIfEmpty(label), "description", nilIfEmpty(desc)))
	}
	return jsonx.Obj(
		"window_days", pregnancyalerts.ListWindowDays,
		"window_note", str("window_note"),
		"title", str("title"),
		"alerts", alerts,
		"legend", leg,
		"disclaimer", str("disclaimer"),
	), nil
}

// legend is the legend row of message_contents group pregnancy_alert in the request locale (else the default).
func legend(ctx context.Context, q pstore.Querier, l v2.Lang) (map[string]any, error) {
	list, err := q.ListLiveMessageGroup(ctx, pregnancyalerts.Group)
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: legend: %w", err)
	}
	var out, def map[string]any
	for _, r := range list {
		if r.ItemKey != pregnancyalerts.LegendKey {
			continue
		}
		var m map[string]any
		if json.Unmarshal(r.Payload, &m) != nil {
			continue
		}
		switch r.Locale {
		case l.Locale:
			out = m
		case l.Default:
			def = m
		}
	}
	if out == nil {
		out = def
	}
	return out, nil
}

// listLimit bounds one list read (open rows plus the window's history; a pregnancy has a handful of rules).
const listLimit = 200

// rows are the user's open preg.* rows and the history rows last hit since, newest first.
func (s *Service) rows(ctx context.Context, userID uint64, since time.Time) ([]row, error) {
	open, err := s.mq.ListUserInboxOpenPage(ctx, mstore.ListUserInboxOpenPageParams{UserID: userID, Limit: listLimit})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: open page: %w", err)
	}
	hist, err := s.mq.ListUserInboxHistoryPage(ctx, mstore.ListUserInboxHistoryPageParams{UserID: userID, Limit: listLimit})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: history page: %w", err)
	}
	var out []row
	for _, r := range open {
		out = append(out, row{ID: r.ID, RuleID: r.RuleID, RuleKey: r.RuleKey, Level: r.Level, State: r.State,
			Evidence: r.Evidence, FirstAt: r.FirstAt, LastAt: r.LastAt, AckedAt: r.AckedAt, NotedAt: r.NotedAt})
	}
	for _, r := range hist {
		if r.LastAt.Valid && r.LastAt.Time.Before(since) {
			continue
		}
		out = append(out, row{ID: r.ID, RuleID: r.RuleID, RuleKey: r.RuleKey, Level: r.Level, State: r.State,
			Evidence: r.Evidence, FirstAt: r.FirstAt, LastAt: r.LastAt, AckedAt: r.AckedAt, NotedAt: r.NotedAt})
	}
	out = slices.DeleteFunc(out, func(r row) bool { return !strings.HasPrefix(r.RuleKey.String, KeyPrefix) })
	slices.SortStableFunc(out, func(a, b row) int {
		if c := b.LastAt.Time.Compare(a.LastAt.Time); c != 0 {
			return c
		}
		return int(b.ID) - int(a.ID) //nolint:gosec // ids are small
	})
	return out, nil
}

// Act implements pregnancyalerts.Backend: ack acknowledges an open alert (idempotent); add_to_visit_note appends its
// «what we saw» line (its title when none) to today's visit note once and marks an open alert added_to_visit_note.
func (s *Service) Act(ctx context.Context, q pstore.Querier, userID, id uint64, act string, now time.Time, l v2.Lang,
) (*jsonx.OrderedMap, error) {
	r, err := s.get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	snap, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	ts := sql.NullTime{Time: now, Valid: true}
	var note any
	switch act {
	case "ack":
		if r.open() {
			if _, err := s.mq.AckInboxItem(ctx, mstore.AckInboxItemParams{Now: ts, ID: id, UserID: userID}); err != nil {
				return nil, fmt.Errorf("pregnancy alerts: ack: %w", err)
			}
		}
	case "add_to_visit_note":
		t := s.texts(snap, &r, l.Locale)
		line := t.Seen
		if line == "" {
			line = t.Title
		}
		n, err := s.appendNote(ctx, userID, civildate.FromTime(now.In(civildate.Tehran)), line, ts)
		if err != nil {
			return nil, err
		}
		note = n
		if r.open() {
			if _, err := s.mq.NoteInboxItem(ctx, mstore.NoteInboxItemParams{Now: ts, ID: id, UserID: userID}); err != nil {
				return nil, fmt.Errorf("pregnancy alerts: visit note: %w", err)
			}
		}
	default:
		return nil, pregnancyalerts.ErrNotFound
	}
	if r, err = s.get(ctx, userID, id); err != nil {
		return nil, err
	}
	a, err := withContactPhone(ctx, q, l, []*jsonx.OrderedMap{s.alertJSON(snap, &r, l)})
	if err != nil {
		return nil, err
	}
	return jsonx.Obj("alert", a[0], "visit_note", note), nil
}

func (s *Service) get(ctx context.Context, userID, id uint64) (row, error) {
	x, err := s.mq.GetUserInboxItem(ctx, mstore.GetUserInboxItemParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return row{}, pregnancyalerts.ErrNotFound
	}
	if err != nil {
		return row{}, fmt.Errorf("pregnancy alerts: get: %w", err)
	}
	if !strings.HasPrefix(x.RuleKey.String, KeyPrefix) {
		return row{}, pregnancyalerts.ErrNotFound
	}
	return row{ID: x.ID, RuleID: x.RuleID, RuleKey: x.RuleKey, Level: x.Level, State: x.State, Evidence: x.Evidence,
		FirstAt: x.FirstAt, LastAt: x.LastAt, AckedAt: x.AckedAt, NotedAt: x.NotedAt}, nil
}

func (s *Service) appendNote(ctx context.Context, userID uint64, day civildate.Date, line string, ts sql.NullTime) (string, error) {
	cur, err := s.mq.GetInboxVisitNote(ctx, mstore.GetInboxVisitNoteParams{UserID: userID, LogDate: day})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("pregnancy alerts: visit note read: %w", err)
	}
	note, line := strings.TrimSpace(cur.String), strings.TrimSpace(line)
	if line == "" || strings.Contains(note, line) {
		return note, nil
	}
	next := line
	if note != "" {
		next = note + "\n" + line
	}
	if rs := []rune(next); len(rs) > inbox.MaxVisitNote {
		next = string(rs[:inbox.MaxVisitNote])
	}
	if err := s.mq.SetInboxVisitNote(ctx, mstore.SetInboxVisitNoteParams{UserID: userID, LogDate: day,
		VisitNote: sql.NullString{String: next, Valid: true}, Now: ts}); err != nil {
		return "", fmt.Errorf("pregnancy alerts: visit note write: %w", err)
	}
	return next, nil
}

// ---- rendering (the v2 alert shape) ----

// row is one inbox row as the v2 shape reads it.
type row struct {
	ID, RuleID       uint64
	RuleKey          sql.NullString
	Level, State     string
	Evidence         db.NullRawJSON
	FirstAt, LastAt  sql.NullTime
	AckedAt, NotedAt sql.NullTime
}

func rowOfOpen(r *mstore.ListUserOpenInboxRow) row {
	return row{ID: r.ID, RuleID: r.RuleID, RuleKey: r.RuleKey, Level: r.Level, State: r.State, Evidence: r.Evidence,
		FirstAt: r.FirstAt, LastAt: r.LastAt, AckedAt: r.AckedAt, NotedAt: r.NotedAt}
}

func (r *row) open() bool {
	return r.State == inbox.StateActive || r.State == inbox.StateAcknowledged || r.State == inbox.StateNoted
}

// vars are the evidence variables stored at the row's last hit.
func (r *row) vars() map[string]any {
	var d struct {
		Vars map[string]any `json:"vars"`
	}
	if !r.Evidence.Valid || json.Unmarshal(r.Evidence.V, &d) != nil || d.Vars == nil {
		return map[string]any{}
	}
	return d.Vars
}

// texts are the row's rule texts in locale, rendered with its stored variables (localized).
func (s *Service) texts(snap *snapshot, r *row, locale string) Copy {
	var raw Copy
	rule := snap.byID[r.RuleID]
	switch {
	case rule == nil || snap.codeHeld[r.RuleID]:
		raw, _ = CopyOf(r.RuleKey.String, locale)
	default:
		cnt := rule.Content
		if t, ok := snap.translations[r.RuleID][locale]; ok {
			if t.A.Title != "" || t.A.Body != "" {
				cnt.A = t.A
			}
			for _, p := range []struct{ dst, src *string }{{&cnt.Seen, &t.Seen}, {&cnt.Confidence, &t.Confidence}, {&cnt.Contact, &t.Contact}} {
				if *p.src != "" {
					*p.dst = *p.src
				}
			}
		}
		raw = Copy{Title: cnt.A.Title, Body: cnt.A.Body, Seen: cnt.Seen, Confidence: cnt.Confidence, Contact: cnt.Contact}
	}
	vars := r.vars()
	fill := func(t string) string { return Render(t, vars, locale) }
	return Copy{Title: fill(raw.Title), Body: fill(raw.Body), Seen: fill(raw.Seen), Confidence: fill(raw.Confidence),
		Contact: fill(raw.Contact)}
}

// Render fills the {variables} of t with vars, localized for locale: numbers in the locale's digits, Y-m-d dates as
// the full date, "Y-m-d H:i" as date + time, symptom / status / basis codes as their labels. A missing variable stays
// as written.
func Render(t string, vars map[string]any, locale string) string {
	for _, name := range rules.Placeholders(t) {
		v, ok := vars[name]
		if !ok {
			continue
		}
		t = strings.ReplaceAll(t, "{"+name+"}", value(name, v, locale))
	}
	return t
}

func value(name string, v any, locale string) string {
	switch x := v.(type) {
	case float64:
		return v2.Digits(strconv.FormatFloat(x, 'f', -1, 64), locale)
	case []any:
		parts := make([]string, 0, len(x))
		for _, p := range x {
			parts = append(parts, value(name, p, locale))
		}
		return strings.Join(parts, pregnancyalerts.T("list_separator", locale))
	case []string:
		parts := make([]string, 0, len(x))
		for _, p := range x {
			parts = append(parts, value(name, p, locale))
		}
		return strings.Join(parts, pregnancyalerts.T("list_separator", locale))
	case string:
		switch {
		case x == "—":
			return pregnancyalerts.T("none", locale)
		case name == "preg_symptoms":
			return pregnancyalerts.T("symptoms."+x, locale)
		case name == "preg_fetal_status":
			return enums.FetalMovementStatus(x).Label(locale)
		case name == "preg_basis":
			return enums.PregnancyAgeSource(x).Label(locale)
		}
		if d, err := civildate.Parse(x); err == nil && len(x) == len("2006-01-02") {
			return v2.FullDate(d, locale)
		}
		if t, err := time.ParseInLocation("2006-01-02 15:04", x, civildate.Tehran); err == nil {
			return v2.FullDate(civildate.FromTime(t), locale) + " " + v2.Digits(t.Format("15:04"), locale)
		}
		return v2.Digits(x, locale)
	case bool:
		return strconv.FormatBool(x)
	}
	return fmt.Sprint(v)
}

// factDay is the day the alert's fact happened (the evidence date variable), else its last hit's day.
func factDay(r *row, vars map[string]any) (civildate.Date, bool) {
	for _, k := range []string{"preg_symptom_on", "preg_fetal_on", "preg_bp_on", "preg_gdm_on", "preg_mood_on", "preg_week_on"} {
		if s, ok := vars[k].(string); ok {
			if d, err := civildate.Parse(s); err == nil {
				return d, true
			}
		}
	}
	for _, k := range []string{"preg_kick_at", "preg_labor_at"} {
		if s, ok := vars[k].(string); ok {
			if t, err := time.ParseInLocation("2006-01-02 15:04", s, civildate.Tehran); err == nil {
				return civildate.FromTime(t), true
			}
		}
	}
	at := r.LastAt
	if !at.Valid {
		at = r.FirstAt
	}
	if !at.Valid {
		return civildate.Date{}, false
	}
	return civildate.FromTime(at.Time.In(civildate.Tehran)), true
}

// alertJSON is a row in the v2 alert shape (GET /pregnancy/v2/alerts, the write responses): rule_key is the v2 key
// the clients know, level the spec level, alert_level its v1 enum.
func (s *Service) alertJSON(snap *snapshot, r *row, l v2.Lang) *jsonx.OrderedMap {
	d, _ := DefOf(r.RuleKey.String)
	key := d.Legacy
	if key == "" {
		key = r.RuleKey.String
	}
	level := rules.SpecLevel(r.Level)
	t := s.texts(snap, r, l.Locale)
	actions := []any{}
	for _, a := range d.Actions {
		key := a
		if a == "call_maternity" { // the call action with the maternity unit label (the client dials on "call")
			key = "call"
		}
		actions = append(actions, jsonx.Obj("key", key, "label", pregnancyalerts.T("actions."+a, l.Locale)))
	}
	var contact any
	if level == "urgent" && t.Contact != "" {
		contact = t.Contact
	}
	vars := r.vars()
	var week, created, factDate, dateLabel any
	if w, ok := vars["preg_week"].(float64); ok {
		week = int(w)
	}
	if r.FirstAt.Valid {
		created = r.FirstAt.Time.In(civildate.Tehran).Format(time.RFC3339)
	}
	if day, ok := factDay(r, vars); ok {
		factDate, dateLabel = day.String(), v2.FullDate(day, l.Locale)
	}
	title := t.Title
	if title == "" {
		title = key
	}
	return jsonx.Obj(
		"id", r.ID,
		"rule_key", key,
		"level", level,
		"alert_level", pregnancyalerts.V1Level(level),
		"title", title,
		"what_we_saw", nilIfEmpty(t.Seen),
		"how_sure", nilIfEmpty(t.Confidence),
		"advice", nilIfEmpty(t.Body),
		"actions", actions,
		"contact", contact,
		"pregnancy_week", week,
		"created_at", created,
		"fact_date", factDate,
		"date_label", dateLabel,
		"is_read", r.State != inbox.StateActive,
		"is_acked", r.AckedAt.Valid,
	)
}

// contactPhoneParam is the optional param of a legacy pregnancy_alert row (message_contents, B-N5-09: the maternity
// unit's number of contractions_511) whose phone the urgent alert's call action dials.
const contactPhoneParam = "contact_phone"

// withContactPhone turns the contact of an urgent alert into {text, phone} when its legacy pregnancy_alert row
// (message_contents, request locale else the default) has a contact_phone param, so the app's «تماس» dials it; without
// one the contact stays the plain text.
func withContactPhone(ctx context.Context, q pstore.Querier, l v2.Lang, list []*jsonx.OrderedMap) ([]*jsonx.OrderedMap, error) {
	need := false
	for _, a := range list {
		if c, _ := a.Get("contact"); c != nil {
			need = true
		}
	}
	if !need {
		return list, nil
	}
	rows, err := q.ListLiveMessageGroup(ctx, pregnancyalerts.Group)
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: contact phone: %w", err)
	}
	phones := map[string]string{}
	for _, loc := range []string{l.Default, l.Locale} { // the request locale wins
		for _, r := range rows {
			if r.Locale != loc {
				continue
			}
			var p struct {
				Params map[string]any `json:"params"`
			}
			if json.Unmarshal(r.Payload, &p) != nil {
				continue
			}
			if ph, ok := p.Params[contactPhoneParam].(string); ok && strings.TrimSpace(ph) != "" {
				phones[r.ItemKey] = strings.TrimSpace(ph)
			}
		}
	}
	for _, a := range list {
		c, _ := a.Get("contact")
		text, ok := c.(string)
		key, _ := a.Get("rule_key")
		k, _ := key.(string)
		if ok && phones[k] != "" {
			a.Set("contact", jsonx.Obj("text", text, "phone", phones[k]))
		}
	}
	return list, nil
}

// V1Trigger is the evidence of a rule in the v1 alert terms (pregnancy_alerts.trigger_symptoms of the retired v1
// rules): the BP numbers (+ severe), the sugar values as decimal strings, the fetal movement status, the critical
// symptoms, multiple_severe; `[]` for the rules v1 did not have.
func V1Trigger(key string, vars map[string]any) *jsonx.OrderedMap {
	num := func(name string) any {
		s, _ := vars[name].(string)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
		return nil
	}
	dec := func(name string) any {
		s, _ := vars[name].(string)
		if d, err := jsonx.Decimal(s, 2); err == nil {
			return d
		}
		return nil
	}
	switch key {
	case "preg.bp_high", "preg.bp_high_urgent":
		return jsonx.Obj("systolic", num("preg_bp_sys"), "diastolic", num("preg_bp_dia"))
	case "preg.bp_severe":
		return jsonx.Obj("systolic", num("preg_bp_sys"), "diastolic", num("preg_bp_dia"), "severe", true)
	case "preg.gdm":
		return jsonx.Obj("fasting", dec("preg_fasting"), "post_meal", dec("preg_post_meal"))
	case "preg.fetal_movement":
		return jsonx.Obj("fetal_movement", vars["preg_fetal_status"])
	case "preg.fetal_movement_kicks":
		return jsonx.Obj("fetal_movement", facts.FetalLowSession)
	case "preg.critical_symptom":
		o := jsonx.NewObject()
		list, _ := vars["preg_symptoms"].([]any)
		for _, x := range list {
			if name, ok := x.(string); ok {
				o.Set(name, true)
			}
		}
		return o
	case "preg.severe_symptoms":
		return jsonx.Obj("multiple_severe", true)
	}
	return jsonx.NewArray()
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
