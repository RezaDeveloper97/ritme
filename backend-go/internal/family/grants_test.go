package family_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/family"
)

// legacyScope is what the /companions grant edit can name (companion.legacyScope).
var legacyScope = []family.Domain{family.DomainCyclePhase, family.DomainPregnancy, family.DomainMeds,
	family.DomainAppointments, family.DomainTeenPeriodWeek, family.DomainTeenKit, family.DomainTeenNotes}

func (e *env) applyScoped(link, owner, member uint64, scope []family.Domain, set map[family.Domain]family.Level) family.Applied {
	e.t.Helper()
	tx, err := e.db.Begin()
	require.NoError(e.t, err)
	out, err := family.Grants{}.Apply(context.Background(), tx, family.Change{LinkID: link, OwnerID: owner, MemberID: member,
		ActorID: owner, Role: family.RolePartner, Set: set, Scope: scope, Reason: family.ChangeOwnerEdit, Now: now})
	require.NoError(e.t, err)
	require.NoError(e.t, tx.Commit())
	return out
}

func (e *env) levels(link uint64) map[string]string {
	e.t.Helper()
	rows, err := e.db.Query(`SELECT section, level FROM companion_grants WHERE companion_id = ?`, link)
	require.NoError(e.t, err)
	defer func() { _ = rows.Close() }()
	out := map[string]string{}
	for rows.Next() {
		var s, l string
		require.NoError(e.t, rows.Scan(&s, &l))
		out[s] = l
	}
	require.NoError(e.t, rows.Err())
	return out
}

// T-M12-17b: a scoped change (the legacy grant edit) changes only its scope's domains. The v6-only grants survive with
// their level and epoch (a paused one stays paused: the owner did not re-approve what she could not name, F-23), the
// urgent alert stays live, and the audit row and the result diff list only what really changed.
func TestGrants_ScopedApplyKeepsV6OnlyGrants(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	link := e.link(owner, member, map[family.Domain]family.Level{
		family.DomainMeds: family.LevelNudge, family.DomainAppointments: family.LevelWrite,
		family.DomainUrgentAlert: family.LevelView, family.DomainVitals: family.LevelView, family.DomainHousehold: family.LevelWrite})

	e.exec(`UPDATE user_life_profiles SET life_mode = 'menopause' WHERE user_id = ?`, owner) // a mode change: new epoch
	paused := family.Decision{Outcome: family.Deny, Reason: family.ReasonModeInactive}
	require.Equal(t, paused, decide(t, e.db, owner, member, family.DomainVitals, family.ActionRead))

	// The owner's review through the legacy edit: meds nudge → view, appointments kept at write (re-approved).
	out := e.applyScoped(link, owner, member, legacyScope, map[family.Domain]family.Level{
		family.DomainMeds: family.LevelView, family.DomainAppointments: family.LevelWrite})
	assert.Equal(t, map[string][2]string{"meds": {"nudge", "view"}}, out.Diff, "only the real change")
	assert.Equal(t, map[string]string{"meds": "view", "appointments": "write", "urgent_alert": "view", "vitals": "view",
		"household": "write"}, e.levels(link))

	assert.Equal(t, family.Allow, decide(t, e.db, owner, member, family.DomainUrgentAlert, family.ActionRead).Outcome,
		"the urgent alert stays on (F-23)")
	assert.Equal(t, family.Allow, decide(t, e.db, owner, member, family.DomainMeds, family.ActionRead).Outcome, "re-approved")
	assert.Equal(t, family.Propose, decide(t, e.db, owner, member, family.DomainAppointments, family.ActionWrite).Outcome)
	assert.Equal(t, paused, decide(t, e.db, owner, member, family.DomainVitals, family.ActionRead),
		"a kept epoch-bound grant the edit could not name stays paused")

	var diff json.RawMessage
	require.NoError(t, e.db.QueryRow(`SELECT diff_json FROM companion_audit_logs WHERE owner_id = ? AND action = 'grants_changed' ORDER BY id DESC LIMIT 1`, owner).Scan(&diff))
	var got map[string][2]string
	require.NoError(t, json.Unmarshal(diff, &got))
	assert.Equal(t, out.Diff, got)
	// The consent event records everything the member is granted after the change, kept rows included.
	var scope json.RawMessage
	require.NoError(t, e.db.QueryRow(`SELECT scope FROM consent_events WHERE user_id = ? AND consent = 'family_sharing' ORDER BY id DESC LIMIT 1`, owner).Scan(&scope))
	var sc map[string]any
	require.NoError(t, json.Unmarshal(scope, &sc))
	assert.Equal(t, map[string]any{"meds": "view", "appointments": "write", "urgent_alert": "view", "vitals": "view",
		"household": "write"}, sc["domains"])

	// Turning every legacy domain off leaves the v6-only grants — not a consent revoke while something is still shared.
	out = e.applyScoped(link, owner, member, legacyScope, map[family.Domain]family.Level{})
	assert.Equal(t, map[string][2]string{"meds": {"view", "off"}, "appointments": {"write", "off"}}, out.Diff)
	assert.Equal(t, map[string]string{"urgent_alert": "view", "vitals": "view", "household": "write"}, e.levels(link))
	var action string
	require.NoError(t, e.db.QueryRow(`SELECT action FROM consent_events WHERE user_id = ? AND consent = 'family_sharing' ORDER BY id DESC LIMIT 1`, owner).Scan(&action))
	assert.Equal(t, "grant", action)
}

// Explicit removal still removes: a scope that names the domain turns it off, and an unscoped (complete) set keeps
// the old meaning — every domain left out is off.
func TestGrants_ExplicitRemovalStillRemoves(t *testing.T) {
	e := newEnv(t)
	owner, member := e.user("female", "cycle"), e.user("male", "")
	link := e.link(owner, member, map[family.Domain]family.Level{family.DomainMeds: family.LevelNudge,
		family.DomainUrgentAlert: family.LevelView, family.DomainVitals: family.LevelView})

	out := e.applyScoped(link, owner, member, []family.Domain{family.DomainUrgentAlert}, map[family.Domain]family.Level{})
	assert.Equal(t, map[string][2]string{"urgent_alert": {"view", "off"}}, out.Diff)
	assert.Equal(t, map[string]string{"meds": "nudge", "vitals": "view"}, e.levels(link))
	assert.Equal(t, family.Deny, decide(t, e.db, owner, member, family.DomainUrgentAlert, family.ActionRead).Outcome)

	out = e.apply(link, owner, member, map[family.Domain]family.Level{family.DomainMeds: family.LevelNudge})
	assert.Equal(t, map[string][2]string{"vitals": {"view", "off"}}, out.Diff)
	assert.Equal(t, map[string]string{"meds": "nudge"}, e.levels(link))
}
