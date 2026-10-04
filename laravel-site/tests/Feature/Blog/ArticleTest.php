<?php

declare(strict_types=1);

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Rendering\ArticleBodyRenderer;
use App\Domain\Blog\Support\ViewCounter;
use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Models\Media;
use App\Domain\Seo\Models\SeoMeta;
use App\Http\Controllers\Blog\ShowPostController;
use Database\Seeders\BlogSeeder;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\Facades\Storage;
use Tests\Feature\Media\MediaFixtures;

/**
 * @return array<string, mixed>
 */
function articleGraph(string $html): array
{
    preg_match('#<script type="application/ld\+json">(.*?)</script>#s', $html, $m);

    return json_decode($m[1] ?? '{}', true, flags: JSON_THROW_ON_ERROR);
}

/**
 * @return array<string, mixed>|null
 */
function articleNode(string $html, string $type): ?array
{
    return collect(articleGraph($html)['@graph'] ?? [])->firstWhere('@type', $type);
}

function articleImage(string $alt = 'تصویر آزمایشی'): Media
{
    return app(StoreMedia::class)->handle(new MediaUpload(MediaFixtures::jpeg(1600, 1000), 'body.jpg', alt: $alt));
}

beforeEach(function (): void {
    Storage::fake('public');
});

it('routes /blog/{slug} to ShowPostController', function (): void {
    expect(Route::getRoutes()->getByName('blog.show')?->getActionName())->toBe(ShowPostController::class);
});

it('renders the demo article with one h1, meta line, TOC, sources, disclaimer, reviewer box and related posts', function (): void {
    $this->seed(BlogSeeder::class);

    $html = (string) $this->get('/blog/period-pain')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟')
        ->toContain('دقیقه مطالعه')
        ->toContain('به‌روزرسانی: <time datetime=')
        ->toContain('بازبینی علمی: [نام متخصص]')
        ->toContain('بازبینی پزشکی')
        ->toContain('تاریخ بازبینی:')
        ->toContain('در این مقاله')
        ->toContain('href="#درد-پریود-چرا-ایجاد-می-شود"')
        ->toContain('id="درد-پریود-چرا-ایجاد-می-شود"')
        ->toContain('منابع:')
        ->toContain('این مطلب جایگزین مشاوره پزشکی نیست.')
        ->toContain('id="download"')
        ->toContain('مقاله‌های مرتبط')
        ->toContain(route('blog.show', 'first-period-guide')) // shares a tag
        ->toContain('aria-current="page"')
        ->not->toMatch('/\sstyle\s*=/i')
        ->not->toContain('<script src="http')
        ->not->toContain('حتماً')
        ->not->toContain('قطعاً')
        ->not->toContain('تضمینی');
});

it('describes the article for search engines: title, canonical, og article tags and the JSON-LD graph', function (): void {
    config(['app.env' => 'production']);
    $this->seed(BlogSeeder::class);

    $html = (string) $this->get('/blog/period-pain')->assertOk()->getContent();

    expect($html)->toContain('<title>درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟ — ریتمی</title>')
        ->toContain('<link rel="canonical" href="https://')
        ->toContain('/blog/period-pain">')
        ->toContain('<meta name="robots" content="index,follow,max-image-preview:large')
        ->toContain('<meta property="og:type" content="article">')
        ->toContain('<meta property="article:published_time" content="')
        ->toContain('+03:30">')
        ->toContain('<meta property="article:modified_time"')
        ->toContain('<meta property="article:section" content="چرخه و پریود">')
        ->toContain('<meta property="article:tag" content="درد پریود">')
        ->toContain('<meta property="article:author" content="https://');

    $types = array_column(articleGraph($html)['@graph'], '@type');
    expect($types)->toContain('Organization', 'WebSite', 'WebPage', 'BreadcrumbList', 'BlogPosting');

    $article = articleNode($html, 'BlogPosting');
    $page = articleNode($html, 'WebPage');
    $crumbs = articleNode($html, 'BreadcrumbList');
    expect($article['headline'])->toBe('درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟')
        ->and($article['datePublished'])->toEndWith('+03:30')
        ->and($article['dateModified'])->toEndWith('+03:30')
        ->and($article['author'][0]['name'])->toBe('تیم محتوای ریتمی')
        ->and($article['author'][0]['url'])->toStartWith('https://')
        ->and($article['publisher'])->toHaveKey('@id')
        ->and($article['mainEntityOfPage']['@id'])->toBe($page['@id'])
        ->and($article['articleSection'])->toBe('چرخه و پریود')
        ->and($article['keywords'])->toContain('درد پریود')
        ->and($page['reviewedBy']['@type'])->toBe('Person')
        ->and($page['reviewedBy']['name'])->toBe('[نام متخصص]')
        ->and($page['lastReviewed'])->toMatch('/^\d{4}-\d{2}-\d{2}$/')
        ->and(array_column($crumbs['itemListElement'], 'name'))->toBe(['خانه', 'مجله', 'چرخه و پریود', 'درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟']);
});

it('lets the admin seo_meta of the post win over the post title', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'seo-post', 'title' => 'عنوان مقاله برای آزمون سئو']);
    SeoMeta::query()->create(['seoable_type' => $post->getMorphClass(), 'seoable_id' => $post->id, 'title' => 'عنوان دلخواه مدیر سایت برای این مقاله']);

    $this->get('/blog/seo-post')->assertOk()->assertSee('<title>عنوان دلخواه مدیر سایت برای این مقاله — ریتمی</title>', false);
});

it('301s an old slug to the current URL (query kept) and 404s unknown or unpublished posts', function (): void {
    $this->seed(BlogSeeder::class);
    Post::factory()->create(['slug' => 'draft-post']); // not published

    $this->get('/blog/dard-period?utm_source=tg')->assertStatus(301)
        ->assertHeader('Location', route('blog.show', 'period-pain').'?utm_source=tg');
    $this->get('/blog/no-such-post')->assertNotFound();
    $this->get('/blog/draft-post')->assertNotFound();
});

it('renders body images through <x-picture>, lazy, the first one eager without a cover; tables scroll; external links noopener', function (): void {
    $first = articleImage('نمودار درد در چرخه');
    $second = articleImage('گرما روی شکم');
    Post::factory()->published()->create([
        'slug' => 'with-images',
        'cover_media_id' => null,
        'body' => '<p>مقدمه</p><figure><img data-media-id="'.$first->id.'" alt=""><figcaption>زیرنویس</figcaption></figure>'
            .'<h2>جدول</h2><table><tr><th>روز</th><td>۱</td></tr></table>'
            .'<p><img data-media-id="'.$second->id.'" alt="متن جایگزین نوشته"></p>'
            .'<p><a href="https://example.org/study">منبع</a></p>',
    ]);

    $html = (string) $this->get('/blog/with-images')->assertOk()->getContent();

    preg_match_all('#<img [^>]*>#', $html, $imgs);
    $body = array_values(array_filter($imgs[0], static fn (string $img): bool => str_contains($img, 'rounded-4xl')));
    expect($body)->toHaveCount(2)
        ->and($body[0])->toContain('loading="eager"')->toContain('fetchpriority="high"')->toContain('alt="نمودار درد در چرخه"')
        ->toContain('width="')->toContain('height="')
        ->and($body[1])->toContain('loading="lazy"')->toContain('alt="متن جایگزین نوشته"')
        ->and($html)->toContain('<picture')
        ->toContain('sizes="'.ArticleBodyRenderer::SIZES.'"')
        ->toContain('<div class="rt-table-scroll" role="region" tabindex="0"')
        ->toContain('rel="noopener"')
        ->not->toContain('data-media-id');
});

it('drops images whose media is gone and re-sanitises bodies written around the observer', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'raw-body']);
    DB::table('blog_posts')->where('id', $post->id)->update([
        'body' => '<p onclick="x()">متن</p><script>alert(1)</script><img data-media-id="999999" alt="گم شده"><iframe src="https://evil.test"></iframe>',
    ]);
    app('cache.store')->flush();

    $html = (string) $this->get('/blog/raw-body')->assertOk()->getContent();

    expect($html)->toContain('<p>متن</p>')
        ->not->toContain('alert(1)')
        ->not->toContain('onclick')
        ->not->toContain('evil.test')
        ->not->toContain('گم شده');
});

it('gives the cover picture LCP priority (preload) and uses its 1200×630 crop for og:image and BlogPosting', function (): void {
    $cover = articleImage('دختری با کیسه آب گرم');
    Post::factory()->published()->create(['slug' => 'with-cover', 'cover_media_id' => $cover->id]);

    $html = (string) $this->get('/blog/with-cover')->assertOk()->getContent();

    expect($html)->toContain('<link rel="preload" as="image"')
        ->toContain('alt="دختری با کیسه آب گرم"')
        ->toContain('<meta property="og:image" content="')
        ->toContain('<meta property="og:image:width" content="1200">');
    expect(articleNode($html, 'BlogPosting')['image'])->not->toBeNull();
});

it('links the previous and next article of the same category and offers plain share links', function (): void {
    $cycle = Category::factory()->create(['slug' => 'cycle-x']);
    Post::factory()->published(now()->subDays(3))->create(['slug' => 'older', 'title' => 'مقاله قدیمی‌تر', 'category_id' => $cycle->id]);
    Post::factory()->published(now()->subDays(2))->create(['slug' => 'middle', 'category_id' => $cycle->id]);
    Post::factory()->published(now()->subDay())->create(['slug' => 'newer', 'title' => 'مقاله تازه‌تر', 'category_id' => $cycle->id]);

    $html = (string) $this->get('/blog/middle')->assertOk()->getContent();

    expect($html)->toContain('مقاله قبلی')->toContain('href="'.route('blog.show', 'older').'"')
        ->toContain('مقاله بعدی')->toContain('href="'.route('blog.show', 'newer').'"')
        ->toContain('https://t.me/share/url?url=')
        ->toContain('https://wa.me/?text=')
        ->toContain('https://x.com/intent/post?url=')
        ->toContain('rel="noopener noreferrer nofollow"')
        ->toContain('id="article-share-url"');

    expect(app(PostRepository::class)->adjacentInCategory(Post::query()->where('slug', 'older')->value('id'))['previous'])->toBeNull();
});

it('leaves view counting to the beacon and serves cache hits without queries', function (): void {
    $author = Author::factory()->create();
    $post = Post::factory()->published()->create(['slug' => 'counted', 'author_id' => $author->id]);

    $this->get('/blog/counted')->assertOk()->assertHeader('X-Page-Cache', 'MISS')
        ->assertSee('data-module="view-beacon"', false);
    expect(app(ViewCounter::class)->pending($post->id))->toBe(0); // counted by PostViewController (L4-03b)

    DB::enableQueryLog();
    $this->get('/blog/counted')->assertOk()->assertHeader('X-Page-Cache', 'HIT');
    expect(DB::getQueryLog())->toBe([]);
});
