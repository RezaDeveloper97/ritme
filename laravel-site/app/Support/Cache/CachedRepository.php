<?php

declare(strict_types=1);

namespace App\Support\Cache;

use Closure;
use DateInterval;

/**
 * Base for Cached* repository decorators (see docs/ARCHITECTURE.md). The subclass receives the Eloquent
 * implementation as `$inner`, declares its namespace and wraps each read:
 *
 *     final class CachedPostRepository extends CachedRepository implements PostRepository
 *     {
 *         public function __construct(private readonly PostRepository $inner, CacheAside $cache)
 *         {
 *             parent::__construct($cache);
 *         }
 *
 *         protected function namespace(): string { return 'blog'; }
 *
 *         public function findBySlug(string $slug): ?PostData
 *         {
 *             return $this->remember(['post', $slug], fn () => $this->inner->findBySlug($slug));
 *         }
 *     }
 *
 * Writes stay in the Eloquent implementation; invalidation happens by bumping the namespace on model
 * changes (a CacheBumpingObserver in the context, which also bumps `pages`).
 */
abstract class CachedRepository
{
    public function __construct(protected readonly CacheAside $cache) {}

    abstract protected function namespace(): string;

    /**
     * @template T
     *
     * @param  string|int|list<string|int>  $key
     * @param  Closure(): T  $loader
     * @param  int|DateInterval|null  $ttl  null = the namespace's default TTL
     * @return T
     */
    protected function remember(string|int|array $key, Closure $loader, int|DateInterval|null $ttl = null, bool $cacheNull = false): mixed
    {
        return $this->cache->remember($this->key($key), $ttl, $loader, $cacheNull);
    }

    /**
     * @template T
     *
     * @param  string|int|list<string|int>  $key
     * @param  Closure(): T  $loader
     * @return T
     */
    protected function rememberForever(string|int|array $key, Closure $loader, bool $cacheNull = false): mixed
    {
        return $this->cache->rememberForever($this->key($key), $loader, $cacheNull);
    }

    /**
     * @param  string|int|list<string|int>  $key
     */
    protected function forget(string|int|array $key): bool
    {
        return $this->cache->forget($this->key($key));
    }

    /**
     * @param  string|int|list<string|int>  $key
     */
    protected function key(string|int|array $key): CacheKey
    {
        return CacheKey::make($this->namespace(), ...(is_array($key) ? $key : [$key]));
    }
}
