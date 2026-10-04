<?php

declare(strict_types=1);

namespace App\Domain\Blog\Actions;

use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Support\ViewCounter;

/**
 * Adds the cached view counts to `blog_posts.views` (scheduler, every five minutes). Writes through the query
 * builder: no updated_at change and no observer, so a view never invalidates caches.
 */
final class FlushPostViews
{
    public function __construct(private readonly ViewCounter $counter) {}

    /**
     * @return int number of posts updated
     */
    public function handle(): int
    {
        $updated = 0;
        foreach ($this->counter->drain() as $postId => $views) {
            $updated += Post::query()->whereKey($postId)->toBase()->increment('views', $views) > 0 ? 1 : 0;
        }

        return $updated;
    }
}
