package insurance_test

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/insurance"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func itoa(n uint64) string { return strconv.FormatUint(n, 10) }

// Insurer catalog + contracted centres CRUD (CB-INS-01 admin, content capability, CSRF, validation).
func TestAdminInsurersAndCentres(t *testing.T) {
	e := admintest.New(t)
	box, err := loss.NewNoteBox(nil, nil, false)
	require.NoError(t, err)
	insurance.NewAdmin(insurance.NewService(e.DB, box), admintest.Quiet).Routes(e.Route(), e.Kit)
	a := e.As(admintest.EditorID)

	assert.Equal(t, 401, e.Anonymous().Get("/insurance/insurers").Status)
	assert.Equal(t, httpadmin.CapContent, httpadmin.RouteCapability(httpadmin.Prefix+"/insurance/centres"))
	noCSRF := e.As(admintest.EditorID)
	noCSRF.CSRF = ""
	assert.Equal(t, 419, noCSRF.JSON(fiber.MethodPost, "/insurance/insurers", map[string]any{}).Status)

	assert.Equal(t, 422, a.JSON(fiber.MethodPost, "/insurance/insurers", map[string]any{"code": "Bad Code", "name": "x"}).Status)
	assert.Equal(t, 422, a.JSON(fiber.MethodPost, "/insurance/insurers", map[string]any{"code": "x", "name": "x", "website": "javascript:alert(1)"}).Status)
	r := a.JSON(fiber.MethodPost, "/insurance/insurers", map[string]any{"code": "alpha", "name": "Alpha", "website": "https://alpha.example"})
	require.Equal(t, 201, r.Status, r.Body)
	iid := uint64(r.Data()["item"].(map[string]any)["id"].(float64))
	assert.Equal(t, 422, a.JSON(fiber.MethodPost, "/insurance/insurers", map[string]any{"code": "alpha", "name": "Dup"}).Status)
	r = a.JSON(fiber.MethodPut, "/insurance/insurers/"+itoa(iid), map[string]any{"code": "alpha", "name": "Alpha 2", "is_active": false})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, false, r.Data()["item"].(map[string]any)["is_active"])
	assert.Equal(t, 404, a.JSON(fiber.MethodPut, "/insurance/insurers/999", map[string]any{}).Status)

	assert.Equal(t, 422, a.JSON(fiber.MethodPost, "/insurance/centres", map[string]any{"name": "C", "type": "spa"}).Status)
	assert.Equal(t, 422, a.JSON(fiber.MethodPost, "/insurance/centres", map[string]any{"name": "C", "type": "lab", "insurer_id": 999}).Status)
	assert.Equal(t, 422, a.JSON(fiber.MethodPost, "/insurance/centres", map[string]any{"name": "C", "type": "lab", "lat": 35.7}).Status, "lat needs lng")
	r = a.JSON(fiber.MethodPost, "/insurance/centres", map[string]any{"name": "Alpha Lab", "type": "lab", "insurer_id": iid,
		"lat": 35.7, "lng": 51.4, "is_direct": true, "services": "Home sampling"})
	require.Equal(t, 201, r.Status, r.Body)
	item := r.Data()["item"].(map[string]any)
	cid := uint64(item["id"].(float64))
	assert.Equal(t, "35.7000000", item["lat"])
	assert.Equal(t, true, item["is_direct"])
	r = a.JSON(fiber.MethodPut, "/insurance/centres/"+itoa(cid), map[string]any{"name": "Alpha Lab", "type": "lab"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Nil(t, r.Data()["item"].(map[string]any)["insurer_id"])
	assert.Len(t, a.Get("/insurance/centres").Data()["items"], 1)

	assert.Equal(t, 200, a.JSON(fiber.MethodDelete, "/insurance/centres/"+itoa(cid), nil).Status)
	assert.Equal(t, 404, a.JSON(fiber.MethodDelete, "/insurance/centres/"+itoa(cid), nil).Status)
	assert.Equal(t, 200, a.JSON(fiber.MethodDelete, "/insurance/insurers/"+itoa(iid), nil).Status)
	assert.Empty(t, a.Get("/insurance/insurers").Data()["items"])
}
