<?php

declare(strict_types=1);

use App\Domain\Directory\Actions\SyncPlaceAmenities;
use App\Domain\Directory\Actions\SyncPlaceGallery;
use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceService;
use App\Domain\Media\Actions\FindMediaUsages;
use App\Domain\Media\Models\Media;

function directoryMedia(string $name): Media
{
    return Media::query()->create([
        'disk' => 'public', 'directory' => '2026/10/'.$name, 'filename' => $name.'.jpg', 'mime' => 'image/jpeg',
        'size' => 100, 'width' => 1200, 'height' => 800, 'alt' => 'تصویر '.$name, 'hash' => hash('sha256', $name),
    ]);
}

it('assigns unique Persian-friendly slugs and keeps history of published places', function (): void {
    $a = Place::factory()->published()->create(['name' => 'استخر آب‌پری', 'slug' => '']);
    $b = Place::factory()->create(['name' => 'استخر آب‌پری', 'slug' => '']);

    expect($a->slug)->toBe('استخر-آب-پری')
        ->and($b->slug)->toBe('استخر-آب-پری-2');

    $a->update(['slug' => 'ab-pari']);
    $repo = app(PlaceRepository::class);
    expect($repo->currentSlugFor('استخر-آب-پری'))->toBe('ab-pari')
        ->and($repo->findPublishedBySlug('ab-pari')?->name)->toBe('استخر آب‌پری')
        ->and($repo->findPublishedBySlug('استخر-آب-پری'))->toBeNull();

    // A new place may not take a slug that still redirects.
    expect(Place::factory()->create(['slug' => 'استخر-آب-پری'])->slug)->toBe('استخر-آب-پری-3');

    // Drafts never public get no history.
    $draft = Place::factory()->create(['slug' => 'draft-one']);
    $draft->update(['slug' => 'draft-two']);
    expect($repo->currentSlugFor('draft-one'))->toBeNull();
});

it('normalises opening hours and phones and drops a district of another city', function (): void {
    $tehran = City::factory()->create();
    $karaj = City::factory()->create();
    $karajDistrict = District::factory()->create(['city_id' => $karaj->id]);

    $place = Place::factory()->create([
        'city_id' => $tehran->id,
        'district_id' => $karajDistrict->id,
        'opening_hours' => ['saturday' => [['9:00', '۲۰:۰۰']], 'nope' => []],
        'phones' => [' ۰۲۱۸۸۰۰۰۰۰۰ ', '', '02188000000'],
    ]);

    expect($place->district_id)->toBeNull()
        ->and($place->opening_hours)->toBe(['saturday' => [['opens' => '09:00', 'closes' => '20:00']]])
        ->and($place->phones)->toBe(['02188000000']);
});

it('keeps price_from at the cheapest announced service', function (): void {
    $place = Place::factory()->create();
    $cheap = PlaceService::factory()->create(['place_id' => $place->id, 'price' => 320_000]);
    PlaceService::factory()->create(['place_id' => $place->id, 'price' => 650_000]);
    PlaceService::factory()->create(['place_id' => $place->id, 'price' => null]);

    expect($place->refresh()->price_from)->toBe(320_000);

    $cheap->update(['price' => 700_000]);
    expect($place->refresh()->price_from)->toBe(650_000);

    PlaceService::query()->where('place_id', $place->id)->get()->each->delete();
    expect($place->refresh()->price_from)->toBeNull();
});

it('serves the full place page DTO and refreshes it after edits, pivots and services', function (): void {
    $place = Place::factory()->published()->withHours()->create(['slug' => 'ab-pari', 'rules' => "قانون یک\nقانون دو"]);
    $amenity = Amenity::factory()->create(['name' => 'اتاق شیردهی']);
    [$m1, $m2] = [directoryMedia('one'), directoryMedia('two')];
    $repo = app(PlaceRepository::class);

    expect($repo->findPublishedBySlug('ab-pari')?->amenities)->toBe([]);

    app(SyncPlaceAmenities::class)->handle($place, [$amenity->id]);
    app(SyncPlaceGallery::class)->handle($place, [$m2->id, $m1->id]);
    PlaceService::factory()->create(['place_id' => $place->id, 'name' => 'آشنایی با آب', 'price' => 320_000]);
    $place->update(['cover_media_id' => $m1->id]);

    $data = $repo->findPublishedBySlug('ab-pari');
    expect($data?->amenities[0]->name)->toBe('اتاق شیردهی')
        ->and($data?->galleryMediaIds)->toBe([$m2->id, $m1->id])
        ->and($data?->imageMediaIds())->toBe([$m1->id, $m2->id])
        ->and($data?->services[0]->name)->toBe('آشنایی با آب')
        ->and($data?->priceFrom)->toBe(320_000)
        ->and($data?->ruleLines())->toBe(['قانون یک', 'قانون دو'])
        ->and($data?->openingHours()->rows()[6]['hours'])->toBe('تعطیل');

    $place->update(['status' => PlaceStatus::Suspended]);
    expect($repo->findPublishedBySlug('ab-pari'))->toBeNull();
});

it('registers cover and gallery media as usages so they are never bulk-deleted', function (): void {
    $place = Place::factory()->create(['name' => 'آب‌پری']);
    [$cover, $photo, $unused] = [directoryMedia('cover'), directoryMedia('photo'), directoryMedia('unused')];
    $place->update(['cover_media_id' => $cover->id]);
    app(SyncPlaceGallery::class)->handle($place, [$photo->id]);

    $usages = app(FindMediaUsages::class)->handle([$cover->id, $photo->id, $unused->id]);

    expect($usages[$cover->id])->toBe(['تصویر کاور مجموعه: آب‌پری'])
        ->and($usages[$photo->id][0])->toStartWith('گالری مجموعه: ')
        ->and($usages)->not->toHaveKey($unused->id);
});
