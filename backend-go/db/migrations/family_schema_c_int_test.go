package migrations_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rootdb "github.com/ritme/backend-go/db"
	babylogstore "github.com/ritme/backend-go/internal/babylog/store"
	childrenstore "github.com/ritme/backend-go/internal/children/store"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	todostore "github.com/ritme/backend-go/internal/todo/store"
)

const familySchemaCVersion int64 = 108 // 00108_family_schema_c.sql

// familySchemaCActorTables get created_by / updated_by in 00108 (baby_med_doses is created with them).
var familySchemaCActorTables = []string{
	"children", "child_measurements", "child_vaccine_doses", "child_milestone_checks",
	"baby_feeds", "baby_sleeps", "baby_diapers", "todo_tasks", "todo_items",
}

// familySchemaCObjects counts the 00108 tables, the added columns (2 per actor table + todo scope / assigned_to) and
// their SET NULL foreign keys to users, and the household key.
func familySchemaCObjects(t *testing.T, db *sql.DB) (tables, columns, fks, index int) {
	t.Helper()
	tables = countRows(t, db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()
		AND table_name IN ('child_guardians', 'baby_med_doses')`)
	for _, tb := range familySchemaCActorTables {
		columns += countRows(t, db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE()
			AND table_name = ? AND column_name IN ('created_by', 'updated_by')`, tb)
		fks += countRows(t, db, `SELECT COUNT(*) FROM information_schema.referential_constraints WHERE constraint_schema = DATABASE()
			AND table_name = ? AND referenced_table_name = 'users' AND delete_rule = 'SET NULL'
			AND constraint_name IN (CONCAT(?, '_created_by_foreign'), CONCAT(?, '_updated_by_foreign'))`, tb, tb, tb)
	}
	columns += countRows(t, db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE()
		AND table_name = 'todo_tasks' AND column_name IN ('scope', 'assigned_to')`)
	fks += countRows(t, db, `SELECT COUNT(*) FROM information_schema.referential_constraints WHERE constraint_schema = DATABASE()
		AND constraint_name = 'todo_tasks_assigned_to_foreign' AND delete_rule = 'SET NULL'`)
	index = countRows(t, db, `SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics WHERE table_schema = DATABASE()
		AND table_name = 'todo_tasks' AND index_name = 'todo_tasks_user_id_scope_done_at_index'`)
	return
}

type familyCInserter struct {
	t   *testing.T
	db  *sql.DB
	seq int
}

func (f *familyCInserter) exec(q string, args ...any) int64 {
	f.t.Helper()
	res, err := f.db.ExecContext(context.Background(), q, args...)
	require.NoError(f.t, err)
	id, err := res.LastInsertId()
	require.NoError(f.t, err)
	return id
}

func (f *familyCInserter) user(name string) int64 {
	f.t.Helper()
	f.seq++
	return f.exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, name, fmt.Sprintf("0912888%04d", f.seq))
}

func (f *familyCInserter) child(owner int64, name string) int64 {
	f.t.Helper()
	return f.exec(`INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, ?, '2026-01-10', NOW(), NOW())`, owner, name)
}

// spouseFamily links owner → spouse with a spouse companion link in status and shares the children with it.
func (f *familyCInserter) spouseFamily(owner, spouse int64, status string, children ...int64) (link int64) {
	f.t.Helper()
	link = f.exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, created_at, updated_at)
		VALUES (?, ?, 'spouse', ?, NOW(), NOW())`, owner, spouse, status)
	family := f.exec(`INSERT INTO families (owner_id, spouse_user_id, companion_id, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`,
		owner, spouse, link)
	for _, c := range children {
		f.exec(`INSERT INTO family_children (family_id, child_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, family, c)
	}
	return link
}

// guardianSet is "user:level:status[:invited_by:link_id]" per guardian of a child, in id order.
func guardianSet(t *testing.T, db *sql.DB, child int64) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `SELECT user_id, level, status, invited_by, link_id, sub_scopes IS NULL
		FROM child_guardians WHERE child_id = ? ORDER BY id`, child)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var user int64
		var level, status string
		var invitedBy, link sql.NullInt64
		var noScopes bool
		require.NoError(t, rows.Scan(&user, &level, &status, &invitedBy, &link, &noScopes))
		assert.True(t, noScopes, "backfilled guardians have no sub_scopes")
		s := fmt.Sprintf("%d:%s:%s", user, level, status)
		if invitedBy.Valid || link.Valid {
			s += fmt.Sprintf(":%d:%d", invitedBy.Int64, link.Int64)
		}
		out = append(out, s)
	}
	require.NoError(t, rows.Err())
	return out
}

func nullRaw(s string) rootdb.NullRawJSON {
	return rootdb.NullRawJSON{V: json.RawMessage(s), Valid: true}
}

// T-M11-04: Down removes every 00108 object; Up backfills guardians — the owner of every child, and the spouse of an
// active spouse link for each shared child (F-17) — never from a revoked link, never created_by; a re-run is a no-op
// that keeps rows v6 code changed; Down with v6 rows in place still works.
func TestFamilySchemaCMigration_UpDownBackfill(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	full := []int{2, 2*len(familySchemaCActorTables) + 2, 2*len(familySchemaCActorTables) + 1, 1}
	tables, columns, fks, index := familySchemaCObjects(t, db)
	assert.Equal(t, full, []int{tables, columns, fks, index})

	_, err = p.DownTo(ctx, familySchemaCVersion-1)
	require.NoError(t, err)
	tables, columns, fks, index = familySchemaCObjects(t, db)
	assert.Equal(t, []int{0, 0, 0, 0}, []int{tables, columns, fks, index})

	f := &familyCInserter{t: t, db: db}
	// Owner only.
	solo := f.user("Solo")
	soloChild := f.child(solo, "S1")
	// Owner + spouse (active link) with two shared children and one not shared.
	mother, father := f.user("Mother"), f.user("Father")
	shared1, shared2, notShared := f.child(mother, "B1"), f.child(mother, "B2"), f.child(mother, "B3")
	link := f.spouseFamily(mother, father, "active", shared1, shared2)
	// Owner whose spouse link was revoked: the ex-spouse gets nothing. A pending (invited) link gives nothing either.
	other, exSpouse, invitee := f.user("Other"), f.user("Ex"), f.user("Invitee")
	otherChild := f.child(other, "C1")
	f.spouseFamily(other, exSpouse, "revoked", otherChild)
	f.spouseFamily(other, invitee, "invited", otherChild)
	task := f.exec(`INSERT INTO todo_tasks (user_id, title, category, created_at, updated_at) VALUES (?, 'T', 'home', NOW(), NOW())`, mother)

	_, err = p.UpTo(ctx, familySchemaCVersion)
	require.NoError(t, err)
	tables, columns, fks, index = familySchemaCObjects(t, db)
	assert.Equal(t, full, []int{tables, columns, fks, index})

	ownerOnly := func(owner int64) []string { return []string{fmt.Sprintf("%d:write:active", owner)} }
	coGuarded := []string{fmt.Sprintf("%d:write:active", mother), fmt.Sprintf("%d:write:active:%d:%d", father, mother, link)}
	assert.Equal(t, ownerOnly(solo), guardianSet(t, db, soloChild))
	assert.Equal(t, coGuarded, guardianSet(t, db, shared1))
	assert.Equal(t, coGuarded, guardianSet(t, db, shared2))
	assert.Equal(t, ownerOnly(mother), guardianSet(t, db, notShared), "a child not shared with the family has only its owner")
	assert.Equal(t, ownerOnly(other), guardianSet(t, db, otherChild), "revoked / pending spouse links give no guardian")
	assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM children WHERE created_by IS NOT NULL OR updated_by IS NOT NULL`),
		"created_by is not backfilled (unknown history stays NULL)")
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM todo_tasks WHERE id = ? AND scope = 'personal' AND assigned_to IS NULL
		AND created_by IS NULL`, task), "existing tasks stay personal")

	// Idempotent: a guardian row changed by v6 code is kept; a re-run (goose bookkeeping forgotten) adds nothing.
	f.exec(`UPDATE child_guardians SET status = 'left' WHERE child_id = ? AND user_id = ?`, shared1, father)
	_, err = db.ExecContext(ctx, `DELETE FROM goose_db_version WHERE version_id >= ?`, familySchemaCVersion)
	require.NoError(t, err)
	_, err = p.UpTo(ctx, familySchemaCVersion)
	require.NoError(t, err, "re-run over an existing schema (Laravel twin first) is a no-op")
	assert.Equal(t, 7, countRows(t, db, `SELECT COUNT(*) FROM child_guardians`))
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM child_guardians WHERE child_id = ? AND user_id = ? AND status = 'left'`,
		shared1, father))
	tables, columns, fks, index = familySchemaCObjects(t, db)
	assert.Equal(t, full, []int{tables, columns, fks, index})

	// Down with v6 rows in place: the dose log and guardians go, the child rows and the task stay.
	f.exec(`INSERT INTO baby_med_doses (child_id, name, given_at, created_by, updated_by, created_at, updated_at)
		VALUES (?, 'Drops', '2026-10-23 08:00:00', ?, ?, NOW(), NOW())`, shared1, father, father)
	f.exec(`UPDATE todo_tasks SET scope = 'household', assigned_to = ?, created_by = ? WHERE id = ?`, father, mother, task)
	_, err = p.DownTo(ctx, familySchemaCVersion-1)
	require.NoError(t, err)
	tables, columns, fks, index = familySchemaCObjects(t, db)
	assert.Equal(t, []int{0, 0, 0, 0}, []int{tables, columns, fks, index})
	assert.Equal(t, 5, countRows(t, db, `SELECT COUNT(*) FROM children`))
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM todo_tasks WHERE id = ?`, task))
	assert.Equal(t, 2, countRows(t, db, `SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'todo_tasks' AND index_name <> 'PRIMARY'`),
		"Down leaves exactly the two v1 keys of todo_tasks")
}

// T-M11-04: deleting an account nulls its actor columns everywhere (the entries stay, shown as "—"), removes only its
// own guardian rows, and — the frozen Laravel deletion path, a plain DELETE FROM users — still cascades a sole
// owner's child with all its logs (children.owner_id is unchanged).
func TestFamilySchemaC_AccountDeletion(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	f := &familyCInserter{t: t, db: db, seq: 100}
	mother, father := f.user("Mother"), f.user("Father")
	child := f.child(mother, "B1")
	f.spouseFamily(mother, father, "active", child)
	f.exec(`INSERT INTO child_guardians (child_id, user_id, level, status, created_at, updated_at) VALUES (?, ?, 'write', 'active', NOW(), NOW())`,
		child, mother)
	f.exec(`INSERT INTO child_guardians (child_id, user_id, level, status, invited_by, created_at, updated_at)
		VALUES (?, ?, 'write', 'active', ?, NOW(), NOW())`, child, father, mother)

	_, err := db.ExecContext(ctx, `UPDATE children SET created_by = ?, updated_by = ? WHERE id = ?`, father, father, child)
	require.NoError(t, err)
	f.exec(`INSERT INTO child_measurements (child_id, measured_on, weight_kg, created_by, updated_by) VALUES (?, '2026-10-01', 4.2, ?, ?)`, child, father, father)
	f.exec(`INSERT INTO child_vaccine_doses (child_id, dose_code, given_on, created_by, updated_by) VALUES (?, 'bcg', '2026-01-11', ?, ?)`, child, father, father)
	f.exec(`INSERT INTO child_milestone_checks (child_id, milestone_code, checked_on, created_by, updated_by) VALUES (?, 'smile', '2026-03-01', ?, ?)`, child, father, father)
	f.exec(`INSERT INTO baby_feeds (child_id, type, started_at, created_by, updated_by) VALUES (?, 'bottle', '2026-10-23 07:00:00', ?, ?)`, child, father, father)
	f.exec(`INSERT INTO baby_sleeps (child_id, started_at, created_by, updated_by) VALUES (?, '2026-10-23 09:00:00', ?, ?)`, child, father, father)
	f.exec(`INSERT INTO baby_diapers (child_id, changed_at, kind, created_by, updated_by) VALUES (?, '2026-10-23 10:00:00', 'wet', ?, ?)`, child, father, father)
	f.exec(`INSERT INTO baby_med_doses (child_id, name, given_at, created_by, updated_by) VALUES (?, 'Vit D', '2026-10-23 11:00:00', ?, ?)`, child, father, father)
	task := f.exec(`INSERT INTO todo_tasks (user_id, title, category, scope, assigned_to, created_by, updated_by)
		VALUES (?, 'شیفت شب', 'home', 'household', ?, ?, ?)`, mother, father, father, father)
	f.exec(`INSERT INTO todo_items (task_id, user_id, title, created_by, updated_by) VALUES (?, ?, 'Bottle', ?, ?)`, task, mother, father, father)

	_, err = db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, father)
	require.NoError(t, err)
	for _, tb := range append(append([]string{}, familySchemaCActorTables...), "baby_med_doses") {
		assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM `+tb+` WHERE created_by IS NULL AND updated_by IS NULL`), tb)
	}
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM todo_tasks WHERE id = ? AND assigned_to IS NULL`, task))
	assert.Equal(t, []string{fmt.Sprintf("%d:write:active", mother)}, guardianSet(t, db, child),
		"the deleted account's guardian row goes, the owner's stays")

	// The sole owner deletes her account through the Laravel path: the child and every child row go with it.
	_, err = db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, mother)
	require.NoError(t, err)
	for _, tb := range []string{"children", "child_guardians", "child_measurements", "child_vaccine_doses", "child_milestone_checks",
		"baby_feeds", "baby_sleeps", "baby_diapers", "baby_med_doses", "todo_tasks", "todo_items"} {
		assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM `+tb), tb)
	}

	// Cascade unchanged even with a co-guardian left: moving the child first is T-M11-11's job in Go, not the schema's.
	owner, co := f.user("Owner"), f.user("Co")
	kept := f.child(owner, "K1")
	f.exec(`INSERT INTO child_guardians (child_id, user_id, level, status) VALUES (?, ?, 'write', 'active'), (?, ?, 'write', 'active')`,
		kept, owner, kept, co)
	_, err = db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, owner)
	require.NoError(t, err)
	assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM children WHERE id = ?`, kept))
	assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM child_guardians WHERE user_id = ?`, co))
}

// T-M11-04: the sqlc statements — guardian upsert / guarded status moves / reads by child set and by user, the dose
// log CRUD scoped by child, and the household list reads (scope, owners, assignee).
func TestFamilySchemaC_Queries(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	f := &familyCInserter{t: t, db: db, seq: 200}
	mother, father, carer, stranger := f.user("Mother"), f.user("Father"), f.user("Carer"), f.user("Stranger")
	baby, older := f.child(mother, "Baby"), f.child(mother, "Older")
	now := sql.NullTime{Time: time.Date(2026, 10, 23, 10, 0, 0, 0, time.Local), Valid: true}
	id := func(v int64) sql.NullInt64 { return sql.NullInt64{Int64: v, Valid: true} }

	cq := childrenstore.New(db)
	for _, g := range []childrenstore.UpsertGuardianParams{
		{ChildID: uint64(baby), UserID: uint64(mother), Level: "write", Status: "active", Now: now},
		{ChildID: uint64(older), UserID: uint64(mother), Level: "write", Status: "active", Now: now},
		{ChildID: uint64(baby), UserID: uint64(father), Level: "write", Status: "invited", InvitedBy: id(mother), Now: now},
		{ChildID: uint64(baby), UserID: uint64(carer), Level: "limited", Status: "active", InvitedBy: id(mother), Now: now,
			SubScopes: nullRaw(`["feeding","diaper","sleep"]`)},
	} {
		require.NoError(t, cq.UpsertGuardian(ctx, g))
	}
	n, err := cq.CountActiveWriteGuardians(ctx, uint64(baby))
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "the invited father and the limited carer do not count")

	moved, err := cq.SetGuardianStatus(ctx, childrenstore.SetGuardianStatusParams{
		Status: "active", Now: now, ChildID: uint64(baby), UserID: uint64(father), FromStatus: "invited"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, moved)
	moved, err = cq.SetGuardianStatus(ctx, childrenstore.SetGuardianStatusParams{
		Status: "active", Now: now, ChildID: uint64(baby), UserID: uint64(father), FromStatus: "invited"})
	require.NoError(t, err)
	assert.EqualValues(t, 0, moved, "a guarded move only applies from from_status")
	n, err = cq.CountActiveWriteGuardians(ctx, uint64(baby))
	require.NoError(t, err)
	assert.EqualValues(t, 2, n)

	// Upsert rewrites in place (one row per child and user).
	require.NoError(t, cq.UpsertGuardian(ctx, childrenstore.UpsertGuardianParams{
		ChildID: uint64(baby), UserID: uint64(carer), Level: "limited", Status: "removed", InvitedBy: id(mother), Now: now,
		SubScopes: nullRaw(`["feeding"]`)}))
	g, err := cq.GetGuardian(ctx, childrenstore.GetGuardianParams{ChildID: uint64(baby), UserID: uint64(carer)})
	require.NoError(t, err)
	assert.Equal(t, "removed", g.Status)
	assert.JSONEq(t, `["feeding"]`, string(g.SubScopes.V))
	assert.Equal(t, 4, countRows(t, db, `SELECT COUNT(*) FROM child_guardians`))

	all, err := cq.ListGuardiansForChildren(ctx, []uint64{uint64(baby), uint64(older)})
	require.NoError(t, err)
	var got []string
	for _, r := range all {
		got = append(got, fmt.Sprintf("%d:%s:%s", r.ChildID, r.UserName.String, r.Status))
	}
	assert.Equal(t, []string{
		fmt.Sprintf("%d:Mother:active", baby), fmt.Sprintf("%d:Father:active", baby), fmt.Sprintf("%d:Carer:removed", baby),
		fmt.Sprintf("%d:Mother:active", older),
	}, got)
	mine, err := cq.ListActiveGuardianshipsForUser(ctx, uint64(father))
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, []uint64{uint64(baby), uint64(mother)}, []uint64{mine[0].ChildID, mine[0].OwnerID})
	mine, err = cq.ListActiveGuardianshipsForUser(ctx, uint64(carer))
	require.NoError(t, err)
	assert.Empty(t, mine, "a removed guardian guards nothing")

	// Dose log.
	bq := babylogstore.New(db)
	at := func(h int) time.Time { return time.Date(2026, 10, 23, h, 0, 0, 0, time.Local) }
	dose, err := bq.InsertMedDose(ctx, babylogstore.InsertMedDoseParams{ChildID: uint64(baby), Name: "Vit D",
		DoseText: sql.NullString{String: "1 قطره", Valid: true}, GivenAt: at(8), ActorID: id(father), Now: now})
	require.NoError(t, err)
	_, err = bq.InsertMedDose(ctx, babylogstore.InsertMedDoseParams{ChildID: uint64(baby), Name: "Iron", GivenAt: at(12),
		ActorID: id(mother), Now: now})
	require.NoError(t, err)
	_, err = bq.InsertMedDose(ctx, babylogstore.InsertMedDoseParams{ChildID: uint64(older), Name: "Syrup", GivenAt: at(9), Now: now})
	require.NoError(t, err)
	changed, err := bq.UpdateMedDose(ctx, babylogstore.UpdateMedDoseParams{Name: "Vit D", GivenAt: at(8), ActorID: id(mother),
		Now: now, ID: uint64(dose), ChildID: uint64(older)})
	require.NoError(t, err)
	assert.EqualValues(t, 0, changed, "a dose is only reachable through its own child")
	changed, err = bq.UpdateMedDose(ctx, babylogstore.UpdateMedDoseParams{Name: "Vit D3", GivenAt: at(8), ActorID: id(mother),
		Now: now, ID: uint64(dose), ChildID: uint64(baby)})
	require.NoError(t, err)
	assert.EqualValues(t, 1, changed)
	d, err := bq.GetMedDose(ctx, babylogstore.GetMedDoseParams{ID: uint64(dose), ChildID: uint64(baby)})
	require.NoError(t, err)
	assert.Equal(t, []any{"Vit D3", father, mother}, []any{d.Name, d.CreatedBy.Int64, d.UpdatedBy.Int64},
		"created_by keeps the first writer, updated_by moves")
	list, err := bq.ListMedDosesBetween(ctx, babylogstore.ListMedDosesBetweenParams{ChildID: uint64(baby), FromAt: at(0), ToAt: at(23), Limit: 50})
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, []string{"Iron:Mother", "Vit D3:Father"}, []string{
		list[0].Name + ":" + list[0].CreatedByName.String, list[1].Name + ":" + list[1].CreatedByName.String}, "newest first")
	gone, err := bq.DeleteMedDose(ctx, babylogstore.DeleteMedDoseParams{ID: uint64(dose), ChildID: uint64(older)})
	require.NoError(t, err)
	assert.EqualValues(t, 0, gone)
	gone, err = bq.DeleteMedDose(ctx, babylogstore.DeleteMedDoseParams{ID: uint64(dose), ChildID: uint64(baby)})
	require.NoError(t, err)
	assert.EqualValues(t, 1, gone)

	// Household list: one list per owner, never merged; personal tasks never show; ticked tasks only since done_since.
	household := func(owner int64, title string, assignee sql.NullInt64, doneAt string) int64 {
		t.Helper()
		return f.exec(`INSERT INTO todo_tasks (user_id, title, category, scope, assigned_to, done_at, created_at, updated_at)
			VALUES (?, ?, 'home', 'household', ?, NULLIF(?, ''), NOW(), NOW())`, owner, title, assignee, doneAt)
	}
	night := household(mother, "شیفت شب", id(father), "")
	household(mother, "Old", sql.NullInt64{}, "2026-10-01 10:00:00")
	household(mother, "Recent", sql.NullInt64{}, "2026-10-22 10:00:00")
	fatherOwn := household(father, "Groceries", id(father), "")
	household(stranger, "Not shared", id(father), "")
	f.exec(`INSERT INTO todo_tasks (user_id, title, category, assigned_to, created_at, updated_at) VALUES (?, 'Private', 'home', ?, NOW(), NOW())`,
		mother, father)
	f.exec(`INSERT INTO todo_items (task_id, user_id, title) VALUES (?, ?, 'a'), (?, ?, 'b')`, night, mother, night, mother)

	tq := todostore.New(db)
	board, err := tq.ListHouseholdTasks(ctx, todostore.ListHouseholdTasksParams{OwnerIds: []uint64{uint64(mother), uint64(father)},
		DoneSince: sql.NullTime{Time: time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local), Valid: true}, Limit: 100})
	require.NoError(t, err)
	got = got[:0]
	for _, r := range board {
		got = append(got, fmt.Sprintf("%d:%s:%s:%d", r.UserID, r.Title, r.AssigneeName.String, r.ItemsTotal))
	}
	assert.Equal(t, []string{
		fmt.Sprintf("%d:شیفت شب:Father:2", mother), fmt.Sprintf("%d:Recent::0", mother), fmt.Sprintf("%d:Groceries:Father:0", father),
	}, got)
	_, err = tq.GetHouseholdTask(ctx, todostore.GetHouseholdTaskParams{ID: uint64(night), OwnerID: uint64(father)})
	assert.ErrorIs(t, err, sql.ErrNoRows, "a household task is read through its owner only")
	ht, err := tq.GetHouseholdTask(ctx, todostore.GetHouseholdTaskParams{ID: uint64(night), OwnerID: uint64(mother)})
	require.NoError(t, err)
	assert.Equal(t, "household", ht.Scope)
	assigned, err := tq.ListOpenHouseholdTasksAssignedTo(ctx, todostore.ListOpenHouseholdTasksAssignedToParams{
		AssignedTo: id(father), OwnerIds: []uint64{uint64(mother), uint64(father)}})
	require.NoError(t, err)
	var ids []uint64
	for _, r := range assigned {
		ids = append(ids, r.ID)
	}
	assert.Equal(t, []uint64{uint64(night), uint64(fatherOwn)}, ids, "the stranger's list and personal tasks are left out")
}
