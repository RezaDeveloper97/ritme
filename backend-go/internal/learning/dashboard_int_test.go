package learning_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ins_Dashboard (B-N8-06): owner-scoped counts; a student counts as active only after she opened a course (H1).
func TestInstructorDashboard(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	otok := e.instructor(t, "09120000002", "Other")
	cid, lessons := e.publishedCourse(t, itok, "Course", 2)
	r := e.do(t, http.MethodPost, "/api/instructor/v1/courses", itok, map[string]any{"kind": "course", "title": "Draft"})
	require.Equal(t, 201, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/instructor/v1/groups", itok, map[string]any{"name": "Mehr", "course_ids": []int{cid}})
	require.Equal(t, 201, r.status, r.raw)
	gid := idOf(t, r.data()["group"].(map[string]any)["id"])

	_, stok := e.user(t, "09121110001")
	r = e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, map[string]any{
		"phones": []string{"09121110001", "09121110002"}, "scope": "group", "target_id": gid, "duration": "unlimited",
	})
	require.Equal(t, 201, r.status, r.raw)

	stats := func(tok string) map[string]any {
		t.Helper()
		r := e.do(t, http.MethodGet, "/api/instructor/v1/dashboard", tok, nil)
		require.Equal(t, 200, r.status, r.raw)
		assert.NotContains(t, r.raw, "09121110001")
		return r.data()["stats"].(map[string]any)
	}
	s := stats(itok)
	assert.EqualValues(t, 0, s["active_students"]) // not seen yet: never reveals a registered number
	assert.EqualValues(t, 2, s["new_students_week"])
	assert.EqualValues(t, 2, s["courses"])
	assert.EqualValues(t, 1, s["drafts"])
	assert.EqualValues(t, 1, s["groups"])
	assert.EqualValues(t, 2, s["group_members"])

	// The student opens the course and watches: active + watch time.
	require.Equal(t, 200, e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/courses/%d", cid), stok, nil).status)
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/learning/lessons/%d/progress", lessons[0]), stok, map[string]any{"position_seconds": 300})
	require.Equal(t, 200, r.status, r.raw)
	s = stats(itok)
	assert.EqualValues(t, 1, s["active_students"])
	assert.EqualValues(t, 300, s["watch_seconds_week"])

	r = e.do(t, http.MethodGet, "/api/instructor/v1/dashboard", itok, nil)
	assert.Len(t, r.data()["courses"], 2)
	kinds := map[string]any{}
	for _, c := range r.data()["courses"].([]any) {
		m := c.(map[string]any)
		kinds[m["title"].(string)] = m["lesson_kinds"]
	}
	assert.Equal(t, []any{"video"}, kinds["Course"])
	assert.Equal(t, []any{}, kinds["Draft"])
	act := r.data()["activity"].([]any)
	require.NotEmpty(t, act)
	types := ""
	for _, a := range act {
		types += a.(map[string]any)["type"].(string) + ","
	}
	assert.True(t, strings.Contains(types, "lesson_published") && strings.Contains(types, "students_added"), types)

	// Another instructor sees only her own (empty) numbers.
	o := stats(otok)
	for _, k := range []string{"active_students", "new_students_week", "courses", "watch_seconds_week", "groups"} {
		assert.EqualValues(t, 0, o[k], k)
	}

	// Guards: 401 without a token, 403 for a non-instructor.
	assert.Equal(t, 401, e.do(t, http.MethodGet, "/api/instructor/v1/dashboard", "", nil).status)
	assert.Equal(t, 403, e.do(t, http.MethodGet, "/api/instructor/v1/dashboard", stok, nil).status)
}
