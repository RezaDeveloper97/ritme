<?php

declare(strict_types=1);

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\PostSlug;

it('derives a Persian slug from the title and keeps slugs unique', function (): void {
    $a = Post::factory()->create(['title' => 'درد پریود؛ کی عادی است؟', 'slug' => null]);
    $b = Post::factory()->create(['title' => 'درد پریود؛ کی عادی است؟', 'slug' => '']);
    $c = Post::factory()->create(['slug' => 'Custom Slug!']);

    expect($a->slug)->toBe('درد-پریود-کی-عادی-است')
        ->and($b->slug)->toBe('درد-پریود-کی-عادی-است-2')
        ->and($c->slug)->toBe('custom-slug');
});

it('records the previous slug of a published post and resolves it to the current one', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'old-slug']);

    $post->update(['slug' => 'new-slug']);
    $post->update(['slug' => 'newest-slug']);

    $posts = app(PostRepository::class);

    expect(PostSlug::query()->where('post_id', $post->id)->orderBy('slug')->pluck('slug')->all())->toBe(['new-slug', 'old-slug'])
        ->and($posts->currentSlugFor('old-slug'))->toBe('newest-slug')
        ->and($posts->currentSlugFor('new-slug'))->toBe('newest-slug')
        ->and($posts->currentSlugFor('unknown'))->toBeNull()
        ->and($posts->findPublishedBySlug('old-slug'))->toBeNull()
        ->and($posts->findPublishedBySlug('newest-slug')?->id)->toBe($post->id);
});

it('takes a slug back out of history when the post reuses it', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'first']);
    $post->update(['slug' => 'second']);
    $post->update(['slug' => 'first']);

    expect(PostSlug::query()->pluck('slug')->all())->toBe(['second']);
});

it('does not keep history for drafts that were never published', function (): void {
    $post = Post::factory()->create(['slug' => 'draft-a']);
    $post->update(['slug' => 'draft-b']);

    expect(PostSlug::query()->count())->toBe(0);
});

it('never hands another post a slug that still redirects', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'taken']);
    $post->update(['slug' => 'moved']);

    $other = Post::factory()->create(['slug' => 'taken']);

    expect($other->slug)->toBe('taken-2')
        ->and(app(PostRepository::class)->currentSlugFor('taken'))->toBe('moved');
});

it('does not redirect old slugs of unpublished posts', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'live']);
    $post->update(['slug' => 'renamed']);
    $post->update(['status' => 'archived']);

    expect(app(PostRepository::class)->currentSlugFor('live'))->toBeNull();
});
