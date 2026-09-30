package content_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
)

// B-N1-12b: the optional `key` of an info box (the stable id screens read, e.g. support/email) is editable —
// a lowercase slug, unique within its group; absent on update = unchanged, null = cleared.
func TestInfoSectionKey(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	box := func(kv ...any) map[string]any {
		m := map[string]any{"group": "support", "heading": map[string]any{"fa": "ایمیل"}, "body": map[string]any{"fa": "متن"}}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}

	r := c.Get("/info-sections/options?group=support")
	require.Equal(t, 200, r.Status)
	assert.Contains(t, r.Data()["groups"], "support")
	assert.Equal(t, "support", r.Data()["group"])

	r = c.JSON(fiber.MethodPost, "/info-sections", box("key", "telegram"))
	require.Equal(t, 201, r.Status, r.Body)
	assert.Equal(t, "telegram", r.Obj("info_section")["key"])
	rid := id(t, r.Obj("info_section"))

	for _, bad := range []string{"Bad Key", "tele gram", "-x", "x-", "ایمیل", "a__b"} {
		r = c.JSON(fiber.MethodPost, "/info-sections", box("key", bad))
		assert.Equal(t, 422, r.Status, bad)
		assert.Contains(t, r.Errors(), "key", bad)
	}
	r = c.JSON(fiber.MethodPost, "/info-sections", box("key", "telegram"))
	require.Equal(t, 422, r.Status, "unique within the group")
	assert.Contains(t, r.Errors(), "key")

	r = c.JSON(fiber.MethodPost, "/info-sections", box("group", "help", "key", "telegram"))
	require.Equal(t, 201, r.Status, "the same key in another group is fine")

	r = c.JSON(fiber.MethodPost, "/info-sections", box())
	require.Equal(t, 201, r.Status, "key is optional")
	assert.Nil(t, r.Obj("info_section")["key"])

	r = c.JSON(fiber.MethodPut, "/info-sections/"+rid, box("key", "telegram"))
	require.Equal(t, 200, r.Status, "a box keeps its own key")

	r = c.JSON(fiber.MethodPut, "/info-sections/"+rid, box())
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "telegram", r.Obj("info_section")["key"], "absent key = unchanged")

	r = c.JSON(fiber.MethodPut, "/info-sections/"+rid, box("key", "phone_line"))
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "phone_line", r.Obj("info_section")["key"])

	r = c.JSON(fiber.MethodPut, "/info-sections/"+rid, box("key", ""))
	require.Equal(t, 200, r.Status, r.Body)
	assert.Nil(t, r.Obj("info_section")["key"], "empty key = cleared")
}
