package profile_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	authstore "github.com/ritme/backend-go/internal/auth/store"
)

// freshUser is a user exactly as verify-otp creates it: mobile only, no name, no profile.
func (e *env) freshUser(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (mobile, created_at, updated_at) VALUES (?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

func (e *env) profileCompleted(t *testing.T, id uint64) bool {
	t.Helper()
	q := authstore.New(e.db)
	u, err := q.GetUserByID(context.Background(), id)
	require.NoError(t, err)
	done, err := auth.ProfileCompleted(context.Background(), q, &u)
	require.NoError(t, err)
	return done
}

func obData(t *testing.T, r response) map[string]any {
	t.Helper()
	require.Equal(t, 200, r.status, r.raw)
	data, ok := r.body["data"].(map[string]any)
	require.True(t, ok, r.raw)
	return data
}

func TestOnboarding_WomenPathEndToEnd(t *testing.T) {
	e := setup(t)
	id, tok := e.freshUser(t, "09120000201")

	r := e.do(t, "GET", "/api/v1/onboarding", tok, "")
	d := obData(t, r)
	assert.Nil(t, d["name"])
	assert.Nil(t, d["started_at"])
	assert.Equal(t, false, d["completed"])
	assert.Equal(t, "cycle", d["life_stage"].(map[string]any)["mode"])
	assert.False(t, e.profileCompleted(t, id))

	steps := []struct{ step, body string }{
		{"name", `{"name":"سارا"}`},
		{"gender", `{"gender":"female"}`},
		{"goal", `{"goal":"ttc"}`},
		{"cycle", `{"last_period_start":"2026-09-10","period_duration":5,"cycle_duration":29}`},
		{"conditions", `{"chronic_illnesses":["migraine","diabetes","migraine"],"gyn_conditions":[],"medications":null}`},
		{"health", `{"birthday":"1993-03-05","height":164,"weight":58}`},
	}
	for _, s := range steps {
		r = e.do(t, "PUT", "/api/v1/onboarding/steps/"+s.step, tok, s.body)
		require.Equal(t, 200, r.status, s.step+": "+r.raw)
		// Mid-flow the user has a name and a profile row, but onboarding is not finished.
		assert.False(t, e.profileCompleted(t, id), s.step)
	}
	d = obData(t, r)
	assert.Equal(t, "سارا", d["name"])
	assert.Equal(t, "female", d["gender"])
	assert.Equal(t, "ttc", d["goal"])
	assert.Equal(t, map[string]any{"mode": "ttc", "stored_mode": "ttc", "ivf_iui": false, "track_contraception": false}, d["life_stage"])
	assert.Equal(t, map[string]any{"last_period_start": "2026-09-10", "period_duration": float64(5), "cycle_duration": float64(29)}, d["cycle"])
	cond := d["conditions"].(map[string]any)
	assert.Equal(t, []any{"diabetes", "migraine"}, cond["chronic_illnesses"]) // screen order, deduplicated
	assert.Equal(t, []any{}, cond["gyn_conditions"])                          // «هیچ‌کدام»
	assert.Nil(t, cond["medications"])                                        // skipped
	assert.Equal(t, map[string]any{"birthday": "1993-03-05", "height": float64(164), "weight": float64(58)}, d["health"])
	assert.NotNil(t, d["started_at"])

	// Legacy columns kept in step: user_goal ttc + intention trying; the cycle step seeded the period log.
	var goal, intention string
	require.NoError(t, e.db.QueryRow(`SELECT user_goal, pregnancy_intention FROM user_profiles WHERE user_id = ?`, id).Scan(&goal, &intention))
	assert.Equal(t, [2]string{"ttc", "trying"}, [2]string{goal, intention})
	var seeds int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM cycle_histories WHERE user_id = ? AND period_start_date = '2026-09-10'`, id).Scan(&seeds))
	assert.Equal(t, 1, seeds)

	r = e.do(t, "POST", "/api/v1/onboarding/complete", tok, "")
	d = obData(t, r)
	assert.Equal(t, true, d["completed"])
	first := d["completed_at"]
	assert.True(t, e.profileCompleted(t, id))

	// Complete is idempotent (first time kept); so is a step.
	r = e.do(t, "POST", "/api/v1/onboarding/complete", tok, "")
	assert.Equal(t, first, obData(t, r)["completed_at"])
	before := obData(t, e.do(t, "GET", "/api/v1/onboarding", tok, ""))
	e.do(t, "PUT", "/api/v1/onboarding/steps/conditions", tok, `{"chronic_illnesses":["migraine","diabetes","migraine"],"gyn_conditions":[],"medications":null}`)
	after := obData(t, e.do(t, "GET", "/api/v1/onboarding", tok, ""))
	assert.Equal(t, before, after)
}

func TestOnboarding_MenopauseAndLifeStage(t *testing.T) {
	e := setup(t)
	id, tok := e.freshUser(t, "09120000202")

	r := e.do(t, "PUT", "/api/v1/onboarding/steps/goal", tok, `{"goal":"menopause"}`)
	assert.Equal(t, "menopause", obData(t, r)["life_stage"].(map[string]any)["mode"])
	r = e.do(t, "PUT", "/api/v1/onboarding/steps/menopause", tok, `{"stage":"peri","last_period":"2025-08-01","surgical":false,"hrt":null}`)
	assert.Equal(t, map[string]any{"stage": "peri", "last_period": "2025-08-01", "surgical": false, "hrt": nil}, obData(t, r)["menopause"])
	// Replacing the step clears what is no longer sent.
	r = e.do(t, "PUT", "/api/v1/onboarding/steps/menopause", tok, `{"stage":"unsure"}`)
	assert.Equal(t, map[string]any{"stage": "unsure", "last_period": nil, "surgical": nil, "hrt": nil}, obData(t, r)["menopause"])

	var goal string
	require.NoError(t, e.db.QueryRow(`SELECT user_goal FROM user_profiles WHERE user_id = ?`, id).Scan(&goal))
	assert.Equal(t, "non_ttc", goal)

	// Life stage: switches, then teen; a TTC switch syncs user_goal.
	r = e.do(t, "PUT", "/api/v1/profile/life-stage", tok, `{"mode":"ttc","ivf_iui":true,"track_contraception":true}`)
	assert.Equal(t, map[string]any{"mode": "ttc", "stored_mode": "ttc", "ivf_iui": true, "track_contraception": true}, obData(t, r))
	require.NoError(t, e.db.QueryRow(`SELECT user_goal FROM user_profiles WHERE user_id = ?`, id).Scan(&goal))
	assert.Equal(t, "ttc", goal)
	r = e.do(t, "PUT", "/api/v1/profile/life-stage", tok, `{"mode":"teen"}`)
	assert.Equal(t, map[string]any{"mode": "teen", "stored_mode": "teen", "ivf_iui": true, "track_contraception": true}, obData(t, r))
	require.NoError(t, e.db.QueryRow(`SELECT user_goal FROM user_profiles WHERE user_id = ?`, id).Scan(&goal))
	assert.Equal(t, "non_ttc", goal)

	// An active pregnancy profile wins over the stored mode.
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, created_at, updated_at) VALUES (?, 1, NOW(), NOW())`, id)
	require.NoError(t, err)
	r = e.do(t, "GET", "/api/v1/profile/life-stage", tok, "")
	assert.Equal(t, "pregnancy", obData(t, r)["mode"])
	assert.Equal(t, "teen", obData(t, r)["stored_mode"])
}

func TestOnboarding_ExistingUserUnchanged(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000203") // named
	// A legacy user: profile from the old onboarding, TTC.
	r := e.do(t, "POST", "/api/v1/profile", tok, `{"user_goal":"ttc","period_duration":5,"cycle_duration":28}`)
	require.Equal(t, 200, r.status, r.raw)
	require.True(t, e.profileCompleted(t, id))

	r = e.do(t, "GET", "/api/v1/profile/life-stage", tok, "")
	assert.Equal(t, map[string]any{"mode": "ttc", "stored_mode": nil, "ivf_iui": false, "track_contraception": false}, obData(t, r))
	var rows int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM user_life_profiles WHERE user_id = ?`, id).Scan(&rows))
	assert.Zero(t, rows, "reads never create a row")

	// Replaying an onboarding step does not send a finished user back to onboarding.
	r = e.do(t, "PUT", "/api/v1/onboarding/steps/gender", tok, `{"gender":"female"}`)
	assert.Equal(t, true, obData(t, r)["completed"])
	assert.True(t, e.profileCompleted(t, id))
	// Nor does the life-stage switch.
	e.do(t, "PUT", "/api/v1/profile/life-stage", tok, `{"track_contraception":true}`)
	assert.True(t, e.profileCompleted(t, id))

	// The export carries the row.
	r = e.do(t, "GET", "/api/v1/profile/export", tok, "")
	lp := obData(t, r)["life_profile"].(map[string]any)
	assert.Equal(t, "female", lp["gender"])
	assert.Equal(t, true, lp["track_contraception"])
}

func TestOnboarding_ValidationIsolationAnd401(t *testing.T) {
	e := setup(t)
	_, alice := e.freshUser(t, "09120000204")
	bobID, bob := e.freshUser(t, "09120000205")

	r := e.do(t, "PUT", "/api/v1/onboarding/steps/gender", alice, `{"gender":"other"}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Contains(t, r.body["errors"], "gender")
	r = e.do(t, "PUT", "/api/v1/onboarding/steps/conditions", alice, `{"chronic_illnesses":["pcos"],"medications":"iud"}`)
	require.Equal(t, 422, r.status, r.raw)
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "chronic_illnesses.0")
	assert.Contains(t, errs, "medications")
	r = e.do(t, "PUT", "/api/v1/onboarding/steps/cycle", alice, `{"cycle_duration":99,"last_period_start":"2030-01-01"}`)
	require.Equal(t, 422, r.status, r.raw)
	r = e.do(t, "PUT", "/api/v1/onboarding/steps/menopause", alice, `{}`)
	require.Equal(t, 422, r.status, r.raw)
	r = e.do(t, "PUT", "/api/v1/profile/life-stage", alice, `{"mode":"retired","ivf_iui":"x"}`)
	require.Equal(t, 422, r.status, r.raw)
	r = e.do(t, "PUT", "/api/v1/onboarding/steps/nope", alice, `{}`)
	assert.Equal(t, 404, r.status, r.raw)

	// Complete without answers names the missing steps.
	r = e.do(t, "POST", "/api/v1/onboarding/complete", alice, "")
	require.Equal(t, 422, r.status, r.raw)
	errs = r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "name")
	assert.Contains(t, errs, "gender")
	e.do(t, "PUT", "/api/v1/onboarding/steps/gender", alice, `{"gender":"female"}`)
	r = e.do(t, "POST", "/api/v1/onboarding/complete", alice, "")
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "goal")

	// Alice's answers never reach Bob.
	e.do(t, "PUT", "/api/v1/onboarding/steps/goal", alice, `{"goal":"menopause"}`)
	d := obData(t, e.do(t, "GET", "/api/v1/onboarding", bob, ""))
	assert.Nil(t, d["gender"])
	assert.Nil(t, d["goal"])
	var rows int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM user_life_profiles WHERE user_id = ?`, bobID).Scan(&rows))
	assert.Zero(t, rows)

	for _, c := range []struct{ method, path string }{
		{"GET", "/api/v1/onboarding"}, {"PUT", "/api/v1/onboarding/steps/name"}, {"POST", "/api/v1/onboarding/complete"},
		{"GET", "/api/v1/profile/life-stage"}, {"PUT", "/api/v1/profile/life-stage"},
	} {
		r = e.do(t, c.method, c.path, "nope", `{}`)
		assert.Equal(t, 401, r.status, c.path)
		assert.Equal(t, "unauthenticated", r.body["error_code"], c.path)
	}
}
