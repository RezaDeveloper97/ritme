package migrations_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const adminMetricsVersion int64 = 105 // 00105_admin_metrics_audit_chain.sql

var adminMetricsTables = []string{
	"user_activity_days", "metric_alert_rules", "metric_alerts", "admin_user_activity", "admin_segments",
	"admin_audit_chain_head", "safety_scenario_runs",
}

var appendOnlyTables = []string{"admin_audit_logs", "clinical_config_approvals"}

func countRows(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRowContext(context.Background(), q, args...).Scan(&n))
	return n
}

func adminMetricsObjects(t *testing.T, db *sql.DB) (tables, triggers, columns, indexes int) {
	t.Helper()
	for _, tb := range adminMetricsTables {
		tables += countRows(t, db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, tb)
	}
	triggers = countRows(t, db, `SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = DATABASE()
		AND event_object_table IN ('admin_audit_logs', 'clinical_config_approvals')`)
	columns = countRows(t, db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE()
		AND ((table_name = 'users' AND column_name = 'ref_code')
		  OR (table_name = 'admin_audit_logs' AND column_name IN ('prev_hash', 'row_hash')))`)
	indexes = countRows(t, db, `SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics WHERE table_schema = DATABASE()
		AND index_name IN ('daily_health_logs_log_date_user_id_index', 'cycle_histories_period_start_date_index',
		                   'users_created_at_index', 'users_ref_code_unique')`)
	return
}

// T-M10-25: Down removes every object and Up brings them back with the seeds; a second Up on a database where the
// Laravel twin already ran (goose bookkeeping forgotten) is a no-op that does not duplicate seeds.
func TestAdminMetricsMigration_UpDownIdempotent(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	tables, triggers, columns, indexes := adminMetricsObjects(t, db)
	assert.Equal(t, []int{len(adminMetricsTables), 4, 3, 4}, []int{tables, triggers, columns, indexes})
	assert.Equal(t, 5, countRows(t, db, `SELECT COUNT(*) FROM metric_alert_rules WHERE enabled = 1`))
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM metric_alert_rules
		WHERE metric_key = 'message_ctr' AND comparator = 'drop_vs_mean' AND threshold = 0.3 AND window_days = 7 AND min_volume = 200`))
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM admin_audit_chain_head WHERE id = 1 AND last_hash IS NULL`))

	_, err = p.DownTo(ctx, adminMetricsVersion-1)
	require.NoError(t, err)
	tables, triggers, columns, indexes = adminMetricsObjects(t, db)
	assert.Equal(t, []int{0, 0, 0, 0}, []int{tables, triggers, columns, indexes})

	_, err = p.UpTo(ctx, adminMetricsVersion)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DELETE FROM goose_db_version WHERE version_id >= ?`, adminMetricsVersion)
	require.NoError(t, err)
	_, err = p.UpTo(ctx, adminMetricsVersion) // IF NOT EXISTS everywhere, INSERT IGNORE seeds
	require.NoError(t, err)
	tables, triggers, columns, indexes = adminMetricsObjects(t, db)
	assert.Equal(t, []int{len(adminMetricsTables), 4, 3, 4}, []int{tables, triggers, columns, indexes})
	assert.Equal(t, 5, countRows(t, db, `SELECT COUNT(*) FROM metric_alert_rules`))
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM admin_audit_chain_head`))

	_, err = db.ExecContext(ctx, `INSERT INTO admin_audit_chain_head (id) VALUES (2)`)
	require.Error(t, err, "the chain head is a singleton")
}

// UPDATE and DELETE on the audit log and on clinical approvals fail; inserting works; deleting the admin or the user
// a row refers to still works (neither table has a foreign key) and leaves the rows in place; the user's activity
// rows cascade away.
func TestAdminMetricsMigration_AppendOnlyTriggers(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	exec := func(q string, args ...any) sql.Result {
		t.Helper()
		res, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		return res
	}
	id := func(res sql.Result) int64 {
		t.Helper()
		v, err := res.LastInsertId()
		require.NoError(t, err)
		return v
	}

	admin := id(exec(`INSERT INTO admins (name, email, password, role, created_at, updated_at) VALUES ('A', 'a@ritme.test', 'x', 'super', NOW(), NOW())`))
	user := id(exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', '09129990105', NOW(), NOW())`))
	exec(`INSERT INTO admin_audit_logs (admin_id, action, target_type, target_id, ip, created_at) VALUES (?, 'user.block', 'user', ?, '', NOW())`, admin, user)
	exec(`INSERT INTO clinical_config_approvals (version_id, admin_id, created_at) VALUES (7, ?, NOW())`, admin)

	for _, tb := range appendOnlyTables {
		_, err := db.ExecContext(ctx, "UPDATE `"+tb+"` SET created_at = NOW() - INTERVAL 1 DAY") //nolint:gosec // fixed names
		require.ErrorContains(t, err, tb+" is append-only")
		_, err = db.ExecContext(ctx, "DELETE FROM `"+tb+"`") //nolint:gosec // fixed names
		require.ErrorContains(t, err, tb+" is append-only")
	}
	_, err := db.ExecContext(ctx, `INSERT INTO clinical_config_approvals (version_id, admin_id, created_at) VALUES (7, ?, NOW())
		ON DUPLICATE KEY UPDATE created_at = NOW()`, admin)
	require.ErrorContains(t, err, "append-only", "an upsert that hits a row is an UPDATE too")

	exec(`INSERT INTO user_activity_days (user_id, day, session) VALUES (?, '2026-10-23', 1)`, user)
	exec(`INSERT INTO user_activity_days (user_id, day, logged) VALUES (?, '2026-10-23', 1) ON DUPLICATE KEY UPDATE logged = 1`, user)
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM user_activity_days WHERE user_id = ? AND session = 1 AND logged = 1`, user))
	exec(`INSERT INTO admin_user_activity (user_id, churn, mode, plan, updated_on) VALUES (?, 'low', 'cycle', 'free', '2026-10-23')`, user)
	exec(`UPDATE users SET ref_code = 'RT-ABC234' WHERE id = ?`, user)
	_, err = db.ExecContext(ctx, `INSERT INTO users (mobile, ref_code, created_at, updated_at) VALUES ('09129990106', 'RT-ABC234', NOW(), NOW())`)
	require.Error(t, err, "ref_code is unique")

	exec(`DELETE FROM admins WHERE id = ?`, admin)
	exec(`DELETE FROM users WHERE id = ?`, user)
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM admin_audit_logs`), "the audit row outlives its admin and target")
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM clinical_config_approvals`))
	assert.Zero(t, countRows(t, db, `SELECT COUNT(*) FROM user_activity_days`))
	assert.Zero(t, countRows(t, db, `SELECT COUNT(*) FROM admin_user_activity`))
}

// open_flag allows one open alert per (rule, subject) and any number of resolved ones; the OpenMetricAlert upsert
// refreshes the open alert instead of adding one.
func TestAdminMetricsMigration_OneOpenAlertPerSubject(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	var rule int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT id FROM metric_alert_rules WHERE metric_key = 'message_ctr'`).Scan(&rule))

	open := func(subject, value string) error {
		_, err := db.ExecContext(ctx, `INSERT INTO metric_alerts (rule_id, subject, day, value, threshold, state, opened_at)
			VALUES (?, ?, '2026-10-23', ?, 0.3, 'open', NOW())`, rule, subject, value)
		return err
	}
	resolve := func(subject string) {
		_, err := db.ExecContext(ctx, `UPDATE metric_alerts SET state = 'resolved', resolved_at = NOW() WHERE rule_id = ? AND subject = ? AND state = 'open'`, rule, subject)
		require.NoError(t, err)
	}

	require.NoError(t, open("m1:v1", "0.4"))
	require.Error(t, open("m1:v1", "0.5"), "a second open alert for the same subject")
	require.NoError(t, open("m2:v1", "0.4"), "another subject")
	require.NoError(t, open("", "0.4"), "the empty subject")
	resolve("m1:v1")
	require.NoError(t, open("m1:v1", "0.6"))
	resolve("m1:v1")
	require.NoError(t, open("m1:v1", "0.7"))

	assert.Equal(t, 2, countRows(t, db, `SELECT COUNT(*) FROM metric_alerts WHERE subject = 'm1:v1' AND state = 'resolved' AND open_flag IS NULL`))
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM metric_alerts WHERE subject = 'm1:v1' AND state = 'open' AND open_flag = 1`))

	_, err := db.ExecContext(ctx, `INSERT INTO metric_alerts (rule_id, subject, day, value, threshold, state, opened_at)
		VALUES (?, 'm1:v1', '2026-10-24', 0.8, 0.3, 'open', NOW())
		ON DUPLICATE KEY UPDATE day = VALUES(day), value = VALUES(value), threshold = VALUES(threshold)`, rule)
	require.NoError(t, err)
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM metric_alerts WHERE subject = 'm1:v1' AND state = 'open' AND value = 0.8 AND day = '2026-10-24'`))
	assert.Equal(t, 3, countRows(t, db, `SELECT COUNT(*) FROM metric_alerts WHERE subject = 'm1:v1'`), "two resolved + one open, no new row")
}

// The KPI range scans use the new indexes.
func TestAdminMetricsMigration_KPIIndexesUsed(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	for i := range 40 {
		res, err := db.ExecContext(ctx, `INSERT INTO users (mobile, created_at, updated_at) VALUES (?, NOW() - INTERVAL ? DAY, NOW())`,
			"0912990"+string(rune('0'+i/10))+string(rune('0'+i%10))+"00", i)
		require.NoError(t, err)
		uid, err := res.LastInsertId()
		require.NoError(t, err)
		for d := range 25 {
			_, err = db.ExecContext(ctx, `INSERT INTO daily_health_logs (user_id, log_date, created_at, updated_at)
				VALUES (?, '2026-01-01' + INTERVAL ? DAY, NOW(), NOW())`, uid, d*12)
			require.NoError(t, err)
			_, err = db.ExecContext(ctx, `INSERT INTO cycle_histories (user_id, period_start_date, created_at, updated_at)
				VALUES (?, '2024-01-01' + INTERVAL ? DAY, NOW(), NOW())`, uid, d*28+i%5)
			require.NoError(t, err)
		}
	}
	for _, tb := range []string{"users", "daily_health_logs", "cycle_histories"} {
		_, err := db.ExecContext(ctx, "ANALYZE TABLE `"+tb+"`") //nolint:gosec // fixed names
		require.NoError(t, err)
	}

	key := func(q string, args ...any) string {
		t.Helper()
		rows, err := db.QueryContext(ctx, "EXPLAIN "+q, args...)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		cols, err := rows.Columns()
		require.NoError(t, err)
		require.True(t, rows.Next())
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		require.NoError(t, rows.Scan(ptrs...))
		require.NoError(t, rows.Err())
		for i, c := range cols {
			if c == "key" {
				return vals[i].String
			}
		}
		return ""
	}
	assert.Equal(t, "daily_health_logs_log_date_user_id_index",
		key(`SELECT user_id, log_date FROM daily_health_logs WHERE log_date >= ?`, "2026-12-01"))
	assert.Equal(t, "cycle_histories_period_start_date_index",
		key(`SELECT user_id, period_start_date FROM cycle_histories WHERE period_start_date BETWEEN ? AND ?`, "2025-06-01", "2025-06-30"))
	assert.Equal(t, "users_created_at_index",
		key(`SELECT COUNT(*) FROM users WHERE created_at >= NOW() - INTERVAL 2 DAY`))
}
