<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

/**
 * A shop product. `rating` only from real customer reviews (see AggregateRatingData).
 */
final readonly class ProductData
{
    /**
     * @param  list<string>  $imageUrls  absolute URLs, ideally 1:1, 4:3 and 16:9 crops
     */
    public function __construct(
        public string $url,
        public string $name,
        public array $imageUrls,
        public OfferData $offer,
        public ?string $description = null,
        public ?string $sku = null,
        public ?string $brand = null,
        public ?string $gtin = null,
        public ?string $category = null,
        public ?AggregateRatingData $rating = null,
    ) {}
}
