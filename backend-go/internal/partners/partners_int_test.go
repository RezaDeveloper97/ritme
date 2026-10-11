package partners_test

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/partners"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

var at = time.Date(2026, 10, 11, 9, 0, 0, 0, civildate.Tehran)

type fakeImporter struct {
	calls int
	pdf   []byte
	meta  partners.ImportMeta
}

func (f *fakeImporter) ImportResult(_ context.Context, _ uint64, m partners.ImportMeta, pdf []byte, _ string, _ time.Time) (uint64, error) {
	f.calls++
	f.pdf = append([]byte(nil), pdf...)
	f.meta = m
	return 777, nil
}

type env struct {
	*admintest.Env
	svc *partners.Service
	imp *fakeImporter
	a   *admintest.Client
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := admintest.New(t)
	box, err := loss.NewNoteBox(nil, nil, false)
	require.NoError(t, err)
	imp := &fakeImporter{}
	svc := partners.NewService(partners.Options{DB: e.DB, Sealer: box, Importer: imp, Logger: admintest.Quiet})
	partners.NewAdmin(svc, admintest.Quiet).Routes(e.Route(), e.Kit)
	return &env{Env: e, svc: svc, imp: imp, a: e.As(admintest.EditorID)}
}

func (e *env) user(mobile string) uint64 {
	res := e.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, NOW(), NOW())`, mobile)
	id, err := res.LastInsertId()
	require.NoError(e.T, err)
	return uint64(id) //nolint:gosec // test ids
}

func (e *env) service(code string, active bool) uint64 {
	r := e.a.JSON(fiber.MethodPost, "/partners/lab-services", map[string]any{"code": code, "kind": "home_draw",
		"title": map[string]any{"fa": "آزمایش در منزل", "en": "Home blood draw"}, "partner_name": "Lab X", "is_active": active})
	require.Equal(e.T, 201, r.Status, r.Body)
	return uint64(r.Data()["id"].(float64))
}

func input(serviceID uint64) partners.Input {
	return partners.Input{ServiceID: serviceID, PreferredDate: civildate.New(2026, 10, 14), Window: partners.WindowMorning,
		ConsentShare: true, ConsentImport: true,
		Payload: partners.Payload{FullName: "Sara Karimi", Phone: "09120000000", City: "Tehran", Address: "Valiasr St 12", Note: "floor 3"}}
}

func TestAdminCatalogCRUDAndGuards(t *testing.T) {
	e := newEnv(t)
	assert.Equal(t, 401, e.Anonymous().Get("/partners/lab-services").Status)
	assert.Equal(t, httpadmin.CapSupport, httpadmin.RouteCapability(httpadmin.Prefix+"/partners/lab-requests"))
	noCSRF := e.As(admintest.EditorID)
	noCSRF.CSRF = ""
	assert.Equal(t, 419, noCSRF.JSON(fiber.MethodPost, "/partners/lab-services", map[string]any{}).Status)

	assert.Equal(t, 422, e.a.JSON(fiber.MethodPost, "/partners/lab-services", map[string]any{"code": "Bad Code"}).Status)
	id := e.service("home_blood_draw", true)
	dup := e.a.JSON(fiber.MethodPost, "/partners/lab-services", map[string]any{"code": "home_blood_draw", "kind": "checkup",
		"title": map[string]any{"fa": "x"}, "partner_name": "p"})
	assert.Equal(t, 422, dup.Status, "code is unique")
	unknown := e.a.JSON(fiber.MethodPost, "/partners/lab-services", map[string]any{"code": "c2", "kind": "checkup",
		"title": map[string]any{"fa": "x"}, "partner_name": "p", "partner_code": "nope"})
	assert.Equal(t, 422, unknown.Status, "partner adapter must exist")

	r := e.a.JSON(fiber.MethodPut, "/partners/lab-services/"+itoa(id), map[string]any{"code": "home_blood_draw", "kind": "home_draw",
		"title": map[string]any{"fa": "x", "en": "Y"}, "partner_name": "Lab Y", "is_active": false})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, false, r.Data()["is_active"])
	assert.Equal(t, 404, e.a.JSON(fiber.MethodPut, "/partners/lab-services/999", map[string]any{}).Status)

	active, err := e.svc.Services(context.Background())
	require.NoError(t, err)
	assert.Empty(t, active, "an inactive service is not on offer")
	assert.Len(t, e.a.Get("/partners/lab-services").Items(), 1)
}

func itoa(n uint64) string { return strconv.FormatUint(n, 10) }

func TestCreateSealsAndIsolates(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	sid := e.service("home_blood_draw", true)
	sara, ali := e.user("09120001001"), e.user("09120001002")

	r, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)
	assert.Equal(t, partners.StatusRequested, r.Status)
	assert.True(t, strings.HasPrefix(r.PartnerRef.String, partners.FakeRefPrefix))

	// Nothing personal is stored in the clear.
	var enc sql.NullString
	require.NoError(t, e.DB.QueryRow("SELECT payload_enc FROM partner_lab_requests WHERE id = ?", r.ID).Scan(&enc))
	require.True(t, enc.Valid)
	for _, secret := range []string{"Sara", "09120000000", "Valiasr", "Tehran", "floor"} {
		assert.NotContains(t, enc.String, secret)
	}
	p, ok := e.svc.Details(sara, r.Reference, enc)
	require.True(t, ok)
	assert.Equal(t, "Valiasr St 12", p.Address)
	// Bound to the owner and the reference: another user / reference cannot open the blob.
	_, ok = e.svc.Details(ali, r.Reference, enc)
	assert.False(t, ok)
	_, ok = e.svc.Details(sara, "LRXXXXXXXX", enc)
	assert.False(t, ok)

	// IDOR: another user's request is the same ErrNotFound as a missing one, for read, cancel and import.
	_, err = e.svc.Get(ctx, ali, r.ID)
	assert.ErrorIs(t, err, partners.ErrNotFound)
	assert.ErrorIs(t, e.svc.Cancel(ctx, ali, r.ID, at), partners.ErrNotFound)
	_, err = e.svc.Import(ctx, ali, r.ID, true, "en", "fa", at)
	assert.ErrorIs(t, err, partners.ErrNotFound)
	list, err := e.svc.List(ctx, ali)
	require.NoError(t, err)
	assert.Empty(t, list)

	// Inactive / unknown service, and the open-request cap.
	off := e.service("old", false)
	_, err = e.svc.Create(ctx, sara, input(off), at)
	assert.ErrorIs(t, err, partners.ErrServiceUnavail)
	_, err = e.svc.Create(ctx, sara, input(999), at)
	assert.ErrorIs(t, err, partners.ErrServiceUnavail)
	for i := 0; i < partners.MaxOpenRequests-1; i++ {
		_, err = e.svc.Create(ctx, sara, input(sid), at)
		require.NoError(t, err)
	}
	_, err = e.svc.Create(ctx, sara, input(sid), at)
	assert.ErrorIs(t, err, partners.ErrTooManyOpen)
}

func TestStatusMachineCancelAndAdminQueue(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	sid := e.service("home_blood_draw", true)
	sara := e.user("09120001001")
	r, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)
	url := "/partners/lab-requests/" + itoa(r.ID) + "/status"

	// The admin queue shows no personal data and masks the account mobile.
	q := e.a.Get("/partners/lab-requests")
	require.Equal(t, 200, q.Status, q.Body)
	require.Len(t, q.Items(), 1)
	for _, leak := range []string{"Sara", "Valiasr", "09120001001", "floor"} {
		assert.NotContains(t, string(q.Raw), leak)
	}
	assert.Equal(t, 422, e.a.JSON(fiber.MethodPost, url, map[string]any{"status": "cancelled"}).Status, "cancel is the user's")
	assert.Equal(t, 404, e.a.JSON(fiber.MethodPost, "/partners/lab-requests/999/status", map[string]any{"status": "accepted"}).Status)

	assert.Equal(t, 200, e.a.JSON(fiber.MethodPost, url, map[string]any{"status": "accepted"}).Status)
	assert.Equal(t, 409, e.a.JSON(fiber.MethodPost, url, map[string]any{"status": "accepted"}).Status, "no repeat / backwards")
	assert.Equal(t, 200, e.a.JSON(fiber.MethodPost, url, map[string]any{"status": "sampled"}).Status)
	got, err := e.svc.Get(ctx, sara, r.ID)
	require.NoError(t, err)
	assert.Equal(t, partners.StatusSampled, got.Status)
	assert.ErrorIs(t, e.svc.Cancel(ctx, sara, r.ID, at), partners.ErrNotCancellable, "after sampling")
	assert.Equal(t, 409, e.a.JSON(fiber.MethodPost, url, map[string]any{"status": "rejected"}).Status)

	// A requested one can be cancelled once, and the partner cannot move it afterwards.
	r2, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)
	require.NoError(t, e.svc.Cancel(ctx, sara, r2.ID, at))
	assert.ErrorIs(t, e.svc.Cancel(ctx, sara, r2.ID, at), partners.ErrNotCancellable)
	assert.Equal(t, 409, e.a.JSON(fiber.MethodPost, "/partners/lab-requests/"+itoa(r2.ID)+"/status", map[string]any{"status": "accepted"}).Status)

	// The sync loop applies what the partner reports (here: the fake's stored status) and ignores closed requests.
	r3, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)
	e.Exec("UPDATE partner_lab_requests SET partner_status = 'accepted' WHERE id = ?", r3.ID)
	n, err := e.svc.Sync(context.Background(), at)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, "accepted", e.String("SELECT status FROM partner_lab_requests WHERE id = ?", r3.ID))
}

func TestImportNeedsBothConsents(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	sid := e.service("home_blood_draw", true)
	sara := e.user("09120001001")

	noImport := input(sid)
	noImport.ConsentImport = false
	a, err := e.svc.Create(ctx, sara, noImport, at)
	require.NoError(t, err)
	b, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)

	// Not ready yet.
	_, err = e.svc.Import(ctx, sara, b.ID, true, "en", "fa", at)
	assert.ErrorIs(t, err, partners.ErrNoResult)
	for _, id := range []uint64{a.ID, b.ID} {
		require.NoError(t, e.svc.Advance(ctx, id, partners.StatusResultReady, at))
	}
	// Ticked nothing on the request → never; ticked on the request but not confirmed now → not either.
	_, err = e.svc.Import(ctx, sara, a.ID, true, "en", "fa", at)
	assert.ErrorIs(t, err, partners.ErrImportConsent)
	_, err = e.svc.Import(ctx, sara, b.ID, false, "en", "fa", at)
	assert.ErrorIs(t, err, partners.ErrImportConsent)
	assert.Zero(t, e.imp.calls)

	id, err := e.svc.Import(ctx, sara, b.ID, true, "en", "fa", at)
	require.NoError(t, err)
	assert.EqualValues(t, 777, id)
	assert.Equal(t, 1, e.imp.calls)
	assert.True(t, strings.HasPrefix(string(e.imp.pdf), "%PDF-"), "the canned fake sheet")
	assert.Equal(t, "blood", e.imp.meta.Category)
	assert.Equal(t, "Home blood draw", e.imp.meta.Title)
	_, err = e.svc.Import(ctx, sara, b.ID, true, "en", "fa", at)
	assert.ErrorIs(t, err, partners.ErrAlreadyImported)
	assert.Equal(t, 1, e.imp.calls)
}

func TestPurgeWipesThePersonalBlobAfter30Days(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	sid := e.service("home_blood_draw", true)
	sara := e.user("09120001001")
	open, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)
	done, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)
	require.NoError(t, e.svc.Cancel(ctx, sara, done.ID, at))

	n, err := e.svc.Purge(ctx, at.Add(29*24*time.Hour))
	require.NoError(t, err)
	assert.Zero(t, n, "still inside the retention window")
	n, err = e.svc.Purge(ctx, at.Add(31*24*time.Hour))
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "only the closed request")

	var enc sql.NullString
	require.NoError(t, e.DB.QueryRow("SELECT payload_enc FROM partner_lab_requests WHERE id = ?", done.ID).Scan(&enc))
	assert.False(t, enc.Valid)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM partner_lab_requests WHERE id = ? AND purged_at IS NOT NULL", done.ID))
	got, err := e.svc.Get(ctx, sara, done.ID)
	require.NoError(t, err)
	_, ok := e.svc.Details(sara, got.Reference, got.PayloadEnc)
	assert.False(t, ok)
	still, err := e.svc.Get(ctx, sara, open.ID)
	require.NoError(t, err)
	assert.True(t, still.PayloadEnc.Valid, "an open request keeps its details")
}
