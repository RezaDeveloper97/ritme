<?php

declare(strict_types=1);

namespace App\Domain\Blog\Actions;

use App\Domain\Blog\Models\Post;
use Illuminate\Database\ConnectionInterface;

/**
 * Creates or updates a post from editor input (admin): only fillable attributes are taken, the save runs through
 * PostObserver (sanitised body/sources, slug + history, schedule normalisation, cache bumps) and the tags are
 * replaced through SyncPostTags (pivot writes bump the caches). One transaction.
 */
final class SavePost
{
    public function __construct(
        private readonly SyncPostTags $syncTags,
        private readonly ConnectionInterface $db,
    ) {}

    /**
     * @param  array<string, mixed>  $attributes
     * @param  list<int>|null  $tagIds  null = leave the tags as they are
     */
    public function handle(Post $post, array $attributes, ?array $tagIds = null): Post
    {
        return $this->db->transaction(function () use ($post, $attributes, $tagIds): Post {
            $post->fill(array_intersect_key($attributes, array_flip($post->getFillable())));
            $post->save();

            if ($tagIds !== null) {
                $this->syncTags->handle($post, $tagIds);
            }

            return $post;
        });
    }
}
