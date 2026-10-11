package telemed_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/telemed"
)

type cenv struct {
	*benv
	files *files.Service
	box   *loss.NoteBox
}

// doctorAdminID is the `doctor`-role admin setupChat links to the booking's doctor.
const doctorAdminID uint64 = 3

var chatPDF = []byte("%PDF-1.4\n1 0 obj << /Report 1 >> endobj\n%%EOF")

func setupChat(t *testing.T) *cenv {
	t.Helper()
	be := setupBooking(t, true)
	box, err := loss.NewNoteBox(nil, nil, false)
	require.NoError(t, err)
	vault, err := files.NewVault(be.Storage, files.DevKey("ritme-file-dev-key"), nil, false)
	require.NoError(t, err)
	svc := files.NewService(files.Options{DB: be.DB, Vault: vault, BaseURL: appURL})
	ch := telemed.NewChat(be.bk, box, svc, telemed.FakeVideo{})
	h := telemed.NewChatHandlers(ch, be.clk)
	locale := i18n.Middleware(be.Registry)
	b := "/api/v1/telemed/bookings"
	g := be.guard
	be.App.Get(b+"/:id/messages", locale, g, h.Messages)
	be.App.Post(b+"/:id/messages", locale, g, h.SendMessage)
	be.App.Get(b+"/:id/prescription", locale, g, h.Prescription)
	be.App.Post(b+"/:id/prescription/:item/reminder", locale, g, h.AddReminder)
	be.App.Get(b+"/:id/video", locale, g, h.Video)
	telemed.NewDoctorAdmin(ch).Routes(be.Route(), be.Kit)
	// The doctor inbox needs the telemed_doctor capability, which only the doctor role (and super) holds since
	// B-N9-02b narrowed editor; the doctor is a `doctor` admin linked to the booking's doctor.
	be.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at)
		SELECT ?, 'Doctor', 'doctor@ritme.test', password, 'doctor', 1, created_at, updated_at FROM admins WHERE id = ?`,
		doctorAdminID, admintest.EditorID)
	be.Exec(`UPDATE telemed_doctors SET admin_id = ? WHERE id = ?`, doctorAdminID, be.a)
	return &cenv{benv: be, files: svc, box: box}
}

func (ce *cenv) path(bid uint64, rest string) string {
	return fmt.Sprintf("/api/v1/telemed/bookings/%d%s", bid, rest)
}

func (ce *cenv) at(hh, mm, ss int) time.Time {
	return time.Date(2030, 1, 8, hh, mm, ss, 0, fixed.Location())
}

func TestChat_Unauthenticated(t *testing.T) {
	ce := setupChat(t)
	for _, tc := range []struct{ m, p string }{
		{http.MethodGet, "/messages"}, {http.MethodPost, "/messages"}, {http.MethodGet, "/prescription"},
		{http.MethodPost, "/prescription/1/reminder"}, {http.MethodGet, "/video"},
	} {
		r := ce.do(t, tc.m, ce.path(1, tc.p), "", "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, tc.p)
	}
	assert.Equal(t, http.StatusUnauthorized, ce.Anonymous().Get("/telemed/doctor/bookings").Status)
}

func TestChat_IDORAndEncryption(t *testing.T) {
	ce := setupChat(t)
	_, tokA := ce.user(t, "A", "09120000001")
	_, tokB := ce.user(t, "B", "09120000002")
	bid := ce.pay(t, tokA, "16:00", nil)

	// Another user: uniform 404 everywhere.
	for _, tc := range []struct {
		m, p string
		body any
	}{
		{http.MethodGet, "/messages", nil}, {http.MethodPost, "/messages", map[string]any{"body": "x"}},
		{http.MethodGet, "/prescription", nil}, {http.MethodPost, "/prescription/1/reminder", nil},
		{http.MethodGet, "/video", nil},
	} {
		r := ce.do(t, tc.m, ce.path(bid, tc.p), tokB, "en", tc.body)
		assert.Equal(t, http.StatusNotFound, r.status, tc.p)
		assert.Equal(t, "booking_not_found", r.body["error_code"], tc.p)
	}
	// An admin without a doctor: 403; another doctor's admin: 404.
	dp := fmt.Sprintf("/telemed/doctor/bookings/%d/messages", bid)
	r := ce.As(admintest.SuperID).Get(dp)
	assert.Equal(t, http.StatusForbidden, r.Status)
	assert.Equal(t, "doctor_required", r.Code())
	ce.Exec(`UPDATE telemed_doctors SET admin_id = ? WHERE id = ?`, admintest.SuperID, ce.other)
	assert.Equal(t, http.StatusNotFound, ce.As(admintest.SuperID).Get(dp).Status)
	assert.Equal(t, http.StatusNotFound, ce.As(admintest.SuperID).JSON(http.MethodPost, dp, map[string]any{"body": "x"}).Status)
	assert.Empty(t, ce.As(admintest.SuperID).Get("/telemed/doctor/bookings").Items())

	// The user writes; the stored body is ciphertext bound to booking + sender.
	const secret = "سلام دکتر، نتیجه آزمایش فریتین ۹ بود"
	s := ce.do(t, http.MethodPost, ce.path(bid, "/messages"), tokA, "fa", map[string]any{"body": secret})
	require.Equal(t, http.StatusCreated, s.status, s.body)
	assert.Equal(t, secret, s.obj("message")["body"])
	stored := ce.String(`SELECT body FROM telemed_messages WHERE booking_id = ?`, bid)
	assert.True(t, strings.HasPrefix(stored, "v1:"))
	assert.NotContains(t, stored, "فریتین")
	plain, err := ce.box.OpenWith(stored, []byte(fmt.Sprintf("telemed-message:v1:%d:user", bid)))
	require.NoError(t, err)
	assert.Equal(t, secret, plain)
	_, err = ce.box.OpenWith(stored, []byte(fmt.Sprintf("telemed-message:v1:%d:user", bid+1)))
	require.ErrorIs(t, err, loss.ErrNoteUnreadable, "copied to another booking")

	// The doctor reads (marks read) and answers.
	dr := ce.As(doctorAdminID).Get(fmt.Sprintf("/telemed/doctor/bookings/%d/messages", bid))
	require.Equal(t, http.StatusOK, dr.Status, dr.Body)
	require.Len(t, dr.Items(), 1)
	m := dr.Items()[0].(map[string]any)
	assert.Equal(t, secret, m["body"])
	assert.Equal(t, false, m["mine"])
	assert.Equal(t, true, dr.Obj("chat")["open"])
	reply := ce.As(doctorAdminID).JSON(http.MethodPost, fmt.Sprintf("/telemed/doctor/bookings/%d/messages", bid), map[string]any{"body": "سلام، در ویزیت بررسی میکنیم"})
	require.Equal(t, http.StatusCreated, reply.Status, reply.Body)
	assert.Len(t, ce.As(doctorAdminID).Get("/telemed/doctor/bookings").Items(), 1)

	u := ce.do(t, http.MethodGet, ce.path(bid, "/messages"), tokA, "fa", nil)
	require.Equal(t, http.StatusOK, u.status)
	require.Len(t, u.items(), 2)
	assert.NotNil(t, u.items()[0].(map[string]any)["read_at"], "read by the doctor")
	assert.Equal(t, "doctor", u.items()[1].(map[string]any)["sender"])
	after := ce.do(t, http.MethodGet, ce.path(bid, fmt.Sprintf("/messages?after_id=%d", id(u.items()[0].(map[string]any)["id"]))), tokA, "fa", nil)
	assert.Len(t, after.items(), 1)

	// Validation.
	v := ce.do(t, http.MethodPost, ce.path(bid, "/messages"), tokA, "en", map[string]any{})
	assert.Equal(t, http.StatusUnprocessableEntity, v.status)
	v = ce.do(t, http.MethodPost, ce.path(bid, "/messages"), tokA, "en", map[string]any{"body": strings.Repeat("a", telemed.MaxMessageLen+1)})
	assert.Equal(t, http.StatusUnprocessableEntity, v.status)
}

func TestChat_Attachments(t *testing.T) {
	ce := setupChat(t)
	userA, tokA := ce.user(t, "A", "09120000001")
	userB, _ := ce.user(t, "B", "09120000002")
	bid := ce.pay(t, tokA, "16:00", nil)
	ctx := context.Background()
	mine, err := ce.files.Put(ctx, userA, files.PurposeTelemedAttachment, chatPDF, fixed)
	require.NoError(t, err)
	record, err := ce.files.Put(ctx, userA, files.PurposeRecordDocument, chatPDF, fixed)
	require.NoError(t, err)
	foreign, err := ce.files.Put(ctx, userB, files.PurposeTelemedAttachment, chatPDF, fixed)
	require.NoError(t, err)

	for _, fid := range []uint64{record.ID, foreign.ID} {
		r := ce.do(t, http.MethodPost, ce.path(bid, "/messages"), tokA, "en", map[string]any{"file_id": fid})
		assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.body)
	}
	r := ce.do(t, http.MethodPost, ce.path(bid, "/messages"), tokA, "en", map[string]any{"file_id": mine.ID})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	att, _ := r.obj("message")["attachment"].(map[string]any)
	require.NotNil(t, att)
	assert.Equal(t, "application/pdf", att["mime"])
	assert.Contains(t, att["url"], "/api/v1/files/")

	dr := ce.As(doctorAdminID).Get(fmt.Sprintf("/telemed/doctor/bookings/%d/messages", bid))
	require.Len(t, dr.Items(), 1)
	datt, _ := dr.Items()[0].(map[string]any)["attachment"].(map[string]any)
	require.NotNil(t, datt, "the booked doctor gets a signed link")
}

func TestChat_VideoWindow(t *testing.T) {
	ce := setupChat(t)
	_, tok := ce.user(t, "A", "09120000001")
	bid := ce.pay(t, tok, "16:00", nil)
	for _, tc := range []struct {
		at     time.Time
		active bool
	}{
		{ce.at(15, 49, 59), false}, {ce.at(15, 50, 0), true}, {ce.at(16, 19, 59), true}, {ce.at(16, 20, 0), false},
	} {
		ce.clk.set(tc.at)
		r := ce.do(t, http.MethodGet, ce.path(bid, "/video"), tok, "fa", nil)
		require.Equal(t, http.StatusOK, r.status, r.body)
		v := r.obj("video")
		assert.Equal(t, tc.active, v["active"], tc.at)
		if tc.active {
			assert.Equal(t, telemed.FakeVideoBase+strings.ToLower(ce.String(`SELECT reference FROM telemed_bookings WHERE id = ?`, bid)), v["url"])
		} else {
			assert.Nil(t, v["url"], tc.at)
		}
	}
	d := ce.As(doctorAdminID).Get(fmt.Sprintf("/telemed/doctor/bookings/%d/video", bid))
	require.Equal(t, http.StatusOK, d.Status, d.Body)
	assert.NotNil(t, d.Obj("video"))
}

func TestChat_PrescriptionToReminder(t *testing.T) {
	ce := setupChat(t)
	userA, tok := ce.user(t, "A", "09120000001")
	bid := ce.pay(t, tok, "16:00", nil)
	dp := fmt.Sprintf("/telemed/doctor/bookings/%d/prescription", bid)
	doc := ce.As(doctorAdminID)

	bad := doc.JSON(http.MethodPost, dp, map[string]any{"drug": "Folic acid", "form": "powder", "times": []any{"25:00"}})
	assert.Equal(t, http.StatusUnprocessableEntity, bad.Status)
	assert.Contains(t, bad.Errors(), "form")
	assert.Contains(t, bad.Errors(), "times.0")
	r := doc.JSON(http.MethodPost, dp, map[string]any{"drug": "Folic acid", "dose": "400", "form": "tablet",
		"times": []any{"20:00", "08:00"}, "duration_days": 30, "instructions": "after food"})
	require.Equal(t, http.StatusCreated, r.Status, r.Body)
	itemID := id(r.Obj("item")["id"])

	l := ce.do(t, http.MethodGet, ce.path(bid, "/prescription"), tok, "fa", nil)
	require.Len(t, l.items(), 1)
	add := ce.do(t, http.MethodPost, ce.path(bid, fmt.Sprintf("/prescription/%d/reminder", itemID)), tok, "fa", nil)
	require.Equal(t, http.StatusCreated, add.status, add.body)
	medID := id(add.data()["medication_id"])
	assert.Equal(t, float64(medID), add.obj("item")["reminder_id"])
	assert.Equal(t, 1, ce.Int(`SELECT COUNT(*) FROM reminders WHERE id = ? AND user_id = ? AND type = 'medication' AND title = 'Folic acid'
		AND starts_on = '2030-01-08' AND ends_on = '2030-02-06' AND recurrence_time = '08:00:00'`, medID, userA))

	again := ce.do(t, http.MethodPost, ce.path(bid, fmt.Sprintf("/prescription/%d/reminder", itemID)), tok, "fa", nil)
	assert.Equal(t, http.StatusConflict, again.status)
	assert.Equal(t, "prescription_already_added", again.body["error_code"])
	missing := ce.do(t, http.MethodPost, ce.path(bid, "/prescription/999999/reminder"), tok, "fa", nil)
	assert.Equal(t, http.StatusNotFound, missing.status)

	del := doc.JSON(http.MethodDelete, fmt.Sprintf("%s/%d", dp, itemID), nil)
	assert.Equal(t, http.StatusOK, del.Status, del.Body)
	assert.Equal(t, 1, ce.Int(`SELECT COUNT(*) FROM reminders WHERE id = ?`, medID), "the reminder stays")
}

func TestChat_ClosedAfterWindow(t *testing.T) {
	ce := setupChat(t)
	_, tok := ce.user(t, "A", "09120000001")
	bid := ce.pay(t, tok, "16:00", nil)
	ce.clk.set(time.Date(2030, 1, 11, 16, 20, 1, 0, fixed.Location()))
	r := ce.do(t, http.MethodPost, ce.path(bid, "/messages"), tok, "fa", map[string]any{"body": "x"})
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "chat_closed", r.body["error_code"])
	g := ce.do(t, http.MethodGet, ce.path(bid, "/messages"), tok, "fa", nil)
	assert.Equal(t, http.StatusOK, g.status)
	assert.Equal(t, false, g.obj("chat")["open"])
}
