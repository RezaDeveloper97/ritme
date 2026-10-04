<?php

declare(strict_types=1);

namespace App\Domain\Blog\Actions;

use App\Domain\Blog\Support\ViewCounter;

/**
 * Counts one article view in the cache (no database write per request). Flushed by FlushPostViews.
 */
final class RecordPostView
{
    public function __construct(private readonly ViewCounter $counter) {}

    public function handle(int $postId): void
    {
        $this->counter->hit($postId);
    }
}
