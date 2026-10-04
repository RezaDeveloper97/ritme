<?php

declare(strict_types=1);

use App\Domain\Blog\Actions\SyncPostTags;
use App\Domain\Blog\Contracts\AuthorRepository;
use App\Domain\Blog\Contracts\CategoryRepository;
use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Contracts\TagRepository;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use App\Domain\Blog\Repositories\CachedPostRepository;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Support\Facades\DB;

function blogQueries(Closure $callback): int
{
    DB::flushQueryLog();
    DB::enableQueryLog();
    $callback();
    $count = count(DB::getQueryLog());
    DB::disableQueryLog();

    return $count;
}

it('binds the contracts to cached decorators', function (): void {
    expect(app(PostRepository::class))->toBeInstanceOf(CachedPostRepository::class);
});

it('serves only published posts, with author, reviewer, tags and category', function (): void {
    $category = Category::factory()->create(['name' => 'چرخه و پریود', 'label' => 'چرخه', 'life_stage' => LifeStage::Cycle]);
    $author = Author::factory()->create();
    $reviewer = Author::factory()->reviewer()->create(['same_as' => ['https://example.test/dr']]);
    $post = Post::factory()->published()->create([
        'slug' => 'period-pain', 'category_id' => $category->id, 'author_id' => $author->id,
        'reviewer_id' => $reviewer->id, 'reviewed_at' => now()->subDays(3), 'life_stage' => null,
    ]);
    app(SyncPostTags::class)->handle($post, Tag::factory()->count(2)->create()->pluck('id')->all());
    Post::factory()->create(['slug' => 'draft']);
    Post::factory()->scheduled()->create(['slug' => 'scheduled']);

    $posts = app(PostRepository::class);
    $data = $posts->findPublishedBySlug('period-pain');

    expect($data?->category?->label)->toBe('چرخه')
        ->and($data?->lifeStage)->toBe(LifeStage::Cycle) // inherited from the category
        ->and($data?->tags)->toHaveCount(2)
        ->and($data?->author?->id)->toBe($author->id)
        ->and($data?->reviewer?->isMedicalReviewer)->toBeTrue()
        ->and($data?->reviewer?->sameAs)->toBe(['https://example.test/dr'])
        ->and($data?->reviewedAt)->not->toBeNull()
        ->and($posts->findPublishedBySlug('draft'))->toBeNull()
        ->and($posts->findPublishedBySlug('scheduled'))->toBeNull()
        ->and($posts->findPublishedBySlug(''))->toBeNull();
});

it('answers warm reads without queries and refreshes after a change', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'cached', 'title' => 'قدیمی']);
    $posts = app(PostRepository::class);
    $categories = app(CategoryRepository::class);

    $posts->findPublishedBySlug('cached');
    $posts->latest();
    $posts->related($post->id);
    $categories->all();

    expect(blogQueries(function () use ($posts, $categories, $post): void {
        $posts->findPublishedBySlug('cached');
        $posts->latest();
        $posts->related($post->id);
        $categories->all();
        $categories->findBySlug('nothing');
    }))->toBe(0);

    $before = app(NamespaceVersions::class)->version('pages');
    $post->update(['title' => 'تازه']);

    expect($posts->findPublishedBySlug('cached')?->title)->toBe('تازه')
        ->and(app(NamespaceVersions::class)->version('pages'))->toBeGreaterThan($before);
});

it('bumps blog and sitemap when tags are synced or taxonomy changes', function (): void {
    $versions = app(NamespaceVersions::class);
    $post = Post::factory()->published()->create();
    $tag = Tag::factory()->create();
    $blog = $versions->version('blog');
    $sitemap = $versions->version('sitemap');

    app(SyncPostTags::class)->handle($post, [$tag->id]);
    expect($versions->version('blog'))->toBeGreaterThan($blog)
        ->and($versions->version('sitemap'))->toBeGreaterThan($sitemap);

    $blog = $versions->version('blog');
    app(SyncPostTags::class)->handle($post, [$tag->id]); // no change → no bump
    expect($versions->version('blog'))->toBe($blog);

    Category::factory()->create();
    expect($versions->version('blog'))->toBeGreaterThan($blog);
});

it('paginates latest posts newest first and filters by life stage', function (): void {
    $stageCategory = Category::factory()->create(['life_stage' => LifeStage::Teen]);
    Post::factory()->count(3)->published()->create(['life_stage' => LifeStage::Cycle]);
    $teen = Post::factory()->published(now()->subMinute())->create(['life_stage' => LifeStage::Teen]);
    $viaCategory = Post::factory()->published(now()->subMinutes(2))->create(['life_stage' => null, 'category_id' => $stageCategory->id]);

    $posts = app(PostRepository::class);
    $page = $posts->latest(page: 2, perPage: 2);

    expect($page->total)->toBe(5)
        ->and($page->items)->toHaveCount(2)
        ->and($page->lastPage())->toBe(3)
        ->and($posts->latest(page: 9, perPage: 2)->isOutOfRange())->toBeTrue()
        ->and(array_map(static fn ($c): int => $c->id, $posts->latest(stage: LifeStage::Teen)->items))->toBe([$teen->id, $viaCategory->id]);
});

it('lists posts by category including direct sub-categories, by tag and by author', function (): void {
    $parent = Category::factory()->create();
    $child = Category::factory()->create(['parent_id' => $parent->id]);
    $inParent = Post::factory()->published()->create(['category_id' => $parent->id]);
    $inChild = Post::factory()->published()->create(['category_id' => $child->id]);
    Post::factory()->published()->create();
    $tag = Tag::factory()->create();
    $inChild->tags()->attach($tag->id);
    $author = Author::factory()->create();
    $inParent->update(['author_id' => $author->id]);

    $posts = app(PostRepository::class);

    expect($posts->byCategory($parent->id)->total)->toBe(2)
        ->and($posts->byCategory($child->id)->total)->toBe(1)
        ->and(array_map(static fn ($c): int => $c->id, $posts->byTag($tag->id)->items))->toBe([$inChild->id])
        ->and(array_map(static fn ($c): int => $c->id, $posts->byAuthor($author->id)->items))->toBe([$inParent->id]);
});

it('returns the newest featured post', function (): void {
    Post::factory()->published(now()->subDays(3))->featured()->create();
    $newest = Post::factory()->published(now()->subDay())->featured()->create();
    Post::factory()->published()->create();
    Post::factory()->featured()->create(); // draft

    expect(app(PostRepository::class)->featured()?->id)->toBe($newest->id);
});

it('reads categories with published counts, tags and authors by slug', function (): void {
    $b = Category::factory()->create(['slug' => 'b', 'sort_order' => 2]);
    $a = Category::factory()->create(['slug' => 'a', 'sort_order' => 1]);
    Post::factory()->count(2)->published()->create(['category_id' => $b->id]);
    Post::factory()->create(['category_id' => $b->id]);
    $tag = Tag::factory()->create(['name' => 'تخمک‌گذاری', 'slug' => 'تخمک-گذاری']);
    Post::factory()->published()->create()->tags()->attach($tag->id);
    Author::factory()->reviewer()->create(['slug' => 'dr-x']);

    $categories = app(CategoryRepository::class)->all();

    expect(array_map(static fn ($c): string => $c->slug, $categories))->toBe(['a', 'b'])
        ->and($categories[1]->postCount)->toBe(2)
        ->and(app(CategoryRepository::class)->findBySlug('a')?->id)->toBe($a->id)
        ->and(app(TagRepository::class)->findBySlug('تخمک-گذاری')?->postCount)->toBe(1)
        ->and(app(AuthorRepository::class)->findBySlug('dr-x')?->isMedicalReviewer)->toBeTrue()
        ->and(app(AuthorRepository::class)->findBySlug('none'))->toBeNull();
});

it('fills taxonomy slugs from the name', function (): void {
    $tag = Tag::query()->create(['name' => 'اولین پریود']);
    $again = Tag::query()->create(['name' => 'اولین پریود']);

    expect($tag->slug)->toBe('اولین-پریود')->and($again->slug)->toBe('اولین-پریود-2');
});
