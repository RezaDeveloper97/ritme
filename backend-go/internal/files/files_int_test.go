package files_test

// CB-CORE-05 end to end against MariaDB: upload → link → download round trip per purpose, encryption at rest, the
// 401 / 422 / 503 bodies, quotas (also under concurrency), the IDOR matrix, signed links (expiry, tampering, reuse
// on another user's file), the public route (public purposes only, never a private file) and account deletion.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffe40"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db      *sql.DB
	app     *fiber.App
	iss     *passport.Issuer
	svc     *files.Service
	storage string
}

func setup(t *testing.T, disabled bool, purposes map[string]files.Purpose) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))

	storage := t.TempDir()
	var current []byte
	if !disabled {
		current = files.DevKey("ritme-file-dev-key")
	}
	vault, err := files.NewVault(storage, current, nil, disabled)
	require.NoError(t, err)
	svc := files.NewService(files.Options{DB: db, Vault: vault, BaseURL: "https://api.test", Purposes: purposes})
	h := files.NewHandlers(svc, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet), BodyLimit: 25 << 20})
	app.Use(clock.Middleware(clock.Real{}, true))
	const p = "/api/v1/files"
	app.Post(p, locale, guard, h.Upload)
	app.Get(p+"/public/:id/:name", locale, h.Public)
	app.Get(p+"/:id/download", locale, h.Download)
	app.Get(p+"/:id", locale, guard, h.Show)
	app.Delete(p+"/:id", locale, guard, h.Destroy)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), svc: svc, storage: storage}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Test', ?, NOW(), NOW())`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test ids
}

type resp struct {
	status int
	header http.Header
	raw    []byte
	body   map[string]any
}

func (e *env) do(t *testing.T, method, path, token string, body io.Reader, ctype string, hdr ...string) resp {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Accept", "application/json")
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	out := resp{status: res.StatusCode, header: res.Header, raw: raw}
	_ = json.Unmarshal(raw, &out.body)
	return out
}

func multipartBody(t *testing.T, purpose string, name string, data []byte) (io.Reader, string) {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if purpose != "" {
		require.NoError(t, w.WriteField("purpose", purpose))
	}
	if data != nil {
		fw, err := w.CreateFormFile("file", name)
		require.NoError(t, err)
		_, err = fw.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return &b, w.FormDataContentType()
}

func (e *env) upload(t *testing.T, token, purpose string, data []byte, hdr ...string) resp {
	t.Helper()
	body, ct := multipartBody(t, purpose, "doc.bin", data)
	return e.do(t, http.MethodPost, "/api/v1/files", token, body, ct, hdr...)
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for i := range 16 {
		img.Set(i, i, color.NRGBA{G: 180, A: 255})
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	return b.Bytes()
}

var pdf = []byte("%PDF-1.4\n1 0 obj << /Ferritin 9 >> endobj\n%%EOF")

func data(r resp) map[string]any { d, _ := r.body["data"].(map[string]any); return d }

func pathOf(t *testing.T, u string) string {
	t.Helper()
	p, err := url.Parse(u)
	require.NoError(t, err)
	assert.Equal(t, "api.test", p.Host, "absolute links on APP_URL")
	return p.RequestURI()
}

// Round trip per user-uploadable purpose: 201 view, encrypted on disk, the link serves the same bytes.
func TestUploadRoundTripPerPurpose(t *testing.T) {
	e := setup(t, false, nil)
	uid, tok := e.user(t, "09120000001")
	for _, tc := range []struct {
		purpose string
		in      []byte
		mime    string
		public  bool
	}{
		{files.PurposeRecordDocument, pdf, "application/pdf", false},
		{files.PurposeClaimDocument, pdf, "application/pdf", false},
		{files.PurposeRecordDocument, pngBytes(t), "image/webp", false},
		{files.PurposeClaimDocument, pngBytes(t), "image/webp", false},
		// Not user uploads yet (CB-DIR opens them): written through the Go API, served the same way.
		{files.PurposePlaceLicence, pdf, "application/pdf", false},
		{files.PurposePlaceLicence, pngBytes(t), "image/webp", false},
		{files.PurposePlacePhoto, pngBytes(t), "image/webp", true},
	} {
		var d map[string]any
		if p, _ := files.Lookup(tc.purpose); p.UserUpload {
			r := e.upload(t, tok, tc.purpose, tc.in, "Accept-Language", "en")
			require.Equal(t, 201, r.status, "%s %s", tc.purpose, r.raw)
			assert.Equal(t, true, r.body["success"])
			assert.Equal(t, "File uploaded", r.body["message"])
			assert.NotContains(t, string(r.raw), "app/private", "the storage path never leaves the server")
			d = data(r)
		} else {
			assert.Equal(t, 422, e.upload(t, tok, tc.purpose, tc.in).status, "%s is not a user upload", tc.purpose)
			f, err := e.svc.Put(context.Background(), uid, tc.purpose, tc.in, time.Now())
			require.NoError(t, err, tc.purpose)
			show := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/files/%d", f.ID), tok, nil, "")
			require.Equal(t, 200, show.status)
			d = data(show)
		}
		assert.Equal(t, tc.purpose, d["purpose"])
		assert.Equal(t, tc.mime, d["mime"])
		assert.Len(t, d["sha256"], 64)
		var dl resp
		if tc.public {
			assert.Equal(t, "public", d["visibility"])
			assert.Nil(t, d["url_expires_at"])
			assert.Contains(t, d["url"], "/api/v1/files/public/")
			dl = e.do(t, http.MethodGet, pathOf(t, d["url"].(string)), "", nil, "")
			assert.Equal(t, "public, max-age=3600", dl.header.Get("Cache-Control"))
		} else {
			assert.Equal(t, "private", d["visibility"])
			assert.NotNil(t, d["url_expires_at"])
			dl = e.do(t, http.MethodGet, pathOf(t, d["url"].(string)), "", nil, "")
			assert.Equal(t, "private, no-store", dl.header.Get("Cache-Control"))
			assert.Contains(t, dl.header.Get("Content-Disposition"), "attachment")
		}
		require.Equal(t, 200, dl.status, string(dl.raw))
		assert.Equal(t, tc.mime, dl.header.Get("Content-Type"))
		assert.Equal(t, "nosniff", dl.header.Get("X-Content-Type-Options"))
		if tc.mime == "application/pdf" {
			assert.Equal(t, tc.in, dl.raw)
		} else {
			assert.True(t, bytes.HasPrefix(dl.raw, []byte("RIFF")), "re-encoded to WebP")
		}
	}

	// At rest: nothing readable.
	var rel string
	require.NoError(t, e.db.QueryRow(`SELECT path FROM files WHERE mime = 'application/pdf' LIMIT 1`).Scan(&rel))
	raw, err := os.ReadFile(filepath.Join(e.storage, filepath.FromSlash(rel)))
	require.NoError(t, err)
	assert.False(t, bytes.Contains(raw, []byte("Ferritin")))
	assert.True(t, strings.HasPrefix(rel, "app/private/files/"))
}

// Go callers (admin) may write product images as the platform; users may not.
func TestPlatformProductImage(t *testing.T) {
	e := setup(t, false, nil)
	_, tok := e.user(t, "09120000002")
	r := e.upload(t, tok, files.PurposeProductImage, pngBytes(t))
	assert.Equal(t, 422, r.status, "product images are not user uploads")
	assert.Contains(t, r.body["errors"], "purpose")

	now := time.Now()
	f, err := e.svc.Put(context.Background(), 0, files.PurposeProductImage, pngBytes(t), now)
	require.NoError(t, err)
	assert.Zero(t, f.Owner)
	link, err := e.svc.Link(f, now, 0)
	require.NoError(t, err)
	assert.Nil(t, link.ExpiresAt)
	dl := e.do(t, http.MethodGet, pathOf(t, link.URL), "", nil, "")
	assert.Equal(t, 200, dl.status)

	_, err = e.svc.Put(context.Background(), 0, files.PurposeRecordDocument, pdf, now)
	require.ErrorIs(t, err, files.ErrPurpose, "the platform never owns a private file")
	_, err = e.svc.Put(context.Background(), 5, files.PurposeLab, pdf, now)
	require.ErrorIs(t, err, files.ErrPurpose, "lab sheets go through internal/labs")
	require.NoError(t, e.svc.Delete(context.Background(), 0, f.ID))
	assert.Equal(t, 404, e.do(t, http.MethodGet, pathOf(t, link.URL), "", nil, "").status)
}

func TestValidationAuthAndDisabled(t *testing.T) {
	e := setup(t, false, nil)
	_, tok := e.user(t, "09120000003")

	r := e.upload(t, "", files.PurposeRecordDocument, pdf)
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
	assert.Equal(t, 401, e.do(t, http.MethodGet, "/api/v1/files/1", "", nil, "").status)
	assert.Equal(t, 401, e.do(t, http.MethodDelete, "/api/v1/files/1", "", nil, "").status)

	for _, tc := range []struct {
		purpose string
		data    []byte
		field   string
	}{
		{"", pdf, "purpose"},
		{"x-ray", pdf, "purpose"},
		{files.PurposeLab, pdf, "purpose"},
		{files.PurposeRecordDocument, nil, "file"},
		{files.PurposeRecordDocument, []byte("hello"), "file"},
		{files.PurposePlacePhoto, pngBytes(t), "purpose"},
		{files.PurposePlaceLicence, pdf, "purpose"},
		{files.PurposeRecordDocument, []byte("%PDF-1.4 no trailer"), "file"},
		{files.PurposeRecordDocument, append(pngBytes(t), make([]byte, 10<<20)...), "file"},
	} {
		r := e.upload(t, tok, tc.purpose, tc.data, "Accept-Language", "fa")
		require.Equal(t, 422, r.status, "%s %s", tc.purpose, r.raw)
		assert.Equal(t, false, r.body["success"])
		assert.Contains(t, r.body["errors"], tc.field, string(r.raw))
	}
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&n))
	assert.Zero(t, n)

	off := setup(t, true, nil)
	_, tok2 := off.user(t, "09120000004")
	r = off.upload(t, tok2, files.PurposeRecordDocument, pdf)
	assert.Equal(t, 503, r.status)
	assert.Equal(t, "storage_unavailable", r.body["error_code"])
}

func TestQuota(t *testing.T) {
	small := map[string]files.Purpose{}
	for k, p := range files.Registry {
		small[k] = p
	}
	rec := small[files.PurposeRecordDocument]
	rec.QuotaFiles = 3
	small[files.PurposeRecordDocument] = rec
	claim := small[files.PurposeClaimDocument]
	claim.QuotaBytes = int64(len(pdf) * 2)
	small[files.PurposeClaimDocument] = claim
	e := setup(t, false, small)
	_, tok := e.user(t, "09120000005")
	_, tok2 := e.user(t, "09120000006")

	// Concurrent uploads cannot overshoot the count quota.
	var wg sync.WaitGroup
	codes := make(chan int, 6)
	for range 6 {
		wg.Go(func() { codes <- e.upload(t, tok, files.PurposeRecordDocument, pdf).status })
	}
	wg.Wait()
	close(codes)
	got := map[int]int{}
	for c := range codes {
		got[c]++
	}
	assert.Equal(t, map[int]int{201: 3, 422: 3}, got)
	r := e.upload(t, tok, files.PurposeRecordDocument, pdf, "Accept-Language", "en")
	assert.Equal(t, 422, r.status)
	assert.Contains(t, string(r.raw), "storage limit")
	assert.Equal(t, 201, e.upload(t, tok2, files.PurposeRecordDocument, pdf).status, "quotas are per owner")
	assert.Equal(t, 201, e.upload(t, tok, files.PurposeClaimDocument, pdf).status, "and per purpose")
	assert.Equal(t, 201, e.upload(t, tok, files.PurposeClaimDocument, pdf).status)
	assert.Equal(t, 422, e.upload(t, tok, files.PurposeClaimDocument, pdf).status, "byte quota")

	// Deleting frees the quota.
	var id uint64
	require.NoError(t, e.db.QueryRow(`SELECT id FROM files WHERE purpose = 'claim_document' LIMIT 1`).Scan(&id))
	assert.Equal(t, 200, e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/files/%d", id), tok, nil, "").status)
	assert.Equal(t, 201, e.upload(t, tok, files.PurposeClaimDocument, pdf).status)
}

// IDOR: user B can neither read, link nor delete user A's file; A's signed link cannot open B's file.
func TestIDORAndSignedLinks(t *testing.T) {
	e := setup(t, false, nil)
	uidA, tokA := e.user(t, "09120000007")
	_, tokB := e.user(t, "09120000008")
	a := data(e.upload(t, tokA, files.PurposeRecordDocument, pdf))
	b := data(e.upload(t, tokB, files.PurposeRecordDocument, []byte("%PDF-1.4 other\n%%EOF")))
	aID, bID := uint64(a["id"].(float64)), uint64(b["id"].(float64))

	for _, m := range []string{http.MethodGet, http.MethodDelete} {
		r := e.do(t, m, fmt.Sprintf("/api/v1/files/%d", aID), tokB, nil, "", "Accept-Language", "en")
		assert.Equal(t, 404, r.status, m)
		assert.Equal(t, "file_not_found", r.body["error_code"])
	}
	assert.Equal(t, 404, e.do(t, http.MethodGet, "/api/v1/files/999999", tokA, nil, "").status)
	assert.Equal(t, 404, e.do(t, http.MethodGet, "/api/v1/files/abc", tokA, nil, "").status)

	// Show gives the owner a fresh link.
	show := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/files/%d", aID), tokA, nil, "")
	require.Equal(t, 200, show.status)
	aLink := pathOf(t, data(show)["url"].(string))
	assert.Equal(t, 200, e.do(t, http.MethodGet, aLink, "", nil, "").status)
	assert.Equal(t, 200, e.do(t, http.MethodGet, aLink, tokB, nil, "").status, "the link is the credential, not the bearer")

	// A's link moved onto B's id → 403 link_invalid (B's file never served).
	u, _ := url.Parse(aLink)
	moved := fmt.Sprintf("/api/v1/files/%d/download?%s", bID, u.RawQuery)
	r := e.do(t, http.MethodGet, moved, "", nil, "", "Accept-Language", "en")
	assert.Equal(t, 403, r.status)
	assert.Equal(t, "link_invalid", r.body["error_code"])
	assert.NotContains(t, string(r.raw), "other")
	// Tampered expiry / signature, missing params, unknown id: 403 link_invalid (never 401).
	q := u.Query()
	q.Set("expires", fmt.Sprint(time.Now().Add(10*time.Minute).Unix()))
	for _, p := range []string{
		fmt.Sprintf("/api/v1/files/%d/download?%s", aID, q.Encode()),
		fmt.Sprintf("/api/v1/files/%d/download?expires=%s&signature=AAAA", aID, u.Query().Get("expires")),
		fmt.Sprintf("/api/v1/files/%d/download", aID),
		"/api/v1/files/999999/download?" + u.RawQuery,
	} {
		r := e.do(t, http.MethodGet, p, "", nil, "")
		assert.Equal(t, 403, r.status, p)
		assert.Equal(t, "link_invalid", r.body["error_code"], p)
	}
	// Expired: a frozen clock past the expiry.
	exp := u.Query().Get("expires")
	var expUnix int64
	_, _ = fmt.Sscan(exp, &expUnix)
	later := time.Unix(expUnix+1, 0).In(time.UTC).Format(time.RFC3339)
	r = e.do(t, http.MethodGet, aLink, "", nil, "", clock.Header, later)
	assert.Equal(t, 403, r.status, string(r.raw))
	assert.Equal(t, "link_expired", r.body["error_code"])

	// The public route never serves a private file, even with the right name.
	var rel string
	require.NoError(t, e.db.QueryRow(`SELECT path FROM files WHERE id = ?`, aID).Scan(&rel))
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/files/public/%d/%s", aID, files.Name(rel)), "", nil, "")
	assert.Equal(t, 404, r.status)
	// A row forced public but of a private purpose is still refused.
	_, err := e.db.Exec(`UPDATE files SET visibility = 'public' WHERE id = ?`, aID)
	require.NoError(t, err)
	assert.Equal(t, 404, e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/files/public/%d/%s", aID, files.Name(rel)), "", nil, "").status)

	// A public file needs its random name.
	phf, err := e.svc.Put(context.Background(), uidA, files.PurposePlacePhoto, pngBytes(t), time.Now())
	require.NoError(t, err)
	phl, err := e.svc.Link(phf, time.Now(), 0)
	require.NoError(t, err)
	ph := map[string]any{"id": phf.ID}
	pu := pathOf(t, phl.URL)
	assert.Equal(t, 200, e.do(t, http.MethodGet, pu, "", nil, "").status)
	last := "0"
	if strings.HasSuffix(pu, "0") {
		last = "1"
	}
	assert.Equal(t, 404, e.do(t, http.MethodGet, pu[:len(pu)-1]+last, "", nil, "").status, "another name")
	assert.Equal(t, 404, e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/files/public/%v/x", ph["id"]), "", nil, "").status)

	// Delete: owner only, then link and file are gone.
	assert.Equal(t, 200, e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/files/%d", bID), tokB, nil, "").status)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM files WHERE id = ?`, bID).Scan(&n))
	assert.Zero(t, n)
}

// Account deletion: rows cascade, blobs of every purpose go with RemoveUser (internal/profile calls it).
func TestAccountDeletion(t *testing.T) {
	e := setup(t, false, nil)
	uid, tok := e.user(t, "09120000009")
	_, tok2 := e.user(t, "09120000010")
	require.Equal(t, 201, e.upload(t, tok, files.PurposeRecordDocument, pdf).status)
	_, err := e.svc.Put(context.Background(), uid, files.PurposePlacePhoto, pngBytes(t), time.Now())
	require.NoError(t, err)
	require.Equal(t, 201, e.upload(t, tok2, files.PurposeRecordDocument, pdf).status)

	svc := &profile.Service{DB: e.db, Q: profilestore.New(e.db), StoragePath: e.storage, Logger: quiet}
	require.NoError(t, svc.DeleteAccount(context.Background(), uid))
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM files WHERE user_id = ?`, uid).Scan(&n))
	assert.Zero(t, n)
	for _, p := range []string{files.PurposeRecordDocument, files.PurposePlacePhoto} {
		owners, err := files.Owners(e.storage, files.Registry[p])
		require.NoError(t, err)
		assert.NotContains(t, owners, uid, p)
	}
	owners, err := files.Owners(e.storage, files.Registry[files.PurposeRecordDocument])
	require.NoError(t, err)
	assert.Len(t, owners, 1, "the other user's files stay")
}

// The orphan sweep removes old blobs without a row (any owner, platform included) and keeps everything else.
func TestSweep(t *testing.T) {
	e := setup(t, false, nil)
	_, tok := e.user(t, "09120000011")
	kept := data(e.upload(t, tok, files.PurposeRecordDocument, pdf))
	gone := data(e.upload(t, tok, files.PurposeClaimDocument, pdf))
	pf, err := e.svc.Put(context.Background(), 0, files.PurposeProductImage, pngBytes(t), time.Now())
	require.NoError(t, err)
	var keptPath, gonePath string
	require.NoError(t, e.db.QueryRow(`SELECT path FROM files WHERE id = ?`, kept["id"]).Scan(&keptPath))
	require.NoError(t, e.db.QueryRow(`SELECT path FROM files WHERE id = ?`, gone["id"]).Scan(&gonePath))
	// Rows vanish without the blobs (a failed removal / another deletion path).
	_, err = e.db.Exec(`DELETE FROM files WHERE id IN (?, ?)`, gone["id"], pf.ID)
	require.NoError(t, err)

	n, err := e.svc.Sweep(context.Background(), time.Now(), files.SweepGrace)
	require.NoError(t, err)
	assert.Zero(t, n, "young blobs are left alone (an upload may be in flight)")
	n, err = e.svc.Sweep(context.Background(), time.Now().Add(2*time.Hour), files.SweepGrace)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	_, err = os.Stat(filepath.Join(e.storage, filepath.FromSlash(gonePath)))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(e.storage, filepath.FromSlash(pf.Path)))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(e.storage, filepath.FromSlash(keptPath)))
	require.NoError(t, err, "a referenced blob stays")
}
