<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Models\City;

/**
 * A city; `districts` is filled only where the city list needs them (TaxonomyRepository::cities()).
 */
final readonly class CityData
{
    /**
     * @param  list<DistrictData>  $districts
     */
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public ?string $province = null,
        public array $districts = [],
    ) {}

    public static function fromModel(City $city, bool $withDistricts = false): self
    {
        return new self(
            id: $city->id,
            name: $city->name,
            slug: $city->slug,
            province: $city->province,
            districts: $withDistricts ? $city->districts->map(DistrictData::fromModel(...))->values()->all() : [],
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'name' => $this->name, 'slug' => $this->slug, 'province' => $this->province,
            'districts' => array_map(static fn (DistrictData $d): array => $d->toArray(), $this->districts),
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        /** @var list<array<string, mixed>> $districts */
        $districts = (array) ($data['districts'] ?? []);

        return new self(
            id: (int) $data['id'],
            name: (string) $data['name'],
            slug: (string) $data['slug'],
            province: isset($data['province']) ? (string) $data['province'] : null,
            districts: array_map(DistrictData::fromArray(...), $districts),
        );
    }
}
