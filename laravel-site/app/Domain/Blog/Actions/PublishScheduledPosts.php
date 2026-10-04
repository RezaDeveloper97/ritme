<?php

declare(strict_types=1);

namespace App\Domain\Blog\Actions;

use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Post;
use Carbon\CarbonInterface;
use Illuminate\Support\Carbon;

/**
 * Publishes every scheduled post whose publish time has come. Runs every minute from the scheduler
 * (`blog:publish-scheduled`; cPanel cron → `schedule:run`). Each post is saved through Eloquent, so PostObserver
 * normalises it and bumps `blog`, `sitemap` and `pages`.
 */
final class PublishScheduledPosts
{
    /**
     * @return int number of posts published
     */
    public function handle(?CarbonInterface $now = null): int
    {
        $now = Carbon::instance($now ?? Carbon::now());
        $published = 0;

        Post::query()
            ->where('status', PostStatus::Scheduled->value)
            ->whereNotNull('published_at')
            ->where('published_at', '<=', $now)
            ->orderBy('published_at')
            ->each(static function (Post $post) use (&$published): void {
                $post->status = PostStatus::Published;
                $post->save();
                $published++;
            });

        return $published;
    }
}
