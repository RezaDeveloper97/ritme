<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Sitemap\CategorySitemapProvider;
use App\Domain\Blog\Sitemap\PostSitemapProvider;
use App\Domain\Media\Models\Media;
use App\Domain\Seo\Models\SeoMeta;
use Illuminate\Support\Carbon;

afterEach(fn () => Carbon::setTestNow());

it('lists indexable published posts with lastmod and cover image', function (): void {
    config(['app.url' => 'https://ritme.ir']);
    url()->forceRootUrl('https://ritme.ir');
    url()->forceScheme('https');
    $media = Media::query()->create([
        'disk' => 'public', 'directory' => '2026/10/abc', 'filename' => 'cover.jpg', 'mime' => 'image/jpeg',
        'size' => 100, 'width' => 1200, 'height' => 630, 'alt' => 'درد پریود', 'hash' => str_repeat('a', 64),
    ]);

    Carbon::setTestNow('2026-10-04 10:00:00');
    $post = Post::factory()->published(now()->subDays(2))->create(['slug' => 'period-pain', 'cover_media_id' => $media->id]);
    Carbon::setTestNow('2026-10-04 12:00:00');
    $post->update(['body' => '<p>به‌روز</p>']);

    $hidden = Post::factory()->published()->create(['slug' => 'hidden']);
    SeoMeta::query()->create(['seoable_type' => $hidden->getMorphClass(), 'seoable_id' => $hidden->id, 'robots' => 'noindex, follow']);
    $excluded = Post::factory()->published()->create(['slug' => 'excluded']);
    SeoMeta::query()->create(['seoable_type' => $excluded->getMorphClass(), 'seoable_id' => $excluded->id, 'sitemap_include' => false]);
    $withMeta = Post::factory()->published()->create(['slug' => 'with-meta']);
    SeoMeta::query()->create(['seoable_type' => $withMeta->getMorphClass(), 'seoable_id' => $withMeta->id, 'title' => 'x']);
    Post::factory()->create(['slug' => 'draft']);

    $provider = app(PostSitemapProvider::class);
    $entries = collect($provider->entries())->keyBy('loc');

    expect($provider->key())->toBe('posts')
        ->and($provider->count())->toBe(2)
        ->and($entries->keys()->all())->toEqualCanonicalizing(['https://ritme.ir/blog/period-pain', 'https://ritme.ir/blog/with-meta'])
        ->and($entries['https://ritme.ir/blog/period-pain']->lastmod?->toDateTimeString())->toBe('2026-10-04 12:00:00')
        ->and($entries['https://ritme.ir/blog/period-pain']->imageUrl)->toBe('https://ritme.ir/media/2026/10/abc/cover.jpg')
        ->and($entries['https://ritme.ir/blog/period-pain']->imageTitle)->toBe('درد پریود')
        ->and($post->getMorphClass())->toBe('blog_post');
});

it('refreshes after seo_meta or post changes (sitemap namespace)', function (): void {
    $post = Post::factory()->published()->create();
    $provider = app(PostSitemapProvider::class);
    expect($provider->count())->toBe(1);

    SeoMeta::query()->create(['seoable_type' => $post->getMorphClass(), 'seoable_id' => $post->id, 'robots' => 'noindex']);
    expect($provider->count())->toBe(0);

    Post::factory()->published()->create();
    expect($provider->count())->toBe(1);
});

it('lists only categories with published posts, lastmod from their newest post', function (): void {
    config(['app.url' => 'https://ritme.ir']);
    url()->forceRootUrl('https://ritme.ir');
    url()->forceScheme('https');
    $full = Category::factory()->create(['slug' => 'cycle']);
    Category::factory()->create(['slug' => 'empty']);
    $draftOnly = Category::factory()->create(['slug' => 'draft-only']);
    Post::factory()->published(Carbon::parse('2026-09-01 09:00:00'))->create(['category_id' => $full->id]);
    Post::factory()->published(Carbon::parse('2026-09-20 09:00:00'))->create(['category_id' => $full->id]);
    Post::factory()->create(['category_id' => $draftOnly->id]);

    $entries = app(CategorySitemapProvider::class)->entries();

    expect($entries)->toHaveCount(1)
        ->and($entries[0]->loc)->toBe('https://ritme.ir/blog/category/cycle')
        ->and($entries[0]->lastmod?->toDateTimeString())->toBe('2026-09-20 09:00:00')
        ->and(app(CategorySitemapProvider::class)->key())->toBe('blog-categories');
});
