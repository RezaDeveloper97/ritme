package migrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const familySchemaAVersion int64 = 106 // 00106_family_schema_a.sql

// familySchemaAObjects counts the 00106 columns, the family_owner_state table and the 00106 indexes, plus the
// (companion_user_id, status) index that 00106 replaces.
func familySchemaAObjects(t *testing.T, db *sql.DB) (columns, tables, indexes, oldIndex int) {
	t.Helper()
	columns = countRows(t, db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE()
		AND ((table_name = 'companions' AND column_name IN ('role', 'template', 'expires_at', 'urgent_sms', 'owner_confirmed_at'))
		  OR (table_name = 'companion_grants' AND column_name = 'mode_epoch')
		  OR (table_name = 'companion_audit_logs' AND column_name IN ('subject_type', 'subject_id', 'diff_json',
		      'payload_sealed', 'payload_key_id', 'undo_deadline', 'undone_at', 'undone_by', 'proposal_id')))`)
	tables = countRows(t, db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'family_owner_state'`)
	indexes = countRows(t, db, `SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics WHERE table_schema = DATABASE()
		AND index_name IN ('companions_companion_user_id_status_owner_id_index', 'companions_owner_id_companion_user_id_status_index',
		  'companions_status_expires_at_index', 'companion_audit_logs_subject_type_subject_id_created_at_index',
		  'companion_audit_logs_undo_deadline_index', 'companion_audit_logs_created_at_index')`)
	oldIndex = countRows(t, db, `SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics WHERE table_schema = DATABASE()
		AND index_name = 'companions_companion_user_id_status_index'`)
	return
}

// familyQuery returns one named statement of db/queries/companion/family.sql with every sqlc.arg(x) replaced by
// args[x] (an SQL literal), so the tests run the exact text sqlc compiles.
func familyQuery(t *testing.T, name string, args map[string]string) string {
	t.Helper()
	raw, err := os.ReadFile("../queries/companion/family.sql")
	require.NoError(t, err)
	_, rest, ok := strings.Cut(string(raw), "-- name: "+name+" ")
	require.True(t, ok, "statement %s not found", name)
	var lines []string
	for _, l := range strings.Split(rest, "\n")[1:] { // drop the rest of the name line, then the comment lines
		if !strings.HasPrefix(strings.TrimSpace(l), "--") {
			lines = append(lines, l)
		}
	}
	stmt, _, ok := strings.Cut(strings.Join(lines, "\n"), ";")
	require.True(t, ok)
	q := regexp.MustCompile(`sqlc\.arg\(([a-z_]+)\)`).ReplaceAllStringFunc(stmt, func(m string) string {
		key := m[len("sqlc.arg(") : len(m)-1]
		v, ok := args[key]
		require.True(t, ok, "no value for %s", key)
		return v
	})
	return q
}

// T-M11-02: Down removes every 00106 object and brings the replaced index back; Up backfills role from type
// (partner → partner, spouse → partner, parent → teen_parent; F-17) and template = 'legacy', leaves an already-set
// role alone, and a second Up on a database whose Laravel twin already ran is a no-op.
func TestFamilySchemaAMigration_UpDownBackfill(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	columns, tables, indexes, oldIndex := familySchemaAObjects(t, db)
	assert.Equal(t, []int{15, 1, 6, 0}, []int{columns, tables, indexes, oldIndex})

	_, err = p.DownTo(ctx, familySchemaAVersion-1)
	require.NoError(t, err)
	columns, tables, indexes, oldIndex = familySchemaAObjects(t, db)
	assert.Equal(t, []int{0, 0, 0, 1}, []int{columns, tables, indexes, oldIndex})

	exec := func(q string, args ...any) int64 {
		t.Helper()
		res, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	owner := exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('O', '09129991060', NOW(), NOW())`)
	links := map[string]int64{}
	for i, typ := range []string{"partner", "spouse", "parent"} {
		member := exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('M', ?, NOW(), NOW())`, fmt.Sprintf("0912999107%d", i))
		links[typ] = exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, created_at, updated_at)
			VALUES (?, ?, ?, 'active', NOW(), NOW())`, owner, member, typ)
		exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, 'cycle', 'view', NOW(), NOW())`, links[typ])
	}
	revoked := exec(`INSERT INTO companions (owner_id, type, status, revoked_by, created_at, updated_at)
		VALUES (?, 'spouse', 'revoked', 'owner', NOW(), NOW())`, owner)

	_, err = p.UpTo(ctx, familySchemaAVersion)
	require.NoError(t, err)
	columns, tables, indexes, oldIndex = familySchemaAObjects(t, db)
	assert.Equal(t, []int{15, 1, 6, 0}, []int{columns, tables, indexes, oldIndex})

	roleOf := func(id int64) (role, template string) {
		t.Helper()
		require.NoError(t, db.QueryRowContext(ctx, `SELECT role, template FROM companions WHERE id = ?`, id).Scan(&role, &template))
		return
	}
	for typ, want := range map[string]string{"partner": "partner", "spouse": "partner", "parent": "teen_parent"} {
		role, template := roleOf(links[typ])
		assert.Equal(t, []string{want, "legacy"}, []string{role, template}, typ)
	}
	role, template := roleOf(revoked)
	assert.Equal(t, []string{"partner", "legacy"}, []string{role, template}, "revoked links are backfilled too (audit, KPIs)")
	assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM companions WHERE urgent_sms <> 0 OR expires_at IS NOT NULL OR owner_confirmed_at IS NOT NULL`))
	assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM companion_grants WHERE mode_epoch IS NOT NULL`),
		"grant stamping is the T-M11-05 data migration")

	// Idempotent: a role written by v6 code is kept; a re-run (goose bookkeeping forgotten) changes nothing else.
	exec(`UPDATE companions SET role = 'carer', template = 'custom' WHERE id = ?`, links["partner"])
	_, err = db.ExecContext(ctx, `DELETE FROM goose_db_version WHERE version_id >= ?`, familySchemaAVersion)
	require.NoError(t, err)
	_, err = p.UpTo(ctx, familySchemaAVersion)
	require.NoError(t, err)
	role, template = roleOf(links["partner"])
	assert.Equal(t, []string{"carer", "custom"}, []string{role, template})
	columns, tables, indexes, oldIndex = familySchemaAObjects(t, db)
	assert.Equal(t, []int{15, 1, 6, 0}, []int{columns, tables, indexes, oldIndex})

	// The v6 statuses fit the existing varchar(16) columns.
	exec(`UPDATE companions SET status = 'awaiting_owner' WHERE id = ?`, links["spouse"])
	for _, by := range []string{"expired", "declined", "system"} {
		exec(`UPDATE companions SET revoked_by = ? WHERE id = ?`, by, revoked)
	}

	// Down with v6 rows in place still works and leaves the links (and the v1 columns) alone.
	exec(`INSERT INTO family_owner_state (owner_id, mode, updated_at) VALUES (?, 'pregnancy', NOW())`, owner)
	_, err = p.DownTo(ctx, familySchemaAVersion-1)
	require.NoError(t, err)
	assert.Equal(t, 4, countRows(t, db, `SELECT COUNT(*) FROM companions WHERE owner_id = ?`, owner))
	assert.Equal(t, 3, countRows(t, db, `SELECT COUNT(*) FROM companion_grants`))
}

// T-M11-02: both snapshot statements read through the new composite indexes, return one row per (link, grant) with
// the owner state and the owner's blocked_at, and skip non-active links; the owner-state upsert bumps the epoch only on
// a mode change; the activity insert stores the v6 columns; family_owner_state goes with the owner's account.
func TestFamilySchemaA_SnapshotUpsertActivity(t *testing.T) {
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
	user := func(i int) int64 {
		return id(exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, NOW(), NOW())`, fmt.Sprintf("0912888%04d", i)))
	}

	// 40 owners × 10 members, so the optimizer has real choices; owner 0 shares with every member (10 active links:
	// (owner_id, status) alone matches all of them, the pair key exactly one).
	var owners, members []int64
	for i := range 40 {
		owners = append(owners, user(i))
	}
	for i := range 10 {
		members = append(members, user(100+i))
	}
	for i, o := range owners {
		for j, m := range members {
			status := []string{"active", "revoked", "invited"}[(i+j)%3]
			if i == 0 {
				status = "active"
			}
			cid := id(exec(`INSERT INTO companions (owner_id, companion_user_id, type, role, template, status, created_at, updated_at)
				VALUES (?, ?, 'partner', 'partner', 'legacy', ?, NOW(), NOW())`, o, m, status))
			exec(`INSERT INTO companion_grants (companion_id, section, level, mode_epoch, created_at, updated_at) VALUES (?, 'cycle', 'view', 1, NOW(), NOW())`, cid)
			exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, 'meds', 'view', NOW(), NOW())`, cid)
		}
	}
	exec(`ANALYZE TABLE companions, companion_grants, users`)

	owner, member := owners[0], members[0] // (0+0)%3 = active
	pair := familyQuery(t, "FamilySnapshotForPair", map[string]string{"owner_id": fmt.Sprint(owner), "member_id": fmt.Sprint(member)})
	forMember := familyQuery(t, "FamilySnapshotForMember", map[string]string{"member_id": fmt.Sprint(member)})

	companionsKey := func(q string) string {
		t.Helper()
		rows, err := db.QueryContext(ctx, "EXPLAIN "+q)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		cols, err := rows.Columns()
		require.NoError(t, err)
		key := ""
		for rows.Next() {
			vals := make([]sql.NullString, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			require.NoError(t, rows.Scan(ptrs...))
			row := map[string]string{}
			for i, c := range cols {
				row[c] = vals[i].String
			}
			if row["table"] == "c" {
				key = row["key"]
			}
		}
		require.NoError(t, rows.Err())
		return key
	}
	// The pair lookup is three equalities: either new composite key covers all of them (both are a one-row ref), the
	// v1 two-column keys do not.
	assert.Contains(t, []string{"companions_owner_id_companion_user_id_status_index", "companions_companion_user_id_status_owner_id_index"},
		companionsKey(pair))
	assert.Equal(t, "companions_companion_user_id_status_owner_id_index", companionsKey(forMember))

	type snapRow struct {
		owner          int64
		section        sql.NullString
		grantEpoch     sql.NullInt64
		ownerMode      sql.NullString
		ownerEpoch     sql.NullInt64
		ownerBlockedAt sql.NullTime
	}
	snapshot := func(q string) []snapRow {
		t.Helper()
		rows, err := db.QueryContext(ctx, q)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		var out []snapRow
		for rows.Next() {
			var r snapRow
			var linkID, ownerID, memberID int64
			var typ, status string
			var role, template, level sql.NullString
			var expiresAt, confirmedAt sql.NullTime
			var urgent bool
			var review sql.NullBool
			require.NoError(t, rows.Scan(&linkID, &ownerID, &memberID, &typ, &role, &template, &status, &expiresAt, &urgent,
				&confirmedAt, &r.section, &level, &r.grantEpoch, &r.ownerMode, &r.ownerEpoch, &review, &r.ownerBlockedAt))
			assert.Equal(t, "active", status)
			r.owner = ownerID
			out = append(out, r)
		}
		require.NoError(t, rows.Err())
		return out
	}

	got := snapshot(pair)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"cycle", "meds"}, []string{got[0].section.String, got[1].section.String})
	assert.Equal(t, sql.NullInt64{Int64: 1, Valid: true}, got[0].grantEpoch)
	assert.False(t, got[1].grantEpoch.Valid)
	assert.False(t, got[0].ownerMode.Valid, "no owner state yet")
	assert.False(t, got[0].ownerBlockedAt.Valid)
	assert.Empty(t, snapshot(familyQuery(t, "FamilySnapshotForPair",
		map[string]string{"owner_id": fmt.Sprint(owners[1]), "member_id": fmt.Sprint(member)})), "revoked link") // (1+0)%3

	all := snapshot(forMember)
	assert.Len(t, all, 2*14, "14 of 40 owners have an active link with member 0, two grants each")
	for i := 1; i < len(all); i++ {
		assert.LessOrEqual(t, all[i-1].owner, all[i].owner, "ordered by owner")
	}

	// Owner-state upsert: created (1), same mode (0), new mode (2, epoch + 1, review pending).
	upsert := func(mode string) int64 {
		t.Helper()
		n, err := exec(familyQuery(t, "UpsertFamilyOwnerState", map[string]string{
			"owner_id": fmt.Sprint(owner), "mode": "'" + mode + "'", "now": "'2026-10-23 10:00:00'",
		})).RowsAffected()
		require.NoError(t, err)
		return n
	}
	state := func() (mode string, epoch int, review bool) {
		t.Helper()
		require.NoError(t, db.QueryRowContext(ctx, `SELECT mode, mode_epoch, review_pending FROM family_owner_state WHERE owner_id = ?`, owner).
			Scan(&mode, &epoch, &review))
		return
	}
	assert.EqualValues(t, 1, upsert("cycle"))
	mode, epoch, review := state()
	assert.Equal(t, []any{"cycle", 1, false}, []any{mode, epoch, review})
	assert.EqualValues(t, 0, upsert("cycle"))
	assert.EqualValues(t, 2, upsert("pregnancy"))
	mode, epoch, review = state()
	assert.Equal(t, []any{"pregnancy", 2, true}, []any{mode, epoch, review})
	exec(`UPDATE family_owner_state SET review_pending = 0 WHERE owner_id = ?`, owner)
	assert.EqualValues(t, 0, upsert("pregnancy"))
	mode, epoch, review = state()
	assert.Equal(t, []any{"pregnancy", 2, false}, []any{mode, epoch, review}, "the same mode keeps the epoch and the review state")

	exec(`UPDATE users SET blocked_at = NOW() WHERE id = ?`, owner)
	got = snapshot(pair)
	require.Len(t, got, 2)
	assert.Equal(t, "pregnancy", got[0].ownerMode.String)
	assert.EqualValues(t, 2, got[0].ownerEpoch.Int64)
	assert.True(t, got[0].ownerBlockedAt.Valid)

	// Activity insert with the v6 columns.
	actID := id(exec(familyQuery(t, "InsertFamilyActivity", map[string]string{
		"owner_id": fmt.Sprint(owner), "actor_id": fmt.Sprint(member), "companion_id": "NULL", "section": "'children'",
		"action": "'child_write'", "subject_type": "'child'", "subject_id": "42", "diff_json": "NULL",
		"payload_sealed": "x'00ff10'", "payload_key_id": "1", "undo_deadline": "'2026-10-24 10:00:00'", "proposal_id": "NULL",
		"now": "'2026-10-23 10:00:00'",
	})))
	var sealed []byte
	var subjectType string
	var subjectID int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT subject_type, subject_id, payload_sealed FROM companion_audit_logs WHERE id = ?`, actID).
		Scan(&subjectType, &subjectID, &sealed))
	assert.Equal(t, "child", subjectType)
	assert.EqualValues(t, 42, subjectID)
	assert.Equal(t, []byte{0x00, 0xff, 0x10}, sealed)
	_, err := db.ExecContext(ctx, familyQuery(t, "InsertFamilyActivity", map[string]string{
		"owner_id": fmt.Sprint(owner), "actor_id": "NULL", "companion_id": "NULL", "section": "NULL", "action": "'grant_change'",
		"subject_type": "NULL", "subject_id": "NULL", "diff_json": "'not json'", "payload_sealed": "NULL", "payload_key_id": "NULL",
		"undo_deadline": "NULL", "proposal_id": "NULL", "now": "NOW()",
	}))
	require.Error(t, err, "diff_json must be valid JSON")

	// The owner's state goes with her account.
	exec(`DELETE FROM users WHERE id = ?`, owner)
	assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM family_owner_state WHERE owner_id = ?`, owner))
}
