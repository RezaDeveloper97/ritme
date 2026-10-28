package family

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/consent"
)

// ErrorCodeLevelNotAllowed is the 422 error_code of a grant above what the registry allows for the role / owner.
const ErrorCodeLevelNotAllowed = "level_not_allowed"

// ActivityGrantsChanged is the activity action of a grant change. It keeps the legacy `grants_changed` code so the
// /companions audit trail (D-204 compatibility layer) reads it unchanged; diff_json carries the v6 per-domain diff.
const ActivityGrantsChanged = "grants_changed"

// LevelError is a grant above the registry's maximum for the domain, role and owner (never clamped silently).
type LevelError struct {
	Domain Domain
	Level  Level
	Max    Level
}

func (e *LevelError) Error() string {
	return fmt.Sprintf("family: %s level %s above %s", e.Domain, e.Level, e.Max)
}

// ErrLevelNotAllowed matches every *LevelError (errors.Is).
var ErrLevelNotAllowed = errors.New("family: level not allowed")

// Is makes errors.Is(err, ErrLevelNotAllowed) true.
func (e *LevelError) Is(target error) bool { return target == ErrLevelNotAllowed }

// Grant-change reasons (kept in the consent scope).
const (
	ChangeOwnerEdit    = "owner_edit"
	ChangeInvite       = "invite"
	ChangeAccept       = "accept"
	ChangeReapprove    = "reapprove"
	ChangeShareRequest = "share_request"
)

// Change is one grant change: the complete new grant set of a link (a domain left out is off).
type Change struct {
	LinkID   uint64
	OwnerID  uint64
	MemberID uint64 // 0 while the link is pending
	ActorID  uint64 // who made the change (the owner; the member on an accepted share request is still owner-approved)
	Role     Role
	Set      map[Domain]Level
	// ExpiresAt is the link's expiry (zero = none), recorded in the consent scope.
	ExpiresAt time.Time
	Reason    string
	Now       time.Time
}

// Applied is the outcome: the per-domain diff (code → [old, new]) and the epoch the grants were stamped with.
type Applied struct {
	Diff  map[string][2]string
	Epoch uint32
}

// Grants is the grant-change service: the only way a grant changes (owner edit, invite, accept, re-approve,
// share-request approval).
type Grants struct{}

// Validate checks a set against the registry: each domain grantable, each level within MaxFor(role, male).
func Validate(set map[Domain]Level, r Role, male bool) error {
	for d, l := range set {
		if l == LevelOff {
			continue
		}
		if l > LevelWrite {
			return &LevelError{Domain: d, Level: l, Max: LevelOff}
		}
		if m := MaxFor(d, r, male); l > m {
			return &LevelError{Domain: d, Level: l, Max: m}
		}
	}
	return nil
}

// Apply validates c against the registry (a *LevelError → 422 level_not_allowed), syncs the owner's mode, replaces
// the link's grants with c.Set stamped with the owner's current mode epoch, and writes the activity row (diff_json
// {domain: [old, new]}, FAM-L5) and a `family_sharing` consent_events row (source family, scope {link_id, member_id,
// role, domains, expires_at}) — all through tx, the caller's transaction.
func (Grants) Apply(ctx context.Context, tx store.DBTX, c Change) (Applied, error) {
	q := store.New(tx)
	in, err := q.FamilyOwnerModeInputs(ctx, c.OwnerID)
	if err != nil {
		return Applied{}, fmt.Errorf("family: apply: owner: %w", err)
	}
	male := in.Gender.Valid && in.Gender.String == "male"
	if err := Validate(c.Set, c.Role, male); err != nil {
		return Applied{}, err
	}
	st, err := SyncMode(ctx, tx, c.OwnerID, c.Now)
	if err != nil {
		return Applied{}, err
	}
	current, err := q.ListGrants(ctx, c.LinkID)
	if err != nil {
		return Applied{}, fmt.Errorf("family: apply: grants: %w", err)
	}
	now := sql.NullTime{Time: c.Now, Valid: true}
	diff := map[string][2]string{}
	old := map[string]string{}
	for _, g := range current {
		old[g.Section] = g.Level
	}
	reapprove := c.Reason == ChangeOwnerEdit || c.Reason == ChangeReapprove
	for code, lv := range old { // rows not in the new set: removed (also every legacy / unknown code)
		d, ok := ParseDomain(code)
		if ok && c.Set[d] > LevelOff {
			continue
		}
		if err := q.DeleteGrant(ctx, store.DeleteGrantParams{CompanionID: c.LinkID, Section: code}); err != nil {
			return Applied{}, fmt.Errorf("family: apply: delete: %w", err)
		}
		diff[code] = [2]string{lv, LevelOff.String()}
	}
	domains := map[string]string{}
	for _, d := range GrantDomains {
		l := c.Set[d]
		if l == LevelOff {
			continue
		}
		domains[d.String()] = l.String()
		prev, had := old[d.String()]
		if !had {
			prev = LevelOff.String()
		}
		// Epoch rule (L2): a grant whose level did not change keeps its stored mode_epoch — a paused grant stays paused
		// (accept, share-request answers). Only the owner's own choice stamps the current epoch: a new or changed level,
		// or an owner PUT of the full set — a PUT that includes a paused domain at a level > off is the explicit
		// re-approval of that domain (the review card's «دوباره تأیید» sends exactly that PUT).
		if had && prev == l.String() && !reapprove {
			continue
		}
		if err := q.UpsertFamilyGrant(ctx, store.UpsertFamilyGrantParams{
			CompanionID: c.LinkID, Section: d.String(), Level: l.String(),
			ModeEpoch: sql.NullInt32{Int32: int32(st.Epoch), Valid: st.Epoch > 0}, Now: now, //nolint:gosec // small epochs
		}); err != nil {
			return Applied{}, fmt.Errorf("family: apply: grant: %w", err)
		}
		if prev != l.String() {
			diff[d.String()] = [2]string{prev, l.String()}
		}
	}
	rawDiff, err := json.Marshal(diff)
	if err != nil {
		return Applied{}, fmt.Errorf("family: apply: diff: %w", err)
	}
	if _, err := q.InsertFamilyActivity(ctx, store.InsertFamilyActivityParams{
		OwnerID: c.OwnerID, ActorID: nid(c.ActorID), CompanionID: nid(c.LinkID), Action: ActivityGrantsChanged,
		DiffJson: rootdb.NullRawJSON{V: rawDiff, Valid: true}, Now: now,
	}); err != nil {
		return Applied{}, fmt.Errorf("family: apply: activity: %w", err)
	}
	action := consent.ActionGrant
	if len(domains) == 0 {
		action = consent.ActionRevoke
	}
	var expires any
	if !c.ExpiresAt.IsZero() {
		expires = c.ExpiresAt.Format(time.RFC3339)
	}
	var member any
	if c.MemberID != 0 {
		member = c.MemberID
	}
	if err := consent.AppendScoped(ctx, tx, consent.Scoped{
		UserID: c.OwnerID, Consent: consent.FamilySharing, Action: action, Source: consent.SourceFamily, At: c.Now,
		Scope: map[string]any{"link_id": c.LinkID, "member_id": member, "role": c.Role.String(), "domains": domains,
			"expires_at": expires, "reason": c.Reason},
	}); err != nil {
		return Applied{}, err
	}
	return Applied{Diff: diff, Epoch: st.Epoch}, nil
}

func nid(id uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(id), Valid: id != 0} } //nolint:gosec // ids fit
