package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

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
// approval (the clinical capability, manage) and ANY edit withdraws it. Preview and conflict check are read-only.

// Error codes (the admin-web translates them).
const (
	CodeApprovalRequired  = "approval_required"   // 409: activating an unapproved medical message
	CodeNotPending        = "not_pending"         // 409: approve / reject only while pending_approval
	CodeApprovalNotNeeded = "approval_not_needed" // 422: this message is not a medical one
)

const statsWindow = 7 * 24 * time.Hour

// rulesRoutes registers the v2 routes; call before the /messages/:id ones (static segment first).
func (h *Handlers) rulesRoutes(route func(method, path string, chain httpadmin.Chain), kit *httpadmin.Kit) {
	h.engineRoutes(route, kit) // B-N9-06: static /messages/rules/engine|simulate
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
		"id", m.ID, "title", m.Title, "level", m.Level, "kind", m.Kind, "priority", m.Priority,
		"placement", m.Placement, "content", h.rawJSON(m.Content), "variants_enabled", m.VariantsEnabled,
		"ab_split", m.AbSplit, "conditions", h.rawJSON(m.Conditions), "exceptions", h.rawJSON(m.Exceptions),
		"cap_count", nullInt16Out(m.CapCount), "cap_period", httpadmin.NullString(m.CapPeriod),
		"cooldown_days", m.CooldownDays, "hours_from", nullInt16Out(m.HoursFrom), "hours_to", nullInt16Out(m.HoursTo),
		"starts_on", dateOut(m.StartsOn), "ends_on", dateOut(m.EndsOn), "audience_percent", m.AudiencePercent,
		"goal", httpadmin.NullString(m.Goal), "status", m.Status, "is_active", m.IsActive, "version", m.Version,
		"needs_medical_approval", rules.NeedsMedicalApproval(m.Level, m.Kind),
		"approved_by", nullInt64Out(m.ApprovedBy), "approved_at", httpadmin.Time(m.ApprovedAt),
		"created_at", httpadmin.Time(m.CreatedAt), "updated_at", httpadmin.Time(m.UpdatedAt),
	)
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
	if errs := in.Validate(); len(errs) > 0 {
		ve := httpx.NewValidationError()
		for _, f := range []string{"title", "level", "kind", "priority", "placement", "content.a", "content.b", "content.seen",
			"content.confidence", "content.contact", "ab_split", "cap_count", "cap_period", "cooldown_days", "hours_from",
			"hours_to", "starts_on", "ends_on", "audience_percent", "goal", "conditions", "exceptions"} {
			if m, ok := errs[f]; ok {
				ve.Add(f, m)
			}
		}
		return nil, httpadmin.Invalid(ve)
	}
	return &in, nil
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
	now := httpadmin.Now(c)
	res, err := h.mq.InsertMessageRule(c.Context(), mstore.InsertMessageRuleParams{
		Title: in.Title, Level: in.Level, Kind: in.Kind, Priority: uint16(in.Priority), //nolint:gosec // validated band
		Placement: in.Placement, Content: mustJSON(in.Content), VariantsEnabled: in.VariantsEnabled, AbSplit: ab(in),
		Conditions: mustJSON(orEmpty(in.Conditions)), Exceptions: mustJSON(orEmptyC(in.Exceptions)),
		CapCount: nullInt16(in.CapCount), CapPeriod: capPeriod(in), CooldownDays: uint16(in.CooldownDays), //nolint:gosec // validated
		HoursFrom: nullInt16(in.HoursFrom), HoursTo: nullInt16(in.HoursTo),
		StartsOn: nullDate(in.StartsOn), EndsOn: nullDate(in.EndsOn), AudiencePercent: aud(in), Goal: nullStr(in.Goal),
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

// RuleUpdate is PUT /messages/rules/:id. The whole definition is replaced; the version grows, a snapshot is stored
// and the medical approval is withdrawn (status back to draft) whatever was edited.
func (h *Handlers) RuleUpdate(c fiber.Ctx) error {
	cur, err := h.findRule(c)
	if err != nil {
		return err
	}
	in, err := parseInput(c)
	if err != nil {
		return err
	}
	now := httpadmin.Now(c)
	if err := h.mq.UpdateMessageRule(c.Context(), mstore.UpdateMessageRuleParams{
		Title: in.Title, Level: in.Level, Kind: in.Kind, Priority: uint16(in.Priority), //nolint:gosec // validated band
		Placement: in.Placement, Content: mustJSON(in.Content), VariantsEnabled: in.VariantsEnabled, AbSplit: ab(in),
		Conditions: mustJSON(orEmpty(in.Conditions)), Exceptions: mustJSON(orEmptyC(in.Exceptions)),
		CapCount: nullInt16(in.CapCount), CapPeriod: capPeriod(in), CooldownDays: uint16(in.CooldownDays), //nolint:gosec // validated
		HoursFrom: nullInt16(in.HoursFrom), HoursTo: nullInt16(in.HoursTo),
		StartsOn: nullDate(in.StartsOn), EndsOn: nullDate(in.EndsOn), AudiencePercent: aud(in), Goal: nullStr(in.Goal),
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
		slog.Int("version", int(m.Version)), slog.Bool("approval_withdrawn", cur.Status == rules.StatusApproved))
	return httpadmin.OK(c, jsonx.Obj("rule", h.ruleJSON(&m)), "Message rule updated.")
}

// RuleDestroy is DELETE /messages/rules/:id.
func (h *Handlers) RuleDestroy(c fiber.Ctx) error {
	m, err := h.findRule(c)
	if err != nil {
		return err
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

// RuleApprove is POST /messages/rules/:id/approve — the medical advisor (clinical capability, manage) only.
func (h *Handlers) RuleApprove(c fiber.Ctx) error {
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
	}
	if err := h.mq.SetMessageRuleStatus(c.Context(), p); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, action, "message_rule", m.ID)
	return h.respondRule(c, m.ID, msg)
}

// RuleToggle is POST /messages/rules/:id/toggle: publish / unpublish. A medical message must be approved first.
func (h *Handlers) RuleToggle(c fiber.Ctx) error {
	m, err := h.findRule(c)
	if err != nil {
		return err
	}
	if !m.IsActive && rules.NeedsMedicalApproval(m.Level, m.Kind) && m.Status != rules.StatusApproved {
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

// RuleList is GET /messages/rules?placement=&kind=&status=&page= with 7-day impressions and CTR per rule.
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
	since := httpadmin.DBTime(httpadmin.Now(c).Add(-statsWindow))
	srows, err := h.mq.MessageRuleEventStats(c.Context(), since)
	if err != nil {
		return err
	}
	stats := map[uint64]*stat{}
	for _, r := range srows {
		s := stats[r.RuleID]
		if s == nil {
			s = &stat{}
			stats[r.RuleID] = s
		}
		s.add(r.Event, r.Total)
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		o := h.ruleJSON(&rows[i])
		s := stat{}
		if st := stats[rows[i].ID]; st != nil {
			s = *st
		}
		o.Set("stats_7d", s.json())
		items = append(items, o)
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("enums", jsonx.Obj(
		"placements", jsonx.List(rules.Placements), "levels", jsonx.List(rules.Levels), "kinds", jsonx.List(rules.Kinds),
		"statuses", jsonx.List([]string{rules.StatusDraft, rules.StatusPending, rules.StatusApproved}),
		"variables", jsonx.List(rules.Variables), "ops", jsonx.List(rules.Ops), "cap_periods", jsonx.List(rules.CapPeriods),
		"bands", bands(),
	))
	return httpadmin.OK(c, page)
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
	for _, c := range cs {
		out = append(out, jsonx.Obj("rule_id", c.RuleID, "title", titles[c.RuleID], "priority", c.Priority, "kind", c.Kind,
			"winner", c.Winner, "safety", c.Safety))
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
	in := b.Rule
	r := &rules.Rule{
		ID: b.RuleID, Level: in.Level, Kind: in.Kind, Priority: in.Priority, Placement: in.Placement, Content: in.Content,
		VariantsEnabled: in.VariantsEnabled, ABSplit: 50, Groups: in.Conditions, Exceptions: in.Exceptions,
		CooldownDays: in.CooldownDays, HoursFrom: -1, HoursTo: -1, AudiencePercent: 100, IsActive: true,
		Status: rules.StatusApproved, // a preview shows what the message would do once published
	}
	if in.ABSplit != nil {
		r.ABSplit = *in.ABSplit
	}
	if in.AudiencePercent != nil {
		r.AudiencePercent = *in.AudiencePercent
	}
	if in.CapCount != nil {
		r.CapCount, r.CapPeriod = *in.CapCount, in.CapPeriod
	}
	if in.HoursFrom != nil && in.HoursTo != nil {
		r.HoursFrom, r.HoursTo = *in.HoursFrom, *in.HoursTo
	}
	if d, err := civildate.Parse(in.StartsOn); err == nil {
		r.StartsOn = d
	}
	if d, err := civildate.Parse(in.EndsOn); err == nil {
		r.EndsOn = d
	}
	return r, &b, nil
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
		if facts, err = messages.BuildFacts(ctx, h.user, b.UserID, civildate.InTehran(now)); err != nil {
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
