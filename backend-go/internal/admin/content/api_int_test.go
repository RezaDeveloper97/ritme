package content_test

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/media"
	publiccontent "github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const appURL = "https://api.ritme.test"

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	content.New(content.Deps{DB: e.DB, Disk: media.NewDisk(e.Storage), AppURL: appURL, Logger: admintest.Quiet}).
		Routes(e.Route(), e.Kit)
	// The public disk as the app reads it (T-M2-10's /storage handler).
	e.App.Get("/storage/*", publiccontent.New(publiccontent.Deps{StoragePath: e.Storage, AppURL: appURL}).Storage)
	return e
}

func picture(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y += 4 {
		for x := 0; x < w; x += 4 {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	return img
}

func jpegOf(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, jpeg.Encode(&b, picture(w, h), &jpeg.Options{Quality: 85}))
	return b.Bytes()
}

func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, picture(w, h)))
	return b.Bytes()
}

func gifOf(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, gif.Encode(&b, picture(900, 500), nil))
	return b.Bytes()
}

func id(t *testing.T, obj map[string]any) string {
	t.Helper()
	n, ok := obj["id"].(float64)
	require.True(t, ok, obj)
	return strconv.Itoa(int(n))
}

func fileOnDisk(e *admintest.Env, rel string) string {
	return filepath.Join(e.Storage, "app", "public", filepath.FromSlash(rel))
}

// ---------------------------------------------------------------------------
// Guards

func TestGuards(t *testing.T) {
	e := newEnv(t)
	assert.Equal(t, 401, e.Anonymous().Get("/articles").Status)

	c := e.As(admintest.EditorID)
	assert.Equal(t, 200, c.Get("/articles").Status, "editors manage content")

	c.CSRF = ""
	r := c.JSON(fiber.MethodPost, "/affirmations", map[string]any{"text": map[string]any{"fa": "x"}})
	assert.Equal(t, 419, r.Status)
	assert.Equal(t, "csrf_mismatch", r.Code())

	c = e.As(admintest.EditorID)
	c.CSRF = "wrong"
	assert.Equal(t, 419, c.JSON(fiber.MethodDelete, "/articles/1", nil).Status)

	c = e.As(admintest.SuperID)
	assert.Equal(t, 404, c.Get("/articles/999").Status)
	assert.Equal(t, 404, c.Get("/articles/abc").Status)
	assert.Equal(t, "not_found", c.Get("/banners/999").Code())
}

// ---------------------------------------------------------------------------
// Articles + covers

func TestArticlesCRUDAndSanitizedBody(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.JSON(fiber.MethodPost, "/articles", map[string]any{"slug": "x"})
	require.Equal(t, 422, r.Status, r.Body)
	assert.Equal(t, "validation_failed", r.Code())
	assert.Contains(t, r.Errors(), "title", "only the default language is required: title itself")
	assert.Contains(t, r.Errors(), "title.fa")
	assert.NotContains(t, r.Errors(), "title.en")

	r = c.JSON(fiber.MethodPost, "/articles", map[string]any{
		"slug": "sleep-tips", "title": map[string]any{"fa": "خواب"},
		"body":         map[string]any{"fa": `<p onclick="x()">سلام<script>alert(1)</script></p>`, "en": "   "},
		"cycle_phases": []string{"menstruation", "menstruation", "follicular"},
		"is_published": true, "read_time_minutes": 5,
	})
	require.Equal(t, 201, r.Status, r.Body)
	a := r.Obj("article")
	assert.Equal(t, map[string]any{"fa": "خواب"}, a["title"])
	body := a["body"].(map[string]any)
	assert.Equal(t, "<p>سلام</p>", body["fa"], "sanitised on save")
	assert.Nil(t, body["en"])
	assert.Equal(t, []any{"menstruation", "follicular"}, a["cycle_phases"], "unique")
	assert.Equal(t, true, a["is_published"])
	assert.NotNil(t, a["published_at"])
	aid := id(t, a)

	// Unique slug; bad phase; read time out of range.
	r = c.JSON(fiber.MethodPost, "/articles", map[string]any{
		"slug": "sleep-tips", "title": map[string]any{"fa": "x"}, "cycle_phases": []string{"nope"},
		"read_time_minutes": 500,
	})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "slug")
	assert.Contains(t, r.Errors(), "cycle_phases.0")
	assert.Contains(t, r.Errors(), "read_time_minutes")

	// Update: excerpt not sent → unchanged; category cleared by an empty value.
	e.Exec(`UPDATE articles SET excerpt = '{"fa":"خلاصه"}', category = 'sleep' WHERE id = ?`, aid)
	r = c.JSON(fiber.MethodPut, "/articles/"+aid, map[string]any{
		"slug": "sleep-tips", "title": map[string]any{"fa": "خواب ۲", "en": "Sleep"}, "category": "",
	})
	require.Equal(t, 200, r.Status, r.Body)
	a = r.Obj("article")
	assert.Equal(t, map[string]any{"fa": "خلاصه"}, a["excerpt"])
	assert.Nil(t, a["category"])
	assert.Nil(t, a["cycle_phases"], "no phase ticked = general (null)")
	assert.Equal(t, false, a["is_published"])
	assert.NotNil(t, a["published_at"], "unpublishing keeps published_at")

	r = c.JSON(fiber.MethodPost, "/articles/"+aid+"/toggle", nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, true, r.Obj("article")["is_published"])

	r = c.Get("/articles")
	require.Equal(t, 200, r.Status)
	assert.Len(t, r.Items(), 1)
	assert.EqualValues(t, 1, r.Data()["meta"].(map[string]any)["total"])

	r = c.Get("/articles/options")
	require.Equal(t, 200, r.Status)
	assert.NotEmpty(t, r.Data()["phases"])

	r = c.JSON(fiber.MethodDelete, "/articles/"+aid, nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM articles`))
	assert.Equal(t, 404, c.JSON(fiber.MethodDelete, "/articles/"+aid, nil).Status)
}

func TestArticleCoverUpload(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	fields := map[string]string{"slug": "cover", "title[fa]": "عنوان", "is_published": "1"}

	// Wrong type (GIF), not an image (SVG with script, renamed .jpg), oversize.
	r := c.Multipart(fiber.MethodPost, "/articles", fields,
		admintest.File{Field: "image", Name: "a.gif", ContentType: "image/gif", Data: gifOf(t)})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors()["image"].([]any)[0], "jpeg, jpg, png, webp")

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><script>alert(1)</script></svg>`)
	r = c.Multipart(fiber.MethodPost, "/articles", fields,
		admintest.File{Field: "image", Name: "a.jpg", ContentType: "image/jpeg", Data: svg})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "image")

	big := append([]byte{0xFF, 0xD8, 0xFF}, bytes.Repeat([]byte{0}, 8192*1024)...)
	r = c.Multipart(fiber.MethodPost, "/articles", fields,
		admintest.File{Field: "image", Name: "a.jpg", ContentType: "image/jpeg", Data: big})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors()["image"].([]any)[0], "8192")
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM articles`))

	// A valid 2000×1000 JPEG becomes a ≤1080px WebP on the volume, served by /storage.
	r = c.Multipart(fiber.MethodPost, "/articles", fields,
		admintest.File{Field: "image", Name: "photo.jpg", ContentType: "image/jpeg", Data: jpegOf(t, 2000, 1000)})
	require.Equal(t, 201, r.Status, r.Body)
	a := r.Obj("article")
	rel, _ := a["image_path"].(string)
	assert.Regexp(t, `^articles/[A-Za-z0-9]{40}\.webp$`, rel)
	assert.Equal(t, appURL+"/storage/"+rel, a["cover_url"])
	stored, err := os.ReadFile(fileOnDisk(e, rel))
	require.NoError(t, err)
	img, err := media.Inspect(stored)
	require.NoError(t, err)
	assert.Equal(t, media.FormatWebP, img.Format)
	assert.Equal(t, [2]int{1080, 540}, [2]int{img.Width, img.Height})

	res, err := e.App.Test(httptest.NewRequest(fiber.MethodGet, "/storage/"+rel, nil))
	require.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "image/webp", res.Header.Get("Content-Type"))
	_ = res.Body.Close()

	// Replacing the cover deletes the old file; remove_image clears it.
	aid := id(t, a)
	r = c.Multipart(fiber.MethodPut, "/articles/"+aid, map[string]string{"slug": "cover", "title[fa]": "عنوان"},
		admintest.File{Field: "image", Name: "b.png", Data: pngOf(t, 400, 300)})
	require.Equal(t, 200, r.Status, r.Body)
	rel2 := r.Obj("article")["image_path"].(string)
	assert.NotEqual(t, rel, rel2)
	assert.NoFileExists(t, fileOnDisk(e, rel))
	assert.FileExists(t, fileOnDisk(e, rel2))

	r = c.Multipart(fiber.MethodPost, "/articles/"+aid, map[string]string{"slug": "cover", "title[fa]": "عنوان",
		"remove_image": "1"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Nil(t, r.Obj("article")["image_path"])
	assert.NoFileExists(t, fileOnDisk(e, rel2))
}

// ---------------------------------------------------------------------------
// Banners

func TestBannerUploads(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	fields := map[string]string{"position": "home_top", "title[fa]": "بنر", "is_active": "1"}

	r := c.Multipart(fiber.MethodPost, "/banners", fields)
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "image", "required on create")

	r = c.Multipart(fiber.MethodPost, "/banners", fields,
		admintest.File{Field: "image", Name: "s.png", Data: pngOf(t, 700, 400)})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors()["image"].([]any)[0], "dimensions")

	big := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1}, 4096*1024)...)
	r = c.Multipart(fiber.MethodPost, "/banners", fields, admintest.File{Field: "image", Name: "b.png", Data: big})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors()["image"].([]any)[0], "4096")

	r = c.Multipart(fiber.MethodPost, "/banners", fields,
		admintest.File{Field: "image", Name: "g.png", Data: gifOf(t)})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "image")

	// Links: an external link must be a URL; a link type needs a URL.
	bad := map[string]string{"position": "home_top", "link_type": "external", "link_url": "not a url"}
	r = c.Multipart(fiber.MethodPost, "/banners", bad, admintest.File{Field: "image", Name: "a.png", Data: pngOf(t, 1080, 540)})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "link_url")
	r = c.Multipart(fiber.MethodPost, "/banners", map[string]string{"position": "home_top", "link_type": "internal"},
		admintest.File{Field: "image", Name: "a.png", Data: pngOf(t, 1080, 540)})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "link_url")
	r = c.Multipart(fiber.MethodPost, "/banners", map[string]string{"position": "nowhere",
		"starts_at": "2026-10-10", "ends_at": "2026-10-01"},
		admintest.File{Field: "image", Name: "a.png", Data: pngOf(t, 1080, 540)})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "position")
	assert.Contains(t, r.Errors(), "ends_at")

	// Valid: stored as uploaded under banners/.
	fields["link_type"], fields["link_url"] = "external", "https://ritme.app/x"
	fields["starts_at"], fields["ends_at"] = "2026-10-01 08:00", "2026-10-31"
	pngData := pngOf(t, 1080, 540)
	r = c.Multipart(fiber.MethodPost, "/banners", fields, admintest.File{Field: "image", Name: "a.png", Data: pngData})
	require.Equal(t, 201, r.Status, r.Body)
	b := r.Obj("banner")
	rel := b["image_path"].(string)
	assert.Regexp(t, `^banners/[A-Za-z0-9]{40}\.png$`, rel)
	assert.Equal(t, appURL+"/storage/"+rel, b["image_url"])
	assert.Equal(t, "2026-10-01T08:00:00+03:30", b["starts_at"])
	onDisk, err := os.ReadFile(fileOnDisk(e, rel))
	require.NoError(t, err)
	assert.Equal(t, pngData, onDisk)
	bid := id(t, b)

	// Update without a file keeps the image; clearing the URL drops both link fields.
	r = c.JSON(fiber.MethodPut, "/banners/"+bid, map[string]any{"position": "home_top", "link_url": ""})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, rel, r.Obj("banner")["image_path"])
	assert.Nil(t, r.Obj("banner")["link_type"])
	assert.Equal(t, false, r.Obj("banner")["is_active"])

	r = c.Multipart(fiber.MethodPost, "/banners/"+bid, map[string]string{"position": "home_top"},
		admintest.File{Field: "image", Name: "n.jpg", Data: jpegOf(t, 1200, 600)})
	require.Equal(t, 200, r.Status, r.Body)
	rel2 := r.Obj("banner")["image_path"].(string)
	assert.Regexp(t, `\.jpg$`, rel2)
	assert.NoFileExists(t, fileOnDisk(e, rel))

	r = c.JSON(fiber.MethodPost, "/banners/"+bid+"/toggle", nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, true, r.Obj("banner")["is_active"])

	require.Equal(t, 200, c.JSON(fiber.MethodDelete, "/banners/"+bid, nil).Status)
	assert.NoFileExists(t, fileOnDisk(e, rel2))
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM banners`))
}

// ---------------------------------------------------------------------------
// The simple resources: create / update / toggle / delete + a 422 each

type resourceCase struct {
	path, key string
	valid     map[string]any
	invalid   map[string]any
	wantErr   string
	update    map[string]any
	check     func(t *testing.T, obj map[string]any)
	toggles   string // field flipped by /toggle ("" = no toggle endpoint)
	listQuery string
}

func TestSimpleResources(t *testing.T) {
	cases := []resourceCase{
		{
			path: "/affirmations", key: "affirmation",
			valid:   map[string]any{"text": map[string]any{"fa": "من کافی هستم"}, "cycle_phase": "luteal", "is_active": true},
			invalid: map[string]any{"text": map[string]any{"en": "only en"}, "cycle_phase": "x"}, wantErr: "text.fa",
			update: map[string]any{"text": map[string]any{"fa": "ب", "en": "B"}, "sort_order": 3},
			check: func(t *testing.T, o map[string]any) {
				assert.Equal(t, "luteal", o["cycle_phase"], "absent optional field unchanged")
				assert.EqualValues(t, 3, o["sort_order"])
			},
			toggles: "is_active",
		},
		{
			path: "/challenges", key: "challenge",
			valid:   map[string]any{"title": map[string]any{"fa": "آب"}, "cycle_day_from": 3, "cycle_day_to": 7, "is_active": "1"},
			invalid: map[string]any{"title": map[string]any{"fa": "x"}, "cycle_day_from": 9, "cycle_day_to": 4}, wantErr: "cycle_day_to",
			update: map[string]any{"title": map[string]any{"fa": "آب"}, "category": "water"},
			check: func(t *testing.T, o map[string]any) {
				assert.Nil(t, o["cycle_day_from"], "day range is always rewritten")
				assert.Equal(t, "water", o["category"])
			},
			toggles: "is_active",
		},
		{
			path: "/recommendations", key: "recommendation",
			valid: map[string]any{"type": "general", "text": map[string]any{"fa": "استراحت"},
				"title": map[string]any{"fa": "", "en": ""}, "cycle_phase": "luteal", "cycle_subphases": []string{"late_luteal"}},
			invalid: map[string]any{"type": "general", "text": map[string]any{"fa": "x"}, "cycle_phase": "menstruation",
				"cycle_subphases": []string{"late_luteal"}},
			wantErr: "cycle_subphases.0",
			update:  map[string]any{"type": "general", "text": map[string]any{"fa": "x"}},
			check: func(t *testing.T, o map[string]any) {
				assert.Equal(t, "luteal", o["cycle_phase"])
				assert.Nil(t, o["cycle_subphases"])
				assert.Nil(t, o["title"])
			},
			toggles: "is_active",
		},
		{
			path: "/task-templates", key: "task_template",
			valid:   map[string]any{"key": "drink", "title": map[string]any{"fa": "آب"}, "category": "hydration"},
			invalid: map[string]any{"key": "drink", "title": map[string]any{"fa": "x"}, "category": "hydration"}, wantErr: "key",
			update:  map[string]any{"key": "drink", "title": map[string]any{"fa": "آب"}, "category": "hydration", "icon": "💧"},
			check:   func(t *testing.T, o map[string]any) { assert.Equal(t, "💧", o["icon"]) },
			toggles: "is_active",
		},
		{
			path: "/info-sections", key: "info_section",
			valid: map[string]any{"group": "help", "heading": map[string]any{"fa": "تماس"}, "body": map[string]any{"fa": "متن"},
				"link_url": "mailto:hi@ritme.app", "link_label": map[string]any{"fa": "ایمیل", "en": ""}},
			invalid: map[string]any{"group": "help", "heading": map[string]any{"fa": "x"}, "body": map[string]any{"fa": "x"},
				"link_url": "javascript:alert(1)"},
			wantErr: "link_url",
			update:  map[string]any{"group": "about", "heading": map[string]any{"fa": "ما"}, "body": map[string]any{"fa": "متن"}},
			check: func(t *testing.T, o map[string]any) {
				assert.Equal(t, "about", o["group"])
				assert.Nil(t, o["link_url"], "a cleared link becomes null")
				assert.Nil(t, o["link_label"])
			},
			toggles:   "is_active",
			listQuery: "?group=about",
		},
		{
			path: "/pregnancy-weeks", key: "pregnancy_week",
			valid:   map[string]any{"week_number": 12, "faq": map[string]any{"fa": "سوال"}},
			invalid: map[string]any{"week_number": 43}, wantErr: "week_number",
			update: map[string]any{"week_number": 13, "care_plan": map[string]any{"fa": "برنامه"}},
			check: func(t *testing.T, o map[string]any) {
				assert.EqualValues(t, 13, o["week_number"])
				assert.Equal(t, map[string]any{"fa": "سوال"}, o["faq"], "absent field unchanged")
				assert.Equal(t, map[string]any{"fa": "برنامه"}, o["care_plan"])
			},
		},
		{
			path: "/phase-contents", key: "phase_content",
			valid:   map[string]any{"phase": "mid_luteal", "sleep": map[string]any{"fa": "خواب"}},
			invalid: map[string]any{"phase": "not_a_phase"}, wantErr: "phase",
			update: map[string]any{"phase": "mid_luteal", "sleep": nil},
			check:  func(t *testing.T, o map[string]any) { assert.Nil(t, o["sleep"]) },
		},
	}
	for _, tc := range cases {
		t.Run(strings.TrimPrefix(tc.path, "/"), func(t *testing.T) {
			e := newEnv(t)
			c := e.As(admintest.EditorID)

			r := c.JSON(fiber.MethodPost, tc.path, tc.valid)
			require.Equal(t, 201, r.Status, r.Body)
			obj := r.Obj(tc.key)
			rid := id(t, obj)

			r = c.JSON(fiber.MethodPost, tc.path, tc.invalid)
			require.Equal(t, 422, r.Status, r.Body)
			assert.Contains(t, r.Errors(), tc.wantErr)

			r = c.Get(tc.path + "/" + rid)
			require.Equal(t, 200, r.Status)
			assert.Equal(t, obj["id"], r.Obj(tc.key)["id"])

			r = c.JSON(fiber.MethodPut, tc.path+"/"+rid, tc.update)
			require.Equal(t, 200, r.Status, r.Body)
			tc.check(t, r.Obj(tc.key))

			if tc.toggles != "" {
				before := r.Obj(tc.key)[tc.toggles]
				r = c.JSON(fiber.MethodPost, tc.path+"/"+rid+"/toggle", nil)
				require.Equal(t, 200, r.Status, r.Body)
				assert.NotEqual(t, before, r.Obj(tc.key)[tc.toggles])
			}

			r = c.Get(tc.path + tc.listQuery)
			require.Equal(t, 200, r.Status, r.Body)
			assert.NotEmpty(t, r.Items())

			r = c.JSON(fiber.MethodDelete, tc.path+"/"+rid, nil)
			require.Equal(t, 200, r.Status, r.Body)
			assert.Equal(t, 404, c.Get(tc.path+"/"+rid).Status)
		})
	}
}

func TestTranslatableFollowsLanguagesTable(t *testing.T) {
	e := newEnv(t)
	e.Exec(`INSERT INTO languages (code, name, english_name, direction, is_active, is_default, sort_order, created_at, updated_at)
		VALUES ('ar', 'العربية', 'Arabic', 'rtl', 1, 0, 30, NOW(), NOW())`)
	c := e.As(admintest.SuperID)
	r := c.JSON(fiber.MethodPost, "/affirmations", map[string]any{
		"text": map[string]any{"fa": "الف", "ar": "أ", "xx": "dropped"}, "is_active": true,
	})
	require.Equal(t, 201, r.Status, r.Body)
	assert.Equal(t, map[string]any{"fa": "الف", "ar": "أ"}, r.Obj("affirmation")["text"],
		"one value per active language; unknown keys are not stored")

	r = c.JSON(fiber.MethodPost, "/affirmations", map[string]any{"text": map[string]any{"fa": "x", "ar": 5}})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "text.ar")
}

func TestChallengeFiltersAndCompletionsReport(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	e.Exec(`INSERT INTO challenges (id, title, description, cycle_day_from, cycle_day_to, category, is_active, sort_order, created_at, updated_at) VALUES
		(1, '{"fa":"آب بنوش"}', NULL, 1, 5, 'water', 1, 0, NOW(), NOW()),
		(2, '{"fa":"Walk"}', NULL, NULL, NULL, 'move', 0, 0, NOW(), NOW()),
		(3, '{"fa":"Stretch"}', NULL, 10, 20, NULL, 1, 0, NOW(), NOW())`)
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (7, 'Sara', '09120000007', NOW(), NOW()),
		(8, 'Mina', '09120000008', NOW(), NOW())`)
	e.Exec(`INSERT INTO user_challenge_completions (user_id, challenge_id, completion_date, completed_at, created_at, updated_at) VALUES
		(7, 1, '2026-09-01', NOW(), NOW(), NOW()), (8, 1, '2026-09-02', NOW(), NOW(), NOW()), (7, 3, '2026-09-10', NOW(), NOW(), NOW())`)

	ids := func(r admintest.Resp) []float64 {
		var out []float64
		for _, it := range r.Items() {
			out = append(out, it.(map[string]any)["id"].(float64))
		}
		return out
	}
	assert.ElementsMatch(t, []float64{1, 2}, ids(c.Get("/challenges?cycle_day=3")), "untargeted rows match every day")
	assert.ElementsMatch(t, []float64{1, 3}, ids(c.Get("/challenges?status=active")))
	assert.ElementsMatch(t, []float64{1}, ids(c.Get("/challenges?q=%D8%A2%D8%A8")), "Persian search on escaped JSON")
	assert.ElementsMatch(t, []float64{2}, ids(c.Get("/challenges?q=move")))

	r := c.JSON(fiber.MethodPost, "/challenges", map[string]any{"title": map[string]any{"fa": "x"}, "cycle_day_to": 4})
	require.Equal(t, 422, r.Status, "gte:cycle_day_from fails without a from (Laravel)")

	r = c.Get("/challenge-completions")
	require.Equal(t, 200, r.Status, r.Body)
	stats := r.Data()["stats"].(map[string]any)
	assert.EqualValues(t, 3, stats["total"])
	assert.EqualValues(t, 2, stats["users"])
	assert.Len(t, r.Items(), 3)
	assert.Len(t, r.Data()["per_challenge"], 2)
	assert.Len(t, r.Data()["challenges"], 3)

	r = c.Get("/challenge-completions?challenge_id=1&from=2026-09-02&q=Mina")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 1)
	assert.Equal(t, "Mina", r.Items()[0].(map[string]any)["user"].(map[string]any)["name"])

	r = c.Get("/challenge-completions?challenge_id=99&from=nope")
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "challenge_id")
	assert.Contains(t, r.Errors(), "from")
}
