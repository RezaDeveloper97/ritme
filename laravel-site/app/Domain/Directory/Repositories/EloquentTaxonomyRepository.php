<?php

declare(strict_types=1);

namespace App\Domain\Directory\Repositories;

use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Data\AmenityData;
use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\LandingData;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\Landing;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Queries\LandingCombos;

final class EloquentTaxonomyRepository implements TaxonomyRepository
{
    public function cities(): array
    {
        return City::query()
            ->active()
            ->with('districts')
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get()
            ->map(static fn (City $city): CityData => CityData::fromModel($city, withDistricts: true))
            ->values()
            ->all();
    }

    public function findCity(string $slug): ?CityData
    {
        $city = City::query()
            ->active()
            ->where('slug', $slug)
            ->with('districts')
            ->first();

        return $city === null ? null : CityData::fromModel($city, withDistricts: true);
    }

    public function categories(): array
    {
        return PlaceCategory::query()->active()->orderBy('sort_order')->orderBy('id')->get()
            ->map(CategoryData::fromModel(...))->values()->all();
    }

    public function findCategory(string $slug): ?CategoryData
    {
        $category = PlaceCategory::query()->active()->where('slug', $slug)->first();

        return $category === null ? null : CategoryData::fromModel($category);
    }

    public function amenities(): array
    {
        return Amenity::query()->orderBy('sort_order')->orderBy('id')->get()
            ->map(AmenityData::fromModel(...))->values()->all();
    }

    public function landing(int $cityId, ?int $categoryId = null): ?LandingData
    {
        $landing = Landing::query()
            ->where('city_id', $cityId)
            ->when($categoryId === null, static fn ($q) => $q->whereNull('category_id'), static fn ($q) => $q->where('category_id', $categoryId))
            ->orderBy('id')
            ->first();

        return $landing === null ? null : LandingData::fromModel($landing);
    }

    public function combos(): array
    {
        return (new LandingCombos)->get();
    }
}
