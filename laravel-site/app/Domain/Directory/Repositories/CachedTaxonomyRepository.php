<?php

declare(strict_types=1);

namespace App\Domain\Directory\Repositories;

use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Data\AmenityData;
use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\LandingComboData;
use App\Domain\Directory\Data\LandingData;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * The taxonomy is small: full lists are cached (`directory` namespace) and single lookups are served from them, so a
 * random slug costs no query once warm.
 */
final class CachedTaxonomyRepository extends CachedRepository implements TaxonomyRepository
{
    public function __construct(private readonly TaxonomyRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'directory';
    }

    public function cities(): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember('cities', fn (): array => array_map(static fn (CityData $c): array => $c->toArray(), $this->inner->cities()));

        return array_map(CityData::fromArray(...), $data);
    }

    public function findCity(string $slug): ?CityData
    {
        foreach ($this->cities() as $city) {
            if ($city->slug === $slug) {
                return $city;
            }
        }

        return null;
    }

    public function categories(): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember('categories', fn (): array => array_map(static fn (CategoryData $c): array => $c->toArray(), $this->inner->categories()));

        return array_map(CategoryData::fromArray(...), $data);
    }

    public function findCategory(string $slug): ?CategoryData
    {
        foreach ($this->categories() as $category) {
            if ($category->slug === $slug) {
                return $category;
            }
        }

        return null;
    }

    public function amenities(): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember('amenities', fn (): array => array_map(static fn (AmenityData $a): array => $a->toArray(), $this->inner->amenities()));

        return array_map(AmenityData::fromArray(...), $data);
    }

    public function landing(int $cityId, ?int $categoryId = null): ?LandingData
    {
        /** @var array<string, mixed>|null $data */
        $data = $this->remember(
            ['landing', $cityId, $categoryId ?? 0],
            fn (): ?array => $this->inner->landing($cityId, $categoryId)?->toArray(),
            cacheNull: true,
        );

        return $data === null ? null : LandingData::fromArray($data);
    }

    public function combos(): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember('combos', fn (): array => array_map(static fn (LandingComboData $c): array => $c->toArray(), $this->inner->combos()));

        return array_map(LandingComboData::fromArray(...), $data);
    }
}
