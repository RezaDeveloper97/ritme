package media_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/media"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type mutClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *mutClock) Now() time.Time  { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *mutClock) Set(t time.Time) { c.mu.Lock(); defer c.mu.Unlock(); c.t = t }

var t0 = time.Date(2026, 10, 7, 10, 0, 0, 0, civildate.Tehran)

type scanner struct{ err error }

func (s *scanner) Scan(context.Context, string, string, string) error { return s.err }

type env struct {
	db    *sql.DB
	app   *fiber.App
	iss   *passport.Issuer
	lsvc  *learning.Service
	svc   *media.Service
	clock *mutClock
	root  string
	scan  *scanner
}

func setup(t *testing.T, signer *media.Signer) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	iss := passport.NewIssuer(key, q, clock.Real{}, 365)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	clk := &mutClock{t: t0}
	lsvc := learning.NewService(learning.Options{DB: db, Logger: quiet})
	lh := learning.NewHandlers(lsvc, clk)
	root := t.TempDir()
	scan := &scanner{}
	svc := media.NewService(media.Options{DB: db, Disk: media.NewDisk(root), Signer: signer, Scanner: scan,
		BaseURL: "https://api.example.test", Logger: quiet})
	h := media.NewHandlers(svc, lsvc, clk)

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet), BodyLimit: 25 << 20})
	p := "/api/instructor/v1"
	ins := lh.RequireInstructor
	app.Post(p+"/apply", locale, guard, lh.Apply)
	app.Post(p+"/courses", locale, guard, ins, lh.StoreCourse)
	app.Post(p+"/courses/:id/lessons", locale, guard, ins, lh.StoreLesson)
	app.Delete(p+"/courses/:id/lessons/:lesson", locale, guard, ins, lh.DestroyLesson)
	app.Post(p+"/grants", locale, guard, ins, lh.StoreGrants)
	app.Get("/api/v1/learning/lessons/:id", locale, guard, lh.ShowLesson)

	app.Get("/api/v1/learning/lessons/:id/playback", locale, guard, h.LessonPlayback)
	app.Get("/api/v1/media/:id/stream", locale, h.Stream)
	app.Post(p+"/media", locale, guard, ins, h.Create)
	app.Get(p+"/media/:id", locale, guard, ins, h.Show)
	app.Patch(p+"/media/:id", locale, guard, ins, h.Patch)
	app.Delete(p+"/media/:id", locale, guard, ins, h.Destroy)
	app.Get(p+"/media/:id/playback", locale, guard, ins, h.InstructorPlayback)
	return &env{db: db, app: app, iss: iss, lsvc: lsvc, svc: svc, clock: clk, root: root, scan: scan}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES (?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, "User "+mobile[7:], mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

func (e *env) instructor(t *testing.T, mobile string, approve bool) string {
	t.Helper()
	id, tok := e.user(t, mobile)
	r := e.do(t, http.MethodPost, "/api/instructor/v1/apply", tok, map[string]any{"display_name": "Leila " + mobile[7:]})
	require.Equal(t, 201, r.status, r.raw)
	if approve {
		ins, ok, err := e.lsvc.InstructorByUser(context.Background(), id)
		require.NoError(t, err)
		require.True(t, ok)
		require.NoError(t, e.lsvc.SetInstructorStatus(context.Background(), ins.ID, learning.InstructorApproved, 1, t0))
	}
	return tok
}

type response struct {
	status int
	body   map[string]any
	raw    string
	header http.Header
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (r response) media() map[string]any {
	m, _ := r.data()["media"].(map[string]any)
	return m
}

func (e *env) send(t *testing.T, req *http.Request, token string) response {
	t.Helper()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw), header: resp.Header}
}

func (e *env) do(t *testing.T, method, path, token string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return e.send(t, req, token)
}

func (e *env) chunk(t *testing.T, token string, id int, offset string, data []byte) response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/instructor/v1/media/%d", id), bytes.NewReader(data))
	req.Header.Set("Content-Type", media.ChunkContentType)
	if offset != "" {
		req.Header.Set(media.HeaderUploadOffset, offset)
	}
	return e.send(t, req, token)
}

func (e *env) get(t *testing.T, rawURL, rangeHeader string) response {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, u.RequestURI(), nil)
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	return e.send(t, req, "")
}

func idOf(t *testing.T, v any) int {
	t.Helper()
	f, ok := v.(float64)
	require.True(t, ok, "id %v", v)
	return int(f)
}

// lesson creates a published standalone item with one lesson of kind and returns (course id, lesson id).
func (e *env) lesson(t *testing.T, tok, kind string) (int, int) {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/instructor/v1/courses", tok, map[string]any{"kind": "course", "title": "Birth prep", "status": "published"})
	require.Equal(t, 201, r.status, r.raw)
	cid := idOf(t, r.data()["course"].(map[string]any)["id"])
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons", cid), tok, map[string]any{
		"kind": kind, "title": "Lesson 1", "status": "published",
	})
	require.Equal(t, 201, r.status, r.raw)
	return cid, idOf(t, r.data()["lesson"].(map[string]any)["id"])
}

func (e *env) grant(t *testing.T, tok string, course int, phone string) {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/instructor/v1/grants", tok, map[string]any{
		"phones": []string{phone}, "scope": "course", "target_id": course, "duration": "unlimited",
	})
	require.Equal(t, 201, r.status, r.raw)
}

// video is an mp4-looking payload of n bytes (ftyp box, then deterministic filler).
func video(n int) []byte {
	b := make([]byte, n)
	copy(b, []byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 2, 0})
	for i := 16; i < n; i++ {
		b[i] = byte(i * 7 % 251)
	}
	return b
}

func (e *env) create(t *testing.T, tok string, lesson, size int) int {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/instructor/v1/media", tok, map[string]any{"lesson_id": lesson, "size": size})
	require.Equal(t, 201, r.status, r.raw)
	return idOf(t, r.media()["id"])
}

func (e *env) lessonMedia(t *testing.T, lesson int) (sql.NullInt64, string) {
	t.Helper()
	var id sql.NullInt64
	var status string
	require.NoError(t, e.db.QueryRow("SELECT media_id, media_status FROM learning_lessons WHERE id = ?", lesson).Scan(&id, &status))
	return id, status
}

func (e *env) files(t *testing.T) []string {
	t.Helper()
	var out []string
	require.NoError(t, filepath.WalkDir(e.root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(e.root, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return err
	}))
	return out
}

// Acceptance: an upload resumes after an interruption, then a student with access plays it through a signed URL
// with byte ranges; an unauthorised / tampered URL is 403, an expired one 410.
func TestResumableUploadAndProtectedStreaming(t *testing.T) {
	e := setup(t, media.NewSigner(media.DevKey()))
	itok := e.instructor(t, "09120000001", true)
	cid, lid := e.lesson(t, itok, "video")
	studentID, stok := e.user(t, "09120000050")
	e.grant(t, itok, cid, "09120000050")

	payload := video(20_000)
	r := e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{"lesson_id": lid, "size": len(payload), "mime": "video/mp4"})
	require.Equal(t, 201, r.status, r.raw)
	m := r.media()
	id := idOf(t, m["id"])
	assert.Equal(t, "uploading", m["status"])
	assert.Equal(t, float64(0), m["offset_bytes"])
	assert.Nil(t, m["mime"], "the type comes from the bytes, not the declaration")
	assert.Equal(t, "0", r.header.Get("Upload-Offset"))
	assert.Equal(t, strconv.Itoa(len(payload)), r.header.Get("Upload-Length"))
	assert.Equal(t, float64(media.ChunkSize), r.data()["upload"].(map[string]any)["chunk_size"])
	_, status := e.lessonMedia(t, lid)
	assert.Equal(t, "uploading", status)

	// First chunk.
	r = e.chunk(t, itok, id, "0", payload[:8000])
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, float64(8000), r.media()["offset_bytes"])
	assert.Equal(t, "video/mp4", r.media()["mime"])
	assert.Equal(t, "8000", r.header.Get("Upload-Offset"))

	// Interruption: the client lost the response and retries from 0 → 409 with the server offset.
	r = e.chunk(t, itok, id, "0", payload[:8000])
	require.Equal(t, 409, r.status, r.raw)
	assert.Equal(t, "media_offset_mismatch", r.body["error_code"])
	assert.Equal(t, float64(8000), r.body["offset"])
	assert.Equal(t, "8000", r.header.Get("Upload-Offset"))
	// A chunk ahead of the offset is refused the same way.
	r = e.chunk(t, itok, id, "12000", payload[12000:])
	require.Equal(t, 409, r.status, r.raw)

	// The resume point from GET; a crashed earlier attempt left uncommitted bytes past it on disk.
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d", id), itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, float64(8000), r.media()["offset_bytes"])
	parts := e.files(t)
	require.Len(t, parts, 1)
	f, err := os.OpenFile(filepath.Join(e.root, parts[0]), os.O_WRONLY|os.O_APPEND, 0)
	require.NoError(t, err)
	_, err = f.Write([]byte("garbage from an interrupted chunk"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	r = e.chunk(t, itok, id, "8000", payload[8000:16000])
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "uploading", r.media()["status"])
	r = e.chunk(t, itok, id, "16000", payload[16000:])
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "ready", r.media()["status"])
	assert.Equal(t, float64(len(payload)), r.media()["offset_bytes"])
	assert.NotNil(t, r.media()["completed_at"])
	assert.Nil(t, r.media()["upload_expires_at"])
	mediaID, status := e.lessonMedia(t, lid)
	assert.Equal(t, "ready", status)
	assert.Equal(t, int64(id), mediaID.Int64)
	stored := e.files(t)
	require.Len(t, stored, 1)
	assert.True(t, strings.HasSuffix(stored[0], ".bin"))
	onDisk, err := os.ReadFile(filepath.Join(e.root, stored[0]))
	require.NoError(t, err)
	assert.Equal(t, payload, onDisk)

	// A finished upload takes no more chunks.
	r = e.chunk(t, itok, id, strconv.Itoa(len(payload)), []byte{1})
	require.Equal(t, 409, r.status, r.raw)
	assert.Equal(t, "media_not_uploading", r.body["error_code"])

	// The lesson shows the media reference to the student.
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d", lid), stok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "ready", r.data()["lesson"].(map[string]any)["media"].(map[string]any)["status"])

	// Student playback URL.
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d/playback", lid), stok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "no-store", r.header.Get("Cache-Control"))
	pb := r.data()["playback"].(map[string]any)
	assert.Equal(t, "video/mp4", pb["mime"])
	assert.Equal(t, float64(len(payload)), pb["size_bytes"])
	link := pb["url"].(string)
	assert.True(t, strings.HasPrefix(link, fmt.Sprintf("https://api.example.test/api/v1/media/%d/stream?", id)), link)
	assert.Contains(t, link, "u="+strconv.FormatUint(studentID, 10))
	assert.Equal(t, "2026-10-07T10:30:00+03:30", pb["expires_at"])

	// Whole file.
	r = e.get(t, link, "")
	require.Equal(t, 200, r.status)
	assert.Equal(t, payload, []byte(r.raw))
	assert.Equal(t, "video/mp4", r.header.Get("Content-Type"))
	assert.Equal(t, "bytes", r.header.Get("Accept-Ranges"))
	assert.Equal(t, "private, no-store", r.header.Get("Cache-Control"))
	assert.Equal(t, "nosniff", r.header.Get("X-Content-Type-Options"))

	// Range 206.
	r = e.get(t, link, "bytes=100-199")
	require.Equal(t, 206, r.status)
	assert.Equal(t, payload[100:200], []byte(r.raw))
	assert.Equal(t, "bytes 100-199/20000", r.header.Get("Content-Range"))
	assert.Equal(t, "100", r.header.Get("Content-Length"))
	r = e.get(t, link, "bytes=19990-")
	require.Equal(t, 206, r.status)
	assert.Equal(t, payload[19990:], []byte(r.raw))
	r = e.get(t, link, "bytes=-10")
	require.Equal(t, 206, r.status)
	assert.Equal(t, payload[19990:], []byte(r.raw))
	r = e.get(t, link, "bytes=20000-")
	require.Equal(t, 416, r.status, r.raw)
	assert.Equal(t, "bytes */20000", r.header.Get("Content-Range"))
	assert.Equal(t, "range_not_satisfiable", r.body["error_code"])

	// Unauthorised / tampered URLs: 403 link_invalid.
	u, err := url.Parse(link)
	require.NoError(t, err)
	tamper := func(key, val string) string {
		q := u.Query()
		q.Set(key, val)
		v := *u
		v.RawQuery = q.Encode()
		return v.String()
	}
	for name, bad := range map[string]string{
		"other viewer":   tamper("u", "999"),
		"other expiry":   tamper("expires", strconv.FormatInt(t0.Add(20*time.Minute).Unix(), 10)),
		"bad signature":  tamper("signature", "AAAA"),
		"no signature":   fmt.Sprintf("https://api.example.test/api/v1/media/%d/stream", id),
		"other media id": strings.Replace(link, fmt.Sprintf("/media/%d/", id), fmt.Sprintf("/media/%d/", id+1), 1),
	} {
		r = e.get(t, bad, "")
		assert.Equal(t, 403, r.status, name)
		assert.Equal(t, "link_invalid", r.body["error_code"], name)
	}

	// Expired: 410 link_expired.
	e.clock.Set(t0.Add(31 * time.Minute))
	r = e.get(t, link, "")
	require.Equal(t, 410, r.status, r.raw)
	assert.Equal(t, "link_expired", r.body["error_code"])

	// Another student without a grant cannot mint a URL (the lesson's own 404).
	_, otok := e.user(t, "09120000051")
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d/playback", lid), otok, nil)
	require.Equal(t, 404, r.status, r.raw)
	assert.Equal(t, "learning_lesson_not_found", r.body["error_code"])
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d/playback", lid), "", nil)
	require.Equal(t, 401, r.status, r.raw)

	// The owner previews it; another approved instructor sees 404 everywhere (IDOR).
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d/playback", id), itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	other := e.instructor(t, "09120000002", true)
	for _, rr := range []response{
		e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d", id), other, nil),
		e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d/playback", id), other, nil),
		e.chunk(t, other, id, "0", payload[:10]),
		e.do(t, http.MethodDelete, fmt.Sprintf("/api/instructor/v1/media/%d", id), other, nil),
	} {
		assert.Equal(t, 404, rr.status, rr.raw)
		assert.Equal(t, "media_not_found", rr.body["error_code"])
	}
	// Nor can she upload into the first instructor's lesson.
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", other, map[string]any{"lesson_id": lid, "size": 100})
	require.Equal(t, 404, r.status, r.raw)
	assert.Equal(t, "learning_lesson_not_found", r.body["error_code"])
	assert.Len(t, e.files(t), 1)
}

func TestUploadLimitsAndGuards(t *testing.T) {
	e := setup(t, media.NewSigner(media.DevKey()))
	itok := e.instructor(t, "09120000001", true)
	_, vid := e.lesson(t, itok, "video")
	_, pdf := e.lesson(t, itok, "pdf")

	// Guards: 401 without a token, 403 for a non-instructor / a pending applicant.
	r := e.do(t, http.MethodPost, "/api/instructor/v1/media", "", map[string]any{"lesson_id": vid, "size": 10})
	assert.Equal(t, 401, r.status, r.raw)
	_, utok := e.user(t, "09120000060")
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", utok, map[string]any{"lesson_id": vid, "size": 10})
	assert.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "instructor_required", r.body["error_code"])
	ptok := e.instructor(t, "09120000003", false)
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", ptok, map[string]any{"lesson_id": vid, "size": 10})
	assert.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "instructor_pending", r.body["error_code"])

	// Validation 422.
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{})
	require.Equal(t, 422, r.status, r.raw)
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "lesson_id")
	assert.Contains(t, errs, "size")
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{"lesson_id": vid, "size": 0, "mime": "image/png"})
	require.Equal(t, 422, r.status, r.raw)
	errs = r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "size")
	assert.Contains(t, errs, "mime")
	// A declared type of another kind.
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{"lesson_id": vid, "size": 10, "mime": "application/pdf"})
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "mime")

	// Oversize 413 (2 GiB video, 100 MiB pdf).
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{"lesson_id": vid, "size": int64(2<<30) + 1})
	require.Equal(t, 413, r.status, r.raw)
	assert.Equal(t, "media_too_large", r.body["error_code"])
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{"lesson_id": pdf, "size": 100<<20 + 1})
	require.Equal(t, 413, r.status, r.raw)
	_, vid2 := e.lesson(t, itok, "video")
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{"lesson_id": vid2, "size": int64(2 << 30)})
	require.Equal(t, 201, r.status, "exactly 2 GiB is accepted: %s", r.raw)
	big := idOf(t, r.media()["id"])

	// Wrong type: a PDF into a video lesson → 415 on the first chunk; nothing advances.
	id := e.create(t, itok, vid, 1000)
	r = e.chunk(t, itok, id, "0", append([]byte("%PDF-1.7\n"), make([]byte, 100)...))
	require.Equal(t, 415, r.status, r.raw)
	assert.Equal(t, "media_type_not_accepted", r.body["error_code"])
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d", id), itok, nil)
	assert.Equal(t, float64(0), r.media()["offset_bytes"])

	// Chunk transport: wrong Content-Type / a content encoding → 415; a bad Upload-Offset → 422.
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/instructor/v1/media/%d", id), bytes.NewReader(video(100)))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set(media.HeaderUploadOffset, "0")
	r = e.send(t, req, itok)
	require.Equal(t, 415, r.status, r.raw)
	assert.Equal(t, "media_content_type", r.body["error_code"])
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/instructor/v1/media/%d", id), bytes.NewReader(video(100)))
	req.Header.Set("Content-Type", media.ChunkContentType)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set(media.HeaderUploadOffset, "0")
	r = e.send(t, req, itok)
	require.Equal(t, 415, r.status, r.raw)
	for _, off := range []string{"", "-1", "01", "abc"} {
		r = e.chunk(t, itok, id, off, video(100))
		require.Equal(t, 422, r.status, "offset %q: %s", off, r.raw)
		assert.Contains(t, r.body["errors"], "upload_offset")
	}

	// Past the declared size → 413; a chunk above MaxChunkBytes → 413.
	r = e.chunk(t, itok, id, "0", video(1001))
	require.Equal(t, 413, r.status, r.raw)
	assert.Equal(t, "media_beyond_size", r.body["error_code"])
	r = e.chunk(t, itok, big, "0", video(int(media.MaxChunkBytes)+1))
	require.Equal(t, 413, r.status, r.raw)
	assert.Equal(t, "media_chunk_too_large", r.body["error_code"])

	// A PDF lesson takes a PDF.
	doc := append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("x"), 200)...)
	pid := e.create(t, itok, pdf, len(doc))
	r = e.chunk(t, itok, pid, "0", doc)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "ready", r.media()["status"])
	assert.Equal(t, "application/pdf", r.media()["mime"])

	// Unknown media id / unknown lesson: 404.
	r = e.do(t, http.MethodGet, "/api/instructor/v1/media/999999", itok, nil)
	assert.Equal(t, 404, r.status)
	r = e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{"lesson_id": 999999, "size": 10})
	assert.Equal(t, 404, r.status)

	// An expired upload no longer takes chunks (410).
	e.clock.Set(t0.Add(media.UploadTTL + time.Minute))
	r = e.chunk(t, itok, big, "0", video(100))
	require.Equal(t, 410, r.status, r.raw)
	assert.Equal(t, "media_upload_expired", r.body["error_code"])
}

func TestActiveUploadLimit(t *testing.T) {
	e := setup(t, media.NewSigner(media.DevKey()))
	itok := e.instructor(t, "09120000001", true)
	for range media.MaxActiveUploads {
		_, lid := e.lesson(t, itok, "video")
		e.create(t, itok, lid, 100)
	}
	_, lid := e.lesson(t, itok, "video")
	r := e.do(t, http.MethodPost, "/api/instructor/v1/media", itok, map[string]any{"lesson_id": lid, "size": 100})
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "lesson_id")
}

func TestScannerRejectsUpload(t *testing.T) {
	e := setup(t, media.NewSigner(media.DevKey()))
	e.scan.err = media.ErrInfected
	itok := e.instructor(t, "09120000001", true)
	_, lid := e.lesson(t, itok, "video")
	id := e.create(t, itok, lid, 500)
	r := e.chunk(t, itok, id, "0", video(500))
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, "media_rejected", r.body["error_code"])
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d", id), itok, nil)
	assert.Equal(t, "failed", r.media()["status"])
	mediaID, status := e.lessonMedia(t, lid)
	assert.Equal(t, "failed", status)
	assert.False(t, mediaID.Valid)
	assert.Empty(t, e.files(t))

	// Retrying the lesson clears the failed upload.
	e.scan.err = nil
	id2 := e.create(t, itok, lid, 500)
	r = e.chunk(t, itok, id2, "0", video(500))
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d", id), itok, nil)
	assert.Equal(t, 404, r.status)
}

func TestReplaceCancelAndDisabledPlayback(t *testing.T) {
	e := setup(t, nil) // no URL key: playback 503, uploads work
	itok := e.instructor(t, "09120000001", true)
	_, lid := e.lesson(t, itok, "audio")
	mp3 := append([]byte("ID3\x04\x00\x00"), bytes.Repeat([]byte{0x55}, 300)...)

	first := e.create(t, itok, lid, len(mp3))
	require.Equal(t, 200, e.chunk(t, itok, first, "0", mp3).status)
	// A new upload for the same lesson: the ready one stays until the new one completes.
	second := e.create(t, itok, lid, len(mp3))
	mediaID, status := e.lessonMedia(t, lid)
	assert.Equal(t, int64(first), mediaID.Int64)
	assert.Equal(t, "uploading", status)
	// Cancelling the new upload puts the lesson back on the ready file.
	r := e.do(t, http.MethodDelete, fmt.Sprintf("/api/instructor/v1/media/%d", second), itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	mediaID, status = e.lessonMedia(t, lid)
	assert.Equal(t, int64(first), mediaID.Int64)
	assert.Equal(t, "ready", status)

	third := e.create(t, itok, lid, len(mp3))
	r = e.chunk(t, itok, third, "0", mp3)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "audio/mpeg", r.media()["mime"])
	mediaID, status = e.lessonMedia(t, lid)
	assert.Equal(t, int64(third), mediaID.Int64)
	assert.Equal(t, "ready", status)
	assert.Len(t, e.files(t), 1, "the replaced file is removed")
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d", first), itok, nil)
	assert.Equal(t, 404, r.status)

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/media/%d/playback", third), itok, nil)
	require.Equal(t, 503, r.status, r.raw)
	assert.Equal(t, "media_unavailable", r.body["error_code"])
	r = e.get(t, fmt.Sprintf("/api/v1/media/%d/stream?u=1&expires=1&signature=x", third), "")
	require.Equal(t, 503, r.status, r.raw)

	// Deleting the ready file empties the lesson.
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/instructor/v1/media/%d", third), itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	mediaID, status = e.lessonMedia(t, lid)
	assert.False(t, mediaID.Valid)
	assert.Equal(t, "none", status)
	assert.Empty(t, e.files(t))
}

func TestSweep(t *testing.T) {
	e := setup(t, media.NewSigner(media.DevKey()))
	itok := e.instructor(t, "09120000001", true)
	_, lid := e.lesson(t, itok, "video")
	_, lid2 := e.lesson(t, itok, "video")

	abandoned := e.create(t, itok, lid, 1000)
	require.Equal(t, 200, e.chunk(t, itok, abandoned, "0", video(400)).status)
	done := e.create(t, itok, lid2, 300)
	require.Equal(t, 200, e.chunk(t, itok, done, "0", video(300)).status)
	require.Len(t, e.files(t), 2)

	// Nothing is due yet.
	n, err := e.svc.Sweep(context.Background(), t0.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// After the resume window the abandoned upload goes; the lesson is back to «none».
	n, err = e.svc.Sweep(context.Background(), t0.Add(media.UploadTTL+time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	_, status := e.lessonMedia(t, lid)
	assert.Equal(t, "none", status)
	assert.Len(t, e.files(t), 1)

	// A deleted lesson's media is an orphan.
	var course int
	require.NoError(t, e.db.QueryRow("SELECT course_id FROM learning_lessons WHERE id = ?", lid2).Scan(&course))
	r := e.do(t, http.MethodDelete, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons/%d", course, lid2), itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	// A stray file nobody points at (older than the grace period) goes too.
	ins := strings.Split(e.files(t)[0], "/")[0]
	stray := filepath.Join(e.root, ins, strings.Repeat("ab", 20)+".part")
	require.NoError(t, os.WriteFile(stray, []byte("x"), 0o600))
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(stray, old, old))
	// A directory of an instructor that no longer exists.
	require.NoError(t, os.MkdirAll(filepath.Join(e.root, "987654"), 0o700))

	n, err = e.svc.Sweep(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Empty(t, e.files(t))
	_, err = os.Stat(filepath.Join(e.root, "987654"))
	assert.True(t, os.IsNotExist(err))
	var rows int
	require.NoError(t, e.db.QueryRow("SELECT COUNT(*) FROM learning_media").Scan(&rows))
	assert.Equal(t, 0, rows)
}
