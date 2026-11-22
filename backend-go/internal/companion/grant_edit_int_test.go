package companion_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
)

// T-M12-17b: the legacy grant edit (SetGrants, PUT /companions/{id}/grants) changes only what its sections can name.
// A v6-only grant on the link (here the urgent alert and vitals, as a v6 invite stores them) is kept with its level
// and epoch, and the grants_changed audit row lists only the real change.
func TestSetGrants_KeepsV6OnlyGrants(t *testing.T) {
	ctx := context.Background()
	e := setup(t)
	owner := e.user(t, "09120017001")
	ali := e.user(t, "09120017002")
	inv, err := e.svc.CreateInvite(ctx, owner, companion.InviteInput{Type: companion.TypePartner,
		Grants: companion.Grants{companion.SectionMeds: companion.LevelView, companion.SectionAppointments: companion.LevelView}})
	require.NoError(t, err)
	_, err = e.svc.Accept(ctx, ali, inv.Code)
	require.NoError(t, err)
	link := inv.Link.ID
	for _, sec := range []string{"urgent_alert", "vitals"} {
		_, err = e.db.Exec(`INSERT INTO companion_grants (companion_id, section, level, mode_epoch, created_at, updated_at)
			SELECT companion_id, ?, 'view', mode_epoch, NOW(), NOW() FROM companion_grants WHERE companion_id = ? AND section = 'meds'`, sec, link)
		require.NoError(t, err)
	}
	epoch := func(sec string) any {
		var ep any
		require.NoError(t, e.db.QueryRow(`SELECT mode_epoch FROM companion_grants WHERE companion_id = ? AND section = ?`, link, sec).Scan(&ep))
		return ep
	}
	urgentEpoch := epoch("urgent_alert")

	_, err = e.svc.SetGrants(ctx, owner, link, companion.Grants{companion.SectionMeds: companion.LevelEdit,
		companion.SectionAppointments: companion.LevelView})
	require.NoError(t, err)

	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'urgent_alert' AND level = 'view'`, link),
		"the urgent alert survives the owner's grant edit")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'vitals' AND level = 'view'`, link))
	assert.Equal(t, urgentEpoch, epoch("urgent_alert"), "a kept grant keeps its epoch")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'meds' AND level = 'write'`, link))

	var raw json.RawMessage
	require.NoError(t, e.db.QueryRow(`SELECT diff_json FROM companion_audit_logs WHERE companion_id = ? AND action = 'grants_changed' ORDER BY id DESC LIMIT 1`, link).Scan(&raw))
	var diff map[string][2]string
	require.NoError(t, json.Unmarshal(raw, &diff))
	assert.Equal(t, map[string][2]string{"meds": {"view", "write"}}, diff, "the audit row lists only the real change")

	// Removing a legacy section still removes it; the v6-only grants stay.
	_, err = e.svc.SetGrants(ctx, owner, link, companion.Grants{})
	require.NoError(t, err)
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM companion_grants WHERE companion_id = ?`, link))
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section IN ('meds', 'appointments')`, link))
}
