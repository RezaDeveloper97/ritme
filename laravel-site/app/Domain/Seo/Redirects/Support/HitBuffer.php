<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Support;

use Closure;
use Illuminate\Contracts\Cache\Factory as CacheFactory;
use Illuminate\Contracts\Cache\LockProvider;
use Illuminate\Contracts\Cache\Repository;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Cheap, batched counters for redirect hits and 404s (same idea as Blog\Support\ViewCounter): a hit increments a
 * cache counter (no database write); the first hit since the last flush adds the id to the bucket's pending list
 * together with its metadata. FlushRedirectStats drains the buffer into the database every few minutes.
 *
 * Storm protection: a bucket holds at most MAX_PENDING ids between two flushes — further new ids are dropped (a
 * scanner probing thousands of random paths cannot grow the buffer or the table). Best-effort: a cache clear loses
 * unflushed counts.
 */
final class HitBuffer
{
    public const MAX_PENDING = 500;

    private const PREFIX = 'rt:seo-hits:';

    private const META_TTL = 604800; // 7 days

    private readonly Repository $store;

    public function __construct(CacheFactory $cache, Config $config)
    {
        $name = $config->get('cacheaside.store');
        $this->store = $cache->store(is_string($name) && $name !== '' ? $name : null);
    }

    /**
     * @param  array<string, string|null>  $meta  stored with the first hit of a flush window
     */
    public function hit(string $bucket, string $id, array $meta = []): bool
    {
        $key = $this->counterKey($bucket, $id);
        $count = $this->store->increment($key);

        if ($count === false) { // stores that don't create missing keys on increment (database)
            $count = $this->store->add($key, 1) ? 1 : (int) $this->store->increment($key);
        }

        if ((int) $count !== 1) {
            return true;
        }

        if (! $this->markPending($bucket, $id)) {
            $this->store->forget($key);

            return false;
        }

        if ($meta !== []) {
            $this->store->put($this->metaKey($bucket, $id), $meta, self::META_TTL);
        }

        return true;
    }

    /**
     * Takes every pending count of a bucket out of the cache.
     *
     * @return array<string, array{hits: int, meta: array<string, string|null>}>
     */
    public function drain(string $bucket): array
    {
        /** @var list<string> $pending */
        $pending = $this->locked($bucket, fn (): array => (array) $this->store->pull($this->pendingKey($bucket), []));

        $drained = [];
        foreach (array_unique($pending) as $id) {
            $id = (string) $id;
            $key = $this->counterKey($bucket, $id);
            $hits = (int) $this->store->get($key, 0);
            if ($hits <= 0) {
                continue;
            }

            $meta = $this->store->get($this->metaKey($bucket, $id), []);
            $drained[$id] = ['hits' => $hits, 'meta' => is_array($meta) ? $meta : []];

            $left = $this->store->decrement($key, $hits);
            if (is_int($left) && $left > 0) {
                $this->markPending($bucket, $id); // hits that arrived meanwhile wait for the next flush
            }
        }

        return $drained;
    }

    public function pending(string $bucket, string $id): int
    {
        return (int) $this->store->get($this->counterKey($bucket, $id), 0);
    }

    private function markPending(string $bucket, string $id): bool
    {
        return $this->locked($bucket, function () use ($bucket, $id): bool {
            /** @var list<string> $pending */
            $pending = (array) $this->store->get($this->pendingKey($bucket), []);
            if (in_array($id, $pending, true)) {
                return true;
            }
            if (count($pending) >= self::MAX_PENDING) {
                return false;
            }

            $pending[] = $id;
            $this->store->forever($this->pendingKey($bucket), $pending);

            return true;
        });
    }

    private function counterKey(string $bucket, string $id): string
    {
        return self::PREFIX.$bucket.':'.$id;
    }

    private function metaKey(string $bucket, string $id): string
    {
        return self::PREFIX.$bucket.':meta:'.$id;
    }

    private function pendingKey(string $bucket): string
    {
        return self::PREFIX.$bucket.':pending';
    }

    /**
     * @template T
     *
     * @param  Closure(): T  $callback
     * @return T
     */
    private function locked(string $bucket, Closure $callback): mixed
    {
        $store = $this->store->getStore();
        if (! $store instanceof LockProvider) {
            return $callback();
        }

        return $store->lock(self::PREFIX.$bucket.':lock', 5)->block(3, $callback);
    }
}
