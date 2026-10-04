package companion_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// tick is a settable clock.
type tick struct {
	mu sync.Mutex
	t  time.Time
}

func (c *tick) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *tick) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type env struct {
	db  *sql.DB
	clk *tick
	svc *companion.Service
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	clk := &tick{t: time.Date(2026, 10, 2, 10, 0, 0, 0, civildate.Tehran)}
	return &env{db: db, clk: clk, svc: companion.NewService(db, clk, companion.Options{CodePepper: []byte("test-pepper")})}
}

func (e *env) user(t *testing.T, mobile string) uint64 {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // positive id
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

func TestInviteAccept_ActivatesWithChosenGrants(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")

	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{
		Type: companion.TypePartner, DisplayName: "Ali",
		Grants: companion.Grants{companion.SectionMeds: companion.LevelEdit, companion.SectionCycle: companion.LevelView, companion.SectionSymptoms: companion.LevelNone},
	})
	require.NoError(t, err)
	assert.Len(t, inv.Code, companion.CodeLength)
	assert.Equal(t, e.clk.Now().Add(24*time.Hour), inv.ExpiresAt)
	assert.Equal(t, companion.StatusInvited, inv.Link.Status)
	assert.Equal(t, companion.Grants{companion.SectionMeds: companion.LevelEdit, companion.SectionCycle: companion.LevelView}, inv.Link.Grants,
		"none is not stored")

	// The plain code is never stored.
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM companion_invites WHERE code_hash = ? OR code_hash LIKE ?", inv.Code, "%"+inv.Code+"%"))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM companion_invites WHERE CHAR_LENGTH(code_hash) = 64"))

	// Pending: nothing is visible yet.
	ok, err := e.svc.CanRead(ctx, owner, ali, companion.SectionMeds)
	require.NoError(t, err)
	assert.False(t, ok, "an invited link grants nothing")

	link, err := e.svc.Accept(ctx, ali, " "+inv.Code[:3]+"-"+inv.Code[3:]+" ")
	require.NoError(t, err)
	assert.Equal(t, companion.StatusActive, link.Status)
	assert.Equal(t, ali, link.CompanionUserID)
	assert.Equal(t, e.clk.Now(), link.AcceptedAt)

	for _, c := range []struct {
		sec         companion.Section
		read, write bool
	}{
		{companion.SectionMeds, true, true},
		{companion.SectionCycle, true, false},
		{companion.SectionSymptoms, false, false},
		{companion.SectionAppointments, false, false},
		{companion.SectionPregnancy, false, false},
		{companion.Section("unknown"), false, false},
	} {
		r, err := e.svc.CanRead(ctx, owner, ali, c.sec)
		require.NoError(t, err)
		w, err := e.svc.CanWrite(ctx, owner, ali, c.sec)
		require.NoError(t, err)
		assert.Equal(t, c.read, r, c.sec)
		assert.Equal(t, c.write, w, c.sec)
	}

	// Direction matters: the owner has no access to Ali's data.
	r, err := e.svc.CanRead(ctx, ali, owner, companion.SectionMeds)
	require.NoError(t, err)
	assert.False(t, r)
	// The owner always has full access to her own data.
	w, err := e.svc.CanWrite(ctx, owner, owner, companion.SectionPregnancy)
	require.NoError(t, err)
	assert.True(t, w)

	// Both sides list the link.
	own, err := e.svc.ListForOwner(ctx, owner)
	require.NoError(t, err)
	require.Len(t, own, 1)
	assert.Equal(t, "Ali", own[0].DisplayName)
	assert.True(t, own[0].InviteExpiresAt.IsZero(), "no open invite after accept")
	theirs, err := e.svc.ListForCompanion(ctx, ali)
	require.NoError(t, err)
	require.Len(t, theirs, 1)
	assert.Equal(t, owner, theirs[0].OwnerID)
	assert.Equal(t, companion.LevelEdit, theirs[0].Grants.Of(companion.SectionMeds))
}

func TestAccept_OneTimeUse(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")
	reza := e.user(t, "09120000003")

	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.NoError(t, err)

	_, err = e.svc.Accept(ctx, reza, inv.Code)
	require.ErrorIs(t, err, companion.ErrInviteUsed)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.ErrorIs(t, err, companion.ErrInviteUsed)

	theirs, err := e.svc.ListForCompanion(ctx, reza)
	require.NoError(t, err)
	assert.Empty(t, theirs)
}

func TestAccept_ConcurrentRedeemsOnlyOnce(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	users := []uint64{e.user(t, "09120000002"), e.user(t, "09120000003"), e.user(t, "09120000004"), e.user(t, "09120000005")}
	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner, Grants: companion.Grants{companion.SectionCycle: companion.LevelView}})
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make([]error, len(users))
	for i, u := range users {
		wg.Go(func() { _, errs[i] = e.svc.Accept(ctx, u, inv.Code) })
	}
	wg.Wait()
	wins := 0
	for _, err := range errs {
		if err == nil {
			wins++
		} else {
			require.ErrorIs(t, err, companion.ErrInviteUsed)
		}
	}
	assert.Equal(t, 1, wins)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM companions WHERE owner_id = ? AND status = 'active'", owner))
}

func TestAccept_Expiry(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")

	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.NoError(t, err)
	e.clk.add(24*time.Hour - time.Second)
	own, err := e.svc.ListForOwner(ctx, owner)
	require.NoError(t, err)
	assert.Equal(t, inv.ExpiresAt, own[0].InviteExpiresAt, "still open one second before expiry")

	e.clk.add(time.Second)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.ErrorIs(t, err, companion.ErrInviteExpired)
	own, err = e.svc.ListForOwner(ctx, owner)
	require.NoError(t, err)
	assert.True(t, own[0].InviteExpiresAt.IsZero(), "an expired invite is not open")

	// Renewing issues a new code; the old one stays dead.
	renewed, err := e.svc.RenewInvite(ctx, owner, inv.Link.ID, "")
	require.NoError(t, err)
	assert.NotEqual(t, inv.Code, renewed.Code)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.ErrorIs(t, err, companion.ErrInviteUsed, "renewal revokes older invites")
	link, err := e.svc.Accept(ctx, ali, renewed.Code)
	require.NoError(t, err)
	assert.Equal(t, inv.Link.ID, link.ID, "same link, new code")

	_, err = e.svc.RenewInvite(ctx, owner, inv.Link.ID, "")
	require.ErrorIs(t, err, companion.ErrNotPending)
}

func TestAccept_InvalidCodes(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	ali := e.user(t, "09120000002")
	for _, code := range []string{"", "ABC", "RT0K2M", "ZZZZZZ"} {
		_, err := e.svc.Accept(ctx, ali, code)
		require.ErrorIs(t, err, companion.ErrInviteInvalid, code)
	}
}

func TestAccept_PhoneBoundInviteCountsAttemptsAndLocks(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")
	stranger := e.user(t, "09120000009")

	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner, Phone: "+98 912 000 0002"})
	require.NoError(t, err)
	assert.Equal(t, "09120000002", inv.Phone)

	for range companion.DefaultMaxAttempts {
		_, err := e.svc.Accept(ctx, stranger, inv.Code)
		require.ErrorIs(t, err, companion.ErrInvitePhoneMismatch)
	}
	assert.Equal(t, companion.DefaultMaxAttempts, e.count(t, "SELECT attempts FROM companion_invites WHERE companion_id = ?", inv.Link.ID))
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.ErrorIs(t, err, companion.ErrInviteLocked, "locked even for the right number")

	// A renewal keeps the phone binding and resets the counter.
	renewed, err := e.svc.RenewInvite(ctx, owner, inv.Link.ID, "")
	require.NoError(t, err)
	assert.Equal(t, "09120000002", renewed.Phone)
	_, err = e.svc.Accept(ctx, stranger, renewed.Code)
	require.ErrorIs(t, err, companion.ErrInvitePhoneMismatch)
	_, err = e.svc.Accept(ctx, ali, renewed.Code)
	require.NoError(t, err)
}

func TestInvite_Guards(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")

	_, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner, Phone: "09120000001"})
	require.ErrorIs(t, err, companion.ErrSelfInvite)

	self, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, owner, self.Code)
	require.ErrorIs(t, err, companion.ErrSelfInvite)

	// Already linked: a second invite to the same account cannot be accepted.
	first, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, ali, first.Code)
	require.NoError(t, err)
	second, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypeSpouse})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, ali, second.Code)
	require.ErrorIs(t, err, companion.ErrAlreadyLinked)

	// One open spouse per owner.
	_, err = e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypeSpouse})
	require.ErrorIs(t, err, companion.ErrSpouseExists)

	// Cap on open links (self + first + second = 3 open).
	small := companion.NewService(e.db, e.clk, companion.Options{MaxOpenCompanions: 3})
	_, err = small.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.ErrorIs(t, err, companion.ErrTooManyCompanions)

	_, err = e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: "friend"})
	require.ErrorIs(t, err, companion.ErrInvalidType)
	_, err = e.svc.CreateInvite(ctx, 999999, companion.InviteInput{Type: companion.TypePartner})
	require.ErrorIs(t, err, companion.ErrNotFound)
}

func TestRevoke_RemovesAccess(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")
	other := e.user(t, "09120000003")

	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner, Grants: companion.Grants{companion.SectionCycle: companion.LevelEdit}})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.NoError(t, err)

	// A stranger cannot revoke (and does not learn the link exists).
	require.ErrorIs(t, e.svc.Revoke(ctx, other, inv.Link.ID), companion.ErrNotFound)
	require.ErrorIs(t, e.svc.Revoke(ctx, owner, 999999), companion.ErrNotFound)

	require.NoError(t, e.svc.Revoke(ctx, owner, inv.Link.ID))
	r, err := e.svc.CanRead(ctx, owner, ali, companion.SectionCycle)
	require.NoError(t, err)
	assert.False(t, r)
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM companion_grants WHERE companion_id = ?", inv.Link.ID), "grants deleted")
	assert.Equal(t, "owner", func() string {
		var by string
		require.NoError(t, e.db.QueryRow("SELECT revoked_by FROM companions WHERE id = ?", inv.Link.ID).Scan(&by))
		return by
	}())
	require.ErrorIs(t, e.svc.Revoke(ctx, owner, inv.Link.ID), companion.ErrNotFound, "already revoked")
	_, err = e.svc.SetGrants(ctx, owner, inv.Link.ID, companion.Grants{companion.SectionCycle: companion.LevelView})
	require.ErrorIs(t, err, companion.ErrNotFound, "a revoked link cannot be re-granted")

	own, err := e.svc.ListForOwner(ctx, owner)
	require.NoError(t, err)
	assert.Empty(t, own)

	// Re-inviting creates a fresh link.
	again, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.NoError(t, err)
	assert.NotEqual(t, inv.Link.ID, again.Link.ID)
	_, err = e.svc.Accept(ctx, ali, again.Code)
	require.NoError(t, err)
	r, err = e.svc.CanRead(ctx, owner, ali, companion.SectionCycle)
	require.NoError(t, err)
	assert.False(t, r, "the new link starts with no grants")
}

func TestRevoke_PendingInviteDies(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")
	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.NoError(t, err)
	require.ErrorIs(t, e.svc.Revoke(ctx, ali, inv.Link.ID), companion.ErrNotFound, "not a party yet")
	require.NoError(t, e.svc.Revoke(ctx, owner, inv.Link.ID))
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.ErrorIs(t, err, companion.ErrInviteUsed)
}

func TestRevoke_CompanionLeaves(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")
	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner, Grants: companion.Grants{companion.SectionMeds: companion.LevelView}})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.NoError(t, err)
	require.NoError(t, e.svc.Revoke(ctx, ali, inv.Link.ID))
	var by string
	require.NoError(t, e.db.QueryRow("SELECT revoked_by FROM companions WHERE id = ?", inv.Link.ID).Scan(&by))
	assert.Equal(t, "companion", by)
	r, err := e.svc.CanRead(ctx, owner, ali, companion.SectionMeds)
	require.NoError(t, err)
	assert.False(t, r)
}

func TestGrantMatrix(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")
	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.NoError(t, err)

	levels := []companion.Level{companion.LevelNone, companion.LevelView, companion.LevelEdit}
	for _, sec := range companion.Sections {
		for _, lvl := range levels {
			// Every other section gets a different level, so a leak across sections shows up.
			g := companion.Grants{sec: lvl}
			for _, other := range companion.Sections {
				if other != sec {
					g[other] = companion.LevelNone
				}
			}
			link, err := e.svc.SetGrants(ctx, owner, inv.Link.ID, g)
			require.NoError(t, err)
			for _, s := range companion.Sections {
				want := companion.LevelNone
				if s == sec {
					want = lvl
				}
				got, err := e.svc.Level(ctx, owner, ali, s)
				require.NoError(t, err)
				assert.Equal(t, want, got, "%s=%s → %s", sec, lvl, s)
				assert.Equal(t, want, link.Grants.Of(s))
				r, err := e.svc.CanRead(ctx, owner, ali, s)
				require.NoError(t, err)
				w, err := e.svc.CanWrite(ctx, owner, ali, s)
				require.NoError(t, err)
				assert.Equal(t, want != companion.LevelNone, r)
				assert.Equal(t, want == companion.LevelEdit, w)
			}
		}
	}

	// Only the owner sets grants; bad input is refused.
	_, err = e.svc.SetGrants(ctx, ali, inv.Link.ID, companion.Grants{companion.SectionCycle: companion.LevelEdit})
	require.ErrorIs(t, err, companion.ErrNotFound)
	_, err = e.svc.SetGrants(ctx, owner, inv.Link.ID, companion.Grants{"diary": companion.LevelView})
	require.ErrorIs(t, err, companion.ErrInvalidSection)
	_, err = e.svc.SetGrants(ctx, owner, inv.Link.ID, companion.Grants{companion.SectionCycle: "admin"})
	require.ErrorIs(t, err, companion.ErrInvalidLevel)

	// A grant on one owner's link says nothing about another owner.
	owner2 := e.user(t, "09120000003")
	r, err := e.svc.CanRead(ctx, owner2, ali, companion.SectionCycle)
	require.NoError(t, err)
	assert.False(t, r)
}

func TestSpouse_FamilyAndSharedChildren(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")
	// family_children.child_id references children (B-N5-02): the owner's children 3, 7 and 9.
	for _, id := range []int{3, 7, 9} {
		_, err := e.db.Exec(`INSERT INTO children (id, owner_id, name, birth_date, created_at, updated_at)
			VALUES (?, ?, 'Child', '2026-01-01', NOW(), NOW())`, id, owner)
		require.NoError(t, err)
	}

	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypeSpouse, ChildIDs: []uint64{7, 3, 7}})
	require.NoError(t, err)
	assert.NotZero(t, inv.Link.FamilyID)
	assert.Equal(t, []uint64{3, 7}, inv.Link.SharedChildIDs)
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM families WHERE spouse_user_id IS NOT NULL"), "spouse set on accept")

	link, err := e.svc.Accept(ctx, ali, inv.Code)
	require.NoError(t, err)
	assert.Equal(t, inv.Link.FamilyID, link.FamilyID)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM families WHERE owner_id = ? AND spouse_user_id = ?", owner, ali))

	link, err = e.svc.SetSharedChildren(ctx, owner, inv.Link.ID, []uint64{9})
	require.NoError(t, err)
	assert.Equal(t, []uint64{9}, link.SharedChildIDs)
	theirs, err := e.svc.ListForCompanion(ctx, ali)
	require.NoError(t, err)
	require.Len(t, theirs, 1)
	assert.Equal(t, []uint64{9}, theirs[0].SharedChildIDs)

	_, err = e.svc.SetSharedChildren(ctx, ali, inv.Link.ID, []uint64{1})
	require.ErrorIs(t, err, companion.ErrNotFound, "only the owner")

	partner, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner})
	require.NoError(t, err)
	_, err = e.svc.SetSharedChildren(ctx, owner, partner.Link.ID, []uint64{1})
	require.ErrorIs(t, err, companion.ErrChildrenNoSpouse)

	// Revoking the spouse dissolves the family.
	require.NoError(t, e.svc.Revoke(ctx, owner, inv.Link.ID))
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM families"))
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM family_children"))
	// …and a new spouse can be invited.
	_, err = e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypeSpouse})
	require.NoError(t, err)
}

func TestAudit_NoPayloadAndOwnerScoped(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000001")
	ali := e.user(t, "09120000002")
	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner, Grants: companion.Grants{companion.SectionMeds: companion.LevelEdit}})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.NoError(t, err)
	e.clk.add(time.Minute)
	require.NoError(t, e.svc.Audit(ctx, owner, ali, inv.Link.ID, companion.SectionMeds, companion.ActionRead))
	require.NoError(t, e.svc.Audit(ctx, owner, ali, 0, companion.SectionMeds, companion.ActionWrite))
	require.NoError(t, e.svc.Audit(ctx, owner, owner, 0, companion.SectionMeds, companion.ActionRead), "own reads are not recorded")
	require.ErrorIs(t, e.svc.Audit(ctx, owner, ali, 0, "diary", companion.ActionRead), companion.ErrInvalidSection)

	trail, err := e.svc.AuditTrail(ctx, owner, 0)
	require.NoError(t, err)
	require.Len(t, trail, 4)
	assert.Equal(t, companion.ActionWrite, trail[0].Action)
	assert.Equal(t, companion.ActionRead, trail[1].Action)
	assert.Equal(t, companion.SectionMeds, trail[1].Section)
	assert.Equal(t, ali, trail[1].ActorID)
	assert.Equal(t, inv.Link.ID, trail[1].CompanionID)
	assert.Equal(t, e.clk.Now(), trail[1].At)
	assert.Equal(t, companion.ActionAccepted, trail[2].Action)
	assert.Equal(t, companion.ActionInvited, trail[3].Action)

	others, err := e.svc.AuditTrail(ctx, ali, 10)
	require.NoError(t, err)
	assert.Empty(t, others)

	// The audit trail survives the companion's account deletion (actor SET NULL), dies with the owner's.
	_, err = e.db.Exec("DELETE FROM users WHERE id = ?", ali)
	require.NoError(t, err)
	trail, err = e.svc.AuditTrail(ctx, owner, 10)
	require.NoError(t, err)
	require.Len(t, trail, 4)
	assert.Zero(t, trail[0].ActorID)
	_, err = e.db.Exec("DELETE FROM users WHERE id = ?", owner)
	require.NoError(t, err)
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM companion_audit_logs"))
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM companions"))
}
