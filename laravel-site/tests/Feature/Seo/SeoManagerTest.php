<?php

declare(strict_types=1);

use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Data\PageContext;
use App\Domain\Seo\Data\SeoImage;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Repositories\CachedSeoMetaRepository;
use App\Domain\Seo\SeoManager;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Route;
use Tests\Feature\Seo\SeoFixtures;

beforeEach(function (): void {
    SeoFixtures::boot();
});

function seo(string $url = 'https://ritme.test/cycle', ?string $route = 'stage.cycle'): SeoManager
{
    return app(SeoManager::class)->context(new PageContext($url, $route));
}

function seoMetaQueries(Closure $callback): int
{
    DB::flushQueryLog();
    DB::enableQueryLog();
    $callback();
    $count = count(array_filter(DB::getQueryLog(), static fn (array $q): bool => str_contains((string) $q['query'], '"seo_meta"')));
    DB::disableQueryLog();

    return $count;
}

it('falls back to the settings defaults', function (): void {
    $defaults = app(SettingsRepository::class)->all()->seo;

    $head = seo()->resolve();

    expect($head->title)->toBe($defaults->defaultTitle)
        ->and($head->description)->toBe($defaults->defaultDescription)
        ->and($head->canonical)->toBe('https://ritme.test/cycle')
        ->and($head->openGraph)->toMatchArray([
            'og:locale' => 'fa_IR',
            'og:site_name' => 'ریتمی',
            'og:type' => 'website',
            'og:url' => 'https://ritme.test/cycle',
            'og:title' => $defaults->defaultTitle,
        ])
        ->and($head->twitter['twitter:card'])->toBe('summary')
        ->and(array_column($head->feeds, 'href'))->toBe([route('blog.feed')]); // L4-04
});

it('layers defaults, then static page meta, then model meta, then controller overrides', function (): void {
    SeoFixtures::routeMeta('stage.cycle', [
        'title' => 'پیگیری چرخه', 'description' => 'توضیح صفحه', 'og_type' => 'website', 'og_title' => 'عنوان شبکه‌ای صفحه',
    ]);
    $page = SeoFixtures::page(7);
    SeoMeta::query()->create([
        'seoable_type' => $page->getMorphClass(), 'seoable_id' => 7, 'description' => 'توضیح مدل', 'og_type' => 'article',
    ]);

    $routeOnly = seo()->resolve();
    expect($routeOnly->title)->toBe('پیگیری چرخه — ریتمی')
        ->and($routeOnly->description)->toBe('توضیح صفحه')
        ->and($routeOnly->openGraph['og:title'])->toBe('عنوان شبکه‌ای صفحه');

    app()->forgetInstance(SeoManager::class);
    $withModel = seo()->for($page)->resolve();
    expect($withModel->title)->toBe('پیگیری چرخه — ریتمی')       // inherited from the page layer
        ->and($withModel->description)->toBe('توضیح مدل')          // model wins over page
        ->and($withModel->openGraph['og:type'])->toBe('article');

    app()->forgetInstance(SeoManager::class);
    $overridden = seo()->for($page)->title('عنوان کنترلر')->description('توضیح کنترلر')->type('product')->resolve();
    expect($overridden->title)->toBe('عنوان کنترلر — ریتمی')
        ->and($overridden->description)->toBe('توضیح کنترلر')
        ->and($overridden->openGraph['og:type'])->toBe('product')
        ->and($overridden->openGraph['og:title'])->toBe('عنوان شبکه‌ای صفحه');

    expect(seo()->rawTitle('عنوان کامل بدون قالب')->resolve()->title)->toBe('عنوان کامل بدون قالب');
});

it('derives the description from the excerpt only when nothing else provides one', function (): void {
    $excerpt = '<p>'.str_repeat('درد پریود برای خیلی‌ها آشناست و راه‌های ساده‌ای برای کم کردنش هست. ', 5).'</p>';

    $head = seo()->excerpt($excerpt)->resolve();
    expect(mb_strlen($head->description))->toBeLessThanOrEqual(155)->and($head->description)->toEndWith('…');

    app()->forgetInstance(SeoManager::class);
    SeoFixtures::routeMeta('stage.cycle', ['description' => 'توضیح ادمین']);
    expect(seo()->excerpt($excerpt)->resolve()->description)->toBe('توضیح ادمین');
});

it('normalises the canonical and lets keepQuery() keep a parameter', function (): void {
    expect(seo('http://ritme.test//blog/?utm_source=ig&page=2&fbclid=x#top', 'blog.index')->resolve()->canonical)
        ->toBe('https://ritme.test/blog?page=2')
        ->and(seo('https://ritme.test/blog?page=1', 'blog.index')->resolve()->canonical)->toBe('https://ritme.test/blog');

    app()->forgetInstance(SeoManager::class);
    expect(seo('https://ritme.test/blog/tag?t=x', 'blog.tag')->keepQuery('t')->resolve()->canonical)
        ->toBe('https://ritme.test/blog/tag?t=x');

    app()->forgetInstance(SeoManager::class);
    SeoFixtures::routeMeta('stage.cycle', ['canonical_url' => '/cycle/']);
    expect(seo('https://ritme.test/cycle?x=1')->resolve()->canonical)->toBe('https://ritme.test/cycle')
        ->and(seo()->canonical('https://other.example/source')->resolve()->canonical)->toBe('https://other.example/source');
});

it('is always noindex outside production', function (): void {
    expect(app()->environment())->toBe('testing')
        ->and(seo()->resolve()->robots)->toBe('noindex,follow')
        ->and(seo()->robots('index,follow')->resolve()->indexable)->toBeFalse();
});

it('indexes clean production pages and forces noindex on search, cart, checkout, done and filtered pages', function (string $url, ?string $route, bool $indexable): void {
    config(['app.env' => 'production']);

    $head = seo($url, $route)->resolve();

    expect($head->indexable)->toBe($indexable)
        ->and($head->robots)->toBe($indexable ? 'index,follow,max-image-preview:large,max-snippet:-1,max-video-preview:-1' : 'noindex,follow');
})->with([
    'page' => ['https://ritme.test/cycle', 'stage.cycle', true],
    'tracking only' => ['https://ritme.test/cycle?utm_source=ig&gclid=1', 'stage.cycle', true],
    'paginated' => ['https://ritme.test/blog?page=3', 'blog.index', true],
    'search' => ['https://ritme.test/blog/search?q=درد', 'blog.search', false],
    'cart' => ['https://ritme.test/shop/cart', 'shop.cart', false],
    'checkout' => ['https://ritme.test/shop/checkout', 'shop.checkout', false],
    'order done' => ['https://ritme.test/shop/order/AB12', 'shop.order', false],
    'join done' => ['https://ritme.test/directory/join/done', 'directory.join.done', false],
    'booked' => ['https://ritme.test/directory/booked/X1', 'directory.booked', false],
    'filtered listing' => ['https://ritme.test/shop/category/baby?color=red', 'shop.category', false],
]);

it('takes robots from seo_meta, then controller overrides, in production', function (): void {
    config(['app.env' => 'production']);
    SeoFixtures::routeMeta('stage.cycle', ['robots' => 'noindex,nofollow']);

    expect(seo()->resolve()->robots)->toBe('noindex,nofollow')
        ->and(seo()->robots('index,follow')->resolve()->robots)->toBe('index,follow');

    app()->forgetInstance(SeoManager::class);
    expect(seo('https://ritme.test/shop', 'shop.index')->filtered()->resolve()->indexable)->toBeFalse()
        ->and(seo('https://ritme.test/x', 'x')->noindex()->resolve()->robots)->toBe('noindex,follow');
});

it('emits image, twitter, verification and feed tags when available', function (): void {
    app(SettingsRepository::class)->put(SettingGroup::Seo, [
        'twitter_handle' => 'ritmeapp',
        'verification' => ['google' => 'g-token', 'bing' => 'b-token', 'custom-verify' => 'c', 'yandex' => ' '],
    ]);
    Route::get('/blog/feed', fn (): string => '')->name('blog.feed');
    app('router')->getRoutes()->refreshNameLookups();

    $head = seo()->image(new SeoImage('https://ritme.test/media/og.jpg', 1200, 630, 'جلد', 'image/jpeg'))->resolve();

    expect($head->openGraph)->toMatchArray([
        'og:image' => 'https://ritme.test/media/og.jpg',
        'og:image:width' => '1200',
        'og:image:height' => '630',
        'og:image:alt' => 'جلد',
        'og:image:type' => 'image/jpeg',
    ])
        ->and($head->twitter)->toMatchArray([
            'twitter:card' => 'summary_large_image', 'twitter:site' => '@ritmeapp', 'twitter:image' => 'https://ritme.test/media/og.jpg',
        ])
        ->and($head->verification)->toBe(['google-site-verification' => 'g-token', 'msvalidate.01' => 'b-token', 'custom-verify' => 'c'])
        ->and($head->feeds)->toHaveCount(1)
        ->and($head->feeds[0]['href'])->toEndWith('/blog/feed');
});

it('reads seo_meta through the cache and sees changes immediately', function (): void {
    expect(app(SeoMetaRepository::class))->toBeInstanceOf(CachedSeoMetaRepository::class);

    $meta = SeoFixtures::routeMeta('stage.cycle', ['title' => 'نسخه اول']);

    expect(seoMetaQueries(fn () => seo()->resolve()))->toBe(1)
        ->and(seoMetaQueries(fn () => seo()->resolve()))->toBe(0)
        ->and(seoMetaQueries(fn () => seo('https://ritme.test/x', 'no.meta')->resolve()))->toBe(1)
        ->and(seoMetaQueries(fn () => seo('https://ritme.test/x', 'no.meta')->resolve()))->toBe(0); // misses cached too

    $meta->update(['title' => 'نسخه دوم']);

    expect(seo()->resolve()->title)->toBe('نسخه دوم — ریتمی');
});

it('gives every request a fresh SeoManager', function (): void {
    Route::get('/seo-override', function (SeoManager $seo) {
        $seo->title('عنوان مخصوص یک درخواست');

        return view('seo-fixtures::page');
    });
    Route::view('/seo-plain', 'seo-fixtures::page');

    $this->get('/seo-override')->assertSee('<title>عنوان مخصوص یک درخواست — ریتمی</title>', false);
    $this->get('/seo-plain')->assertDontSee('عنوان مخصوص یک درخواست');
});
