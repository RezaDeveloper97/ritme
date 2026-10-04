<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Contracts;

use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Data\ProductData;
use App\Domain\Shop\Catalog\Data\ProductPage;
use App\Domain\Shop\Catalog\Data\ReviewPage;

/**
 * Read side of the shop products (published products only). Cached in the `shop` namespace. Stock shown from here may
 * be up to one cache lifetime old: cart and checkout (L6-04/05) must re-check through AdjustStock / the models.
 */
interface ProductRepository
{
    public function findPublishedBySlug(string $slug): ?ProductData;

    /**
     * Current slug of the published product that used to live at $previousSlug (slug history → 301), or null.
     */
    public function currentSlugFor(string $previousSlug): ?string;

    /** Listing with filters, sort and facets (ProductsInCategory). */
    public function list(ProductCriteria $criteria): ProductPage;

    /**
     * @return list<ProductCardData>
     */
    public function bestSellers(int $limit = 10, ?int $categoryId = null): array;

    /**
     * «معمولاً با این می‌خرند» for a product page.
     *
     * @return list<ProductCardData>
     */
    public function frequentlyBoughtWith(int $productId, int $limit = 5): array;

    /**
     * Approved reviews of a product, newest first (demo samples included and flagged).
     */
    public function reviews(int $productId, int $page = 1, int $perPage = 10): ReviewPage;
}
