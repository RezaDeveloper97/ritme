package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/clock"
)

// CB-REC-03: record sharing (24h doctor code + QR, access log, family scope) and the emergency card, end to end
// through the real router (Mount), MariaDB and miniredis.

var shareCodeRe = regexp.MustCompile(`^[2-9A-HJKMNP-TV-Z]{4}-[2-9A-HJKMNP-TV-Z]{4}$`)

const iPhoneSafari = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1"

// doUA is do with a User-Agent header.
func (e *companionEnv) doUA(method, path, token, ua string, body any, now ...string) reply {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(e.t, err)
		rd = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("User-Agent", ua)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	at := companionNow
	if len(now) > 0 {
		at = now[0]
	}
	req.Header.Set(clock.Header, at)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	out := reply{Status: resp.StatusCode, Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

// recordFile uploads a tiny PNG as a record_document file of token and returns its id.
func (e *companionEnv) recordFile(token string) uint64 {
	e.t.Helper()
	// The blob lands under STORAGE_PATH (the committed key fixtures directory): remove it afterwards.
	e.t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(keysDir, "app")) })
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 200, A: 255})
	var pic bytes.Buffer
	require.NoError(e.t, png.Encode(&pic, img))
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	require.NoError(e.t, w.WriteField("purpose", "record_document"))
	part, err := w.CreateFormFile("file", "scan.png")
	require.NoError(e.t, err)
	_, err = part.Write(pic.Bytes())
	require.NoError(e.t, err)
	require.NoError(e.t, w.Close())
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/files", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, companionNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(e.t, fiber.StatusCreated, resp.StatusCode, string(raw))
	var out map[string]any
	require.NoError(e.t, json.Unmarshal(raw, &out))
	return idOf(out["data"].(map[string]any)["id"])
}

// recordDocument creates a record document of token (with the given files) and returns its id.
func (e *companionEnv) recordDocument(token, title string, fileIDs ...uint64) uint64 {
	e.t.Helper()
	ids := []uint64{}
	ids = append(ids, fileIDs...)
	r := e.do(fiber.MethodPost, "/api/v1/health-record/documents", token, map[string]any{
		"kind": "imaging", "title": title, "date": "2026-09-20", "centre": "Centre", "note": "private note",
		"file_ids": ids})
	require.Equal(e.t, fiber.StatusCreated, r.Status, r.Raw)
	return idOf(r.data()["id"])
}

func persianDigits(s string) string {
	return strings.NewReplacer("2", "۲", "3", "۳", "4", "۴", "5", "۵", "6", "۶", "7", "۷", "8", "۸", "9", "۹").Replace(s)
}

func TestRecordSharing_DoctorCodeFlow(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009001")
	nina := e.user("Nina", "09120009002")
	fileID := e.recordFile(sara)
	docID := e.recordDocument(sara, "Ultrasound", fileID)
	ninaDoc := e.recordDocument(nina, "Nina scan")

	const codes = "/api/v1/health-record/share-codes"
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, codes, "", nil).Status)

	r := e.do(fiber.MethodPost, codes, sara, map[string]any{"document_ids": []uint64{ninaDoc}})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "document_ids.0", "another user's document is refused")
	r = e.do(fiber.MethodPost, codes, sara, map[string]any{"label": strings.Repeat("x", 61)})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)

	r = e.do(fiber.MethodPost, codes, sara, map[string]any{"label": "Dr Ahmadi", "document_ids": []uint64{docID}})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	d := r.data()
	code, token := d["code"].(string), d["token"].(string)
	id := idOf(d["id"])
	assert.Regexp(t, shareCodeRe, code)
	assert.Len(t, token, 43)
	assert.Equal(t, "/en/shared/report/"+token, d["qr_path"])
	assert.Nil(t, d["qr_url"], "no SHARE_WEB_URL in tests")
	assert.Equal(t, "summary", d["kind"])
	assert.Equal(t, "active", d["status"])
	assert.Equal(t, float64(1), d["documents_count"])
	assert.Equal(t, "2026-09-24T10:00:00+03:30", d["expires_at"])

	// At rest: no code, no token, no report in clear.
	var codeHash, codePayload, payload string
	require.NoError(t, e.db.QueryRow(`SELECT code_hash, code_payload, payload FROM health_share_links WHERE id = ?`, id).
		Scan(&codeHash, &codePayload, &payload))
	plain := strings.ReplaceAll(code, "-", "")
	for _, s := range []string{codeHash, codePayload, payload} {
		assert.NotContains(t, s, plain)
		assert.NotContains(t, s, token)
		assert.NotContains(t, s, "Ultrasound")
	}

	// The QR link (token) opens the summary with fresh file links; no ids, no note, no label.
	r = e.doUA(fiber.MethodGet, "/api/v1/shared-reports/"+token, "", iPhoneSafari, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "summary", r.data()["kind"])
	docs := r.data()["documents"].([]any)
	require.Len(t, docs, 1)
	doc := docs[0].(map[string]any)
	assert.Equal(t, "Ultrasound", doc["title"])
	assert.NotContains(t, doc, "file_ids")
	assert.NotContains(t, doc, "note")
	files := doc["files"].([]any)
	require.Len(t, files, 1)
	assert.Contains(t, files[0].(map[string]any)["url"], "/download")
	assert.NotContains(t, r.Raw, "Dr Ahmadi")
	assert.NotContains(t, r.Raw, "private note")

	// The typed code: Persian digits, lower case, no dash.
	r = e.do(fiber.MethodPost, "/api/v1/shared-reports/code", "", map[string]any{"code": strings.ToLower(persianDigits(plain))})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "summary", r.data()["kind"])

	// Access log: newest first, coarse client only; isolated per owner.
	r = e.do(fiber.MethodGet, "/api/v1/health-record/share-access", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	items := r.data()["items"].([]any)
	require.Len(t, items, 2)
	first, second := items[0].(map[string]any), items[1].(map[string]any)
	assert.Equal(t, "code", first["via"])
	assert.Equal(t, "link", second["via"])
	assert.Equal(t, map[string]any{"device": "mobile", "browser": "safari"}, second["client"])
	assert.Equal(t, "Dr Ahmadi", second["label"])
	assert.Equal(t, float64(id), second["link_id"])
	assert.Empty(t, e.do(fiber.MethodGet, "/api/v1/health-record/share-access", nina, nil).data()["items"])
	assert.Equal(t, fiber.StatusNotFound,
		e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/health-record/share-access?link_id=%d", id), nina, nil).Status)
	assert.Len(t, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/health-record/share-access?link_id=%d", id), sara, nil).
		data()["items"], 2)

	// Overview.
	r = e.do(fiber.MethodGet, "/api/v1/health-record/sharing", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	doctor := r.data()["doctor"].(map[string]any)
	assert.Equal(t, true, doctor["available"])
	assert.Equal(t, float64(1), doctor["active_count"])
	assert.Equal(t, float64(24), doctor["ttl_hours"])
	require.Len(t, doctor["items"], 1)
	assert.NotContains(t, r.Raw, token)
	assert.Len(t, r.data()["access_log"], 2)
	assert.Empty(t, r.data()["family"])
	assert.Empty(t, e.do(fiber.MethodGet, "/api/v1/health-record/sharing", nina, nil).data()["doctor"].(map[string]any)["items"])

	// Bloom's report links do not list or revoke a summary.
	assert.Empty(t, e.do(fiber.MethodGet, "/api/v1/health-record/share-links", sara, nil).data()["items"])
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/health-record/share-links/%d", id), sara, nil).Status)

	// Revoke: another user cannot; the owner can; then token and code are a uniform 404 (never 410).
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, fmt.Sprintf("%s/%d", codes, id), nina, nil).Status)
	r = e.do(fiber.MethodDelete, fmt.Sprintf("%s/%d", codes, id), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "revoked", r.data()["status"])
	r = e.do(fiber.MethodGet, "/api/v1/shared-reports/"+token, "", nil)
	assert.Equal(t, fiber.StatusNotFound, r.Status)
	r = e.do(fiber.MethodPost, "/api/v1/shared-reports/code", "", map[string]any{"code": code})
	assert.Equal(t, fiber.StatusNotFound, r.Status)
	assert.Equal(t, "share_code_not_found", r.Body["error_code"])
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM health_share_links WHERE id = ? AND (payload IS NOT NULL OR code_payload IS NOT NULL)`, id))
}

func TestRecordSharing_ExpiryAndCap(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009101")
	const codes = "/api/v1/health-record/share-codes"
	r := e.do(fiber.MethodPost, codes, sara, map[string]any{"sections": []string{"basics"}, "range": "3m"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	code, token := r.data()["code"].(string), r.data()["token"].(string)

	later := "2026-09-24T10:00:01+03:30" // 24 h + 1 s
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/shared-reports/"+token, "", nil, "2026-09-24T09:59:59+03:30").Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shared-reports/"+token, "", nil, later).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/shared-reports/code", "", map[string]any{"code": code}, later).Status)
	ov := e.do(fiber.MethodGet, "/api/v1/health-record/sharing", sara, nil, later).data()["doctor"].(map[string]any)
	assert.Equal(t, float64(0), ov["active_count"])
	assert.Equal(t, "expired", ov["items"].([]any)[0].(map[string]any)["status"])

	// At most 5 live codes (the expired one does not count).
	for i := range 5 {
		require.Equal(t, fiber.StatusCreated, e.do(fiber.MethodPost, codes, sara, nil, later).Status, i)
	}
	r = e.do(fiber.MethodPost, codes, sara, nil, later)
	assert.Equal(t, fiber.StatusConflict, r.Status)
	assert.Equal(t, "share_code_limit", r.Body["error_code"])
}

func TestRecordSharing_CodeBruteForceGuards(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009201")
	r := e.do(fiber.MethodPost, "/api/v1/health-record/share-codes", sara, map[string]any{"sections": []string{"basics"}})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	code := r.data()["code"].(string)
	lookup := func(c string) reply {
		return e.do(fiber.MethodPost, "/api/v1/shared-reports/code", "", map[string]any{"code": c})
	}

	// A missing code is a 422 (not a guess); a malformed one is the uniform 404.
	assert.Equal(t, fiber.StatusUnprocessableEntity, lookup("").Status)
	e.clearThrottles("share-code")
	assert.Equal(t, fiber.StatusNotFound, lookup("hello").Status)

	// Per IP: ShareCodeLookupsPerIP lookups per window, whatever the result.
	e.clearThrottles("share-code")
	for i := range ShareCodeLookupsPerIP {
		require.Equal(t, fiber.StatusNotFound, lookup("2222-2222").Status, i)
	}
	assert.Equal(t, fiber.StatusTooManyRequests, lookup(code).Status, "even the right code once the IP is over its budget")

	// Global breaker: ShareCodeFailuresPerMinute refusals (any caller) pause every lookup for the cool-down.
	for _, k := range e.mr.Keys() {
		if strings.Contains(k, "share-code-breaker") {
			e.mr.Del(k)
		}
	}
	for i := range ShareCodeFailuresPerMinute {
		if i%ShareCodeLookupsPerIP == 0 {
			e.clearThrottles("share-code")
		}
		require.Equal(t, fiber.StatusNotFound, lookup("3333-3333").Status, i)
	}
	e.clearThrottles("share-code")
	r = lookup(code)
	require.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	assert.Equal(t, "too_many_attempts", r.Body["error_code"])
	assert.Positive(t, r.Body["retry_after"])
	e.mr.FastForward(ShareCodeBreakerCooldown + time.Second)
	e.clearThrottles("share-code")
	assert.Equal(t, fiber.StatusOK, lookup(code).Status, "open again after the cool-down")
}

func TestRecordSharing_FamilyScopeNeverDocuments(t *testing.T) {
	e := newCompanionEnv(t)
	sara, ali, linkID := securityPair(t, e)
	saraID := e.ids["Sara"]
	fileID := e.recordFile(sara)
	docID := e.recordDocument(sara, "Sara scan", fileID)

	// Every record / document / file route reads the caller's own rows; for_user_id is ignored.
	for _, p := range []string{
		fmt.Sprintf("/api/v1/health-record/documents/%d", docID),
		fmt.Sprintf("/api/v1/health-record/documents/%d?for_user_id=%d", docID, saraID),
		fmt.Sprintf("/api/v1/files/%d", fileID),
		fmt.Sprintf("/api/v1/files/%d?for_user_id=%d", fileID, saraID),
	} {
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, p, ali, nil).Status, p)
	}
	r := e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/health-record/timeline?for_user_id=%d", saraID), ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotContains(t, r.Raw, "Sara scan")
	r = e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/health-record?for_user_id=%d", saraID), ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "Ali", r.data()["person"].(map[string]any)["name"])
	for _, p := range []string{"/api/v1/health-record/sharing", "/api/v1/health-record/share-access",
		"/api/v1/health-record/emergency-card"} {
		r = e.do(fiber.MethodGet, fmt.Sprintf("%s?for_user_id=%d", p, saraID), ali, nil)
		require.Equal(t, fiber.StatusOK, r.Status, p)
		assert.NotContains(t, r.Raw, "Sara", p)
	}

	// No grant can name the record documents.
	for _, s := range []string{"documents", "record", "health_record"} {
		r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", linkID), sara,
			map[string]any{"grants": map[string]any{s: "view"}})
		assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, s)
	}

	// The owner's overview: meds & appointments only, documents always false.
	r = e.do(fiber.MethodGet, "/api/v1/health-record/sharing", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	fam := r.data()["family"].([]any)
	require.Len(t, fam, 1)
	assert.Equal(t, map[string]any{"companion_id": float64(linkID), "type": "partner", "status": "active", "name": "Ali",
		"meds": "edit", "appointments": "edit", "documents": false}, fam[0])

	// A teen-mode owner's partner link grants nothing (B-N4-08b), and the overview says so.
	_, err := e.db.Exec(`UPDATE user_life_profiles SET life_mode = 'teen' WHERE user_id = ?`, saraID)
	require.NoError(t, err)
	fam = e.do(fiber.MethodGet, "/api/v1/health-record/sharing", sara, nil).data()["family"].([]any)
	assert.Equal(t, "none", fam[0].(map[string]any)["meds"])
}

func TestEmergencyCard_OwnerAndPublic(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009301")
	nina := e.user("Nina", "09120009302")
	saraID := e.ids["Sara"]
	const p = "/api/v1/health-record/emergency-card"
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, p, "", nil).Status)

	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, "/api/v1/health-record/basics", sara,
		map[string]any{"blood_type": "O+", "allergies": []string{"penicillin"}}).Status)
	require.Equal(t, fiber.StatusCreated, e.do(fiber.MethodPost, "/api/v1/care/medications", sara,
		medBody(map[string]any{"title": "Levothyroxine", "duration": "ongoing"})).Status)
	require.Equal(t, fiber.StatusCreated, e.do(fiber.MethodPost, "/api/v1/care/medications", sara,
		medBody(map[string]any{"title": "Antibiotic", "duration": "until_date", "ends_on": "2026-10-01"})).Status)

	r := e.do(fiber.MethodGet, p, sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	card := r.data()["card"].(map[string]any)
	assert.Equal(t, "Sara", card["name"])
	assert.Equal(t, "O+", card["blood_type"])
	assert.Equal(t, []any{"penicillin"}, card["allergies"])
	meds := card["medications"].([]any)
	require.Len(t, meds, 1, "only permanent (ongoing) medications")
	assert.Equal(t, "Levothyroxine", meds[0].(map[string]any)["title"])
	assert.Nil(t, card["pregnancy"])
	assert.Nil(t, card["emergency_contact"])
	assert.Nil(t, card["insurance"])
	assert.Equal(t, map[string]any{"show_on_lock_screen": false, "show_pregnancy": false, "allergies_on_card": true},
		r.data()["settings"])
	assert.Equal(t, false, r.data()["public_link"].(map[string]any)["enabled"])

	// Validation: a phone that is not one, a full insurance number.
	r = e.do(fiber.MethodPut, p, sara, map[string]any{"emergency_contact": map[string]any{"name": "Ali", "phone": "call me"}})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "emergency_contact.phone")
	r = e.do(fiber.MethodPut, p, sara, map[string]any{"insurance": map[string]any{"label": "x", "last4": "6037991234564821"}})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "insurance.last4")
	r = e.do(fiber.MethodPut, p, sara, map[string]any{"show_on_lock_screen": "maybe"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)

	// Pregnancy shows only with the flag and an active pregnancy (week only).
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-07-29', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, saraID)
	require.NoError(t, err)
	r = e.do(fiber.MethodPut, p, sara, map[string]any{
		"show_on_lock_screen": true, "show_pregnancy": true,
		"emergency_contact": map[string]any{"name": " Ali ", "relation": "همسر", "phone": "۰۹۱۲ ۰۰۰ ۰۰۰۰"},
		"insurance":         map[string]any{"label": "تکمیلی", "last4": "۴۸۲۱"},
	})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	card = r.data()["card"].(map[string]any)
	assert.Equal(t, map[string]any{"week": float64(9)}, card["pregnancy"])
	assert.Equal(t, map[string]any{"name": "Ali", "relation": "همسر", "phone": "09120000000"}, card["emergency_contact"])
	assert.Equal(t, map[string]any{"label": "تکمیلی", "last4": "4821", "masked": "•••• 4821"}, card["insurance"])
	assert.Equal(t, true, r.data()["settings"].(map[string]any)["show_on_lock_screen"])

	// The record's allergies flag hides allergies.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, "/api/v1/health-record/extras", sara,
		map[string]any{"allergies_on_emergency_card": false}).Status)
	assert.Nil(t, e.do(fiber.MethodGet, p, sara, nil).data()["card"].(map[string]any)["allergies"])

	// Public link: off → 404; on → minimal card (no insurance); rotate kills the old token; off → 404.
	const pub = "/api/v1/emergency-cards/"
	fake := strings.Repeat("A", 43)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, pub+fake, "", nil).Status)
	r = e.do(fiber.MethodPost, p+"/public-link", sara, nil)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	tok1 := r.data()["token"].(string)
	assert.Len(t, tok1, 43)
	assert.Equal(t, "/en/emergency/"+tok1, r.data()["path"])
	r = e.do(fiber.MethodGet, pub+tok1, "", nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "Sara", r.data()["name"])
	assert.NotContains(t, r.data(), "insurance")
	assert.NotContains(t, r.Raw, "4821")
	assert.Equal(t, 1, e.count(`SELECT public_view_count FROM emergency_cards WHERE user_id = ?`, saraID))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM emergency_cards WHERE public_token_hash = ?`, tok1), "only the hash is stored")
	r = e.do(fiber.MethodPost, p+"/public-link", sara, nil)
	require.Equal(t, fiber.StatusCreated, r.Status)
	tok2 := r.data()["token"].(string)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, pub+tok1, "", nil).Status)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, pub+tok2, "", nil).Status)
	r = e.do(fiber.MethodDelete, p+"/public-link", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, false, r.data()["public_link"].(map[string]any)["enabled"])
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, pub+tok2, "", nil).Status)

	// Isolation: another user sees only her own (empty) card.
	r = e.do(fiber.MethodGet, p, nina, nil)
	require.Equal(t, fiber.StatusOK, r.Status)
	assert.Equal(t, "Nina", r.data()["card"].(map[string]any)["name"])
	assert.NotContains(t, r.Raw, "Ali")
	assert.NotContains(t, r.Raw, "O+")

	// Public reads are throttled per IP.
	e.clearThrottles("emergency-card")
	for i := range EmergencyCardPublicMax {
		require.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, pub+fake, "", nil).Status, i)
	}
	assert.Equal(t, fiber.StatusTooManyRequests, e.do(fiber.MethodGet, pub+fake, "", nil).Status)
}
