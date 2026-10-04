<?php

declare(strict_types=1);

use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Landing;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Support\LandingCopy;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use Illuminate\Support\Facades\DB;

it('lists active cities with districts, categories and amenities in sort order (cached)', function (): void {
    $tehran = City::factory()->create(['name' => 'تهران', 'slug' => 'tehran', 'sort_order' => 1]);
    City::factory()->create(['name' => 'کرج', 'slug' => 'karaj', 'sort_order' => 2]);
    City::factory()->create(['slug' => 'hidden', 'is_active' => false]);
    District::factory()->create(['city_id' => $tehran->id, 'name' => 'ونک', 'slug' => 'vanak', 'sort_order' => 2]);
    District::factory()->create(['city_id' => $tehran->id, 'name' => 'جردن', 'slug' => 'jordan', 'sort_order' => 1]);
    PlaceCategory::factory()->create(['slug' => 'pool', 'schema_type' => LocalBusinessType::SportsActivityLocation, 'sort_order' => 2]);
    PlaceCategory::factory()->create(['slug' => 'class', 'schema_type' => LocalBusinessType::ChildCare, 'sort_order' => 1]);
    PlaceCategory::factory()->create(['slug' => 'off', 'is_active' => false]);
    Amenity::factory()->filter()->create(['slug' => 'nursing-room']);
    $repo = app(TaxonomyRepository::class);

    expect(array_map(static fn ($c) => $c->slug, $repo->cities()))->toBe(['tehran', 'karaj'])
        ->and(array_map(static fn ($d) => $d->slug, $repo->findCity('tehran')->districts ?? []))->toBe(['jordan', 'vanak'])
        ->and($repo->findCity('hidden'))->toBeNull()
        ->and(array_map(static fn ($c) => $c->slug, $repo->categories()))->toBe(['class', 'pool'])
        ->and($repo->findCategory('pool')?->schemaType)->toBe(LocalBusinessType::SportsActivityLocation)
        ->and($repo->findCategory('off'))->toBeNull()
        ->and($repo->amenities()[0]->isFilter)->toBeTrue();

    DB::enableQueryLog();
    $repo->findCity('tehran');
    $repo->findCity('random-slug');
    $repo->findCategory('pool');
    expect(DB::getQueryLog())->toBe([]);
    DB::disableQueryLog();

    $tehran->update(['name' => 'تهران بزرگ']);
    expect($repo->findCity('tehran')?->name)->toBe('تهران بزرگ');
});

it('fills empty slugs, scopes district slugs to their city and keeps city slugs off fixed routes', function (): void {
    $a = City::factory()->create(['name' => 'Place', 'slug' => '']);
    $b = City::factory()->create(['name' => 'شیراز', 'slug' => '']);
    $c = City::factory()->create(['name' => 'Tabriz', 'slug' => '']);

    expect($a->slug)->toBe('place-2')
        ->and($b->slug)->toBe('شیراز')
        ->and(District::factory()->create(['city_id' => $b->id, 'name' => 'مرکز', 'slug' => ''])->slug)->toBe('مرکز')
        ->and(District::factory()->create(['city_id' => $c->id, 'name' => 'مرکز', 'slug' => ''])->slug)->toBe('مرکز')
        ->and(District::factory()->create(['city_id' => $c->id, 'name' => 'مرکز', 'slug' => ''])->slug)->toBe('مرکز-2');
});

it('returns admin landing copy and resolves it with the default templates', function (): void {
    $city = City::factory()->create(['name' => 'تهران', 'slug' => 'tehran']);
    $pool = PlaceCategory::factory()->create(['name' => 'استخر مادر و کودک', 'slug' => 'pool']);
    $repo = app(TaxonomyRepository::class);

    expect($repo->landing($city->id))->toBeNull();

    Landing::query()->create(['city_id' => $city->id, 'intro' => 'مقدمه تهران']);
    Landing::query()->create(['city_id' => $city->id, 'category_id' => $pool->id, 'h1' => 'استخرهای مادر و کودک تهران']);

    $cityCopy = LandingCopy::resolve($repo->findCity('tehran'), null, $repo->landing($city->id));
    $poolCopy = LandingCopy::resolve($repo->findCity('tehran'), $repo->findCategory('pool'), $repo->landing($city->id, $pool->id));

    expect($cityCopy->intro)->toBe('مقدمه تهران')
        ->and($cityCopy->h1)->toBe('خدمات مادر و کودک در تهران')
        ->and($poolCopy->h1)->toBe('استخرهای مادر و کودک تهران')
        ->and($poolCopy->title)->toBe('استخر مادر و کودک در تهران');
});

it('lists cities and city x category combos that have published places', function (): void {
    $tehran = City::factory()->create(['slug' => 'tehran', 'sort_order' => 1]);
    $karaj = City::factory()->create(['slug' => 'karaj', 'sort_order' => 2]);
    City::factory()->create(['slug' => 'empty', 'sort_order' => 3]);
    $pool = PlaceCategory::factory()->create(['slug' => 'pool', 'sort_order' => 1]);
    $class = PlaceCategory::factory()->create(['slug' => 'class', 'sort_order' => 2]);
    Place::factory()->count(2)->published()->create(['city_id' => $tehran->id, 'category_id' => $pool->id]);
    Place::factory()->published()->create(['city_id' => $tehran->id, 'category_id' => $class->id]);
    Place::factory()->create(['city_id' => $karaj->id, 'category_id' => $pool->id]); // draft
    Place::factory()->published()->demo()->create(['city_id' => $karaj->id, 'category_id' => $class->id]);

    $combos = array_map(static fn ($c): string => $c->citySlug.'/'.($c->categorySlug ?? '*').':'.$c->placeCount, app(TaxonomyRepository::class)->combos());

    expect($combos)->toBe(['tehran/*:3', 'tehran/pool:2', 'tehran/class:1', 'karaj/*:1', 'karaj/class:1']);
});
