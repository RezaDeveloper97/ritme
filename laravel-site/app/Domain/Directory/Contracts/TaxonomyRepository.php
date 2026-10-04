<?php

declare(strict_types=1);

namespace App\Domain\Directory\Contracts;

use App\Domain\Directory\Data\AmenityData;
use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\LandingComboData;
use App\Domain\Directory\Data\LandingData;

/**
 * Cities (with districts), categories, amenities, landing copy and the landing pages that have places. Cached in the
 * `directory` namespace.
 */
interface TaxonomyRepository
{
    /**
     * Active cities in sort order, each with its districts.
     *
     * @return list<CityData>
     */
    public function cities(): array;

    public function findCity(string $slug): ?CityData;

    /**
     * Active categories in sort order.
     *
     * @return list<CategoryData>
     */
    public function categories(): array;

    public function findCategory(string $slug): ?CategoryData;

    /**
     * @return list<AmenityData>
     */
    public function amenities(): array;

    /**
     * The admin-edited copy of a landing page (raw; resolve with LandingCopy), or null.
     */
    public function landing(int $cityId, ?int $categoryId = null): ?LandingData;

    /**
     * Cities and city × category combinations with at least one published place.
     *
     * @return list<LandingComboData>
     */
    public function combos(): array;
}
