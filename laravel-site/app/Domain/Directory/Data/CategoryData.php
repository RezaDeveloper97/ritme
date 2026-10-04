<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;

final readonly class CategoryData
{
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public LocalBusinessType $schemaType = LocalBusinessType::LocalBusiness,
        public ?string $icon = null,
        public ?string $description = null,
    ) {}

    public static function fromModel(PlaceCategory $category): self
    {
        return new self($category->id, $category->name, $category->slug, $category->schema_type, $category->icon, $category->description);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'name' => $this->name, 'slug' => $this->slug, 'schemaType' => $this->schemaType->value,
            'icon' => $this->icon, 'description' => $this->description,
        ];
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
            schemaType: LocalBusinessType::tryFrom((string) ($data['schemaType'] ?? '')) ?? LocalBusinessType::LocalBusiness,
            icon: isset($data['icon']) ? (string) $data['icon'] : null,
            description: isset($data['description']) ? (string) $data['description'] : null,
        );
    }
}
