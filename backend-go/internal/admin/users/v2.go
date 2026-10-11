package users

// B-N9-04: users list v2 (GET /users/directory, /users/directory/export) and the user-detail overview
// (GET /users/:id/overview). Nothing here reads a health value: the list carries a masked mobile, the plan, the
// last activity, a 30-day logging count and a churn bucket (counts of log DAYS only, never what was logged); the
// overview's engine / message-rule titles / goal stay null until an approved health-access grant (B-N9-02, D-84).

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

const exportMaxRows = 5000

var cohortRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// RefCode is the support-facing reference of a user ("RT-1F2A"): no name, no mobile.
func RefCode(id uint64) string { return fmt.Sprintf("RT-%04X", id) }

// MaskMobile keeps the first four and last four digits: 09121234567 -> 0912•••4567.
func MaskMobile(m string) string {
	r := []rune(m)
	if len(r) < 9 {
		return "•••"
	}
	return string(r[:4]) + "•••" + string(r[len(r)-4:])
}

type dirFilter struct {
	q, status, plan, activity, cohort, churn string
}

func readDirFilter(c fiber.Ctx) (dirFilter, error) {
	f := dirFilter{
		q: strings.TrimSpace(c.Query("q")), status: c.Query("status"), plan: c.Query("plan"),
		activity: c.Query("activity"), cohort: c.Query("cohort"), churn: c.Query("churn"),
	}
	in := func(v string, ok ...string) bool {
		if v == "" {
			return true
		}
		for _, o := range ok {
			if v == o {
				return true
			}
		}
		return false
	}
	switch {
	case !in(f.status, "active", "blocked", "all"),
		!in(f.plan, "free", "plus", "trial"),
		!in(f.activity, "today", "7d", "30d", "inactive"),
		!in(f.churn, "low", "medium", "high"),
		f.cohort != "" && !cohortRe.MatchString(f.cohort):
		return f, httpadmin.Fail(fiber.StatusUnprocessableEntity, "validation_error", "Invalid filter.")
	}
	return f, nil
}

// dirQuery builds the inner select (one row per user) plus the outer filters.
func (f dirFilter) build(now time.Time) (inner string, innerArgs []any, where string, whereArgs []any) {
	day := func(n int) string { return now.AddDate(0, 0, -n).Format("2006-01-02") }
	inner = "SELECT u.id, u.mobile, u.blocked_at, u.created_at, u.name, u.email, " +
		"CASE WHEN ps.id IS NULL THEN 'free' WHEN ps.source = 'trial_offer' THEN 'trial' ELSE 'plus' END AS plan, " +
		"GREATEST(u.updated_at, IFNULL(a.last_log, u.updated_at)) AS last_activity, " +
		"IFNULL(a.d30, 0) AS d30, IFNULL(a.a14, 0) AS a14, IFNULL(a.b14, 0) AS b14 " +
		"FROM users u " +
		"LEFT JOIN plus_subscriptions ps ON ps.id = (SELECT MAX(s.id) FROM plus_subscriptions s " +
		"WHERE s.user_id = u.id AND s.status = 'active' AND s.ends_at > ?) " +
		"LEFT JOIN (SELECT user_id, MAX(updated_at) AS last_log, COUNT(*) AS d30, " +
		"SUM(log_date >= ?) AS a14, SUM(log_date >= ? AND log_date < ?) AS b14 " +
		"FROM daily_health_logs WHERE log_date >= ? GROUP BY user_id) a ON a.user_id = u.id"
	innerArgs = []any{now.Format("2006-01-02 15:04:05"), day(14), day(28), day(14), day(30)}

	conds := []string{"1=1"}
	add := func(cond string, args ...any) {
		conds = append(conds, cond)
		whereArgs = append(whereArgs, args...)
	}
	if f.q != "" {
		p := "%" + escapeLike(f.q) + "%"
		add("(IFNULL(name,'') LIKE ? OR IFNULL(mobile,'') LIKE ? OR IFNULL(email,'') LIKE ?)", p, p, p)
	}
	switch f.status {
	case "active":
		add("blocked_at IS NULL")
	case "blocked":
		add("blocked_at IS NOT NULL")
	}
	if f.plan != "" {
		add("plan = ?", f.plan)
	}
	switch f.activity {
	case "today":
		add("last_activity >= ?", now.Format("2006-01-02")+" 00:00:00")
	case "7d":
		add("last_activity >= ?", day(7)+" 00:00:00")
	case "30d":
		add("last_activity >= ?", day(30)+" 00:00:00")
	case "inactive":
		add("last_activity < ?", day(30)+" 00:00:00")
	}
	if f.cohort != "" {
		add("DATE_FORMAT(created_at, '%Y-%m') = ?", f.cohort)
	}
	switch f.churn {
	case "low":
		add("(b14 = 0 OR a14 * 10 >= b14 * 7)")
	case "medium":
		add("(b14 > 0 AND a14 * 10 < b14 * 7 AND a14 * 10 >= b14 * 3)")
	case "high":
		add("(b14 > 0 AND a14 * 10 < b14 * 3)")
	}
	where = strings.Join(conds, " AND ")
	return inner, innerArgs, where, whereArgs
}

func churnOf(a14, b14 int64) string {
	switch {
	case b14 == 0 || a14*10 >= b14*7:
		return "low"
	case a14*10 >= b14*3:
		return "medium"
	default:
		return "high"
	}
}

type dirRow struct {
	id            uint64
	mobile        sql.NullString
	blockedAt     sql.NullTime
	createdAt     sql.NullTime
	plan          string
	lastActivity  sql.NullTime
	d30, a14, b14 int64
}

func (h *Handlers) dirRows(ctx context.Context, f dirFilter, now time.Time, limit, offset int) ([]dirRow, int64, error) {
	inner, ia, where, wa := f.build(now)
	var total int64
	//nolint:gosec // G202: inner/where are fixed fragments; every value is a bound argument
	if err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ("+inner+") t WHERE "+where,
		append(append([]any{}, ia...), wa...)...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args := append(append(append([]any{}, ia...), wa...), limit, offset)
	//nolint:gosec // G202: see above
	rs, err := h.db.QueryContext(ctx, "SELECT id, mobile, blocked_at, created_at, plan, last_activity, d30, a14, b14 FROM ("+
		inner+") t WHERE "+where+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rs.Close()
	var out []dirRow
	for rs.Next() {
		var r dirRow
		if err := rs.Scan(&r.id, &r.mobile, &r.blockedAt, &r.createdAt, &r.plan, &r.lastActivity, &r.d30, &r.a14, &r.b14); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rs.Err()
}

// Directory is GET /users/directory?q=&status=&plan=free|plus|trial&activity=today|7d|30d|inactive&cohort=YYYY-MM
// &churn=low|medium|high&page=&per_page= — the v2 list: masked mobile, no name / email / health value.
func (h *Handlers) Directory(c fiber.Ctx) error {
	f, err := readDirFilter(c)
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, total, err := h.dirRows(c.Context(), f, httpadmin.Now(c), p.PerPage, min(p.Offset(), 1<<30))
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonx.Obj(
			"id", r.id,
			"ref", RefCode(r.id),
			"mobile_masked", MaskMobile(r.mobile.String),
			"plan", r.plan,
			"is_blocked", r.blockedAt.Valid,
			"last_activity_at", httpadmin.Time(r.lastActivity),
			"logged_days_30", r.d30,
			"churn_risk", churnOf(r.a14, r.b14),
			"created_at", httpadmin.Time(r.createdAt),
		))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("q", f.q, "status", f.status, "plan", f.plan, "activity", f.activity,
		"cohort", f.cohort, "churn", f.churn))
	return httpadmin.OK(c, page)
}

// ExportDirectory is GET /users/directory/export (same filters): an anonymised CSV (ref, plan, signup month,
// logged days, churn) — no name, mobile, email or health value. Needs the export capability besides users.
func (h *Handlers) ExportDirectory(c fiber.Ctx) error {
	if !httpadmin.Can(c, httpadmin.CapExport, httpadmin.LevelView) {
		return httpadmin.ForbiddenPermission(httpadmin.CapExport)
	}
	f, err := readDirFilter(c)
	if err != nil {
		return err
	}
	rows, _, err := h.dirRows(c.Context(), f, httpadmin.Now(c), exportMaxRows, 0)
	if err != nil {
		return err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"ref", "plan", "signup_month", "logged_days_30", "churn_risk", "blocked"})
	for _, r := range rows {
		month := ""
		if r.createdAt.Valid {
			month = r.createdAt.Time.Format("2006-01")
		}
		_ = w.Write([]string{RefCode(r.id), r.plan, month, strconv.FormatInt(r.d30, 10), churnOf(r.a14, r.b14),
			strconv.FormatBool(r.blockedAt.Valid)})
	}
	w.Flush()
	httpadmin.Audit(c, h.logger, "users.export", "users", 0, slog.Int("rows", len(rows)))
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="users-anonymised.csv"`)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.SendString(b.String())
}

const overviewLimit = 20

// Overview is GET /users/:id/overview: the non-health tabs of the detail page in one payload (summary, events,
// messages & challenges, subscription, consents, support) plus `health_access` and `health` — the health-gated
// part (goal, message-rule titles) is null unless the admin holds an approved grant for this user.
func (h *Handlers) Overview(c fiber.Ctx) error {
	id, err := h.userID(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	var (
		created, verified, blocked, updated sql.NullTime
		goal                                sql.NullString
	)
	err = h.db.QueryRowContext(ctx, `SELECT u.created_at, u.mobile_verified_at, u.blocked_at, u.updated_at,
		(SELECT p.user_goal FROM user_profiles p WHERE p.user_id = u.id ORDER BY p.id LIMIT 1)
		FROM users u WHERE u.id = ?`, id).Scan(&created, &verified, &blocked, &updated, &goal)
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("User")
	}
	if err != nil {
		return err
	}
	grant, err := httpadmin.HealthGrant(c, h.q, id)
	if err != nil {
		return err
	}
	now := httpadmin.Now(c)
	day30 := now.AddDate(0, 0, -30).Format("2006-01-02")
	var d30 int64
	var lastLog sql.NullTime
	if err := h.db.QueryRowContext(ctx, `SELECT COUNT(*), MAX(updated_at) FROM daily_health_logs
		WHERE user_id = ? AND log_date >= ?`, id, day30).Scan(&d30, &lastLog); err != nil {
		return err
	}
	var companions, courses int64
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM companions WHERE owner_id = ? AND status = 'active'`, id).Scan(&companions)
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT course_id) FROM learning_progress WHERE user_id = ?`, id).Scan(&courses)
	last := updated
	if lastLog.Valid && (!last.Valid || lastLog.Time.After(last.Time)) {
		last = lastLog
	}

	subs, err := h.queryRows(ctx, `SELECT s.status, s.source, s.starts_at, s.ends_at, s.auto_renew,
		JSON_UNQUOTE(JSON_EXTRACT(pl.title, '$.fa')) FROM plus_subscriptions s
		LEFT JOIN plus_plans pl ON pl.id = s.plan_id WHERE s.user_id = ? ORDER BY s.id DESC LIMIT ?`, 6,
		func(rs *sql.Rows) (*jsonx.OrderedMap, error) {
			var status, source string
			var from, to sql.NullTime
			var renew bool
			var plan sql.NullString
			if err := rs.Scan(&status, &source, &from, &to, &renew, &plan); err != nil {
				return nil, err
			}
			return jsonx.Obj("status", status, "source", source, "starts_at", httpadmin.Time(from),
				"ends_at", httpadmin.Time(to), "auto_renew", renew, "plan", httpadmin.NullString(plan)), nil
		}, id, overviewLimit)
	if err != nil {
		return err
	}
	consents, err := h.queryRows(ctx, `SELECT consent, granted, version, granted_at, revoked_at FROM user_consents
		WHERE user_id = ? ORDER BY consent LIMIT ?`, 6,
		func(rs *sql.Rows) (*jsonx.OrderedMap, error) {
			var key string
			var granted bool
			var ver sql.NullInt64
			var gAt, rAt sql.NullTime
			if err := rs.Scan(&key, &granted, &ver, &gAt, &rAt); err != nil {
				return nil, err
			}
			var v any
			if ver.Valid {
				v = ver.Int64
			}
			return jsonx.Obj("consent", key, "granted", granted, "version", v,
				"granted_at", httpadmin.Time(gAt), "revoked_at", httpadmin.Time(rAt)), nil
		}, id, overviewLimit)
	if err != nil {
		return err
	}
	support, err := h.queryRows(ctx, `SELECT id, status, app_version, created_at FROM support_reports
		WHERE user_id = ? ORDER BY id DESC LIMIT ?`, 6,
		func(rs *sql.Rows) (*jsonx.OrderedMap, error) { // never the report text: it is free-form and may hold health data
			var rid uint64
			var status string
			var app sql.NullString
			var at sql.NullTime
			if err := rs.Scan(&rid, &status, &app, &at); err != nil {
				return nil, err
			}
			return jsonx.Obj("id", rid, "status", status, "app_version", httpadmin.NullString(app),
				"created_at", httpadmin.Time(at)), nil
		}, id, overviewLimit)
	if err != nil {
		return err
	}
	challenges, err := h.queryRows(ctx, `SELECT JSON_UNQUOTE(JSON_EXTRACT(ch.title, '$.fa')), e.status, e.started_on
		FROM challenge_enrollments e JOIN challenges ch ON ch.id = e.challenge_id
		WHERE e.user_id = ? ORDER BY e.id DESC LIMIT ?`, 6,
		func(rs *sql.Rows) (*jsonx.OrderedMap, error) {
			var title sql.NullString
			var status string
			var started sql.NullTime
			if err := rs.Scan(&title, &status, &started); err != nil {
				return nil, err
			}
			return jsonx.Obj("title", httpadmin.NullString(title), "status", status,
				"started_on", httpadmin.Time(started)), nil
		}, id, overviewLimit)
	if err != nil {
		return err
	}
	// Messages shown: placement + outcome always; the rule title (it can name a condition) only with a grant.
	messages, err := h.queryRows(ctx, `SELECT r.title, r.placement, e.event, e.created_at FROM message_events e
		JOIN message_rules r ON r.id = e.rule_id WHERE e.user_id = ? ORDER BY e.id DESC LIMIT ?`, 6,
		func(rs *sql.Rows) (*jsonx.OrderedMap, error) {
			var title, placement, event string
			var at sql.NullTime
			if err := rs.Scan(&title, &placement, &event, &at); err != nil {
				return nil, err
			}
			var t any
			if grant != nil {
				t = title
			}
			return jsonx.Obj("title", t, "placement", placement, "event", event, "created_at", httpadmin.Time(at)), nil
		}, id, overviewLimit)
	if err != nil {
		return err
	}
	events := accountEvents(created, verified, blocked, subs, consents, support, challenges)

	access := jsonx.Obj("granted", grant != nil, "expires_at", nil)
	var health any
	if grant != nil {
		access.Set("expires_at", httpadmin.Time(grant.ExpiresAt))
		httpadmin.Audit(c, h.logger, "health_access.read", "user", id, slog.Uint64("request_id", grant.ID))
		health = jsonx.Obj("user_goal", httpadmin.NullString(goal))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"ref", RefCode(id),
		"summary", jsonx.Obj(
			"signed_up_at", httpadmin.Time(created),
			"last_activity_at", httpadmin.Time(last),
			"logged_days_30", d30,
			"companions", companions,
			"courses", courses,
			"is_blocked", blocked.Valid,
		),
		"events", events,
		"messages", messages,
		"challenges", challenges,
		"subscriptions", subs,
		"consents", consents,
		"support_reports", support,
		"health_access", access,
		"health", health,
	))
}

func (h *Handlers) queryRows(ctx context.Context, q string, _ int, scan func(*sql.Rows) (*jsonx.OrderedMap, error),
	args ...any) ([]*jsonx.OrderedMap, error) {
	rs, err := h.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	out := []*jsonx.OrderedMap{}
	for rs.Next() {
		m, err := scan(rs)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rs.Err()
}

// accountEvents is the non-health timeline: sign-up, verification, block, plan changes, consents, support
// tickets, challenge joins (newest first, ISO strings sort lexically within one zone).
func accountEvents(created, verified, blocked sql.NullTime, groups ...[]*jsonx.OrderedMap) []*jsonx.OrderedMap {
	out := []*jsonx.OrderedMap{}
	add := func(kind string, at any) {
		if at != nil {
			out = append(out, jsonx.Obj("kind", kind, "at", at))
		}
	}
	add("signed_up", httpadmin.Time(created))
	add("mobile_verified", httpadmin.Time(verified))
	add("blocked", httpadmin.Time(blocked))
	kinds := []struct{ kind, field string }{
		{"subscription", "starts_at"}, {"consent", "granted_at"}, {"support_report", "created_at"},
		{"challenge_joined", "started_on"},
	}
	for i, g := range groups {
		if i >= len(kinds) {
			break
		}
		for _, m := range g {
			if v, ok := m.Get(kinds[i].field); ok {
				add(kinds[i].kind, v)
			}
		}
	}
	// stable newest-first by the ISO string
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && fmt.Sprint(atOf(out[j])) > fmt.Sprint(atOf(out[j-1])); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func atOf(m *jsonx.OrderedMap) any { v, _ := m.Get("at"); return v }
