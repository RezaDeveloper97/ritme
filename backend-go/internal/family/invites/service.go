// Package invites is family invites v6 (engine v6 M11, T-M11-08; SPEC §1.7–§1.8; docs/engine-v6/family.md §2, §7;
// DECISIONS.md F-11, F-12, F-13, F-17; D-196): the owner picks a role, a template and row edits; the member previews
// what she would see and accepts or declines, optionally answering a mutual request with an independent reverse link;
// a phone invite binds to the account with that number as a pending acceptance; a code-only invite that includes child
// write, the urgent alert or vitals waits for the owner's confirmation; a daily sweep expires stale invites and
// time-boxed links.
//
// It is the one invite service: /companions/* (internal/companion, the D-204 compatibility layer) issues and redeems
// its codes through Issue and Accept as well (D-196). Codes are 6 chars from a 32-symbol alphabet, stored as an
// HMAC-SHA256 under a pepper that is mandatory outside local / testing / contract (FAM-L1); code paths take at least
// Options.MinDuration (CMP-L2); a wrong-account try of a phone-bound code counts once per account (CMP-L8), and the
// bound account can always answer through its pending acceptance, which the attempt lock never blocks.
package invites

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Defaults of Options (family.md §7, F-11).
const (
	DefaultCodeTTL  = 24 * time.Hour
	DefaultPhoneTTL = 7 * 24 * time.Hour
	// DefaultConfirmTTL is how long an accepted high-impact code invite waits for the owner (awaiting_owner); past it
	// the confirmation is refused and the sweep revokes the link (expired).
	DefaultConfirmTTL  = 24 * time.Hour
	DefaultMaxAttempts = 5
	DefaultMaxOpen     = 10
	// DefaultMinDuration is the floor of every code path (preview, accept, decline) in production wiring (CMP-L2): an
	// unknown code answers no faster than a real one that opens a transaction and locks rows.
	DefaultMinDuration = 400 * time.Millisecond
	maxDisplayName     = 100
	maxCodeRetries     = 5
	sweepPage          = 200
)

// Link statuses and revoked_by values (F-11, F-12).
const (
	StatusInvited       = "invited"
	StatusActive        = "active"
	StatusAwaitingOwner = "awaiting_owner"
	StatusRevoked       = "revoked"

	RevokedByOwner    = "owner"
	RevokedByDeclined = "declined"
	RevokedByExpired  = "expired"
)

// Activity actions (companion_audit_logs.action; family.md §7 `invite_*`, `link_*`). invited / accepted / revoked keep
// the legacy codes the /companions audit trail reads.
const (
	ActionInvited       = "invited"
	ActionAccepted      = "accepted"
	ActionRevoked       = "revoked"
	ActionDeclined      = "invite_declined"
	ActionConfirmed     = "invite_confirmed"
	ActionInviteExpired = "invite_expired"
	ActionLinkExpire    = "link_expire"
	ActionLinkCreate    = "link_create"
)

// ConsentVersion is the version of the family-sharing text the owner must have seen to invite (consent_version). It
// mirrors the family_sharing evidence version that Grants.Apply stamps on consent_events (internal/consent keeps it
// evidence-only, so consent.CurrentVersion reports 0 for it); TestConsent_RecordedPerInviteAndChange pins the two.
const ConsentVersion = 1

// linkType is companions.type of a v6 link: the legacy column stays NOT NULL; role carries the v6 relation.
const linkType = "partner"

// Gate is a fixed-window counter (internal/http adapts the Redis limiter): Allow counts one hit on key.
type Gate interface {
	Allow(ctx context.Context, key string, maxHits int, window time.Duration) (bool, error)
}

// Options tunes the service; zero values take the defaults.
type Options struct {
	// Pepper keys the HMAC of codes at rest (COMPANION_CODE_PEPPER); empty → DevPepper (local / testing / contract).
	Pepper []byte
	// CodeTTL is how long a code-only invite lives (24 h); PhoneTTL a phone invite (7 days, F-11).
	CodeTTL, PhoneTTL time.Duration
	// ConfirmTTL is the owner's confirmation window of an awaiting_owner link, from acceptance (24 h).
	ConfirmTTL time.Duration
	// MaxAttempts locks a code after this many wrong-account tries (5).
	MaxAttempts int
	// MaxOpen caps an owner's open (invited, awaiting or active) links (10).
	MaxOpen int
	// Rand is the code entropy source (crypto/rand).
	Rand io.Reader
	// MinDuration is the floor of the code paths (CMP-L2); 0 = none (tests).
	MinDuration time.Duration
	// Mismatch counts the wrong-account tries of a phone-bound code once per account (CMP-L8); nil = every try counts.
	Mismatch Gate
}

// Conn is the database the service writes through (a *sql.DB).
type Conn interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Service is the invite domain.
type Service struct {
	conn  Conn
	clock clock.Clock
	opt   Options
}

// NewService returns a Service on conn.
func NewService(conn Conn, clk clock.Clock, opt Options) *Service {
	if len(opt.Pepper) == 0 {
		opt.Pepper = DevPepper
	}
	if opt.CodeTTL <= 0 {
		opt.CodeTTL = DefaultCodeTTL
	}
	if opt.PhoneTTL <= 0 {
		opt.PhoneTTL = DefaultPhoneTTL
	}
	if opt.ConfirmTTL <= 0 {
		opt.ConfirmTTL = DefaultConfirmTTL
	}
	if opt.MaxAttempts <= 0 {
		opt.MaxAttempts = DefaultMaxAttempts
	}
	if opt.MaxOpen <= 0 {
		opt.MaxOpen = DefaultMaxOpen
	}
	if opt.Rand == nil {
		opt.Rand = rand.Reader
	}
	if clk == nil {
		clk = clock.Real{}
	}
	return &Service{conn: conn, clock: clk, opt: opt}
}

// MaxAttempts is the lock threshold in force.
func (s *Service) MaxAttempts() int { return s.opt.MaxAttempts }

func (s *Service) now() time.Time { return s.clock.Now().In(civildate.Tehran).Truncate(time.Second) }

// Hash is the at-rest form of a code under the service pepper.
func (s *Service) Hash(code string) string { return HashCode(s.opt.Pepper, code) }

// TTL is the lifetime of an invite: 7 days bound to a phone, 24 h code-only (F-11).
func (s *Service) TTL(phone string) time.Duration {
	if phone != "" {
		return s.opt.PhoneTTL
	}
	return s.opt.CodeTTL
}

// floor holds a code path until MinDuration has passed since start (CMP-L2). The wait uses the monotonic clock, not
// the request clock (tests pin that).
func (s *Service) floor(ctx context.Context, start time.Time) {
	left := s.opt.MinDuration - time.Since(start)
	if left <= 0 {
		return
	}
	t := time.NewTimer(left)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: !t.IsZero()} }

func nid(id uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(id), Valid: id != 0} } //nolint:gosec // ids fit

func nstr(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }

func u64(v sql.NullInt64) uint64 {
	if !v.Valid || v.Int64 < 0 {
		return 0
	}
	return uint64(v.Int64)
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func (s *Service) inTx(ctx context.Context, fn func(tx *sql.Tx, q *store.Queries) error) error {
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("family invites: begin: %w", err)
	}
	if err := fn(tx, store.New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("family invites: commit: %w", err)
	}
	return nil
}

// lock takes the per-owner mutex rows (users FOR UPDATE) in id order, so two accepts that lock the same pair never
// deadlock.
func lock(ctx context.Context, q *store.Queries, ids ...uint64) error {
	slices.Sort(ids)
	for _, id := range ids {
		if _, err := q.LockOwner(ctx, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("family invites: lock: %w", err)
		}
	}
	return nil
}

func notFound(err error) error {
	if err == nil || errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("family invites: %w", err)
}

// Issue creates one invite of a pending link (in the caller's transaction): a fresh code, 24 h code-only or 7 days
// bound to a phone (F-11). A phone invite to an existing account is bound to it at once (pending acceptance); one to a
// number without an account is bound by the signup hook (BindPhone). The plain code exists only in the return value.
func (s *Service) Issue(ctx context.Context, q *store.Queries, ownerID, linkID uint64, phone string, now time.Time) (string, time.Time, error) {
	expires := now.Add(s.TTL(phone))
	for range maxCodeRetries {
		code, err := NewCode(s.opt.Rand)
		if err != nil {
			return "", time.Time{}, err
		}
		id, err := q.CreateInvite(ctx, store.CreateInviteParams{
			CompanionID: linkID, OwnerID: ownerID, Phone: nstr(phone), CodeHash: s.Hash(code),
			ExpiresAt: nt(expires), Now: nt(now),
		})
		if isDuplicate(err) {
			continue
		}
		if err != nil {
			return "", time.Time{}, fmt.Errorf("family invites: invite: %w", err)
		}
		if phone != "" {
			if err := bindExisting(ctx, q, uint64(id), ownerID, phone, now); err != nil { //nolint:gosec // auto-increment id
				return "", time.Time{}, err
			}
		}
		return code, expires, nil
	}
	return "", time.Time{}, errors.New("family invites: invite: no free code")
}

func bindExisting(ctx context.Context, q *store.Queries, inviteID, ownerID uint64, phone string, now time.Time) error {
	uid, err := q.FamilyUserIDByMobile(ctx, nstr(phone))
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("family invites: bind: %w", err)
	}
	if uid == ownerID {
		return nil
	}
	if _, err := q.BindInviteToUser(ctx, store.BindInviteToUserParams{ID: inviteID, UserID: nid(uid), Now: nt(now)}); err != nil {
		return fmt.Errorf("family invites: bind: %w", err)
	}
	return nil
}

// BindPhone is the auth signup hook (F-11): the open phone invites of the new account's number become its pending
// acceptances. Nothing becomes visible to anyone; the member answers through GET /family/invites/pending.
func (s *Service) BindPhone(ctx context.Context, userID uint64, mobile string, now time.Time) error {
	m, ok := NormalizeMobile(mobile)
	if !ok || userID == 0 {
		return nil
	}
	if now.IsZero() {
		now = s.now()
	}
	now = now.In(civildate.Tehran).Truncate(time.Second)
	q := store.New(s.conn)
	ids, err := q.ListUnboundPhoneInvites(ctx, store.ListUnboundPhoneInvitesParams{Phone: nstr(m), Now: nt(now), UserID: userID})
	if err != nil {
		return fmt.Errorf("family invites: signup bind: %w", err)
	}
	for _, id := range ids {
		if _, err := q.BindInviteToUser(ctx, store.BindInviteToUserParams{ID: id, UserID: nid(userID), Now: nt(now)}); err != nil {
			return fmt.Errorf("family invites: signup bind: %w", err)
		}
	}
	return nil
}

// SignupHook is BindPhone as an auth.SignupHook.
func (s *Service) SignupHook() func(ctx context.Context, userID uint64, mobile string, now time.Time) error {
	return s.BindPhone
}

// resolve finds the invite of a typed code, or the caller's pending acceptance of a link (pending = true).
func (s *Service) resolve(ctx context.Context, viewerID uint64, typed string, linkID uint64) (store.CompanionInvite, bool, error) {
	q := store.New(s.conn)
	if typed == "" && linkID != 0 {
		inv, err := q.GetBoundInvite(ctx, store.GetBoundInviteParams{CompanionID: linkID, UserID: nid(viewerID)})
		if err != nil {
			return store.CompanionInvite{}, true, notFound(err)
		}
		return inv, true, nil
	}
	code, ok := NormalizeCode(typed)
	if !ok {
		return store.CompanionInvite{}, false, ErrInvalid
	}
	inv, err := q.GetInviteByHash(ctx, s.Hash(code))
	if errors.Is(err, sql.ErrNoRows) {
		return store.CompanionInvite{}, false, ErrInvalid
	}
	if err != nil {
		return store.CompanionInvite{}, false, fmt.Errorf("family invites: invite: %w", err)
	}
	return inv, false, nil
}

// check applies the refusal rules to an invite (re-read under the owner lock by the writers). A pending acceptance
// is not subject to the attempt lock (CMP-L8: strangers locking a phone invite never block the bound account).
func (s *Service) check(ctx context.Context, q *store.Queries, inv store.CompanionInvite, viewerID uint64, now time.Time, pending bool) error {
	switch {
	case inv.UsedAt.Valid || inv.RevokedAt.Valid:
		return ErrUsed
	case !inv.ExpiresAt.Valid || !now.Before(inv.ExpiresAt.Time):
		return ErrExpired
	case !pending && int(inv.Attempts) >= s.opt.MaxAttempts:
		return ErrLocked
	case inv.OwnerID == viewerID:
		return ErrSelf
	}
	if pending {
		if u64(inv.UsedByID) != viewerID {
			return ErrNotFound
		}
		return nil
	}
	if inv.Phone.Valid && inv.Phone.String != "" {
		mobile, err := q.GetUserMobile(ctx, viewerID)
		if err != nil {
			return notFound(err)
		}
		if !mobile.Valid || mobile.String != inv.Phone.String {
			return ErrPhoneMismatch
		}
	}
	return nil
}

// countMismatch records one wrong-account try of a phone-bound code: once per (invite, account) with a gate
// (CMP-L8), every try without one. A gate error counts (fail closed toward the lock).
func (s *Service) countMismatch(ctx context.Context, inviteID, viewerID uint64, now time.Time) error {
	if s.opt.Mismatch != nil {
		key := "family-invite-mismatch:" + strconv.FormatUint(inviteID, 10) + ":" + strconv.FormatUint(viewerID, 10)
		if first, err := s.opt.Mismatch.Allow(ctx, key, 1, s.opt.PhoneTTL); err == nil && !first {
			return nil
		}
	}
	if err := store.New(s.conn).IncrementInviteAttempts(ctx, store.IncrementInviteAttemptsParams{ID: inviteID, Now: nt(now)}); err != nil {
		return fmt.Errorf("family invites: attempts: %w", err)
	}
	return nil
}

// teenRule is CB-TEEN-01 on accept: a parent (teen) link needs a teen-mode owner, any other link a non-teen one.
func teenRule(ctx context.Context, q *store.Queries, ownerID uint64, linkType string) error {
	mode, err := q.GetUserLifeMode(ctx, ownerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("family invites: life mode: %w", err)
	}
	teen := mode.Valid && mode.String == string(enums.LifeModeTeen)
	if (linkType == "parent") != teen {
		return ErrNotAllowed
	}
	return nil
}

// intent is what an invite asked for beyond its grants, kept on its `invited` activity row (diff_json, levels only).
type intent struct {
	Mutual         []string `json:"mutual_request,omitempty"`
	Children       string   `json:"children,omitempty"`
	Channel        string   `json:"channel,omitempty"`
	ConsentVersion int      `json:"consent_version,omitempty"`
}

func readIntent(ctx context.Context, q *store.Queries, linkID uint64) (intent, error) {
	var in intent
	raw, err := q.GetLinkInviteIntent(ctx, nid(linkID))
	if errors.Is(err, sql.ErrNoRows) {
		return in, nil
	}
	if err != nil {
		return in, fmt.Errorf("family invites: intent: %w", err)
	}
	if raw.Valid && len(raw.V) > 0 {
		_ = json.Unmarshal(raw.V, &in) // a legacy row has no intent
	}
	return in, nil
}

func (in intent) mutualDomains() []family.Domain {
	out := make([]family.Domain, 0, len(in.Mutual))
	for _, c := range in.Mutual {
		if d, ok := family.ParseDomain(c); ok {
			out = append(out, d)
		}
	}
	return out
}

// activity writes one family activity row (no health payload).
func activity(ctx context.Context, q *store.Queries, ownerID, actorID, linkID uint64, action string, diff any, now time.Time) error {
	var raw rootdb.NullRawJSON
	if diff != nil {
		b, err := json.Marshal(diff)
		if err != nil {
			return fmt.Errorf("family invites: activity: %w", err)
		}
		raw = rootdb.NullRawJSON{V: b, Valid: true}
	}
	if _, err := q.InsertFamilyActivity(ctx, store.InsertFamilyActivityParams{
		OwnerID: ownerID, ActorID: nid(actorID), CompanionID: nid(linkID), Action: action, DiffJson: raw, Now: nt(now),
	}); err != nil {
		return fmt.Errorf("family invites: activity: %w", err)
	}
	return nil
}

// grantSet reads a link's stored grants as v6 levels (unknown or legacy codes are left out).
func grantSet(ctx context.Context, q *store.Queries, linkID uint64) (map[family.Domain]family.Level, error) {
	rows, err := q.ListGrants(ctx, linkID)
	if err != nil {
		return nil, fmt.Errorf("family invites: grants: %w", err)
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

// needsOwner reports whether accepting the invite waits for the owner's confirmation (F-11, FAM-M5): a code-only
// invite (anyone holding the code could redeem it) that includes any write level, the urgent alert, vitals, child
// write (v6 intent), or — on a legacy /companions spouse invite — shared children. It reads the link's stored grants
// and family, so legacy invites without an intent row are covered too. A phone-bound invite never waits.
func needsOwner(ctx context.Context, q *store.Queries, inv store.CompanionInvite, c store.Companion,
	set map[family.Domain]family.Level, in intent) (bool, error) {
	if inv.Phone.Valid && inv.Phone.String != "" {
		return false, nil
	}
	for _, l := range set {
		if l >= family.LevelWrite {
			return true, nil
		}
	}
	if set[family.DomainUrgentAlert] > family.LevelOff || set[family.DomainVitals] > family.LevelOff ||
		in.Children == family.LevelWrite.String() || in.Children == childrenLimited {
		return true, nil
	}
	if c.Type != "spouse" {
		return false, nil
	}
	f, err := q.GetFamilyByCompanion(ctx, c.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("family invites: family: %w", err)
	}
	kids, err := q.ListFamilyChildIDs(ctx, f.ID)
	if err != nil {
		return false, fmt.Errorf("family invites: family children: %w", err)
	}
	return len(kids) > 0, nil
}

// ownerNotTeen is CB-TEEN-01 for a link the caller is about to own outside the invite flow (a mutual back-link, an
// approved share request): a teen owns only her parent link.
func ownerNotTeen(ctx context.Context, q *store.Queries, ownerID uint64) error {
	mode, err := q.GetUserLifeMode(ctx, ownerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("family invites: life mode: %w", err)
	}
	if mode.Valid && mode.String == string(enums.LifeModeTeen) {
		return ErrTeenParentOnly
	}
	return nil
}

// cleanupLink ends what an ended link leaves behind: open invites, grants, the spouse family, open share requests and
// the linked member notices (the link row itself stays for the audit trail).
func cleanupLink(ctx context.Context, q *store.Queries, linkID uint64, now time.Time) error {
	if err := q.RevokeOpenInvites(ctx, store.RevokeOpenInvitesParams{CompanionID: linkID, Now: nt(now)}); err != nil {
		return fmt.Errorf("family invites: revoke invites: %w", err)
	}
	if err := q.DeleteGrants(ctx, linkID); err != nil {
		return fmt.Errorf("family invites: delete grants: %w", err)
	}
	if err := q.DeleteFamilyByCompanion(ctx, linkID); err != nil {
		return fmt.Errorf("family invites: delete family: %w", err)
	}
	if _, err := q.ExpirePendingFamilyShareRequestsForLink(ctx, store.ExpirePendingFamilyShareRequestsForLinkParams{
		LinkID: nid(linkID), Now: nt(now)}); err != nil {
		return fmt.Errorf("family invites: share requests: %w", err)
	}
	if _, err := q.DeleteUserNotificationsForLink(ctx, nid(linkID)); err != nil {
		return fmt.Errorf("family invites: notices: %w", err)
	}
	if err := q.DeletePregnancyNoticesForLink(ctx, int64(linkID)); err != nil { //nolint:gosec // auto-increment id
		return fmt.Errorf("family invites: notices: %w", err)
	}
	return nil
}
