<?php

declare(strict_types=1);

namespace App\Domain\Blog\Support;

use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Post;
use Carbon\CarbonInterface;
use Illuminate\Support\Carbon;

/**
 * Keeps status, published_at and updated_content_at consistent on save:
 *
 *  - published without a date → published now; published with a future date → scheduled;
 *  - scheduled without a date → draft; scheduled with a past date → published (the scheduler saves it this way);
 *  - first publication sets updated_content_at = published_at; later title/excerpt/body changes of a published post
 *    set it to now (dateModified), unless the editor set it explicitly.
 */
final class PostSchedule
{
    public static function normalize(Post $post, ?CarbonInterface $now = null): void
    {
        $now = Carbon::instance($now ?? Carbon::now());
        $status = $post->status;

        if ($status === PostStatus::Published) {
            if ($post->published_at === null) {
                $post->published_at = $now->copy();
            } elseif ($post->published_at->greaterThan($now)) {
                $post->status = PostStatus::Scheduled;
            }
        } elseif ($status === PostStatus::Scheduled) {
            if ($post->published_at === null) {
                $post->status = PostStatus::Draft;
            } elseif ($post->published_at->lessThanOrEqualTo($now)) {
                $post->status = PostStatus::Published;
            }
        }

        if ($post->status !== PostStatus::Published || $post->isDirty('updated_content_at')) {
            return;
        }

        $original = $post->getOriginal('status');
        $wasPublished = $post->exists && ($original === PostStatus::Published || $original === PostStatus::Published->value);

        if (! $wasPublished) {
            $post->updated_content_at ??= $post->published_at?->copy();
        } elseif ($post->isDirty(['title', 'excerpt', 'body'])) {
            $post->updated_content_at = $now->copy();
        }
    }
}
