<?php

declare(strict_types=1);

namespace App\Domain\Blog\Support;

use Illuminate\Contracts\Cache\Factory as CacheFactory;
use Illuminate\Contracts\Cache\LockProvider;
use Illuminate\Contracts\Cache\Repository;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Cheap post view counter: a view increments a per-post cache counter (no database write); the first view since the
 * last flush adds the post to a pending list. FlushPostViews moves the counts into `blog_posts.views` in one batch.
 * Uses the cache-aside store. Counts are best-effort (a cache clear loses unflushed views).
 */
final class ViewCounter
{
    private const PREFIX = 'rt:blog-views:';

    private readonly Repository $store;

    public function __construct(CacheFactory $cache, Config $config)
    {
        $name = $config->get('cacheaside.store');
        $this->store = $cache->store(is_string($name) && $name !== '' ? $name : null);
    }

    public function hit(int $postId): void
    {
        $key = self::PREFIX.$postId;
        $count = $this->store->increment($key);

        if ($count === false) { // stores that don't create missing keys on increment (database)
            $count = $this->store->add($key, 1) ? 1 : (int) $this->store->increment($key);
        }

        if ((int) $count === 1) {
            $this->markPending($postId);
        }
    }

    /**
     * Takes every pending count out of the cache.
     *
     * @return array<int, int> post id => views since the last flush
     */
    public function drain(): array
    {
        /** @var list<int> $pending */
        $pending = $this->locked(fn (): array => (array) $this->store->pull(self::PREFIX.'pending', []));

        $counts = [];
        foreach (array_unique($pending) as $postId) {
            $key = self::PREFIX.$postId;
            $views = (int) $this->store->get($key, 0);
            if ($views <= 0) {
                continue;
            }

            $counts[(int) $postId] = $views;
            $left = $this->store->decrement($key, $views);
            if (is_int($left) && $left > 0) {
                $this->markPending((int) $postId); // views that arrived meanwhile wait for the next flush
            }
        }

        return $counts;
    }

    public function pending(int $postId): int
    {
        return (int) $this->store->get(self::PREFIX.$postId, 0);
    }

    private function markPending(int $postId): void
    {
        $this->locked(function () use ($postId): void {
            /** @var list<int> $pending */
            $pending = (array) $this->store->get(self::PREFIX.'pending', []);
            if (! in_array($postId, $pending, true)) {
                $pending[] = $postId;
                $this->store->forever(self::PREFIX.'pending', $pending);
            }
        });
    }

    /**
     * @template T
     *
     * @param  \Closure(): T  $callback
     * @return T
     */
    private function locked(\Closure $callback): mixed
    {
        $store = $this->store->getStore();
        if (! $store instanceof LockProvider) {
            return $callback();
        }

        return $store->lock(self::PREFIX.'lock', 5)->block(3, $callback);
    }
}
