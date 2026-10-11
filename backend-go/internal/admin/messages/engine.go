package messages

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/messages/rules"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Engine view and simulator (bloom B-N9-06): GET /messages/rules/engine lists the live rules per placement in the
// engine's order with conflict counts; POST /messages/rules/simulate runs the pure engine over synthetic facts
// (no database writes, no real user data) and returns the winner and the evaluation trace per placement. A real
// user's facts are only reachable through the audited /messages/rules/preview.

func (h *Handlers) engineRoutes(route func(method, path string, chain httpadmin.Chain), kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/messages/rules/engine", kit.Admin(h.EngineView))
	route(fiber.MethodPost, "/messages/rules/simulate", kit.Admin(h.Simulate))
}

func (h *Handlers) activeByPlacement(c fiber.Ctx, only []string) (map[string][]rules.Rule, map[uint64]string, error) {
	rows, err := h.mq.ListActiveMessageRules(c.Context(), mstore.ListActiveMessageRulesParams{})
	if err != nil {
		return nil, nil, err
	}
	by := map[string][]rules.Rule{}
	titles := map[uint64]string{}
	for i := range rows {
		if len(only) > 0 && !slices.Contains(only, rows[i].Placement) {
			continue
		}
		by[rows[i].Placement] = append(by[rows[i].Placement], rules.FromRow(&rows[i]))
		titles[rows[i].ID] = rows[i].Title
	}
	return by, titles, nil
}

// EngineView is GET /messages/rules/engine.
func (h *Handlers) EngineView(c fiber.Ctx) error {
	by, titles, err := h.activeByPlacement(c, nil)
	if err != nil {
		return err
	}
	places := make([]*jsonx.OrderedMap, 0, len(rules.Placements))
	for _, p := range rules.Placements {
		list := by[p]
		slices.SortStableFunc(list, func(a, b rules.Rule) int {
			if a.Priority != b.Priority {
				return b.Priority - a.Priority
			}
			return int(a.ID) - int(b.ID) //nolint:gosec // small ids
		})
		items := make([]*jsonx.OrderedMap, 0, len(list))
		for i := range list {
			r := &list[i]
			o := jsonx.Obj("rule_id", r.ID, "title", titles[r.ID], "kind", r.Kind, "level", r.Level, "priority", r.Priority,
				"served", r.Served(), "unlimited", r.Unlimited(), "cooldown_days", r.CooldownDays,
				"audience_percent", r.AudiencePercent, "variants", r.VariantsEnabled && r.Content.B != nil,
				"conflicts", len(rules.FindConflicts(r, list)))
			if r.CapCount > 0 {
				o.Set("cap", jsonx.Obj("count", r.CapCount, "period", r.CapPeriod))
			} else {
				o.Set("cap", nil)
			}
			items = append(items, o)
		}
		places = append(places, jsonx.Obj("placement", p, "urgent", slices.Contains(rules.UrgentPlacements, p),
			"count", len(items), "rules", items))
	}
	return httpadmin.OK(c, jsonx.Obj("placements", places, "bands", bands()))
}

type simulateBody struct {
	Facts      map[string]any `json:"facts"`
	At         string         `json:"at"`
	UserID     uint64         `json:"user_id"`
	Placements []string       `json:"placements"`
}

// Simulate is POST /messages/rules/simulate {facts, at?, user_id?, placements?}. user_id only seeds the stable A/B
// and audience buckets (no data is read for it). History is empty: caps and cooldown never trigger.
func (h *Handlers) Simulate(c fiber.Ctx) error {
	var b simulateBody
	if err := json.Unmarshal(c.Body(), &b); err != nil {
		return httpadmin.FieldError("facts", "The request body is not valid JSON.")
	}
	for _, p := range b.Placements {
		if !slices.Contains(rules.Placements, p) {
			return httpadmin.FieldError("placements", "The placement is invalid.")
		}
	}
	now := httpadmin.Now(c)
	if b.At != "" {
		t, err := time.Parse(time.RFC3339, b.At)
		if err != nil {
			return httpadmin.FieldError("at", "The date is invalid.")
		}
		now = t
	}
	uid := b.UserID
	if uid == 0 {
		uid = 1
	}
	facts := rules.Facts{}
	for k, v := range b.Facts {
		facts[k] = v
	}
	by, titles, err := h.activeByPlacement(c, b.Placements)
	if err != nil {
		return err
	}
	out := []*jsonx.OrderedMap{}
	for _, p := range rules.Placements {
		if len(b.Placements) > 0 && !slices.Contains(b.Placements, p) {
			continue
		}
		vs, err := rules.Simulate(by[p], facts, uid, now, nil)
		if err != nil {
			return err
		}
		trace := make([]*jsonx.OrderedMap, 0, len(vs))
		var winner any
		for _, v := range vs {
			r := v.Rule
			trace = append(trace, jsonx.Obj("rule_id", r.ID, "title", titles[r.ID], "kind", r.Kind, "level", r.Level,
				"priority", r.Priority, "eligible", v.Eligible, "reason", v.Reason, "failed_groups", intsOrEmpty(v.FailedGroups),
				"variant", v.Variant, "winner", v.Winner))
			if v.Winner {
				rv, _ := rules.RenderVariant(r.VariantContent(v.Variant), facts)
				winner = jsonx.Obj("rule_id", r.ID, "title", titles[r.ID], "variant", v.Variant, "level", r.Level,
					"kind", r.Kind, "priority", r.Priority, "message_title", rv.Title, "message_body", rv.Body)
			}
		}
		out = append(out, jsonx.Obj("placement", p, "winner", winner, "trace", trace))
	}
	return httpadmin.OK(c, jsonx.Obj("at", now.UTC().Format(time.RFC3339), "placements", out))
}

func intsOrEmpty(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}
