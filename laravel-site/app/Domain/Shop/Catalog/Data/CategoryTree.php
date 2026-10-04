<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

/**
 * The visible category tree: active categories whose ancestors are all active, in display order (sort_order, id).
 * Breadcrumbs use path(), listings descendantIds() (a category lists its subcategories' products too), the subnav
 * roots() / children().
 */
final readonly class CategoryTree
{
    /** @var array<int, CategoryData> */
    private array $byId;

    /**
     * @param  list<CategoryData>  $categories
     */
    public function __construct(public array $categories)
    {
        $byId = [];
        foreach ($categories as $category) {
            $byId[$category->id] = $category;
        }
        $this->byId = $byId;
    }

    public function find(int $id): ?CategoryData
    {
        return $this->byId[$id] ?? null;
    }

    public function findBySlug(string $slug): ?CategoryData
    {
        foreach ($this->categories as $category) {
            if ($category->slug === $slug) {
                return $category;
            }
        }

        return null;
    }

    /**
     * @return list<CategoryData>
     */
    public function roots(): array
    {
        return $this->children(null);
    }

    /**
     * @return list<CategoryData>
     */
    public function children(?int $parentId): array
    {
        return array_values(array_filter($this->categories, static fn (CategoryData $c): bool => $c->parentId === $parentId));
    }

    /**
     * Ancestors and the category itself, root first (empty when the id is not visible).
     *
     * @return list<CategoryData>
     */
    public function path(int $id): array
    {
        $path = [];
        for ($current = $this->find($id); $current !== null && count($path) <= count($this->categories); $current = $current->parentId === null ? null : $this->find($current->parentId)) {
            array_unshift($path, $current);
        }

        return $path;
    }

    /**
     * The category and every visible descendant.
     *
     * @return list<int>
     */
    public function descendantIds(int $id): array
    {
        if ($this->find($id) === null) {
            return [];
        }

        $ids = [$id];
        for ($i = 0; $i < count($ids); $i++) {
            foreach ($this->children($ids[$i]) as $child) {
                $ids[] = $child->id;
            }
        }

        return $ids;
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function toArray(): array
    {
        return array_map(static fn (CategoryData $c): array => $c->toArray(), $this->categories);
    }

    /**
     * @param  list<array<string, mixed>>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(array_map(CategoryData::fromArray(...), $data));
    }
}
