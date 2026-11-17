package companion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/family/invites"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Defaults of Options.
const (
	DefaultInviteTTL         = invites.DefaultCodeTTL
	DefaultMaxAttempts       = invites.DefaultMaxAttempts
	DefaultMaxOpenCompanions = invites.DefaultMaxOpen
	maxDisplayName           = 100
)

// Options tunes the service; zero values take the defaults.
type Options struct {
	// CodePepper keys the HMAC of invite codes at rest (secret, from env; empty → invites.DevPepper, local / testing /
	// contract only — the server refuses to start without one elsewhere, FAM-L1).
	CodePepper []byte
	// InviteTTL is how long a code-only invite stays valid (24 h); a phone invite lives 7 days (F-11).
	InviteTTL time.Duration
	// MaxAttempts locks an invite after this many failed attempts (5).
	MaxAttempts int
	// MaxOpenCompanions caps an owner's invited + active links (10).
	MaxOpenCompanions int
	// Rand is the code entropy source (crypto/rand).
	Rand io.Reader
	// ReadAudit coalesces member read-audit rows in Redis (F-16, D-203); nil → the DB window check.
	ReadAudit *family.ReadAudit
	// MinAcceptDuration is the floor of Accept (CMP-L2, invites.Options.MinDuration); 0 = none (tests).
	MinAcceptDuration time.Duration
	// Mismatch counts a phone-bound code's wrong-account tries once per account (CMP-L8); nil = every try.
	Mismatch invites.Gate
}

// Conn is the database the service writes through (a *sql.DB).
type Conn interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Service is the companion domain.
type Service struct {
	conn      Conn
	clock     clock.Clock
	maxOpen   int
	readAudit *family.ReadAudit
	// inv is the one invite service (T-M11-08, D-196): codes are issued and redeemed there.
	inv *invites.Service
}

// NewService returns a Service on conn.
func NewService(conn Conn, clk clock.Clock, opt Options) *Service {
	s := &Service{conn: conn, clock: clk, maxOpen: opt.MaxOpenCompanions, readAudit: opt.ReadAudit}
	if s.maxOpen <= 0 {
		s.maxOpen = DefaultMaxOpenCompanions
	}
	var invConn invites.Conn
	if conn != nil {
		invConn = conn
	}
	s.inv = invites.NewService(invConn, clk, invites.Options{Pepper: opt.CodePepper, CodeTTL: opt.InviteTTL,
		MaxAttempts: opt.MaxAttempts, MaxOpen: s.maxOpen, Rand: opt.Rand, MinDuration: opt.MinAcceptDuration,
		Mismatch: opt.Mismatch})
	return s
}

// Invites is the invite service the companion routes run on (D-196).
func (s *Service) Invites() *invites.Service { return s.inv }

// Link is one owner ↔ companion relation as either party sees it.
type Link struct {
	ID              uint64
	OwnerID         uint64
	CompanionUserID uint64 // 0 until accepted
	Type            Type
	Status          Status
	DisplayName     string
	InvitedAt       time.Time
	AcceptedAt      time.Time // zero until accepted
	Grants          Grants
	// InviteExpiresAt is the expiry of the open invite of a pending link (zero when none is open).
	InviteExpiresAt time.Time
	// InvitePhone is the masked-by-caller mobile the open invite is bound to ("" for code-only invites).
	InvitePhone string
	// FamilyID / SharedChildIDs are set for spouse links (children rows arrive in B-N5-02).
	FamilyID       uint64
	SharedChildIDs []uint64
	// Preferences are the owner's «ریتمی همراه» v2 settings of the link (goal, signal consents, action kinds).
	Preferences Preferences
}

// InviteInput is the Hamdam_Type → Access → Children → Invite flow.
type InviteInput struct {
	Type        Type
	DisplayName string // «اسمش», optional
	Phone       string // optional; binds the invite to this mobile
	Grants      Grants
	// ChildIDs are the shared children (spouse only). The caller verifies ownership (B-N5-02 children service).
	ChildIDs []uint64
	// Preferences are the optional «ریتمی همراه» v2 settings (zero value = DefaultPreferences).
	Preferences Preferences
}

// CreatedInvite is the result of CreateInvite / RenewInvite. Code is the only place the plain code exists.
type CreatedInvite struct {
	Link      Link
	Code      string
	Phone     string
	ExpiresAt time.Time
}

// AuditEntry is one row of the owner's audit trail.
type AuditEntry struct {
	ID          uint64
	ActorID     uint64 // 0 when the actor's account is gone
	CompanionID uint64 // 0 when the link is gone
	Section     Section
	Action      Action
	At          time.Time
}

func (s *Service) now() time.Time { return s.clock.Now().In(civildate.Tehran).Truncate(time.Second) }

func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func nid(id uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(id), Valid: id != 0} } //nolint:gosec // ids fit

func nstr(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }

// inTx runs fn on one READ COMMITTED transaction.
func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	return s.inTxTx(ctx, func(_ *sql.Tx, q *store.Queries) error { return fn(q) })
}

// inTxTx is inTx with the transaction itself (family.Grants.Apply writes through it).
func (s *Service) inTxTx(ctx context.Context, fn func(tx *sql.Tx, q *store.Queries) error) error {
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("companion: begin: %w", err)
	}
	if err := fn(tx, store.New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("companion: commit: %w", err)
	}
	return nil
}

func lockOwner(ctx context.Context, q *store.Queries, ownerID uint64) error {
	if _, err := q.LockOwner(ctx, ownerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("companion: lock owner: %w", err)
	}
	return nil
}

func validateInput(in *InviteInput) error {
	if !in.Type.Valid() {
		return ErrInvalidType
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if utf8.RuneCountInString(in.DisplayName) > maxDisplayName {
		return ErrInvalidName
	}
	if in.Type == TypeParent && strings.TrimSpace(in.Phone) == "" {
		return ErrPhoneRequired // a minor's card goes only to the account with that number
	}
	if in.Phone != "" {
		m, ok := NormalizeMobile(in.Phone)
		if !ok {
			return ErrInvalidPhone
		}
		in.Phone = m
	}
	if err := in.Grants.ValidateFor(in.Type); err != nil {
		return err
	}
	if len(in.ChildIDs) > 0 && in.Type != TypeSpouse {
		return ErrChildrenNoSpouse
	}
	return in.Preferences.Validate()
}

// CreateInvite makes a pending link with the chosen type, grants and (spouse) shared children, and a one-time code
// valid for InviteTTL. Nothing is visible to anyone until the code is accepted.
func (s *Service) CreateInvite(ctx context.Context, ownerID uint64, in InviteInput) (CreatedInvite, error) {
	if err := validateInput(&in); err != nil {
		return CreatedInvite{}, err
	}
	now := s.now()
	var out CreatedInvite
	err := s.inTxTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		if err := lockOwner(ctx, q, ownerID); err != nil {
			return err
		}
		if err := checkTeenRule(ctx, q, ownerID, in.Type); err != nil {
			return err
		}
		if in.Phone != "" {
			own, err := q.GetUserMobile(ctx, ownerID)
			if err != nil {
				return fmt.Errorf("companion: owner mobile: %w", err)
			}
			if own.Valid && own.String == in.Phone {
				return ErrSelfInvite
			}
		}
		open, err := q.CountOpenCompanions(ctx, store.CountOpenCompanionsParams{OwnerID: ownerID})
		if err != nil {
			return fmt.Errorf("companion: count: %w", err)
		}
		if open >= int64(s.maxOpen) {
			return ErrTooManyCompanions
		}
		if in.Type == TypeSpouse {
			spouses, err := q.CountOpenCompanions(ctx, store.CountOpenCompanionsParams{OwnerID: ownerID, Type: string(TypeSpouse)})
			if err != nil {
				return fmt.Errorf("companion: count spouses: %w", err)
			}
			if spouses > 0 {
				return ErrSpouseExists
			}
		}
		id, err := q.CreateCompanion(ctx, store.CreateCompanionParams{
			OwnerID: ownerID, Type: string(in.Type), DisplayName: nstr(in.DisplayName), Now: nt(now),
			Goal: goalColumn(in.Preferences.Goals), SharePain: in.Preferences.Signals.On(SignalPain),
			ShareMood: in.Preferences.Signals.On(SignalMood), ShareEnergy: in.Preferences.Signals.On(SignalEnergy),
			ShareSleep: in.Preferences.Signals.On(SignalSleep), SharePeriodDays: in.Preferences.PeriodDays,
			ActionPrefs: actionsColumn(in.Preferences.Actions),
		})
		if err != nil {
			return fmt.Errorf("companion: create: %w", err)
		}
		cid := uint64(id) //nolint:gosec // AUTO_INCREMENT ids are positive
		if err := applyGrants(ctx, tx, store.Companion{ID: cid, OwnerID: ownerID, Type: string(in.Type)}, ownerID,
			toV6(in.Grants), family.ChangeInvite, now); err != nil {
			return err
		}
		if in.Type == TypeSpouse {
			fid, err := q.CreateFamily(ctx, store.CreateFamilyParams{OwnerID: ownerID, CompanionID: cid, Now: nt(now)})
			if err != nil {
				return fmt.Errorf("companion: family: %w", err)
			}
			for _, child := range in.ChildIDs {
				if err := q.AddFamilyChild(ctx, store.AddFamilyChildParams{FamilyID: uint64(fid), ChildID: child, Now: nt(now)}); err != nil { //nolint:gosec // positive id
					return fmt.Errorf("companion: family child: %w", err)
				}
			}
		}
		code, expires, err := s.issueInvite(ctx, q, ownerID, cid, in.Phone, now)
		if err != nil {
			return err
		}
		if err := audit(ctx, q, ownerID, ownerID, cid, "", ActionInvited, now); err != nil {
			return err
		}
		link, err := loadLink(ctx, q, cid)
		if err != nil {
			return err
		}
		link.InviteExpiresAt, link.InvitePhone = expires, in.Phone
		out = CreatedInvite{Link: link, Code: code, Phone: in.Phone, ExpiresAt: expires}
		return nil
	})
	return out, err
}

// RenewInvite revokes the open invites of a pending link and issues a fresh code (the old code stops working). Phone
// binding: phone "" keeps the previous binding of the latest invite; pass a number to change it.
func (s *Service) RenewInvite(ctx context.Context, ownerID, companionID uint64, phone string) (CreatedInvite, error) {
	if phone != "" {
		m, ok := NormalizeMobile(phone)
		if !ok {
			return CreatedInvite{}, ErrInvalidPhone
		}
		phone = m
	}
	now := s.now()
	var out CreatedInvite
	err := s.inTx(ctx, func(q *store.Queries) error {
		if err := lockOwner(ctx, q, ownerID); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, companionID)
		if err != nil || c.OwnerID != ownerID || Status(c.Status) == StatusRevoked {
			return notFound(err)
		}
		if Status(c.Status) != StatusInvited {
			return ErrNotPending
		}
		if err := checkTeenRule(ctx, q, ownerID, Type(c.Type)); err != nil {
			return err // the owner's mode changed since the invite (CB-TEEN-01)
		}
		if phone == "" {
			phone, err = latestPhone(ctx, q, ownerID, companionID)
			if err != nil {
				return err
			}
		} else if own, err := q.GetUserMobile(ctx, ownerID); err == nil && own.Valid && own.String == phone {
			return ErrSelfInvite
		}
		if err := q.RevokeOpenInvites(ctx, store.RevokeOpenInvitesParams{CompanionID: companionID, Now: nt(now)}); err != nil {
			return fmt.Errorf("companion: revoke invites: %w", err)
		}
		code, expires, err := s.issueInvite(ctx, q, ownerID, companionID, phone, now)
		if err != nil {
			return err
		}
		link, err := loadLink(ctx, q, companionID)
		if err != nil {
			return err
		}
		link.InviteExpiresAt, link.InvitePhone = expires, phone
		out = CreatedInvite{Link: link, Code: code, Phone: phone, ExpiresAt: expires}
		return nil
	})
	return out, err
}

// latestPhone is the phone binding of the newest invite of a link (open or not).
func latestPhone(ctx context.Context, q *store.Queries, ownerID, companionID uint64) (string, error) {
	invites, err := q.ListLinkInvites(ctx, store.ListLinkInvitesParams{OwnerID: ownerID, CompanionID: companionID})
	if err != nil {
		return "", fmt.Errorf("companion: invites: %w", err)
	}
	if len(invites) == 0 {
		return "", nil
	}
	return invites[0].Phone.String, nil
}

func (s *Service) issueInvite(ctx context.Context, q *store.Queries, ownerID, companionID uint64, phone string, now time.Time) (string, time.Time, error) {
	return s.inv.Issue(ctx, q, ownerID, companionID, phone, now)
}

// Accept redeems a code for viewerID on the invite service (D-196): the pending link becomes active with the grants
// the owner chose (or awaiting_owner for a high-impact code-only v6 invite, F-11: the member then sees nothing), and a
// spouse joins the owner's family. A bound invite redeemed by another number counts a failed attempt; MaxAttempts
// locks it. Callers rate-limit Accept per user (B-N4-02).
func (s *Service) Accept(ctx context.Context, viewerID uint64, typed string) (Link, error) {
	a, err := s.inv.Accept(ctx, viewerID, invites.AcceptInput{Code: typed})
	if err != nil {
		if errors.Is(err, invites.ErrNotFound) {
			return Link{}, ErrNotFound
		}
		return Link{}, err
	}
	link, err := loadLink(ctx, store.New(s.conn), a.LinkID)
	if err != nil {
		return Link{}, err
	}
	if link.Status == StatusAwaitingOwner {
		link.Grants = Grants{} // nothing is shared before the owner confirms
		link.SharedChildIDs = nil
	}
	return link, nil
}

// Revoke ends a link: the owner removes a companion or the companion leaves. Grants, open invites and the spouse
// family (with its shared-children links) are deleted; the link row stays as revoked for the audit trail.
func (s *Service) Revoke(ctx context.Context, actorID, companionID uint64) error {
	c, err := store.New(s.conn).GetCompanion(ctx, companionID)
	if err != nil {
		return notFound(err)
	}
	now := s.now()
	return s.inTx(ctx, func(q *store.Queries) error {
		if err := lockOwner(ctx, q, c.OwnerID); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, companionID)
		if err != nil {
			return notFound(err)
		}
		by := RevokedByOwner
		switch {
		case c.OwnerID == actorID:
		case c.CompanionUserID.Valid && uint64(c.CompanionUserID.Int64) == actorID && Status(c.Status) == StatusActive: //nolint:gosec // positive id
			by = RevokedByCompanion
		default:
			return ErrNotFound
		}
		n, err := q.RevokeCompanion(ctx, store.RevokeCompanionParams{ID: c.ID, RevokedBy: nstr(by), Now: nt(now)})
		if err != nil {
			return fmt.Errorf("companion: revoke: %w", err)
		}
		if n == 0 {
			return ErrNotFound // already revoked
		}
		if err := q.RevokeOpenInvites(ctx, store.RevokeOpenInvitesParams{CompanionID: c.ID, Now: nt(now)}); err != nil {
			return fmt.Errorf("companion: revoke invites: %w", err)
		}
		if err := q.DeleteGrants(ctx, c.ID); err != nil {
			return fmt.Errorf("companion: delete grants: %w", err)
		}
		if err := q.DeleteFamilyByCompanion(ctx, c.ID); err != nil {
			return fmt.Errorf("companion: delete family: %w", err)
		}
		// CB-LOSS-01: the one-line pregnancy notice the companion got does not outlive the link.
		if err := q.DeletePregnancyNoticesForLink(ctx, int64(c.ID)); err != nil { //nolint:gosec // auto-increment id
			return fmt.Errorf("companion: delete notices: %w", err)
		}
		return audit(ctx, q, c.OwnerID, actorID, c.ID, "", ActionRevoked, now)
	})
}

// SetGrants replaces the grants of an owner's invited or active link: sections missing from grants (or set to none)
// lose access. The grants must fit the link type (Grants.ValidateFor: a parent link holds only view grants on the
// teen sections).
func (s *Service) SetGrants(ctx context.Context, ownerID, companionID uint64, grants Grants) (Link, error) {
	if err := grants.Validate(); err != nil {
		return Link{}, err
	}
	now := s.now()
	var link Link
	err := s.inTxTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		if err := lockOwner(ctx, q, ownerID); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, companionID)
		if err != nil || c.OwnerID != ownerID || Status(c.Status) == StatusRevoked {
			return notFound(err)
		}
		if err := grants.ValidateFor(Type(c.Type)); err != nil {
			return err
		}
		// family.Grants.Apply writes the grants_changed activity row (with the v6 diff) and the consent event.
		if err := applyGrants(ctx, tx, c, ownerID, toV6(grants), family.ChangeOwnerEdit, now); err != nil {
			return err
		}
		link, err = loadLink(ctx, q, c.ID)
		return err
	})
	return link, err
}

// SetSharedChildren replaces the shared children of an owner's spouse link. The caller verifies that every child
// belongs to the owner (children arrive in B-N5-02, which also adds the FK).
func (s *Service) SetSharedChildren(ctx context.Context, ownerID, companionID uint64, childIDs []uint64) (Link, error) {
	now := s.now()
	var link Link
	err := s.inTx(ctx, func(q *store.Queries) error {
		if err := lockOwner(ctx, q, ownerID); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, companionID)
		if err != nil || c.OwnerID != ownerID || Status(c.Status) == StatusRevoked {
			return notFound(err)
		}
		if Type(c.Type) != TypeSpouse {
			return ErrChildrenNoSpouse
		}
		f, err := q.GetFamilyByCompanion(ctx, c.ID)
		if err != nil {
			return notFound(err)
		}
		if err := q.DeleteFamilyChildren(ctx, f.ID); err != nil {
			return fmt.Errorf("companion: delete children: %w", err)
		}
		for _, child := range childIDs {
			if err := q.AddFamilyChild(ctx, store.AddFamilyChildParams{FamilyID: f.ID, ChildID: child, Now: nt(now)}); err != nil {
				return fmt.Errorf("companion: family child: %w", err)
			}
		}
		link, err = loadLink(ctx, q, c.ID)
		return err
	})
	return link, err
}

// ListForOwner returns the owner's invited and active links (Hamdam_List), oldest first, with grants, the open
// invite's expiry and the spouse family.
func (s *Service) ListForOwner(ctx context.Context, ownerID uint64) ([]Link, error) {
	q := store.New(s.conn)
	rows, err := q.ListOwnerCompanions(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("companion: list: %w", err)
	}
	grants, err := q.ListOwnerGrants(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("companion: grants: %w", err)
	}
	invites, err := q.ListOpenInvitesForOwner(ctx, store.ListOpenInvitesForOwnerParams{OwnerID: ownerID, Now: nt(s.now())})
	if err != nil {
		return nil, fmt.Errorf("companion: invites: %w", err)
	}
	links := make([]Link, 0, len(rows))
	for _, c := range rows {
		l := toLink(c, grants)
		for _, inv := range invites { // newest first: the first match is the live one
			if inv.CompanionID == c.ID {
				l.InviteExpiresAt, l.InvitePhone = inv.ExpiresAt.Time, inv.Phone.String
				break
			}
		}
		if err := loadFamily(ctx, q, &l); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

// ListForCompanion returns the active links in which viewerID is the companion (whose data she/he may see; the
// RecordFor picker filters them by CanWrite), oldest first.
func (s *Service) ListForCompanion(ctx context.Context, viewerID uint64) ([]Link, error) {
	q := store.New(s.conn)
	rows, err := q.ListCompanionLinks(ctx, nid(viewerID))
	if err != nil {
		return nil, fmt.Errorf("companion: list: %w", err)
	}
	grants, err := q.ListCompanionUserGrants(ctx, nid(viewerID))
	if err != nil {
		return nil, fmt.Errorf("companion: grants: %w", err)
	}
	// The member sees what Decide gives, not the stored rows: one snapshot statement for every owner (T-M11-05).
	snaps, err := family.Request(ctx, s.conn, s.now).ForMember(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	links := make([]Link, 0, len(rows))
	for _, c := range rows {
		l := toLink(c, grants)
		l.Grants = Grants{}
		gone := true // no snapshot row (blocked owner aside, the statement sees every active link) → leave it out
		for _, snap := range snaps {
			if snap.LinkID == c.ID {
				gone = snap.OwnerBlocked || (!snap.ExpiresAt.IsZero() && !now.Before(snap.ExpiresAt))
				l.Grants = effectiveGrants(snap, l.Type, now)
			}
		}
		if gone { // expired or blocked owner: like a revoked link, it is not listed (F-21)
			continue
		}
		if err := loadFamily(ctx, q, &l); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

// Level is viewerID's access to a section of ownerID's data: edit for the owner herself, the granted level through
// an active link, none otherwise (also for an unknown section).
func (s *Service) Level(ctx context.Context, ownerID, viewerID uint64, section Section) (Level, error) {
	l, _, err := s.access(ctx, ownerID, viewerID, section)
	return l, err
}

// access is Level plus the id of the active link that grants it (0 for the owner herself or none).
func (s *Service) access(ctx context.Context, ownerID, viewerID uint64, section Section) (Level, uint64, error) {
	if ownerID == 0 || viewerID == 0 || !section.Valid() {
		return LevelNone, 0, nil
	}
	if ownerID == viewerID {
		return LevelEdit, 0, nil
	}
	// family.Decide over the pair snapshot (one statement, memoised per request): role, expiry, blocked owner, male
	// owner, mode scope and epoch, teen-mode owner (B-N4-08b) — all decided in one place (T-M11-05).
	snap, err := family.Request(ctx, s.conn, s.now).Pair(ctx, ownerID, viewerID)
	if err != nil {
		return LevelNone, 0, err
	}
	l := effectiveLevel(snap, section, s.now())
	if l == LevelNone {
		return LevelNone, 0, nil
	}
	return l, snap.LinkID, nil
}

// CanRead reports whether viewerID may read the section of ownerID's data.
func (s *Service) CanRead(ctx context.Context, ownerID, viewerID uint64, section Section) (bool, error) {
	l, err := s.Level(ctx, ownerID, viewerID, section)
	return l.CanRead(), err
}

// CanWrite reports whether viewerID may write the section of ownerID's data («ثبت برای …»).
func (s *Service) CanWrite(ctx context.Context, ownerID, viewerID uint64, section Section) (bool, error) {
	l, err := s.Level(ctx, ownerID, viewerID, section)
	return l.CanWrite(), err
}

// Audit records that actorID read or wrote a section of ownerID's data (no payload). The owner acting on her own
// data is not recorded. companionID may be 0 when unknown to the caller.
func (s *Service) Audit(ctx context.Context, ownerID, actorID, companionID uint64, section Section, action Action) error {
	if ownerID == actorID {
		return nil
	}
	if !section.Valid() {
		return ErrInvalidSection
	}
	q, now := store.New(s.conn), s.now()
	if action != ActionRead {
		return audit(ctx, q, ownerID, actorID, companionID, section, action, now)
	}
	// One read row per (owner, member, section) per ReadAuditWindow: Redis SET NX first, the DB window on a Redis
	// error or without Redis (F-16, D-203).
	// A claimed window still checks the DB (cheap, indexed): reads through a path without Redis (the companion home)
	// share the same window.
	key := string(section)
	if d, ok := sectionDomain(section); ok {
		key = d.String() // v6 domain name (cycle → cycle_phase), the same key family.ReadAudit uses everywhere (L6)
	}
	claim := s.readAudit.Claim(ctx, ownerID, actorID, key, now)
	if claim == family.ClaimSkip {
		return nil
	}
	if seen, err := readAuditedRecently(ctx, q, ownerID, actorID, companionID, section, now); err != nil || seen {
		return err
	}
	if err := audit(ctx, q, ownerID, actorID, companionID, section, action, now); err != nil {
		if claim == family.ClaimRecord {
			s.readAudit.Release(ctx, ownerID, actorID, key)
		}
		return err
	}
	return nil
}

// AuditTrail returns the owner's latest audit entries, newest first.
func (s *Service) AuditTrail(ctx context.Context, ownerID uint64, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := store.New(s.conn).ListOwnerAudit(ctx, store.ListOwnerAuditParams{OwnerID: ownerID, Limit: int32(limit)}) //nolint:gosec // ≤ 500
	if err != nil {
		return nil, fmt.Errorf("companion: audit: %w", err)
	}
	out := make([]AuditEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, AuditEntry{
			ID: r.ID, ActorID: uint64(r.ActorID.Int64), CompanionID: uint64(r.CompanionID.Int64), //nolint:gosec // positive ids
			Section: Section(r.Section.String), Action: Action(r.Action), At: r.CreatedAt.Time,
		})
	}
	return out, nil
}

func audit(ctx context.Context, q *store.Queries, ownerID, actorID, companionID uint64, section Section, action Action, now time.Time) error {
	if err := q.InsertAudit(ctx, store.InsertAuditParams{
		OwnerID: ownerID, ActorID: nid(actorID), CompanionID: nid(companionID), Section: nstr(string(section)),
		Action: string(action), Now: nt(now),
	}); err != nil {
		return fmt.Errorf("companion: audit: %w", err)
	}
	return nil
}

// checkTeenRule enforces the teen ↔ parent rule (CB-TEEN-01) under the owner lock: only an owner whose stored life mode
// is teen may invite a parent, and such an owner (a minor) may invite nobody else.
func checkTeenRule(ctx context.Context, q *store.Queries, ownerID uint64, t Type) error {
	mode, err := q.GetUserLifeMode(ctx, ownerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("companion: life mode: %w", err)
	}
	teen := mode.Valid && mode.String == string(enums.LifeModeTeen)
	switch {
	case t == TypeParent && !teen:
		return ErrParentNeedsTeen
	case t != TypeParent && teen:
		return ErrTeenParentOnly
	}
	return nil
}

// applyGrants is the only grant writer: family.Grants.Apply (registry check, epoch stamp, activity diff, consent
// event) in the caller's transaction. ownerID is the actor (only the owner changes grants).
func applyGrants(ctx context.Context, tx *sql.Tx, c store.Companion, ownerID uint64, set map[family.Domain]family.Level,
	reason string, now time.Time) error {
	_, err := family.Grants{}.Apply(ctx, tx, family.Change{
		LinkID: c.ID, OwnerID: c.OwnerID, MemberID: uint64(max(c.CompanionUserID.Int64, 0)), ActorID: ownerID,
		Role: family.ParseRole(c.Role.String, c.Type), Set: set, ExpiresAt: c.ExpiresAt.Time, Reason: reason, Now: now,
	})
	if errors.Is(err, family.ErrLevelNotAllowed) {
		return ErrInvalidLevel
	}
	return err
}

func notFound(err error) error {
	if err == nil || errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("companion: %w", err)
}

func toLink(c store.Companion, grants []store.CompanionGrant) Link {
	l := Link{
		ID: c.ID, OwnerID: c.OwnerID, CompanionUserID: uint64(c.CompanionUserID.Int64), //nolint:gosec // positive id
		Type: Type(c.Type), Status: Status(c.Status), DisplayName: c.DisplayName.String,
		InvitedAt: c.InvitedAt.Time, AcceptedAt: c.AcceptedAt.Time, Grants: Grants{}, Preferences: preferencesOf(c),
	}
	for _, g := range grants {
		if g.CompanionID != c.ID {
			continue
		}
		if sec, lv, ok := legacyOf(g.Section, g.Level); ok && sec.AllowedFor(l.Type) {
			l.Grants[sec] = lv
		}
	}
	return l
}

func loadLink(ctx context.Context, q *store.Queries, companionID uint64) (Link, error) {
	c, err := q.GetCompanion(ctx, companionID)
	if err != nil {
		return Link{}, notFound(err)
	}
	grants, err := q.ListGrants(ctx, companionID)
	if err != nil {
		return Link{}, fmt.Errorf("companion: grants: %w", err)
	}
	l := toLink(c, grants)
	return l, loadFamily(ctx, q, &l)
}

func loadFamily(ctx context.Context, q *store.Queries, l *Link) error {
	if l.Type != TypeSpouse {
		return nil
	}
	f, err := q.GetFamilyByCompanion(ctx, l.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("companion: family: %w", err)
	}
	l.FamilyID = f.ID
	l.SharedChildIDs, err = q.ListFamilyChildIDs(ctx, f.ID)
	if err != nil {
		return fmt.Errorf("companion: family children: %w", err)
	}
	return nil
}
