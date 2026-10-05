<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Support;

use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\ProductCardData;

/**
 * Which shop lists may be indexed (L6-02, L7-05b) — one rule for ShopHomeController / CategoryController (robots) and
 * PagesSitemapProvider (`/shop` in `pages.xml`): a list that shows no real (non-demo) product is thin content →
 * `noindex,follow` and out of the sitemap. `homeDepartments()` is what the shop home shows, so both sides judge the
 * same cards.
 */
final class ProductListIndexing
{
    /** Product cards per department row on the shop home (design: 5). */
    public const PRODUCTS_PER_DEPARTMENT = 5;

    public function __construct(
        private readonly CatalogRepository $catalog,
        private readonly ProductRepository $products,
    ) {}

    /**
     * @param  list<ProductCardData>  $cards  the cards the page shows
     */
    public static function showsRealProduct(array $cards): bool
    {
        foreach ($cards as $card) {
            if (! $card->isDemo) {
                return true;
            }
        }

        return false;
    }

    /**
     * Shop home rows: every department (root category, admin order) with its best-seller cards.
     *
     * @return list<array{0: CategoryData, 1: list<ProductCardData>}>
     */
    public function homeDepartments(): array
    {
        $rows = [];
        foreach ($this->catalog->categoryTree()->roots() as $root) {
            $rows[] = [$root, $this->products->bestSellers(self::PRODUCTS_PER_DEPARTMENT, $root->id)];
        }

        return $rows;
    }

    public function homeIndexable(): bool
    {
        foreach ($this->homeDepartments() as [, $cards]) {
            if (self::showsRealProduct($cards)) {
                return true;
            }
        }

        return false;
    }
}
