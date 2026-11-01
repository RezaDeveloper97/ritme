package safety_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/safety"
)

func TestMain(m *testing.M) { testdb.Main(m) }

var quiet = admintest.Quiet

func failed(r safety.Run) map[string][]string {
	out := map[string][]string{}
	for _, res := range r.Results {
		if !res.Passed {
			out[res.Scenario] = res.Detail.Failures()
		}
	}
	return out
}

func noon() time.Time { return civildate.InTehran(time.Now()).TehranMidnight().Add(12 * time.Hour) }

// Every scenario passes on the migrated configuration, and the run is persisted (one row per scenario) and read back
// by a fresh service (a restarted process has no in-memory run).
func TestNightly_PassesAndPersists(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	db := testdb.New(t)
	ctx := context.Background()
	r := safety.NewRunner(db, quiet)
	run, ran, err := r.Nightly(ctx, noon())
	require.NoError(t, err)
	require.True(t, ran)
	assert.Empty(t, failed(run))
	assert.Equal(t, len(safety.Scenarios()), run.Total)

	var rows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM safety_scenario_runs WHERE run_id = ?", run.ID).Scan(&rows))
	assert.Equal(t, run.Total, rows)

	// Idempotent on the Tehran day: the second call writes nothing.
	_, ran, err = r.Nightly(ctx, noon().Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, ran)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM safety_scenario_runs").Scan(&rows))
	assert.Equal(t, run.Total, rows)

	svc := safety.New(db)
	runs, err := svc.Runs(ctx, safety.RunsShown)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, run.ID, runs[0].RunID)
	assert.Equal(t, 0, runs[0].Failed)
	assert.Equal(t, noon().Format(time.DateTime), runs[0].StartedAt.Format(time.DateTime))
	last, err := svc.LatestRun(ctx)
	require.NoError(t, err)
	assert.Equal(t, run.Total, last.Passed)
	results, err := svc.RunResults(ctx, run.ID)
	require.NoError(t, err)
	var d safety.Detail
	require.NoError(t, json.Unmarshal(results[0].Detail.V, &d))
	assert.NotEmpty(t, d.Threshold)
}

// A deactivated locked rule fails exactly config.locked_rules: the path scenarios still pass on the code-held
// definition (the emergency guarantee) and its copy still calls 115.
func TestNightly_DeactivatedLockedRule(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	db := testdb.New(t)
	_, err := db.Exec("UPDATE message_rules SET is_active = 0 WHERE rule_key = 'preg.bp_severe'")
	require.NoError(t, err)
	run, err := safety.NewRunner(db, quiet).Run(context.Background(), noon())
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"config.locked_rules": {"preg.bp_severe: inactive"}}, failed(run))
}

// Copy without 115 fails exactly config.emergency_copy.
func TestNightly_CopyWithout115(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	db := testdb.New(t)
	var content string
	require.NoError(t, db.QueryRow("SELECT content FROM message_rules WHERE rule_key = 'preg.contractions_511'").Scan(&content))
	var c map[string]any
	require.NoError(t, json.Unmarshal([]byte(content), &c))
	tr, _ := c["translations"].(map[string]any)
	en, _ := tr["en"].(map[string]any)
	require.NotNil(t, en)
	en["contact"] = "Call your midwife."
	b, err := json.Marshal(c)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE message_rules SET content = ? WHERE rule_key = 'preg.contractions_511'", string(b))
	require.NoError(t, err)
	run, err := safety.NewRunner(db, quiet).Run(context.Background(), noon())
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"config.emergency_copy": {"preg.contractions_511: en text without 115"}}, failed(run))
}

// A registry boundary moved outside the key's range fails exactly config.registry; a valid move only moves the probes.
func TestNightly_RegistryBoundary(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	db := testdb.New(t)
	from := civildate.InTehran(noon()).AddDays(-1)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyBPCrisis, Version: 2,
		Value: clinicalconfig.Value{"systolic": 170, "diastolic": 115}, EffectiveFrom: from}})
	run, err := safety.NewRunner(db, quiet).Run(context.Background(), noon())
	require.NoError(t, err)
	assert.Empty(t, failed(run))

	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyBPCrisis, Version: 3,
		Value: clinicalconfig.Value{"systolic": 90, "diastolic": 115}, EffectiveFrom: from}})
	run, err = safety.NewRunner(db, quiet).Run(context.Background(), noon().Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"config.registry": {"bp.crisis v3: range:systolic"}}, failed(run))
}

func TestRunsAPI(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	e := admintest.New(t)
	safety.NewRunsHandlers(e.DB).Routes(e.Route(), e.Kit)
	assert.Equal(t, 401, e.Anonymous().Get("/safety/runs").Status)
	empty := e.As(admintest.SuperID).Get("/safety/runs")
	require.Equal(t, 200, empty.Status)
	assert.Empty(t, empty.Data()["runs"])

	_, err := e.DB.Exec("UPDATE message_rules SET is_active = 0 WHERE rule_key = 'preg.fetal_movement'")
	require.NoError(t, err)
	run, err := safety.NewRunner(e.DB, quiet).Run(context.Background(), noon())
	require.NoError(t, err)

	list := e.As(admintest.SuperID).Get("/safety/runs")
	require.Equal(t, 200, list.Status)
	runs, _ := list.Data()["runs"].([]any)
	require.Len(t, runs, 1)
	r0, _ := runs[0].(map[string]any)
	assert.Equal(t, run.ID, r0["run_id"])
	assert.EqualValues(t, 1, r0["failed"])
	assert.EqualValues(t, run.Total-1, r0["passed"])
	assert.EqualValues(t, 30, list.Data()["limit"])

	show := e.As(admintest.SuperID).Get("/safety/runs/" + run.ID)
	require.Equal(t, 200, show.Status)
	results, _ := show.Data()["results"].([]any)
	require.Len(t, results, run.Total)
	first, _ := results[0].(map[string]any)
	assert.Equal(t, "config.locked_rules", first["scenario"])
	assert.Equal(t, "config", first["kind"])
	assert.Equal(t, false, first["passed"])
	detail, _ := first["detail"].(map[string]any)
	assert.Equal(t, []any{"preg.fetal_movement: inactive"}, detail["problems"])

	assert.Equal(t, 404, e.As(admintest.SuperID).Get("/safety/runs/01ARZ3NDEKTSV4RRFFQ69G5FAV").Status)
	assert.Equal(t, 404, e.As(admintest.SuperID).Get("/safety/runs/nope").Status)
}
