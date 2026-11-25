package messages

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/rules"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Smart messages v2 editor (bloom B-N9-05, D-86): /messages/rules*. A rule has a user level, a priority band, A/B
// variants, AND/OR condition groups + exceptions, placement, caps, cooldown, hours, window, audience share and a
// goal. Messages of the clinical kinds or the «worth follow-up» / «follow up soon» levels need the medical advisor's
// approval (the clinical capability, manage). Preview and conflict check are read-only.
//
// Message model v6 (T-M10-02, D-160): N caps, N weighted variants, datetime schedule with several hour windows,
// topics, goal events, rule keys and locked rules; spec-name aliases on input (type, slot, name; follow_up, urgent).
// Versioned approval (M-02, M-03): a content / condition / level / kind / slot / topic / variant-key edit of a medical
// rule bumps the version and leaves it pending while approved_version keeps serving; an ops edit (priority, audience,
// schedule, hours, caps, cooldown) bumps it and keeps the approval. Authority (SPEC §5.8): creating, editing or
// setting a safety / clinical / content rule, and toggling or deleting a safety rule, need the clinical capability.

// Error codes (the admin-web translates them).
const (
	CodeApprovalRequired  = "approval_required"   // 409: activating an unapproved medical message
	CodeNotPending        = "not_pending"         // 409: approve / reject only while pending_approval
	CodeApprovalNotNeeded = "approval_not_needed" // 422: this message is not a medical one
	CodeLocked            = "locked"              // 409: a locked rule cannot be switched off or deleted
)

const statsWindow = 7 * 24 * time.Hour

// rulesRoutes registers the v2 routes; call before the /messages/:id ones (static segment first).
func (h *Handlers) rulesRoutes(route func(method, path string, chain httpadmin.Chain), kit *httpadmin.Kit) {
	h.engineRoutes(route, kit)                                       // B-N9-06: static /messages/rules/engine|simulate
	route(fiber.MethodGet, "/messages/facts", kit.Admin(h.FactList)) // T-M10-03: the fact registry for the editor
	route(fiber.MethodGet, "/messages/rules", kit.Admin(h.RuleList))
	route(fiber.MethodPost, "/messages/rules", kit.Admin(httpadmin.Publishes(h.RuleStore, "message_rules")))
	route(fiber.MethodPost, "/messages/rules/conflicts", kit.Admin(h.RuleConflicts))
	route(fiber.MethodPost, "/messages/rules/preview", kit.Admin(h.RulePreview))
	route(fiber.MethodGet, "/messages/rules/:id", kit.Admin(h.RuleShow))
	route(fiber.MethodPut, "/messages/rules/:id", kit.Admin(httpadmin.Publishes(h.RuleUpdate, "message_rules")))
	route(fiber.MethodDelete, "/messages/rules/:id", kit.Admin(httpadmin.Publishes(h.RuleDestroy, "message_rules")))
	route(fiber.MethodPost, "/messages/rules/:id/submit", kit.Admin(httpadmin.Publishes(h.RuleSubmit, "message_rules")))
	route(fiber.MethodPost, "/messages/rules/:id/approve", kit.Admin(httpadmin.Publishes(h.RuleApprove, "message_rules")))
	route(fiber.MethodPost, "/messages/rules/:id/reject", kit.Admin(httpadmin.Publishes(h.RuleReject, "message_rules")))
	route(fiber.MethodPost, "/messages/rules/:id/toggle", kit.Admin(httpadmin.Publishes(h.RuleToggle, "message_rules")))
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func nullInt16(p *int) sql.NullInt16 {
	if p == nil {
		return sql.NullInt16{}
	}
	return sql.NullInt16{Int16: int16(*p), Valid: true} //nolint:gosec // validated ranges
}

func nullDate(s string) civildate.NullDate {
	if s == "" {
		return civildate.NullDate{}
	}
	d, err := civildate.Parse(s)
	return civildate.NullDate{Date: d, Valid: err == nil}
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func dateOut(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date.String()
}

func nullInt64Out(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func (h *Handlers) rawJSON(s string) any {
	var v any
	if json.Unmarshal([]byte(s), &v) != nil {
		return nil
	}
	return v
}

func (h *Handlers) ruleJSON(m *mstore.MessageRule) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", m.ID, "rule_key", httpadmin.NullString(m.RuleKey), "title", m.Title, "level", m.Level, "kind", m.Kind,
		"priority", m.Priority, "placement", m.Placement, "content", h.rawJSON(m.Content),
		"variants_enabled", m.VariantsEnabled, "ab_split", m.AbSplit, "variants", jsonOut(m.Variants),
		"conditions", h.rawJSON(m.Conditions), "exceptions", h.rawJSON(m.Exceptions), "topics", jsonOut(m.Topics),
		"cap_count", nullInt16Out(m.CapCount), "cap_period", httpadmin.NullString(m.CapPeriod), "caps", jsonOut(m.Caps),
		"cooldown_days", m.CooldownDays, "hours_from", nullInt16Out(m.HoursFrom), "hours_to", nullInt16Out(m.HoursTo),
		"hour_windows", jsonOut(m.HourWindows), "starts_on", dateOut(m.StartsOn), "ends_on", dateOut(m.EndsOn),
		"starts_at", scheduleOut(m.StartsAt), "ends_at", scheduleOut(m.EndsAt), "audience_percent", m.AudiencePercent,
		"goal", httpadmin.NullString(m.Goal), "goal_event", httpadmin.NullString(m.GoalEvent), "locked", m.Locked,
		"status", m.Status, "is_active", m.IsActive, "version", m.Version,
		"approved_version", nullInt32Out(m.ApprovedVersion), "served_version", servedVersion(m),
		"needs_medical_approval", rules.NeedsMedicalApproval(m.Level, m.Kind),
		"approved_by", nullInt64Out(m.ApprovedBy), "approved_at", httpadmin.Time(m.ApprovedAt),
		"created_at", httpadmin.Time(m.CreatedAt), "updated_at", httpadmin.Time(m.UpdatedAt),
		// Spec names (M-01), next to the canonical keys.
		"type", m.Kind, "slot", m.Placement, "name", m.Title, "level_name", rules.SpecLevel(m.Level),
	)
}

func jsonOut(j db.NullRawJSON) any {
	if !j.Valid {
		return nil
	}
	var v any
	if json.Unmarshal(j.V, &v) != nil {
		return nil
	}
	return v
}

func scheduleOut(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return t.Time.Format(rules.ScheduleLayout)
}

func nullInt32Out(n sql.NullInt32) any {
	if !n.Valid {
		return nil
	}
	return n.Int32
}

// servedVersion is the version users get: the head when it needs no approval or is approved, else the approved
// version (M-03), else none (null).
func servedVersion(m *mstore.MessageRule) any {
	if !rules.NeedsMedicalApproval(m.Level, m.Kind) || m.Status == rules.StatusApproved {
		return m.Version
	}
	return nullInt32Out(m.ApprovedVersion)
}

func nullInt16Out(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return n.Int16
}

func (h *Handlers) findRule(c fiber.Ctx) (*mstore.MessageRule, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return nil, httpadmin.NotFound("Message rule")
	}
	m, err := h.mq.GetMessageRule(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpadmin.NotFound("Message rule")
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// parseInput reads and validates the editor payload (422 with the field messages).
func parseInput(c fiber.Ctx) (*rules.Input, error) {
	var in rules.Input
	if err := json.Unmarshal(c.Body(), &in); err != nil {
		return nil, httpadmin.FieldError("title", "The request body is not valid JSON.")
	}
	in.Normalize()
	if errs, code := in.Check(); len(errs) > 0 {
		ve := httpx.NewValidationError()
		for _, f := range rules.Fields {
			if m, ok := errs[f]; ok {
				ve.Add(f, m)
			}
		}
		if code == "" { // D-161: unknown_fact / bad_value name a fact-registry failure
			return nil, httpadmin.Invalid(ve)
		}
		errs, _ := ve.Body().Get("errors")
		return nil, httpadmin.Fail(fiber.StatusUnprocessableEntity, code, ve.Summary(), "errors", errs)
	}
	return &in, nil
}

// needsClinical reports whether authoring a rule of this kind needs the clinical capability (SPEC §5.3/§5.8).
func needsClinical(kind string) bool { return slices.Contains(rules.ApprovalKinds, kind) }

// checkRuleKey is the 422 of a rule key another rule already holds.
func (h *Handlers) checkRuleKey(ctx context.Context, key string, self uint64) error {
	if key == "" {
		return nil
	}
	id, err := h.mq.GetMessageRuleIDByKey(ctx, nullStr(key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if id != self {
		return httpadmin.FieldError("rule_key", "The key is already taken.")
	}
	return nil
}

func jsonCol[T any](v []T) db.NullRawJSON {
	if v == nil {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: json.RawMessage(mustJSON(v)), Valid: true}
}

func scheduleCol(s string) sql.NullTime {
	t, ok := rules.ParseSchedule(s)
	if s == "" || !ok {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// v6Cols are the message model v6 columns of a payload.
type v6Cols struct {
	ruleKey, goalEvent                  sql.NullString
	topics, caps, variants, hourWindows db.NullRawJSON
	startsAt, endsAt                    sql.NullTime
}

func colsOf(in *rules.Input) v6Cols {
	return v6Cols{
		ruleKey: nullStr(in.RuleKey), goalEvent: nullStr(in.GoalEvent), topics: jsonCol(in.Topics), caps: jsonCol(in.Caps),
		variants: jsonCol(in.Variants), hourWindows: jsonCol(in.HourWindows),
		startsAt: scheduleCol(in.StartsAt), endsAt: scheduleCol(in.EndsAt),
	}
}

func (h *Handlers) snapshot(ctx context.Context, m *mstore.MessageRule, adminID uint64, now time.Time) error {
	m2 := h.ruleJSON(m)
	b, err := jsonx.Marshal(m2, 0)
	if err != nil {
		return err
	}
	return h.mq.InsertMessageRuleVersion(ctx, mstore.InsertMessageRuleVersionParams{
		RuleID: m.ID, Version: m.Version, Snapshot: string(b),
		ChangedBy: sql.NullInt64{Int64: int64(adminID), Valid: adminID > 0}, //nolint:gosec // admin ids are small
		Now:       httpadmin.DBTime(now),
	})
}

func adminID(c fiber.Ctx) uint64 {
	if a := httpadmin.CurrentAdmin(c); a != nil {
		return a.ID
	}
	return 0
}

func ab(in *rules.Input) uint8 {
	if in.ABSplit == nil {
		return 50
	}
	return uint8(*in.ABSplit) //nolint:gosec // validated 1–99
}

func aud(in *rules.Input) uint8 {
	if in.AudiencePercent == nil {
		return 100
	}
	return uint8(*in.AudiencePercent) //nolint:gosec // validated 1–100
}

func capPeriod(in *rules.Input) sql.NullString {
	if in.CapCount == nil {
		return sql.NullString{}
	}
	return nullStr(in.CapPeriod)
}

// RuleStore is POST /messages/rules.
func (h *Handlers) RuleStore(c fiber.Ctx) error {
	in, err := parseInput(c)
	if err != nil {
		return err
	}
	if needsClinical(in.Kind) {
		if err := h.requireMedical(c); err != nil {
			return err
		}
	}
	if err := h.checkRuleKey(c.Context(), in.RuleKey, 0); err != nil {
		return err
	}
	v6 := colsOf(in)
	now := httpadmin.Now(c)
	res, err := h.mq.InsertMessageRule(c.Context(), mstore.InsertMessageRuleParams{
		Title: in.Title, Level: in.Level, Kind: in.Kind, Priority: uint16(in.Priority), //nolint:gosec // validated band
		Placement: in.Placement, Content: mustJSON(in.Content), VariantsEnabled: in.VariantsEnabled, AbSplit: ab(in),
		Conditions: mustJSON(orEmpty(in.Conditions)), Exceptions: mustJSON(orEmptyC(in.Exceptions)),
		CapCount: nullInt16(in.CapCount), CapPeriod: capPeriod(in), CooldownDays: uint16(in.CooldownDays), //nolint:gosec // validated
		HoursFrom: nullInt16(in.HoursFrom), HoursTo: nullInt16(in.HoursTo),
		StartsOn: nullDate(in.StartsOn), EndsOn: nullDate(in.EndsOn), AudiencePercent: aud(in), Goal: nullStr(in.Goal),
		RuleKey: v6.ruleKey, Topics: v6.topics, Caps: v6.caps, Variants: v6.variants, StartsAt: v6.startsAt,
		EndsAt: v6.endsAt, HourWindows: v6.hourWindows, GoalEvent: v6.goalEvent,
		CreatedBy: sql.NullInt64{Int64: int64(adminID(c)), Valid: adminID(c) > 0}, //nolint:gosec // small
		Now:       httpadmin.DBTime(now),
	})
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	m, err := h.mq.GetMessageRule(c.Context(), uint64(id)) //nolint:gosec // auto-increment id
	if err != nil {
		return err
	}
	if err := h.snapshot(c.Context(), &m, adminID(c), now); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "message_rule.create", "message_rule", m.ID, slog.String("kind", m.Kind))
	return httpadmin.Created(c, jsonx.Obj("rule", h.ruleJSON(&m)), "Message rule created.")
}

func orEmpty(g []rules.Group) []rules.Group {
	if g == nil {
		return []rules.Group{}
	}
	return g
}

func orEmptyC(c []rules.Condition) []rules.Condition {
	if c == nil {
		return []rules.Condition{}
	}
	return c
}

// approvalAfter is the approval state of a rule after an edit (M-02, M-03): a non-medical rule goes back to draft as
// before; a medical one whose served fields changed waits for approval (pending) with the last approved version still
// serving; an ops-only edit keeps the state, and an approved head stays approved at the new version.
type approvalAfter struct {
	status          string
	approvedBy      sql.NullInt64
	approvedAt      sql.NullTime
	approvedVersion sql.NullInt32
	withdrawn       bool
}

func nextApproval(cur *mstore.MessageRule, in *rules.Input) approvalAfter {
	old, updated := rules.FromRow(cur), in.Rule(cur.ID)
	// A head approved before T-M10-02 has no approved_version: its own version is the approved one.
	approved := cur.ApprovedVersion
	if cur.Status == rules.StatusApproved {
		approved = sql.NullInt32{Int32: int32(cur.Version), Valid: true} //nolint:gosec // versions are small
	}
	if !rules.NeedsMedicalApproval(in.Level, in.Kind) {
		return approvalAfter{status: rules.StatusDraft, approvedVersion: approved}
	}
	if !rules.NeedsMedicalApproval(cur.Level, cur.Kind) || rules.VoidsApproval(&old, &updated) {
		return approvalAfter{status: rules.StatusPending, approvedVersion: approved, withdrawn: cur.Status == rules.StatusApproved}
	}
	if cur.Status == rules.StatusApproved {
		return approvalAfter{status: rules.StatusApproved, approvedBy: cur.ApprovedBy, approvedAt: cur.ApprovedAt,
			approvedVersion: sql.NullInt32{Int32: int32(cur.Version) + 1, Valid: true}} //nolint:gosec // versions are small
	}
	return approvalAfter{status: cur.Status, approvedVersion: cur.ApprovedVersion}
}

// ensureSnapshot stores the snapshot of an approved head that has none (a seeded or pre-v6 row), so its approved
// version can keep serving while the next version waits for approval.
func (h *Handlers) ensureSnapshot(ctx context.Context, m *mstore.MessageRule, now time.Time) error {
	if m.Status != rules.StatusApproved {
		return nil
	}
	_, err := h.mq.GetMessageRuleVersion(ctx, mstore.GetMessageRuleVersionParams{RuleID: m.ID, Version: m.Version})
	if errors.Is(err, sql.ErrNoRows) {
		return h.snapshot(ctx, m, 0, now)
	}
	return err
}

// RuleUpdate is PUT /messages/rules/:id. The whole definition is replaced; the version grows and a snapshot is
// stored; the approval follows nextApproval (M-02, M-03).
func (h *Handlers) RuleUpdate(c fiber.Ctx) error {
	cur, err := h.findRule(c)
	if err != nil {
		return err
	}
	in, err := parseInput(c)
	if err != nil {
		return err
	}
	if needsClinical(cur.Kind) || needsClinical(in.Kind) {
		if err := h.requireMedical(c); err != nil {
			return err
		}
	}
	if err := h.checkRuleKey(c.Context(), in.RuleKey, cur.ID); err != nil {
		return err
	}
	now := httpadmin.Now(c)
	if err := h.ensureSnapshot(c.Context(), cur, now); err != nil {
		return err
	}
	ap := nextApproval(cur, in)
	v6 := colsOf(in)
	if err := h.mq.UpdateMessageRule(c.Context(), mstore.UpdateMessageRuleParams{
		Title: in.Title, Level: in.Level, Kind: in.Kind, Priority: uint16(in.Priority), //nolint:gosec // validated band
		Placement: in.Placement, Content: mustJSON(in.Content), VariantsEnabled: in.VariantsEnabled, AbSplit: ab(in),
		Conditions: mustJSON(orEmpty(in.Conditions)), Exceptions: mustJSON(orEmptyC(in.Exceptions)),
		CapCount: nullInt16(in.CapCount), CapPeriod: capPeriod(in), CooldownDays: uint16(in.CooldownDays), //nolint:gosec // validated
		HoursFrom: nullInt16(in.HoursFrom), HoursTo: nullInt16(in.HoursTo),
		StartsOn: nullDate(in.StartsOn), EndsOn: nullDate(in.EndsOn), AudiencePercent: aud(in), Goal: nullStr(in.Goal),
		RuleKey: v6.ruleKey, Topics: v6.topics, Caps: v6.caps, Variants: v6.variants, StartsAt: v6.startsAt,
		EndsAt: v6.endsAt, HourWindows: v6.hourWindows, GoalEvent: v6.goalEvent,
		Status: ap.status, ApprovedBy: ap.approvedBy, ApprovedAt: ap.approvedAt, ApprovedVersion: ap.approvedVersion,
		Now: httpadmin.DBTime(now), ID: cur.ID,
	}); err != nil {
		return err
	}
	m, err := h.mq.GetMessageRule(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	if err := h.snapshot(c.Context(), &m, adminID(c), now); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "message_rule.update", "message_rule", m.ID,
		slog.Int("version", int(m.Version)), slog.Bool("approval_withdrawn", ap.withdrawn), slog.String("status", m.Status))
	return httpadmin.OK(c, jsonx.Obj("rule", h.ruleJSON(&m)), "Message rule updated.")
}

// RuleDestroy is DELETE /messages/rules/:id. A safety rule needs the clinical capability; a locked rule is 409.
func (h *Handlers) RuleDestroy(c fiber.Ctx) error {
	m, err := h.findRule(c)
	if err != nil {
		return err
	}
	if m.Kind == rules.KindSafety {
		if err := h.requireMedical(c); err != nil {
			return err
		}
	}
	if m.Locked {
		return httpadmin.Fail(fiber.StatusConflict, CodeLocked, "A locked message cannot be deleted.")
	}
	if err := h.mq.DeleteMessageRule(c.Context(), m.ID); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "message_rule.delete", "message_rule", m.ID)
	return httpadmin.OK(c, nil, "Message rule deleted.")
}

// RuleSubmit is POST /messages/rules/:id/submit: draft → pending_approval (medical messages only).
func (h *Handlers) RuleSubmit(c fiber.Ctx) error {
	m, err := h.findRule(c)
	if err != nil {
		return err
	}
	if !rules.NeedsMedicalApproval(m.Level, m.Kind) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeApprovalNotNeeded, "This message does not need medical approval.")
	}
	if m.Status != rules.StatusDraft {
		return httpadmin.Fail(fiber.StatusConflict, CodeNotPending, "Only a draft can be submitted.")
	}
	return h.setStatus(c, m, rules.StatusPending, 0, "message_rule.submit", "Submitted for medical approval.")
}

func (h *Handlers) requireMedical(c fiber.Ctx) error {
	if !httpadmin.Can(c, httpadmin.CapClinical, httpadmin.LevelManage) {
		return httpadmin.ForbiddenPermission(httpadmin.CapClinical)
	}
	return nil
}

// RuleApprove is POST /messages/rules/:id/approve — the clinical capability (manage) only. One approval is enough and
// the author may give it (D-215): a draft is approved directly, without the submit step a non-clinical author uses.
func (h *Handlers) RuleApprove(c fiber.Ctx) error {
	if err := h.requireMedical(c); err != nil {
		return err
	}
	m, err := h.findRule(c)
	if err != nil {
		return err
	}
	if !rules.NeedsMedicalApproval(m.Level, m.Kind) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeApprovalNotNeeded, "This message does not need medical approval.")
	}
	if m.Status != rules.StatusPending && m.Status != rules.StatusDraft {
		return httpadmin.Fail(fiber.StatusConflict, CodeNotPending, "The message is not waiting for approval.")
	}
	return h.setStatus(c, m, rules.StatusApproved, adminID(c), "message_rule.approve", "Message approved.")
}

// RuleReject is POST /messages/rules/:id/reject: back to draft.
func (h *Handlers) RuleReject(c fiber.Ctx) error {
	if err := h.requireMedical(c); err != nil {
		return err
	}
	m, err := h.findRule(c)
	if err != nil {
		return err
	}
	if m.Status != rules.StatusPending {
		return httpadmin.Fail(fiber.StatusConflict, CodeNotPending, "The message is not waiting for approval.")
	}
	return h.setStatus(c, m, rules.StatusDraft, 0, "message_rule.reject", "Message returned to draft.")
}

func (h *Handlers) setStatus(c fiber.Ctx, m *mstore.MessageRule, status string, approver uint64, action, msg string) error {
	now := httpadmin.DBTime(httpadmin.Now(c))
	p := mstore.SetMessageRuleStatusParams{Status: status, Now: now, ID: m.ID}
	if status == rules.StatusApproved {
		p.ApprovedBy = sql.NullInt64{Int64: int64(approver), Valid: true} //nolint:gosec // small
		p.ApprovedAt = now
		p.ApprovedVersion = sql.NullInt32{Int32: int32(m.Version), Valid: true} //nolint:gosec // versions are small
	}
	if err := h.mq.SetMessageRuleStatus(c.Context(), p); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, action, "message_rule", m.ID)
	return h.respondRule(c, m.ID, msg)
}

// RuleToggle is POST /messages/rules/:id/toggle: publish / unpublish. A medical message must be approved first (or
// have an approved version that keeps serving); a safety rule needs the clinical capability; a locked rule cannot be
// switched off (409).
func (h *Handlers) RuleToggle(c fiber.Ctx) error {
	m, err := h.findRule(c)
	if err != nil {
		return err
	}
	if m.Kind == rules.KindSafety {
		if err := h.requireMedical(c); err != nil {
			return err
		}
	}
	if m.IsActive && m.Locked {
		return httpadmin.Fail(fiber.StatusConflict, CodeLocked, "A locked message cannot be switched off.")
	}
	if !m.IsActive && rules.NeedsMedicalApproval(m.Level, m.Kind) && m.Status != rules.StatusApproved &&
		(!m.ApprovedVersion.Valid || m.ApprovedVersion.Int32 <= 0) {
		return httpadmin.Fail(fiber.StatusConflict, CodeApprovalRequired, "A medical message must be approved before it is published.")
	}
	if err := h.mq.SetMessageRuleActive(c.Context(), mstore.SetMessageRuleActiveParams{
		IsActive: !m.IsActive, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: m.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "message_rule.toggle", "message_rule", m.ID, slog.Bool("is_active", !m.IsActive))
	return h.respondRule(c, m.ID, "Message status changed.")
}

func (h *Handlers) respondRule(c fiber.Ctx, id uint64, msg string) error {
	m, err := h.mq.GetMessageRule(c.Context(), id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("rule", h.ruleJSON(&m)), msg)
}

type stat struct{ impressions, clicks, dismissals, goals int64 }

func (s stat) json() *jsonx.OrderedMap {
	ctr := 0.0
	if s.impressions > 0 {
		ctr = float64(s.clicks) / float64(s.impressions)
	}
	return jsonx.Obj("impressions", s.impressions, "clicks", s.clicks, "dismissals", s.dismissals, "goals", s.goals, "ctr", ctr)
}

func (s *stat) add(event string, n int64) {
	switch event {
	case "impression":
		s.impressions += n
	case "click":
		s.clicks += n
	case "dismiss":
		s.dismissals += n
	case "goal":
		s.goals += n
	}
}

// versionKey is one (version, variant) of a rule.
type versionKey struct {
	version uint64
	variant string
}

// versionStats are the 7-day counts per rule, version and variant.
type versionStats map[uint64]map[versionKey]*stat

func (v versionStats) add(rule, version uint64, variant, event string, n int64) {
	m := v[rule]
	if m == nil {
		m = map[versionKey]*stat{}
		v[rule] = m
	}
	k := versionKey{version, variant}
	if m[k] == nil {
		m[k] = &stat{}
	}
	m[k].add(event, n)
}

// json is [{version, variant, impressions, clicks, dismissals, goals, ctr}] newest version first (version 0 = events
// logged before T-M10-06 recorded versions).
func (v versionStats) json(rule uint64) []*jsonx.OrderedMap {
	keys := make([]versionKey, 0, len(v[rule]))
	for k := range v[rule] {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b versionKey) int {
		if a.version != b.version {
			return cmp.Compare(b.version, a.version)
		}
		return cmp.Compare(a.variant, b.variant)
	})
	out := make([]*jsonx.OrderedMap, 0, len(keys))
	for _, k := range keys {
		st := v[rule][k].json()
		o := jsonx.Obj("version", k.version, "variant", k.variant)
		for _, key := range st.Keys() {
			val, _ := st.Get(key)
			o.Set(key, val)
		}
		out = append(out, o)
	}
	return out
}

// ruleStats are the last 7 Tehran days of events per rule (and per version and variant): the daily rollup for the 6
// days before today (messages.rollup, M-13) plus today's raw rows, instead of a 7-day scan of message_events
// (T-M10-06).
func (h *Handlers) ruleStats(c fiber.Ctx) (map[uint64]*stat, versionStats, error) {
	today := civildate.InTehran(httpadmin.Now(c))
	daily, err := h.mq.MessageRuleDailyStats(c.Context(), mstore.MessageRuleDailyStatsParams{SinceDay: today.AddDays(-6), UntilDay: today})
	if err != nil {
		return nil, nil, err
	}
	raw, err := h.mq.MessageRuleRawStats(c.Context(), sql.NullTime{Time: today.TehranMidnight(), Valid: true})
	if err != nil {
		return nil, nil, err
	}
	stats, byVersion := map[uint64]*stat{}, versionStats{}
	add := func(rule, version uint64, variant, event string, n int64) {
		s := stats[rule]
		if s == nil {
			s = &stat{}
			stats[rule] = s
		}
		s.add(event, n)
		byVersion.add(rule, version, variant, event, n)
	}
	for _, r := range daily {
		add(r.RuleID, uint64(r.Version), r.Variant, r.Event, r.Total)
	}
	for _, r := range raw {
		add(r.RuleID, uint64(max(r.Version, 0)), r.Variant, r.Event, r.Total) //nolint:gosec // non-negative
	}
	return stats, byVersion, nil
}

// RuleList is GET /messages/rules?placement=&kind=&status=&page= with 7-day impressions and CTR per rule and per
// version and variant (stats_by_version).
func (h *Handlers) RuleList(c fiber.Ctx) error {
	placement, kind, status := c.Query("placement"), c.Query("kind"), c.Query("status")
	total, err := h.mq.CountMessageRules(c.Context(), mstore.CountMessageRulesParams{Placement: placement, Kind: kind, Status: status})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.mq.ListMessageRules(c.Context(), mstore.ListMessageRulesParams{
		Placement: placement, Kind: kind, Status: status, Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // bounded
	})
	if err != nil {
		return err
	}
	stats, byVersion, err := h.ruleStats(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		o := h.ruleJSON(&rows[i])
		s := stat{}
		if st := stats[rows[i].ID]; st != nil {
			s = *st
		}
		o.Set("stats_7d", s.json())
		o.Set("stats_by_version", byVersion.json(rows[i].ID))
		items = append(items, o)
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("enums", jsonx.Obj(
		"placements", jsonx.List(rules.Placements), "levels", jsonx.List(rules.Levels), "kinds", jsonx.List(rules.Kinds),
		"statuses", jsonx.List([]string{rules.StatusDraft, rules.StatusPending, rules.StatusApproved}),
		"variables", jsonx.List(rules.Variables), "ops", jsonx.List(rules.Ops), "cap_periods", jsonx.List(rules.CapPeriods),
		"bands", bands(), "goal_events", jsonx.List(rules.GoalEvents), "topics", jsonx.List(rules.Topics),
		"level_kinds", matrix(rules.Levels, rules.LevelKinds), "slot_kinds", matrix(rules.Placements, rules.SlotKinds),
		"approval_kinds", jsonx.List(rules.ApprovalKinds), "max_variants", rules.MaxVariants,
	))
	return httpadmin.OK(c, page)
}

func matrix(keys []string, m map[string][]string) *jsonx.OrderedMap {
	out := jsonx.Obj()
	for _, k := range keys {
		out.Set(k, jsonx.List(m[k]))
	}
	return out
}

func bands() *jsonx.OrderedMap {
	out := jsonx.Obj()
	for _, k := range rules.Kinds {
		lo, hi := rules.Band(k)
		out.Set(k, jsonx.Obj("min", lo, "max", hi))
	}
	return out
}

func conflictJSON(cs []rules.Conflict, titles map[uint64]string) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(cs))
	for i := range cs {
		out = append(out, ConflictV2JSON(&cs[i], titles[cs[i].RuleID]))
	}
	return out
}

func (h *Handlers) conflicts(ctx context.Context, r *rules.Rule) ([]*jsonx.OrderedMap, error) {
	rows, err := h.mq.ListActiveMessageRules(ctx, mstore.ListActiveMessageRulesParams{Placement: r.Placement})
	if err != nil {
		return nil, err
	}
	others := make([]rules.Rule, 0, len(rows))
	titles := map[uint64]string{}
	for i := range rows {
		others = append(others, rules.FromRow(&rows[i]))
		titles[rows[i].ID] = rows[i].Title
	}
	return conflictJSON(rules.FindConflicts(r, others), titles), nil
}

// RuleShow is GET /messages/rules/:id: the rule, its version history, per-variant stats and conflicts.
func (h *Handlers) RuleShow(c fiber.Ctx) error {
	m, err := h.findRule(c)
	if err != nil {
		return err
	}
	vers, err := h.mq.ListMessageRuleVersions(c.Context(), m.ID)
	if err != nil {
		return err
	}
	versions := make([]*jsonx.OrderedMap, 0, len(vers))
	for _, v := range vers {
		versions = append(versions, jsonx.Obj("version", v.Version, "changed_by", nullInt64Out(v.ChangedBy),
			"created_at", httpadmin.Time(v.CreatedAt)))
	}
	vs, err := h.mq.MessageRuleVariantStats(c.Context(), mstore.MessageRuleVariantStatsParams{
		RuleID: m.ID, CreatedAt: httpadmin.DBTime(httpadmin.Now(c).Add(-statsWindow)),
	})
	if err != nil {
		return err
	}
	byVariant := map[string]*stat{"a": {}, "b": {}}
	for _, r := range vs {
		if s := byVariant[r.Variant]; s != nil {
			s.add(r.Event, r.Total)
		}
	}
	r := rules.FromRow(m)
	cf, err := h.conflicts(c.Context(), &r)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("rule", h.ruleJSON(m), "versions", versions,
		"variant_stats_7d", jsonx.Obj("a", byVariant["a"].json(), "b", byVariant["b"].json()), "conflicts", cf))
}

// RuleConflicts is POST /messages/rules/conflicts {…editor payload, id?}: the conflict check of a draft.
func (h *Handlers) RuleConflicts(c fiber.Ctx) error {
	r, _, err := h.draftRule(c)
	if err != nil {
		return err
	}
	cf, err := h.conflicts(c.Context(), r)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("conflicts", cf))
}

// draftRule builds an engine rule from a posted editor payload ({rule: {...}, id?} — the payload itself is the rule
// when no wrapper is used). Validation errors are not raised here: a draft may be incomplete while previewing.
func (h *Handlers) draftRule(c fiber.Ctx) (*rules.Rule, *previewBody, error) {
	var b previewBody
	if err := json.Unmarshal(c.Body(), &b); err != nil {
		return nil, nil, httpadmin.FieldError("rule", "The request body is not valid JSON.")
	}
	if b.RuleID != 0 && b.Rule == nil {
		m, err := h.mq.GetMessageRule(c.Context(), b.RuleID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, httpadmin.NotFound("Message rule")
		}
		if err != nil {
			return nil, nil, err
		}
		r := rules.FromRow(&m)
		return &r, &b, nil
	}
	if b.Rule == nil {
		return nil, nil, httpadmin.FieldError("rule", "A rule or a rule_id is required.")
	}
	b.Rule.Normalize()
	r := b.Rule.Rule(b.RuleID)
	r.IsActive, r.Status = true, rules.StatusApproved // a preview shows what the message would do once published
	return &r, &b, nil
}

type previewBody struct {
	RuleID uint64         `json:"rule_id"`
	Rule   *rules.Input   `json:"rule"`
	UserID uint64         `json:"user_id"`
	Facts  map[string]any `json:"facts"`
}

// WithEngine gives the rule preview the user-facing v6 facts (T-M9-38, D-130: the same facts /messages/selected
// reads under CYCLE_ENGINE=shadow|v6); nil keeps v1.
func (h *Handlers) WithEngine(states state.UserStates) *Handlers {
	h.user = h.user.WithEngine(states)
	return h
}

// RulePreview is POST /messages/rules/preview {rule_id | rule, user_id?, facts?}: what that user (or those facts)
// would see — the variant, the rendered text and why a rule does not show. Fact values are never returned; reading a
// real user's facts is audited.
func (h *Handlers) RulePreview(c fiber.Ctx) error {
	r, b, err := h.draftRule(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	now := httpadmin.Now(c)
	facts := rules.Facts{}
	var hist rules.History
	if b.UserID > 0 {
		if _, err := h.mq.GetMessageUserName(ctx, b.UserID); errors.Is(err, sql.ErrNoRows) {
			return httpadmin.NotFound("User")
		} else if err != nil {
			return err
		}
		today, err := h.user.UserToday(ctx, b.UserID, now) // T-M9-38b: her local day under v6, not Tehran's
		if err != nil {
			return err
		}
		if facts, err = messages.BuildFacts(ctx, h.user, b.UserID, today); err != nil {
			return err
		}
		hist = rules.DBHistory{Ctx: ctx, Q: h.mq, UserID: b.UserID}
		httpadmin.Audit(c, h.logger, "message_rule.preview", "user", b.UserID, slog.Uint64("rule_id", r.ID))
	}
	for k, v := range b.Facts {
		facts[k] = v
	}
	// Served is judged as if published; the other gates run for real.
	r.IsActive, r.Status = true, rules.StatusApproved
	eligible, err := rules.Eligible(r, facts, b.UserID, now, hist)
	if err != nil {
		return err
	}
	variant := r.Assign(b.UserID)
	v, renderable := rules.RenderVariant(r.VariantContent(variant), facts)
	reasons := []string{}
	if !rules.MatchRule(r, facts) {
		reasons = append(reasons, "conditions")
	}
	if !rules.InWindow(r, now) {
		reasons = append(reasons, "window")
	}
	if b.UserID > 0 && !r.InAudience(b.UserID) {
		reasons = append(reasons, "audience")
	}
	if !eligible && len(reasons) == 0 {
		reasons = append(reasons, "cap_or_cooldown")
	}
	if !renderable {
		reasons = append(reasons, "missing_variable")
	}
	return httpadmin.OK(c, jsonx.Obj("eligible", eligible && renderable, "reasons", reasons, "variant", variant,
		"title", v.Title, "body", v.Body, "cta_label", v.CTALabel, "level", r.Level))
}
