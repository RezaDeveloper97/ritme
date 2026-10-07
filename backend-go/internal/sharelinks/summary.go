package sharelinks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/sharelinks/store"
)

// The 24h record summary (canvas-build CB-REC-03, D-71; nbl_Rec_Share «پزشک — دسترسی موقت با کد»): «پزشک با اسکن کد،
// خلاصه پرونده را ۲۴ ساعت می‌بیند». A summary is a share link of kind `summary` next to bloom's 7-day `report`
// links (same table, same token + snapshot encryption, same purge): it lives SummaryTTL, it also opens with a short
// human code (code.go), it may carry documents the owner explicitly picked (documents.go), and once it expired or was
// revoked every public read is the uniform 404 — no 410 that would tell a guesser a code once existed. Every
// successful public open of either kind writes one access log row (viewer.go: link | code, coarse device + browser).

// Link kinds (health_share_links.kind).
const (
	KindReport  = "report"  // bloom B-N6-04: the 7-day doctor report link
	KindSummary = "summary" // CB-REC-03: the 24h summary with a code
)

// Summary limits.
const (
	SummaryTTL         = 24 * time.Hour
	MaxActiveSummaries = 5
	MaxLabelLen        = 60
	DefaultAccessLimit = 50
	MaxAccessLimit     = 200
	codeAttempts       = 4 // fresh codes tried when a code_hash collides
)

// Summary errors.
var (
	ErrSummaryLimit  = errors.New("sharelinks: too many active summary codes")
	ErrCodesDisabled = errors.New("sharelinks: doctor codes disabled (no SHARE_CODE_PEPPER in production)")
)

// WithCodes sets the code pepper; disabled (production without SHARE_CODE_PEPPER) turns summaries off (503).
func (s *Service) WithCodes(pepper []byte, disabled bool) *Service {
	if disabled {
		s.coder = nil
	} else {
		s.coder = NewCoder(pepper)
	}
	return s
}

// WithDocuments wires the owner's record documents and their file links (nil = summaries carry no documents).
func (s *Service) WithDocuments(docs DocumentSource, files FileLinker) *Service {
	s.docs, s.files = docs, files
	return s
}

// CodesDisabled reports whether summaries are off.
func (s *Service) CodesDisabled() bool { return s.coder == nil }

// SummaryRequest is a validated POST /health-record/share-codes.
type SummaryRequest struct {
	Report      healthrecord.ReportRequest // window + sections (no question)
	Label       string                     // "" = none
	DocumentIDs []uint64                   // explicitly picked record documents, in order
}

// Summary is one summary link's metadata for the owner (never the report, the token or the code).
type Summary struct {
	Link
	Label string
}

// JSON is the owner's view of the summary at now.
func (l Summary) JSON(now time.Time) *jsonx.OrderedMap {
	sections := l.Sections
	if sections == nil {
		sections = []string{}
	}
	return jsonx.Obj(
		"id", l.ID,
		"kind", KindSummary,
		"label", orNil(l.Label),
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

func summaryOf(r store.ListSummaryLinksRow) Summary {
	return Summary{
		Link: Link{
			ID: r.ID, Sections: sectionsOf(r.Sections), From: r.RangeFrom, To: r.RangeTo,
			ExpiresAt: nullTime(r.ExpiresAt), RevokedAt: nullTime(r.RevokedAt), Views: int(r.ViewCount),
			LastViewedAt: nullTime(r.LastViewedAt), CreatedAt: nullTime(r.CreatedAt),
		},
		Label: str(r.Label),
	}
}

// CreatedSummary is a new summary with its token (the QR URL) and its code — shown to the owner once; the server
// cannot recover either.
type CreatedSummary struct {
	Summary   Summary
	Token     string
	Code      string // unformatted
	Documents int
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// CreateSummary builds the share-audience record of userID (plus the picked documents), seals it under a new token,
// wraps the token under a new code and stores the 24h link (ErrSummaryLimit past MaxActiveSummaries live summaries,
// *ErrDocument for a foreign document id, ErrCodesDisabled without a pepper in production).
func (s *Service) CreateSummary(ctx context.Context, userID uint64, req SummaryRequest, now time.Time,
	locale, def string,
) (CreatedSummary, error) {
	if s.coder == nil {
		return CreatedSummary{}, ErrCodesDisabled
	}
	n, err := s.q.CountActiveSummaryLinks(ctx, store.CountActiveSummaryLinksParams{UserID: userID, Now: stamp(now)})
	if err != nil {
		return CreatedSummary{}, fmt.Errorf("sharelinks: count summaries: %w", err)
	}
	if n >= MaxActiveSummaries {
		return CreatedSummary{}, ErrSummaryLimit
	}
	var docs []SharedDocument
	if len(req.DocumentIDs) > 0 {
		if s.docs == nil {
			return CreatedSummary{}, &ErrDocument{Index: 0}
		}
		if docs, err = s.docs.SharedDocuments(ctx, userID, req.DocumentIDs); err != nil {
			return CreatedSummary{}, err
		}
	}
	body, err := s.reports.BuildReport(ctx, userID, req.Report, now, locale, def)
	if err != nil {
		return CreatedSummary{}, err
	}
	rng, _ := body.Get("range")
	rec, _ := body.Get("record")
	snapshot := jsonx.Obj("version", 1, "kind", KindSummary, "locale", locale, "created_at", jsonx.ISO8601(now),
		"range", rng, "record", rec, "question", nil, "documents", snapshotDocuments(docs))
	plain, err := jsonx.Marshal(snapshot, jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	if err != nil {
		return CreatedSummary{}, fmt.Errorf("sharelinks: snapshot: %w", err)
	}
	sections, _ := json.Marshal(req.Report.Sections)
	from, to := req.Report.Window(civildate.InTehran(now))
	expires := now.Add(SummaryTTL)
	for range codeAttempts {
		token, err := NewToken(s.rand)
		if err != nil {
			return CreatedSummary{}, err
		}
		payload, err := Seal(token, plain, s.rand)
		if err != nil {
			return CreatedSummary{}, err
		}
		code, err := NewCode(s.rand)
		if err != nil {
			return CreatedSummary{}, err
		}
		wrapped, err := s.coder.Wrap(code, token, s.rand)
		if err != nil {
			return CreatedSummary{}, err
		}
		id, err := s.q.InsertSummaryLink(ctx, store.InsertSummaryLinkParams{
			UserID: userID, Label: sql.NullString{String: req.Label, Valid: req.Label != ""}, TokenHash: HashToken(token),
			CodeHash:    sql.NullString{String: s.coder.Hash(code), Valid: true},
			CodePayload: sql.NullString{String: wrapped, Valid: true},
			Payload:     sql.NullString{String: payload, Valid: true}, Sections: sections, RangeFrom: from, RangeTo: to,
			ExpiresAt: stamp(expires), Now: stamp(now),
		})
		if isDuplicate(err) { // a live or not yet deleted code with the same HMAC: draw again
			continue
		}
		if err != nil {
			return CreatedSummary{}, fmt.Errorf("sharelinks: insert summary: %w", err)
		}
		return CreatedSummary{
			Summary: Summary{
				Link: Link{ID: uint64(id), Sections: req.Report.Sections, From: from, To: to, //nolint:gosec // auto-increment id
					ExpiresAt: expires, CreatedAt: now},
				Label: req.Label,
			},
			Token: token, Code: code, Documents: len(docs),
		}, nil
	}
	return CreatedSummary{}, errors.New("sharelinks: no free code after retries")
}

// ListSummaries is the owner's summaries created within ListWindow, newest first.
func (s *Service) ListSummaries(ctx context.Context, userID uint64, now time.Time) ([]Summary, error) {
	rows, err := s.q.ListSummaryLinks(ctx, store.ListSummaryLinksParams{UserID: userID, Since: stamp(now.Add(-ListWindow))})
	if err != nil {
		return nil, fmt.Errorf("sharelinks: list summaries: %w", err)
	}
	out := make([]Summary, 0, len(rows))
	for _, r := range rows {
		out = append(out, summaryOf(r))
	}
	return out, nil
}

// RevokeSummary ends a summary of userID now and wipes its report and wrapped token (ErrNotFound for a foreign or
// unknown id, or the id of a report link).
func (s *Service) RevokeSummary(ctx context.Context, userID, id uint64, now time.Time) (Summary, error) {
	n, err := s.q.RevokeSummaryLink(ctx, store.RevokeSummaryLinkParams{ID: id, UserID: userID, Now: stamp(now)})
	if err != nil {
		return Summary{}, fmt.Errorf("sharelinks: revoke summary: %w", err)
	}
	if n == 0 {
		return Summary{}, ErrNotFound
	}
	r, err := s.q.GetSummaryLinkMeta(ctx, store.GetSummaryLinkMetaParams{ID: id, UserID: userID})
	if err != nil {
		return Summary{}, fmt.Errorf("sharelinks: get summary: %w", err)
	}
	return summaryOf(store.ListSummaryLinksRow(r)), nil
}

// OpenAs is Open with the viewer's access log entry.
func (s *Service) OpenAs(ctx context.Context, token string, now time.Time, v Viewer) (Opened, error) {
	if !ValidToken(token) {
		return Opened{}, ErrNotFound
	}
	r, err := s.q.GetShareLinkForOpen(ctx, HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		return Opened{}, ErrNotFound
	}
	if err != nil {
		return Opened{}, fmt.Errorf("sharelinks: lookup: %w", err)
	}
	summary := r.Kind == KindSummary
	switch {
	case r.RevokedAt.Valid:
		if summary {
			return Opened{}, ErrNotFound
		}
		return Opened{}, ErrRevoked
	case !r.ExpiresAt.Valid || !now.Before(r.ExpiresAt.Time) || !r.Payload.Valid:
		if summary {
			return Opened{}, ErrNotFound
		}
		return Opened{}, ErrExpired
	}
	v.Via = ViaLink
	return s.openPayload(ctx, r, token, now, v)
}

// OpenCode opens the summary behind a typed code: ErrNotFound for anything but a live summary's code (malformed,
// unknown, expired, revoked — all the same), ErrCodesDisabled without a pepper in production.
func (s *Service) OpenCode(ctx context.Context, typed string, now time.Time, v Viewer) (Opened, error) {
	if s.coder == nil {
		return Opened{}, ErrCodesDisabled
	}
	code, ok := NormalizeCode(typed)
	if !ok {
		return Opened{}, ErrNotFound
	}
	c, err := s.q.GetShareLinkByCode(ctx, sql.NullString{String: s.coder.Hash(code), Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return Opened{}, ErrNotFound
	}
	if err != nil {
		return Opened{}, fmt.Errorf("sharelinks: code lookup: %w", err)
	}
	if c.RevokedAt.Valid || !c.ExpiresAt.Valid || !now.Before(c.ExpiresAt.Time) || !c.CodePayload.Valid {
		return Opened{}, ErrNotFound
	}
	token, err := s.coder.Unwrap(code, c.CodePayload.String)
	if err != nil || !ValidToken(token) {
		return Opened{}, ErrNotFound
	}
	r, err := s.q.GetShareLinkForOpen(ctx, HashToken(token))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (r.ID != c.ID || !r.Payload.Valid)) {
		return Opened{}, ErrNotFound
	}
	if err != nil {
		return Opened{}, fmt.Errorf("sharelinks: lookup: %w", err)
	}
	v.Via = ViaCode
	return s.openPayload(ctx, r, token, now, v)
}

// openPayload decrypts a live link's snapshot, signs the summary's document files, counts the view and writes the
// access log row.
func (s *Service) openPayload(ctx context.Context, r store.GetShareLinkForOpenRow, token string, now time.Time,
	v Viewer,
) (Opened, error) {
	plain, err := Open(token, r.Payload.String)
	if err != nil {
		return Opened{}, ErrNotFound
	}
	val, err := phpval.Decode(plain)
	m, isMap := val.(phpval.Map)
	if err != nil || !isMap {
		return Opened{}, ErrNotFound
	}
	if r.Kind == KindSummary {
		raw, _ := m.Get("documents")
		docs, err := s.publicDocuments(ctx, r.UserID, raw, now)
		if err != nil {
			return Opened{}, err
		}
		m.Set("documents", docs)
	}
	if err := s.q.CountShareLinkView(ctx, store.CountShareLinkViewParams{ID: r.ID, Now: stamp(now)}); err != nil {
		return Opened{}, fmt.Errorf("sharelinks: count view: %w", err)
	}
	if err := s.q.InsertShareLinkView(ctx, store.InsertShareLinkViewParams{
		ShareLinkID: r.ID, UserID: r.UserID, Via: v.Via, Device: v.Device, Browser: v.Browser, ViewedAt: stamp(now),
	}); err != nil {
		return Opened{}, fmt.Errorf("sharelinks: access log: %w", err)
	}
	return Opened{Report: m, ExpiresAt: r.ExpiresAt.Time}, nil
}

// AccessEntry is one row of the owner's access log «سابقه دسترسی».
type AccessEntry struct {
	ID, LinkID           uint64
	Kind, Label          string
	Via, Device, Browser string
	ViewedAt             time.Time
	ExpiresAt, RevokedAt time.Time
}

// JSON is {id, link_id, kind, label, via, client{device, browser}, viewed_at, link_status}.
func (a AccessEntry) JSON(now time.Time) *jsonx.OrderedMap {
	l := Link{ExpiresAt: a.ExpiresAt, RevokedAt: a.RevokedAt}
	return jsonx.Obj("id", a.ID, "link_id", a.LinkID, "kind", a.Kind, "label", orNil(a.Label), "via", a.Via,
		"client", jsonx.Obj("device", a.Device, "browser", a.Browser), "viewed_at", jsonx.ISO8601(a.ViewedAt),
		"link_status", l.Status(now))
}

// Access is the owner's access log, newest first: every link (linkID 0) or one of hers (ErrNotFound for a foreign or
// unknown link id). limit is clamped to [1, MaxAccessLimit].
func (s *Service) Access(ctx context.Context, userID, linkID uint64, limit int) ([]AccessEntry, error) {
	limit = min(max(limit, 1), MaxAccessLimit)
	var rows []store.ListShareAccessRow
	if linkID == 0 {
		var err error
		if rows, err = s.q.ListShareAccess(ctx, store.ListShareAccessParams{UserID: userID, Limit: int32(limit)}); err != nil { //nolint:gosec // clamped
			return nil, fmt.Errorf("sharelinks: access log: %w", err)
		}
	} else {
		n, err := s.q.ShareLinkOwned(ctx, store.ShareLinkOwnedParams{ID: linkID, UserID: userID})
		if err != nil {
			return nil, fmt.Errorf("sharelinks: link owner: %w", err)
		}
		if n == 0 {
			return nil, ErrNotFound
		}
		one, err := s.q.ListShareLinkAccess(ctx, store.ListShareLinkAccessParams{UserID: userID, ShareLinkID: linkID,
			Limit: int32(limit)}) //nolint:gosec // clamped
		if err != nil {
			return nil, fmt.Errorf("sharelinks: access log: %w", err)
		}
		for _, r := range one {
			rows = append(rows, store.ListShareAccessRow(r))
		}
	}
	out := make([]AccessEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, AccessEntry{ID: r.ID, LinkID: r.ShareLinkID, Kind: r.Kind, Label: str(r.Label), Via: r.Via,
			Device: r.Device, Browser: r.Browser, ViewedAt: nullTime(r.ViewedAt), ExpiresAt: nullTime(r.ExpiresAt),
			RevokedAt: nullTime(r.RevokedAt)})
	}
	return out, nil
}
