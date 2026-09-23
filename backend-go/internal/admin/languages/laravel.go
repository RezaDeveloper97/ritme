package languages

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
)

// LaravelCache forgets Laravel's LanguageRegistry cache entry (Cache::forget('languages.registry'))
// while both stacks serve traffic, so a language edited here reaches the Laravel API at
// once instead of never (Laravel caches it with rememberForever).
//
// Laravel's redis cache store writes the key as <REDIS_PREFIX><CACHE_PREFIX>languages.registry
// on the `cache` connection (REDIS_CACHE_DB, default 1). With APP_NAME=Ritme and no
// overrides that is `ritme-database-ritme-cache-languages.registry` in DB 1. The only
// thing Go does with it is DEL — it never reads Laravel's PHP-serialized value.
//
// Environment (only needed when Laravel's own settings differ from its defaults):
//
//	LARAVEL_CACHE_KEY_PREFIX  <REDIS_PREFIX><CACHE_PREFIX> of the Laravel app (default "ritme-database-ritme-cache-")
//	LARAVEL_CACHE_REDIS_DB    Laravel's REDIS_CACHE_DB (default 1)
//	LARAVEL_CACHE_FLUSH       "false" disables the flush (after the Laravel API is gone)
type LaravelCache struct {
	rdb *redis.Client
	key string
}

// Laravel cache defaults (APP_NAME=Ritme, REDIS_CACHE_DB unset).
const (
	DefaultLaravelKeyPrefix = "ritme-database-ritme-cache-"
	DefaultLaravelCacheDB   = 1
)

// NewLaravelCache builds the flusher on the same Redis server as c (address and
// credentials), selecting Laravel's cache database. It returns nil when c is nil or the
// flush is disabled.
func NewLaravelCache(c *cache.Client) (*LaravelCache, error) {
	return newLaravelCache(c, os.LookupEnv)
}

func newLaravelCache(c *cache.Client, lookup func(string) (string, bool)) (*LaravelCache, error) {
	if c == nil {
		return nil, nil
	}
	get := func(k string) string { v, _ := lookup(k); return strings.TrimSpace(v) }
	switch strings.ToLower(get("LARAVEL_CACHE_FLUSH")) {
	case "false", "0", "no", "off":
		return nil, nil
	}
	prefix := DefaultLaravelKeyPrefix
	if v, ok := lookup("LARAVEL_CACHE_KEY_PREFIX"); ok {
		prefix = strings.TrimSpace(v)
	}
	db := DefaultLaravelCacheDB
	if v := get("LARAVEL_CACHE_REDIS_DB"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("languages: LARAVEL_CACHE_REDIS_DB %q is not a database number", v)
		}
		db = n
	}
	opts := *c.Redis().Options()
	opts.DB = db
	return &LaravelCache{rdb: redis.NewClient(&opts), key: prefix + i18n.CacheKey}, nil
}

// Key is the Redis key that is deleted.
func (l *LaravelCache) Key() string { return l.key }

// Forget deletes Laravel's cached language list.
func (l *LaravelCache) Forget(ctx context.Context) error {
	if l == nil {
		return nil
	}
	if err := l.rdb.Del(ctx, l.key).Err(); err != nil {
		return fmt.Errorf("languages: forget Laravel %s: %w", i18n.CacheKey, err)
	}
	return nil
}
