package checkups_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/checkups"
)

const physio = `{"title":"فیزیوتراپی","interval_months":6,"performed_by":"doctor"}`

func TestCustomCheckup_Cap(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000951", "1995-01-01")
	_, other := e.user(t, "09120000952", "1995-01-01")

	rows := make([]string, checkups.MaxCustomCheckups)
	args := make([]any, 0, len(rows))
	for i := range rows {
		rows[i] = "(?, 'custom', '{\"fa\":\"x\"}', 'doctor', 'neutral', 12, 1, 1000, '2026-09-01 09:00:00', '2026-09-01 09:00:00')"
		args = append(args, uid)
	}
	_, err := e.db.Exec(`INSERT INTO checkup_types (user_id, category, title, performed_by, tone, interval_months, is_active, sort_order, created_at, updated_at)
		VALUES `+strings.Join(rows, ","), args...)
	require.NoError(t, err)

	for lang, msg := range map[string]string{
		"fa": checkups.T("messages.custom_limit", "fa"),
		"en": "You have reached the maximum number of custom checkups. Delete one you no longer need to add a new one.",
	} {
		r := e.do(t, http.MethodPost, "/api/v1/checkups/custom", tok, lang, physio)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
		assert.Equal(t, false, r.body["success"])
		assert.Equal(t, msg, r.body["message"])
		assert.Equal(t, checkups.ErrorCodeLimitReached, r.body["error_code"])
		assert.Equal(t, map[string]any{"limit": []any{msg}}, r.body["errors"])
	}
	assert.Contains(t, checkups.T("messages.custom_limit", "fa"), "سقف تعداد")
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM checkup_types WHERE user_id = ?`, uid).Scan(&n))
	assert.Equal(t, checkups.MaxCustomCheckups, n)

	// Per user: another account still adds hers.
	r := e.do(t, http.MethodPost, "/api/v1/checkups/custom", other, "fa", physio)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
}
