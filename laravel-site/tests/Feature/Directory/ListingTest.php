<?php

declare(strict_types=1);

use App\Domain\Directory\Actions\SyncPlaceAmenities;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Landing;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use Carbon\CarbonImmutable;

/**
 * @return array<string, mixed>
 */
function directoryGraph(string $html): array
{
    preg_match('#<script type="application/ld\+json">(.*?)</script>#s', $html, $m);

    return json_decode($m[1] ?? '{}', true, flags: JSON_THROW_ON_ERROR);
}

function directoryRobots(string $html): string
{
    preg_match('#<meta name="robots" content="([^"]+)"#', $html, $m);

    return $m[1] ?? '';
}

function directoryCanonical(string $html): string
{
    preg_match('#<link rel="canonical" href="([^"]+)"#', $html, $m);

    return $m[1] ?? '';
}

beforeEach(function (): void {
    $this->tehran = City::factory()->create(['slug' => 'tehran', 'name' => 'تهران']);
    $this->vanak = District::factory()->create(['city_id' => $this->tehran->id, 'slug' => 'vanak', 'name' => 'ونک']);
    $this->pool = PlaceCategory::factory()->create(['slug' => 'pool', 'name' => 'استخر مادر و کودک', 'icon' => 'waves']);
    $this->playhouse = PlaceCategory::factory()->create(['slug' => 'playhouse', 'name' => 'خانه بازی', 'icon' => 'home']);
});

function directoryPlace(array $attributes = []): Place
{
    return Place::factory()->published()->create([
        'city_id' => test()->tehran->id,
        'category_id' => test()->pool->id,
        ...$attributes,
    ]);
}

// ---- /directory --------------------------------------------------------------------------------------------------

it('renders the directory with one h1, place cards, crawlable chips, the business CTA and no map', function (): void {
    directoryPlace(['name' => 'استخر آب‌پری', 'slug' => 'ab-pari', 'district_id' => $this->vanak->id, 'age_min_months' => 6, 'age_max_months' => 48, 'price_from' => 320000]);
    directoryPlace(['name' => 'خانه بازی تاب‌تاب', 'slug' => 'tab-tab', 'category_id' => $this->playhouse->id]);

    $html = (string) $this->get('/directory')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('جاهای خوب برای تو و کودکت، نزدیک خانه')
        ->and($html)->toContain('href="'.route('directory.place', 'ab-pari').'"')
        ->and($html)->toContain('مناسب ۶ ماه تا ۴ سال')
        ->and($html)->toContain('href="'.route('directory.index').'?category=pool"') // no city → query (noindex)
        ->and($html)->toContain('href="'.route('directory.city', 'tehran').'"')      // side panel → indexable landing
        ->and($html)->toContain('href="'.route('directory.category', ['tehran', 'pool']).'"')
        ->and($html)->toContain('href="'.route('directory.business').'"')
        ->and($html)->toContain('method="get"')
        ->and($html)->toContain('name="where"')
        ->and($html)->not->toContain('style=')
        ->and($html)->not->toContain('بزرگ‌نمایی')            // no map zoom controls
        ->and($html)->not->toMatch('#(src|href)="https?://(?!localhost|127\.0\.0\.1)#');

    $graph = directoryGraph($html)['@graph'];
    $types = array_column($graph, '@type');
    $itemList = collect($graph)->firstWhere('@type', 'ItemList');
    $breadcrumb = collect($graph)->firstWhere('@type', 'BreadcrumbList');
    expect($types)->toContain('CollectionPage', 'ItemList', 'BreadcrumbList')
        ->and($itemList['numberOfItems'])->toBe(2)
        ->and($breadcrumb['itemListElement'])->toHaveCount(2);
});

it('is indexable in production with real places and noindex when it lists only demo places', function (): void {
    config(['app.env' => 'production']);
    directoryPlace(['slug' => 'demo', 'is_demo' => true]);

    expect(directoryRobots((string) $this->get('/directory/tehran')->getContent()))->toStartWith('noindex');

    directoryPlace(['slug' => 'real']);
    $html = (string) $this->get('/directory/tehran')->getContent();
    expect(directoryRobots($html))->toStartWith('index')
        ->and(directoryCanonical($html))->toEndWith('/directory/tehran');
});

// ---- landings ----------------------------------------------------------------------------------------------------

it('renders city and city × category landings with template copy, admin overrides and breadcrumbs', function (): void {
    config(['app.env' => 'production']);
    directoryPlace(['slug' => 'pool-a']);
    directoryPlace(['slug' => 'play-a', 'category_id' => $this->playhouse->id]);

    $city = (string) $this->get('/directory/tehran')->assertOk()->getContent();
    expect($city)->toContain('<title>خدمات مادر و کودک در تهران')
        ->and($city)->toContain('href="'.route('directory.category', ['tehran', 'pool']).'"')
        ->and(collect(directoryGraph($city)['@graph'])->firstWhere('@type', 'ItemList')['numberOfItems'])->toBe(2);

    Landing::query()->create(['city_id' => $this->tehran->id, 'category_id' => $this->pool->id, 'h1' => 'استخرهای مادر و کودک تهران', 'meta_title' => 'استخر مادر و کودک تهران', 'intro' => 'مقدمه ویرایش‌شده استخرها']);
    $category = (string) $this->get('/directory/tehran/pool')->assertOk()->getContent();
    $breadcrumb = collect(directoryGraph($category)['@graph'])->firstWhere('@type', 'BreadcrumbList');

    expect(substr_count($category, '<h1'))->toBe(1)
        ->and($category)->toMatch('#<h1[^>]*>استخرهای مادر و کودک تهران</h1>#')
        ->and($category)->toContain('<title>استخر مادر و کودک تهران')
        ->and($category)->toContain('مقدمه ویرایش‌شده استخرها')
        ->and($category)->toContain('aria-label="مسیر"')
        ->and($category)->toContain('href="'.route('directory.place', 'pool-a').'"')
        ->and($category)->not->toContain('href="'.route('directory.place', 'play-a').'"')
        ->and(directoryRobots($category))->toStartWith('index')
        ->and($breadcrumb['itemListElement'])->toHaveCount(4)
        ->and($breadcrumb['itemListElement'][3]['name'])->toBe('استخر مادر و کودک');
});

it('404s unknown cities and categories and keeps the fixed directory routes first', function (): void {
    $this->get('/directory/karaj')->assertNotFound();
    $this->get('/directory/tehran/nope')->assertNotFound();

    expect(route('directory.business'))->toEndWith('/directory/business');
    $this->get('/directory/business')->assertOk();
    $this->get('/directory/join')->assertOk();
});

it('shows an empty, noindex landing for a category without places', function (): void {
    config(['app.env' => 'production']);

    $html = (string) $this->get('/directory/tehran/playhouse')->assertOk()->getContent();

    expect($html)->toContain('با این فیلترها مجموعه‌ای پیدا نکردیم')
        ->and(directoryRobots($html))->toStartWith('noindex');
});

// ---- filters -----------------------------------------------------------------------------------------------------

it('makes filtered combinations noindex,follow with the canonical on the nearest landing', function (): void {
    config(['app.env' => 'production']);
    directoryPlace(['slug' => 'p']);

    $filtered = (string) $this->get('/directory/tehran/pool?q=%D8%A7%D8%B3%D8%AA%D8%AE%D8%B1')->assertOk()->getContent();
    expect(directoryRobots($filtered))->toBe('noindex,follow')
        ->and(directoryCanonical($filtered))->toEndWith('/directory/tehran/pool');

    $categoryOnly = (string) $this->get('/directory?category=pool')->assertOk()->getContent();
    expect(directoryRobots($categoryOnly))->toBe('noindex,follow')
        ->and(directoryCanonical($categoryOnly))->toEndWith('/directory');
});

it('301s form submissions and non-canonical queries to one canonical URL per filter state', function (): void {
    $amenityB = Amenity::factory()->create(['slug' => 'b-nursing']);
    $amenityA = Amenity::factory()->create(['slug' => 'a-coach']);
    expect($amenityA->id + $amenityB->id)->toBeGreaterThan(0);

    $this->get('/directory?city=tehran&q=&age=')->assertRedirect(route('directory.city', 'tehran'))->assertStatus(301);
    $this->get('/directory?where=tehran/vanak&category=pool&q=')
        ->assertStatus(301)->assertRedirect(route('directory.category', ['tehran', 'pool']).'?district=vanak');
    $this->get('/directory/tehran?page=1')->assertStatus(301)->assertRedirect(route('directory.city', 'tehran'));
    $this->get('/directory/tehran?foo=bar&utm_source=x')->assertStatus(301)->assertRedirect(route('directory.city', 'tehran').'?utm_source=x');
    $this->get('/directory/tehran?amenity[]=b-nursing&amenity[]=a-coach&amenity[]=unknown')
        ->assertStatus(301)->assertRedirect(route('directory.city', 'tehran').'?amenity%5B%5D=a-coach&amenity%5B%5D=b-nursing');

    // Canonical shapes answer directly (no redirect loop).
    $this->get('/directory/tehran?amenity%5B%5D=a-coach&amenity%5B%5D=b-nursing')->assertOk();
    $this->get('/directory/tehran/pool?district=vanak')->assertOk();
    $this->get('/directory?category=pool&open=1&sort=price')->assertOk();
});

it('filters by district, child age, amenities and open now through GET parameters', function (): void {
    $nursing = Amenity::factory()->create(['slug' => 'nursing-room', 'name' => 'اتاق شیردهی']);
    $inVanak = directoryPlace(['name' => 'استخر ونک', 'slug' => 'in-vanak', 'district_id' => $this->vanak->id, 'age_min_months' => 1, 'age_max_months' => 12]);
    directoryPlace(['name' => 'استخر بزرگ‌ترها', 'slug' => 'older', 'age_min_months' => 24, 'age_max_months' => 72]);
    app(SyncPlaceAmenities::class)->handle($inVanak, [$nursing->id]);

    $byDistrict = (string) $this->get('/directory/tehran?district=vanak')->getContent();
    $byAge = (string) $this->get('/directory/tehran?age=36')->getContent();
    $byAmenity = (string) $this->get('/directory/tehran?amenity%5B%5D=nursing-room')->getContent();

    expect($byDistrict)->toContain('استخر ونک')->and($byDistrict)->not->toContain('استخر بزرگ‌ترها')
        ->and($byDistrict)->toContain('در ونک')
        ->and($byAge)->toContain('استخر بزرگ‌ترها')->and($byAge)->not->toContain('استخر ونک')
        ->and($byAmenity)->toContain('استخر ونک')->and($byAmenity)->not->toContain('استخر بزرگ‌ترها')
        ->and($byAmenity)->toContain('aria-current="true"'); // the quick toggle is shown as active
});

it('shows «باز است» only when the place stays open for the whole page-cache lifetime', function (): void {
    config(['pagecache.ttl' => 3600]);
    // Saturday 2026-10-03 (Tehran): open 09:00–20:00.
    directoryPlace(['name' => 'استخر ساعت‌دار', 'slug' => 'hours'])->forceFill(['opening_hours' => [
        'saturday' => [['opens' => '09:00', 'closes' => '20:00']],
    ]])->save();

    $this->travelTo(CarbonImmutable::parse('2026-10-03 12:00', 'Asia/Tehran'));
    expect((string) $this->get('/directory/tehran')->getContent())->toContain('باز است</span>');

    $this->travelTo(CarbonImmutable::parse('2026-10-03 19:30', 'Asia/Tehran'));
    app('cache')->flush();
    $late = (string) $this->get('/directory/tehran')->getContent();
    expect(substr_count($late, 'باز است</span>'))->toBe(0);
});

// ---- pagination --------------------------------------------------------------------------------------------------

it('paginates with self-canonical pages and 404s invalid or out-of-range pages', function (): void {
    config(['app.env' => 'production']);
    Place::factory()->published()->count(13)->create(['city_id' => $this->tehran->id, 'category_id' => $this->pool->id]);

    $first = (string) $this->get('/directory/tehran')->assertOk()->getContent();
    expect($first)->toContain('rel="next"')->and($first)->toContain('href="'.route('directory.city', 'tehran').'?page=2"');

    $second = (string) $this->get('/directory/tehran?page=2')->assertOk()->getContent();
    expect(directoryCanonical($second))->toEndWith('/directory/tehran?page=2')
        ->and(directoryRobots($second))->toStartWith('index')
        ->and($second)->toContain('صفحه ۲');

    $this->get('/directory/tehran?page=3')->assertNotFound();
    $this->get('/directory/tehran?page=abc')->assertNotFound();
});
