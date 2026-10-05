<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Repositories;

use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Data\BrandData;
use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\CategoryTree;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Queries\VisibleCategories;

final class EloquentCatalogRepository implements CatalogRepository
{
    public function categoryTree(): CategoryTree
    {
        return (new VisibleCategories)->get();
    }

    public function findCategory(string $slug): ?CategoryData
    {
        return $this->categoryTree()->findBySlug($slug);
    }

    public function brands(): array
    {
        return array_values(Brand::query()->active()->orderBy('sort_order')->orderBy('name')->get()
            ->map(BrandData::fromModel(...))
            ->all());
    }
}
