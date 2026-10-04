<?php

declare(strict_types=1);

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;

it('ranks related posts by shared tags, category and recency', function (): void {
    [$cycle, $pregnancy] = Category::factory()->count(2)->create();
    [$pain, $cramps, $sleep] = Tag::factory()->count(3)->create();

    $post = Post::factory()->published()->create(['category_id' => $cycle->id, 'life_stage' => null]);
    $post->tags()->sync([$pain->id, $cramps->id]);

    $twoTags = Post::factory()->published(now()->subDays(200))->create(['category_id' => $pregnancy->id, 'life_stage' => null]);
    $twoTags->tags()->sync([$pain->id, $cramps->id]);

    $oneTagFresh = Post::factory()->published(now()->subDay())->create(['category_id' => $pregnancy->id, 'life_stage' => null]);
    $oneTagFresh->tags()->sync([$pain->id]);

    $oneTagOld = Post::factory()->published(now()->subDays(300))->create(['category_id' => $pregnancy->id, 'life_stage' => null]);
    $oneTagOld->tags()->sync([$pain->id, $sleep->id]);

    $sameCategory = Post::factory()->published(now()->subDays(2))->create(['category_id' => $cycle->id, 'life_stage' => null]);

    $unrelated = Post::factory()->published(now()->subHours(3))->create(['category_id' => $pregnancy->id, 'life_stage' => null]);
    $draft = Post::factory()->create(['category_id' => $cycle->id]);
    $draft->tags()->sync([$pain->id, $cramps->id]);

    $ids = array_map(static fn ($card): int => $card->id, app(PostRepository::class)->related($post->id, 4));

    // 2 shared tags (6 + old) > 1 tag fresh (3 + ~2) ≈ same category fresh (2 + ~2) > 1 tag old (3 + ~0.5)
    expect($ids)->toBe([$twoTags->id, $oneTagFresh->id, $sameCategory->id, $oneTagOld->id])
        ->and($ids)->not->toContain($post->id, $draft->id, $unrelated->id);
});

it('tops up with the latest posts when too few are related', function (): void {
    $post = Post::factory()->published()->create();
    $latest = Post::factory()->published(now()->subHour())->create();
    $older = Post::factory()->published(now()->subDays(5))->create();
    Post::factory()->published(now()->subDays(9))->create();

    $ids = array_map(static fn ($card): int => $card->id, app(PostRepository::class)->related($post->id, 2));

    expect($ids)->toBe([$latest->id, $older->id]);
});

it('returns nothing for an unknown post', function (): void {
    expect(app(PostRepository::class)->related(999))->toBe([]);
});
