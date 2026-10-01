package messages_test

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
)

func weekTip(locale string) map[string]any {
	return map[string]any{
		"group": "pregnancy_week_tip", "item_key": "8", "locale": locale,
		"payload": map[string]any{
			"title": "Tip", "body": "Body", "read_minutes": 2, "article_url": "/pregnancy/weeks/8", "extra": "dropped",
		},
	}
}

func alertRow(locale string) map[string]any {
	return map[string]any{
		"group": "pregnancy_alert", "item_key": "severe_symptom_count", "locale": locale,
		"payload": map[string]any{
			"enabled": true, "level": "follow_up", "window_days": 7,
			"params": map[string]any{"min_count": 3, "symptoms": []any{"nausea", "fatigue"}},
			"title":  "{count} severe", "what_we_saw": "Saw", "how_sure": "Sure", "advice": "Advice",
			"actions": []any{map[string]any{"key": "ack", "label": "OK"}}, "contact": nil,
		},
	}
}

func TestStoreRegistered(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.JSON(fiber.MethodPost, "/messages", weekTip("fa"))
	require.Equal(t, 201, r.Status, r.Body)
	m := r.Obj("message")
	assert.Equal(t, "pregnancy_week_tip", m["group"])
	assert.Equal(t, "8", m["item_key"])
	assert.Equal(t, "fa", m["locale"])
	assert.Equal(t, "pregnancy_week_tip / 8", m["label"])
	assert.Equal(t, true, m["is_active"])
	assert.Equal(t, true, m["is_approved"])
	assert.Equal(t, map[string]any{"title": "Tip", "body": "Body", "read_minutes": float64(2),
		"article_url": "/pregnancy/weeks/8"}, m["payload"], "schema order, unknown keys dropped")
	// Stored raw, like the seed rows: no \/ or \uXXXX escaping (QA 2026-09-29-c L8).
	assert.Equal(t, `{"title":"Tip","body":"Body","read_minutes":2,"article_url":"/pregnancy/weeks/8"}`,
		e.String("SELECT payload FROM message_contents WHERE `group` = 'pregnancy_week_tip' AND item_key = '8' AND locale = 'fa'"))

	// The same row again → unique.
	r = c.JSON(fiber.MethodPost, "/messages", weekTip("fa"))
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "item_key")

	// Unregistered group / key, inactive locale.
	r = c.JSON(fiber.MethodPost, "/messages", map[string]any{"group": "free_text", "item_key": "x", "locale": "fa",
		"payload": map[string]any{"a": "b"}})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "group")
	body := weekTip("fa")
	body["item_key"] = "43"
	r = c.JSON(fiber.MethodPost, "/messages", body)
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "item_key")
	r = c.JSON(fiber.MethodPost, "/messages", weekTip("de"))
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "locale")

	// Payload shape.
	bad := weekTip("en")
	bad["payload"] = map[string]any{"body": "B", "read_minutes": 0, "article_url": "javascript:alert(1)"}
	r = c.JSON(fiber.MethodPost, "/messages", bad)
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "payload.title")
	assert.Contains(t, r.Errors(), "payload.read_minutes")
	assert.Contains(t, r.Errors(), "payload.article_url")

	// Flags and label.
	body = weekTip("en")
	body["is_approved"] = false
	body["label"] = "Week 8"
	r = c.JSON(fiber.MethodPost, "/messages", body)
	require.Equal(t, 201, r.Status, r.Body)
	assert.Equal(t, false, r.Obj("message")["is_approved"])
	assert.Equal(t, "Week 8", r.Obj("message")["label"])

	// Guards.
	assert.Equal(t, 401, e.Anonymous().JSON(fiber.MethodPost, "/messages", weekTip("fa")).Status)
	noToken := e.As(admintest.EditorID)
	noToken.CSRF = ""
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/messages", weekTip("fa")).Status)
}

func TestStoreAlertRuleParams(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	bad := alertRow("fa")
	p := bad["payload"].(map[string]any)
	p["params"] = map[string]any{"min_count": 0, "symptoms": []any{"nausea", "nausea", "bleeding"}}
	p["level"] = "panic"
	p["actions"] = []any{map[string]any{"key": "dance", "label": "x"}}
	r := c.JSON(fiber.MethodPost, "/messages", bad)
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"payload.params.min_count", "payload.params.symptoms.2", "payload.level", "payload.actions.0.key"} {
		assert.Contains(t, r.Errors(), f)
	}

	r = c.JSON(fiber.MethodPost, "/messages", alertRow("fa"))
	require.Equal(t, 201, r.Status, r.Body)
	id := r.Obj("message")["id"].(float64)

	// Typed PUT: a partial payload keeps the nested params / actions (the legacy merge would
	// flatten them into string lists) and is validated against the schema.
	r = c.JSON(fiber.MethodPut, "/messages/"+ftoa(id), map[string]any{"payload": map[string]any{"title": "New"}})
	require.Equal(t, 200, r.Status, r.Body)
	pl := r.Obj("message")["payload"].(map[string]any)
	assert.Equal(t, "New", pl["title"])
	assert.Equal(t, map[string]any{"min_count": float64(3), "symptoms": []any{"nausea", "fatigue"}}, pl["params"])
	assert.Equal(t, []any{map[string]any{"key": "ack", "label": "OK"}}, pl["actions"])

	r = c.JSON(fiber.MethodPut, "/messages/"+ftoa(id), map[string]any{"payload": map[string]any{"window_days": 99}})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "payload.window_days")
}

func TestMissingAndRegistry(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.Get("/messages?group=pregnancy_setup&locale=en")
	require.Equal(t, 200, r.Status, r.Body)
	missing := r.Data()["missing"].([]any)
	assert.Len(t, missing, 10) // the 10 registered pregnancy_setup items (calendar_note since T-M7-20, loss_exit since B-N2-03)
	assert.Equal(t, map[string]any{"group": "pregnancy_setup", "item_key": "welcome", "locale": "en"}, missing[0])
	assert.Contains(t, r.Data()["registered_groups"], "pregnancy_alert")

	r = c.Get("/messages?group=pregnancy_week_tip")
	assert.Len(t, r.Data()["missing"], 42*2, "every week × active language")
	require.Equal(t, 201, c.JSON(fiber.MethodPost, "/messages", weekTip("fa")).Status)
	r = c.Get("/messages?group=pregnancy_week_tip")
	assert.Len(t, r.Data()["missing"], 42*2-1)

	r = c.Get("/messages/registry")
	require.Equal(t, 200, r.Status)
	groups := r.Data()["groups"].([]any)
	first := groups[0].(map[string]any)
	assert.Equal(t, "pregnancy_week_tip", first["group"])
	assert.Equal(t, true, first["typed"])

	r = c.Get("/messages/registry/pregnancy_week_tip/8")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Len(t, r.Data()["existing"], 1)
	assert.Equal(t, "Tip", r.Data()["template"].(map[string]any)["title"], "template = the existing row")
	fields := r.Data()["fields"].([]any)
	assert.Equal(t, "title", fields[0].(map[string]any)["key"])

	r = c.Get("/messages/registry/pregnancy_alert/bp_high")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, []any{"systolic", "diastolic"}, r.Data()["placeholders"])

	r = c.Get("/messages/registry/cycle_base_non_ttc/menstruation")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, false, r.Data()["typed"])
	assert.NotEmpty(t, r.Data()["template"].(map[string]any)["short"], "template = code fallback copy")

	assert.Equal(t, 404, c.Get("/messages/registry/pregnancy_alert/nope").Status)
}

// condition_nudge (CB-COND-06b): the heavy pain / bleeding nudge copy is created and edited like any typed group;
// the engine (internal/messages/conditionnudges) reads the row by (group, rule key, locale).
func TestStoreAndEditConditionNudge(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.JSON(fiber.MethodPost, "/messages", map[string]any{
		"group": "condition_nudge", "item_key": "heavy_pain", "locale": "fa",
		"payload": map[string]any{"title": "درد زیاد", "body": "{days} روز درد", "action": "دفترچه درد",
			"doctor_action": "پزشک", "extra": "dropped"},
	})
	require.Equal(t, 201, r.Status, r.Body)
	m := r.Obj("message")
	assert.Equal(t, "condition_nudge", m["group"])
	assert.Equal(t, "heavy_pain", m["item_key"])
	assert.Equal(t, map[string]any{"title": "درد زیاد", "body": "{days} روز درد", "action": "دفترچه درد",
		"doctor_action": "پزشک"}, m["payload"], "schema order, unknown keys dropped")

	id := ftoa(m["id"].(float64))
	r = c.JSON(fiber.MethodPut, "/messages/"+id, map[string]any{
		"payload": map[string]any{"title": "درد شدید", "body": "{days} روز", "action": "باز کن", "doctor_action": "پزشک"},
	})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "درد شدید", r.Obj("message")["payload"].(map[string]any)["title"])

	// Every text is required; an unknown rule key is refused.
	r = c.JSON(fiber.MethodPost, "/messages", map[string]any{
		"group": "condition_nudge", "item_key": "heavy_bleeding", "locale": "en",
		"payload": map[string]any{"title": "Heavy flow"},
	})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "payload.body")
	r = c.JSON(fiber.MethodPost, "/messages", map[string]any{
		"group": "condition_nudge", "item_key": "spotting", "locale": "en",
		"payload": map[string]any{"title": "T", "body": "B", "action": "A", "doctor_action": "D"},
	})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "item_key")

	r = c.Get("/messages/registry/condition_nudge/heavy_bleeding")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, true, r.Data()["typed"])
	assert.Equal(t, []any{"days"}, r.Data()["placeholders"])
}

func ftoa(f float64) string { return strconv.FormatInt(int64(f), 10) }
