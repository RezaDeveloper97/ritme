// Package inbox is the alerts inbox (T-M10-07, D-164; SPEC §5.4.1; DECISIONS.md M-09, M-16): every mode's alerts are
// messages of slot `alerts_inbox`, selected by the same engine (all eligible, no history caps) and kept in
// `message_inbox` with a lifecycle — active → acknowledged / added_to_visit_note / resolved → archived after 30 days.
//
// Evaluate diffs the slot's matches with the user's open rows: a new match opens an active row (level ≥ follow_up only
// with its evidence); the same message again merges into its open row (same evidence key → nothing, new key → count
// + 1, last_at, new evidence; an acknowledged urgent row is active again, M-16); an open row whose conditions no
// longer hold over known facts is resolved. Nothing is written when nothing changed.
package inbox

import (
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// Slot is the inbox slot.
const Slot = rules.SlotAlertsInbox

// States (open: active, acknowledged, added_to_visit_note; history: resolved, archived).
const (
	StateActive       = feed.StateActive
	StateAcknowledged = feed.StateAcknowledged
	StateNoted        = feed.StateNoted
	StateResolved     = "resolved"
	StateArchived     = "archived"
)

var (
	missingEvidence = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "inbox_missing_evidence_total",
		Help: "Alerts inbox matches at level follow_up or urgent that opened no row because their evidence (what we saw / how sure) could not be filled.",
	})
	pushSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_push_slot_sent_total",
		Help: "In-app notifications written by the push slot (messages.push_slot).",
	})
	jobUserErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "messages_inbox_job_user_errors_total",
		Help: "Users skipped by an inbox job because their evaluation failed.",
	}, []string{"job"})
)

func init() { metrics.Registry().MustRegister(missingEvidence, pushSent, jobUserErrors) }

// The inbox copy (API messages, the neutral push line) is data: lang/<code>/inbox.json; a language without the file
// (or a key) falls back to English (lang.FallbackLocale).
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the inbox line for key ("messages.ack_required") in locale.
func T(key, locale string) string { return translator().Trans("inbox."+key, nil, locale) }

// UserFacts builds a user's facts outside a request (jobs, EvaluateNow): every fact, and the user-local now of the
// instant at.
type UserFacts func(ctx context.Context, userID uint64, at time.Time) (rules.Facts, time.Time, error)

// Languages are the codes of the active languages (i18n.Registry; the list is data).
type Languages func(ctx context.Context) []string

// Service is the alerts inbox.
type Service struct {
	q      store.Querier
	db     *sql.DB // optional: the push slot's two writes in one transaction
	load   func(ctx context.Context) (*compiled.Compiled, error)
	facts  UserFacts
	langs  Languages
	prefs  notifications.Getter
	now    func() time.Time
	logger *slog.Logger
	need   atomic.Pointer[needMemo]
}

type needMemo struct {
	comp *compiled.Compiled
	need []string
}

// New wires the service on the messages store; db may be nil.
func New(q store.Querier, sqlDB *sql.DB, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{q: q, db: sqlDB, logger: logger, now: time.Now,
		load: func(ctx context.Context) (*compiled.Compiled, error) { return compiled.Load(ctx, q) }}
}

// WithFacts sets the out-of-request fact builder (EvaluateNow, the jobs).
func (s *Service) WithFacts(f UserFacts) *Service { s.facts = f; return s }

// WithLanguages sets the language list (the push slot's neutral text in every language).
func (s *Service) WithLanguages(l Languages) *Service { s.langs = l; return s }

// WithPreferences sets the notification preferences store (the push slot).
func (s *Service) WithPreferences(p notifications.Getter) *Service { s.prefs = p; return s }

// WithLoader replaces the compiled snapshot source (tests).
func (s *Service) WithLoader(load func(ctx context.Context) (*compiled.Compiled, error)) *Service {
	s.load = load
	return s
}

// WithClock replaces the wall clock (tests).
func (s *Service) WithClock(now func() time.Time) *Service { s.now = now; return s }

var _ feed.Inbox = (*Service)(nil)

// Need are the facts the slot reads: its conditions and every variable of its texts and evidence (cached per
// snapshot).
func (s *Service) Need(comp *compiled.Compiled) []string {
	if m := s.need.Load(); m != nil && m.comp == comp {
		return m.need
	}
	out := slices.Clone(comp.Need(Slot))
	for _, r := range comp.BySlot[Slot] {
		out = append(out, rules.Placeholders(allTexts(&r.Src.Content)...)...)
	}
	slices.Sort(out)
	out = slices.Compact(out)
	s.need.Store(&needMemo{comp: comp, need: out})
	return out
}

// allTexts are every text of the content: the variants and the evidence fields.
func allTexts(c *rules.Content) []string {
	return append(c.Texts(), c.Seen, c.Confidence, c.Contact)
}

// Evaluate runs the alerts_inbox slot for userID over f at now (the user-local clock) and diffs it with her open rows;
// it returns the open rows after the diff, newest first.
func (s *Service) Evaluate(ctx context.Context, userID uint64, f rules.Facts, now time.Time) ([]feed.OpenAlert, error) {
	comp, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	return s.Sync(ctx, comp, userID, f, now, -1)
}

// Sync is Evaluate on a loaded snapshot (the feed). open is the known open count: 0 skips reading the open rows,
// < 0 = unknown (read them).
func (s *Service) Sync(ctx context.Context, comp *compiled.Compiled, userID uint64, f rules.Facts, now time.Time,
	open int64,
) ([]feed.OpenAlert, error) {
	if len(comp.BySlot[Slot]) == 0 && open == 0 {
		return nil, nil
	}
	var rows []store.ListUserOpenInboxRow
	if open != 0 {
		var err error
		if rows, err = s.q.ListUserOpenInbox(ctx, userID); err != nil {
			return nil, fmt.Errorf("inbox: open rows: %w", err)
		}
	}
	return s.SyncRows(ctx, comp, userID, f, now, rows)
}

// SyncRows is Sync over the open rows the caller already read (feed.RowSyncer, T-M9-49b).
func (s *Service) SyncRows(ctx context.Context, comp *compiled.Compiled, userID uint64, f rules.Facts, now time.Time,
	rows []store.ListUserOpenInboxRow,
) ([]feed.OpenAlert, error) {
	if len(comp.BySlot[Slot]) == 0 && len(rows) == 0 {
		return nil, nil
	}
	winners := comp.Select(Slot, f, nil, userID, now)
	if len(winners) == 0 && len(rows) == 0 {
		return nil, nil
	}
	p := Diff(winners, rows, ruleIndex(comp), f)
	if p.Missing > 0 {
		missingEvidence.Add(float64(p.Missing))
	}
	if err := s.apply(ctx, userID, p, now); err != nil {
		return nil, err
	}
	return p.Open, nil
}

// ruleIndex are the served inbox rules by id.
func ruleIndex(comp *compiled.Compiled) map[uint64]*rules.Rule {
	list := comp.BySlot[Slot]
	out := make(map[uint64]*rules.Rule, len(list))
	for _, r := range list {
		out[r.Src.ID] = r.Src
	}
	return out
}

// Match is one evaluated match with its evidence.
type Match struct {
	Rule     *rules.Rule
	Key      string
	Evidence json.RawMessage
}

// Merge is an open row that fired again with new evidence.
type Merge struct {
	Match
	RowID   uint64
	State   string
	AckedAt sql.NullTime
}

// Plan is what one evaluation writes, and the open rows after it.
type Plan struct {
	Inserts  []Match
	Merges   []Merge
	Resolves []uint64
	Missing  int // matches at level ≥ follow_up without evidence (no row)
	Open     []feed.OpenAlert
}

// Changed reports whether the plan writes anything.
func (p *Plan) Changed() bool { return len(p.Inserts)+len(p.Merges)+len(p.Resolves) > 0 }

// Diff is the pure diff of the slot's winners with the open rows; byID are the served inbox rules (an open row whose
// rule is not served any more is left alone).
func Diff(winners []rules.Choice, rows []store.ListUserOpenInboxRow, byID map[uint64]*rules.Rule, f rules.Facts) Plan {
	var p Plan
	matched := make(map[uint64]Match, len(winners))
	noEvidence := map[uint64]bool{}
	for _, w := range winners {
		ev, key, ok := EvidenceOf(w.Rule, f)
		if !ok {
			noEvidence[w.Rule.ID] = true
			continue
		}
		matched[w.Rule.ID] = Match{Rule: w.Rule, Key: key, Evidence: ev}
	}
	seen := map[uint64]bool{}
	for i := range rows {
		row := &rows[i]
		seen[row.RuleID] = true
		o := feed.OpenAlert{ID: row.ID, Rule: byID[row.RuleID], Kind: row.Kind, Level: row.Level, State: row.State,
			LastAt: row.LastAt.Time}
		m, hit := matched[row.RuleID]
		switch {
		case hit && m.Key != row.EvidenceKey.String:
			mg := Merge{Match: m, RowID: row.ID, State: row.State, AckedAt: row.AckedAt}
			if m.Rule.Level == rules.LevelFollowSoon && row.State != StateActive {
				mg.State, mg.AckedAt = StateActive, sql.NullTime{} // M-16: urgent must be acknowledged again
			}
			p.Merges = append(p.Merges, mg)
			o.State, o.Level, o.LastAt = mg.State, m.Rule.Level, time.Time{} // newest
		case hit || noEvidence[row.RuleID]:
			// same evidence, or the condition still holds without fillable evidence: unchanged
		case o.Rule != nil && !rules.MatchRule(o.Rule, f) && factsKnown(o.Rule, f):
			p.Resolves = append(p.Resolves, row.ID)
			continue
		}
		p.Open = append(p.Open, o)
	}
	for _, w := range winners {
		id := w.Rule.ID
		if seen[id] {
			continue
		}
		if noEvidence[id] {
			p.Missing++
			continue
		}
		p.Inserts = append(p.Inserts, matched[id])
		p.Open = append(p.Open, feed.OpenAlert{Rule: w.Rule, Kind: w.Rule.Kind, Level: w.Rule.Level, State: StateActive})
	}
	// newest first: rows written now (zero LastAt) lead, then last_at DESC, id DESC
	slices.SortStableFunc(p.Open, func(a, b feed.OpenAlert) int {
		az, bz := a.LastAt.IsZero(), b.LastAt.IsZero()
		if az != bz {
			if az {
				return -1
			}
			return 1
		}
		if c := b.LastAt.Compare(a.LastAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	return p
}

// factsKnown reports whether every fact the rule's condition groups read is present (a missing fact never resolves).
func factsKnown(r *rules.Rule, f rules.Facts) bool {
	for _, g := range r.Groups {
		for _, c := range g.Items {
			if _, ok := f[c.Fact]; !ok {
				return false
			}
		}
	}
	return true
}

// evidenceDoc is the stored evidence: the values of the variables the rule's texts read, so a row renders in any
// language later with the data of the hit.
type evidenceDoc struct {
	Vars map[string]any `json:"vars"`
}

// EvidenceOf is the stored evidence and the evidence key of a match. ok is false for a level ≥ follow_up match whose
// «what we saw» / «how sure» cannot be filled from f (SPEC §5.4.1: no row without them). The key digests the values
// of the evidence variables (seen / confidence / contact), or of the condition facts when the evidence has none: the
// same data → the same key.
func EvidenceOf(r *rules.Rule, f rules.Facts) (json.RawMessage, string, bool) {
	c := &r.Content
	if r.Level == rules.LevelWorthFollowUp || r.Level == rules.LevelFollowSoon {
		seen, ok1 := rules.Render(c.Seen, f)
		conf, ok2 := rules.Render(c.Confidence, f)
		if !ok1 || !ok2 || seen == "" || conf == "" {
			return nil, "", false
		}
	}
	vars := map[string]any{}
	for _, name := range rules.Placeholders(allTexts(c)...) {
		if v, ok := f[name]; ok {
			vars[name] = v
		}
	}
	keyed := map[string]any{}
	for _, name := range rules.Placeholders(c.Seen, c.Confidence, c.Contact) {
		if v, ok := f[name]; ok {
			keyed[name] = v
		}
	}
	if len(keyed) == 0 {
		for _, g := range r.Groups {
			for _, it := range g.Items {
				if v, ok := f[it.Fact]; ok {
					keyed[it.Fact] = v
				}
			}
		}
	}
	ev, err := json.Marshal(evidenceDoc{Vars: vars}) // map keys are sorted: canonical
	if err != nil {
		return nil, "", false
	}
	k, err := json.Marshal(keyed)
	if err != nil {
		return nil, "", false
	}
	sum := sha256.Sum256(append([]byte(fmt.Sprintf("%d:", r.ID)), k...))
	return ev, hex.EncodeToString(sum[:16]), true
}

// varsOf decodes the stored evidence variables.
func varsOf(raw db.NullRawJSON) rules.Facts {
	if !raw.Valid {
		return rules.Facts{}
	}
	var d evidenceDoc
	if json.Unmarshal(raw.V, &d) != nil || d.Vars == nil {
		return rules.Facts{}
	}
	return rules.Facts(d.Vars)
}

// apply writes the plan; rows a concurrent request already changed are left as that request wrote them.
func (s *Service) apply(ctx context.Context, userID uint64, p Plan, now time.Time) error {
	if !p.Changed() {
		return nil
	}
	ts := sql.NullTime{Time: now, Valid: true}
	for _, m := range p.Inserts {
		if _, err := s.q.InsertInboxItem(ctx, store.InsertInboxItemParams{
			UserID: userID, RuleID: m.Rule.ID, Version: servedVersion(m.Rule), RuleKey: nullStr(m.Rule.RuleKey),
			Kind: m.Rule.Kind, Level: m.Rule.Level, EvidenceKey: nullStr(m.Key),
			Evidence: db.NullRawJSON{V: m.Evidence, Valid: true}, Now: ts,
		}); err != nil {
			return fmt.Errorf("inbox: open row: %w", err)
		}
	}
	for _, m := range p.Merges {
		if _, err := s.q.MergeInboxItem(ctx, store.MergeInboxItemParams{
			Now: ts, EvidenceKey: nullStr(m.Key), Evidence: db.NullRawJSON{V: m.Evidence, Valid: true},
			Version: servedVersion(m.Rule), Level: m.Rule.Level, State: m.State, AckedAt: m.AckedAt,
			ID: m.RowID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("inbox: merge row: %w", err)
		}
	}
	for _, id := range p.Resolves {
		if _, err := s.q.ResolveInboxItem(ctx, store.ResolveInboxItemParams{Now: ts, ID: id, UserID: userID}); err != nil {
			return fmt.Errorf("inbox: resolve row: %w", err)
		}
	}
	return nil
}

func servedVersion(r *rules.Rule) uint32 { return uint32(max(feed.ServedVersion(r), 0)) } //nolint:gosec // small

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

var defaultService atomic.Pointer[Service]

// Install makes s the service EvaluateNow uses (wired once at mount).
func Install(s *Service) { defaultService.Store(s) }

// EvaluateNow evaluates userID's inbox now, for a write path that needs the alert in its own response (T-M10-08):
// her facts, her local clock, the open rows after the diff. Without an installed service it does nothing.
func EvaluateNow(ctx context.Context, userID uint64) ([]feed.OpenAlert, error) {
	s := defaultService.Load()
	if s == nil {
		return nil, nil
	}
	return s.EvaluateUser(ctx, userID, s.now())
}

// EvaluateUser builds userID's facts at the instant at (UserFacts) and evaluates her inbox.
func (s *Service) EvaluateUser(ctx context.Context, userID uint64, at time.Time) ([]feed.OpenAlert, error) {
	if s.facts == nil {
		return nil, fmt.Errorf("inbox: no fact builder wired")
	}
	f, local, err := s.facts(ctx, userID, at)
	if err != nil {
		return nil, err
	}
	return s.Evaluate(ctx, userID, f, local)
}
