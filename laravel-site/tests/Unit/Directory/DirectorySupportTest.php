<?php

declare(strict_types=1);

use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\LandingData;
use App\Domain\Directory\Data\PlaceSearchCriteria;
use App\Domain\Directory\Enums\PlaceSort;
use App\Domain\Directory\Support\AgeRange;
use App\Domain\Directory\Support\LandingCopy;
use App\Domain\Directory\Support\MapLinks;
use Carbon\CarbonImmutable;

beforeEach(function (): void {
    $this->previousTimezone = date_default_timezone_get();
    date_default_timezone_set('Asia/Tehran');
});

afterEach(function (): void {
    date_default_timezone_set($this->previousTimezone);
});

it('labels age ranges like the design', function (?int $min, ?int $max, ?string $label): void {
    expect((new AgeRange($min, $max))->label())->toBe($label);
})->with([
    [6, 48, '۶ ماه تا ۴ سال'],
    [12, 72, '۱ تا ۶ سال'],
    [12, 36, '۱۲ تا ۳۶ ماه'],
    [18, 60, '۱۸ ماه تا ۵ سال'],
    [1, 12, '۱ تا ۱۲ ماه'],
    [6, null, 'از ۶ ماه'],
    [null, 48, 'تا ۴ سال'],
    [null, null, null],
]);

it('checks whether an age fits a range', function (): void {
    $range = new AgeRange(6, 48);

    expect($range->contains(6))->toBeTrue()
        ->and($range->contains(48))->toBeTrue()
        ->and($range->contains(5))->toBeFalse()
        ->and($range->contains(49))->toBeFalse()
        ->and((new AgeRange(null, null))->contains(100))->toBeTrue();
});

it('builds map deep links without embedding a map', function (): void {
    $links = MapLinks::at(35.7575, 51.41, 'آب‌پری');

    expect($links->geo)->toStartWith('geo:35.7575,51.41?q=35.7575,51.41(')
        ->and($links->neshan)->toBe('https://neshan.org/maps/@35.7575,51.41,16z,0p')
        ->and($links->balad)->toBe('https://balad.ir/location?latitude=35.7575&longitude=51.41&zoom=16')
        ->and($links->google)->toBe('https://www.google.com/maps/search/?api=1&query=35.7575,51.41');
});

it('fills landing copy from templates where the admin left fields empty', function (): void {
    $city = new CityData(1, 'تهران', 'tehran');
    $category = new CategoryData(2, 'استخر مادر و کودک', 'pool');

    $default = LandingCopy::resolve($city, $category);
    expect($default->h1)->toBe('استخر مادر و کودک در تهران')
        ->and($default->title)->toBe('استخر مادر و کودک در تهران')
        ->and($default->description)->toContain('استخر مادر و کودک در تهران')
        ->and($default->intro)->toBeNull();

    $edited = LandingCopy::resolve($city, null, new LandingData(1, null, 'عنوان سفارشی', ' ', null, 'مقدمه'));
    expect($edited->h1)->toBe('عنوان سفارشی')
        ->and($edited->title)->toBe('خدمات مادر و کودک در تهران')
        ->and($edited->intro)->toBe('مقدمه')
        ->and($edited->categoryId)->toBeNull();

    foreach ([$default, $edited] as $copy) {
        expect((string) $copy->description)->not->toMatch('/حتماً|قطعاً|دقیق‌ترین|تضمینی|بهترین/u');
    }
});

it('normalises search criteria for stable cache keys', function (): void {
    $at = CarbonImmutable::parse('2026-10-03 10:07:42', 'Asia/Tehran');
    $a = new PlaceSearchCriteria(cityId: 1, amenityIds: [3, 1, 3, 0], openAt: $at, text: ' استخر  ك ', page: 0, perPage: 500);
    $b = new PlaceSearchCriteria(cityId: 1, amenityIds: [1, 3], openAt: $at->addMinutes(2), text: 'استخر');

    expect($a->amenityIds)->toBe([1, 3])
        ->and($a->openAt?->format('H:i'))->toBe('10:05')
        ->and($a->page)->toBe(1)
        ->and($a->perPage)->toBe(PlaceSearchCriteria::MAX_PER_PAGE)
        ->and($a->tokens)->toBe(['استخر'])
        ->and($a->hasReferencePoint())->toBeFalse()
        ->and((new PlaceSearchCriteria(cityId: 1, amenityIds: [1, 3], openAt: $at, text: 'استخر', perPage: 60))->cacheKey())->toBe($a->cacheKey())
        ->and($b->cacheKey())->not->toBe($a->cacheKey())
        ->and((new PlaceSearchCriteria(sort: PlaceSort::Price))->cacheKey())->not->toBe((new PlaceSearchCriteria)->cacheKey());
});
