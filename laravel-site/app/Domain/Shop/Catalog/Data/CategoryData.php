<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

use App\Domain\Shop\Catalog\Models\Category;

/**
 * A shop category (flat; the tree lives in CategoryTree). `illustration` is the local SVG cover fallback.
 */
final readonly class CategoryData
{
    public function __construct(
        public int $id,
        public ?int $parentId,
        public string $name,
        public string $slug,
        public ?string $intro = null,
        public ?int $coverMediaId = null,
        public ?string $illustration = null,
        public int $sortOrder = 0,
    ) {}

    public static function fromModel(Category $category): self
    {
        return new self(
            $category->id, $category->parent_id, $category->name, $category->slug, $category->intro,
            $category->cover_media_id, $category->illustration, $category->sort_order,
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'parentId' => $this->parentId, 'name' => $this->name, 'slug' => $this->slug,
            'intro' => $this->intro, 'coverMediaId' => $this->coverMediaId, 'illustration' => $this->illustration,
            'sortOrder' => $this->sortOrder,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            id: (int) $data['id'],
            parentId: isset($data['parentId']) ? (int) $data['parentId'] : null,
            name: (string) $data['name'],
            slug: (string) $data['slug'],
            intro: isset($data['intro']) ? (string) $data['intro'] : null,
            coverMediaId: isset($data['coverMediaId']) ? (int) $data['coverMediaId'] : null,
            illustration: isset($data['illustration']) ? (string) $data['illustration'] : null,
            sortOrder: (int) ($data['sortOrder'] ?? 0),
        );
    }
}
