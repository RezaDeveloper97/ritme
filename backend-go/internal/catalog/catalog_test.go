package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

var langs = i18n.Languages{
	{Code: "fa", Name: "فارسی", EnglishName: "Persian", Direction: "rtl", IsDefault: true},
	{Code: "en", Name: "English", EnglishName: "English", Direction: "ltr"},
}

func encode(t *testing.T, v any) string {
	t.Helper()
	b, err := jsonx.Marshal(v, jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
	require.NoError(t, err)
	return string(b)
}

func TestValidCode(t *testing.T) {
	assert.True(t, ValidGroup("teen_faq"))
	assert.True(t, ValidGroup("meno_score_items2"))
	for _, bad := range []string{"", "Teen", "1abc", "_x", "a-b", "a b", "a.b", string(make([]byte, 65))} {
		assert.False(t, ValidGroup(bad), bad)
	}
}

func TestItemFor(t *testing.T) {
	all := Item{Code: "a"}
	teen := Item{Code: "b", Audiences: []string{"teen", "cycle"}}
	assert.True(t, all.For(""))
	assert.True(t, all.For("teen"))
	assert.True(t, teen.For(""))
	assert.True(t, teen.For("cycle"))
	assert.False(t, teen.For("menopause"))
}

func TestLocalizer_TextFallsBack(t *testing.T) {
	en := Localizer{Locale: "en", Default: "fa", Langs: langs}
	assert.Equal(t, "Hi", en.Text(json.RawMessage(`{"fa":"سلام","en":"Hi"}`)))
	assert.Equal(t, "سلام", en.Text(json.RawMessage(`{"fa":"سلام","en":""}`)), "empty en → default")
	assert.Equal(t, "Only", en.Text(json.RawMessage(`{"de":"Only"}`)), "first non-empty")
	assert.Nil(t, en.Text(json.RawMessage(`{"fa":"","en":null}`)))
	assert.Nil(t, en.Text(nil))
}

func TestLocalizer_MetaPicksNestedTranslations(t *testing.T) {
	raw := json.RawMessage(`{"severity":"urgent","score":{"min":0,"max":4},` +
		`"steps":[{"fa":"یک","en":"One"},{"fa":"دو"}],` +
		`"cta":{"label":{"fa":"تماس","en":"Call"},"href":"tel:115"},` +
		`"codes":{"fa":1,"xx_y":2},"legacy":{"ar":"x"}}`)
	en := Localizer{Locale: "en", Default: "fa", Langs: langs}
	assert.JSONEq(t, `{"severity":"urgent","score":{"min":0,"max":4},"steps":["One","دو"],`+
		`"cta":{"label":"Call","href":"tel:115"},"codes":{"fa":1,"xx_y":2},"legacy":{"ar":"x"}}`,
		encode(t, en.Meta(raw)), "non-language keys and objects without an active language stay as stored")

	fa := Localizer{Locale: "fa", Default: "fa", Langs: langs}
	assert.JSONEq(t, `["یک","دو"]`, encode(t, fa.Meta(json.RawMessage(`[{"fa":"یک","en":"One"},{"fa":"دو"}]`))))
	assert.Nil(t, fa.Meta(nil))
	assert.Nil(t, fa.Meta(json.RawMessage(`{bad`)))
}

func TestLocalizer_Public(t *testing.T) {
	en := Localizer{Locale: "en", Default: "fa", Langs: langs}
	got := encode(t, en.Public(Item{Code: "x", Title: json.RawMessage(`{"fa":"الف","en":"A"}`), NeedsReview: true}))
	assert.Equal(t, `{"code":"x","title":"A","body":null,"meta":null,"audiences":null,"needs_review":true}`, got)
	got = encode(t, en.Public(Item{Code: "y", Title: json.RawMessage(`{"fa":"ب"}`), Audiences: []string{"teen"}}))
	assert.Equal(t, `{"code":"y","title":"ب","body":null,"meta":null,"audiences":["teen"],"needs_review":false}`, got)
}

type fakeLister struct {
	rows  []store.ListActiveCatalogItemsRow
	calls int
	err   error
}

func (f *fakeLister) ListActiveCatalogItems(context.Context, string) ([]store.ListActiveCatalogItemsRow, error) {
	f.calls++
	return f.rows, f.err
}

func TestReader_CachesPerGroupAndFlushes(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	c := cache.NewFromClient(rdb, "ritme-go:")
	f := &fakeLister{rows: []store.ListActiveCatalogItemsRow{{
		Code: "a", Title: json.RawMessage(`{"fa":"الف"}`),
		Audiences: db.NullRawJSON{V: json.RawMessage(`["teen"]`), Valid: true},
		Meta:      db.NullRawJSON{V: json.RawMessage(`{"k":1}`), Valid: true},
	}}}
	r := NewReader(f, c, 0, nil)
	ctx := t.Context()

	items, err := r.Items(ctx, "g")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, []string{"teen"}, items[0].Audiences)
	assert.JSONEq(t, `{"k":1}`, string(items[0].Meta))
	assert.True(t, mr.Exists("ritme-go:catalog.group.g"))
	assert.Positive(t, mr.TTL("ritme-go:catalog.group.g"))

	again, err := r.Items(ctx, "g")
	require.NoError(t, err)
	assert.Equal(t, items, again)
	assert.Equal(t, 1, f.calls, "second read is served from the cache")

	require.NoError(t, r.Flush(ctx, "g"))
	_, err = r.Items(ctx, "g")
	require.NoError(t, err)
	assert.Equal(t, 2, f.calls)

	// An empty group is cached as [] too.
	f.rows = nil
	empty, err := r.Items(ctx, "none")
	require.NoError(t, err)
	assert.Empty(t, empty)

	f.err = errors.New("db down")
	_, err = NewReader(f, nil, 0, nil).Items(ctx, "g")
	require.Error(t, err)
}
