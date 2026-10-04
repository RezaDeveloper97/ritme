<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Contracts;

use App\Domain\Shop\Catalog\Data\BrandData;
use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\CategoryTree;

/**
 * Shop taxonomy: the visible category tree and active brands. Cached in the `shop` namespace.
 */
interface CatalogRepository
{
    public function categoryTree(): CategoryTree;

    public function findCategory(string $slug): ?CategoryData;

    /**
     * @return list<BrandData>
     */
    public function brands(): array;
}
