// Package rbac is the admin API for roles & permissions, the audit log and temporary health-data access
// (bloom B-N9-02, D-84; docs/go-migration/admin-api.md §21).
//
//   - Roles: the capability matrix per role. super / editor are fixed; a super admin edits (or resets to the code
//     defaults) the other roles. Role assignment stays on PUT /admins/:id.
//   - Audit log: every httpadmin.Audit call writes an admin_audit_logs row (ids, flags, short codes only); this
//     package lists, filters and exports them (CSV).
//   - Health access: an admin whose role holds the health capability (clinical support) asks for ONE user's health
//     data with a reason and a ticket; ANOTHER super admin approves; the grant expires after 30 minutes. Reads go
//     through httpadmin.HealthGrant and are audited (health_access.read) before any data leaves: no row, 503
//     (T-M10-33).
//   - Audit chain: GET /audit/verify?from= (super) walks the hash chain of admin_audit_logs (internal/admin/auditchain,
//     SPEC v6 §5.8, D-182) and reports the first break.
package rbac

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/auditchain"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Health access request statuses (stored); "expired" is derived from expires_at.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusRevoked  = "revoked"
	StatusExpired  = "expired"
)

// Error codes.
const (
	CodeRoleLocked   = "role_locked"   // 422: super / editor matrices are fixed
	CodeSelfApproval = "self_approval" // 422: the requester cannot approve their own request
	CodeNotPending   = "not_pending"   // 409
	CodeNotActive    = "not_active"    // 409
	// B-N9-02b: the approver is not independent of the requester (created / changed by them in the last 30 days, or
	// an account younger than 24 hours).
	CodeApproverNotIndependent = "approver_not_independent" // 422
	CodeRequesterNotPermitted  = "requester_not_permitted"  // 422: the requester lost the health capability
)

// B-N9-02b approval rules.
const (
	pendingTTL         = 24 * time.Hour      // a pending request expires unanswered after a day
	approverMinAge     = 24 * time.Hour      // the approver's account must be at least a day old
	independenceWindow = 30 * 24 * time.Hour // no account change of the approver by the requester in this window
)

// exportMaxRows caps one CSV export.
const exportMaxRows = 10000

// Handlers serves the endpoints.
type Handlers struct {
	q      *store.Queries
	chain  *auditchain.Chain
	kit    *httpadmin.Kit
	logger *slog.Logger
}

// NewHandlers wires the handlers.
func NewHandlers(db *sql.DB, kit *httpadmin.Kit, logger *slog.Logger) *Handlers {
	return &Handlers{q: store.New(db), chain: auditchain.New(db), kit: kit, logger: logger}
}

// Routes registers the endpoints (paths under httpadmin.Prefix).
func (h *Handlers) Routes(add func(method, path string, chain httpadmin.Chain), kit *httpadmin.Kit) {
	get, post, put := fiber.MethodGet, fiber.MethodPost, fiber.MethodPut
	add(get, "/roles", kit.Super(h.ListRoles))
	add(put, "/roles/:role", kit.Super(h.UpdateRole))
	add(post, "/roles/:role/reset", kit.Super(h.ResetRole))

	add(get, "/audit-logs", kit.Admin(h.ListAudit))
	add(get, "/audit-logs/export", kit.Admin(h.ExportAudit))
	add(get, "/audit/verify", kit.Super(h.VerifyAudit))

	add(get, "/health-access", kit.Admin(h.ListAccess))
	add(post, "/health-access", kit.Admin(h.RequestAccess))
	add(get, "/health-access/users/:id", kit.Admin(h.ReadHealth))
	add(post, "/health-access/:id/approve", kit.Super(h.Approve))
	add(post, "/health-access/:id/reject", kit.Super(h.Reject))
	add(post, "/health-access/:id/revoke", kit.Admin(h.Revoke))
}

// ---------------------------------------------------------------------------
// Roles

func capabilitiesJSON() []any {
	out := make([]any, 0, len(httpadmin.Capabilities))
	for _, c := range httpadmin.Capabilities {
		out = append(out, jsonx.Obj("key", c.Key, "levels", c.Levels))
	}
	return out
}

// ListRoles is GET /roles: {capabilities:[{key, levels}], roles:[{key, editable, customized, admins, permissions}]}.
func (h *Handlers) ListRoles(c fiber.Ctx) error {
	stored, err := h.q.ListAdminRoles(c.Context())
	if err != nil {
		return err
	}
	customized := map[string]bool{}
	for _, r := range stored {
		customized[r.Role] = true
	}
	counts, err := h.q.CountAdminsByRole(c.Context())
	if err != nil {
		return err
	}
	byRole := map[string]int64{}
	for _, r := range counts {
		byRole[r.Role] = r.Admins
	}
	roles := make([]any, 0, len(httpadmin.Roles))
	for _, role := range httpadmin.Roles {
		perms, err := h.kit.PermissionsFor(c.Context(), role)
		if err != nil {
			return err
		}
		roles = append(roles, jsonx.Obj(
			"key", role,
			"editable", httpadmin.EditableRole(role),
			"customized", customized[role],
			"admins", byRole[role],
			"permissions", httpadmin.PermissionsJSON(perms),
		))
	}
	return httpadmin.OK(c, jsonx.Obj("capabilities", capabilitiesJSON(), "roles", roles))
}

func (h *Handlers) editableRole(c fiber.Ctx) (string, error) {
	role := c.Params("role")
	if !httpadmin.EditableRole(role) {
		if role == httpadmin.RoleSuper || role == httpadmin.RoleEditor {
			return "", httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeRoleLocked, "This role's permissions are fixed.")
		}
		return "", httpadmin.NotFound("Role")
	}
	return role, nil
}

func (h *Handlers) roleJSON(c fiber.Ctx, role string, customized bool) (*jsonx.OrderedMap, error) {
	perms, err := h.kit.PermissionsFor(c.Context(), role)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj("role", jsonx.Obj("key", role, "editable", true, "customized", customized,
		"permissions", httpadmin.PermissionsJSON(perms))), nil
}

// UpdateRole is PUT /roles/:role {permissions:{capability: "view"|"manage"|"none"}} (the whole matrix row set).
func (h *Handlers) UpdateRole(c fiber.Ctx) error {
	role, err := h.editableRole(c)
	if err != nil {
		return err
	}
	if _, err := httpadmin.Validate(c, validation.Rules{validation.F("permissions", "present|array")}); err != nil {
		return err
	}
	var body struct {
		Permissions map[string]string `json:"permissions"`
	}
	if json.Unmarshal(c.Body(), &body) != nil {
		return httpadmin.FieldError("permissions", "The permissions must map capabilities to view, manage or none.")
	}
	perms, ok := httpadmin.ParsePermissions(body.Permissions)
	if !ok {
		return httpadmin.FieldError("permissions", "The permissions must map capabilities to view, manage or none.")
	}
	raw, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	a := httpadmin.CurrentAdmin(c)
	if err := h.q.UpsertAdminRole(c.Context(), store.UpsertAdminRoleParams{
		Role: role, Permissions: string(raw), Now: httpadmin.DBTime(httpadmin.Now(c)),
		UpdatedBy: sql.NullInt64{Int64: int64(a.ID), Valid: true}, //nolint:gosec // ids fit int64
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "role.update", "role", 0, slog.String("role", role), slog.Int("capabilities", len(perms)))
	out, err := h.roleJSON(c, role, true)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, out, "Role updated.")
}

// ResetRole is POST /roles/:role/reset: back to the code defaults.
func (h *Handlers) ResetRole(c fiber.Ctx) error {
	role, err := h.editableRole(c)
	if err != nil {
		return err
	}
	if err := h.q.DeleteAdminRole(c.Context(), role); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "role.reset", "role", 0, slog.String("role", role))
	out, err := h.roleJSON(c, role, false)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, out, "Role reset.")
}

// ---------------------------------------------------------------------------
// Audit log

type auditFilter struct {
	adminID, targetID uint64
	action, target    string
	from, to          sql.NullTime
}

// parseDay reads ?from / ?to as YYYY-MM-DD (Asia/Tehran days); to is inclusive.
func parseDay(s string, next bool) (sql.NullTime, bool) {
	if s == "" {
		return sql.NullTime{}, true
	}
	t, err := time.ParseInLocation(time.DateOnly, s, civildate.Tehran)
	if err != nil {
		return sql.NullTime{}, false
	}
	if next {
		t = t.AddDate(0, 0, 1)
	}
	return sql.NullTime{Time: t, Valid: true}, true
}

func readFilter(c fiber.Ctx) (auditFilter, error) {
	f := auditFilter{action: strings.TrimSpace(c.Query("action")), target: strings.TrimSpace(c.Query("target_type"))}
	var ok bool
	if f.from, ok = parseDay(c.Query("from"), false); !ok {
		return f, httpadmin.FieldError("from", "The from field must be a date (YYYY-MM-DD).")
	}
	if f.to, ok = parseDay(c.Query("to"), true); !ok {
		return f, httpadmin.FieldError("to", "The to field must be a date (YYYY-MM-DD).")
	}
	f.adminID, _ = strconv.ParseUint(c.Query("admin_id"), 10, 64)
	f.targetID, _ = strconv.ParseUint(c.Query("target_id"), 10, 64)
	if len(f.action) > 64 || len(f.target) > 64 {
		return f, httpadmin.FieldError("action", "The action field must not be greater than 64 characters.")
	}
	return f, nil
}

func (f auditFilter) list(limit, offset int) store.ListAdminAuditLogsParams {
	return store.ListAdminAuditLogsParams{
		AdminID: sql.NullInt64{Int64: int64(f.adminID), Valid: true}, //nolint:gosec // ids fit int64
		Action:  f.action, TargetType: f.target, TargetID: f.targetID, FromAt: f.from, ToAt: f.to,
		Limit: int32(limit), Offset: int32(min(offset, 1<<30)), //nolint:gosec // bounded
	}
}

func detailJSON(d sql.NullString) any {
	if !d.Valid {
		return nil
	}
	var m map[string]any
	if json.Unmarshal([]byte(d.String), &m) != nil {
		return nil
	}
	return m
}

func auditJSON(r store.ListAdminAuditLogsRow, showIP bool) *jsonx.OrderedMap {
	var admin any
	if r.AdminID.Valid {
		admin = jsonx.Obj("id", r.AdminID.Int64, "name", httpadmin.NullString(r.AdminName),
			"role", httpadmin.NullString(r.AdminRole))
	}
	return jsonx.Obj(
		"id", r.ID,
		"admin", admin,
		"action", r.Action,
		"target_type", r.TargetType,
		"target_id", r.TargetID,
		"ip", ipOf(r.Ip, showIP),
		"detail", detailJSON(r.Detail),
		"created_at", httpadmin.Time(r.CreatedAt),
	)
}

// isSuper: only a super admin sees client IPs in the audit log (B-N9-02b L6).
func isSuper(c fiber.Ctx) bool {
	a := httpadmin.CurrentAdmin(c)
	return a != nil && a.Role == httpadmin.RoleSuper
}

func ipOf(ip string, show bool) any {
	if !show {
		return nil
	}
	return ip
}

// ListAudit is GET /audit-logs?admin_id&action(prefix)&target_type&target_id&from&to&page&per_page (newest first).
func (h *Handlers) ListAudit(c fiber.Ctx) error {
	f, err := readFilter(c)
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	lp := f.list(p.PerPage, p.Offset())
	total, err := h.q.CountAdminAuditLogs(c.Context(), store.CountAdminAuditLogsParams{
		AdminID: lp.AdminID, Action: lp.Action, TargetType: lp.TargetType, TargetID: lp.TargetID,
		FromAt: lp.FromAt, ToAt: lp.ToAt,
	})
	if err != nil {
		return err
	}
	rows, err := h.q.ListAdminAuditLogs(c.Context(), lp)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, auditJSON(r, isSuper(c)))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// ExportAudit is GET /audit-logs/export (same filters): text/csv, at most exportMaxRows rows. Needs the export
// capability besides audit.
func (h *Handlers) ExportAudit(c fiber.Ctx) error {
	if !httpadmin.Can(c, httpadmin.CapExport, httpadmin.LevelView) {
		return httpadmin.ForbiddenPermission(httpadmin.CapExport)
	}
	f, err := readFilter(c)
	if err != nil {
		return err
	}
	rows, err := h.q.ListAdminAuditLogs(c.Context(), f.list(exportMaxRows, 0))
	if err != nil {
		return err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"id", "created_at", "admin_id", "admin_name", "admin_role", "action", "target_type",
		"target_id", "ip", "detail"})
	showIP := isSuper(c)
	for _, r := range rows {
		adminID := ""
		if r.AdminID.Valid {
			adminID = strconv.FormatInt(r.AdminID.Int64, 10)
		}
		created := ""
		if r.CreatedAt.Valid {
			created = r.CreatedAt.Time.Format(time.DateTime)
		}
		ip := ""
		if showIP {
			ip = r.Ip
		}
		_ = w.Write([]string{strconv.FormatUint(r.ID, 10), created, adminID, csvSafe(r.AdminName.String),
			csvSafe(r.AdminRole.String), csvSafe(r.Action), csvSafe(r.TargetType), strconv.FormatUint(r.TargetID, 10),
			csvSafe(ip), csvSafe(r.Detail.String)})
	}
	w.Flush()
	httpadmin.Audit(c, h.logger, "audit_log.export", "audit_log", 0, slog.Int("rows", len(rows)))
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="audit-log.csv"`)
	return c.SendString(b.String())
}

// VerifyAudit is GET /audit/verify?from=<row id> (super): walks the audit hash chain from that row (default: the
// whole log, genesis included) and answers {ok, started, from, checked, genesis_last_id, head:{id, hash},
// break:{id, reason}|null}. Read-only; it never repairs anything.
func (h *Handlers) VerifyAudit(c fiber.Ctx) error {
	var from uint64
	if s := strings.TrimSpace(c.Query("from")); s != "" {
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return httpadmin.FieldError("from", "The from field must be an audit row id.")
		}
		from = n
	}
	rep, err := h.chain.Verify(c.Context(), from)
	if err != nil {
		return err
	}
	var brk any
	if rep.Break != nil {
		brk = jsonx.Obj("id", rep.Break.ID, "reason", rep.Break.Reason)
	}
	return httpadmin.OK(c, jsonx.Obj(
		"ok", rep.Break == nil,
		"started", rep.Started,
		"from", rep.From,
		"checked", rep.Checked,
		"genesis_last_id", rep.GenesisLastID,
		"head", jsonx.Obj("id", rep.HeadID, "hash", rep.HeadHash),
		"break", brk,
	))
}

// csvSafe defuses spreadsheet formulas: a cell starting with a tab / CR, or with =, +, -, @ after leading
// whitespace (spreadsheets trim it), gets a leading quote.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	if strings.ContainsRune("\t\r", rune(s[0])) {
		return "'" + s
	}
	if t := strings.TrimLeft(s, " \t\r\n"); t != "" && strings.ContainsRune("=+-@", rune(t[0])) {
		return "'" + s
	}
	return s
}

// ---------------------------------------------------------------------------
// Health access

// accessStatus derives "expired": an approved grant past expires_at, or a request pending for over pendingTTL.
func accessStatus(status string, expires, created sql.NullTime, now time.Time) string {
	if status == StatusApproved && (!expires.Valid || !expires.Time.After(now)) {
		return StatusExpired
	}
	if status == StatusPending && pendingExpired(created, now) {
		return StatusExpired
	}
	return status
}

func pendingExpired(created sql.NullTime, now time.Time) bool {
	return !created.Valid || !created.Time.Add(pendingTTL).After(now)
}

func accessJSON(r store.GetHealthAccessRequestRow, now time.Time) *jsonx.OrderedMap {
	var decidedBy any
	if r.DecidedBy.Valid {
		decidedBy = jsonx.Obj("id", r.DecidedBy.Int64, "name", httpadmin.NullString(r.DecidedByName))
	}
	return jsonx.Obj(
		"id", r.ID,
		"admin", jsonx.Obj("id", r.AdminID, "name", httpadmin.NullString(r.AdminName)),
		"user_id", r.UserID,
		"reason", r.Reason,
		"ticket", r.Ticket,
		"status", accessStatus(r.Status, r.ExpiresAt, r.CreatedAt, now),
		"decided_by", decidedBy,
		"decided_at", httpadmin.Time(r.DecidedAt),
		"expires_at", httpadmin.Time(r.ExpiresAt),
		"created_at", httpadmin.Time(r.CreatedAt),
	)
}

// ListAccess is GET /health-access?status&page: a super admin sees every request (the approval queue), anyone else
// only their own.
func (h *Handlers) ListAccess(c fiber.Ctx) error {
	a := httpadmin.CurrentAdmin(c)
	var adminID uint64
	if a.Role != httpadmin.RoleSuper {
		adminID = a.ID
	}
	status := c.Query("status")
	switch status {
	case "", StatusPending, StatusApproved, StatusRejected, StatusRevoked:
	default:
		status = ""
	}
	p := httpadmin.PageOf(c)
	total, err := h.q.CountHealthAccessRequests(c.Context(), store.CountHealthAccessRequestsParams{
		AdminID: adminID, Status: status,
	})
	if err != nil {
		return err
	}
	rows, err := h.q.ListHealthAccessRequests(c.Context(), store.ListHealthAccessRequestsParams{
		AdminID: adminID, Status: status, Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // bounded
	})
	if err != nil {
		return err
	}
	now := httpadmin.DBTime(httpadmin.Now(c)).Time
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, accessJSON(store.GetHealthAccessRequestRow(r), now))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// RequestAccess is POST /health-access {user_id, reason (10–500), ticket (≤64)} → 201 {request} (pending).
func (h *Handlers) RequestAccess(c fiber.Ctx) error {
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("user_id", "required|integer|min:1"),
		validation.F("reason", "required|string|min:10|max:500"),
		validation.F("ticket", "required|string|max:64"),
	})
	if err != nil {
		return err
	}
	userID, _ := strconv.ParseUint(httpadmin.String(data, "user_id"), 10, 64)
	if _, err := h.q.GetUserDetail(c.Context(), userID); errors.Is(err, sql.ErrNoRows) {
		return httpadmin.FieldError("user_id", "The selected user id is invalid.")
	} else if err != nil {
		return err
	}
	a := httpadmin.CurrentAdmin(c)
	res, err := h.q.CreateHealthAccessRequest(c.Context(), store.CreateHealthAccessRequestParams{
		AdminID: a.ID, UserID: userID, Reason: strings.TrimSpace(httpadmin.String(data, "reason")),
		Ticket: strings.TrimSpace(httpadmin.String(data, "ticket")), Now: httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "health_access.request", "user", userID, slog.Int64("request_id", id))
	return h.respond(c, uint64(id), true, "Access requested.") //nolint:gosec // auto-increment id
}

func (h *Handlers) respond(c fiber.Ctx, id uint64, created bool, msg string) error {
	r, err := h.q.GetHealthAccessRequest(c.Context(), id)
	if err != nil {
		return err
	}
	out := jsonx.Obj("request", accessJSON(r, httpadmin.DBTime(httpadmin.Now(c)).Time))
	if created {
		return httpadmin.Created(c, out, msg)
	}
	return httpadmin.OK(c, out, msg)
}

func (h *Handlers) load(c fiber.Ctx) (store.GetHealthAccessRequestRow, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.GetHealthAccessRequestRow{}, httpadmin.NotFound("Access request")
	}
	r, err := h.q.GetHealthAccessRequest(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return r, httpadmin.NotFound("Access request")
	}
	return r, err
}

func (h *Handlers) decide(c fiber.Ctx, approve bool) error {
	r, err := h.load(c)
	if err != nil {
		return err
	}
	a := httpadmin.CurrentAdmin(c)
	if approve && r.AdminID == a.ID {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeSelfApproval,
			"Another super admin must approve your own request.")
	}
	// B-N9-02b L2: the real clock; X-Test-Now must not stretch a grant or revive a stale request.
	now := time.Now()
	if r.Status != StatusPending || pendingExpired(r.CreatedAt, now) {
		return httpadmin.Fail(fiber.StatusConflict, CodeNotPending, "This request is no longer pending.")
	}
	if approve {
		if err := h.checkApproval(c, r, a, now); err != nil {
			return err
		}
	}
	status, expires, action := StatusRejected, sql.NullTime{}, "health_access.reject"
	if approve {
		status, action = StatusApproved, "health_access.approve"
		expires = httpadmin.DBTime(now.Add(httpadmin.HealthAccessTTLMinutes * time.Minute))
	}
	n, err := h.q.DecideHealthAccessRequest(c.Context(), store.DecideHealthAccessRequestParams{
		Status: status, DecidedBy: sql.NullInt64{Int64: int64(a.ID), Valid: true}, //nolint:gosec // ids fit int64
		Now: httpadmin.DBTime(now), ExpiresAt: expires, ID: r.ID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return httpadmin.Fail(fiber.StatusConflict, CodeNotPending, "This request is no longer pending.")
	}
	httpadmin.Audit(c, h.logger, action, "user", r.UserID, slog.Uint64("request_id", r.ID))
	if approve {
		// B-N9-02b: every approval raises an alert line (log-based alerting picks up alert=health_access_approved).
		h.logger.WarnContext(c.Context(), "health access approved", slog.String("alert", "health_access_approved"),
			slog.Uint64("request_id", r.ID), slog.Uint64("requester_id", r.AdminID), slog.Uint64("approver_id", a.ID))
	}
	return h.respond(c, r.ID, false, "Request "+status+".")
}

// checkApproval (B-N9-02b M1 + L5): the requester still holds the health capability and is active; the approver's
// account is at least approverMinAge old and was neither created nor changed by the requester within
// independenceWindow (from admin_audit_logs: admin.create / admin.update by the requester on the approver).
func (h *Handlers) checkApproval(c fiber.Ctx, r store.GetHealthAccessRequestRow, approver *store.Admin, now time.Time) error {
	requester, err := h.q.GetAdminByID(c.Context(), r.AdminID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	permitted := err == nil && requester.IsActive
	if permitted && requester.Role != httpadmin.RoleSuper {
		perms, perr := h.kit.PermissionsFor(c.Context(), requester.Role)
		if perr != nil {
			return perr
		}
		permitted = perms.Allows(httpadmin.CapHealth, httpadmin.LevelManage)
	}
	if !permitted {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeRequesterNotPermitted,
			"The requester's role no longer allows health-data access.")
	}
	notIndependent := httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeApproverNotIndependent,
		"An independent super admin must approve this request.")
	if !approver.CreatedAt.Valid || approver.CreatedAt.Time.After(now.Add(-approverMinAge)) {
		return notIndependent
	}
	n, err := h.q.CountAdminChangesBy(c.Context(), store.CountAdminChangesByParams{
		ActorID:  sql.NullInt64{Int64: int64(r.AdminID), Valid: true}, //nolint:gosec // ids fit int64
		TargetID: approver.ID,
		SinceAt:  httpadmin.DBTime(now.Add(-independenceWindow)),
	})
	if err != nil {
		return err
	}
	if n > 0 {
		return notIndependent
	}
	return nil
}

// Approve is POST /health-access/:id/approve (super, not the requester): approved for 30 minutes.
func (h *Handlers) Approve(c fiber.Ctx) error { return h.decide(c, true) }

// Reject is POST /health-access/:id/reject (super).
func (h *Handlers) Reject(c fiber.Ctx) error { return h.decide(c, false) }

// Revoke is POST /health-access/:id/revoke: the requester or a super admin ends a pending / live grant early.
func (h *Handlers) Revoke(c fiber.Ctx) error {
	r, err := h.load(c)
	if err != nil {
		return err
	}
	a := httpadmin.CurrentAdmin(c)
	if a.Role != httpadmin.RoleSuper && r.AdminID != a.ID {
		return httpadmin.NotFound("Access request") // never reveal another admin's request
	}
	n, err := h.q.RevokeHealthAccessRequest(c.Context(), store.RevokeHealthAccessRequestParams{
		Now: httpadmin.DBTime(httpadmin.Now(c)), ID: r.ID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return httpadmin.Fail(fiber.StatusConflict, CodeNotActive, "This request is no longer active.")
	}
	httpadmin.Audit(c, h.logger, "health_access.revoke", "user", r.UserID, slog.Uint64("request_id", r.ID))
	return h.respond(c, r.ID, false, "Access revoked.")
}

// ReadHealth is GET /health-access/users/:id: the user's health profile and counts, only with a live grant of the
// current admin for this user (403 health_access_required otherwise). Every read is audited.
func (h *Handlers) ReadHealth(c fiber.Ctx) error {
	userID, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("User")
	}
	grant, err := httpadmin.HealthGrant(c, h.q, userID)
	if err != nil {
		return err
	}
	if grant == nil {
		return httpadmin.HealthAccessRequired()
	}
	u, err := h.q.GetUserDetail(c.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("User")
	}
	if err != nil {
		return err
	}
	stats, err := h.q.UserStats(c.Context(), store.UserStatsParams{UserID: userID})
	if err != nil {
		return err
	}
	var profile any
	if u.ProfileID.Valid {
		profile = jsonx.Obj(
			"birthday", nullDate(u.Birthday),
			"last_period_start", nullDate(u.LastPeriodStart),
			"cycle_duration", nullInt(u.CycleDuration),
			"period_duration", nullInt(u.PeriodDuration),
			"pregnancy_intention", httpadmin.NullString(u.PregnancyIntention),
			"user_goal", httpadmin.NullString(u.UserGoal),
		)
	}
	// Fail closed (T-M10-33): no audit row, no health data.
	if err := httpadmin.AuditHealthRead(c, h.logger, userID, grant.ID); err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj(
		"user_id", userID,
		"expires_at", httpadmin.Time(grant.ExpiresAt),
		"profile", profile,
		"stats", jsonx.Obj("health_logs", stats.HealthLogs, "reminders", stats.Reminders),
	))
}

func nullDate(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date
}

func nullInt(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return n.Int16
}
