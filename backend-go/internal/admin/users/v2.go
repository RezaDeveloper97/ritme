package users

// B-N9-04 + T-M10-32 (D-181; SPEC v6 §5.2): the user directory (GET /users/directory, /users/directory/export) and the
// user-detail overview (GET /users/:id/overview). The directory reads the nightly projection (admin_user_activity,
// activity.go) with indexed filters — one count and one page query, no per-request aggregate — and shows the
// reference code (A-09, refcode.go), a masked mobile, mode, plan, last activity, logged days in 30 days, data_quality,
// the engine's main warning and churn (A-08). The overview adds today's engine card (codes from engine_daily_snapshots)
// and the cycle-length history (lengths only, no dates). Symptoms, labs, notes and assistant chats are never read here;
// the goal and clinical rule titles stay behind an approved health-access grant (B-N9-02, D-84, T-M10-33).

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/segments"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

const exportMaxRows = 5000

// MaskMobile keeps the first four and last four digits: 09121234567 -> 0912•••4567.
func MaskMobile(m string) string {
	r := []rune(m)
	if len(r) < 9 {
		return "•••"
	}
	return string(r[:4]) + "•••" + string(r[len(r)-4:])
}

// refOf is the stored reference code (null until the signup hook or the backfill wrote it).
func refOf(s sql.NullString) any { return httpadmin.NullString(s) }

// readDirFilter reads the directory filters (segments.Keys) and an optional saved `segment` id underneath them
// (explicit parameters win). 422 on a bad value or an unknown / broken segment.
func (h *Handlers) readDirFilter(c fiber.Ctx) (segments.Filters, uint64, error) {
	var f segments.Filters
	for _, k := range segments.Keys {
		f.Set(k, strings.TrimSpace(c.Query(k)))
	}
	if f.Status == "all" {
		f.Status = ""
	}
	if bad := f.Invalid(); bad != "" {
		return f, 0, httpadmin.FieldError(bad, "Invalid filter.")
	}
	raw := c.Query("segment")
	if raw == "" {
		return f, 0, nil
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return f, 0, httpadmin.FieldError("segment", "Invalid filter.")
	}
	saved, ok, err := segments.Lookup(c.Context(), h.q, id)
	if err != nil {
		return f, 0, err
	}
	if !ok {
		return f, 0, httpadmin.FieldError("segment", "Invalid filter.")
	}
	return saved.Over(f), id, nil
}

// DirectoryPage is one directory page and the total for f at now: two queries.
func (h *Handlers) DirectoryPage(ctx context.Context, f segments.Filters, now time.Time, limit, offset int) (
	[]store.ListAdminDirectoryRow, int64, error) {
	total, err := h.q.CountAdminDirectory(ctx, f.CountParams(now))
	if err != nil {
		return nil, 0, err
	}
	rows, err := h.q.ListAdminDirectory(ctx, f.ListParams(now,
		int32(min(limit, 1<<30)), int32(min(offset, 1<<30)))) //nolint:gosec // G115: clamped
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func dirItem(r store.ListAdminDirectoryRow) *jsonx.OrderedMap {
	var logged any
	if r.LogDays30.Valid {
		logged = r.LogDays30.Int16
	}
	return jsonx.Obj(
		"id", r.ID,
		"ref", refOf(r.RefCode),
		"mobile_masked", MaskMobile(r.Mobile.String),
		"mode", httpadmin.NullString(r.Mode),
		"plan", httpadmin.NullString(r.Plan),
		"is_blocked", r.BlockedAt.Valid,
		"last_activity_at", httpadmin.Time(r.LastActivityAt),
		"logged_days_30", logged,
		"data_quality", httpadmin.NullString(r.DataQuality),
		"main_warning", httpadmin.NullString(r.MainWarning),
		"churn_risk", httpadmin.NullString(r.Churn),
		"projected", r.ProjectedUserID.Valid,
		"created_at", httpadmin.Time(r.CreatedAt),
	)
}

// Directory is GET /users/directory?q=&status=&plan=&activity=&cohort=&churn=&mode=&data_quality=&main_warning=
// &segment=&page=&per_page= — masked mobile, reference code, projection codes; no name, email or health value.
// q is a substring of name / mobile / email or an exact RT- code.
func (h *Handlers) Directory(c fiber.Ctx) error {
	f, seg, err := h.readDirFilter(c)
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, total, err := h.DirectoryPage(c.Context(), f, httpadmin.Now(c), p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, dirItem(r))
	}
	page := httpadmin.Page(items, p, int(total))
	filters := jsonx.Obj(f.Map()...)
	var segment any
	if seg > 0 {
		segment = seg
	}
	filters.Set("segment", segment)
	page.Set("filters", filters)
	page.Set("options", segments.Options())
	return httpadmin.OK(c, page)
}

// ExportDirectory is GET /users/directory/export (same filters): an anonymised CSV (ref, mode, plan, signup month,
// logged days, data_quality, churn, blocked) — no name, mobile, email or health value. Needs the export capability.
func (h *Handlers) ExportDirectory(c fiber.Ctx) error {
	if !httpadmin.Can(c, httpadmin.CapExport, httpadmin.LevelView) {
		return httpadmin.ForbiddenPermission(httpadmin.CapExport)
	}
	f, _, err := h.readDirFilter(c)
	if err != nil {
		return err
	}
	rows, _, err := h.DirectoryPage(c.Context(), f, httpadmin.Now(c), exportMaxRows, 0)
	if err != nil {
		return err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"ref", "mode", "plan", "signup_month", "logged_days_30", "data_quality", "churn_risk", "blocked"})
	for _, r := range rows {
		month := ""
		if r.CreatedAt.Valid {
			month = r.CreatedAt.Time.Format("2006-01")
		}
		logged := ""
		if r.LogDays30.Valid {
			logged = strconv.Itoa(int(r.LogDays30.Int16))
		}
		_ = w.Write([]string{r.RefCode.String, r.Mode.String, r.Plan.String, month, logged, r.DataQuality.String,
			r.Churn.String, strconv.FormatBool(r.BlockedAt.Valid)})
	}
	w.Flush()
	httpadmin.Audit(c, h.logger, "users.export", "users", 0, slog.Int("rows", len(rows)))
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="users-anonymised.csv"`)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.SendString(b.String())
}

const (
	overviewLimit = 20
	// cycleHistoryLengths is how many recent cycle lengths the detail shows.
	cycleHistoryLengths = 12
)

// CycleLengths turns period starts (newest first) into cycle lengths in days, newest first; the dates are dropped.
func CycleLengths(starts []civildate.Date) []int {
	out := []int{}
	for i := 0; i+1 < len(starts); i++ {
		if n := starts[i+1].DiffDays(starts[i]); n > 0 {
			out = append(out, n)
		}
	}
	return out
}

// engineCard is today's engine output from the newest snapshot (codes only), or nil without one.
func engineCard(s store.GetLatestEngineSnapshotRow) *jsonx.OrderedMap {
	set := engine.WarningSet(s.WarningsMask)
	warnings := []string{}
	for w := range set.All() {
		warnings = append(warnings, w.String())
	}
	var late any
	if s.DaysLate.Valid {
		late = s.DaysLate.Int16
	}
	var main any
	if m := MainWarning(s.WarningsMask); m != "" {
		main = m
	}
	return jsonx.Obj(
		"day", s.Day.String(),
		"mode", s.Mode,
		"main_phase", s.MainPhase,
		"subphase", s.Subphase,
		"confidence", s.Confidence,
		"data_quality", s.DataQuality,
		"warnings", warnings,
		"main_warning", main,
		"requires_user_input", set.RequiresUserInput(),
		"days_late", late,
	)
}

// openTitle reports whether a rule title is shown without a health grant: product and marketing titles name a
// campaign; clinical, safety, content and engine titles can name a condition.
func openTitle(kind string) bool { return kind == rules.KindProduct || kind == rules.KindMarketing }

// Overview is GET /users/:id/overview: the detail page in one payload — summary (projection), today's engine card,
// cycle-length history, recent messages with the rule (id, key, version, title) and challenges, subscriptions,
// consents, support tickets (no text), `health_access` and `health` (the goal, null without a grant).
func (h *Handlers) Overview(c fiber.Ctx) error {
	id, err := h.userID(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	acc, err := h.q.GetOverviewAccount(ctx, id)
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
	if grant != nil {
		// T-M10-33: audited before any health field is built; no row, no data (503).
		if err := httpadmin.AuditHealthRead(c, h.logger, id, grant.ID); err != nil {
			return err
		}
	}
	now := httpadmin.Now(c)
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()

	summary := jsonx.Obj(
		"signed_up_at", httpadmin.Time(acc.CreatedAt),
		"last_activity_at", nil,
		"logged_days_30", nil,
		"mode", nil,
		"plan", nil,
		"churn_risk", nil,
		"data_quality", nil,
		"main_warning", nil,
		"projected_on", nil,
		"companions", acc.Companions,
		"courses", acc.Courses,
		"is_blocked", acc.BlockedAt.Valid,
	)
	if a, err := h.q.GetAdminUserActivity(ctx, id); err == nil {
		summary.Set("last_activity_at", httpadmin.Time(a.LastActivityAt))
		summary.Set("logged_days_30", a.LogDays30)
		summary.Set("mode", a.Mode)
		summary.Set("plan", a.Plan)
		summary.Set("churn_risk", a.Churn)
		summary.Set("data_quality", httpadmin.NullString(a.DataQuality))
		summary.Set("main_warning", httpadmin.NullString(a.MainWarning))
		summary.Set("projected_on", a.UpdatedOn.String())
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var card any
	if s, err := h.q.GetLatestEngineSnapshot(ctx, store.GetLatestEngineSnapshotParams{
		UserID: id, Today: civildate.InTehran(now)}); err == nil {
		card = engineCard(s)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	starts, err := h.q.ListUserPeriodStarts(ctx, store.ListUserPeriodStartsParams{UserID: id, Limit: cycleHistoryLengths + 1})
	if err != nil {
		return err
	}

	subRows, err := h.q.ListOverviewSubscriptions(ctx, store.ListOverviewSubscriptionsParams{UserID: id, Limit: overviewLimit})
	if err != nil {
		return err
	}
	subs := make([]*jsonx.OrderedMap, 0, len(subRows))
	for _, s := range subRows {
		var plan any
		if s.PlanTitle.Valid {
			plan = i18n.PickString(s.PlanTitle.V, locale, def)
		}
		subs = append(subs, jsonx.Obj("status", s.Status, "source", s.Source,
			"starts_at", httpadmin.Time(sql.NullTime{Time: s.StartsAt, Valid: true}),
			"ends_at", httpadmin.Time(sql.NullTime{Time: s.EndsAt, Valid: true}), "auto_renew", s.AutoRenew, "plan", plan))
	}
	conRows, err := h.q.ListOverviewConsents(ctx, store.ListOverviewConsentsParams{UserID: id, Limit: overviewLimit})
	if err != nil {
		return err
	}
	consents := make([]*jsonx.OrderedMap, 0, len(conRows))
	for _, r := range conRows {
		var v any
		if r.Version.Valid {
			v = int64(r.Version.Int16)
		}
		consents = append(consents, jsonx.Obj("consent", r.Consent, "granted", r.Granted, "version", v,
			"granted_at", httpadmin.Time(r.GrantedAt), "revoked_at", httpadmin.Time(r.RevokedAt)))
	}
	supRows, err := h.q.ListOverviewSupportReports(ctx, store.ListOverviewSupportReportsParams{UserID: id, Limit: overviewLimit})
	if err != nil {
		return err
	}
	support := make([]*jsonx.OrderedMap, 0, len(supRows))
	for _, r := range supRows {
		support = append(support, jsonx.Obj("id", r.ID, "status", r.Status, "app_version", httpadmin.NullString(r.AppVersion),
			"created_at", httpadmin.Time(r.CreatedAt)))
	}
	chRows, err := h.q.ListOverviewChallenges(ctx, store.ListOverviewChallengesParams{UserID: id, Limit: overviewLimit})
	if err != nil {
		return err
	}
	challenges := make([]*jsonx.OrderedMap, 0, len(chRows))
	for _, r := range chRows {
		challenges = append(challenges, jsonx.Obj("challenge_id", r.ChallengeID,
			"title", i18n.PickString(r.Title, locale, def), "status", r.Status, "started_on", r.StartedOn.String()))
	}
	msgRows, err := h.q.ListOverviewMessages(ctx, store.ListOverviewMessagesParams{UserID: id, Limit: overviewLimit})
	if err != nil {
		return err
	}
	messages := make([]*jsonx.OrderedMap, 0, len(msgRows))
	for _, r := range msgRows {
		var title, version any
		if grant != nil || openTitle(r.Kind) {
			title = r.Title
		}
		if r.Version.Valid {
			version = r.Version.Int32
		}
		slot := r.Placement
		if r.Slot.Valid && r.Slot.String != "" {
			slot = r.Slot.String
		}
		messages = append(messages, jsonx.Obj(
			"rule", jsonx.Obj("id", r.RuleID, "key", httpadmin.NullString(r.RuleKey), "kind", r.Kind, "title", title,
				"version", version),
			"title", title, "placement", r.Placement, "slot", slot, "variant", r.Variant, "event", r.Event,
			"created_at", httpadmin.Time(r.CreatedAt)))
	}
	events := accountEvents(acc.CreatedAt, acc.MobileVerifiedAt, acc.BlockedAt, subs, consents, support, challenges)

	access := jsonx.Obj("granted", grant != nil, "expires_at", nil)
	var health any
	if grant != nil {
		access.Set("expires_at", httpadmin.Time(grant.ExpiresAt))
		health = jsonx.Obj("user_goal", httpadmin.NullString(acc.UserGoal))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"ref", refOf(acc.RefCode),
		"summary", summary,
		"engine", card,
		"cycle_lengths", CycleLengths(starts),
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
