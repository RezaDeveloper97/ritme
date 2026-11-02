package revision_test

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M10-28 (D-177): every write request to an analysis-only input (vitals, labs, EPDS, the recovery red-flag row, baby
// logs, pregnancy logs) moves user_revisions.analysis_rev by exactly 1 and leaves engine_rev alone, through the
// production wiring (internal/http.Mount); a refused write moves nothing.

// arev is the user's analysis_rev (0 without a user_revisions row).
func (e *env) arev(uid uint64) int {
	e.t.Helper()
	var n int
	err := e.db.QueryRow(`SELECT analysis_rev FROM user_revisions WHERE user_id = ?`, uid).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0
	}
	require.NoError(e.t, err)
	return n
}

// analysisBumps runs one request and asserts its status, that it moved analysis_rev by want and engine_rev by 0.
func (e *env) analysisBumps(want int, uid uint64, tok string, status int, method, path string, body any) reply {
	e.t.Helper()
	beforeA, beforeE := e.arev(uid), e.rev(uid)
	r := e.do(method, path, tok, body)
	require.Equal(e.t, status, r.Status, "%s %s: %s", method, path, r.Raw)
	assert.Equal(e.t, want, e.arev(uid)-beforeA, "%s %s moves analysis_rev by %d", method, path, want)
	assert.Equal(e.t, 0, e.rev(uid)-beforeE, "%s %s leaves engine_rev alone", method, path)
	return r
}

func id(t *testing.T, v any) string {
	t.Helper()
	f, ok := v.(float64)
	require.True(t, ok, "id %v", v)
	return strconv.Itoa(int(f))
}

func TestBumpAnalysis_Vitals(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	r := e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/vitals/readings", map[string]any{
		"type": "bp", "systolic": 118, "diastolic": 76, "pulse": 70, "measured_at": "2026-09-22 08:10",
	})
	rid := id(t, r.data()["reading"].(map[string]any)["id"])
	// A past reading edited: the analysis that read it must not be served again.
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/vitals/readings/"+rid, map[string]any{
		"type": "bp", "systolic": 128, "diastolic": 82, "measured_at": "2026-09-01 08:10",
	})
	e.analysisBumps(1, uid, tok, 200, fiber.MethodDelete, "/api/v1/vitals/readings/"+rid, nil)

	e.analysisBumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/vitals/readings", map[string]any{"type": "mood"})
	e.analysisBumps(0, uid, tok, 404, fiber.MethodDelete, "/api/v1/vitals/readings/"+rid, nil)
}

func TestBumpAnalysis_Labs(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	r := e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/labs/manual", map[string]any{
		"category": "blood", "taken_on": "2026-09-01",
		"markers": []any{map[string]any{"name": "Hb", "value": 10.7, "unit": "g/dL"}},
	})
	lab := id(t, r.data()["id"])
	base := "/api/v1/labs/" + lab
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPut, base, map[string]any{"category": "blood", "taken_on": "2026-09-02"})
	r = e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, base+"/markers", map[string]any{"name": "TSH", "value": 2, "unit": "mIU/L"})
	mid := id(t, r.data()["id"])
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPut, base+"/markers/"+mid, map[string]any{"name": "TSH", "value": 2.5, "unit": "mIU/L"})
	e.analysisBumps(1, uid, tok, 200, fiber.MethodDelete, base+"/markers/"+mid, nil)
	e.analysisBumps(0, uid, tok, 404, fiber.MethodDelete, base+"/markers/"+mid, nil)

	// An uploaded lab waiting for review (as the extraction leaves it): confirming a low-confidence value and verifying
	// the lab both bump.
	res, err := e.db.Exec(`INSERT INTO lab_reports (user_id, source, category, taken_on, status, progress, created_at, updated_at)
		VALUES (?, 'upload', 'blood', '2026-09-10', 'needs_review', 100, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, uid)
	require.NoError(t, err)
	up, err := res.LastInsertId()
	require.NoError(t, err)
	res, err = e.db.Exec(`INSERT INTO lab_markers (lab_id, user_id, name, value, unit, confidence, source, sort_order, created_at, updated_at)
		VALUES (?, ?, 'Ferritin', 20, 'ng/mL', 0.4, 'extracted', 1, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, up, uid)
	require.NoError(t, err)
	umid, err := res.LastInsertId()
	require.NoError(t, err)
	upBase := fmt.Sprintf("/api/v1/labs/%d", up)
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPost, fmt.Sprintf("%s/markers/%d/confirm", upBase, umid), nil)
	e.analysisBumps(1, uid, tok, 202, fiber.MethodPost, upBase+"/verify", nil)

	e.analysisBumps(1, uid, tok, 200, fiber.MethodDelete, base, nil)
	e.analysisBumps(0, uid, tok, 404, fiber.MethodDelete, base, nil)
	e.analysisBumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/labs/manual", map[string]any{"category": "nonsense"})
}

func TestBumpAnalysis_Postpartum(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)
	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/onboarding", map[string]any{"age_source": "lmp", "lmp_date": "2026-01-02"})
	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/postpartum/activate", map[string]any{"birth_date": "2026-09-10"})

	r := e.do(fiber.MethodPost, "/api/v1/postpartum/epds", tok, map[string]any{"kind": "short", "answers": map[string]any{"q3": 0, "q4": 1, "q5": 0}})
	require.Less(t, r.Status, 300, r.Raw)
	before := e.arev(uid)
	status := r.Status
	e.analysisBumps(1, uid, tok, status, fiber.MethodPost, "/api/v1/postpartum/epds", map[string]any{"kind": "short", "answers": map[string]any{"q3": 1, "q4": 1, "q5": 0}})
	assert.Equal(t, before+1, e.arev(uid))
	// The red-flag row only (no taxonomy slot): an analysis-only write.
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/postpartum/recovery", map[string]any{"temperature_c": 37.2})
	e.analysisBumps(0, uid, tok, 422, fiber.MethodPut, "/api/v1/postpartum/recovery", map[string]any{"temperature_c": 37.25})
}

func TestBumpAnalysis_BabyLog(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)
	r := e.do(fiber.MethodPost, "/api/v1/children", tok, map[string]any{"name": "Ava", "birth_date": "2026-06-20", "sex": "girl"})
	require.Equal(t, 201, r.Status, r.Raw)
	base := "/api/v1/children/" + id(t, r.data()["id"])

	r = e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, base+"/feeds/start", map[string]any{"type": "breast", "side": "right"})
	fid := id(t, r.data()["id"])
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPost, base+"/feeds/"+fid+"/side", map[string]any{"side": "left"})
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPost, base+"/feeds/"+fid+"/stop", map[string]any{})
	e.analysisBumps(1, uid, tok, 200, fiber.MethodDelete, base+"/feeds/"+fid, nil)

	r = e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, base+"/sleeps/start", nil)
	sid := id(t, r.data()["id"])
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPost, base+"/sleeps/"+sid+"/stop", map[string]any{})
	e.analysisBumps(1, uid, tok, 200, fiber.MethodDelete, base+"/sleeps/"+sid, nil)
	e.analysisBumps(0, uid, tok, 404, fiber.MethodDelete, base+"/sleeps/"+sid, nil)

	r = e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, base+"/diapers", map[string]any{"kind": "wet"})
	did := id(t, r.data()["id"])
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPut, base+"/diapers/"+did, map[string]any{"kind": "dirty"})
	e.analysisBumps(1, uid, tok, 200, fiber.MethodDelete, base+"/diapers/"+did, nil)
	e.analysisBumps(0, uid, tok, 422, fiber.MethodPost, base+"/diapers", map[string]any{"kind": "glitter"})
}

func TestBumpAnalysis_PregnancyLogs(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)
	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/onboarding", map[string]any{"age_source": "lmp", "lmp_date": "2026-06-02"})

	// v1 logs: weekly (weight / BP), symptoms, fetal movement.
	e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/weekly", map[string]any{
		"pregnancy_week": 16, "log_date": "2026-09-22", "weight": 64.5, "systolic_pressure": 118, "diastolic_pressure": 75,
	})
	e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/symptoms", map[string]any{"log_date": "2026-09-22", "has_nausea": true})
	e.analysisBumps(1, uid, tok, 200, fiber.MethodDelete, "/api/v1/pregnancy/symptoms/2026-09-22", nil)
	e.analysisBumps(0, uid, tok, 404, fiber.MethodDelete, "/api/v1/pregnancy/symptoms/2026-09-22", nil)
	e.analysisBumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/fetal-movement", map[string]any{
		"log_date": "2026-09-22", "pregnancy_week": 16, "movement_status": "not_felt_yet",
	})

	// v2 day log: symptoms, extras and the weekly weight in one transaction — one bump.
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/pregnancy/v2/days/2026-09-22", map[string]any{
		"mood": 4, "weight": 64.8, "symptoms": map[string]any{"nausea": "mild", "heartburn": "mild"},
	})

	// The kick counter writes the day's fetal-movement row when a session stops.
	r := e.analysisBumps(0, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/kick-sessions", nil)
	kid := id(t, r.data()["id"])
	e.analysisBumps(0, uid, tok, 200, fiber.MethodPost, "/api/v1/pregnancy/kick-sessions/"+kid+"/kicks", nil)
	e.analysisBumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/pregnancy/kick-sessions/"+kid+"/stop", nil)
	e.analysisBumps(1, uid, tok, 200, fiber.MethodDelete, "/api/v1/pregnancy/kick-sessions/"+kid, nil)

	e.analysisBumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/pregnancy/weekly", map[string]any{"pregnancy_week": 99})
}
