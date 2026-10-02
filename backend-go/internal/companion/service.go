package companion

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Defaults of Options.
const (
	DefaultInviteTTL         = 24 * time.Hour
	DefaultMaxAttempts       = 5
	DefaultMaxOpenCompanions = 10
	maxDisplayName           = 100
	maxCodeRetries           = 5
)

// defaultPepper keys the code HMAC when Options.CodePepper is empty (tests, local). B-N4-02 wires a secret from env.
var defaultPepper = []byte("ritme-companion-invite")

// Options tunes the service; zero values take the defaults.
type Options struct {
	// CodePepper keys the HMAC of invite codes at rest (secret, from env).
	CodePepper []byte
	// InviteTTL is how long an invite stays valid (24 h).
	InviteTTL time.Duration
	// MaxAttempts locks an invite after this many failed attempts (5).
	MaxAttempts int
	// MaxOpenCompanions caps an owner's invited + active links (10).
	MaxOpenCompanions int
	// Rand is the code entropy source (crypto/rand).
	Rand io.Reader
}

// Conn is the database the service writes through (a *sql.DB).
type Conn interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Service is the companion domain.
type Service struct {
	conn        Conn
	clock       clock.Clock
	pepper      []byte
	ttl         time.Duration
	maxAttempts int
	maxOpen     int
	rand        io.Reader
}

// NewService returns a Service on conn.
func NewService(conn Conn, clk clock.Clock, opt Options) *Service {
	s := &Service{conn: conn, clock: clk, pepper: opt.CodePepper, ttl: opt.InviteTTL, maxAttempts: opt.MaxAttempts,
		maxOpen: opt.MaxOpenCompanions, rand: opt.Rand}
	if len(s.pepper) == 0 {
		s.pepper = defaultPepper
	}
	if s.ttl <= 0 {
		s.ttl = DefaultInviteTTL
	}
	if s.maxAttempts <= 0 {
		s.maxAttempts = DefaultMaxAttempts
	}
	if s.maxOpen <= 0 {
		s.maxOpen = DefaultMaxOpenCompanions
	}
	if s.rand == nil {
		s.rand = rand.Reader
	}
	return s
}

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
}

// InviteInput is the Hamdam_Type → Access → Children → Invite flow.
type InviteInput struct {
	Type        Type
	DisplayName string // «اسمش», optional
	Phone       string // optional; binds the invite to this mobile
	Grants      Grants
	// ChildIDs are the shared children (spouse only). The caller verifies ownership (B-N5-02 children service).
	ChildIDs []uint64
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

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// inTx runs fn on one READ COMMITTED transaction.
func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("companion: begin: %w", err)
	}
	if err := fn(store.New(tx)); err != nil {
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
	if in.Phone != "" {
		m, ok := NormalizeMobile(in.Phone)
		if !ok {
			return ErrInvalidPhone
		}
		in.Phone = m
	}
	if err := in.Grants.Validate(); err != nil {
		return err
	}
	if len(in.ChildIDs) > 0 && in.Type != TypeSpouse {
		return ErrChildrenNoSpouse
	}
	return nil
}

// CreateInvite makes a pending link with the chosen type, grants and (spouse) shared children, and a one-time code
// valid for InviteTTL. Nothing is visible to anyone until the code is accepted.
func (s *Service) CreateInvite(ctx context.Context, ownerID uint64, in InviteInput) (CreatedInvite, error) {
	if err := validateInput(&in); err != nil {
		return CreatedInvite{}, err
	}
	now := s.now()
	var out CreatedInvite
	err := s.inTx(ctx, func(q *store.Queries) error {
		if err := lockOwner(ctx, q, ownerID); err != nil {
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
		})
		if err != nil {
			return fmt.Errorf("companion: create: %w", err)
		}
		cid := uint64(id) //nolint:gosec // AUTO_INCREMENT ids are positive
		if err := writeGrants(ctx, q, cid, in.Grants, now); err != nil {
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
	expires := now.Add(s.ttl)
	for range maxCodeRetries {
		code, err := newCode(s.rand)
		if err != nil {
			return "", time.Time{}, err
		}
		_, err = q.CreateInvite(ctx, store.CreateInviteParams{
			CompanionID: companionID, OwnerID: ownerID, Phone: nstr(phone), CodeHash: s.hashCode(code),
			ExpiresAt: nt(expires), Now: nt(now),
		})
		if err == nil {
			return code, expires, nil
		}
		if !isDuplicate(err) {
			return "", time.Time{}, fmt.Errorf("companion: invite: %w", err)
		}
	}
	return "", time.Time{}, errors.New("companion: invite: no free code")
}

// Accept redeems a code for viewerID: the pending link becomes active with the grants the owner chose, and a spouse
// joins the owner's family. A bound invite redeemed by another number counts a failed attempt; MaxAttempts locks it.
// Callers rate-limit Accept per user (B-N4-02): an unknown code does not touch any row.
func (s *Service) Accept(ctx context.Context, viewerID uint64, typed string) (Link, error) {
	code, ok := NormalizeCode(typed)
	if !ok {
		return Link{}, ErrInviteInvalid
	}
	hash := s.hashCode(code)
	inv, err := store.New(s.conn).GetInviteByHash(ctx, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return Link{}, ErrInviteInvalid
	}
	if err != nil {
		return Link{}, fmt.Errorf("companion: invite: %w", err)
	}
	now := s.now()
	var link Link
	err = s.inTx(ctx, func(q *store.Queries) error {
		if err := lockOwner(ctx, q, inv.OwnerID); err != nil {
			return err
		}
		inv, err := q.GetInvite(ctx, inv.ID) // re-read under the owner lock
		if err != nil {
			return notFound(err)
		}
		switch {
		case inv.UsedAt.Valid || inv.RevokedAt.Valid:
			return ErrInviteUsed
		case !inv.ExpiresAt.Valid || !now.Before(inv.ExpiresAt.Time):
			return ErrInviteExpired
		case int(inv.Attempts) >= s.maxAttempts:
			return ErrInviteLocked
		case inv.OwnerID == viewerID:
			return ErrSelfInvite
		}
		if inv.Phone.Valid && inv.Phone.String != "" {
			mobile, err := q.GetUserMobile(ctx, viewerID)
			if err != nil {
				return notFound(err)
			}
			if !mobile.Valid || mobile.String != inv.Phone.String {
				return ErrInvitePhoneMismatch
			}
		}
		c, err := q.GetCompanion(ctx, inv.CompanionID)
		if err != nil {
			return notFound(err)
		}
		if Status(c.Status) != StatusInvited {
			return ErrInviteUsed
		}
		linked, err := q.CountOpenLinks(ctx, store.CountOpenLinksParams{OwnerID: inv.OwnerID, CompanionUserID: nid(viewerID)})
		if err != nil {
			return fmt.Errorf("companion: links: %w", err)
		}
		if linked > 0 {
			return ErrAlreadyLinked
		}
		if n, err := q.MarkInviteUsed(ctx, store.MarkInviteUsedParams{ID: inv.ID, UsedByID: nid(viewerID), Now: nt(now)}); err != nil {
			return fmt.Errorf("companion: use invite: %w", err)
		} else if n == 0 {
			return ErrInviteUsed
		}
		if n, err := q.ActivateCompanion(ctx, store.ActivateCompanionParams{ID: c.ID, CompanionUserID: nid(viewerID), Now: nt(now)}); err != nil {
			return fmt.Errorf("companion: activate: %w", err)
		} else if n == 0 {
			return ErrInviteUsed
		}
		if err := q.RevokeOpenInvites(ctx, store.RevokeOpenInvitesParams{CompanionID: c.ID, Now: nt(now)}); err != nil {
			return fmt.Errorf("companion: revoke invites: %w", err)
		}
		if Type(c.Type) == TypeSpouse {
			if err := q.SetFamilySpouse(ctx, store.SetFamilySpouseParams{CompanionID: c.ID, SpouseUserID: nid(viewerID), Now: nt(now)}); err != nil {
				return fmt.Errorf("companion: family spouse: %w", err)
			}
		}
		if err := audit(ctx, q, inv.OwnerID, viewerID, c.ID, "", ActionAccepted, now); err != nil {
			return err
		}
		link, err = loadLink(ctx, q, c.ID)
		return err
	})
	if errors.Is(err, ErrInvitePhoneMismatch) {
		if ierr := store.New(s.conn).IncrementInviteAttempts(ctx, store.IncrementInviteAttemptsParams{ID: inv.ID, Now: nt(now)}); ierr != nil {
			return Link{}, fmt.Errorf("companion: attempts: %w", ierr)
		}
	}
	return link, err
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
// lose access.
func (s *Service) SetGrants(ctx context.Context, ownerID, companionID uint64, grants Grants) (Link, error) {
	if err := grants.Validate(); err != nil {
		return Link{}, err
	}
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
		if err := q.DeleteGrants(ctx, c.ID); err != nil {
			return fmt.Errorf("companion: delete grants: %w", err)
		}
		if err := writeGrants(ctx, q, c.ID, grants, now); err != nil {
			return err
		}
		if err := audit(ctx, q, ownerID, ownerID, c.ID, "", ActionGrantsChanged, now); err != nil {
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
	links := make([]Link, 0, len(rows))
	for _, c := range rows {
		l := toLink(c, grants)
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
	row, err := store.New(s.conn).GetAccess(ctx, store.GetAccessParams{OwnerID: ownerID, ViewerID: nid(viewerID), Section: string(section)})
	if errors.Is(err, sql.ErrNoRows) {
		return LevelNone, 0, nil
	}
	if err != nil {
		return LevelNone, 0, fmt.Errorf("companion: access: %w", err)
	}
	if l := Level(row.Level); l.Valid() {
		return l, row.CompanionID, nil
	}
	return LevelNone, 0, nil
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
	return audit(ctx, store.New(s.conn), ownerID, actorID, companionID, section, action, s.now())
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

func writeGrants(ctx context.Context, q *store.Queries, companionID uint64, grants Grants, now time.Time) error {
	for _, sec := range Sections { // stable order
		l := grants.Of(sec)
		if l == LevelNone {
			continue
		}
		if err := q.UpsertGrant(ctx, store.UpsertGrantParams{CompanionID: companionID, Section: string(sec), Level: string(l), Now: nt(now)}); err != nil {
			return fmt.Errorf("companion: grant: %w", err)
		}
	}
	return nil
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
		InvitedAt: c.InvitedAt.Time, AcceptedAt: c.AcceptedAt.Time, Grants: Grants{},
	}
	for _, g := range grants {
		if g.CompanionID == c.ID && Section(g.Section).Valid() && Level(g.Level).Valid() {
			l.Grants[Section(g.Section)] = Level(g.Level)
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
