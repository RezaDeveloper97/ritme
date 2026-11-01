package messages_test

// T-M10-08b: the feed's fact source carries the pregnancy alert input (inboxalerts.Input), so BuildFacts — and the
// alerts inbox the feed evaluates — sees the preg.* facts.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestFactsInt_PregnancyAlertFacts(t *testing.T) {
	db := testdb.New(t)
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Mina', '09120007391', NOW(), NOW())`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	uid := uint64(id) //nolint:gosec // test id
	_, err = db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, age_source, lmp_date, onboarding_completed, created_at, updated_at)
		VALUES (?, 1, 0, 'lmp', ?, 1, NOW(), NOW())`, uid, factsToday.AddDays(-(20*7 + 3)).String())
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO pregnancy_symptom_logs (user_id, log_date, has_bleeding, bleeding_severity, created_at, updated_at)
		VALUES (?, ?, 1, 'moderate', NOW(), NOW())`, uid, factsToday.String())
	require.NoError(t, err)

	v, _ := registryFacts(t, db, uid, "preg.critical_symptom", "preg_symptoms", "preg.week_day")
	assert.Equal(t, []string{"bleeding"}, v["preg.critical_symptom"])
	assert.Contains(t, v, "preg_symptoms")
	assert.Equal(t, 3.0, v["preg.week_day"])

	// Not pregnant: no pregnancy alert facts (and no pregnancy alert reads).
	_, err = db.Exec(`UPDATE pregnancy_profiles SET pregnancy_mode = 0, cycle_mode = 1 WHERE user_id = ?`, uid)
	require.NoError(t, err)
	v, _ = registryFacts(t, db, uid, "preg.critical_symptom")
	assert.NotContains(t, v, "preg.critical_symptom")
}
