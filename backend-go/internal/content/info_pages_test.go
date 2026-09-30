package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// fakeInfoPages answers only ListActiveInfoPageSections; the embedded nil Querier panics on anything else.
type fakeInfoPages struct {
	Querier
	rows  map[string][]store.ListActiveInfoPageSectionsRow
	asked []string
}

func (f *fakeInfoPages) ListActiveInfoPageSections(_ context.Context, group string) ([]store.ListActiveInfoPageSectionsRow, error) {
	f.asked = append(f.asked, strings.Clone(group)) // Fiber reuses the params buffer
	return f.rows[group], nil
}

func TestInfoPage(t *testing.T) {
	at := func(d int) sql.NullTime {
		return sql.NullTime{Time: time.Date(2026, 9, d, 22, 0, 0, 0, civildate.Tehran), Valid: true}
	}
	q := &fakeInfoPages{rows: map[string][]store.ListActiveInfoPageSectionsRow{
		"privacy": {
			{ID: 1, Key: sql.NullString{String: "summary", Valid: true}, Heading: json.RawMessage(`{"fa":"خلاصه","en":"Summary"}`),
				Body: json.RawMessage(`{"fa":"الف\nب","en":"a\nb"}`), UpdatedAt: at(20)},
			{ID: 2, Heading: json.RawMessage(`{"fa":"تماس"}`), Body: json.RawMessage(`{"fa":"متن"}`),
				LinkLabel: db.NullRawJSON{V: json.RawMessage(`{"fa":"ایمیل"}`), Valid: true},
				LinkUrl:   sql.NullString{String: "mailto:a@b.c", Valid: true}, UpdatedAt: at(28)},
			{ID: 3, Heading: json.RawMessage(`{"fa":"بدون لینک"}`), Body: json.RawMessage(`{"fa":"x"}`),
				LinkUrl: sql.NullString{String: "https://x.y", Valid: true}}, // no label → no link
		},
	}}
	h := New(Deps{Queries: q})
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Get("/info-pages/:group", h.InfoPage)
	get := func(path, lang string) (int, map[string]any) {
		req := httptest.NewRequest(fiber.MethodGet, path, nil)
		req.Header.Set("Accept-Language", lang)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m), string(raw))
		return resp.StatusCode, m
	}

	status, body := get("/info-pages/privacy", "en")
	require.Equal(t, 200, status)
	data := body["data"].(map[string]any)
	assert.Equal(t, "privacy", data["group"])
	assert.Equal(t, "2026-09-28", data["updated_at"], "the latest edit of the page, Tehran day")
	secs := data["sections"].([]any)
	require.Len(t, secs, 3)
	first := secs[0].(map[string]any)
	assert.Equal(t, "summary", first["key"])
	assert.Equal(t, "Summary", first["heading"])
	assert.Equal(t, "a\nb", first["body"])
	second := secs[1].(map[string]any)
	assert.Nil(t, second["key"])
	assert.Equal(t, "تماس", second["heading"], "missing language falls back to the default")
	assert.Equal(t, "mailto:a@b.c", second["link_url"])
	third := secs[2].(map[string]any)
	assert.Nil(t, third["link_url"], "a link needs its label")

	status, body = get("/info-pages/support", "fa")
	require.Equal(t, 200, status)
	data = body["data"].(map[string]any)
	assert.Nil(t, data["updated_at"])
	assert.Empty(t, data["sections"])

	status, _ = get("/info-pages/nope", "fa")
	assert.Equal(t, 404, status)
	assert.Equal(t, []string{"privacy", "support"}, q.asked, "an unknown group never reaches the store")
}
