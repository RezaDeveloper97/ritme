package checkups_test

import (
	"context"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/checkups/store"
)

// CB-MENO-01b: the admin API reads and writes checkup_types.audiences (validated life modes, NULL = everyone) and
// the user API follows it on the next request.
func TestAudiences_AdminReadWrite(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	ctx := context.Background()

	r := c.Get("/checkup-types/options")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, []any{"cycle", "ttc", "pregnancy", "postpartum", "menopause", "teen"}, r.Data()["audiences"])

	r = c.Get(path(typeID(e, "dentist")))
	require.Equal(t, 200, r.Status, r.Body)
	dentist := r.Obj("checkup_type")
	require.Contains(t, dentist, "audiences")
	assert.Nil(t, dentist["audiences"], "shared rows are for everyone")

	// Validation: a list of known modes.
	r = c.JSON(fiber.MethodPost, "/checkup-types", with(validBody(), "audiences", []any{"menopause", "grandma"}))
	require.Equal(t, 422, r.Status, r.Body)
	assert.Contains(t, r.Errors(), "audiences.1")
	assert.NotContains(t, r.Errors(), "audiences.0")
	r = c.JSON(fiber.MethodPost, "/checkup-types", with(validBody(), "audiences", "menopause"))
	require.Equal(t, 422, r.Status, r.Body)
	assert.Contains(t, r.Errors(), "audiences")

	// Create for menopause (duplicates dropped).
	r = c.JSON(fiber.MethodPost, "/checkup-types", with(validBody(), "audiences", []any{"menopause", "menopause"}))
	require.Equal(t, 201, r.Status, r.Body)
	created := r.Obj("checkup_type")
	assert.Equal(t, []any{"menopause"}, created["audiences"])
	id := int(created["id"].(float64))

	cycleUser := newUser(e, "09120000201")
	menoUser := newUser(e, "09120000202")
	e.Exec("INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, 'menopause', NOW(), NOW())", menoUser)
	sees := func(userID uint64) bool {
		in, err := checkups.EngineInputs(ctx, store.New(e.DB), userID)
		require.NoError(t, err)
		for _, ty := range in.Types {
			if int(ty.ID) == id { //nolint:gosec // test ids
				return true
			}
		}
		return false
	}
	assert.False(t, sees(cycleUser))
	assert.True(t, sees(menoUser))

	// An update without audiences keeps them; [] / null clears (everyone).
	r = c.JSON(fiber.MethodPut, path(id), without(validBody(), "key"))
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, []any{"menopause"}, r.Obj("checkup_type")["audiences"])
	r = c.JSON(fiber.MethodPut, path(id), with(without(validBody(), "key"), "audiences", []any{"cycle", "ttc"}))
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, []any{"cycle", "ttc"}, r.Obj("checkup_type")["audiences"])
	assert.True(t, sees(cycleUser))
	assert.False(t, sees(menoUser))
	r = c.JSON(fiber.MethodPut, path(id), with(without(validBody(), "key"), "audiences", []any{}))
	require.Equal(t, 200, r.Status, r.Body)
	assert.Nil(t, r.Obj("checkup_type")["audiences"])
	assert.True(t, sees(cycleUser))
	assert.True(t, sees(menoUser))
	r = c.JSON(fiber.MethodPut, path(id), with(without(validBody(), "key"), "audiences", []any{"teen"}))
	require.Equal(t, 200, r.Status, r.Body)
	r = c.JSON(fiber.MethodPut, path(id), with(without(validBody(), "key"), "audiences", nil))
	require.Equal(t, 200, r.Status, r.Body)
	assert.Nil(t, r.Obj("checkup_type")["audiences"])
}
