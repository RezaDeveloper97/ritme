<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Search;

use App\Domain\Search\Contracts\SearchProvider;
use App\Domain\Search\Data\SearchHit;
use App\Domain\Search\Support\SearchTerms;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use Illuminate\Support\Str;

/**
 * Published shop products in site search (`/search`, L4-04), through the cached ProductsInCategory query: title,
 * short description and brand name. Registered with SearchRegistry::TAG by ShopServiceProvider.
 */
final class ProductSearchProvider implements SearchProvider
{
    public function __construct(private readonly ProductRepository $products, private readonly ShopUrls $urls) {}

    public function key(): string
    {
        return 'products';
    }

    public function label(): string
    {
        return 'فروشگاه';
    }

    public function search(SearchTerms $terms, int $limit): array
    {
        if ($terms->isEmpty()) {
            return [];
        }

        $page = $this->products->list(new ProductCriteria(text: $terms->normalized, perPage: max(1, min(ProductCriteria::MAX_PER_PAGE, $limit))));

        return array_map(function (ProductCardData $product) use ($terms): SearchHit {
            $summary = $product->shortDescription === null || trim($product->shortDescription) === '' ? null : Str::limit(trim($product->shortDescription), 150);

            return new SearchHit(
                type: $this->key(),
                typeLabel: $this->label(),
                title: $product->title,
                url: $this->urls->product($product->slug),
                snippet: implode(' · ', array_filter([$product->brand?->name, $product->price->formatShort(), $summary])),
                score: $terms->touches($product->title) ? 2 : 1,
            );
        }, $page->items);
    }
}
