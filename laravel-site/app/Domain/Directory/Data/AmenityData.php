<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Models\Amenity;

final readonly class AmenityData
{
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public ?string $icon = null,
        public bool $isFilter = false,
    ) {}

    public static function fromModel(Amenity $amenity): self
    {
        return new self($amenity->id, $amenity->name, $amenity->slug, $amenity->icon, $amenity->is_filter);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return ['id' => $this->id, 'name' => $this->name, 'slug' => $this->slug, 'icon' => $this->icon, 'isFilter' => $this->isFilter];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            id: (int) $data['id'],
            name: (string) $data['name'],
            slug: (string) $data['slug'],
            icon: isset($data['icon']) ? (string) $data['icon'] : null,
            isFilter: (bool) ($data['isFilter'] ?? false),
        );
    }
}
