<?php

declare(strict_types=1);

use App\Domain\Directory\Data\AmenityData;
use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\DistrictData;
use App\Domain\Directory\Data\LandingComboData;
use App\Domain\Directory\Data\PlaceCardData;
use App\Domain\Directory\Data\PlaceData;
use App\Domain\Directory\Data\PlacePage;
use App\Domain\Directory\Data\PlaceServiceData;
use App\Domain\Directory\Data\RatingSummaryData;
use App\Domain\Directory\Data\ReviewData;
use App\Domain\Directory\Data\ReviewPage;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use Carbon\CarbonImmutable;

function directoryPlaceData(int $ratingCount = 0): PlaceData
{
    return new PlaceData(
        id: 1, name: 'آب‌پری', slug: 'ab-pari',
        category: new CategoryData(2, 'استخر مادر و کودک', 'pool', LocalBusinessType::SportsActivityLocation, 'waves'),
        city: new CityData(1, 'تهران', 'tehran', 'تهران', [new DistrictData(3, 'ونک', 'vanak')]),
        district: new DistrictData(3, 'ونک', 'vanak'),
        summary: 'خلاصه', description: 'توضیح', address: 'خیابان ملاصدرا', postalCode: null,
        latitude: 35.7575, longitude: 51.41, phones: ['02100000000'], website: 'https://example.test',
        ageMinMonths: 6, ageMaxMonths: 48,
        openingHours: ['saturday' => [['opens' => '09:00', 'closes' => '20:00']], 'friday' => []],
        rules: "- کلاه شنا لازم است\n\n• ۱۵ دقیقه زودتر برسید ", cancellationPolicy: 'لغو رایگان تا ۲۴ ساعت قبل',
        coverMediaId: 9, galleryMediaIds: [9, 10, 11],
        amenities: [new AmenityData(4, 'اتاق شیردهی', 'nursing-room', 'bottle', true)],
        services: [
            new PlaceServiceData(5, 'آشنایی با آب', 45, 320_000, 'هر جلسه', 'گروهی', 6, 18),
            new PlaceServiceData(6, 'بسته ۸ جلسه', null, 2_400_000),
            new PlaceServiceData(7, 'مشاوره', null, null),
        ],
        isVerified: true, isDemo: false, ratingAvg: 4.5, ratingCount: $ratingCount, priceFrom: 320_000,
        updatedAt: CarbonImmutable::parse('2026-10-01T09:30:00+03:30'),
    );
}

it('round-trips every DTO through arrays (cache format)', function (): void {
    $place = directoryPlaceData(2);
    $card = new PlaceCardData(1, 'آب‌پری', 'ab-pari', $place->category, $place->city, $place->district, 'خلاصه', 9, true, true, 4.5, 2, 6, 48, 320_000, $place->openingHours, 1.2);
    $review = new ReviewData(1, 'مادر رها', 5, ['cleanliness' => 5], 'متن', CarbonImmutable::parse('2026-09-20T10:00:00+03:30'), true);

    expect(PlaceData::fromArray($place->toArray()))->toEqual($place)
        ->and(PlaceCardData::fromArray($card->toArray()))->toEqual($card)
        ->and(PlacePage::fromArray((new PlacePage([$card], 13, 2, 6))->toArray()))->toEqual(new PlacePage([$card], 13, 2, 6))
        ->and(ReviewPage::fromArray((new ReviewPage([$review], 1, 1, 10))->toArray()))->toEqual(new ReviewPage([$review], 1, 1, 10))
        ->and(RatingSummaryData::fromArray((new RatingSummaryData(4.5, 2, ['staff' => 5.0]))->toArray()))->toEqual(new RatingSummaryData(4.5, 2, ['staff' => 5.0]))
        ->and(LandingComboData::fromArray((new LandingComboData('tehran', 'تهران', null, null, 3, CarbonImmutable::parse('2026-10-01T09:30:00+03:30')))->toArray()))
        ->toEqual(new LandingComboData('tehran', 'تهران', null, null, 3, CarbonImmutable::parse('2026-10-01T09:30:00+03:30')))
        ->and((new PlacePage([$card], 13, 2, 6))->lastPage())->toBe(3);
});

it('exposes page helpers: images, rules, map links, price range', function (): void {
    $place = directoryPlaceData();

    expect($place->imageMediaIds())->toBe([9, 10, 11])
        ->and($place->ruleLines())->toBe(['کلاه شنا لازم است', '۱۵ دقیقه زودتر برسید'])
        ->and($place->mapLinks()?->google)->toContain('35.7575,51.41')
        ->and($place->priceRange())->toBe('۳۲۰٬۰۰۰ تا ۲٬۴۰۰٬۰۰۰ تومان')
        ->and($place->ageRange()->label())->toBe('۶ ماه تا ۴ سال')
        ->and($place->services[0]->ageRange()->label())->toBe('۶ تا ۱۸ ماه');
});

it('builds LocalBusiness data with the category subtype and no rating without real reviews', function (): void {
    $business = directoryPlaceData()->toLocalBusiness('https://ritme.ir/directory/place/ab-pari', ['https://ritme.ir/media/a.jpg']);

    expect($business->type)->toBe(LocalBusinessType::SportsActivityLocation)
        ->and($business->rating)->toBeNull()
        ->and($business->address->streetAddress)->toBe('خیابان ملاصدرا، ونک')
        ->and($business->address->addressLocality)->toBe('تهران')
        ->and($business->telephone)->toBe('02100000000')
        ->and($business->openingHours)->toHaveCount(1)
        ->and($business->priceRange)->toBe('۳۲۰٬۰۰۰ تا ۲٬۴۰۰٬۰۰۰ تومان')
        ->and($business->sameAs)->toBe(['https://example.test']);

    $rated = directoryPlaceData(3)->toLocalBusiness('https://ritme.ir/directory/place/ab-pari');
    expect($rated->rating?->ratingCount)->toBe(3)
        ->and($rated->rating?->ratingValue)->toBe(4.5)
        ->and((new RatingSummaryData)->toSchema())->toBeNull();
});
