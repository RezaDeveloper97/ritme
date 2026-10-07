package sharelinks

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/sharelinks/store"
)

// Errors.
var (
	ErrNotFound = errors.New("sharelinks: link not found")
	ErrExpired  = errors.New("sharelinks: link expired")
	ErrRevoked  = errors.New("sharelinks: link revoked")
	ErrLimit    = errors.New("sharelinks: too many active links")
)

// Reports builds the share-audience report (internal/healthrecord.Service.BuildReport): {range, record}.
type Reports interface {
	BuildReport(ctx context.Context, userID uint64, req healthrecord.ReportRequest, now time.Time, locale, def string,
	) (*jsonx.OrderedMap, error)
}

// Service creates, lists, revokes and opens share links. Owner calls are scoped by the user id they are given.
type Service struct {
	q       store.Querier
	db      *sql.DB // WithDB: link creates run count + insert in one transaction under the user row lock (B-N6-04b)
	reports Reports
	rand    io.Reader
	logger  *slog.Logger
	// CB-REC-03 (summary.go): the 24h code pepper, the owner's picked documents and their file links.
	coder *Coder
	docs  DocumentSource
	files FileLinker
}

// NewService wires the service.
func NewService(q store.Querier, reports Reports, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{q: q, reports: reports, rand: rand.Reader, logger: logger} // coder nil = codes off until WithCodes (fail closed)
}

// WithDB gives the service the database handle its creates take the user row lock on, so the active-link cap holds
// under concurrent POSTs. Without it (unit wiring) count and insert run unlocked.
func (s *Service) WithDB(db *sql.DB) *Service { s.db = db; return s }

// inTx runs fn on queries bound to one transaction (or on s.q without WithDB).
func (s *Service) inTx(ctx context.Context, fn func(q store.Querier) error) error {
	if s.db == nil {
		return fn(s.q)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sharelinks: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(store.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sharelinks: commit: %w", err)
	}
	return nil
}

// lockOwner takes the user row lock that serialises one user's link creates (a no-op without WithDB).
func (s *Service) lockOwner(ctx context.Context, q store.Querier, userID uint64) error {
	if s.db == nil {
		return nil
	}
	if _, err := q.LockShareLinkOwner(ctx, userID); err != nil {
		return fmt.Errorf("sharelinks: lock owner: %w", err)
	}
	return nil
}

// WithRand replaces the token / nonce source (tests).
func (s *Service) WithRand(r io.Reader) *Service { s.rand = r; return s }

func stamp(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

// Link is one link's metadata (never the report, never the token).
type Link struct {
	ID           uint64
	Sections     []string
	From, To     civildate.Date
	ExpiresAt    time.Time
	RevokedAt    time.Time
	Views        int
	LastViewedAt time.Time
	CreatedAt    time.Time
}

// Status is active | expired | revoked at now.
func (l Link) Status(now time.Time) string {
	switch {
	case !l.RevokedAt.IsZero():
		return StatusRevoked
	case !now.Before(l.ExpiresAt):
		return StatusExpired
	}
	return StatusActive
}

// JSON is the owner's view of the link at now.
func (l Link) JSON(now time.Time) *jsonx.OrderedMap {
	sections := l.Sections
	if sections == nil {
		sections = []string{}
	}
	return jsonx.Obj(
		"id", l.ID,
		"status", l.Status(now),
		"sections", sections,
		"range", jsonx.Obj("from", l.From.String(), "to", l.To.String()),
		"created_at", jsonx.ISO8601(l.CreatedAt),
		"expires_at", jsonx.ISO8601(l.ExpiresAt),
		"revoked_at", jsonx.ISO8601(l.RevokedAt),
		"views", l.Views,
		"last_viewed_at", jsonx.ISO8601(l.LastViewedAt),
	)
}

func nullTime(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func sectionsOf(raw json.RawMessage) []string {
	var out []string
	_ = json.Unmarshal(raw, &out)
	return out
}

func linkOf(r store.ListShareLinksRow) Link {
	return Link{
		ID: r.ID, Sections: sectionsOf(r.Sections), From: r.RangeFrom, To: r.RangeTo,
		ExpiresAt: nullTime(r.ExpiresAt), RevokedAt: nullTime(r.RevokedAt), Views: int(r.ViewCount),
		LastViewedAt: nullTime(r.LastViewedAt), CreatedAt: nullTime(r.CreatedAt),
	}
}

// Created is a new link and its token (shown to the owner once; the server cannot recover it).
type Created struct {
	Link  Link
	Token string
}

// Create builds the report of userID for req, seals it under a new token and stores the link (ErrLimit past
// MaxActive live links).
func (s *Service) Create(ctx context.Context, userID uint64, req healthrecord.ReportRequest, now time.Time,
	locale, def string,
) (Created, error) {
	// Unlocked fast path: a user at the cap gets the 409 before the report is built (re-checked under the lock below).
	n, err := s.q.CountActiveShareLinks(ctx, store.CountActiveShareLinksParams{UserID: userID, Now: stamp(now)})
	if err != nil {
		return Created{}, fmt.Errorf("sharelinks: count: %w", err)
	}
	if n >= MaxActive {
		return Created{}, ErrLimit
	}
	body, err := s.reports.BuildReport(ctx, userID, req, now, locale, def)
	if err != nil {
		return Created{}, err
	}
	var question any
	if req.Question != "" {
		question = req.Question
	}
	rng, _ := body.Get("range")
	rec, _ := body.Get("record")
	snapshot := jsonx.Obj("version", 1, "locale", locale, "created_at", jsonx.ISO8601(now), "range", rng,
		"record", rec, "question", question)
	plain, err := jsonx.Marshal(snapshot, jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	if err != nil {
		return Created{}, fmt.Errorf("sharelinks: snapshot: %w", err)
	}
	token, err := NewToken(s.rand)
	if err != nil {
		return Created{}, err
	}
	payload, err := Seal(token, plain, s.rand)
	if err != nil {
		return Created{}, err
	}
	sections, _ := json.Marshal(req.Sections)
	today := civildate.InTehran(now)
	from, to := req.Window(today)
	expires := now.Add(LinkTTL)
	var id int64
	// The cap check and the insert share one transaction under the user row lock: parallel POSTs cannot pass MaxActive.
	err = s.inTx(ctx, func(q store.Querier) error {
		if err := s.lockOwner(ctx, q, userID); err != nil {
			return err
		}
		n, err := q.CountActiveShareLinks(ctx, store.CountActiveShareLinksParams{UserID: userID, Now: stamp(now)})
		if err != nil {
			return fmt.Errorf("sharelinks: count: %w", err)
		}
		if n >= MaxActive {
			return ErrLimit
		}
		id, err = q.InsertShareLink(ctx, store.InsertShareLinkParams{
			UserID: userID, TokenHash: HashToken(token), Payload: sql.NullString{String: payload, Valid: true},
			Sections: sections, RangeFrom: from, RangeTo: to, ExpiresAt: stamp(expires), Now: stamp(now),
		})
		if err != nil {
			return fmt.Errorf("sharelinks: insert: %w", err)
		}
		return nil
	})
	if err != nil {
		return Created{}, err
	}
	return Created{
		Link: Link{
			ID: uint64(id), Sections: req.Sections, From: from, To: to, //nolint:gosec // auto-increment id
			ExpiresAt: expires, CreatedAt: now,
		},
		Token: token,
	}, nil
}

// List is the owner's links created within ListWindow, newest first.
func (s *Service) List(ctx context.Context, userID uint64, now time.Time) ([]Link, error) {
	rows, err := s.q.ListShareLinks(ctx, store.ListShareLinksParams{UserID: userID, Since: stamp(now.Add(-ListWindow))})
	if err != nil {
		return nil, fmt.Errorf("sharelinks: list: %w", err)
	}
	out := make([]Link, 0, len(rows))
	for _, r := range rows {
		out = append(out, linkOf(r))
	}
	return out, nil
}

// Revoke ends a link of userID now and wipes its report (ErrNotFound for a foreign or unknown id). Revoking twice
// keeps the first revocation time.
func (s *Service) Revoke(ctx context.Context, userID, id uint64, now time.Time) (Link, error) {
	n, err := s.q.RevokeShareLink(ctx, store.RevokeShareLinkParams{ID: id, UserID: userID, Now: stamp(now)})
	if err != nil {
		return Link{}, fmt.Errorf("sharelinks: revoke: %w", err)
	}
	if n == 0 {
		return Link{}, ErrNotFound
	}
	r, err := s.q.GetShareLinkMeta(ctx, store.GetShareLinkMetaParams{ID: id, UserID: userID})
	if err != nil {
		return Link{}, fmt.Errorf("sharelinks: get: %w", err)
	}
	return linkOf(store.ListShareLinksRow(r)), nil
}

// Opened is a decrypted report.
type Opened struct {
	Report    phpval.Map // {version, locale, created_at, range, record, question}
	ExpiresAt time.Time
}

// Open decrypts the report behind token and counts the view: ErrNotFound for an unknown token, ErrRevoked /
// ErrExpired for a report link that stopped working (a 24h summary link answers ErrNotFound then, CB-REC-03). The
// access log entry carries no client class; handlers call OpenAs.
func (s *Service) Open(ctx context.Context, token string, now time.Time) (Opened, error) {
	return s.OpenAs(ctx, token, now, Viewer{Via: ViaLink, Device: DeviceUnknown, Browser: BrowserOther})
}

// Purge wipes the reports of expired links and deletes rows RetainAfterExpiry after expiry.
func (s *Service) Purge(ctx context.Context, now time.Time) error {
	if _, err := s.q.PurgeExpiredShareLinks(ctx, store.PurgeExpiredShareLinksParams{Now: stamp(now)}); err != nil {
		return fmt.Errorf("sharelinks: purge: %w", err)
	}
	if _, err := s.q.DeleteStaleShareLinks(ctx, stamp(now.Add(-RetainAfterExpiry))); err != nil {
		return fmt.Errorf("sharelinks: delete stale: %w", err)
	}
	return nil
}

// PurgeLoop runs Purge now and every PurgeEvery until ctx ends (errors are logged without any link data).
func (s *Service) PurgeLoop(ctx context.Context, now func() time.Time) {
	t := time.NewTicker(PurgeEvery)
	defer t.Stop()
	for {
		if err := s.Purge(ctx, now()); err != nil && ctx.Err() == nil {
			s.logger.Warn("share link purge failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
