package teen_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/teen"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// 2026-10-02 is a Friday: this week is Sat 09-26 … Fri 10-02, next week Sat 10-03 … Fri 10-09.
var now = time.Date(2026, 10, 2, 10, 0, 0, 0, civildate.Tehran)

type env struct {
	db  *sql.DB
	clk clock.Clock
	svc *teen.Service
	cmp *companion.Service
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	clk := clock.Fixed(now)
	return &env{db: db, clk: clk,
		svc: teen.NewService(db, catalog.NewReader(catalogstore.New(db), nil, 0, nil)),
		cmp: companion.NewService(db, clk, companion.Options{CodePepper: []byte("test-pepper")})}
}

func (e *env) user(t *testing.T, name, mobile string) uint64 {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES (?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, name, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // positive id
}

func (e *env) teen(t *testing.T, name, mobile string) uint64 {
	t.Helper()
	id := e.user(t, name, mobile)
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, 'female', 'teen', NOW(), NOW())`, id)
	require.NoError(t, err)
	return id
}

func (e *env) periods(t *testing.T, uid uint64, starts ...string) {
	t.Helper()
	for _, s := range starts {
		d := civildate.MustParse(s)
		_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, source, created_at, updated_at)
			VALUES (?, ?, ?, 1, 'user_logged', NOW(), NOW())`, uid, d, d.AddDays(4))
		require.NoError(t, err)
	}
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

// link: teen invites a parent (bound to the parent's number) with grants, the parent accepts.
func (e *env) link(t *testing.T, teenID, parentID uint64, grants companion.Grants) uint64 {
	t.Helper()
	ctx := context.Background()
	var mobile string
	require.NoError(t, e.db.QueryRow(`SELECT mobile FROM users WHERE id = ?`, parentID).Scan(&mobile))
	inv, err := e.cmp.CreateInvite(ctx, teenID, companion.InviteInput{Type: companion.TypeParent, Phone: mobile, Grants: grants})
	require.NoError(t, err)
	_, err = e.cmp.Accept(ctx, parentID, inv.Code)
	require.NoError(t, err)
	return inv.Link.ID
}

func TestProfileKitNote_UserScoped(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	a := e.teen(t, "Nila", "09120000201")
	b := e.teen(t, "Sara", "09120000202")

	p, err := e.svc.Profile(ctx, a)
	require.NoError(t, err)
	assert.Nil(t, p)
	_, err = e.svc.SetParentNote(ctx, a, "hi", now)
	require.ErrorIs(t, err, teen.ErrProfileRequired)

	p, err = e.svc.SaveProfile(ctx, a, teen.Age13to15, teen.MenarcheNotYet, now)
	require.NoError(t, err)
	assert.Equal(t, teen.Age13to15, p.AgeBand)
	p, err = e.svc.SetParentNote(ctx, a, "  کیفم آماده است  ", now)
	require.NoError(t, err)
	assert.Equal(t, "کیفم آماده است", p.ParentNote)
	p, err = e.svc.SaveProfile(ctx, a, teen.Age16to17, teen.MenarcheUnder1y, now)
	require.NoError(t, err)
	assert.Equal(t, "کیفم آماده است", p.ParentNote, "re-saving the answers keeps the note")

	k, err := e.svc.SetKitItem(ctx, a, "pads", true, now)
	require.NoError(t, err)
	assert.Equal(t, 1, k.Checked)
	assert.Len(t, k.Items, 4)
	_, err = e.svc.SetKitItem(ctx, a, "pads", true, now) // idempotent
	require.NoError(t, err)
	_, err = e.svc.SetKitItem(ctx, a, "lipstick", true, now)
	require.ErrorIs(t, err, teen.ErrUnknownKitItem)

	// Teen B sees none of A's rows, and her writes never touch A's.
	pb, err := e.svc.Profile(ctx, b)
	require.NoError(t, err)
	assert.Nil(t, pb)
	kb, err := e.svc.Kit(ctx, b)
	require.NoError(t, err)
	assert.Zero(t, kb.Checked)
	_, err = e.svc.SetKitItem(ctx, b, "pads", false, now)
	require.NoError(t, err)
	k, err = e.svc.Kit(ctx, a)
	require.NoError(t, err)
	assert.Equal(t, 1, k.Checked, "B unticking does not untick A")
	_, err = e.svc.SetParentNote(ctx, b, "x", now)
	require.ErrorIs(t, err, teen.ErrProfileRequired, "B's note never lands on A's profile")
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM teen_profiles"))

	for _, code := range []string{"underwear", "wipes", "pouch"} {
		k, err = e.svc.SetKitItem(ctx, a, code, true, now)
		require.NoError(t, err)
	}
	assert.True(t, k.Ready())
	k, err = e.svc.SetKitItem(ctx, a, "wipes", false, now)
	require.NoError(t, err)
	assert.False(t, k.Ready())
}

func TestContent_ReadinessFollowsAnswers(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	cases := []struct {
		age  teen.AgeBand
		m    teen.Menarche
		want string
	}{
		{teen.Age10to12, teen.MenarcheNotYet, "estimate_year_or_two"},
		{teen.Age13to15, teen.MenarcheNotYet, "estimate_coming_months"},
		{teen.Age16to17, teen.MenarcheNotYet, "estimate_talk"},
		{teen.Age13to15, teen.MenarcheUnder1y, "estimate_first_year"},
		{teen.Age16to17, teen.MenarcheOver1y, "estimate_settling"},
	}
	for _, c := range cases {
		got, err := e.svc.Content(ctx, &teen.Profile{AgeBand: c.age, Menarche: c.m})
		require.NoError(t, err)
		require.NotNil(t, got.Readiness, c.want)
		assert.Equal(t, c.want, got.Readiness.Code)
		assert.True(t, got.Readiness.NeedsReview)
		require.NotNil(t, got.Talk)
		assert.Len(t, got.FAQ, 5)
		if c.m == teen.MenarcheNotYet {
			assert.Len(t, got.Signs, 2)
		} else {
			assert.Empty(t, got.Signs, "approach signs only before the first period")
		}
	}
	got, err := e.svc.Content(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, got.Readiness, "no answers, no estimate")
}

func TestNextPeriodWeek(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	a := e.teen(t, "Nila", "09120000211")
	today := civildate.InTehran(now)

	w, err := e.svc.NextPeriodWeek(ctx, a, nil, today)
	require.NoError(t, err)
	assert.Equal(t, teen.WeekUnknown, w)
	w, err = e.svc.NextPeriodWeek(ctx, a, &teen.Profile{Menarche: teen.MenarcheOver1y}, today)
	require.NoError(t, err)
	assert.Equal(t, teen.WeekUnknown, w, "no period data")

	e.periods(t, a, "2026-07-14", "2026-08-11", "2026-09-08") // 28-day cycles → next 2026-10-06 (next week)
	w, err = e.svc.NextPeriodWeek(ctx, a, &teen.Profile{Menarche: teen.MenarcheOver1y}, today)
	require.NoError(t, err)
	assert.Equal(t, teen.WeekNext, w)
	w, err = e.svc.NextPeriodWeek(ctx, a, &teen.Profile{Menarche: teen.MenarcheNotYet}, today)
	require.NoError(t, err)
	assert.Equal(t, teen.WeekUnknown, w, "before the first period: unknown whatever is logged")
}

func TestParentCards_AccessRules(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	a := e.teen(t, "Nila", "09120000221")
	b := e.teen(t, "Sara", "09120000222")
	mom := e.user(t, "Mom", "09120000223")
	momB := e.user(t, "MomB", "09120000224")
	stranger := e.user(t, "Stranger", "09120000225")

	_, err := e.svc.SaveProfile(ctx, a, teen.Age13to15, teen.MenarcheOver1y, now)
	require.NoError(t, err)
	_, err = e.svc.SetParentNote(ctx, a, "Need new pads", now)
	require.NoError(t, err)
	_, err = e.svc.SaveProfile(ctx, b, teen.Age13to15, teen.MenarcheOver1y, now)
	require.NoError(t, err)
	_, err = e.svc.SetParentNote(ctx, b, "B's secret", now)
	require.NoError(t, err)
	e.periods(t, a, "2026-07-14", "2026-08-11", "2026-09-08")
	for _, code := range []string{"pads", "underwear", "wipes", "pouch"} {
		_, err = e.svc.SetKitItem(ctx, a, code, true, now)
		require.NoError(t, err)
	}

	// Default most private: an accepted parent link with no grants → a card with every part null, nothing read.
	linkA := e.link(t, a, mom, nil)
	cards, err := e.svc.ParentCards(ctx, mom, e.clk)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	assert.Equal(t, a, cards[0].Link.OwnerID)
	assert.Equal(t, "Nila", cards[0].TeenName)
	assert.Nil(t, cards[0].View.PeriodWeek)
	assert.Nil(t, cards[0].View.KitReady)
	assert.Nil(t, cards[0].View.Note)
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM companion_audit_logs WHERE action = 'read'"), "nothing granted, nothing read")

	// The teen grants the period week only.
	_, err = e.cmp.SetGrants(ctx, a, linkA, companion.Grants{companion.SectionTeenPeriodWeek: companion.LevelView})
	require.NoError(t, err)
	cards, err = e.svc.ParentCards(ctx, mom, e.clk)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.NotNil(t, cards[0].View.PeriodWeek)
	assert.Equal(t, teen.WeekNext, *cards[0].View.PeriodWeek)
	assert.Nil(t, cards[0].View.KitReady)
	assert.Nil(t, cards[0].View.Note, "the note is never shown without its grant")
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM companion_audit_logs WHERE action = 'read' AND section = 'teen_period_week' AND actor_id = ? AND owner_id = ?", mom, a))

	// Everything granted.
	_, err = e.cmp.SetGrants(ctx, a, linkA, companion.Grants{companion.SectionTeenPeriodWeek: companion.LevelView,
		companion.SectionTeenKit: companion.LevelView, companion.SectionTeenNotes: companion.LevelView})
	require.NoError(t, err)
	cards, err = e.svc.ParentCards(ctx, mom, e.clk)
	require.NoError(t, err)
	require.NotNil(t, cards[0].View.KitReady)
	assert.True(t, *cards[0].View.KitReady)
	require.NotNil(t, cards[0].View.Note)
	assert.Equal(t, "Need new pads", *cards[0].View.Note)
	// 3, not 4: the second period-week read within 15 minutes is coalesced (B-N4-08b, CMP-L1).
	assert.Equal(t, 3, e.count(t, "SELECT COUNT(*) FROM companion_audit_logs WHERE action = 'read' AND actor_id = ?", mom))

	// Isolation: teen B's parent sees only B; A's parent never sees B; a stranger sees nothing; B is not A's parent.
	e.link(t, b, momB, companion.Grants{companion.SectionTeenNotes: companion.LevelView})
	cards, err = e.svc.ParentCards(ctx, momB, e.clk)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	assert.Equal(t, b, cards[0].Link.OwnerID)
	assert.Equal(t, "B's secret", *cards[0].View.Note)
	cards, err = e.svc.ParentCards(ctx, mom, e.clk)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	assert.Equal(t, a, cards[0].Link.OwnerID)
	for _, viewer := range []uint64{stranger, b, a} {
		cards, err = e.svc.ParentCards(ctx, viewer, e.clk)
		require.NoError(t, err)
		assert.Empty(t, cards, "viewer %d is nobody's parent", viewer)
	}

	// A partner-type link is never a parent card (an adult owner's partner sees nothing here).
	adult := e.user(t, "Adult", "09120000226")
	inv, err := e.cmp.CreateInvite(ctx, adult, companion.InviteInput{Type: companion.TypePartner,
		Grants: companion.Grants{companion.SectionCycle: companion.LevelView}})
	require.NoError(t, err)
	_, err = e.cmp.Accept(ctx, stranger, inv.Code)
	require.NoError(t, err)
	cards, err = e.svc.ParentCards(ctx, stranger, e.clk)
	require.NoError(t, err)
	assert.Empty(t, cards)

	// L2: once the owner left teen mode the card is gone (nothing read); back in teen mode it returns.
	reads := e.count(t, "SELECT COUNT(*) FROM companion_audit_logs WHERE action = 'read' AND actor_id = ?", mom)
	_, err = e.db.Exec(`UPDATE user_life_profiles SET life_mode = 'cycle' WHERE user_id = ?`, a)
	require.NoError(t, err)
	cards, err = e.svc.ParentCards(ctx, mom, e.clk)
	require.NoError(t, err)
	assert.Empty(t, cards)
	assert.Equal(t, reads, e.count(t, "SELECT COUNT(*) FROM companion_audit_logs WHERE action = 'read' AND actor_id = ?", mom))
	_, err = e.db.Exec(`UPDATE user_life_profiles SET life_mode = 'teen' WHERE user_id = ?`, a)
	require.NoError(t, err)
	cards, err = e.svc.ParentCards(ctx, mom, e.clk)
	require.NoError(t, err)
	assert.Len(t, cards, 1)

	// The teen revokes: the card is gone at once.
	require.NoError(t, e.cmp.Revoke(ctx, a, linkA))
	cards, err = e.svc.ParentCards(ctx, mom, e.clk)
	require.NoError(t, err)
	assert.Empty(t, cards)
}

func TestToday_PreviewAndAllows(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	a := e.teen(t, "Nila", "09120000231")
	adult := e.user(t, "Adult", "09120000232")

	td, err := e.svc.Today(ctx, a, e.clk)
	require.NoError(t, err)
	assert.Nil(t, td.Profile)
	assert.True(t, td.TeenMode)
	assert.False(t, td.Allows.Shop || td.Allows.Banners || td.Allows.Ads || td.Allows.PlusUpsell || td.Allows.CommercialRecommendations)
	assert.Len(t, td.Kit.Items, 4)
	assert.Empty(t, td.ParentLinks)

	td, err = e.svc.Today(ctx, adult, e.clk)
	require.NoError(t, err)
	assert.False(t, td.TeenMode)
	assert.True(t, td.Allows.Shop && td.Allows.Banners)

	mom := e.user(t, "Mom", "09120000233")
	e.link(t, a, mom, companion.Grants{companion.SectionTeenKit: companion.LevelView})
	td, err = e.svc.Today(ctx, a, e.clk)
	require.NoError(t, err)
	require.Len(t, td.ParentLinks, 1)
	assert.Equal(t, companion.LevelView, td.ParentLinks[0].Grants.Of(companion.SectionTeenKit))

	pol := teen.NewPolicy(e.db)
	ok, err := pol.AllowsCommercial(ctx, a)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = pol.AllowsCommercial(ctx, adult)
	require.NoError(t, err)
	assert.True(t, ok, "no life profile row = not teen")
}
