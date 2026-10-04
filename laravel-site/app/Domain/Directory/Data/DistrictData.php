<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Models\District;

final readonly class DistrictData
{
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
    ) {}

    public static function fromModel(District $district): self
    {
        return new self($district->id, $district->name, $district->slug);
    }

    /**
     * @return array{id: int, name: string, slug: string}
     */
    public function toArray(): array
    {
        return ['id' => $this->id, 'name' => $this->name, 'slug' => $this->slug];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self((int) $data['id'], (string) $data['name'], (string) $data['slug']);
    }
}
