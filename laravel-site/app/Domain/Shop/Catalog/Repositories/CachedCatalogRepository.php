<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Repositories;

use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Data\BrandData;
use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\CategoryTree;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * The whole visible category tree is one cache entry (a shop has dozens of categories, not thousands); slug lookups
 * read it. Brands are another entry. `shop` namespace, bumped by the taxonomy observer.
 */
final class CachedCatalogRepository extends CachedRepository implements CatalogRepository
{
    public function __construct(private readonly CatalogRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'shop';
    }

    public function categoryTree(): CategoryTree
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember('category-tree', fn (): array => $this->inner->categoryTree()->toArray());

        return CategoryTree::fromArray($data);
    }

    public function findCategory(string $slug): ?CategoryData
    {
        return $slug === '' ? null : $this->categoryTree()->findBySlug($slug);
    }

    public function brands(): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember('brands', fn (): array => array_map(static fn (BrandData $b): array => $b->toArray(), $this->inner->brands()));

        return array_map(BrandData::fromArray(...), $data);
    }
}
