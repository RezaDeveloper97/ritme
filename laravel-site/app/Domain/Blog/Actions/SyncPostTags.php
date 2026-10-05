<?php

declare(strict_types=1);

namespace App\Domain\Blog\Actions;

use App\Domain\Blog\Models\Post;
use App\Support\Cache\NamespaceBumper;

/**
 * Replaces a post's tags. Pivot writes fire no model events, so the caches are bumped here when anything changed.
 */
final class SyncPostTags
{
    public function __construct(private readonly NamespaceBumper $bumper) {}

    /**
     * @param  array<int|string>  $tagIds  any keys; duplicates are dropped
     */
    public function handle(Post $post, array $tagIds): void
    {
        $changes = $post->tags()->sync(array_values(array_unique(array_map(intval(...), $tagIds))));

        if (array_filter($changes) !== []) {
            $this->bumper->bumpFor($post, ['blog', 'sitemap']);
        }
    }
}
