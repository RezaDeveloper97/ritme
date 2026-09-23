package languages

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func decode(t *testing.T, s string) any {
	t.Helper()
	v, err := phpval.Decode([]byte(s))
	require.NoError(t, err)
	return v
}

func TestFlattenUnflattenPrune(t *testing.T) {
	v := decode(t, `{"a":{"b":"x","c":["p","q"],"d":{}},"n":5,"t":true,"z":null}`)
	flat := Flatten(v)
	assert.Equal(t, map[string]string{"a.b": "x", "a.c.0": "p", "a.c.1": "q", "a.d": "", "n": "5", "t": "1", "z": ""}, flat)

	keys := []string{"z", "a.c.1", "a.c.0", "a.b", "n", "10", "9"}
	SortKeys(keys)
	assert.Equal(t, []string{"9", "10", "a.b", "a.c.0", "a.c.1", "n", "z"}, keys)

	back := Unflatten(map[string]string{"a.b": "x", "a.c.0": "p", "a.c.1": "q", "e": ""}, []string{"a.b", "a.c.0", "a.c.1", "e"})
	b, err := json.Marshal(prune(back))
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":{"b":"x","c":["p","q"]}}`, string(b))
}

func TestBundlesWriteAndGuards(t *testing.T) {
	dir := t.TempDir()
	seed := fstest.MapFS{
		"fa/common.json": {Data: []byte(`{"hello":"سلام","nested":{"a":"الف"}}`)},
		"fa/home.json":   {Data: []byte(`{}`)},
		"en/common.json": {Data: []byte(`{"hello":"Hi"}`)},
	}
	langFS := fstest.MapFS{"fa/validation.json": {Data: []byte(`{"required":"x"}`)}, "fa/readme.txt": {Data: []byte("no")}}
	b := NewBundles(i18n.NewTranslationStore(seed, dir), dir, langFS)

	n, err := b.GenerateFor("ar", "en", "fa")
	require.NoError(t, err)
	assert.Equal(t, 1, n, "empty namespaces are skipped")
	raw, err := os.ReadFile(filepath.Join(dir, "app", "translations", "ar", "common.json"))
	require.NoError(t, err)
	assert.Equal(t, "{\n    \"hello\": \"Hi\",\n    \"nested\": {\n        \"a\": \"الف\"\n    }\n}\n", string(raw),
		"pretty JSON, raw UTF-8, backfilled from the default")

	copied, err := b.CopyLangFiles("ar", "fa")
	require.NoError(t, err)
	assert.Equal(t, 1, copied)
	copied, err = b.CopyLangFiles("ar", "fa")
	require.NoError(t, err)
	assert.Zero(t, copied, "never clobbers")
	copied, err = b.CopyLangFiles("fa", "fa")
	require.NoError(t, err)
	assert.Zero(t, copied)

	for _, bad := range []string{"../x", "AR", "a", "ar/..", ""} {
		assert.ErrorIs(t, b.WriteNamespace(bad, "common", map[string]any{}), errUnsafe, bad)
		assert.ErrorIs(t, b.DeleteFor(bad), errUnsafe, bad)
	}
	for _, bad := range []string{"../common", ".hidden", "a/b", ""} {
		assert.ErrorIs(t, b.WriteNamespace("ar", bad, map[string]any{}), errUnsafe, bad)
	}

	require.NoError(t, b.DeleteFor("ar"))
	assert.NoDirExists(t, filepath.Join(dir, "app", "translations", "ar"))
	assert.NoDirExists(t, filepath.Join(dir, "app", "lang", "ar"))
	assert.ErrorIs(t, NewBundles(i18n.NewTranslationStore(seed, ""), "", langFS).WriteNamespace("ar", "common", nil), errUnsafe)
}

func TestLaravelCacheConfig(t *testing.T) {
	mr := miniredis.RunT(t)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	env := map[string]string{}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	l, err := newLaravelCache(c, lookup)
	require.NoError(t, err)
	assert.Equal(t, "ritme-database-ritme-cache-languages.registry", l.Key())
	require.NoError(t, mr.DB(1).Set(l.Key(), "x"))
	require.NoError(t, mr.Set(l.Key(), "db0 is someone else's"))
	require.NoError(t, l.Forget(t.Context()))
	assert.False(t, mr.DB(1).Exists(l.Key()))
	assert.True(t, mr.Exists(l.Key()), "only Laravel's cache database is touched")

	env["LARAVEL_CACHE_KEY_PREFIX"], env["LARAVEL_CACHE_REDIS_DB"] = "app-", "3"
	l, err = newLaravelCache(c, lookup)
	require.NoError(t, err)
	require.NoError(t, mr.DB(3).Set("app-languages.registry", "x"))
	require.NoError(t, l.Forget(t.Context()))
	assert.False(t, mr.DB(3).Exists("app-languages.registry"))

	env["LARAVEL_CACHE_REDIS_DB"] = "x"
	_, err = newLaravelCache(c, lookup)
	require.Error(t, err)

	env["LARAVEL_CACHE_FLUSH"] = "false"
	l, err = newLaravelCache(c, lookup)
	require.NoError(t, err)
	assert.Nil(t, l)
	require.NoError(t, l.Forget(t.Context()), "nil flusher is a no-op")

	l, err = newLaravelCache(nil, lookup)
	require.NoError(t, err)
	assert.Nil(t, l)
}
