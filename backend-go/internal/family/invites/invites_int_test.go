package invites_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/family/invites"
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

var pepper = []byte("test-pepper-0123456789abcdefghijklmn")

type env struct {
	t   *testing.T
	db  *sql.DB
	clk *tick
	svc *invites.Service
	n   int
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	clk := &tick{t: time.Date(2026, 10, 2, 10, 0, 0, 0, civildate.Tehran)}
	return &env{t: t, db: db, clk: clk, svc: invites.NewService(db, clk, invites.Options{Pepper: pepper})}
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

// user makes an account with a life profile; mobile "" draws one.
func (e *env) user(gender, mode, mobile string) uint64 {
	e.t.Helper()
	e.n++
	if mobile == "" {
		mobile = fmt.Sprintf("0912777%04d", e.n)
	}
	id := e.exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, fmt.Sprintf("Sara Test%d", e.n), mobile)
	var lm any
	if mode != "" {
		lm = mode
	}
	e.exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`, id, gender, lm)
	return uint64(id) //nolint:gosec // test id
}

func (e *env) status(link uint64) (string, string) {
	e.t.Helper()
	var st string
	var by sql.NullString
	require.NoError(e.t, e.db.QueryRow(`SELECT status, revoked_by FROM companions WHERE id = ?`, link).Scan(&st, &by))
	return st, by.String
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

func (e *env) create(owner uint64, in invites.CreateInput) invites.Created {
	e.t.Helper()
	if in.ConsentVersion == 0 {
		in.ConsentVersion = invites.ConsentVersion
	}
	out, err := e.svc.Create(context.Background(), owner, in)
	require.NoError(e.t, err)
	return out
}

// partnerBuddy is a partner invite with view / nudge rows only (no write level, so a code accept activates at once).
func partnerBuddy() invites.CreateInput {
	return invites.CreateInput{Role: family.RolePartner, Template: family.TemplateMedsAppointments,
		Edits: map[family.Domain]family.Level{family.DomainAppointments: family.LevelView}}
}

func TestInvite_AcceptWithoutMutual(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "cycle", ""), e.user("male", "", "")
	in := partnerBuddy()
	in.Mutual = []family.Domain{family.DomainAppointments}
	inv := e.create(owner, in)
	assert.Equal(t, invites.StatusInvited, inv.Status)
	assert.Equal(t, []family.Domain{family.DomainAppointments}, inv.Mutual)

	p, err := e.svc.Preview(ctx, member, inv.Code)
	require.NoError(t, err)
	assert.Equal(t, []family.Domain{family.DomainAppointments}, p.Mutual)
	assert.False(t, p.OwnerConfirmation)

	a, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code})
	require.NoError(t, err)
	assert.Equal(t, invites.StatusActive, a.Status)
	assert.Zero(t, a.MutualLinkID)
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companions WHERE owner_id = ?`, member), "no reverse link without a mutual answer")
	st, _ := e.status(inv.LinkID)
	assert.Equal(t, invites.StatusActive, st)
	assert.Equal(t, map[string]string{"appointments": "view", "meds": "nudge"}, e.grants(inv.LinkID))
}

func TestFamily_MutualLinksIndependent(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "cycle", ""), e.user("male", "", "")
	in := partnerBuddy()
	in.Mutual = []family.Domain{family.DomainAppointments, family.DomainMeds}
	inv := e.create(owner, in)

	// Outside the request: refused and nothing written.
	_, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code,
		Mutual: map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView}})
	require.ErrorIs(t, err, invites.ErrMutualNotRequested)

	a, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code,
		Mutual: map[family.Domain]family.Level{family.DomainAppointments: family.LevelView}})
	require.NoError(t, err)
	require.NotZero(t, a.MutualLinkID)

	var revOwner, revMember uint64
	require.NoError(t, e.db.QueryRow(`SELECT owner_id, companion_user_id FROM companions WHERE id = ?`, a.MutualLinkID).Scan(&revOwner, &revMember))
	assert.Equal(t, member, revOwner)
	assert.Equal(t, owner, revMember)
	assert.Equal(t, map[string]string{"appointments": "view"}, e.grants(a.MutualLinkID))
	// B's own consent row for the reverse link.
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM consent_events WHERE user_id = ? AND consent = 'family_sharing'
		AND JSON_EXTRACT(scope, '$.link_id') = ?`, member, a.MutualLinkID))
	revActivity := e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE companion_id = ?`, a.MutualLinkID)
	require.Positive(t, revActivity)

	// A revokes A→B: B→A, its grants and its activity stay as they were.
	comp := companion.NewService(e.db, e.clk, companion.Options{CodePepper: pepper})
	require.NoError(t, comp.Revoke(ctx, owner, inv.LinkID))
	st, _ := e.status(inv.LinkID)
	assert.Equal(t, invites.StatusRevoked, st)
	st, _ = e.status(a.MutualLinkID)
	assert.Equal(t, invites.StatusActive, st)
	assert.Equal(t, map[string]string{"appointments": "view"}, e.grants(a.MutualLinkID))
	assert.Equal(t, revActivity, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE companion_id = ?`, a.MutualLinkID))
}

func TestInvite_PreviewGenericRefusals(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "cycle", ""), e.user("male", "", "")
	inv := e.create(owner, partnerBuddy())

	for name, err := range map[string]error{
		"unknown":   func() error { _, err := e.svc.Preview(ctx, member, "ZZZZZZ"); return err }(),
		"malformed": func() error { _, err := e.svc.Preview(ctx, member, "??"); return err }(),
		"self":      func() error { _, err := e.svc.Preview(ctx, owner, inv.Code); return err }(),
	} {
		assert.True(t, invites.IsRefusal(err), name)
	}
	// A preview writes nothing.
	_, err := e.svc.Preview(ctx, member, inv.Code)
	require.NoError(t, err)
	st, _ := e.status(inv.LinkID)
	assert.Equal(t, invites.StatusInvited, st)
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companion_invites WHERE companion_id = ? AND (used_at IS NOT NULL OR attempts > 0)`, inv.LinkID))

	e.clk.add(25 * time.Hour)
	_, err = e.svc.Preview(ctx, member, inv.Code)
	assert.True(t, invites.IsRefusal(err), "expired")
}

func TestInvite_DeclineMarksRevoked(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "cycle", ""), e.user("male", "", "")
	inv := e.create(owner, partnerBuddy())
	require.NoError(t, e.svc.Decline(ctx, member, inv.Code, 0))
	st, by := e.status(inv.LinkID)
	assert.Equal(t, invites.StatusRevoked, st)
	assert.Equal(t, invites.RevokedByDeclined, by)
	assert.Empty(t, e.grants(inv.LinkID))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE companion_id = ? AND action = ?`, inv.LinkID, invites.ActionDeclined))
	_, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code})
	assert.True(t, invites.IsRefusal(err), "a declined code is spent")
}

func TestSignup_BindsPhoneInviteAsPending(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user("female", "cycle", "")
	in := partnerBuddy()
	in.Phone = "09121112233"
	inv := e.create(owner, in)
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companion_invites WHERE companion_id = ? AND used_by_id IS NOT NULL`, inv.LinkID))

	// She signs up with that number afterwards: the hook binds the invite as a pending acceptance.
	member := e.user("male", "", "09121112233")
	require.NoError(t, e.svc.SignupHook()(ctx, member, "09121112233", e.clk.Now()))
	list, err := e.svc.Pending(ctx, member)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, inv.LinkID, list[0].LinkID)
	assert.Equal(t, family.RolePartner, list[0].Role)

	// Nobody else sees it; the owner's view is unchanged.
	other := e.user("male", "", "")
	list, err = e.svc.Pending(ctx, other)
	require.NoError(t, err)
	assert.Empty(t, list)
	_, err = e.svc.Accept(ctx, other, invites.AcceptInput{LinkID: inv.LinkID})
	require.Error(t, err)

	// Accepted without the code; the attempt lock never blocks the bound account (CMP-L8).
	_, err = e.db.Exec(`UPDATE companion_invites SET attempts = 99 WHERE companion_id = ?`, inv.LinkID)
	require.NoError(t, err)
	a, err := e.svc.Accept(ctx, member, invites.AcceptInput{LinkID: inv.LinkID})
	require.NoError(t, err)
	assert.Equal(t, invites.StatusActive, a.Status, "a phone invite never needs the owner's confirmation")

	// An existing account is bound at invite time.
	in.Phone = e.mobile(other)
	inv2 := e.create(owner, in)
	list, err = e.svc.Pending(ctx, other)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, inv2.LinkID, list[0].LinkID)
}

func (e *env) mobile(id uint64) string {
	e.t.Helper()
	var m string
	require.NoError(e.t, e.db.QueryRow(`SELECT mobile FROM users WHERE id = ?`, id).Scan(&m))
	return m
}

func TestInvite_PhoneTTL7Days(t *testing.T) {
	e := setup(t)
	owner := e.user("female", "cycle", "")
	code := e.create(owner, partnerBuddy())
	assert.Equal(t, e.clk.Now().Add(24*time.Hour), code.ExpiresAt)
	in := partnerBuddy()
	in.Phone = "09125556677"
	phone := e.create(owner, in)
	assert.Equal(t, e.clk.Now().Add(7*24*time.Hour), phone.ExpiresAt)

	member := e.user("male", "", "09125556677")
	e.clk.add(6 * 24 * time.Hour)
	_, err := e.svc.Accept(context.Background(), member, invites.AcceptInput{Code: phone.Code})
	require.NoError(t, err, "a phone invite still opens on day 6")
}

func TestInvite_HighImpactCodeNeedsOwnerConfirm(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "pregnancy", ""), e.user("male", "", "")
	inv := e.create(owner, invites.CreateInput{Role: family.RolePartner, Template: family.TemplateFullFamily})
	p, err := e.svc.Preview(ctx, member, inv.Code)
	require.NoError(t, err)
	assert.True(t, p.OwnerConfirmation)

	a, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code})
	require.NoError(t, err)
	assert.Equal(t, invites.StatusAwaitingOwner, a.Status)
	// The member sees nothing before the owner confirms.
	assert.False(t, decide(t, e, owner, member, family.DomainAppointments).Allowed())

	_, err = e.svc.Confirm(ctx, member, inv.LinkID)
	require.ErrorIs(t, err, invites.ErrNotFound, "only the owner confirms")
	c, err := e.svc.Confirm(ctx, owner, inv.LinkID)
	require.NoError(t, err)
	assert.Equal(t, invites.StatusActive, c.Status)
	assert.True(t, decide(t, e, owner, member, family.DomainAppointments).Allowed())
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE companion_id = ? AND action = ?`, inv.LinkID, invites.ActionConfirmed))

	// A plain code invite (no children write, urgent alert or vitals) activates at once.
	inv2 := e.create(owner, invites.CreateInput{Role: family.RoleFriend, Template: family.TemplateCycleBuddy})
	other := e.user("female", "", "")
	a, err = e.svc.Accept(ctx, other, invites.AcceptInput{Code: inv2.Code})
	require.NoError(t, err)
	assert.Equal(t, invites.StatusActive, a.Status)
}

func decide(t *testing.T, e *env, owner, member uint64, d family.Domain) family.Decision {
	t.Helper()
	now := e.clk.Now()
	s, err := family.NewLoader(e.db, func() time.Time { return now }).Pair(context.Background(), owner, member)
	require.NoError(t, err)
	return family.Decide(s, family.Subject{}, d, family.ActionRead, now)
}

func TestConsent_RecordedPerInviteAndChange(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "cycle", ""), e.user("male", "", "")
	consents := func() []string {
		rows, err := e.db.Query(`SELECT JSON_UNQUOTE(JSON_EXTRACT(scope, '$.reason')) FROM consent_events
			WHERE user_id = ? AND consent = 'family_sharing' ORDER BY id`, owner)
		require.NoError(t, err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var r sql.NullString
			require.NoError(t, rows.Scan(&r))
			out = append(out, r.String)
		}
		require.NoError(t, rows.Err())
		return out
	}
	_, err := e.svc.Create(ctx, owner, invites.CreateInput{Role: family.RolePartner, Template: family.TemplateCycleBuddy, ConsentVersion: 999})
	require.ErrorIs(t, err, invites.ErrConsentVersion, "a stale consent text is refused")
	assert.Empty(t, consents())

	inv := e.create(owner, partnerBuddy())
	assert.Equal(t, []string{family.ChangeInvite}, consents())
	var diff json.RawMessage
	require.NoError(t, e.db.QueryRow(`SELECT diff_json FROM companion_audit_logs WHERE companion_id = ? AND action = ?`, inv.LinkID, invites.ActionInvited).Scan(&diff))
	assert.JSONEq(t, fmt.Sprintf(`{"children":"off","channel":"code","consent_version":%d}`, invites.ConsentVersion), string(diff))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM consent_events WHERE user_id = ? AND consent = ? AND version <> ?`,
		owner, consent.FamilySharing, invites.ConsentVersion), "the invite's text version is the ledger's")

	_, err = e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code})
	require.NoError(t, err)
	assert.Equal(t, []string{family.ChangeInvite, family.ChangeAccept}, consents())
}

func TestCarerExpiry_SweepRevokesAndLogs(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "cycle", ""), e.user("female", "", "")
	carer := e.create(owner, invites.CreateInput{Role: family.RoleCarer, Template: family.TemplateMedsAppointments, DisplayName: "Mina"})
	assert.Equal(t, e.clk.Now().Add(family.CarerTimeBoxDays*24*time.Hour), carer.LinkExpiresAt, "carer default F-13")
	_, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: carer.Code})
	require.NoError(t, err)
	stale := e.create(owner, partnerBuddy())
	langs := func(context.Context) []string { return []string{"fa", "en"} }

	rep, err := e.svc.Sweep(ctx, e.clk.Now(), langs)
	require.NoError(t, err)
	assert.Equal(t, invites.SweepReport{}, rep, "nothing is due yet")

	e.clk.add(31 * 24 * time.Hour)
	rep, err = e.svc.Sweep(ctx, e.clk.Now(), langs)
	require.NoError(t, err)
	assert.Equal(t, invites.SweepReport{Invites: 1, Links: 1}, rep)
	for link, action := range map[uint64]string{carer.LinkID: invites.ActionLinkExpire, stale.LinkID: invites.ActionInviteExpired} {
		st, by := e.status(link)
		assert.Equal(t, invites.StatusRevoked, st)
		assert.Equal(t, invites.RevokedByExpired, by)
		assert.Empty(t, e.grants(link))
		assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE companion_id = ? AND action = ?`, link, action))
	}
	assert.Equal(t, 2, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND type = ?`, owner, invites.NotificationType))
	var title string
	require.NoError(t, e.db.QueryRow(`SELECT JSON_UNQUOTE(JSON_EXTRACT(title, '$.en')) FROM user_notifications
		WHERE user_id = ? AND JSON_EXTRACT(data, '$.link_id') = ?`, owner, carer.LinkID).Scan(&title))
	assert.Equal(t, "Mina's access has ended", title)
	assert.False(t, decide(t, e, owner, member, family.DomainMeds).Allowed())

	rep, err = e.svc.Sweep(ctx, e.clk.Now(), langs)
	require.NoError(t, err)
	assert.Equal(t, invites.SweepReport{}, rep, "idempotent")
}

func TestInvite_CancelAnyTime(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "cycle", ""), e.user("male", "", "")
	inv := e.create(owner, partnerBuddy())
	require.ErrorIs(t, e.svc.Cancel(ctx, member, inv.LinkID), invites.ErrNotFound)
	require.NoError(t, e.svc.Cancel(ctx, owner, inv.LinkID))
	st, by := e.status(inv.LinkID)
	assert.Equal(t, invites.StatusRevoked, st)
	assert.Equal(t, invites.RevokedByOwner, by)
	_, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code})
	assert.True(t, invites.IsRefusal(err))
}

// memGate is an in-memory invites.Gate.
type memGate struct {
	mu   sync.Mutex
	hits map[string]int
}

func (g *memGate) Allow(_ context.Context, key string, maxHits int, _ time.Duration) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.hits[key]++
	return g.hits[key] <= maxHits, nil
}

func TestInvite_PhoneMismatchCountsOncePerAccount(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	e.svc = invites.NewService(e.db, e.clk, invites.Options{Pepper: pepper, Mismatch: &memGate{hits: map[string]int{}}})
	owner := e.user("female", "cycle", "")
	in := partnerBuddy()
	in.Phone = "09129998877"
	inv := e.create(owner, in)
	stranger := e.user("male", "", "")
	for range 10 {
		_, err := e.svc.Accept(ctx, stranger, invites.AcceptInput{Code: inv.Code})
		require.True(t, invites.IsRefusal(err))
	}
	assert.Equal(t, 1, e.count(`SELECT attempts FROM companion_invites WHERE companion_id = ?`, inv.LinkID), "one stranger, one attempt")
	member := e.user("male", "", "09129998877")
	_, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code})
	require.NoError(t, err)
}

func TestInvite_ConstantTimeFloor(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	e.svc = invites.NewService(e.db, e.clk, invites.Options{Pepper: pepper, MinDuration: 150 * time.Millisecond})
	owner, member := e.user("female", "cycle", ""), e.user("male", "", "")
	inv := e.create(owner, partnerBuddy())
	for name, fn := range map[string]func() error{
		"preview unknown":   func() error { _, err := e.svc.Preview(ctx, member, "ZZZZZZ"); return err },
		"preview malformed": func() error { _, err := e.svc.Preview(ctx, member, "x"); return err },
		"accept unknown":    func() error { _, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: "ZZZZZZ"}); return err },
		"preview valid":     func() error { _, err := e.svc.Preview(ctx, member, inv.Code); return err },
		"accept valid":      func() error { _, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code}); return err },
	} {
		start := time.Now()
		_ = fn()
		assert.GreaterOrEqual(t, time.Since(start), 150*time.Millisecond, name)
	}
}

// FAM-M5 (review M1): any write level makes a code-only invite wait for the owner; a phone-bound one does not.
func TestInvite_AnyWriteCodeNeedsOwner(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member, other := e.user("female", "cycle", ""), e.user("male", "", ""), e.user("male", "", "09124443322")
	write := invites.CreateInput{Role: family.RolePartner, Template: family.TemplateMedsAppointments} // appointments: write
	inv := e.create(owner, write)
	p, err := e.svc.Preview(ctx, member, inv.Code)
	require.NoError(t, err)
	assert.True(t, p.OwnerConfirmation)
	a, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code})
	require.NoError(t, err)
	assert.Equal(t, invites.StatusAwaitingOwner, a.Status)
	assert.False(t, decide(t, e, owner, member, family.DomainAppointments).Allowed())

	write.Phone = "09124443322"
	phone := e.create(owner, write)
	a, err = e.svc.Accept(ctx, other, invites.AcceptInput{Code: phone.Code})
	require.NoError(t, err)
	assert.Equal(t, invites.StatusActive, a.Status, "the bound number is the owner's choice of person")
}

// Review M2: an awaiting_owner link is confirmable for ConfirmTTL only; then the sweep revokes it (expired) with an
// activity row and an owner notice.
func TestInvite_AwaitingOwnerExpires(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, member := e.user("female", "pregnancy", ""), e.user("male", "", "")
	inv := e.create(owner, invites.CreateInput{Role: family.RolePartner, Template: family.TemplateFullFamily, DisplayName: "Ali"})
	a, err := e.svc.Accept(ctx, member, invites.AcceptInput{Code: inv.Code})
	require.NoError(t, err)
	require.Equal(t, invites.StatusAwaitingOwner, a.Status)
	require.NoError(t, e.svc.NotifyOwner(ctx, a, []string{"en"}))
	var title string
	require.NoError(t, e.db.QueryRow(`SELECT JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) FROM user_notifications WHERE user_id = ?`, owner).Scan(&title))
	assert.Equal(t, invites.NoticeAwaitingOwner, title, "the owner is asked to confirm, not told it is done")

	langs := func(context.Context) []string { return []string{"en"} }
	rep, err := e.svc.Sweep(ctx, e.clk.Now(), langs)
	require.NoError(t, err)
	assert.Zero(t, rep.Awaiting, "inside the window")

	e.clk.add(invites.DefaultConfirmTTL + time.Minute)
	_, err = e.svc.Confirm(ctx, owner, inv.LinkID)
	require.ErrorIs(t, err, invites.ErrNotFound, "too late to confirm")
	rep, err = e.svc.Sweep(ctx, e.clk.Now(), langs)
	require.NoError(t, err)
	assert.Equal(t, invites.SweepReport{Awaiting: 1}, rep)
	st, by := e.status(inv.LinkID)
	assert.Equal(t, invites.StatusRevoked, st)
	assert.Equal(t, invites.RevokedByExpired, by)
	assert.Empty(t, e.grants(inv.LinkID))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE companion_id = ? AND action = ?`, inv.LinkID, invites.ActionInviteExpired))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND JSON_EXTRACT(data, '$.event') = ?`, owner, invites.NoticeInviteExpired))
	rep, err = e.svc.Sweep(ctx, e.clk.Now(), langs)
	require.NoError(t, err)
	assert.Equal(t, invites.SweepReport{}, rep, "idempotent")

	// Past the link's end date the confirmation is refused even inside the window.
	inv2 := e.create(owner, invites.CreateInput{Role: family.RolePartner, Template: family.TemplateFullFamily,
		ExpiresAt: e.clk.Now().Add(time.Hour)})
	other := e.user("male", "", "")
	_, err = e.svc.Accept(ctx, other, invites.AcceptInput{Code: inv2.Code})
	require.NoError(t, err)
	e.clk.add(2 * time.Hour)
	_, err = e.svc.Confirm(ctx, owner, inv2.LinkID)
	require.ErrorIs(t, err, invites.ErrNotFound)
}

// Review L5: a teen-mode member cannot open a mutual back-link (she would own a non-parent link, CB-TEEN-01).
func TestInvite_MutualRefusedForTeenMember(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner, teen := e.user("female", "cycle", ""), e.user("female", "teen", "")
	in := partnerBuddy()
	in.Mutual = []family.Domain{family.DomainAppointments}
	inv := e.create(owner, in)
	_, err := e.svc.Accept(ctx, teen, invites.AcceptInput{Code: inv.Code,
		Mutual: map[family.Domain]family.Level{family.DomainAppointments: family.LevelView}})
	require.ErrorIs(t, err, invites.ErrTeenParentOnly)
	st, _ := e.status(inv.LinkID)
	assert.Equal(t, invites.StatusInvited, st, "the whole accept rolled back")
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companions WHERE owner_id = ?`, teen))
}
