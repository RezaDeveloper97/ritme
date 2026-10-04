<?php

declare(strict_types=1);

use App\Domain\Directory\Actions\SyncPlaceAmenities;
use App\Domain\Directory\Actions\SyncPlaceGallery;
use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Directory\Models\PlaceService;
use App\Domain\Media\Models\Media;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use App\Http\Controllers\Directory\ShowPlaceController;
use App\Http\Middleware\PageCache;
use Carbon\CarbonImmutable;
use Database\Seeders\SettingsSeeder;
use Illuminate\Support\Facades\RateLimiter;

/**
 * @return array<string, mixed>
 */
function placePageNode(string $html, string $type): ?array
{
    preg_match('#<script type="application/ld\+json">(.*?)</script>#s', $html, $m);
    $graph = json_decode($m[1] ?? '{}', true, flags: JSON_THROW_ON_ERROR)['@graph'] ?? [];

    return collect($graph)->firstWhere('@type', $type);
}

function placePageRobots(string $html): string
{
    preg_match('#<meta name="robots" content="([^"]+)"#', $html, $m);

    return $m[1] ?? '';
}

function placePageMedia(string $name): Media
{
    return Media::query()->create([
        'disk' => 'public', 'directory' => '2026/10/'.$name, 'filename' => $name.'.jpg', 'mime' => 'image/jpeg',
        'size' => 100, 'width' => 1200, 'height' => 800, 'alt' => 'تصویر '.$name, 'hash' => hash('sha256', $name),
    ]);
}

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    $this->tehran = City::factory()->create(['slug' => 'tehran', 'name' => 'تهران']);
    $this->vanak = District::factory()->create(['city_id' => $this->tehran->id, 'slug' => 'vanak', 'name' => 'ونک']);
    $this->pool = PlaceCategory::factory()->create([
        'slug' => 'pool', 'name' => 'استخر مادر و کودک', 'icon' => 'waves', 'schema_type' => LocalBusinessType::SportsActivityLocation,
    ]);

    $this->place = Place::factory()->published()->verified()->withHours()->create([
        'name' => 'استخر آب‌پری', 'slug' => 'ab-pari', 'city_id' => $this->tehran->id, 'district_id' => $this->vanak->id,
        'category_id' => $this->pool->id, 'latitude' => 35.7575, 'longitude' => 51.41, 'age_min_months' => 6, 'age_max_months' => 48,
        'rules' => "کلاه شنا لازم است\n۱۵ دقیقه زودتر برسید", 'cancellation_policy' => 'لغو رایگان تا ۲۴ ساعت قبل',
        'phones' => ['02188776655'],
    ]);
    PlaceService::query()->create(['place_id' => $this->place->id, 'name' => 'آشنایی با آب', 'duration_minutes' => 45, 'price' => 320000, 'price_unit' => 'هر جلسه', 'details' => 'گروهی', 'age_min_months' => 6, 'age_max_months' => 18]);
    PlaceService::query()->create(['place_id' => $this->place->id, 'name' => 'جلسه خصوصی', 'price' => 650000, 'price_unit' => 'هر جلسه']);
    app(SyncPlaceAmenities::class)->handle($this->place, [Amenity::factory()->create(['name' => 'اتاق شیردهی', 'icon' => 'bottle'])->id]);

    // Saturday 10:00 Tehran: open (09–20) for the whole cache hour.
    CarbonImmutable::setTestNow(CarbonImmutable::parse('2026-10-03 10:00', 'Asia/Tehran'));
});

afterEach(function (): void {
    CarbonImmutable::setTestNow();
});

// ---- page --------------------------------------------------------------------------------------------------------

it('renders the place page with one h1, every block, deep links instead of a map and no inline styles', function (): void {
    $html = (string) $this->get('/directory/place/ab-pari')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('استخر آب‌پری</h1>')
        ->and($html)->toContain('مجوز و مدارک بررسی شد')
        ->and($html)->toContain('ونک، تهران')
        ->and($html)->toContain('درباره مجموعه')
        ->and($html)->toContain('۶ ماه تا ۴ سال')                          // age range as the first amenity
        ->and($html)->toContain('اتاق شیردهی')
        ->and($html)->toContain('آشنایی با آب')
        ->and($html)->toContain('۴۵ دقیقه · ۶ تا ۱۸ ماه · گروهی')
        ->and($html)->toContain('۳۲۰ هزار تومان')
        ->and($html)->toContain('href="#book"')                               // «انتخاب» → booking slot (L5-04)
        ->and($html)->toContain('data-booking-slot')
        ->and($html)->toContain('ساعت کاری')
        ->and($html)->toContain('جمعه')
        ->and($html)->toContain('تعطیل')
        ->and($html)->toContain('با پزشک کودک مشورت کن')                       // pool health note
        ->and($html)->toContain('کلاه شنا لازم است')
        ->and($html)->toContain('لغو رایگان تا ۲۴ ساعت قبل')
        ->and($html)->toContain('href="tel:02188776655"')
        ->and($html)->toContain('href="geo:35.7575,51.41?q=')
        ->and($html)->toContain('href="https://neshan.org/maps/@35.7575,51.41,16z,0p"')
        ->and($html)->toContain('href="https://balad.ir/location?latitude=35.7575&amp;longitude=51.41&amp;zoom=16"')
        ->and($html)->toContain('href="'.route('directory.business').'"')
        ->and($html)->not->toContain('<iframe')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($html)->not->toMatch('#\s(src|srcset)="https?://(?!localhost|127\.0\.0\.1)#'); // nothing is loaded from elsewhere
});

it('emits ItemPage, the LocalBusiness subtype with address, geo, hours and price range, and the breadcrumb trail', function (): void {
    $html = (string) $this->get('/directory/place/ab-pari')->assertOk()->getContent();

    $place = placePageNode($html, 'SportsActivityLocation');
    $page = placePageNode($html, 'ItemPage');
    $trail = placePageNode($html, 'BreadcrumbList');

    expect($place)->not->toBeNull()
        ->and($place['@id'])->toEndWith('/directory/place/ab-pari#place')
        ->and($place['address']['@type'])->toBe('PostalAddress')
        ->and($place['address']['addressLocality'])->toBe('تهران')
        ->and($place['geo'])->toBe(['@type' => 'GeoCoordinates', 'latitude' => 35.7575, 'longitude' => 51.41])
        ->and($place['telephone'])->toBe('02188776655')
        ->and($place['openingHoursSpecification'])->toHaveCount(2)
        ->and($place['openingHoursSpecification'][0]['opens'])->toBe('09:00')
        ->and($place['priceRange'])->toBe('۳۲۰٬۰۰۰ تا ۶۵۰٬۰۰۰ تومان')
        ->and($place)->not->toHaveKey('aggregateRating')                      // no real reviews yet
        ->and($place)->not->toHaveKey('review')
        ->and($page['mainEntity']['@id'])->toBe($place['@id'])
        ->and(array_column($trail['itemListElement'], 'name'))->toBe(['خانه', 'خدمات مادر و کودک', 'تهران', 'استخر مادر و کودک', 'استخر آب‌پری'])
        ->and($trail['itemListElement'][3]['item'])->toEndWith('/directory/tehran/pool');
});

it('adds aggregateRating and reviews only from real approved reviews; demo samples are labelled and not counted', function (): void {
    PlaceReview::factory()->approved()->create(['place_id' => $this->place->id, 'author_name' => 'مادر رها', 'rating' => 5, 'body' => 'مربی صبور بود.', 'aspects' => ['staff' => 5]]);
    PlaceReview::factory()->approved()->create(['place_id' => $this->place->id, 'author_name' => 'مادر نیلا', 'rating' => 4, 'body' => 'پارکینگ شلوغ است.']);
    PlaceReview::factory()->approved()->demo()->create(['place_id' => $this->place->id, 'author_name' => 'مادر نمونه', 'rating' => 1, 'body' => 'متن نمونه.']);
    PlaceReview::factory()->create(['place_id' => $this->place->id, 'author_name' => 'در انتظار', 'body' => 'هنوز تأیید نشده.']);

    $html = (string) $this->get('/directory/place/ab-pari')->assertOk()->getContent();
    $place = placePageNode($html, 'SportsActivityLocation');

    expect($place['aggregateRating'])->toMatchArray(['ratingValue' => 4.5, 'ratingCount' => 2])
        ->and(array_column(array_column($place['review'], 'author'), 'name'))->toBe(['مادر نیلا', 'مادر رها'])
        ->and($html)->toContain('مادر نمونه')
        ->and($html)->toContain('نمونه</div>')                                // the demo card's meta label
        ->and($html)->toContain('از ۲ نظر')
        ->and($html)->not->toContain('در انتظار');
});

it('keeps demo places out of the index', function (): void {
    $this->place->update(['is_demo' => true]);

    $html = (string) $this->get('/directory/place/ab-pari')->assertOk()->getContent();

    expect(placePageRobots($html))->toContain('noindex')
        ->and($html)->toContain('مجموعه نمونه');
});

it('shows "open until" only when the place stays open for the whole page-cache TTL, and flags today the same way', function (): void {
    $html = (string) $this->get('/directory/place/ab-pari')->getContent();
    expect($html)->toContain('باز است تا ۲۰:۰۰')
        ->and($html)->toContain('aria-current="date"');

    // 19:30 — would close within the cached hour; 23:30 — the weekday changes within the hour.
    foreach (['2026-10-03 19:30', '2026-10-03 23:30'] as $at) {
        CarbonImmutable::setTestNow(CarbonImmutable::parse($at, 'Asia/Tehran'));
        $this->artisan('cache:ns', ['action' => 'bump', 'namespace' => 'pages']);
        $later = (string) $this->get('/directory/place/ab-pari')->getContent();
        expect($later)->not->toContain('باز است تا');
    }
    expect($later)->not->toContain('aria-current="date"');
});

it('renders photos through <x-picture> (first one LCP) with a no-JS photo list and the lightbox module', function (): void {
    $a = placePageMedia('a');
    $b = placePageMedia('b');
    $this->place->update(['cover_media_id' => $a->id]);
    app(SyncPlaceGallery::class)->handle($this->place, [$b->id]);

    $html = (string) $this->get('/directory/place/ab-pari')->assertOk()->getContent();

    expect($html)->toContain('data-module="gallery"')
        ->and($html)->toContain('fetchpriority="high"')
        ->and($html)->toContain('<details id="photos"')
        ->and($html)->toContain('همه ۲ عکس')
        ->and(substr_count($html, 'data-gallery-item'))->toBe(2)
        ->and($html)->toContain('<dialog data-gallery-dialog');

    $place = placePageNode($html, 'SportsActivityLocation');
    expect($place['image'])->toHaveCount(2);
});

it('shows local cover illustrations and no lightbox for a place without photos', function (): void {
    $html = (string) $this->get('/directory/place/ab-pari')->getContent();

    expect($html)->not->toContain('data-module="gallery"')
        ->and($html)->not->toContain('<img');
});

// ---- URLs --------------------------------------------------------------------------------------------------------

it('301s an old slug to the current one keeping the query, and 404s unknown or unpublished places', function (): void {
    $this->place->update(['slug' => 'ab-pari-vanak']);

    $this->get('/directory/place/ab-pari?utm_source=x')->assertStatus(301)
        ->assertHeader('Location', route('directory.place', 'ab-pari-vanak').'?utm_source=x');
    $this->get('/directory/place/nope')->assertNotFound();

    Place::factory()->create(['slug' => 'draft-place', 'city_id' => $this->tehran->id, 'category_id' => $this->pool->id]);
    $this->get('/directory/place/draft-place')->assertNotFound();
});

it('paginates approved reviews: ?page=1 301s, pages past the end 404, page 2 has its own title', function (): void {
    PlaceReview::factory()->approved()->count(ShowPlaceController::REVIEWS_PER_PAGE + 1)->create(['place_id' => $this->place->id]);

    $this->get('/directory/place/ab-pari?page=1')->assertStatus(301)->assertHeader('Location', route('directory.place', 'ab-pari'));
    $this->get('/directory/place/ab-pari?page=3')->assertNotFound();
    $this->get('/directory/place/ab-pari?page=x')->assertNotFound();

    $first = (string) $this->get('/directory/place/ab-pari')->getContent();
    $second = (string) $this->get('/directory/place/ab-pari?page=2')->assertOk()->getContent();

    expect($first)->toContain('href="'.route('directory.place', ['ab-pari', 'page' => 2]).'#reviews"')
        ->and($second)->toContain('نظرها، صفحه ۲')
        ->and(substr_count($second, '<h1'))->toBe(1);
});

// ---- review form -------------------------------------------------------------------------------------------------

it('keeps the review form CSRF token per visitor on page-cache hits', function (): void {
    $this->get('/directory/place/ab-pari');
    $hit = $this->get('/directory/place/ab-pari');

    expect($hit->headers->get(PageCache::HEADER))->toBe('HIT')
        ->and((string) $hit->getContent())->toMatch('/name="_token" value="[A-Za-z0-9]{20,}"/')
        ->and((string) $hit->getContent())->toContain('action="'.route('directory.place.review', 'ab-pari').'"');
});

it('stores a submitted review as pending and thanks the visitor', function (): void {
    $this->post('/directory/place/ab-pari/reviews', [
        'author_name' => 'مادر سارا', 'rating' => '4', 'body' => 'کلاس آرام و مربی مهربان بود.', 'aspects' => ['staff' => '5', 'access' => ''],
    ])->assertRedirect(route('directory.place', 'ab-pari').'#review-form')->assertSessionHas('review_submitted', true);

    $review = PlaceReview::query()->sole();
    expect($review->status)->toBe(ReviewStatus::Pending)
        ->and($review->author_name)->toBe('مادر سارا')
        ->and($review->aspects)->toBe(['staff' => 5])
        ->and($review->ip_hash)->not->toBeNull();

    // Pending reviews are not shown; the flash note is (and bypasses the page cache).
    $this->followingRedirects()->get('/directory/place/ab-pari')->assertDontSee('کلاس آرام و مربی مهربان بود.');
});

it('sends invalid reviews back with Persian errors and the old input', function (): void {
    $this->post('/directory/place/ab-pari/reviews', ['author_name' => 'م', 'rating' => '9', 'body' => 'کوتاه'])
        ->assertRedirect(route('directory.place', 'ab-pari').'#review-form')
        ->assertSessionHasErrors(['author_name', 'rating', 'body'])
        ->assertSessionHasInput('body', 'کوتاه');

    $html = (string) $this->get('/directory/place/ab-pari')->getContent();
    expect($html)->toContain('نظرت ثبت نشد')
        ->and($html)->toContain('یک امتیاز از ۱ تا ۵ انتخاب کن.')
        ->and(PlaceReview::query()->count())->toBe(0);
});

it('answers the honeypot like a real submit without storing anything', function (): void {
    $this->post('/directory/place/ab-pari/reviews', [
        'author_name' => 'ربات', 'rating' => '5', 'body' => 'متن تبلیغاتی بسیار طولانی', ShowPlaceController::HONEYPOT => 'https://spam.example',
    ])->assertRedirect(route('directory.place', 'ab-pari').'#review-form')->assertSessionHas('review_submitted', true);

    expect(PlaceReview::query()->count())->toBe(0);
});

it('rate limits review posts per IP and 404s reviews for unknown places', function (): void {
    RateLimiter::clear('');
    $payload = ['author_name' => 'مادر سارا', 'rating' => '5', 'body' => 'تجربه خوبی بود، ممنون.'];

    for ($i = 0; $i < ShowPlaceController::REVIEWS_PER_10_MINUTES; $i++) {
        $this->post('/directory/place/ab-pari/reviews', $payload)->assertRedirect();
    }
    $this->post('/directory/place/ab-pari/reviews', $payload)->assertStatus(429);

    $this->withServerVariables(['REMOTE_ADDR' => '10.0.0.9'])->post('/directory/place/nope/reviews', $payload)->assertNotFound();
});
