<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Repositories;

use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Data\ProductData;
use App\Domain\Shop\Catalog\Data\ProductPage;
use App\Domain\Shop\Catalog\Data\ReviewData;
use App\Domain\Shop\Catalog\Data\ReviewPage;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Models\ProductSlug;
use App\Domain\Shop\Catalog\Queries\BestSellers;
use App\Domain\Shop\Catalog\Queries\FrequentlyBoughtWith;
use App\Domain\Shop\Catalog\Queries\ProductsInCategory;

final class EloquentProductRepository implements ProductRepository
{
    public function __construct(private readonly CatalogRepository $catalog) {}

    public function findPublishedBySlug(string $slug): ?ProductData
    {
        $product = Product::query()
            ->published()
            ->where('slug', $slug)
            ->with(['brand', 'primaryCategory', 'categories', 'variants', 'gallery'])
            ->first();

        return $product === null ? null : ProductData::fromModel($product);
    }

    public function currentSlugFor(string $previousSlug): ?string
    {
        $productId = ProductSlug::query()->where('slug', $previousSlug)->value('product_id');
        if ($productId === null) {
            return null;
        }

        $slug = Product::query()->published()->whereKey((int) $productId)->value('slug');

        return is_string($slug) ? $slug : null;
    }

    public function list(ProductCriteria $criteria): ProductPage
    {
        return (new ProductsInCategory($criteria, $this->catalog->categoryTree()))->get();
    }

    public function bestSellers(int $limit = 10, ?int $categoryId = null): array
    {
        $categoryIds = [];
        if ($categoryId !== null) {
            $categoryIds = $this->catalog->categoryTree()->descendantIds($categoryId) ?: [$categoryId];
        }

        return (new BestSellers($limit, $categoryIds))->get();
    }

    public function frequentlyBoughtWith(int $productId, int $limit = 5): array
    {
        return (new FrequentlyBoughtWith($productId, $this->catalog->categoryTree(), $limit))->get();
    }

    public function reviews(int $productId, int $page = 1, int $perPage = 10): ReviewPage
    {
        $page = max(1, $page);
        $perPage = max(1, min(50, $perPage));
        $query = ProductReview::query()->approved()->where('product_id', $productId);
        $total = (clone $query)->count();

        $items = $query
            ->orderByDesc('approved_at')
            ->orderByDesc('id')
            ->forPage($page, $perPage)
            ->get()
            ->map(ReviewData::fromModel(...))
            ->values()
            ->all();

        return new ReviewPage($items, $total, $page, $perPage);
    }
}
