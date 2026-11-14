// Package sharerequests is «درخواست دسترسی بیشتر» (engine v6 M11, T-M11-08; SPEC §1.7 share_request; D-196): either
// side of a family link asks the other for more access to the other's data. Nothing changes before the data owner
// approves; the approval applies through family.Grants.Apply (registry caps, epoch, activity diff, consent event).
//
// Rules: the two accounts must already be linked (an active link in either direction) — otherwise a uniform 404, so
// the endpoint is no account oracle; one request per domain per pair per 7 days (429); only the request's recipient
// (the data owner) approves or declines, and a foreign id is a 404 (IDOR).
package sharerequests

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Window is the per-domain throttle window and the life of a pending request.
const Window = 7 * 24 * time.Hour

// Statuses (family_share_requests.status).
const (
	StatusPending  = "pending"
	StatusAccepted = "accepted"
	StatusDeclined = "declined"
	StatusExpired  = "expired"
)

// listLimit caps each box.
const listLimit = 50

// linkType is companions.type of a link created by an approval (role carries the relation).
const linkType = "partner"

// Errors.
var (
	ErrNotFound = errors.New("share requests: not found")
	ErrSelf     = errors.New("share requests: cannot ask yourself")
	ErrEmpty    = errors.New("share requests: no domain requested")
	// ErrRateLimited is a domain already requested from the same person within Window.
	ErrRateLimited = errors.New("share requests: already requested recently")
	// ErrLinkPending: the owner has an open invite to the requester; she answers that invite instead.
	ErrLinkPending = errors.New("share requests: an invite between the two is still open")
	// ErrTeenParentOnly: approving would create a link owned by a teen-mode account (CB-TEEN-01).
	ErrTeenParentOnly = errors.New("share requests: a teen may only share with a parent")
)

// Conn is the database the service writes through (a *sql.DB).
type Conn interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Service is the share-request domain.
type Service struct {
	conn  Conn
	clock clock.Clock
}

// NewService returns a Service on conn.
func NewService(conn Conn, clk clock.Clock) *Service {
	if clk == nil {
		clk = clock.Real{}
	}
	return &Service{conn: conn, clock: clk}
}

func (s *Service) now() time.Time { return s.clock.Now().In(civildate.Tehran).Truncate(time.Second) }

func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: !t.IsZero()} }

func nid(id uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(id), Valid: id != 0} } //nolint:gosec // ids fit

func (s *Service) inTx(ctx context.Context, fn func(tx *sql.Tx, q *store.Queries) error) error {
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("share requests: begin: %w", err)
	}
	if err := fn(tx, store.New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("share requests: commit: %w", err)
	}
	return nil
}

func lockUser(ctx context.Context, q *store.Queries, id uint64) error {
	if _, err := q.LockOwner(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("share requests: lock: %w", err)
	}
	return nil
}

// pair is what links the requester to the data owner: the target link owner → requester (may not exist yet) and the
// reverse link requester → owner. At least one is active.
type pair struct {
	target, reverse *store.Companion
}

func (p pair) role() family.Role {
	if p.target != nil {
		return family.ParseRole(p.target.Role.String, p.target.Type)
	}
	if p.reverse != nil {
		return family.ParseRole(p.reverse.Role.String, p.reverse.Type)
	}
	return family.RoleNone
}

func (p pair) linkID() uint64 {
	if p.target != nil {
		return p.target.ID
	}
	if p.reverse != nil {
		return p.reverse.ID
	}
	return 0
}

func loadPair(ctx context.Context, q *store.Queries, ownerID, requesterID uint64) (pair, error) {
	var p pair
	get := func(o, m uint64) (*store.Companion, error) {
		c, err := q.GetOpenLinkForPair(ctx, store.GetOpenLinkForPairParams{OwnerID: o, MemberID: nid(m)})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("share requests: link: %w", err)
		}
		return &c, nil
	}
	var err error
	if p.target, err = get(ownerID, requesterID); err != nil {
		return p, err
	}
	if p.reverse, err = get(requesterID, ownerID); err != nil {
		return p, err
	}
	if p.target == nil && p.reverse == nil {
		return p, ErrNotFound
	}
	if p.role() == family.RoleNone || p.role() == family.RoleTeenParent {
		return p, ErrNotFound // the teen card is outside the v6 matrix (F-17)
	}
	return p, nil
}

func ownerNotTeen(ctx context.Context, q *store.Queries, ownerID uint64) error {
	mode, err := q.GetUserLifeMode(ctx, ownerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("share requests: life mode: %w", err)
	}
	if mode.Valid && mode.String == string(enums.LifeModeTeen) {
		return ErrTeenParentOnly
	}
	return nil
}

func ownerMale(ctx context.Context, q *store.Queries, ownerID uint64) (bool, error) {
	in, err := q.FamilyOwnerModeInputs(ctx, ownerID)
	if err != nil {
		return false, fmt.Errorf("share requests: owner: %w", err)
	}
	return in.Gender.Valid && in.Gender.String == "male", nil
}

// Request is one share request as either side sees it.
type Request struct {
	ID         uint64
	FromUserID uint64
	ToUserID   uint64
	Domains    map[family.Domain]family.Level
	Status     string
	ExpiresAt  time.Time
	DecidedAt  time.Time
	CreatedAt  time.Time
}

func decodeDomains(raw json.RawMessage) map[family.Domain]family.Level {
	var m map[string]string
	_ = json.Unmarshal(raw, &m)
	out := map[family.Domain]family.Level{}
	for k, v := range m {
		if d, ok := family.ParseDomain(k); ok {
			if l, ok := family.ParseLevel(v); ok && l > family.LevelOff {
				out[d] = l
			}
		}
	}
	return out
}

// Create records fromID asking toID for more access to toID's data (domain → level). It fails with ErrNotFound when
// the two are not linked, a *family.LevelError above what the registry allows for the relation, and ErrRateLimited
// when a domain was asked of the same person within Window. Nothing changes until toID approves.
func (s *Service) Create(ctx context.Context, fromID, toID uint64, domains map[family.Domain]family.Level) (Request, error) {
	if fromID == toID {
		return Request{}, ErrSelf
	}
	set := map[family.Domain]family.Level{}
	for d, l := range domains {
		if l > family.LevelOff {
			set[d] = l
		}
	}
	if len(set) == 0 {
		return Request{}, ErrEmpty
	}
	now := s.now()
	var out Request
	err := s.inTx(ctx, func(_ *sql.Tx, q *store.Queries) error {
		if err := lockUser(ctx, q, toID); err != nil { // the data owner's mutex serialises the pair's throttle
			return err
		}
		p, err := loadPair(ctx, q, toID, fromID)
		if err != nil {
			return err
		}
		male, err := ownerMale(ctx, q, toID)
		if err != nil {
			return err
		}
		if err := family.Validate(set, p.role(), male); err != nil {
			return err
		}
		prior, err := q.ListFamilyShareRequestDomainsForPairSince(ctx, store.ListFamilyShareRequestDomainsForPairSinceParams{
			FromUserID: fromID, ToUserID: toID, Since: nt(now.Add(-Window))})
		if err != nil {
			return fmt.Errorf("share requests: throttle: %w", err)
		}
		for _, raw := range prior {
			for d := range decodeDomains(raw) {
				if _, again := set[d]; again {
					return ErrRateLimited
				}
			}
		}
		codes := map[string]string{}
		for d, l := range set {
			codes[d.String()] = l.String()
		}
		raw, err := json.Marshal(codes)
		if err != nil {
			return fmt.Errorf("share requests: domains: %w", err)
		}
		id, err := q.InsertFamilyShareRequest(ctx, store.InsertFamilyShareRequestParams{
			FromUserID: fromID, ToUserID: toID, LinkID: nid(p.linkID()), Domains: raw, ExpiresAt: now.Add(Window), Now: nt(now),
		})
		if err != nil {
			return fmt.Errorf("share requests: insert: %w", err)
		}
		out = Request{ID: uint64(id), FromUserID: fromID, ToUserID: toID, Domains: set, Status: StatusPending, //nolint:gosec // auto-increment id
			ExpiresAt: now.Add(Window), CreatedAt: now}
		return nil
	})
	return out, err
}

// Box is which side of the requests to list.
type Box string

// Boxes.
const (
	BoxIn  Box = "in"  // asked of me (I approve)
	BoxOut Box = "out" // asked by me
)

// List is the caller's live incoming requests (in) or own requests of any status, newest first (out).
func (s *Service) List(ctx context.Context, userID uint64, box Box) ([]Request, error) {
	q, now := store.New(s.conn), s.now()
	if box == BoxIn {
		rows, err := q.ListPendingFamilyShareRequestsForOwner(ctx, store.ListPendingFamilyShareRequestsForOwnerParams{
			ToUserID: userID, Now: now, Limit: listLimit})
		if err != nil {
			return nil, fmt.Errorf("share requests: list: %w", err)
		}
		out := make([]Request, 0, len(rows))
		for _, r := range rows {
			out = append(out, Request{ID: r.ID, FromUserID: r.FromUserID, ToUserID: r.ToUserID, Domains: decodeDomains(r.Domains),
				Status: r.Status, ExpiresAt: r.ExpiresAt, DecidedAt: r.DecidedAt.Time, CreatedAt: r.CreatedAt.Time})
		}
		return out, nil
	}
	rows, err := q.ListFamilyShareRequestsFrom(ctx, store.ListFamilyShareRequestsFromParams{FromUserID: userID, Now: now, Limit: listLimit})
	if err != nil {
		return nil, fmt.Errorf("share requests: list: %w", err)
	}
	out := make([]Request, 0, len(rows))
	for _, r := range rows {
		status := r.Status
		if r.Expired == 1 {
			status = StatusExpired
		}
		out = append(out, Request{ID: r.ID, FromUserID: r.FromUserID, ToUserID: r.ToUserID, Domains: decodeDomains(r.Domains),
			Status: status, ExpiresAt: r.ExpiresAt, DecidedAt: r.DecidedAt.Time, CreatedAt: r.CreatedAt.Time})
	}
	return out, nil
}

// Approve applies a request the caller received (only the data owner approves): the requested levels are merged into
// the link owner → requester (never lowering a level), or such a link is created active with the pair's relation,
// through family.Grants.Apply (activity diff and consent event). A request that is not hers, decided or expired is
// ErrNotFound and nothing changes.
func (s *Service) Approve(ctx context.Context, ownerID, id uint64) (uint64, error) {
	now := s.now()
	var linkID uint64
	err := s.inTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		if err := lockUser(ctx, q, ownerID); err != nil {
			return err
		}
		r, err := q.GetFamilyShareRequest(ctx, store.GetFamilyShareRequestParams{ID: id, ToUserID: ownerID, Now: now})
		if errors.Is(err, sql.ErrNoRows) || (err == nil && (r.ToUserID != ownerID || r.Status != StatusPending || r.Expired == 1)) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("share requests: get: %w", err)
		}
		p, err := loadPair(ctx, q, ownerID, r.FromUserID)
		if err != nil {
			return err // the link ended meanwhile: nothing to apply
		}
		role := p.role()
		set := map[family.Domain]family.Level{}
		var expires time.Time
		if p.target != nil {
			linkID, expires = p.target.ID, p.target.ExpiresAt.Time
			if set, err = currentSet(ctx, q, linkID); err != nil {
				return err
			}
		} else {
			open, err := q.CountOpenLinks(ctx, store.CountOpenLinksParams{OwnerID: ownerID, CompanionUserID: nid(r.FromUserID)})
			if err != nil {
				return fmt.Errorf("share requests: links: %w", err)
			}
			if open > 0 {
				return ErrLinkPending
			}
			if err := ownerNotTeen(ctx, q, ownerID); err != nil {
				return err // CB-TEEN-01: a teen owns only her parent link
			}
			created, err := q.CreateFamilyLink(ctx, store.CreateFamilyLinkParams{
				OwnerID: ownerID, CompanionUserID: nid(r.FromUserID), Type: linkType, Status: "active",
				Role: sql.NullString{String: role.String(), Valid: true}, Template: sql.NullString{String: string(family.TemplateCustom), Valid: true},
				Now: nt(now), AcceptedAt: nt(now),
			})
			if err != nil {
				return fmt.Errorf("share requests: link: %w", err)
			}
			linkID = uint64(created) //nolint:gosec // auto-increment id
		}
		for d, l := range decodeDomains(r.Domains) {
			set[d] = max(set[d], l)
		}
		if _, err := (family.Grants{}).Apply(ctx, tx, family.Change{
			LinkID: linkID, OwnerID: ownerID, MemberID: r.FromUserID, ActorID: ownerID, Role: role, Set: set,
			ExpiresAt: expires, Reason: family.ChangeShareRequest, Now: now,
		}); err != nil {
			return err
		}
		n, err := q.DecideFamilyShareRequest(ctx, store.DecideFamilyShareRequestParams{
			Status: StatusAccepted, DecidedAt: nt(now), ID: id, ToUserID: ownerID, Now: now})
		if err != nil {
			return fmt.Errorf("share requests: decide: %w", err)
		}
		if n == 0 {
			return ErrNotFound
		}
		return nil
	})
	return linkID, err
}

func currentSet(ctx context.Context, q *store.Queries, linkID uint64) (map[family.Domain]family.Level, error) {
	rows, err := q.ListGrants(ctx, linkID)
	if err != nil {
		return nil, fmt.Errorf("share requests: grants: %w", err)
	}
	set := map[family.Domain]family.Level{}
	for _, g := range rows {
		if d, ok := family.ParseDomain(g.Section); ok {
			if l, ok := family.ParseLevel(g.Level); ok && l > family.LevelOff {
				set[d] = l
			}
		}
	}
	return set, nil
}

// Decline refuses a request the caller received; nothing else changes. Not hers, decided or expired → ErrNotFound.
func (s *Service) Decline(ctx context.Context, ownerID, id uint64) error {
	now := s.now()
	n, err := store.New(s.conn).DecideFamilyShareRequest(ctx, store.DecideFamilyShareRequestParams{
		Status: StatusDeclined, DecidedAt: nt(now), ID: id, ToUserID: ownerID, Now: now})
	if err != nil {
		return fmt.Errorf("share requests: decline: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
