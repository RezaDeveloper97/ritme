package sharerequests_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/family/sharerequests"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

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
	t   *testing.T
	db  *sql.DB
	clk *tick
	svc *sharerequests.Service
	n   int
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	clk := &tick{t: time.Date(2026, 10, 2, 10, 0, 0, 0, civildate.Tehran)}
	return &env{t: t, db: db, clk: clk, svc: sharerequests.NewService(db, clk)}
}

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

func (e *env) user(gender string) uint64 {
	e.t.Helper()
	e.n++
	id := e.exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, NOW(), NOW())`, fmt.Sprintf("0912888%04d", e.n))
	e.exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, ?, 'cycle', NOW(), NOW())`, id, gender)
	return uint64(id) //nolint:gosec // test id
}

// link is an active partner link owner → member with set applied through Grants.Apply.
func (e *env) link(owner, member uint64, set map[family.Domain]family.Level) uint64 {
	e.t.Helper()
	id := uint64(e.exec(`INSERT INTO companions (owner_id, companion_user_id, type, role, template, status, invited_at, accepted_at, created_at, updated_at)
		VALUES (?, ?, 'partner', 'partner', 'custom', 'active', NOW(), NOW(), NOW(), NOW())`, owner, member)) //nolint:gosec // test id
	tx, err := e.db.Begin()
	require.NoError(e.t, err)
	_, err = family.Grants{}.Apply(context.Background(), tx, family.Change{LinkID: id, OwnerID: owner, MemberID: member,
		ActorID: owner, Role: family.RolePartner, Set: set, Reason: family.ChangeOwnerEdit, Now: e.clk.Now()})
	require.NoError(e.t, err)
	require.NoError(e.t, tx.Commit())
	return id
}

func (e *env) grants(link uint64) map[string]string {
	e.t.Helper()
	rows, err := e.db.Query(`SELECT section, level FROM companion_grants WHERE companion_id = ?`, link)
	require.NoError(e.t, err)
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var s, l string
		require.NoError(e.t, rows.Scan(&s, &l))
		out[s] = l
	}
	require.NoError(e.t, rows.Err())
	return out
}

func ask(d family.Domain, l family.Level) map[family.Domain]family.Level {
	return map[family.Domain]family.Level{d: l}
}

func TestShareRequest_NoChangeUntilOwnerApproves(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female"), e.user("male")
	link := e.link(owner, member, ask(family.DomainCyclePhase, family.LevelView))
	consents := func() int {
		return e.count(`SELECT COUNT(*) FROM consent_events WHERE user_id = ? AND consent = 'family_sharing'`, owner)
	}
	before := consents()

	r, err := e.svc.Create(ctx, member, owner, map[family.Domain]family.Level{family.DomainMeds: family.LevelNudge, family.DomainCyclePhase: family.LevelView})
	require.NoError(t, err)
	assert.Equal(t, sharerequests.StatusPending, r.Status)
	assert.Equal(t, map[string]string{"cycle_phase": "view"}, e.grants(link), "nothing changes on asking")
	assert.Equal(t, before, consents())

	in, err := e.svc.List(ctx, owner, sharerequests.BoxIn)
	require.NoError(t, err)
	require.Len(t, in, 1)
	out, err := e.svc.List(ctx, member, sharerequests.BoxOut)
	require.NoError(t, err)
	require.Len(t, out, 1)
	in, err = e.svc.List(ctx, member, sharerequests.BoxIn)
	require.NoError(t, err)
	assert.Empty(t, in)

	got, err := e.svc.Approve(ctx, owner, r.ID)
	require.NoError(t, err)
	assert.Equal(t, link, got)
	assert.Equal(t, map[string]string{"cycle_phase": "view", "meds": "nudge"}, e.grants(link))
	assert.Equal(t, before+1, consents(), "approval is a consent event")
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM consent_events WHERE user_id = ? AND JSON_UNQUOTE(JSON_EXTRACT(scope, '$.reason')) = ?`,
		owner, family.ChangeShareRequest))
	_, err = e.svc.Approve(ctx, owner, r.ID)
	require.ErrorIs(t, err, sharerequests.ErrNotFound, "decided once")

	// A request to the member's own data (no reverse link yet) creates B→A on approval, owned by the member.
	r2, err := e.svc.Create(ctx, owner, member, ask(family.DomainAppointments, family.LevelView))
	require.NoError(t, err)
	require.NoError(t, e.svc.Decline(ctx, member, r2.ID))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companions WHERE owner_id = ?`, member), "declined: nothing created")
	e.clk.add(sharerequests.Window + time.Hour)
	r3, err := e.svc.Create(ctx, owner, member, ask(family.DomainAppointments, family.LevelView))
	require.NoError(t, err)
	rev, err := e.svc.Approve(ctx, member, r3.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companions WHERE id = ? AND owner_id = ? AND companion_user_id = ? AND status = 'active'`, rev, member, owner))
	assert.Equal(t, map[string]string{"appointments": "view"}, e.grants(rev))
}

func TestShareRequest_RateLimit(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female"), e.user("male")
	e.link(owner, member, ask(family.DomainCyclePhase, family.LevelView))
	r, err := e.svc.Create(ctx, member, owner, ask(family.DomainMeds, family.LevelView))
	require.NoError(t, err)
	_, err = e.svc.Create(ctx, member, owner, ask(family.DomainMeds, family.LevelNudge))
	require.ErrorIs(t, err, sharerequests.ErrRateLimited, "one per domain per pair per 7 days")
	require.NoError(t, e.svc.Decline(ctx, owner, r.ID))
	_, err = e.svc.Create(ctx, member, owner, ask(family.DomainMeds, family.LevelView))
	require.ErrorIs(t, err, sharerequests.ErrRateLimited, "a decline does not reset the window")
	_, err = e.svc.Create(ctx, member, owner, ask(family.DomainAppointments, family.LevelView))
	require.NoError(t, err, "another domain is its own budget")

	e.clk.add(sharerequests.Window)
	_, err = e.svc.Create(ctx, member, owner, ask(family.DomainMeds, family.LevelView))
	require.NoError(t, err, "after 7 days it may be asked again")
}

func TestShareRequest_IDOR(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member, stranger := e.user("female"), e.user("male"), e.user("female")
	link := e.link(owner, member, ask(family.DomainCyclePhase, family.LevelView))

	_, err := e.svc.Create(ctx, stranger, owner, ask(family.DomainMeds, family.LevelView))
	require.ErrorIs(t, err, sharerequests.ErrNotFound, "no link, no request (no account oracle)")
	_, err = e.svc.Create(ctx, member, member, ask(family.DomainMeds, family.LevelView))
	require.ErrorIs(t, err, sharerequests.ErrSelf)
	_, err = e.svc.Create(ctx, member, owner, ask(family.DomainVitals, family.LevelWrite))
	require.ErrorIs(t, err, family.ErrLevelNotAllowed, "the registry's caps apply to requests")

	r, err := e.svc.Create(ctx, member, owner, ask(family.DomainMeds, family.LevelView))
	require.NoError(t, err)
	// The requester cannot approve her own request, nor can a stranger approve or decline it.
	for _, who := range []uint64{member, stranger} {
		_, err = e.svc.Approve(ctx, who, r.ID)
		require.ErrorIs(t, err, sharerequests.ErrNotFound)
		require.ErrorIs(t, e.svc.Decline(ctx, who, r.ID), sharerequests.ErrNotFound)
	}
	list, err := e.svc.List(ctx, stranger, sharerequests.BoxIn)
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.Equal(t, map[string]string{"cycle_phase": "view"}, e.grants(link))

	// Expired requests cannot be approved.
	e.clk.add(sharerequests.Window + time.Minute)
	_, err = e.svc.Approve(ctx, owner, r.ID)
	require.ErrorIs(t, err, sharerequests.ErrNotFound)
	out, err := e.svc.List(ctx, member, sharerequests.BoxOut)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, sharerequests.StatusExpired, out[0].Status)
	assert.Equal(t, map[string]string{"cycle_phase": "view"}, e.grants(link))
}

// Review L3: the approve / decline read is scoped to the data owner in the query itself.
func TestShareRequest_ReadScopedToOwner(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female"), e.user("male")
	e.link(owner, member, ask(family.DomainCyclePhase, family.LevelView))
	r, err := e.svc.Create(ctx, member, owner, ask(family.DomainMeds, family.LevelView))
	require.NoError(t, err)
	q := store.New(e.db)
	_, err = q.GetFamilyShareRequest(ctx, store.GetFamilyShareRequestParams{ID: r.ID, ToUserID: member, Now: e.clk.Now()})
	require.ErrorIs(t, err, sql.ErrNoRows, "the requester's id does not read the row")
	got, err := q.GetFamilyShareRequest(ctx, store.GetFamilyShareRequestParams{ID: r.ID, ToUserID: owner, Now: e.clk.Now()})
	require.NoError(t, err)
	assert.Equal(t, r.ID, got.ID)
}

// Review L5: approving a request that would create a link owned by a teen-mode account is refused (CB-TEEN-01).
func TestShareRequest_TeenCannotOwnNewLink(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	adult, teen := e.user("female"), e.user("female")
	e.link(adult, teen, ask(family.DomainCyclePhase, family.LevelView)) // adult → teen (partner)
	r, err := e.svc.Create(ctx, adult, teen, ask(family.DomainAppointments, family.LevelView))
	require.NoError(t, err)
	_, err = e.db.Exec(`UPDATE user_life_profiles SET life_mode = 'teen' WHERE user_id = ?`, teen)
	require.NoError(t, err)
	_, err = e.svc.Approve(ctx, teen, r.ID)
	require.ErrorIs(t, err, sharerequests.ErrTeenParentOnly)
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companions WHERE owner_id = ?`, teen))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM family_share_requests WHERE id = ? AND status = 'pending'`, r.ID))
}
