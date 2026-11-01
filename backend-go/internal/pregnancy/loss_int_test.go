package pregnancy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CB-LOSS-01: EndForLoss switches pregnancy mode off like /pregnancy/deactivate, closes every open alert (the legacy
// rows and, T-M10-08b, the preg.* inbox rows — other inbox rows are not pregnancy alerts and stay),
// keeps the profile and its history, is idempotent, and touches only the given user.
func TestEndForLoss(t *testing.T) {
	conn, q, uid := setup(t)
	ctx := context.Background()
	res, err := conn.Exec("INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('o', '09900000098', NOW(), NOW())")
	require.NoError(t, err)
	other, err := res.LastInsertId()
	require.NoError(t, err)
	for _, id := range []any{uid, other} {
		_, err := conn.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, age_source, lmp_date, onboarding_completed, created_at, updated_at)
			VALUES (?, 1, 0, 'lmp', '2026-08-02', 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id)
		require.NoError(t, err)
		_, err = conn.Exec(`INSERT INTO pregnancy_alerts (user_id, alert_level, alert_type, title, message, created_at, updated_at) VALUES
			(?, 'emergency', 'bleeding', 't', 'm', '2026-09-20 09:00:00', '2026-09-20 09:00:00'),
			(?, 'info', 'v2:week_entered', 't', 'm', '2026-09-20 09:00:00', '2026-09-20 09:00:00')`, id, id)
		require.NoError(t, err)
		_, err = conn.Exec(`INSERT INTO message_inbox (user_id, rule_id, version, rule_key, kind, level, state, first_at, last_at,
			created_at, updated_at) VALUES
			(?, 1, 1, 'preg.critical_symptom', 'safety', 'follow_soon', 'active', NOW(), NOW(), NOW(), NOW()),
			(?, 2, 1, 'preg.vomiting', 'clinical', 'worth_followup', 'resolved', NOW(), NOW(), NOW(), NOW()),
			(?, 3, 1, 'cycle.late', 'clinical', 'worth_followup', 'active', NOW(), NOW(), NOW(), NOW())`, id, id, id)
		require.NoError(t, err)
	}
	inbox := func(id any) string {
		var s string
		require.NoError(t, conn.QueryRow(`SELECT GROUP_CONCAT(CONCAT(rule_key, '=', state) ORDER BY rule_key) FROM message_inbox
			WHERE user_id = ?`, id).Scan(&s))
		return s
	}

	active, err := EndForLoss(ctx, q, uid, t0)
	require.NoError(t, err)
	assert.True(t, active)
	p, err := LoadProfile(ctx, q, uid)
	require.NoError(t, err)
	require.NotNil(t, p, "the profile stays")
	assert.False(t, p.PregnancyMode)
	assert.True(t, p.CycleMode)
	assert.True(t, p.LmpDate.Valid, "history kept")
	cnt, err := q.CountActiveAlerts(ctx, uid)
	require.NoError(t, err)
	assert.EqualValues(t, 0, cnt.Total)
	assert.Equal(t, "cycle.late=active,preg.critical_symptom=archived,preg.vomiting=archived", inbox(uid))

	active, err = EndForLoss(ctx, q, uid, t0)
	require.NoError(t, err)
	assert.False(t, active, "idempotent")

	op, err := LoadProfile(ctx, q, uint64(other)) //nolint:gosec // test id
	require.NoError(t, err)
	assert.True(t, op.PregnancyMode, "another user is untouched")
	ocnt, err := q.CountActiveAlerts(ctx, uint64(other)) //nolint:gosec // test id
	require.NoError(t, err)
	assert.EqualValues(t, 1, ocnt.Total, "the v1 row (v2 rows are in the inbox since 00113)")
	assert.Equal(t, "cycle.late=active,preg.critical_symptom=active,preg.vomiting=resolved", inbox(other))

	// No profile at all: nothing to stop, no error.
	res, err = conn.Exec("INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('n', '09900000097', NOW(), NOW())")
	require.NoError(t, err)
	none, err := res.LastInsertId()
	require.NoError(t, err)
	active, err = EndForLoss(ctx, q, uint64(none), t0) //nolint:gosec // test id
	require.NoError(t, err)
	assert.False(t, active)
}
