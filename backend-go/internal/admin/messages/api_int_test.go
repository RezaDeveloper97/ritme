package messages_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/messages"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	messages.New(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	e.Exec("DELETE FROM message_contents") // migrations seed rows (e.g. pregnancy v2); start from a known set
	e.Exec("INSERT INTO message_contents (id, `group`, item_key, locale, label, payload, is_active, is_approved, sort_order, created_at, updated_at) VALUES " +
		`(1, 'pattern', 'a', 'fa', 'A', '{"title":"سلام","tips":["x","y"]}', 1, 1, 0, NOW(), NOW()),
		 (2, 'pattern', 'a', 'en', 'A', '{"title":"Hi","tips":["x"]}', 1, 0, 0, NOW(), NOW()),
		 (3, 'sleep_cycle', 'b', 'fa', NULL, '{"text":"خواب"}', 0, 0, 0, NOW(), NOW())`)
	return e
}

func ids(r admintest.Resp) []float64 {
	var out []float64
	for _, it := range r.Items() {
		out = append(out, it.(map[string]any)["id"].(float64))
	}
	return out
}

func TestListFilters(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	r := c.Get("/messages")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, []float64{2, 1, 3}, ids(r), "group, item_key, locale")
	assert.Equal(t, []any{"pattern", "sleep_cycle"}, r.Data()["groups"])
	assert.Equal(t, []any{"en", "fa"}, r.Data()["locales"])

	assert.Equal(t, []float64{2, 1}, ids(c.Get("/messages?group=pattern")))
	assert.Equal(t, []float64{1, 3}, ids(c.Get("/messages?locale=fa")))
	assert.Equal(t, []float64{1}, ids(c.Get("/messages?status=approved")))
	assert.Equal(t, []float64{2, 3}, ids(c.Get("/messages?status=pending")))
	assert.Empty(t, ids(c.Get("/messages?group=%25")), "wildcards are literal")
	assert.Equal(t, 401, e.Anonymous().Get("/messages").Status)
}

func TestEditApproveToggle(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.JSON(fiber.MethodPut, "/messages/1", map[string]any{
		"payload": map[string]any{"title": "درود", "tips": "one\r\n  two \n\n", "extra": "ignored"},
	})
	require.Equal(t, 200, r.Status, r.Body)
	m := r.Obj("message")
	assert.Equal(t, map[string]any{"title": "درود", "tips": []any{"one", "two"}}, m["payload"],
		"shape kept, lists split by line, unknown keys ignored")
	assert.Equal(t, "A", m["label"], "absent label unchanged")

	r = c.JSON(fiber.MethodPut, "/messages/1", map[string]any{
		"payload": map[string]any{"title": "", "tips": []string{"a", " ", "b"}}, "label": "",
	})
	require.Equal(t, 200, r.Status, r.Body)
	m = r.Obj("message")
	assert.Equal(t, map[string]any{"title": "درود", "tips": []any{"a", "b"}}, m["payload"],
		"an emptied scalar keeps its text (ConvertEmptyStringsToNull)")
	assert.Nil(t, m["label"])

	r = c.JSON(fiber.MethodPut, "/messages/1", map[string]any{"payload": "nope"})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "payload")

	r = c.JSON(fiber.MethodPost, "/messages/2/approve", nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, true, r.Obj("message")["is_approved"])
	assert.Equal(t, "Message approved.", r.Body["message"])
	r = c.JSON(fiber.MethodPost, "/messages/2/approve", nil)
	assert.Equal(t, false, r.Obj("message")["is_approved"])

	r = c.JSON(fiber.MethodPost, "/messages/3/toggle", nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, true, r.Obj("message")["is_active"])

	assert.Equal(t, 404, c.Get("/messages/99").Status)
	assert.Equal(t, 404, c.JSON(fiber.MethodPost, "/messages/x/toggle", nil).Status)
	c.CSRF = ""
	assert.Equal(t, 419, c.JSON(fiber.MethodPost, "/messages/3/toggle", nil).Status)
}

// QA 2026-09-29-c L8: an admin save stores Persian as raw UTF-8 (like the seed rows), so the
// column stays compact and a plain `LIKE '%فارسی%'` on the DB finds it.
func TestEditStoresRawUTF8(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	r := c.JSON(fiber.MethodPut, "/messages/1", map[string]any{
		"payload": map[string]any{"title": "درود بر تو", "tips": []string{"آب/بخور"}},
	})
	require.Equal(t, 200, r.Status, r.Body)
	stored := e.String("SELECT payload FROM message_contents WHERE id = 1")
	assert.Contains(t, stored, "درود بر تو")
	assert.Contains(t, stored, "آب/بخور")
	assert.NotContains(t, stored, `\u`)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM message_contents WHERE payload LIKE '%درود بر تو%'"))
}
