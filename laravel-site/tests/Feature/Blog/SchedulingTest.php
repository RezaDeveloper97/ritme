<?php

declare(strict_types=1);

use App\Domain\Blog\Actions\PublishScheduledPosts;
use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Post;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Support\Carbon;

afterEach(fn () => Carbon::setTestNow());

it('normalises status and publish date on save', function (): void {
    Carbon::setTestNow('2026-10-04 10:00:00');

    $now = Post::factory()->create(['status' => PostStatus::Published, 'published_at' => null]);
    $future = Post::factory()->create(['status' => PostStatus::Published, 'published_at' => now()->addHour()]);
    $undated = Post::factory()->create(['status' => PostStatus::Scheduled, 'published_at' => null]);

    expect($now->status)->toBe(PostStatus::Published)
        ->and($now->published_at?->toDateTimeString())->toBe('2026-10-04 10:00:00')
        ->and($now->updated_content_at?->toDateTimeString())->toBe('2026-10-04 10:00:00')
        ->and($future->status)->toBe(PostStatus::Scheduled)
        ->and($undated->status)->toBe(PostStatus::Draft);
});

it('publishes due scheduled posts and leaves future ones', function (): void {
    Carbon::setTestNow('2026-10-04 10:00:00');
    $due = Post::factory()->scheduled(now()->addMinutes(5))->create(['slug' => 'due']);
    $later = Post::factory()->scheduled(now()->addDay())->create(['slug' => 'later']);

    $posts = app(PostRepository::class);
    expect($posts->findPublishedBySlug('due'))->toBeNull()
        ->and($posts->latest()->total)->toBe(0); // now cached

    Carbon::setTestNow('2026-10-04 10:06:00');
    $count = app(PublishScheduledPosts::class)->handle();

    expect($count)->toBe(1)
        ->and($due->fresh()?->status)->toBe(PostStatus::Published)
        ->and($due->fresh()?->updated_content_at?->toDateTimeString())->toBe('2026-10-04 10:05:00')
        ->and($later->fresh()?->status)->toBe(PostStatus::Scheduled)
        ->and($posts->findPublishedBySlug('due')?->id)->toBe($due->id) // observer bumped `blog`
        ->and($posts->latest()->total)->toBe(1);
});

it('runs from the artisan command', function (): void {
    Post::factory()->create(['status' => PostStatus::Scheduled, 'published_at' => now()->addMinute()]);
    Carbon::setTestNow(now()->addMinutes(2));

    $this->artisan('blog:publish-scheduled')->expectsOutputToContain('Published 1')->assertSuccessful();

    expect(Post::query()->where('status', 'published')->count())->toBe(1);
});

it('is scheduled every minute (cPanel cron) and flushes views every five minutes', function (): void {
    $events = collect(app(Schedule::class)->events())->mapWithKeys(
        static fn ($event): array => [trim(str_replace(["'", '"'], '', (string) strstr((string) $event->command, 'blog:'))) => $event->expression],
    );

    expect($events->get('blog:publish-scheduled'))->toBe('* * * * *')
        ->and($events->get('blog:flush-views'))->toBe('*/5 * * * *');
});

it('moves updated_content_at only on content edits of published posts', function (): void {
    Carbon::setTestNow('2026-10-01 09:00:00');
    $post = Post::factory()->published(now()->subDay())->create();
    $first = $post->updated_content_at?->toDateTimeString();

    Carbon::setTestNow('2026-10-03 09:00:00');
    $post->update(['is_featured' => true]);
    expect($post->fresh()?->updated_content_at?->toDateTimeString())->toBe($first);

    $post->update(['body' => '<p>متن تازه</p>']);
    expect($post->fresh()?->updated_content_at?->toDateTimeString())->toBe('2026-10-03 09:00:00');
});
