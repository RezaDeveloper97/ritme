package http

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bloom B-N5-02 through the production wiring: the spouse shared-children picker of the companion routes now accepts
// the owner's own children (and still refuses anyone else's), the spouse sees a shared child read-only on
// /api/v1/children, and the companion home fills its child card.
func TestChildrenRoutes_SharedThroughCompanionLink(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120005001")
	ali := e.user("Ali", "09120005002")
	mina := e.user("Mina", "09120005003")
	e.step(sara, "gender", map[string]any{"gender": "female"})
	e.step(ali, "gender", map[string]any{"gender": "male"})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/onboarding/complete", ali, nil).Status)

	create := func(tok, name string) uint64 {
		r := e.do(fiber.MethodPost, "/api/v1/children", tok, map[string]any{"name": name, "birth_date": "2026-06-20", "sex": "girl"})
		require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
		return idOf(r.data()["id"])
	}
	ava := create(sara, "Ava")
	minas := create(mina, "Nika")

	r := e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{"type": "spouse", "child_ids": []uint64{minas}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "another owner's child")
	assert.Contains(t, r.Raw, `"child_ids"`)

	link, code := e.invite(sara, map[string]any{"type": "spouse", "child_ids": []uint64{ava}})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code}).Status)

	r = e.do(fiber.MethodGet, "/api/v1/children", ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	kids, _ := r.data()["children"].([]any)
	require.Len(t, kids, 1)
	k, _ := kids[0].(map[string]any)
	assert.Equal(t, "shared", k["role"])
	assert.Equal(t, "Sara", k["owner_name"])
	path := fmt.Sprintf("/api/v1/children/%d", ava)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, path+"/vaccines", ali, nil).Status)
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodPost, path+"/measurements", ali, map[string]any{"weight_kg": 6}).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, path, mina, nil).Status)

	r = e.home(ali)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	partners := partnersOf(r)
	require.Len(t, partners, 1)
	p, _ := partners[0].(map[string]any)
	card, _ := p["child"].(map[string]any)
	require.NotNil(t, card, r.Raw)
	assert.InDelta(t, 1, card["count"], 0)
	items, _ := card["items"].([]any)
	first, _ := items[0].(map[string]any)
	assert.Equal(t, "Ava", first["name"])
	next, _ := first["next_vaccine"].(map[string]any)
	assert.Equal(t, "birth", next["code"])

	// PUT /companions/{id}/children: own children only; [] unshares.
	links := fmt.Sprintf("/api/v1/companions/%d/children", link)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPut, links, sara, map[string]any{"child_ids": []uint64{minas}}).Status)
	r = e.do(fiber.MethodPut, links, sara, map[string]any{"child_ids": []uint64{}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, path, ali, nil).Status)
	p, _ = partnersOf(e.home(ali))[0].(map[string]any)
	assert.Nil(t, p["child"])
	r = e.do(fiber.MethodPut, links, sara, map[string]any{"child_ids": []uint64{ava}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, path, ali, nil).Status)

	// Deleting the child removes the share (FK cascade).
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, path, sara, nil).Status)
	assert.Zero(t, e.count("SELECT COUNT(*) FROM family_children"))
}
