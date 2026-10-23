package migrations_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const familySchemaBVersion int64 = 107 // 00107_family_schema_b.sql

// familySchemaBObjects counts the 00107 tables and the user_notifications.link_id column, index and foreign key.
func familySchemaBObjects(t *testing.T, db *sql.DB) (tables, column, index, fk int) {
	t.Helper()
	tables = countRows(t, db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN
		('family_proposals', 'family_share_requests', 'family_nudges', 'family_member_alerts', 'family_member_notices', 'family_attendance')`)
	column = countRows(t, db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE()
		AND table_name = 'user_notifications' AND column_name = 'link_id'`)
	index = countRows(t, db, `SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics WHERE table_schema = DATABASE()
		AND table_name = 'user_notifications' AND index_name = 'user_notifications_link_id_index'`)
	fk = countRows(t, db, `SELECT COUNT(*) FROM information_schema.referential_constraints WHERE constraint_schema = DATABASE()
		AND constraint_name = 'user_notifications_link_id_foreign' AND referenced_table_name = 'companions' AND delete_rule = 'CASCADE'`)
	return
}

// familyFixture is one owner, one member and an active link between them, plus an owner appointment (reminders row).
type familyFixture struct {
	owner, member, link, appointment int64
}

func newFamilyFixture(t *testing.T, db *sql.DB, n int) familyFixture {
	t.Helper()
	ctx := context.Background()
	ins := func(q string, args ...any) int64 {
		t.Helper()
		res, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	var f familyFixture
	f.owner = ins(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('O', ?, NOW(), NOW())`, fmt.Sprintf("0912777%04d", 2*n))
	f.member = ins(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('M', ?, NOW(), NOW())`, fmt.Sprintf("0912777%04d", 2*n+1))
	f.link = ins(`INSERT INTO companions (owner_id, companion_user_id, type, role, template, status, created_at, updated_at)
		VALUES (?, ?, 'partner', 'partner', 'custom', 'active', NOW(), NOW())`, f.owner, f.member)
	f.appointment = ins(`INSERT INTO reminders (user_id, type, title, created_at, updated_at) VALUES (?, 'appointment', 'A', NOW(), NOW())`, f.owner)
	return f
}

// T-M11-03: Down removes the six tables and user_notifications.link_id (with its key and FK) and keeps the
// notifications; Up brings everything back; a second Up on a database whose Laravel twin already ran is a no-op.
func TestFamilySchemaBMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	tables, column, index, fk := familySchemaBObjects(t, db)
	assert.Equal(t, []int{6, 1, 1, 1}, []int{tables, column, index, fk})

	f := newFamilyFixture(t, db, 0)
	_, err = db.ExecContext(ctx, `INSERT INTO user_notifications (user_id, type, title, link_id, created_at, updated_at)
		VALUES (?, 'companion', '{"fa":"x"}', ?, NOW(), NOW())`, f.member, f.link)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO family_nudges (owner_id, member_id, link_id, target_type, target_id, local_date, slot, sent_at)
		VALUES (?, ?, ?, 'appointment', ?, '2026-10-23', '', NOW())`, f.owner, f.member, f.link, f.appointment)
	require.NoError(t, err)

	_, err = p.DownTo(ctx, familySchemaBVersion-1)
	require.NoError(t, err)
	tables, column, index, fk = familySchemaBObjects(t, db)
	assert.Equal(t, []int{0, 0, 0, 0}, []int{tables, column, index, fk})
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, f.member))

	_, err = p.UpTo(ctx, familySchemaBVersion)
	require.NoError(t, err)
	tables, column, index, fk = familySchemaBObjects(t, db)
	assert.Equal(t, []int{6, 1, 1, 1}, []int{tables, column, index, fk})
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND link_id IS NULL`, f.member))

	_, err = db.ExecContext(ctx, `DELETE FROM goose_db_version WHERE version_id >= ?`, familySchemaBVersion)
	require.NoError(t, err)
	_, err = p.UpTo(ctx, familySchemaBVersion)
	require.NoError(t, err, "re-run over an existing schema (Laravel twin first) is a no-op")
	tables, column, index, fk = familySchemaBObjects(t, db)
	assert.Equal(t, []int{6, 1, 1, 1}, []int{tables, column, index, fk})
}

// T-M11-03: one pending proposal per dedupe key (a second pending child_delete fails, a decided one does not block a
// new one, a NULL key never dedupes); decisions only move live pending rows; lazy expiry on read and the sweep.
func TestFamilySchemaB_ProposalsAndShareRequests(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	f := newFamilyFixture(t, db, 1)
	exec := func(q string, args ...any) sql.Result {
		t.Helper()
		res, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		return res
	}
	affected := func(res sql.Result) int64 {
		t.Helper()
		n, err := res.RowsAffected()
		require.NoError(t, err)
		return n
	}
	const now = "'2026-10-23 10:00:00'"
	proposal := func(kind, dedupe, expires string) (int64, error) {
		res, err := db.ExecContext(ctx, familyQuery(t, "InsertFamilyProposal", map[string]string{
			"subject_type": "'child'", "subject_id": "77", "link_id": fmt.Sprint(f.link), "actor_id": fmt.Sprint(f.member),
			"approver_id": fmt.Sprint(f.owner), "kind": "'" + kind + "'", "domain": "'children'", "payload_sealed": "x'0102'",
			"payload_key_id": "1", "dedupe_key": dedupe, "expires_at": expires, "now": now,
		}))
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	isDuplicate := func(err error) bool {
		var me *mysql.MySQLError
		return errors.As(err, &me) && me.Number == 1062
	}

	first, err := proposal("child_delete", "'child_delete:77'", "'2026-10-30 10:00:00'")
	require.NoError(t, err)
	_, err = proposal("child_delete", "'child_delete:77'", "'2026-10-30 10:00:00'")
	assert.True(t, isDuplicate(err), "a second pending proposal with the same dedupe key fails: %v", err)
	_, err = proposal("create_med", "NULL", "'2026-10-30 10:00:00'")
	require.NoError(t, err)
	_, err = proposal("create_med", "NULL", "'2026-10-30 10:00:00'")
	require.NoError(t, err, "a NULL dedupe key never dedupes")

	decide := func(id int64, status, at string) int64 {
		return affected(exec(familyQuery(t, "DecideFamilyProposal", map[string]string{
			"status": "'" + status + "'", "decided_by": fmt.Sprint(f.owner), "decided_at": at, "applied_ref": "NULL",
			"id": fmt.Sprint(id), "now": at,
		})))
	}
	assert.EqualValues(t, 1, decide(first, "rejected", now))
	assert.EqualValues(t, 0, decide(first, "approved", now), "a decided proposal stays decided")
	second, err := proposal("child_delete", "'child_delete:77'", "'2026-10-30 10:00:00'")
	require.NoError(t, err, "a decided proposal does not block a new pending one")
	var pendingKey sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, `SELECT pending_key FROM family_proposals WHERE id = ?`, first).Scan(&pendingKey))
	assert.False(t, pendingKey.Valid)

	// Lazy expiry: past expires_at the row reads as expired, cannot be decided, and is left out of the pending list.
	lapsed, err := proposal("create_appointment", "'appt:1'", "'2026-10-23 09:00:00'")
	require.NoError(t, err)
	var status string
	var expired int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT status, expired FROM (`+
		familyQuery(t, "GetFamilyProposal", map[string]string{"now": now, "id": fmt.Sprint(lapsed)})+`) x`).Scan(&status, &expired))
	assert.Equal(t, "pending", status)
	assert.EqualValues(t, 1, expired)
	assert.EqualValues(t, 0, decide(lapsed, "approved", now))
	pending := countRows(t, db, `SELECT COUNT(*) FROM (`+familyQuery(t, "ListPendingFamilyProposalsForApprover",
		map[string]string{"approver_id": fmt.Sprint(f.owner), "now": now})+`) x`, 50)
	assert.Equal(t, 3, pending, "two NULL-key create_med + the second child_delete; the lapsed one is left out")
	assert.Equal(t, 3, countRows(t, db, familyQuery(t, "CountPendingFamilyProposalsByActor",
		map[string]string{"actor_id": fmt.Sprint(f.member), "now": now})))

	assert.EqualValues(t, 1, affected(exec(familyQuery(t, "ExpireFamilyProposals",
		map[string]string{"updated_at": now, "now": now}), 100)))
	assert.EqualValues(t, 0, affected(exec(familyQuery(t, "ExpireFamilyProposals",
		map[string]string{"updated_at": now, "now": now}), 100)), "the sweep is idempotent")
	_, err = proposal("create_appointment", "'appt:1'", "'2026-10-30 10:00:00'")
	require.NoError(t, err, "an expired proposal frees its dedupe key")

	assert.EqualValues(t, 4, affected(exec(familyQuery(t, "CancelPendingFamilyProposalsForLink",
		map[string]string{"now": now, "link_id": fmt.Sprint(f.link)}))))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT status FROM family_proposals WHERE id = ?`, second).Scan(&status))
	assert.Equal(t, "cancelled", status)
	assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM family_proposals WHERE pending_key IS NOT NULL`))

	// Share requests: domains must be JSON; only the addressee decides, only while live.
	shareReq := func(expires string) int64 {
		t.Helper()
		id, err := exec(familyQuery(t, "InsertFamilyShareRequest", map[string]string{
			"from_user_id": fmt.Sprint(f.member), "to_user_id": fmt.Sprint(f.owner), "link_id": fmt.Sprint(f.link),
			"domains": `'{"meds":"view"}'`, "expires_at": expires, "now": now,
		})).LastInsertId()
		require.NoError(t, err)
		return id
	}
	live, old := shareReq("'2026-10-30 10:00:00'"), shareReq("'2026-10-23 09:00:00'")
	_, err = db.ExecContext(ctx, `INSERT INTO family_share_requests (from_user_id, to_user_id, domains, expires_at) VALUES (?, ?, 'nope', NOW())`,
		f.member, f.owner)
	require.Error(t, err, "domains must be valid JSON")
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM (`+familyQuery(t, "ListPendingFamilyShareRequestsForOwner",
		map[string]string{"to_user_id": fmt.Sprint(f.owner), "now": now})+`) x`, 10))
	assert.Equal(t, 2, countRows(t, db, familyQuery(t, "CountFamilyShareRequestsForPairSince", map[string]string{
		"from_user_id": fmt.Sprint(f.member), "to_user_id": fmt.Sprint(f.owner), "since": "'2026-10-23 00:00:00'"})))
	decideShare := func(id, to int64) int64 {
		return affected(exec(familyQuery(t, "DecideFamilyShareRequest", map[string]string{
			"status": "'accepted'", "decided_at": now, "id": fmt.Sprint(id), "to_user_id": fmt.Sprint(to), "now": now,
		})))
	}
	assert.EqualValues(t, 0, decideShare(live, f.member), "only the owner it was sent to")
	assert.EqualValues(t, 0, decideShare(old, f.owner), "expired")
	assert.EqualValues(t, 1, decideShare(live, f.owner))
}

// T-M11-03: one nudge per (owner, target, local day, slot) and the per-pair day count; member alerts, member notices
// and attendance are idempotent; the purge-by-link statements and the FK cascade on a link delete remove exactly that
// link's rows.
func TestFamilySchemaB_NudgesAlertsNoticesPurge(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	f, other := newFamilyFixture(t, db, 2), newFamilyFixture(t, db, 3)
	exec := func(q string, args ...any) int64 {
		t.Helper()
		res, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		n, err := res.RowsAffected()
		require.NoError(t, err)
		return n
	}
	const now = "'2026-10-23 10:00:00'"

	nudge := func(fx familyFixture, targetType string, target int64, day, slot, sent string) int64 {
		return exec(familyQuery(t, "InsertFamilyNudge", map[string]string{
			"owner_id": fmt.Sprint(fx.owner), "member_id": fmt.Sprint(fx.member), "link_id": fmt.Sprint(fx.link),
			"target_type": "'" + targetType + "'", "target_id": fmt.Sprint(target), "local_date": "'" + day + "'",
			"slot": "'" + slot + "'", "sent_at": "'" + sent + "'",
		}))
	}
	assert.EqualValues(t, 1, nudge(f, "dose", 5, "2026-10-23", "08:00", "2026-10-23 09:05:00"))
	assert.EqualValues(t, 0, nudge(f, "dose", 5, "2026-10-23", "08:00", "2026-10-23 09:06:00"), "same dose slot: idempotent")
	assert.EqualValues(t, 1, nudge(f, "dose", 5, "2026-10-23", "20:00", "2026-10-23 21:05:00"), "another slot")
	assert.EqualValues(t, 1, nudge(f, "dose", 5, "2026-10-24", "08:00", "2026-10-24 09:05:00"), "another day")
	assert.EqualValues(t, 1, nudge(f, "appointment", f.appointment, "2026-10-23", "", "2026-10-23 10:00:00"))
	assert.EqualValues(t, 0, nudge(f, "appointment", f.appointment, "2026-10-23", "", "2026-10-23 10:01:00"))
	assert.EqualValues(t, 1, nudge(f, "appointment", 5, "2026-10-23", "08:00", "2026-10-23 11:00:00"),
		"target_type is part of the key")
	_, err := db.ExecContext(ctx, `INSERT INTO family_nudges (owner_id, member_id, link_id, target_type, target_id, local_date, slot, sent_at)
		VALUES (?, ?, 999999, 'dose', 1, '2026-10-23', '', NOW())`, f.owner, f.member)
	require.Error(t, err, "unlike INSERT IGNORE the idempotent insert still checks the link FK")
	assert.Equal(t, 4, countRows(t, db, familyQuery(t, "CountFamilyNudgesForPair", map[string]string{
		"member_id": fmt.Sprint(f.member), "owner_id": fmt.Sprint(f.owner),
		"from_at": "'2026-10-23 00:00:00'", "to_at": "'2026-10-24 00:00:00'",
	})))
	assert.Equal(t, 4, countRows(t, db, `SELECT COUNT(*) FROM (`+familyQuery(t, "ListFamilyNudgesForOwnerDay",
		map[string]string{"owner_id": fmt.Sprint(f.owner), "local_date": "'2026-10-23'"})+`) x`))

	for _, fx := range []familyFixture{f, other} {
		alert := func(src, hit int) int64 {
			return exec(familyQuery(t, "InsertFamilyMemberAlert", map[string]string{
				"member_id": fmt.Sprint(fx.member), "owner_id": fmt.Sprint(fx.owner), "link_id": fmt.Sprint(fx.link),
				"source_alert_id": fmt.Sprint(src), "source_hit": fmt.Sprint(hit), "level": "'urgent'", "now": now,
			}))
		}
		assert.EqualValues(t, 1, alert(10, 1))
		assert.EqualValues(t, 0, alert(10, 1), "a retried fan-out job relays once")
		assert.EqualValues(t, 1, alert(10, 2), "a re-fired alert relays again")

		notice := func(key string) int64 {
			return exec(familyQuery(t, "InsertFamilyMemberNotice", map[string]string{
				"member_id": fmt.Sprint(fx.member), "owner_id": fmt.Sprint(fx.owner), "link_id": fmt.Sprint(fx.link),
				"kind": "'appointment_tomorrow'", "dedupe_key": "'" + key + "'", "due_at": "'2026-10-23 09:00:00'", "now": now,
			}))
		}
		assert.EqualValues(t, 1, notice("appt_tomorrow:1:2026-10-24"))
		assert.EqualValues(t, 0, notice("appt_tomorrow:1:2026-10-24"))

		exec(familyQuery(t, "InsertLinkedUserNotification", map[string]string{
			"user_id": fmt.Sprint(fx.member), "type": "'companion'", "title": `'{"fa":"پیامی از خانواده در ریتمی"}'`,
			"body": "NULL", "action_url": "NULL", "data": "NULL", "link_id": fmt.Sprint(fx.link), "now": now,
		}))
		attend := familyQuery(t, "InsertFamilyAttendance", map[string]string{
			"reminder_id": fmt.Sprint(fx.appointment), "member_id": fmt.Sprint(fx.member), "now": now,
		})
		assert.EqualValues(t, 1, exec(attend))
		assert.EqualValues(t, 0, exec(attend))
	}
	// An unrelated notification of the member (no link) must survive every purge.
	exec(`INSERT INTO user_notifications (user_id, type, title, created_at, updated_at) VALUES (?, 'reminder', '{"fa":"x"}', NOW(), NOW())`, f.member)

	// Dispatcher round trip: due → sent once.
	dueQ := familyQuery(t, "ListDueFamilyMemberNotices", map[string]string{"now": now})
	assert.Equal(t, 2, countRows(t, db, `SELECT COUNT(*) FROM (`+dueQ+`) x`, 10))
	var noticeID int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT id FROM family_member_notices WHERE member_id = ?`, f.member).Scan(&noticeID))
	assert.EqualValues(t, 1, exec(familyQuery(t, "DeferFamilyMemberNotice", map[string]string{
		"due_at": "'2026-10-24 08:00:00'", "deferred_reason": "'quiet_hours'", "id": fmt.Sprint(noticeID)})))
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM (`+dueQ+`) x`, 10))
	mark := familyQuery(t, "MarkFamilyMemberNoticeSent", map[string]string{"now": now, "notification_id": "1", "id": fmt.Sprint(noticeID)})
	assert.EqualValues(t, 1, exec(mark))
	assert.EqualValues(t, 0, exec(mark))

	// Revoke path: purge by link touches only f's rows.
	link := map[string]string{"link_id": fmt.Sprint(f.link)}
	assert.EqualValues(t, 2, exec(familyQuery(t, "DeleteFamilyMemberAlertsForLink", link)))
	assert.EqualValues(t, 1, exec(familyQuery(t, "DeleteFamilyMemberNoticesForLink", link)))
	assert.EqualValues(t, 1, exec(familyQuery(t, "DeleteUserNotificationsForLink", link)))
	assert.EqualValues(t, 1, exec(familyQuery(t, "DeleteFamilyAttendanceForPair", map[string]string{
		"owner_id": fmt.Sprint(f.owner), "member_id": fmt.Sprint(f.member)})))
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, f.member), "the unlinked row stays")

	// Hard delete of a link (the owner's account deletion): the FK cascade removes every row of that link, and only those.
	cascadeCounts := func(fx familyFixture) []int {
		var out []int
		for _, table := range []string{"family_proposals", "family_share_requests", "family_nudges", "family_member_alerts",
			"family_member_notices"} {
			out = append(out, countRows(t, db, `SELECT COUNT(*) FROM `+table+` WHERE link_id = ?`, fx.link))
		}
		return append(out, countRows(t, db, `SELECT COUNT(*) FROM user_notifications WHERE link_id = ?`, fx.link))
	}
	for _, fx := range []familyFixture{f, other} {
		exec(`INSERT INTO family_proposals (subject_type, subject_id, link_id, actor_id, approver_id, kind, domain, expires_at, created_at)
			VALUES ('owner', ?, ?, ?, ?, 'create_med', 'meds', '2026-10-30 10:00:00', NOW())`, fx.owner, fx.link, fx.member, fx.owner)
		exec(`INSERT INTO family_share_requests (from_user_id, to_user_id, link_id, domains, expires_at, created_at)
			VALUES (?, ?, ?, '{"meds":"view"}', '2026-10-30 10:00:00', NOW())`, fx.member, fx.owner, fx.link)
	}
	exec(familyQuery(t, "InsertFamilyMemberAlert", map[string]string{"member_id": fmt.Sprint(f.member), "owner_id": fmt.Sprint(f.owner),
		"link_id": fmt.Sprint(f.link), "source_alert_id": "11", "source_hit": "1", "level": "'follow_up'", "now": now}))
	exec(familyQuery(t, "InsertFamilyMemberNotice", map[string]string{"member_id": fmt.Sprint(f.member), "owner_id": fmt.Sprint(f.owner),
		"link_id": fmt.Sprint(f.link), "kind": "'missed_dose'", "dedupe_key": "'k2'", "due_at": now, "now": now}))
	exec(familyQuery(t, "InsertLinkedUserNotification", map[string]string{"user_id": fmt.Sprint(f.member), "type": "'companion'",
		"title": `'{"fa":"x"}'`, "body": "NULL", "action_url": "NULL", "data": "NULL", "link_id": fmt.Sprint(f.link), "now": now}))
	assert.Equal(t, []int{1, 1, 5, 1, 1, 1}, cascadeCounts(f))
	assert.Equal(t, []int{1, 1, 0, 2, 1, 1}, cascadeCounts(other))

	exec(`DELETE FROM companions WHERE id = ?`, f.link)
	assert.Equal(t, []int{0, 0, 0, 0, 0, 0}, cascadeCounts(f))
	assert.Equal(t, []int{1, 1, 0, 2, 1, 1}, cascadeCounts(other), "other links are untouched")
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, f.member))

	// Attendance goes with the appointment; the member's account takes the rest of her rows with it.
	exec(`DELETE FROM reminders WHERE id = ?`, other.appointment)
	assert.Equal(t, 0, countRows(t, db, `SELECT COUNT(*) FROM family_attendance WHERE reminder_id = ?`, other.appointment))
	exec(`DELETE FROM users WHERE id = ?`, other.member)
	assert.Equal(t, 0, countRows(t, db, `SELECT
		(SELECT COUNT(*) FROM family_proposals WHERE actor_id = ?) + (SELECT COUNT(*) FROM family_share_requests WHERE from_user_id = ?) +
		(SELECT COUNT(*) FROM family_member_alerts WHERE member_id = ?) + (SELECT COUNT(*) FROM family_member_notices WHERE member_id = ?) +
		(SELECT COUNT(*) FROM family_attendance WHERE member_id = ?)`, other.member, other.member, other.member, other.member, other.member))
}
