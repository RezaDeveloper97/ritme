<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

use App\Domain\Shop\Catalog\Models\Brand;

final readonly class BrandData
{
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public ?int $logoMediaId = null,
    ) {}

    public static function fromModel(Brand $brand): self
    {
        return new self($brand->id, $brand->name, $brand->slug, $brand->logo_media_id);
    }

    /**
     * @return array{id: int, name: string, slug: string, logoMediaId: int|null}
     */
    public function toArray(): array
    {
        return ['id' => $this->id, 'name' => $this->name, 'slug' => $this->slug, 'logoMediaId' => $this->logoMediaId];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self((int) $data['id'], (string) $data['name'], (string) $data['slug'], isset($data['logoMediaId']) ? (int) $data['logoMediaId'] : null);
    }
}
