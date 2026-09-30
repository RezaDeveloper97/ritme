package profile_test

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/profile"
)

// privacyEnv is the shared harness (setup mounts the B-N1-12 routes) and its storage directory.
func privacyEnv(t *testing.T) (*env, string) {
	t.Helper()
	e := setup(t)
	return e, e.storage
}

func consentByCode(t *testing.T, r response, code string) map[string]any {
	t.Helper()
	data, ok := r.body["data"].(map[string]any)
	require.True(t, ok, r.raw)
	for _, c := range data["consents"].([]any) {
		m := c.(map[string]any)
		if m["code"] == code {
			return m
		}
	}
	t.Fatalf("consent %s missing: %s", code, r.raw)
	return nil
}

func TestConsents_DefaultsGrantRevokeAndIsolation(t *testing.T) {
	e, _ := privacyEnv(t)
	_, alice := e.user(t, "09120000001")
	_, bob := e.user(t, "09120000002")

	r := e.do(t, "GET", "/api/v1/profile/consents", alice, "")
	require.Equal(t, 200, r.status, r.raw)
	for _, code := range profile.ConsentCodes {
		c := consentByCode(t, r, code)
		assert.Equal(t, false, c["granted"], code)
		assert.Nil(t, c["granted_at"], code)
	}

	r = e.do(t, "PUT", "/api/v1/profile/consents", alice, `{"consents":{"ai_lab_analysis":true,"anonymous_stats":false,"unknown":true}}`)
	require.Equal(t, 200, r.status, r.raw)
	ai := consentByCode(t, r, "ai_lab_analysis")
	assert.Equal(t, true, ai["granted"])
	assert.Equal(t, "2026-09-23T10:00:00+03:30", ai["granted_at"])
	assert.Nil(t, ai["revoked_at"])
	stats := consentByCode(t, r, "anonymous_stats")
	assert.Equal(t, false, stats["granted"])
	assert.Nil(t, stats["revoked_at"], "a «no» that was never a «yes» has no withdrawal time")

	r = e.do(t, "PUT", "/api/v1/profile/consents", alice, `{"consents":{"ai_lab_analysis":false}}`)
	require.Equal(t, 200, r.status, r.raw)
	ai = consentByCode(t, r, "ai_lab_analysis")
	assert.Equal(t, false, ai["granted"])
	assert.Equal(t, "2026-09-23T10:00:00+03:30", ai["granted_at"], "the last grant is kept")
	assert.Equal(t, "2026-09-23T10:00:00+03:30", ai["revoked_at"])

	// Bob sees none of Alice's answers.
	r = e.do(t, "GET", "/api/v1/profile/consents", bob, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Nil(t, consentByCode(t, r, "ai_lab_analysis")["granted_at"])
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM user_consents WHERE consent = 'unknown'`).Scan(&n))
	assert.Zero(t, n, "unknown codes are ignored")
}

func TestConsents_ValidationAnd401(t *testing.T) {
	e, _ := privacyEnv(t)
	_, tok := e.user(t, "09120000003")

	r := e.do(t, "PUT", "/api/v1/profile/consents", tok, `{"consents":{"assistant_profile":"maybe"}}`)
	assert.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Contains(t, r.body["errors"], "consents.assistant_profile")

	r = e.do(t, "PUT", "/api/v1/profile/consents", tok, `{}`)
	assert.Equal(t, 422, r.status, r.raw)

	r = e.do(t, "GET", "/api/v1/profile/consents", "nope", "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 40, 80))
	for x := range 40 {
		for y := range 80 {
			img.Set(x, y, color.NRGBA{R: 120, G: 80, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func (e *env) multipart(t *testing.T, token string, fields map[string]string, file []byte, fileName string) response {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	if file != nil {
		fw, err := w.CreateFormFile("screenshot", fileName)
		require.NoError(t, err)
		_, err = fw.Write(file)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/support/reports", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "vitest-browser")
	req.Header.Set(clock.Header, now)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	r := response{status: resp.StatusCode, raw: string(raw)}
	_ = json.Unmarshal(raw, &r.body)
	return r
}

func TestSupportReport_StoresPrivateScreenshotAndCleansUpWithAccount(t *testing.T) {
	e, storage := privacyEnv(t)
	uid, tok := e.user(t, "09120000004")

	r := e.multipart(t, tok, map[string]string{"message": "پیش بینی پریودم دقیق نیست", "app_version": "1.1.0"}, pngBytes(t), "shot.png")
	require.Equal(t, 201, r.status, r.raw)
	data := r.body["data"].(map[string]any)
	assert.Equal(t, "open", data["status"])

	var path, ua string
	require.NoError(t, e.db.QueryRow(`SELECT screenshot_path, user_agent FROM support_reports WHERE user_id = ?`, uid).Scan(&path, &ua))
	assert.Equal(t, "vitest-browser", ua)
	assert.Regexp(t, `^app/private/support-reports/[A-Za-z0-9]{40}\.webp$`, path)
	abs := filepath.Join(storage, filepath.FromSlash(path))
	st, err := os.Stat(abs)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), st.Mode().Perm())

	r = e.do(t, "DELETE", "/api/v1/account", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	_, err = os.Stat(abs)
	assert.True(t, os.IsNotExist(err), "the screenshot goes with the account")
}

func TestSupportReport_ValidationAndFloodGuard(t *testing.T) {
	e, _ := privacyEnv(t)
	_, tok := e.user(t, "09120000005")

	r := e.multipart(t, tok, map[string]string{"message": "short"}, []byte("<svg onload=alert(1)>"), "x.png")
	require.Equal(t, 422, r.status, r.raw)
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "message")
	assert.Contains(t, errs, "screenshot", "a renamed SVG is not an image")

	r = e.do(t, "POST", "/api/v1/support/reports", tok, `{}`)
	assert.Equal(t, 422, r.status, r.raw)

	for i := range profile.SupportReportsPerHour {
		r = e.do(t, "POST", "/api/v1/support/reports", tok, `{"message":"گزارش شماره چند برای تست"}`)
		require.Equal(t, 201, r.status, "report %d: %s", i, r.raw)
	}
	r = e.do(t, "POST", "/api/v1/support/reports", tok, `{"message":"گزارش شماره چند برای تست"}`)
	assert.Equal(t, 429, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])

	r = e.do(t, "POST", "/api/v1/support/reports", "nope", `{"message":"گزارش شماره چند برای تست"}`)
	assert.Equal(t, 401, r.status)
}

func TestSupportReport_DataURLScreenshot(t *testing.T) {
	e, _ := privacyEnv(t)
	uid, tok := e.user(t, "09120000006")
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes(t))
	r := e.do(t, "POST", "/api/v1/support/reports", tok, `{"message":"تقویم بعد از آپدیت باز نمی شود","screenshot":"`+url+`"}`)
	require.Equal(t, 201, r.status, r.raw)
	var path string
	require.NoError(t, e.db.QueryRow(`SELECT screenshot_path FROM support_reports WHERE user_id = ?`, uid).Scan(&path))
	_, err := os.Stat(filepath.Join(e.storage, filepath.FromSlash(path)))
	require.NoError(t, err)

	svg := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("<svg onload=alert(1)>"))
	r = e.do(t, "POST", "/api/v1/support/reports", tok, `{"message":"تقویم بعد از آپدیت باز نمی شود","screenshot":"`+svg+`"}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "screenshot")
}

// A PNG whose header passes inspection but whose pixel data is cut off: decoding fails → 422, not 500.
func TestSupportReport_UndecodableAndHugeScreenshotsAre422(t *testing.T) {
	e, _ := privacyEnv(t)
	_, tok := e.user(t, "09120000007")
	good := pngBytes(t)
	truncated := good[:len(good)-30]
	r := e.multipart(t, tok, map[string]string{"message": "تقویم بعد از آپدیت باز نمی شود"}, truncated, "cut.png")
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "screenshot")

	// 5000×5000 header (25 Mpx) — refused from the header, before any decode.
	huge := append([]byte{}, good...)
	binary.BigEndian.PutUint32(huge[16:20], 5000)
	binary.BigEndian.PutUint32(huge[20:24], 5000)
	r = e.multipart(t, tok, map[string]string{"message": "تقویم بعد از آپدیت باز نمی شود"}, huge, "huge.png")
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "screenshot")
}

func TestExport_IncludesConsentsReportsAndNotificationSettings(t *testing.T) {
	e, _ := privacyEnv(t)
	_, tok := e.user(t, "09120000008")
	require.Equal(t, 200, e.do(t, "PUT", "/api/v1/profile/consents", tok, `{"consents":{"anonymous_stats":true}}`).status)
	r := e.multipart(t, tok, map[string]string{"message": "تقویم بعد از آپدیت باز نمی شود"}, pngBytes(t), "shot.png")
	require.Equal(t, 201, r.status, r.raw)

	r = e.do(t, "GET", "/api/v1/profile/export", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	data := r.body["data"].(map[string]any)
	consents := data["consents"].([]any)
	require.Len(t, consents, 1)
	assert.Equal(t, "anonymous_stats", consents[0].(map[string]any)["code"])
	reports := data["support_reports"].([]any)
	require.Len(t, reports, 1)
	report := reports[0].(map[string]any)
	assert.Equal(t, true, report["has_screenshot"])
	assert.NotContains(t, report, "screenshot_path")
	assert.NotContains(t, r.raw, "support-reports/", "no file path leaves the server")
	assert.Contains(t, data["notification_settings"], "quiet_hours")
}
