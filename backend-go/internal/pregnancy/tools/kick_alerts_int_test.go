package tools_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M9-05 / T-M10-08: a kick session short of the target raises the urgent preg.fetal_movement_kicks alert (a locked
// safety rule of the alerts inbox).

type kickAlert struct {
	Level, Kind, State, Evidence string
}

func (e *env) kickAlerts(t *testing.T, user uint64) []kickAlert {
	t.Helper()
	rows, err := e.db.Query(`SELECT level, kind, state, evidence FROM message_inbox
		WHERE user_id = ? AND rule_key = 'preg.fetal_movement_kicks' ORDER BY id`, user)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []kickAlert
	for rows.Next() {
		var a kickAlert
		require.NoError(t, rows.Scan(&a.Level, &a.Kind, &a.State, &a.Evidence))
		out = append(out, a)
	}
	require.NoError(t, rows.Err())
	return out
}

func (e *env) startKicks(t *testing.T, tok string, at time.Time, kicks int) string {
	t.Helper()
	base := "/api/v1/pregnancy/kick-sessions"
	r := e.do(t, at, http.MethodPost, base, tok, "")
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	id := fmt.Sprintf("%s/%.0f", base, r.data()["id"].(float64))
	for i := 1; i <= kicks; i++ {
		r = e.do(t, at.Add(time.Duration(i)*time.Minute), http.MethodPost, id+"/kicks", tok, "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	return id
}

func TestKickAlerts_StopShortOfTarget(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001", true) // 32+1 on t0
	id := e.startKicks(t, tok, t0, 9)
	r := e.do(t, t0.Add(120*time.Minute), http.MethodPost, id+"/stop", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 9, r.data()["kicks"], 0, "the stop response keeps its shape")

	got := e.kickAlerts(t, sara)
	require.Len(t, got, 1)
	assert.Equal(t, kickAlert{Level: "follow_soon", Kind: "safety", State: "active"}, kickAlert{Level: got[0].Level, Kind: got[0].Kind, State: got[0].State},
		"urgent, safety, active: the feed's urgent_modal pick")
	var m struct {
		Vars map[string]any `json:"vars"`
	}
	require.NoError(t, json.Unmarshal([]byte(got[0].Evidence), &m))
	assert.InDelta(t, 9, m.Vars["preg_kick_count"], 0)
	assert.InDelta(t, 10, m.Vars["preg_kick_target"], 0)
	assert.InDelta(t, 120, m.Vars["preg_kick_minutes"], 0)

	// Re-reading does not duplicate it; the fetal day keeps its real status (felt).
	e.do(t, t0.Add(125*time.Minute), http.MethodGet, "/api/v1/pregnancy/kick-sessions", tok, "")
	assert.Len(t, e.kickAlerts(t, sara), 1)
	var status string
	require.NoError(t, e.db.QueryRow(`SELECT movement_status FROM pregnancy_fetal_movements WHERE user_id = ?`, sara).Scan(&status))
	assert.Equal(t, "felt", status)
}

func TestKickAlerts_TargetReachedInWindow(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001", true)
	id := e.startKicks(t, tok, t0, 10)
	r := e.do(t, t0.Add(30*time.Minute), http.MethodPost, id+"/stop", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Empty(t, e.kickAlerts(t, sara))
}

func TestKickAlerts_LazyExpiry(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001", true)
	e.startKicks(t, tok, t0, 4)
	r := e.do(t, t0.Add(119*time.Minute), http.MethodGet, "/api/v1/pregnancy/kick-sessions", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Empty(t, e.kickAlerts(t, sara), "still counting")

	r = e.do(t, t0.Add(121*time.Minute), http.MethodGet, "/api/v1/pregnancy/kick-sessions", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	require.Len(t, e.kickAlerts(t, sara), 1, "a session past its window counts as ended")
	assert.Equal(t, true, obj(r.data(), "active")["is_active"], "the session keeps its real status")
	e.do(t, t0.Add(130*time.Minute), http.MethodGet, "/api/v1/pregnancy/kick-sessions", tok, "")
	assert.Len(t, e.kickAlerts(t, sara), 1)
}

func TestKickAlerts_BeforeWeek28(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001", true)
	// 27+6 on t0: LMP 195 days earlier.
	_, err := e.db.Exec(`UPDATE pregnancy_profiles SET lmp_date = ? WHERE user_id = ?`, t0.AddDate(0, 0, -195).Format(time.DateOnly), sara)
	require.NoError(t, err)
	id := e.startKicks(t, tok, t0, 3)
	e.do(t, t0.Add(2*time.Hour), http.MethodPost, id+"/stop", tok, "")
	assert.Empty(t, e.kickAlerts(t, sara))
}

// The locked rule row inactive or deleted → the alert still fires from the code-held definition (SPEC §4.2).
func TestKickAlerts_WithoutRuleRow(t *testing.T) {
	for name, stmt := range map[string]string{
		"inactive": `UPDATE message_rules SET is_active = 0 WHERE rule_key = 'preg.fetal_movement_kicks'`,
		"deleted":  `DELETE FROM message_rules WHERE rule_key = 'preg.fetal_movement_kicks'`,
	} {
		t.Run(name, func(t *testing.T) {
			e := setup(t)
			_, err := e.db.Exec(stmt)
			require.NoError(t, err)
			sara, tok := e.user(t, "09120000001", true)
			id := e.startKicks(t, tok, t0, 2)
			r := e.do(t, t0.Add(40*time.Minute), http.MethodPost, id+"/stop", tok, "")
			require.Equal(t, http.StatusOK, r.status, r.raw)
			got := e.kickAlerts(t, sara)
			require.Len(t, got, 1)
			assert.Equal(t, "follow_soon", got[0].Level)
			assert.Equal(t, "safety", got[0].Kind)
		})
	}
}
