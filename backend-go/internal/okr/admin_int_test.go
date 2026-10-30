package okr_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/overview"
	"github.com/ritme/backend-go/internal/metrics"
	"github.com/ritme/backend-go/internal/okr"
)

const snapshot = `{"version":1,"dau":{"value":120,"prev":100},"mau":{"value":900,"prev":null},"stickiness_pct":{"value":13.3,"prev":null},
"signups_today":{"value":5,"prev":null},"plus_active":{"value":40,"prev":null},"trial_to_paid":{"pct":38,"count":19,"of":50,"suppressed":false},
"mrr_toman":1500000,"modes":[],"engine":{"status":"healthy","mean_error_days":0.7,"within_2_days_pct":72,"users":12,"samples":30,"suppressed":false},
"alerts":[],"okr":null,"series":[],"notes":{}}`

func setup(t *testing.T) *admintest.Env {
	e := admintest.New(t)
	svc := metrics.New(e.DB, admintest.Quiet)
	o := okr.New(e.DB, svc)
	okr.NewAdmin(o, admintest.Quiet).Routes(e.Route(), e.Kit)
	overview.New(svc, admintest.Quiet).WithOKR(func(ctx context.Context, now time.Time) (any, error) { return o.Summary(ctx, now) }).Routes(e.Route(), e.Kit)
	e.Exec("INSERT INTO metric_snapshots (day, payload, computed_at, created_at, updated_at) VALUES (CURDATE(), ?, NOW(), NOW(), NOW())", snapshot)
	return e
}

func TestOKR_BoundProgressFromSnapshot(t *testing.T) {
	e := setup(t)
	c := e.As(admintest.EditorID)
	r := c.JSON(fiber.MethodPost, "/okr/objectives", map[string]any{"title": "Daily habit", "owner": "Team", "period": "Q-test",
		"starts_on": "2020-01-01", "ends_on": "2099-12-31"})
	require.Equal(t, 201, r.Status, r.Body)
	oid := r.Obj("objective")["id"].(float64)
	path := "/okr/objectives/" + itoa(oid) + "/key-results"

	r = c.JSON(fiber.MethodPost, path, map[string]any{"title": "Engine within 2 days", "metric_key": "engine_within_2_days_pct", "baseline": 60, "target": 80})
	require.Equal(t, 201, r.Status, r.Body)
	kr := r.Obj("key_result")
	assert.Equal(t, "live", kr["source"])
	assert.Equal(t, "pct", kr["unit"])
	assert.InDelta(t, 72, kr["current"], 0)
	assert.InDelta(t, 60, kr["progress_pct"], 0.01, "(72-60)/(80-60)")
	assert.Equal(t, "on_track", kr["status"], "far past the period start the pace floor is ~0")

	r = c.JSON(fiber.MethodPost, path, map[string]any{"title": "Manual", "baseline": 0, "target": 10, "manual_value": 10, "unit": "count"})
	require.Equal(t, 201, r.Status, r.Body)
	assert.Equal(t, "done", r.Obj("key_result")["status"])

	// Validation: bad key, equal baseline/target.
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, path, map[string]any{"title": "x", "metric_key": "nope", "baseline": 0, "target": 1}).Status)
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, path, map[string]any{"title": "x", "baseline": 5, "target": 5}).Status)

	// A new snapshot value moves the progress without touching the KR.
	e.Exec(`UPDATE metric_snapshots SET payload = JSON_SET(payload, '$.engine.within_2_days_pct', 80)`)
	r = c.Get("/okr")
	require.Equal(t, 200, r.Status)
	objs := r.Data()["objectives"].([]any)
	require.Len(t, objs, 1)
	krs := objs[0].(map[string]any)["key_results"].([]any)
	assert.Equal(t, "done", krs[0].(map[string]any)["status"])
	sum := r.Obj("summary")
	assert.EqualValues(t, 2, sum["done"])

	// The overview `okr` block is the same summary.
	r = c.Get("/overview")
	require.Equal(t, 200, r.Status)
	assert.EqualValues(t, 2, r.Obj("metrics")["okr"].(map[string]any)["done"])

	// Delete cascades.
	assert.Equal(t, 200, c.JSON(fiber.MethodDelete, "/okr/objectives/"+itoa(oid), nil).Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM okr_key_results"))
}

func TestOKR_Permissions(t *testing.T) {
	e := setup(t)
	pw := "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi"
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(3, 'Support', 'support@ritme.test', ?, 'support', 1, NOW(), NOW())`, pw)
	// T-M10-33: OKR is its own capability and support has none by default (SPEC §5.8); grant it read-only here.
	e.Exec(`INSERT INTO admin_roles (role, permissions, created_at, updated_at) VALUES ('support', '{"okr":"view"}', NOW(), NOW())`)
	body := map[string]any{"title": "O", "period": "p", "starts_on": "2026-01-01", "ends_on": "2026-03-31"}

	assert.Equal(t, 401, e.Anonymous().Get("/okr").Status)
	// «support» has okr=view: reads, cannot write.
	assert.Equal(t, 200, e.As(3).Get("/okr").Status)
	assert.Equal(t, 403, e.As(3).JSON(fiber.MethodPost, "/okr/objectives", body).Status)

	// Without the capability at all: 403 on read as well.
	e.Exec(`UPDATE admin_roles SET permissions = '{"support":"manage"}' WHERE role = 'support'`)
	assert.Equal(t, 403, e.As(3).Get("/okr").Status)
	assert.Equal(t, 403, e.As(3).JSON(fiber.MethodPost, "/okr/objectives", body).Status)
	assert.Equal(t, 201, e.As(admintest.EditorID).JSON(fiber.MethodPost, "/okr/objectives", body).Status)
}

func itoa(f float64) string { return strconv.Itoa(int(f)) }
