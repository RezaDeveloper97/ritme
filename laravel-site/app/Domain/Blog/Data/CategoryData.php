<?php

declare(strict_types=1);

namespace App\Domain\Blog\Data;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Models\Category;

/**
 * A magazine category for views. `label` is the short card label (falls back to the name); `postCount` counts
 * published posts in the category itself (0 where not loaded, e.g. on article cards).
 */
final readonly class CategoryData
{
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public string $label,
        public ?int $parentId = null,
        public ?LifeStage $lifeStage = null,
        public ?string $description = null,
        public int $postCount = 0,
    ) {}

    public static function fromModel(Category $category, int $postCount = 0): self
    {
        return new self(
            id: $category->id,
            name: $category->name,
            slug: $category->slug,
            label: $category->label !== null && $category->label !== '' ? $category->label : $category->name,
            parentId: $category->parent_id,
            lifeStage: $category->life_stage,
            description: $category->description,
            postCount: $postCount,
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'name' => $this->name, 'slug' => $this->slug, 'label' => $this->label,
            'parentId' => $this->parentId, 'lifeStage' => $this->lifeStage?->value, 'description' => $this->description,
            'postCount' => $this->postCount,
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
            label: (string) $data['label'],
            parentId: isset($data['parentId']) ? (int) $data['parentId'] : null,
            lifeStage: isset($data['lifeStage']) ? LifeStage::tryFrom((string) $data['lifeStage']) : null,
            description: isset($data['description']) ? (string) $data['description'] : null,
            postCount: (int) ($data['postCount'] ?? 0),
        );
    }
}
