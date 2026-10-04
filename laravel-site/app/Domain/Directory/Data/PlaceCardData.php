<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Support\AgeRange;
use App\Domain\Directory\Support\OpeningHours;
use App\Domain\Seo\Schema\Data\AggregateRatingData;
use Carbon\CarbonInterface;

/**
 * A place in lists (directory cards, search). Opening hours travel as data so «باز است» is evaluated at render time,
 * never cached as a boolean. `ratingCount` counts real approved reviews only (0 → show no rating). `distanceKm` is set
 * only when the search had a reference point. `isDemo` marks placeholder businesses from the design seeder.
 */
final readonly class PlaceCardData
{
    /**
     * @param  array<string, mixed>  $openingHours
     */
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public CategoryData $category,
        public CityData $city,
        public ?DistrictData $district,
        public ?string $summary,
        public ?int $coverMediaId,
        public bool $isVerified,
        public bool $isDemo,
        public float $ratingAvg,
        public int $ratingCount,
        public ?int $ageMinMonths,
        public ?int $ageMaxMonths,
        public ?int $priceFrom,
        public array $openingHours = [],
        public ?float $distanceKm = null,
    ) {}

    /**
     * Expects `category`, `city` and `district` loaded.
     */
    public static function fromModel(Place $place, ?float $distanceKm = null): self
    {
        return new self(
            id: $place->id,
            name: $place->name,
            slug: $place->slug,
            category: CategoryData::fromModel($place->category),
            city: CityData::fromModel($place->city),
            district: $place->district === null ? null : DistrictData::fromModel($place->district),
            summary: $place->summary,
            coverMediaId: $place->cover_media_id,
            isVerified: $place->is_verified,
            isDemo: $place->is_demo,
            ratingAvg: $place->rating_avg,
            ratingCount: $place->rating_count,
            ageMinMonths: $place->age_min_months,
            ageMaxMonths: $place->age_max_months,
            priceFrom: $place->price_from,
            openingHours: $place->openingHours()->toArray(),
            distanceKm: $distanceKm === null ? null : round($distanceKm, 1),
        );
    }

    public function ageRange(): AgeRange
    {
        return new AgeRange($this->ageMinMonths, $this->ageMaxMonths);
    }

    public function openingHours(): OpeningHours
    {
        return OpeningHours::fromArray($this->openingHours);
    }

    public function isOpenAt(CarbonInterface $at): bool
    {
        return $this->openingHours()->isOpenAt($at);
    }

    /** Aggregate of real reviews, or null when there are none (never show or emit an invented rating). */
    public function rating(): ?AggregateRatingData
    {
        return $this->ratingCount < 1 ? null : new AggregateRatingData($this->ratingAvg, $this->ratingCount, $this->ratingCount);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'name' => $this->name, 'slug' => $this->slug,
            'category' => $this->category->toArray(), 'city' => $this->city->toArray(), 'district' => $this->district?->toArray(),
            'summary' => $this->summary, 'coverMediaId' => $this->coverMediaId,
            'isVerified' => $this->isVerified, 'isDemo' => $this->isDemo,
            'ratingAvg' => $this->ratingAvg, 'ratingCount' => $this->ratingCount,
            'ageMinMonths' => $this->ageMinMonths, 'ageMaxMonths' => $this->ageMaxMonths, 'priceFrom' => $this->priceFrom,
            'openingHours' => $this->openingHours, 'distanceKm' => $this->distanceKm,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $int = static fn (string $key): ?int => isset($data[$key]) ? (int) $data[$key] : null;
        /** @var array<string, mixed> $category */
        $category = $data['category'];
        /** @var array<string, mixed> $city */
        $city = $data['city'];
        /** @var array<string, mixed>|null $district */
        $district = $data['district'] ?? null;

        return new self(
            id: (int) $data['id'],
            name: (string) $data['name'],
            slug: (string) $data['slug'],
            category: CategoryData::fromArray($category),
            city: CityData::fromArray($city),
            district: $district === null ? null : DistrictData::fromArray($district),
            summary: isset($data['summary']) ? (string) $data['summary'] : null,
            coverMediaId: $int('coverMediaId'),
            isVerified: (bool) $data['isVerified'],
            isDemo: (bool) ($data['isDemo'] ?? false),
            ratingAvg: (float) $data['ratingAvg'],
            ratingCount: (int) $data['ratingCount'],
            ageMinMonths: $int('ageMinMonths'),
            ageMaxMonths: $int('ageMaxMonths'),
            priceFrom: $int('priceFrom'),
            openingHours: (array) ($data['openingHours'] ?? []),
            distanceKm: isset($data['distanceKm']) ? (float) $data['distanceKm'] : null,
        );
    }
}
