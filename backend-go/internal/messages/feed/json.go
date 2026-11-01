package feed

import (
	"encoding/json"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// JSON is data of GET /messages/feed: {slots: {<slot>: [item…]}, alerts: {open, urgent}, rules_rev}.
func (r *Result) JSON(locale string) *jsonx.OrderedMap {
	x := r.x
	slots := jsonx.Obj()
	for _, s := range r.Slots {
		items := make([]*jsonx.OrderedMap, 0, len(s.Choices))
		for i := range s.Choices {
			items = append(items, itemJSON(x, &s.Choices[i], s.Texts[i], locale, s.AlertIDs))
		}
		slots.Set(s.Name, items)
	}
	out := jsonx.Obj("slots", slots, "alerts", jsonx.Obj("open", r.Alerts.Open, "urgent", r.Alerts.Urgent),
		"rules_rev", r.Rev)
	if r.Challenge != nil { // T-M10-10: only when the request computed the pick
		out.Set("daily_challenge", challengeJSON(r.Challenge, locale))
	}
	return out
}

// challengeJSON is the daily_challenge pick: null, or {id, title, description, category, duration_days, day, reason}.
func challengeJSON(c *ChallengeResult, locale string) any {
	if c.Card == nil {
		return nil
	}
	pick := func(raw json.RawMessage) any {
		if len(raw) == 0 {
			return nil
		}
		var v any
		if json.Unmarshal(i18n.Pick(raw, locale, ""), &v) != nil {
			return nil
		}
		return v
	}
	var category, day any
	if c.Card.Category != "" {
		category = c.Card.Category
	}
	if c.Card.Day > 0 {
		day = c.Card.Day
	}
	return jsonx.Obj("id", c.Card.ID, "title", pick(c.Card.Title), "description", pick(c.Card.Description),
		"category", category, "duration_days", c.Card.DurationDays, "day", day, "reason", c.Reason)
}

// itemJSON is one feed item. evidence («what we saw», «how sure we are», the contact of an urgent message) is present
// only for the follow-up and urgent levels; level is the spec name (SPEC §5.3).
func itemJSON(x *extras, ch *rules.Choice, v rules.Variant, locale string, alertIDs map[uint64]uint64) *jsonx.OrderedMap {
	r := ch.Rule
	var key any
	if r.RuleKey != "" {
		key = r.RuleKey
	}
	o := jsonx.Obj("id", r.ID, "rule_key", key, "version", ServedVersion(r), "variant", ch.Variant, "kind", r.Kind,
		"level", rules.SpecLevel(r.Level), "title", v.Title, "body", v.Body)
	if v.CTALabel != "" && v.CTATarget != "" {
		o.Set("cta_text", v.CTALabel)
		o.Set("cta_deeplink", v.CTATarget)
	} else {
		o.Set("cta_text", nil)
		o.Set("cta_deeplink", nil)
	}
	if id, ok := alertIDs[r.ID]; ok { // only items fanned out from the inbox carry the row id for POST /alerts/{id}/ack
		o.Set("alert_id", id)
	}
	o.Set("dismissible", r.Level != rules.LevelFollowSoon)
	if r.Level == rules.LevelWorthFollowUp || r.Level == rules.LevelFollowSoon {
		c := r.Content
		if x != nil {
			c = x.evidence(r, locale)
		}
		ev := jsonx.Obj("seen", c.Seen, "confidence", c.Confidence)
		if r.Level == rules.LevelFollowSoon {
			ev.Set("contact", c.Contact)
		}
		o.Set("evidence", ev)
	}
	return o
}

// Evidence is rule's evidence in locale (for /messages/selected's legacy shape).
func (r *Result) Evidence(rule *rules.Rule, locale string) rules.Content {
	if r.x != nil {
		return r.x.evidence(rule, locale)
	}
	return rule.Content
}
