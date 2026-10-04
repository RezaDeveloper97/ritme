<?php

declare(strict_types=1);

use App\Domain\Blog\Actions\FlushPostViews;
use App\Domain\Blog\Actions\RecordPostView;
use App\Domain\Blog\Models\Post;
use App\Support\Cache\NamespaceVersions;

it('counts views in the cache and flushes them in one batch without touching updated_at or caches', function (): void {
    $a = Post::factory()->published()->create();
    $b = Post::factory()->published()->create();
    $updatedAt = $a->fresh()?->updated_at?->toDateTimeString();
    $record = app(RecordPostView::class);

    $record->handle($a->id);
    $record->handle($a->id);
    $record->handle($b->id);

    expect($a->fresh()?->views)->toBe(0);

    $blog = app(NamespaceVersions::class)->version('blog');
    $this->travel(5)->minutes();
    $this->artisan('blog:flush-views')->expectsOutputToContain('2 post(s)')->assertSuccessful();

    expect($a->fresh()?->views)->toBe(2)
        ->and($b->fresh()?->views)->toBe(1)
        ->and($a->fresh()?->updated_at?->toDateTimeString())->toBe($updatedAt)
        ->and(app(NamespaceVersions::class)->version('blog'))->toBe($blog)
        ->and(app(FlushPostViews::class)->handle())->toBe(0); // nothing pending any more

    $record->handle($a->id);
    app(FlushPostViews::class)->handle();

    expect($a->fresh()?->views)->toBe(3);
});
