package assistant_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/assistant"
)

func TestDepartmentsAdmin_SuperOnlyReviewAndAudit(t *testing.T) {
	e := admintest.New(t)
	var logs bytes.Buffer
	assistant.NewDepartmentsAdmin(e.DB, slog.New(slog.NewJSONHandler(&logs, nil))).Routes(e.Route(), e.Kit)
	e.Exec(`INSERT INTO assistant_departments (code, sort_order, is_active, name, system_prompt, needs_review, created_at, updated_at)
		VALUES ('t_dept', 1, 1, '{"fa":"آزمون"}', 'Be kind.', 0, NOW(), NOW())`)
	id := e.Int(`SELECT id FROM assistant_departments WHERE code = 't_dept'`)
	p := fmt.Sprintf("/assistant/departments/%d", id)

	// 401 anonymous, 403 for an editor on every route.
	assert.Equal(t, http.StatusUnauthorized, e.Anonymous().Get("/assistant/departments").Status)
	for _, tc := range []struct{ m, p string }{
		{http.MethodGet, "/assistant/departments"}, {http.MethodGet, p}, {http.MethodPut, p},
		{http.MethodPost, p + "/approve"}, {http.MethodPost, "/assistant/departments"},
	} {
		assert.Equal(t, http.StatusForbidden, e.As(admintest.EditorID).JSON(tc.m, tc.p, map[string]any{}).Status, tc.p)
	}

	su := e.As(admintest.SuperID)
	body := func(prompt string, active bool) map[string]any {
		return map[string]any{"name": map[string]any{"fa": "آزمون"}, "system_prompt": prompt, "is_active": active,
			"quick_replies": map[string]any{"fa": []any{"سلام"}}}
	}
	// Length caps.
	long := make([]byte, assistant.MaxPromptLen+1)
	for i := range long {
		long[i] = 'a'
	}
	r := su.JSON(http.MethodPut, p, body(string(long), true))
	assert.Equal(t, http.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, r.Errors(), "system_prompt")

	// A prompt change flags needs_review (the live department stays on) and is audit-logged without the text.
	r = su.JSON(http.MethodPut, p, body("Be very kind.", true))
	require.Equal(t, http.StatusOK, r.Status, r.Body)
	assert.Equal(t, true, r.Obj("department")["needs_review"])
	assert.Contains(t, logs.String(), `"audit":"assistant_department.update"`)
	assert.NotContains(t, logs.String(), "Be very kind.")

	// Off, then on again: refused until approved.
	require.Equal(t, http.StatusOK, su.JSON(http.MethodPut, p, body("Be very kind.", false)).Status)
	r = su.JSON(http.MethodPut, p, body("Be very kind.", true))
	assert.Equal(t, http.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, r.Errors(), "is_active")
	require.Equal(t, http.StatusOK, su.JSON(http.MethodPost, p+"/approve", nil).Status)
	assert.Contains(t, logs.String(), `"audit":"assistant_department.approve"`)
	r = su.JSON(http.MethodPut, p, body("Be very kind.", true))
	require.Equal(t, http.StatusOK, r.Status, r.Body)
	assert.Equal(t, false, r.Obj("department")["needs_review"])

	// Create: inactive + unreviewed; duplicate code refused.
	c := su.JSON(http.MethodPost, "/assistant/departments", map[string]any{"code": "new_dept",
		"name": map[string]any{"fa": "جدید"}, "system_prompt": "Hi."})
	require.Equal(t, http.StatusCreated, c.Status, c.Body)
	assert.Equal(t, false, c.Obj("department")["is_active"])
	assert.Equal(t, true, c.Obj("department")["needs_review"])
	c = su.JSON(http.MethodPost, "/assistant/departments", map[string]any{"code": "new_dept",
		"name": map[string]any{"fa": "جدید"}, "system_prompt": "Hi."})
	assert.Equal(t, http.StatusUnprocessableEntity, c.Status)
	assert.Equal(t, http.StatusNotFound, su.Get("/assistant/departments/999999").Status)
}
