package companion_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
)

// CB-TEEN-01: the parent link type and its teen-only, view-only sections.

func (e *env) setMode(t *testing.T, userID uint64, mode string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE life_mode = VALUES(life_mode)`, userID, mode)
	require.NoError(t, err)
}

func TestParent_OnlyTeenInvitesParentAndTeenInvitesOnlyParent(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	adult := e.user(t, "09120000101")
	teen := e.user(t, "09120000102")
	e.setMode(t, teen, "teen")

	_, err := e.svc.CreateInvite(ctx, teen, companion.InviteInput{Type: companion.TypeParent})
	require.ErrorIs(t, err, companion.ErrPhoneRequired, "a parent invite is bound to the parent's number")
	_, err = e.svc.CreateInvite(ctx, adult, companion.InviteInput{Type: companion.TypeParent, Phone: "09120000109"})
	require.ErrorIs(t, err, companion.ErrParentNeedsTeen, "a non-teen owner cannot invite a parent")

	for _, typ := range []companion.Type{companion.TypePartner, companion.TypeSpouse} {
		_, err = e.svc.CreateInvite(ctx, teen, companion.InviteInput{Type: typ})
		require.ErrorIs(t, err, companion.ErrTeenParentOnly, "a minor cannot link a %s", typ)
	}
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM companions"), "refused invites leave nothing behind")

	_, err = e.svc.CreateInvite(ctx, teen, companion.InviteInput{Type: companion.TypeParent, Phone: "09120000109",
		Grants: companion.Grants{companion.SectionCycle: companion.LevelView}})
	require.ErrorIs(t, err, companion.ErrInvalidSection, "a parent never gets a regular section")
	_, err = e.svc.CreateInvite(ctx, teen, companion.InviteInput{Type: companion.TypeParent, Phone: "09120000109",
		Grants: companion.Grants{companion.SectionTeenKit: companion.LevelEdit}})
	require.ErrorIs(t, err, companion.ErrInvalidLevel, "a parent never writes")

	inv, err := e.svc.CreateInvite(ctx, teen, companion.InviteInput{Type: companion.TypeParent, Phone: "09120000109"})
	require.NoError(t, err)
	assert.Empty(t, inv.Link.Grants, "default most private: nothing shared")
}

func TestParent_AccessIsTeenSectionsViewOnly(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	teen := e.user(t, "09120000111")
	mom := e.user(t, "09120000112")
	stranger := e.user(t, "09120000113")
	e.setMode(t, teen, "teen")

	inv, err := e.svc.CreateInvite(ctx, teen, companion.InviteInput{Type: companion.TypeParent, Phone: "09120000112",
		Grants: companion.Grants{companion.SectionTeenKit: companion.LevelView}})
	require.NoError(t, err)

	// Pending: nothing.
	l, err := e.svc.Level(ctx, teen, mom, companion.SectionTeenKit)
	require.NoError(t, err)
	assert.Equal(t, companion.LevelNone, l)

	_, err = e.svc.Accept(ctx, mom, inv.Code)
	require.NoError(t, err)

	l, err = e.svc.Level(ctx, teen, mom, companion.SectionTeenKit)
	require.NoError(t, err)
	assert.Equal(t, companion.LevelView, l)
	for _, s := range []companion.Section{companion.SectionTeenPeriodWeek, companion.SectionTeenNotes} {
		l, err = e.svc.Level(ctx, teen, mom, s)
		require.NoError(t, err)
		assert.Equal(t, companion.LevelNone, l, "ungranted %s", s)
	}
	for _, s := range companion.Sections {
		l, err = e.svc.Level(ctx, teen, mom, s)
		require.NoError(t, err)
		assert.Equal(t, companion.LevelNone, l, "regular section %s", s)
	}
	for _, s := range companion.TeenSections {
		ok, err := e.svc.CanWrite(ctx, teen, mom, s)
		require.NoError(t, err)
		assert.False(t, ok, "a parent never writes %s", s)
		ok, err = e.svc.CanRead(ctx, teen, stranger, s)
		require.NoError(t, err)
		assert.False(t, ok, "no link = nothing")
	}

	// Delegated care writes («ثبت برای …») are refused too.
	d := companion.NewDelegation(e.db, e.clk, nil)
	ok, err := d.Authorize(ctx, teen, mom, companion.SectionMeds, true)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = d.Authorize(ctx, teen, mom, companion.SectionMeds, false)
	require.NoError(t, err)
	assert.False(t, ok)

	// Defence in depth: rows the type may not hold (written behind the service's back) grant nothing / never edit.
	_, err = e.db.Exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES
		(?, 'cycle', 'edit', NOW(), NOW()), (?, 'teen_notes', 'edit', NOW(), NOW())`, inv.Link.ID, inv.Link.ID)
	require.NoError(t, err)
	l, err = e.svc.Level(ctx, teen, mom, companion.SectionCycle)
	require.NoError(t, err)
	assert.Equal(t, companion.LevelNone, l)
	l, err = e.svc.Level(ctx, teen, mom, companion.SectionTeenNotes)
	require.NoError(t, err)
	assert.Equal(t, companion.LevelView, l, "a teen section is never writable")
	links, err := e.svc.ListForCompanion(ctx, mom)
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.NotContains(t, links[0].Grants, companion.SectionCycle, "foreign rows are not shown as grants")

	// SetGrants keeps to the type.
	_, err = e.svc.SetGrants(ctx, teen, inv.Link.ID, companion.Grants{companion.SectionSymptoms: companion.LevelView})
	require.ErrorIs(t, err, companion.ErrInvalidSection)
	_, err = e.svc.SetGrants(ctx, teen, inv.Link.ID, companion.Grants{companion.SectionTeenPeriodWeek: companion.LevelEdit})
	require.ErrorIs(t, err, companion.ErrInvalidLevel)
	link, err := e.svc.SetGrants(ctx, teen, inv.Link.ID, companion.Grants{companion.SectionTeenNotes: companion.LevelView})
	require.NoError(t, err)
	assert.Equal(t, companion.Grants{companion.SectionTeenNotes: companion.LevelView}, link.Grants, "the teen narrowed it")
	l, err = e.svc.Level(ctx, teen, mom, companion.SectionTeenKit)
	require.NoError(t, err)
	assert.Equal(t, companion.LevelNone, l, "takes effect at once")

	// The teen ends it: nothing left.
	require.NoError(t, e.svc.Revoke(ctx, teen, inv.Link.ID))
	l, err = e.svc.Level(ctx, teen, mom, companion.SectionTeenNotes)
	require.NoError(t, err)
	assert.Equal(t, companion.LevelNone, l)
}

// L1: the rule is re-checked on accept and renew under the owner lock — the owner's mode may change after inviting.
func TestParent_ModeChangeAfterInvite(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120000121")
	ali := e.user(t, "09120000122")
	mom := e.user(t, "09120000123")

	// A partner invite made as an adult; the owner then switches to teen mode → it cannot be accepted or renewed.
	partner, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner,
		Grants: companion.Grants{companion.SectionCycle: companion.LevelView}})
	require.NoError(t, err)
	e.setMode(t, owner, "teen")
	_, err = e.svc.Accept(ctx, ali, partner.Code)
	require.ErrorIs(t, err, companion.ErrInviteNotAllowed)
	_, err = e.svc.RenewInvite(ctx, owner, partner.Link.ID, "")
	require.ErrorIs(t, err, companion.ErrTeenParentOnly)
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM companions WHERE status = 'active'"))
	assert.Zero(t, e.count(t, "SELECT attempts FROM companion_invites WHERE companion_id = ?", partner.Link.ID), "not a failed attempt")

	// A parent invite made in teen mode; she leaves teen mode → it cannot be accepted or renewed either.
	parent, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypeParent, Phone: "09120000123"})
	require.NoError(t, err)
	e.setMode(t, owner, "cycle")
	_, err = e.svc.Accept(ctx, mom, parent.Code)
	require.ErrorIs(t, err, companion.ErrInviteNotAllowed)
	_, err = e.svc.RenewInvite(ctx, owner, parent.Link.ID, "")
	require.ErrorIs(t, err, companion.ErrParentNeedsTeen)

	// Back in teen mode the parent invite works.
	e.setMode(t, owner, "teen")
	_, err = e.svc.Accept(ctx, mom, parent.Code)
	require.NoError(t, err)
}
