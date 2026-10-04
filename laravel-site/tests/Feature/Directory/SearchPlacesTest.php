<?php

declare(strict_types=1);

use App\Domain\Directory\Actions\SyncPlaceAmenities;
use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\PlaceSearchCriteria;
use App\Domain\Directory\Enums\PlaceSort;
use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Directory\Models\PlaceService;
use Carbon\CarbonImmutable;
use Illuminate\Support\Facades\DB;

/**
 * @return list<string>
 */
function placeSlugs(PlaceSearchCriteria $criteria): array
{
    return array_map(static fn ($card): string => $card->slug, app(PlaceRepository::class)->search($criteria)->items);
}

beforeEach(function (): void {
    $this->tehran = City::factory()->create(['slug' => 'tehran', 'name' => 'تهران']);
    $this->karaj = City::factory()->create(['slug' => 'karaj', 'name' => 'کرج']);
    $this->vanak = District::factory()->create(['city_id' => $this->tehran->id, 'slug' => 'vanak', 'name' => 'ونک']);
    $this->pool = PlaceCategory::factory()->create(['slug' => 'pool', 'name' => 'استخر مادر و کودک']);
    $this->playhouse = PlaceCategory::factory()->create(['slug' => 'playhouse', 'name' => 'خانه بازی']);
});

it('returns published places only, filtered by city, district and category', function (): void {
    $base = ['city_id' => $this->tehran->id, 'category_id' => $this->pool->id];
    Place::factory()->published()->create([...$base, 'slug' => 'a', 'district_id' => $this->vanak->id]);
    Place::factory()->published()->create([...$base, 'slug' => 'b']);
    Place::factory()->published()->create(['city_id' => $this->tehran->id, 'category_id' => $this->playhouse->id, 'slug' => 'c']);
    Place::factory()->published()->create(['city_id' => $this->karaj->id, 'category_id' => $this->pool->id, 'slug' => 'd']);
    Place::factory()->create([...$base, 'slug' => 'draft']);
    Place::factory()->create([...$base, 'slug' => 'suspended', 'status' => PlaceStatus::Suspended]);

    expect(placeSlugs(new PlaceSearchCriteria))->toEqualCanonicalizing(['a', 'b', 'c', 'd'])
        ->and(placeSlugs(new PlaceSearchCriteria(cityId: $this->tehran->id)))->toEqualCanonicalizing(['a', 'b', 'c'])
        ->and(placeSlugs(new PlaceSearchCriteria(cityId: $this->tehran->id, categoryId: $this->pool->id)))->toEqualCanonicalizing(['a', 'b'])
        ->and(placeSlugs(new PlaceSearchCriteria(districtId: $this->vanak->id)))->toBe(['a']);
});

it('filters by child age inside the place range (open bounds match)', function (): void {
    $base = ['city_id' => $this->tehran->id, 'category_id' => $this->pool->id];
    Place::factory()->published()->create([...$base, 'slug' => 'baby', 'age_min_months' => 1, 'age_max_months' => 12]);
    Place::factory()->published()->create([...$base, 'slug' => 'toddler', 'age_min_months' => 12, 'age_max_months' => 36]);
    Place::factory()->published()->create([...$base, 'slug' => 'any']);
    Place::factory()->published()->create([...$base, 'slug' => 'from-two', 'age_min_months' => 24]);

    expect(placeSlugs(new PlaceSearchCriteria(ageMonths: 6)))->toEqualCanonicalizing(['baby', 'any'])
        ->and(placeSlugs(new PlaceSearchCriteria(ageMonths: 12)))->toEqualCanonicalizing(['baby', 'toddler', 'any'])
        ->and(placeSlugs(new PlaceSearchCriteria(ageMonths: 30)))->toEqualCanonicalizing(['toddler', 'any', 'from-two']);
});

it('requires every selected amenity', function (): void {
    $nursing = Amenity::factory()->create();
    $coach = Amenity::factory()->create();
    $base = ['city_id' => $this->tehran->id, 'category_id' => $this->pool->id];
    $both = Place::factory()->published()->create([...$base, 'slug' => 'both']);
    $one = Place::factory()->published()->create([...$base, 'slug' => 'one']);
    Place::factory()->published()->create([...$base, 'slug' => 'none']);
    app(SyncPlaceAmenities::class)->handle($both, [$nursing->id, $coach->id]);
    app(SyncPlaceAmenities::class)->handle($one, [$nursing->id]);

    expect(placeSlugs(new PlaceSearchCriteria(amenityIds: [$nursing->id])))->toEqualCanonicalizing(['both', 'one'])
        ->and(placeSlugs(new PlaceSearchCriteria(amenityIds: [$nursing->id, $coach->id])))->toBe(['both']);
});

it('matches text tokens with Persian normalisation, incl. the category name', function (): void {
    $base = ['city_id' => $this->tehran->id];
    Place::factory()->published()->create([...$base, 'category_id' => $this->pool->id, 'slug' => 'ab-pari', 'name' => 'آب‌پری', 'summary' => 'کلاس شنا']);
    Place::factory()->published()->create([...$base, 'category_id' => $this->playhouse->id, 'slug' => 'tab', 'name' => 'تاب‌تاب', 'summary' => 'بازي کودك']);

    expect(placeSlugs(new PlaceSearchCriteria(text: 'آبپری')))->toBe(['ab-pari'])      // ZWNJ ignored
        ->and(placeSlugs(new PlaceSearchCriteria(text: 'بازی کودک')))->toBe(['tab'])    // Arabic ي/ك in the data
        ->and(placeSlugs(new PlaceSearchCriteria(text: 'استخر')))->toBe(['ab-pari'])    // category name
        ->and(placeSlugs(new PlaceSearchCriteria(text: 'شنا استخر')))->toBe(['ab-pari'])
        ->and(placeSlugs(new PlaceSearchCriteria(text: 'شنا بازی')))->toBe([])
        ->and(placeSlugs(new PlaceSearchCriteria(text: '100%_')))->toBe([]);
});

it('keeps only places open at the given moment and paginates after filtering', function (): void {
    $base = ['city_id' => $this->tehran->id, 'category_id' => $this->pool->id];
    Place::factory()->published()->withHours()->create([...$base, 'slug' => 'day']);
    Place::factory()->published()->create([...$base, 'slug' => 'night', 'opening_hours' => ['thursday' => [['opens' => '18:00', 'closes' => '02:00']]]]);
    Place::factory()->published()->create([...$base, 'slug' => 'unknown']);

    $thursdayNoon = CarbonImmutable::parse('2026-10-08 12:00', 'Asia/Tehran');
    $thursdayLate = CarbonImmutable::parse('2026-10-08 23:30', 'Asia/Tehran');
    $fridayEarly = CarbonImmutable::parse('2026-10-09 01:00', 'Asia/Tehran');

    expect(placeSlugs(new PlaceSearchCriteria(openAt: $thursdayNoon)))->toBe(['day'])
        ->and(placeSlugs(new PlaceSearchCriteria(openAt: $thursdayLate)))->toBe(['night'])
        ->and(placeSlugs(new PlaceSearchCriteria(openAt: $fridayEarly)))->toBe(['night'])
        ->and(app(PlaceRepository::class)->search(new PlaceSearchCriteria(openAt: $thursdayNoon, perPage: 1))->total)->toBe(1);
});

it('sorts by recommendation, rating, price, newest and distance', function (): void {
    $base = ['city_id' => $this->tehran->id, 'category_id' => $this->pool->id];
    $far = Place::factory()->published()->create([...$base, 'slug' => 'far', 'latitude' => 35.80, 'longitude' => 51.45]);
    $near = Place::factory()->published()->verified()->create([...$base, 'slug' => 'near', 'latitude' => 35.7575, 'longitude' => 51.41]);
    $nowhere = Place::factory()->published()->create([...$base, 'slug' => 'nowhere']);

    PlaceService::factory()->create(['place_id' => $far->id, 'price' => 100_000]);
    PlaceService::factory()->create(['place_id' => $near->id, 'price' => 500_000]);
    PlaceReview::factory()->approved()->create(['place_id' => $far->id, 'rating' => 5]);
    PlaceReview::factory()->approved()->create(['place_id' => $nowhere->id, 'rating' => 3]);

    expect(placeSlugs(new PlaceSearchCriteria))->toBe(['near', 'far', 'nowhere'])              // verified first, no paid boost
        ->and(placeSlugs(new PlaceSearchCriteria(sort: PlaceSort::Rating)))->toBe(['far', 'nowhere', 'near'])
        ->and(placeSlugs(new PlaceSearchCriteria(sort: PlaceSort::Price)))->toBe(['far', 'near', 'nowhere'])
        ->and(placeSlugs(new PlaceSearchCriteria(sort: PlaceSort::Newest)))->toBe(['nowhere', 'near', 'far'])
        ->and(placeSlugs(new PlaceSearchCriteria(sort: PlaceSort::Nearest)))->toBe(['near', 'far', 'nowhere']); // no point → recommended

    $page = app(PlaceRepository::class)->search(new PlaceSearchCriteria(sort: PlaceSort::Nearest, latitude: 35.7600, longitude: 51.4100));
    expect(array_map(static fn ($c) => $c->slug, $page->items))->toBe(['near', 'far', 'nowhere'])
        ->and($page->items[0]->distanceKm)->toBe(0.3)
        ->and($page->items[1]->distanceKm)->toBeGreaterThan(5.0)
        ->and($page->items[2]->distanceKm)->toBeNull();
});

it('paginates with totals and serves warm searches from the directory cache', function (): void {
    Place::factory()->count(5)->published()->create(['city_id' => $this->tehran->id, 'category_id' => $this->pool->id]);
    $repo = app(PlaceRepository::class);

    $page = $repo->search(new PlaceSearchCriteria(page: 2, perPage: 2));
    expect($page->total)->toBe(5)->and($page->items)->toHaveCount(2)->and($page->lastPage())->toBe(3)
        ->and($repo->search(new PlaceSearchCriteria(page: 4, perPage: 2))->isOutOfRange())->toBeTrue();

    DB::enableQueryLog();
    $repo->search(new PlaceSearchCriteria(page: 2, perPage: 2));
    expect(DB::getQueryLog())->toBe([]);
    DB::disableQueryLog();

    Place::factory()->published()->create(['city_id' => $this->tehran->id, 'category_id' => $this->pool->id]);
    expect($repo->search(new PlaceSearchCriteria(page: 2, perPage: 2))->total)->toBe(6);
});

it('returns cards with taxonomy, price, rating and render-time opening hours', function (): void {
    $place = Place::factory()->published()->verified()->withHours()->create([
        'city_id' => $this->tehran->id, 'district_id' => $this->vanak->id, 'category_id' => $this->pool->id,
        'age_min_months' => 6, 'age_max_months' => 48,
    ]);
    PlaceService::factory()->create(['place_id' => $place->id, 'price' => 320_000]);

    $card = app(PlaceRepository::class)->search(new PlaceSearchCriteria)->items[0];

    expect($card->category->slug)->toBe('pool')
        ->and($card->city->slug)->toBe('tehran')
        ->and($card->district?->name)->toBe('ونک')
        ->and($card->priceFrom)->toBe(320_000)
        ->and($card->rating())->toBeNull()
        ->and($card->ageRange()->label())->toBe('۶ ماه تا ۴ سال')
        ->and($card->isOpenAt(CarbonImmutable::parse('2026-10-03 10:00', 'Asia/Tehran')))->toBeTrue()
        ->and($card->isOpenAt(CarbonImmutable::parse('2026-10-09 10:00', 'Asia/Tehran')))->toBeFalse();
});
