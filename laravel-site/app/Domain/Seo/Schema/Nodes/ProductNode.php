<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Data\ProductData;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * Product (`#product`) with its Offer and — only from real reviews — AggregateRating.
 */
final class ProductNode
{
    /**
     * @return array<string, mixed>
     */
    public static function make(ProductData $product): array
    {
        return Node::clean([
            '@type' => 'Product',
            '@id' => SchemaIds::product($product->url),
            'name' => $product->name,
            'description' => $product->description,
            'url' => $product->url,
            'image' => Node::strings($product->imageUrls),
            'sku' => $product->sku,
            'gtin' => $product->gtin,
            'category' => $product->category,
            'brand' => $product->brand !== null ? ['@type' => 'Brand', 'name' => $product->brand] : null,
            'offers' => [...$product->offer->toNode(), 'url' => $product->offer->url ?? $product->url],
            'aggregateRating' => $product->rating?->toNode(),
            'mainEntityOfPage' => Node::ref(SchemaIds::webPage($product->url)),
        ]);
    }
}
