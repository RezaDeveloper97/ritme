package family_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

var now = time.Date(2026, 10, 28, 10, 0, 0, 0, time.Local)

// countingDB counts every statement sent through it.
type countingDB struct {
	*sql.DB
	n atomic.Int64
}

func (c *countingDB) ExecContext(ctx context.Context, q string, a ...any) (sql.Result, error) {
	c.n.Add(1)
	return c.DB.ExecContext(ctx, q, a...)
}

func (c *countingDB) QueryContext(ctx context.Context, q string, a ...any) (*sql.Rows, error) {
	c.n.Add(1)
	return c.DB.QueryContext(ctx, q, a...)
}

func (c *countingDB) QueryRowContext(ctx context.Context, q string, a ...any) *sql.Row {
	c.n.Add(1)
	return c.DB.QueryRowContext(ctx, q, a...)
}

type env struct {
	t  *testing.T
	db *sql.DB
	n  int
}

func newEnv(t *testing.T) *env { return &env{t: t, db: testdb.New(t)} }

func (e *env) exec(q string, args ...any) int64 {
	e.t.Helper()
	res, err := e.db.Exec(q, args...)
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	return id
}

func (e *env) count(q string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

// user makes an account with a life profile (mode "" = no stored mode → cycle).
func (e *env) user(gender, mode string) uint64 {
	e.t.Helper()
	e.n++
	id := e.exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, NOW(), NOW())`, fmt.Sprintf("0912666%04d", e.n))
	var lm any
	if mode != "" {
		lm = mode
	}
	e.exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`, id, gender, lm)
	return uint64(id) //nolint:gosec // test id
}

// link is an active partner link with grants applied through family.Grants.Apply.
func (e *env) link(owner, member uint64, set map[family.Domain]family.Level) uint64 {
	e.t.Helper()
	id := uint64(e.exec(`INSERT INTO companions (owner_id, companion_user_id, type, role, template, status, invited_at, accepted_at, created_at, updated_at)
		VALUES (?, ?, 'partner', 'partner', 'custom', 'active', NOW(), NOW(), NOW(), NOW())`, owner, member)) //nolint:gosec // test id
	e.apply(id, owner, member, set)
	return id
}

func (e *env) apply(link, owner, member uint64, set map[family.Domain]family.Level) family.Applied {
	e.t.Helper()
	tx, err := e.db.Begin()
	require.NoError(e.t, err)
	out, err := family.Grants{}.Apply(context.Background(), tx, family.Change{LinkID: link, OwnerID: owner, MemberID: member,
		ActorID: owner, Role: family.RolePartner, Set: set, Reason: family.ChangeOwnerEdit, Now: now})
	require.NoError(e.t, err)
	require.NoError(e.t, tx.Commit())
	return out
}

func decide(t *testing.T, db *sql.DB, owner, member uint64, d family.Domain, a family.Action) family.Decision {
	t.Helper()
	s, err := family.NewLoader(db, func() time.Time { return now }).Pair(context.Background(), owner, member)
	require.NoError(t, err)
	return family.Decide(s, family.Subject{}, d, a, now)
}

func TestSnapshot_OneStatementPerPair(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	e.link(owner, member, map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView, family.DomainMeds: family.LevelNudge})

	cdb := &countingDB{DB: e.db}
	l := family.NewLoader(cdb, func() time.Time { return now })
	s, err := l.Pair(context.Background(), owner, member)
	require.NoError(t, err)
	assert.EqualValues(t, 1, cdb.n.Load(), "one authorization statement (state already synced by Apply)")
	assert.True(t, s.HasLink)
	assert.Equal(t, family.LevelNudge, s.Grants[family.DomainMeds].Level)
	for range 3 {
		_, err = l.Pair(context.Background(), owner, member)
		require.NoError(t, err)
	}
	assert.EqualValues(t, 1, cdb.n.Load(), "memoised for the request")
	assert.Equal(t, family.Allow, family.Decide(s, family.Subject{}, family.DomainCyclePhase, family.ActionRead, now).Outcome)
	assert.Equal(t, family.Allow, family.Decide(s, family.Subject{}, family.DomainMeds, family.ActionNudge, now).Outcome)
	assert.Equal(t, family.Deny, family.Decide(s, family.Subject{}, family.DomainMeds, family.ActionWrite, now).Outcome)

	// The owner herself needs no statement; the actor's guardianships of a set of children take one.
	_, err = l.Pair(context.Background(), owner, owner)
	require.NoError(t, err)
	assert.EqualValues(t, 1, cdb.n.Load())
	child := uint64(e.exec(`INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, 'C', '2026-01-01', NOW(), NOW())`, owner)) //nolint:gosec // test id
	e.exec(`INSERT INTO child_guardians (child_id, user_id, level, sub_scopes, status, created_at, updated_at)
		VALUES (?, ?, 'limited', '["feeding"]', 'active', NOW(), NOW())`, child, member)
	s, err = l.WithGuardians(context.Background(), s, child, child+1)
	require.NoError(t, err)
	assert.EqualValues(t, 2, cdb.n.Load())
	assert.Equal(t, family.Allow, family.Decide(s, family.Subject{Kind: family.SubjectChild, ChildID: child, Scope: family.ScopeFeeding}, family.DomainChildren, family.ActionWrite, now).Outcome)
	assert.Equal(t, family.Deny, family.Decide(s, family.Subject{Kind: family.SubjectChild, ChildID: child, Scope: family.ScopeSleep}, family.DomainChildren, family.ActionWrite, now).Outcome)
	assert.Equal(t, family.Deny, family.Decide(s, family.Subject{Kind: family.SubjectChild, ChildID: child + 1, Scope: family.ScopeFeeding}, family.DomainChildren, family.ActionRead, now).Outcome)
}

func TestSnapshot_AllOwnersOneStatement(t *testing.T) {
	e := newEnv(t)
	member := e.user("male", "")
	var owners []uint64
	for range 5 {
		o := e.user("female", "cycle")
		owners = append(owners, o)
		e.link(o, member, map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView, family.DomainAppointments: family.LevelView})
	}
	cdb := &countingDB{DB: e.db}
	l := family.NewLoader(cdb, func() time.Time { return now })
	all, err := l.ForMember(context.Background(), member)
	require.NoError(t, err)
	assert.EqualValues(t, 1, cdb.n.Load())
	require.Len(t, all, 5)
	for i, s := range all {
		assert.Equal(t, owners[i], s.OwnerID)
		assert.Equal(t, family.Allow, family.Decide(s, family.Subject{}, family.DomainAppointments, family.ActionRead, now).Outcome)
	}
	// Every pair of the member is now served from memory.
	for _, o := range owners {
		_, err := l.Pair(context.Background(), o, member)
		require.NoError(t, err)
	}
	assert.EqualValues(t, 1, cdb.n.Load())
}

// A mode change nobody told the family layer about (no hook ran): the next snapshot resolves the new mode, bumps the
// epoch and the owner-data grants pause — the stale grant can only deny.
func TestSyncMode_SelfHealsToDeny(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	e.link(owner, member, map[family.Domain]family.Level{family.DomainPregnancy: family.LevelView,
		family.DomainMeds: family.LevelView, family.DomainUrgentAlert: family.LevelView})
	assert.Equal(t, 1, e.count(`SELECT mode_epoch FROM family_owner_state WHERE owner_id = ?`, owner))
	assert.Equal(t, family.Allow, decide(t, e.db, owner, member, family.DomainMeds, family.ActionRead).Outcome)
	assert.Equal(t, family.ReasonModeInactive, decide(t, e.db, owner, member, family.DomainPregnancy, family.ActionRead).Reason)

	// She becomes pregnant behind the family layer's back.
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at) VALUES (?, 1, 'lmp', '2026-09-01', NOW(), NOW())`, owner)
	for _, d := range []family.Domain{family.DomainPregnancy, family.DomainMeds} {
		got := decide(t, e.db, owner, member, d, family.ActionRead)
		assert.Equal(t, family.Decision{Outcome: family.Deny, Reason: family.ReasonModeInactive}, got, d.String())
	}
	assert.Equal(t, family.Allow, decide(t, e.db, owner, member, family.DomainUrgentAlert, family.ActionRead).Outcome, "urgent alert stays on (F-23)")
	var mode string
	var epoch, review int
	require.NoError(t, e.db.QueryRow(`SELECT mode, mode_epoch, review_pending FROM family_owner_state WHERE owner_id = ?`, owner).Scan(&mode, &epoch, &review))
	assert.Equal(t, []any{"pregnancy", 2, 1}, []any{mode, epoch, review})

	// The owner re-approves: the grant is stamped with the new epoch and the pregnancy becomes visible.
	link := uint64(e.count(`SELECT id FROM companions WHERE owner_id = ?`, owner)) //nolint:gosec // test id
	out := e.apply(link, owner, member, map[family.Domain]family.Level{family.DomainPregnancy: family.LevelView})
	assert.EqualValues(t, 2, out.Epoch)
	assert.Equal(t, family.Allow, decide(t, e.db, owner, member, family.DomainPregnancy, family.ActionRead).Outcome)

	// SyncMode itself is idempotent.
	st, err := family.SyncMode(context.Background(), e.db, owner, now)
	require.NoError(t, err)
	assert.Equal(t, family.OwnerState{Mode: "pregnancy", Epoch: 2, ReviewPending: true}, st, "the review card is cleared by T-M11-07")
}

func TestGrants_ApplyWritesDiffAndConsent(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	link := e.link(owner, member, map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView, family.DomainMeds: family.LevelNudge})
	out := e.apply(link, owner, member, map[family.Domain]family.Level{family.DomainMeds: family.LevelWrite, family.DomainAppointments: family.LevelView})
	assert.Equal(t, map[string][2]string{"cycle_phase": {"view", "off"}, "meds": {"nudge", "write"}, "appointments": {"off", "view"}}, out.Diff)

	var diff json.RawMessage
	require.NoError(t, e.db.QueryRow(`SELECT diff_json FROM companion_audit_logs WHERE owner_id = ? AND action = 'grants_changed' ORDER BY id DESC LIMIT 1`, owner).Scan(&diff))
	var got map[string][2]string
	require.NoError(t, json.Unmarshal(diff, &got))
	assert.Equal(t, out.Diff, got, "diff_json {domain: [old, new]} (FAM-L5)")

	var scope json.RawMessage
	var action, source string
	var version int
	require.NoError(t, e.db.QueryRow(`SELECT action, source, version, scope FROM consent_events WHERE user_id = ? AND consent = 'family_sharing' ORDER BY id DESC LIMIT 1`, owner).
		Scan(&action, &source, &version, &scope))
	assert.Equal(t, []any{"grant", "family", 1}, []any{action, source, version})
	var sc map[string]any
	require.NoError(t, json.Unmarshal(scope, &sc))
	assert.EqualValues(t, link, sc["link_id"])
	assert.EqualValues(t, member, sc["member_id"])
	assert.Equal(t, "partner", sc["role"])
	assert.Equal(t, map[string]any{"meds": "write", "appointments": "view"}, sc["domains"])
	assert.Contains(t, sc, "expires_at")
	assert.Equal(t, 2, e.count(`SELECT COUNT(*) FROM consent_events WHERE user_id = ? AND consent = 'family_sharing'`, owner))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND (mode_epoch IS NULL OR section = 'cycle_phase')`, link))

	// Revoking every domain is a consent revoke.
	e.apply(link, owner, member, map[family.Domain]family.Level{})
	require.NoError(t, e.db.QueryRow(`SELECT action FROM consent_events WHERE user_id = ? AND consent = 'family_sharing' ORDER BY id DESC LIMIT 1`, owner).Scan(&action))
	assert.Equal(t, "revoke", action)

	// In the caller's transaction: a rollback leaves nothing behind.
	before := e.count(`SELECT COUNT(*) FROM consent_events`)
	tx, err := e.db.Begin()
	require.NoError(t, err)
	_, err = family.Grants{}.Apply(context.Background(), tx, family.Change{LinkID: link, OwnerID: owner, MemberID: member, ActorID: owner,
		Role: family.RolePartner, Set: map[family.Domain]family.Level{family.DomainVitals: family.LevelView}, Now: now})
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	assert.Equal(t, before, e.count(`SELECT COUNT(*) FROM consent_events`))
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ?`, link))
}

func TestGrants_RejectsAboveMaxLevel(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	link := e.link(owner, member, map[family.Domain]family.Level{family.DomainMeds: family.LevelView})
	try := func(owner uint64, role family.Role, set map[family.Domain]family.Level) error {
		tx, err := e.db.Begin()
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()
		_, err = family.Grants{}.Apply(context.Background(), tx, family.Change{LinkID: link, OwnerID: owner, MemberID: member,
			ActorID: owner, Role: role, Set: set, Now: now})
		return err
	}
	for name, c := range map[string]struct {
		role family.Role
		set  map[family.Domain]family.Level
	}{
		"cycle phase above view":    {family.RolePartner, map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelWrite}},
		"urgent alert write":        {family.RolePartner, map[family.Domain]family.Level{family.DomainUrgentAlert: family.LevelNudge}},
		"friend on meds":            {family.RoleFriend, map[family.Domain]family.Level{family.DomainMeds: family.LevelView}},
		"carer appointments write":  {family.RoleCarer, map[family.Domain]family.Level{family.DomainAppointments: family.LevelWrite}},
		"other parent on cycle":     {family.RoleOtherParent, map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView}},
		"teen parent off the card":  {family.RoleTeenParent, map[family.Domain]family.Level{family.DomainMeds: family.LevelView}},
		"never-shared symptoms":     {family.RolePartner, map[family.Domain]family.Level{family.NeverSymptoms: family.LevelView}},
		"unknown role":              {family.RoleNone, map[family.Domain]family.Level{family.DomainMeds: family.LevelView}},
		"ttc switch for the friend": {family.RoleFriend, map[family.Domain]family.Level{family.DomainTTCFertility: family.LevelView}},
	} {
		err := try(owner, c.role, c.set)
		require.ErrorIs(t, err, family.ErrLevelNotAllowed, name)
	}
	// F-20: a male owner may grant only meds, appointments and household.
	male := e.user("male", "")
	require.ErrorIs(t, try(male, family.RolePartner, map[family.Domain]family.Level{family.DomainVitals: family.LevelView}), family.ErrLevelNotAllowed)
	require.NoError(t, try(male, family.RolePartner, map[family.Domain]family.Level{family.DomainMeds: family.LevelWrite, family.DomainHousehold: family.LevelWrite}))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ?`, link), "nothing written on refusal")
}

func TestRecheckTx_RevokeWins(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	link := e.link(owner, member, map[family.Domain]family.Level{family.DomainMeds: family.LevelWrite})
	ctx := context.Background()
	// The request's first check passed (snapshot before the revoke) …
	assert.Equal(t, family.Propose, decide(t, e.db, owner, member, family.DomainMeds, family.ActionWrite).Outcome)
	tx, err := e.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	var x int
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&x)) // the transaction's read view starts here
	// … then the owner revokes in another transaction.
	e.exec(`UPDATE companions SET status = 'revoked', revoked_by = 'owner' WHERE id = ?`, link)
	e.exec(`DELETE FROM companion_grants WHERE companion_id = ?`, link)
	s, err := family.RecheckTx(ctx, tx, owner, member)
	require.NoError(t, err)
	assert.Equal(t, family.Decision{Outcome: family.Deny, Reason: family.ReasonNoLink}, family.Decide(s, family.Subject{}, family.DomainMeds, family.ActionWrite, now))
}

func TestRevoke_DeniesImmediately(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	link := e.link(owner, member, map[family.Domain]family.Level{family.DomainAppointments: family.LevelView})
	assert.Equal(t, family.Allow, decide(t, e.db, owner, member, family.DomainAppointments, family.ActionRead).Outcome)
	e.exec(`UPDATE companions SET status = 'revoked', revoked_by = 'owner' WHERE id = ?`, link)
	assert.Equal(t, family.Decision{Outcome: family.Deny, Reason: family.ReasonNoLink}, decide(t, e.db, owner, member, family.DomainAppointments, family.ActionRead))

	// Expiry needs no sweep either.
	other := e.user("male", "")
	link2 := e.link(owner, other, map[family.Domain]family.Level{family.DomainAppointments: family.LevelView})
	e.exec(`UPDATE companions SET expires_at = ? WHERE id = ?`, now.Add(-time.Minute), link2)
	assert.Equal(t, family.Decision{Outcome: family.Deny, Reason: family.ReasonExpired}, decide(t, e.db, owner, other, family.DomainAppointments, family.ActionRead))
}

func TestBlockedOwner_Denied(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	e.link(owner, member, map[family.Domain]family.Level{family.DomainUrgentAlert: family.LevelView})
	e.exec(`UPDATE users SET blocked_at = NOW() WHERE id = ?`, owner)
	got := decide(t, e.db, owner, member, family.DomainUrgentAlert, family.ActionRead)
	assert.Equal(t, family.Decision{Outcome: family.Deny, Reason: family.ReasonBlocked}, got)
	assert.True(t, got.Reason.NotFound(), "404, no existence oracle (F-21)")
}

func (e *env) applyAs(link, owner, member uint64, reason string, set map[family.Domain]family.Level) {
	e.t.Helper()
	tx, err := e.db.Begin()
	require.NoError(e.t, err)
	_, err = family.Grants{}.Apply(context.Background(), tx, family.Change{LinkID: link, OwnerID: owner, MemberID: member,
		ActorID: owner, Role: family.RolePartner, Set: set, Reason: reason, Now: now})
	require.NoError(e.t, err)
	require.NoError(e.t, tx.Commit())
}

// FAM-H1: a stored teen life mode beats an active pregnancy — the partner's pregnancy, meds and appointments grants
// stay suspended (enums.ResolveLifeMode alone would resolve "pregnancy" and let them through).
func TestTeenOwnerWithPregnancy_StaysSuspended(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	e.link(owner, member, map[family.Domain]family.Level{family.DomainPregnancy: family.LevelView,
		family.DomainMeds: family.LevelWrite, family.DomainAppointments: family.LevelWrite})
	e.exec(`UPDATE user_life_profiles SET life_mode = 'teen' WHERE user_id = ?`, owner)
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-08-01', NOW(), NOW())`, owner)

	for _, c := range []struct {
		d family.Domain
		a family.Action
	}{
		{family.DomainPregnancy, family.ActionRead},
		{family.DomainMeds, family.ActionRead}, {family.DomainMeds, family.ActionWrite},
		{family.DomainAppointments, family.ActionRead}, {family.DomainAppointments, family.ActionWrite},
	} {
		assert.Equal(t, family.Decision{Outcome: family.Deny, Reason: family.ReasonModeInactive},
			decide(t, e.db, owner, member, c.d, c.a), "%s %v", c.d, c.a)
	}
}

// L2: Apply keeps a grant's stored epoch when its level does not change, unless it is the owner's own PUT (which
// re-approves every domain it includes at a level > off). M2 relies on it: accept (ChangeAccept) never re-stamps.
func TestApply_UnchangedLevelKeepsEpochUnlessOwnerReapproves(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	set := map[family.Domain]family.Level{family.DomainMeds: family.LevelWrite, family.DomainAppointments: family.LevelView}
	link := e.link(owner, member, set)
	require.Equal(t, family.Propose, decide(t, e.db, owner, member, family.DomainMeds, family.ActionWrite).Outcome)

	e.exec(`UPDATE user_life_profiles SET life_mode = 'menopause' WHERE user_id = ?`, owner) // epoch bump on next load
	paused := family.Decision{Outcome: family.Deny, Reason: family.ReasonModeInactive}
	require.Equal(t, paused, decide(t, e.db, owner, member, family.DomainMeds, family.ActionWrite))

	for _, reason := range []string{family.ChangeAccept, family.ChangeShareRequest} {
		e.applyAs(link, owner, member, reason, set)
		assert.Equal(t, paused, decide(t, e.db, owner, member, family.DomainMeds, family.ActionWrite), reason)
		assert.Equal(t, paused, decide(t, e.db, owner, member, family.DomainAppointments, family.ActionRead), reason)
	}
	// A changed level is a fresh owner choice even outside a PUT: stamped with the current epoch.
	e.applyAs(link, owner, member, family.ChangeShareRequest, map[family.Domain]family.Level{
		family.DomainMeds: family.LevelWrite, family.DomainAppointments: family.LevelWrite})
	assert.Equal(t, family.Propose, decide(t, e.db, owner, member, family.DomainAppointments, family.ActionWrite).Outcome)
	assert.Equal(t, paused, decide(t, e.db, owner, member, family.DomainMeds, family.ActionWrite), "unchanged meds stays paused")

	// The owner's PUT that includes the paused domain at a level > off re-approves it.
	e.applyAs(link, owner, member, family.ChangeOwnerEdit, map[family.Domain]family.Level{
		family.DomainMeds: family.LevelWrite, family.DomainAppointments: family.LevelWrite})
	assert.Equal(t, family.Propose, decide(t, e.db, owner, member, family.DomainMeds, family.ActionWrite).Outcome)
}

// L3: an unstamped row is read as a pre-00110 legacy grant (epoch 1) only when it cannot be a v6 row — legacy code
// cycle, or meds / appointments at the legacy level edit. A NULL-epoch meds / appointments row at a v6 level is
// inactive (fail closed).
func TestUnstampedRows_OnlyLegacyShapesAreLive(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	link := e.exec(`INSERT INTO companions (owner_id, companion_user_id, type, role, template, status, invited_at, accepted_at, created_at, updated_at)
		VALUES (?, ?, 'partner', 'partner', 'custom', 'active', NOW(), NOW(), NOW(), NOW())`, owner, member)
	for _, g := range [][2]string{{"cycle", "view"}, {"meds", "view"}, {"appointments", "edit"}} {
		e.exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`, link, g[0], g[1])
	}
	assert.Equal(t, family.Allow, decide(t, e.db, owner, member, family.DomainCyclePhase, family.ActionRead).Outcome, "legacy cycle")
	assert.Equal(t, family.Propose, decide(t, e.db, owner, member, family.DomainAppointments, family.ActionWrite).Outcome, "legacy appointments edit")
	assert.Equal(t, family.Decision{Outcome: family.Deny, Reason: family.ReasonModeInactive},
		decide(t, e.db, owner, member, family.DomainMeds, family.ActionRead), "ambiguous NULL-epoch meds view")
}
