package httpadmin_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
)

func TestPageOfAndID(t *testing.T) {
	app := fiber.New()
	var page httpadmin.Pagination
	var id uint64
	var ok bool
	app.Get("/x/:id", func(c fiber.Ctx) error {
		page = httpadmin.PageOf(c)
		id, ok = httpadmin.ID(c, "id")
		return nil
	})
	cases := []struct {
		url      string
		page, pp int
		id       uint64
		ok       bool
	}{
		{"/x/5", 1, 20, 5, true},
		{"/x/5?page=3&per_page=50", 3, 50, 5, true},
		{"/x/5?page=0&per_page=1000", 1, 100, 5, true},
		{"/x/5?page=abc&per_page=-1", 1, 20, 5, true},
		{"/x/abc", 1, 20, 0, false},
		{"/x/0", 1, 20, 0, false},
		{"/x/007", 1, 20, 0, false},
		{"/x/99999999999999999999999", 1, 20, 0, false},
	}
	for _, tc := range cases {
		_, err := app.Test(httptest.NewRequest(fiber.MethodGet, tc.url, nil))
		require.NoError(t, err)
		assert.Equal(t, tc.page, page.Page, tc.url)
		assert.Equal(t, tc.pp, page.PerPage, tc.url)
		assert.Equal(t, tc.ok, ok, tc.url)
		if tc.ok {
			assert.Equal(t, tc.id, id, tc.url)
		}
	}
}
