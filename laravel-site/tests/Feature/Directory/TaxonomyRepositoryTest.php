<?php

declare(strict_types=1);

use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Repositories\CachedTaxonomyRepository;
use App\Domain\Directory\Repositories\EloquentTaxonomyRepository;

it('finds active cities (with districts) and categories by slug, never inactive ones', function (): void {
    $city = City::factory()->create(['slug' => 'tehran']);
    District::factory()->create(['city_id' => $city->id, 'slug' => 'tajrish']);
    City::factory()->create(['slug' => 'hidden', 'is_active' => false]);
    PlaceCategory::factory()->create(['slug' => 'clinic']);
    PlaceCategory::factory()->create(['slug' => 'closed', 'is_active' => false]);

    $eloquent = app(EloquentTaxonomyRepository::class);
    $found = $eloquent->findCity('tehran');

    expect($found?->slug)->toBe('tehran')
        ->and(array_map(static fn ($d) => $d->slug, $found->districts ?? []))->toBe(['tajrish'])
        ->and($eloquent->findCity('hidden'))->toBeNull()
        ->and($eloquent->findCity('nowhere'))->toBeNull()
        ->and($eloquent->findCategory('clinic')?->slug)->toBe('clinic')
        ->and($eloquent->findCategory('closed'))->toBeNull();
});

it('answers lookups from the cached list exactly like the database', function (): void {
    $city = City::factory()->create(['slug' => 'shiraz']);
    District::factory()->create(['city_id' => $city->id]);
    City::factory()->create(['slug' => 'off', 'is_active' => false]);
    PlaceCategory::factory()->create(['slug' => 'pool']);

    $cached = app(TaxonomyRepository::class);
    $eloquent = app(EloquentTaxonomyRepository::class);

    expect($cached)->toBeInstanceOf(CachedTaxonomyRepository::class);
    foreach (['shiraz', 'off', 'missing'] as $slug) {
        expect($cached->findCity($slug))->toEqual($eloquent->findCity($slug));
    }
    foreach (['pool', 'missing'] as $slug) {
        expect($cached->findCategory($slug))->toEqual($eloquent->findCategory($slug));
    }
});
