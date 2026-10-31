package content_test

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
)

func v2Body() map[string]any {
	return map[string]any{
		"title": map[string]any{"fa": "کشش ۷ روزه"}, "is_active": true,
		"duration_days": 7, "start_rule": "anytime", "reminder_time": "21:30",
		"modes_include": []string{"cycle", "ttc"}, "phases_include": []string{"luteal", "menstrual"},
		"modes_exclude":     []string{"pregnancy", "postpartum"},
		"reward_badge_code": "calm", "reward_badge_title": map[string]any{"fa": "ریتم آرام"}, "reward_plus_days": 1,
		"completion_min_days": 5,
		"tasks": []map[string]any{
			{"day_from": 1, "day_to": 7, "title": map[string]any{"fa": "کشش"}},
			{"day_from": 4, "day_to": 7, "title": map[string]any{"fa": "تنفس"}},
		},
	}
}

func TestChallengesV2_EditorAndStats(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)

	r := c.JSON(fiber.MethodPost, "/challenges", v2Body())
	require.Equal(t, 201, r.Status, r.Body)
	ch := r.Obj("challenge")
	prog := ch["program"].(map[string]any)
	assert.EqualValues(t, 7, prog["duration_days"])
	assert.Equal(t, "21:30", prog["reminder_time"])
	assert.Equal(t, []any{"luteal", "menstrual"}, prog["phases_include"])
	assert.Equal(t, []any{"pregnancy", "postpartum"}, prog["modes_exclude"])
	assert.EqualValues(t, 1, prog["reward_plus_days"])
	assert.EqualValues(t, 5, prog["completion_min_days"])
	tasks := ch["tasks"].([]any)
	require.Len(t, tasks, 2)
	cid := ch["id"].(float64)
	first := tasks[0].(map[string]any)["id"].(float64)

	// invalid: no tasks, a task past the duration, a bad phase, fixed_date without a date, minimum past the duration
	bad := v2Body()
	bad["tasks"] = []map[string]any{{"day_from": 1, "day_to": 9, "title": map[string]any{"fa": "x"}}}
	bad["phases_include"] = []string{"nope"}
	bad["completion_min_days"] = 8
	bad["start_rule"] = "fixed_date"
	r = c.JSON(fiber.MethodPost, "/challenges", bad)
	require.Equal(t, 422, r.Status, r.Body)
	for _, f := range []string{"tasks.0.day_to", "phases_include.0", "completion_min_days", "starts_on"} {
		assert.Contains(t, r.Errors(), f)
	}
	empty := v2Body()
	delete(empty, "tasks")
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/challenges", empty).Status, "a run needs tasks")

	// update: keep task 1 (edited), drop task 2, add a new one
	upd := v2Body()
	upd["tasks"] = []map[string]any{
		{"id": first, "day_from": 1, "day_to": 3, "title": map[string]any{"fa": "کشش جدید"}},
		{"day_from": 5, "day_to": 7, "title": map[string]any{"fa": "پیاده روی"}},
	}
	r = c.JSON(fiber.MethodPut, "/challenges/"+idStr(cid), upd)
	require.Equal(t, 200, r.Status, r.Body)
	tasks = r.Obj("challenge")["tasks"].([]any)
	require.Len(t, tasks, 2)
	assert.Equal(t, first, tasks[0].(map[string]any)["id"], "the task keeps its id")
	assert.EqualValues(t, 3, tasks[0].(map[string]any)["day_to"])

	// a legacy update that leaves the program fields out keeps the program
	r = c.JSON(fiber.MethodPut, "/challenges/"+idStr(cid), map[string]any{"title": map[string]any{"fa": "عنوان"}})
	require.Equal(t, 200, r.Status, r.Body)
	assert.NotNil(t, r.Obj("challenge")["program"])
	assert.Len(t, r.Obj("challenge")["tasks"], 2)

	// stats: 2 participants, 1 completed (+ its reward)
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (7, 'Sara', '09120000007', NOW(), NOW()), (8, 'Mina', '09120000008', NOW(), NOW())`)
	e.Exec(`INSERT INTO challenge_enrollments (id, user_id, challenge_id, started_on, status, completed_at, created_at, updated_at) VALUES
		(1, 7, ?, '2026-10-01', 'completed', NOW(), NOW(), NOW()), (2, 8, ?, '2026-10-02', 'active', NULL, NOW(), NOW())`, cid, cid)
	e.Exec(`INSERT INTO challenge_task_completions (enrollment_id, task_id, day_number, completed_at) VALUES (1, ?, 1, NOW()), (2, ?, 1, NOW()), (2, ?, 2, NOW())`, first, first, first)
	e.Exec(`INSERT INTO challenge_reward_grants (user_id, challenge_id, enrollment_id, plus_days, granted_at) VALUES (7, ?, 1, 1, NOW())`, cid)
	r = c.Get("/challenges/" + idStr(cid) + "/stats")
	require.Equal(t, 200, r.Status, r.Body)
	s := r.Data()["stats"].(map[string]any)
	assert.EqualValues(t, 2, s["participants"])
	assert.EqualValues(t, 1, s["completed"])
	assert.EqualValues(t, 50, s["completion_rate"])
	assert.EqualValues(t, 1, s["rewards_granted"])
	assert.EqualValues(t, 1, s["plus_days_granted"])
	days := s["per_day"].([]any)
	require.Len(t, days, 2)
	assert.EqualValues(t, 2, days[0].(map[string]any)["users"])

	// the list carries program + stats; a legacy challenge has program null
	e.Exec(`INSERT INTO challenges (id, title, is_active, sort_order, created_at, updated_at) VALUES (900, '{"fa":"قدیمی"}', 1, 5, NOW(), NOW())`)
	r = c.Get("/challenges")
	require.Equal(t, 200, r.Status, r.Body)
	for _, it := range r.Items() {
		m := it.(map[string]any)
		if m["id"].(float64) == 900 {
			assert.Nil(t, m["program"])
		} else {
			assert.EqualValues(t, 2, m["stats"].(map[string]any)["participants"])
		}
	}
	assert.Equal(t, 404, c.Get("/challenges/9999/stats").Status)
}

func idStr(f float64) string { return strconv.Itoa(int(f)) }

// T-M10-10 (D-167): the v6 rule fields — eligibility / contraindications in the {all_of, any_of, exclude} shape,
// on_phase_start + start_phase, the legacy lists converted on save, the day-by-day drop-off.
func TestChallengesV2_RulesV6(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)

	// a legacy editor save is stored in the v6 shape and still echoes its lists; anytime is stored as on_join
	r := c.JSON(fiber.MethodPost, "/challenges", v2Body())
	require.Equal(t, 201, r.Status, r.Body)
	prog := r.Obj("challenge")["program"].(map[string]any)
	assert.Equal(t, "on_join", prog["start_rule"])
	assert.Equal(t, []any{"cycle", "ttc"}, prog["modes_include"])
	elig := prog["eligibility"].(map[string]any)
	assert.Len(t, elig["all_of"], 2)
	assert.Len(t, elig["exclude"], 1)
	assert.Nil(t, prog["contraindications"])
	var stored string
	require.NoError(t, e.DB.QueryRow(`SELECT eligibility FROM challenge_programs WHERE challenge_id = ?`, r.Obj("challenge")["id"]).Scan(&stored))
	assert.Contains(t, stored, `"all_of"`)
	assert.NotContains(t, stored, "modes_include")

	// v6 body
	body := v2Body()
	for _, k := range []string{"modes_include", "modes_exclude", "phases_include"} {
		delete(body, k)
	}
	body["start_rule"] = "on_phase_start"
	body["start_phase"] = "luteal"
	body["eligibility"] = map[string]any{"all_of": []any{map[string]any{"fact": "life_mode", "op": "in", "value": []string{"cycle", "postpartum"}}}}
	body["contraindications"] = map[string]any{"any_of": []any{[]any{
		map[string]any{"fact": "mode", "op": "eq", "value": "pregnancy"},
		map[string]any{"fact": "postpartum_week", "op": "lte", "value": 6},
	}}}
	r = c.JSON(fiber.MethodPost, "/challenges", body)
	require.Equal(t, 201, r.Status, r.Body)
	prog = r.Obj("challenge")["program"].(map[string]any)
	cid := r.Obj("challenge")["id"].(float64)
	assert.Equal(t, "on_phase_start", prog["start_rule"])
	assert.Equal(t, "luteal", prog["start_phase"])
	assert.Len(t, prog["contraindications"].(map[string]any)["any_of"], 1)
	assert.Equal(t, []any{"cycle", "postpartum"}, prog["modes_include"], "expressible as the legacy list")

	// an older editor's update (no contraindications key) keeps them
	upd := v2Body()
	r = c.JSON(fiber.MethodPut, "/challenges/"+idStr(cid), upd)
	require.Equal(t, 200, r.Status, r.Body)
	assert.NotNil(t, r.Obj("challenge")["program"].(map[string]any)["contraindications"], "kept")
	assert.Nil(t, r.Obj("challenge")["program"].(map[string]any)["start_phase"], "on_join has no phase")

	// invalid: unknown fact, bad value, on_phase_start without a phase, unknown start phase
	bad := v2Body()
	bad["eligibility"] = map[string]any{"all_of": []any{map[string]any{"fact": "nope", "op": "eq", "value": 1}}}
	bad["contraindications"] = map[string]any{"all_of": []any{map[string]any{"fact": "postpartum_week", "op": "lte", "value": "six"}}}
	bad["start_rule"] = "on_phase_start"
	r = c.JSON(fiber.MethodPost, "/challenges", bad)
	require.Equal(t, 422, r.Status, r.Body)
	for _, f := range []string{"eligibility.all_of.0", "contraindications.all_of.0", "start_phase"} {
		assert.Contains(t, r.Errors(), f)
	}
	bad = v2Body()
	bad["start_phase"] = "unknown"
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/challenges", bad).Status)

	// drop-off: 2 runs started long ago (the whole 7 days reached); one did days 1-2, one nothing
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (7, 'Sara', '09120000007', NOW(), NOW()), (8, 'Mina', '09120000008', NOW(), NOW())`)
	e.Exec(`INSERT INTO challenge_enrollments (id, user_id, challenge_id, started_on, status, created_at, updated_at) VALUES
		(1, 7, ?, '2026-01-01', 'active', NOW(), NOW()), (2, 8, ?, '2026-01-01', 'active', NOW(), NOW())`, cid, cid)
	var task float64
	require.NoError(t, e.DB.QueryRow(`SELECT MIN(id) FROM challenge_tasks WHERE challenge_id = ?`, cid).Scan(&task))
	e.Exec(`INSERT INTO challenge_task_completions (enrollment_id, task_id, day_number, completed_at) VALUES (1, ?, 1, NOW()), (1, ?, 2, NOW())`, task, task)
	r = c.Get("/challenges/" + idStr(cid) + "/stats")
	require.Equal(t, 200, r.Status, r.Body)
	drop := r.Data()["stats"].(map[string]any)["drop_off"].([]any)
	require.Len(t, drop, 7)
	d1, d3 := drop[0].(map[string]any), drop[2].(map[string]any)
	assert.EqualValues(t, 2, d1["reached"])
	assert.EqualValues(t, 50, d1["retained_rate"])
	assert.EqualValues(t, 50, d1["drop_off_rate"])
	assert.EqualValues(t, 0, d3["retained_rate"])
	assert.EqualValues(t, 50, d3["drop_off_rate"], "Sara dropped on day 3")
}
