<?php

declare(strict_types=1);

namespace App\Domain\Blog\Actions;

use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Post;
use Illuminate\Database\ConnectionInterface;

/**
 * Bulk publish / unpublish (admin). Publishing keeps a post's own date (a future date becomes «scheduled» through
 * PostSchedule) or uses now; unpublishing turns it back into a draft. Every post is saved through Eloquent, so the
 * observer normalises it and bumps the caches.
 */
final class SetPostsStatus
{
    public function __construct(private readonly ConnectionInterface $db) {}

    /**
     * @param  iterable<Post>  $posts
     * @return list<int> ids of the posts that changed
     */
    public function handle(iterable $posts, PostStatus $status): array
    {
        return $this->db->transaction(static function () use ($posts, $status): array {
            $changed = [];

            foreach ($posts as $post) {
                $post->status = $status;
                if ($post->isDirty()) {
                    $post->save();
                    $changed[] = $post->id;
                }
            }

            return $changed;
        });
    }
}
