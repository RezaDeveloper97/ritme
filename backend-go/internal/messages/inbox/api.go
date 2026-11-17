package inbox

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// PageSize is the rows of one GET /alerts page.
const PageSize = 20

// List states.
const (
	ListOpen    = "open"
	ListHistory = "history"
)

// Errors of the actions (the handlers map them).
var (
	ErrNotFound    = errors.New("inbox: not found")      // no such row for this user → 404
	ErrAckRequired = errors.New("inbox: ack required")   // urgent, not acknowledged → 409 ack_required
	ErrClosed      = errors.New("inbox: alert closed")   // ack / visit note on a history row → 409 alert_closed
	ErrBadCursor   = errors.New("inbox: invalid cursor") // → 422
)

// Row is one stored inbox row (the shape every read returns).
type Row struct {
	ID, UserID, RuleID uint64
	Version            uint32
	RuleKey            sql.NullString
	Kind, Level, State string
	EvidenceKey        sql.NullString
	HitCount           uint32
	Evidence           db.NullRawJSON
	FirstAt, LastAt    sql.NullTime
	AckedAt, NotedAt   sql.NullTime
	ResolvedAt         sql.NullTime
	ArchivedAt         sql.NullTime
	rank               int64
}

func rowOf(r store.GetUserInboxItemRow) Row {
	return Row{ID: r.ID, UserID: r.UserID, RuleID: r.RuleID, Version: r.Version, RuleKey: r.RuleKey, Kind: r.Kind,
		Level: r.Level, State: r.State, EvidenceKey: r.EvidenceKey, HitCount: r.HitCount, Evidence: r.Evidence,
		FirstAt: r.FirstAt, LastAt: r.LastAt, AckedAt: r.AckedAt, NotedAt: r.NotedAt, ResolvedAt: r.ResolvedAt,
		ArchivedAt: r.ArchivedAt}
}

// Open reports whether the row is open.
func (r *Row) Open() bool {
	return r.State == StateActive || r.State == StateAcknowledged || r.State == StateNoted
}

// Closable reports whether the user may dismiss the row: an urgent row only once acknowledged (SPEC §5.4.1) — a
// resolved one is already closed — and never an archived one.
func (r *Row) Closable() bool {
	if r.State == StateArchived {
		return false
	}
	return r.Level != rules.LevelFollowSoon || r.State == StateResolved || r.AckedAt.Valid && r.State != StateActive
}

// Cursor is a page position: (level rank, last_at, id) of the last row of the previous page.
type Cursor struct {
	Rank int64
	At   time.Time
	ID   uint64
}

// Encode is the opaque cursor string.
func (c Cursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil, "%d:%d:%d", c.Rank, c.At.Unix(), c.ID))
}

// ParseCursor decodes a cursor ("" = the first page).
func ParseCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrBadCursor
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 3 {
		return nil, ErrBadCursor
	}
	rank, err1 := strconv.ParseInt(parts[0], 10, 64)
	at, err2 := strconv.ParseInt(parts[1], 10, 64)
	id, err3 := strconv.ParseUint(parts[2], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || rank < 0 || rank > 3 {
		return nil, ErrBadCursor
	}
	return &Cursor{Rank: rank, At: time.Unix(at, 0).In(civildate.Tehran), ID: id}, nil
}

// Page is one list page.
type Page struct {
	Rows []Row
	Next *Cursor
}

// List is one page of the user's open or history rows, urgent → follow_up → suggestion → info, then newest first.
func (s *Service) List(ctx context.Context, userID uint64, state string, cur *Cursor) (Page, error) {
	var has, rank int64
	var at sql.NullTime
	var after uint64
	if cur != nil {
		has, rank, at, after = 1, cur.Rank, sql.NullTime{Time: cur.At, Valid: true}, cur.ID
	}
	var rows []Row
	if state == ListHistory {
		list, err := s.q.ListUserInboxHistoryPage(ctx, store.ListUserInboxHistoryPageParams{
			UserID: userID, HasCursor: has, AfterRank: rank, AfterAt: at, AfterID: after, Limit: PageSize + 1})
		if err != nil {
			return Page{}, fmt.Errorf("inbox: history page: %w", err)
		}
		for _, r := range list {
			rows = append(rows, rowOf(store.GetUserInboxItemRow{ID: r.ID, UserID: r.UserID, RuleID: r.RuleID,
				Version: r.Version, RuleKey: r.RuleKey, Kind: r.Kind, Level: r.Level, State: r.State,
				EvidenceKey: r.EvidenceKey, HitCount: r.HitCount, Evidence: r.Evidence, FirstAt: r.FirstAt,
				LastAt: r.LastAt, AckedAt: r.AckedAt, NotedAt: r.NotedAt, ResolvedAt: r.ResolvedAt,
				ArchivedAt: r.ArchivedAt}))
			rows[len(rows)-1].rank = r.LevelRank
		}
	} else {
		list, err := s.q.ListUserInboxOpenPage(ctx, store.ListUserInboxOpenPageParams{
			UserID: userID, HasCursor: has, AfterRank: rank, AfterAt: at, AfterID: after, Limit: PageSize + 1})
		if err != nil {
			return Page{}, fmt.Errorf("inbox: open page: %w", err)
		}
		for _, r := range list {
			rows = append(rows, rowOf(store.GetUserInboxItemRow{ID: r.ID, UserID: r.UserID, RuleID: r.RuleID,
				Version: r.Version, RuleKey: r.RuleKey, Kind: r.Kind, Level: r.Level, State: r.State,
				EvidenceKey: r.EvidenceKey, HitCount: r.HitCount, Evidence: r.Evidence, FirstAt: r.FirstAt,
				LastAt: r.LastAt, AckedAt: r.AckedAt, NotedAt: r.NotedAt, ResolvedAt: r.ResolvedAt,
				ArchivedAt: r.ArchivedAt}))
			rows[len(rows)-1].rank = r.LevelRank
		}
	}
	p := Page{Rows: rows}
	if len(rows) > PageSize {
		p.Rows = rows[:PageSize]
		last := p.Rows[PageSize-1]
		p.Next = &Cursor{Rank: last.rank, At: last.LastAt.Time, ID: last.ID}
	}
	return p, nil
}

// get is the user's row id (ErrNotFound for another user's or a missing id).
func (s *Service) get(ctx context.Context, userID, id uint64) (Row, error) {
	r, err := s.q.GetUserInboxItem(ctx, store.GetUserInboxItemParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Row{}, ErrNotFound
	}
	if err != nil {
		return Row{}, fmt.Errorf("inbox: get: %w", err)
	}
	return rowOf(r), nil
}

// Ack is «دیدم، ممنون»: an open row becomes acknowledged.
func (s *Service) Ack(ctx context.Context, userID, id uint64, now time.Time) (Row, error) {
	r, err := s.get(ctx, userID, id)
	if err != nil {
		return Row{}, err
	}
	if !r.Open() {
		return r, ErrClosed
	}
	if _, err := s.q.AckInboxItem(ctx, store.AckInboxItemParams{Now: sql.NullTime{Time: now, Valid: true}, ID: id,
		UserID: userID}); err != nil {
		return Row{}, fmt.Errorf("inbox: ack: %w", err)
	}
	return s.get(ctx, userID, id)
}

// Dismiss closes a row (→ archived): 409 ack_required for an urgent row not yet acknowledged. An archived row stays
// as it is (idempotent).
func (s *Service) Dismiss(ctx context.Context, userID, id uint64, now time.Time) (Row, error) {
	r, err := s.get(ctx, userID, id)
	if err != nil {
		return Row{}, err
	}
	if r.State == StateArchived {
		return r, nil
	}
	if !r.Closable() {
		return r, ErrAckRequired
	}
	if _, err := s.q.DismissInboxItem(ctx, store.DismissInboxItemParams{Now: sql.NullTime{Time: now, Valid: true},
		ID: id, UserID: userID}); err != nil {
		return Row{}, fmt.Errorf("inbox: dismiss: %w", err)
	}
	return s.get(ctx, userID, id)
}

// ruleOf is the rule a row was opened for: the served inbox rule, else the stored row (a rule no longer served still
// renders its texts in history).
func (s *Service) ruleOf(ctx context.Context, comp *compiled.Compiled, id uint64) (*rules.Rule, error) {
	for _, r := range comp.BySlot[Slot] {
		if r.Src.ID == id {
			return r.Src, nil
		}
	}
	row, err := s.q.GetMessageRule(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inbox: rule: %w", err)
	}
	r := rules.FromRow(&row)
	return &r, nil
}

type rowTexts struct {
	title, body, ctaText, ctaLink string
	seen, confidence, contact     string
}

// texts renders a row's texts in locale with the variables stored at its last hit.
func (s *Service) texts(ctx context.Context, comp *compiled.Compiled, r *Row, rule *rules.Rule, locale string) rowTexts {
	if rule == nil {
		return rowTexts{}
	}
	ren := feed.RendererOf(ctx, s.q, comp)
	vars := varsOf(r.Evidence)
	v := feed.RenderTexts(rule.RuleKey, ren.Text(rule, rule.Assign(r.UserID), locale), vars, locale)
	ev := ren.Evidence(rule, locale)
	t := rowTexts{title: v.Title, body: v.Body}
	if v.CTALabel != "" && v.CTATarget != "" {
		t.ctaText, t.ctaLink = v.CTALabel, v.CTATarget
	}
	t.seen = feed.RenderText(rule.RuleKey, ev.Seen, vars, locale)
	t.confidence = feed.RenderText(rule.RuleKey, ev.Confidence, vars, locale)
	t.contact = feed.RenderText(rule.RuleKey, ev.Contact, vars, locale)
	return t
}

// Items renders rows in locale (GET /alerts and the actions).
func (s *Service) Items(ctx context.Context, rows []Row, locale string) ([]*jsonx.OrderedMap, error) {
	comp, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	cache := map[uint64]*rules.Rule{}
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		rule, ok := cache[r.RuleID]
		if !ok {
			if rule, err = s.ruleOf(ctx, comp, r.RuleID); err != nil {
				return nil, err
			}
			cache[r.RuleID] = rule
		}
		out = append(out, itemJSON(r, s.texts(ctx, comp, r, rule, locale)))
	}
	return out, nil
}

// itemJSON is one inbox item. level is the spec name; evidence only for follow_up and urgent (contact for urgent).
func itemJSON(r *Row, t rowTexts) *jsonx.OrderedMap {
	var key, cta, link any
	if r.RuleKey.Valid {
		key = r.RuleKey.String
	}
	if t.ctaText != "" {
		cta, link = t.ctaText, t.ctaLink
	}
	o := jsonx.Obj("id", r.ID, "rule_id", r.RuleID, "rule_key", key, "version", r.Version, "kind", r.Kind,
		"level", rules.SpecLevel(r.Level), "state", r.State, "title", t.title, "body", t.body,
		"cta_text", cta, "cta_deeplink", link)
	if r.Level == rules.LevelWorthFollowUp || r.Level == rules.LevelFollowSoon {
		ev := jsonx.Obj("seen", t.seen, "confidence", t.confidence)
		if r.Level == rules.LevelFollowSoon {
			ev.Set("contact", t.contact)
		}
		o.Set("evidence", ev)
	} else {
		o.Set("evidence", nil)
	}
	o.Set("hit_count", r.HitCount)
	o.Set("dismissible", r.Closable())
	o.Set("first_at", iso(r.FirstAt))
	o.Set("last_at", iso(r.LastAt))
	o.Set("acked_at", iso(r.AckedAt))
	o.Set("noted_at", iso(r.NotedAt))
	o.Set("resolved_at", iso(r.ResolvedAt))
	o.Set("archived_at", iso(r.ArchivedAt))
	return o
}

// iso is an ISO-8601 Tehran time (null when unset).
func iso(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return t.Time.In(civildate.Tehran).Format(time.RFC3339)
}
