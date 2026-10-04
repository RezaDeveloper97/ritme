<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

use App\Domain\Seo\Schema\Data\AggregateRatingData;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Support\Money\Money;

/**
 * A product in lists (shop cards, best sellers, «معمولاً با این می‌خرند», search). The card shows the brand (single
 * seller). `ratingCount` counts real approved reviews only (0 → show no rating). `isDemo` marks design samples.
 */
final readonly class ProductCardData
{
    public function __construct(
        public int $id,
        public string $title,
        public string $slug,
        public ?BrandData $brand,
        public Money $price,
        public ?Money $compareAtPrice,
        public StockStatus $stockStatus,
        public ?int $coverMediaId = null,
        public ?string $illustration = null,
        public ?string $badge = null,
        public float $ratingAvg = 0.0,
        public int $ratingCount = 0,
        public bool $hasVariants = false,
        public bool $isDemo = false,
        public ?string $shortDescription = null,
    ) {}

    /**
     * Expects `brand` loaded; `variants_count` (withCount) is used when present.
     */
    public static function fromModel(Product $product): self
    {
        $variants = $product->getAttribute('variants_count');

        return new self(
            id: $product->id,
            title: $product->title,
            slug: $product->slug,
            brand: $product->brand === null ? null : BrandData::fromModel($product->brand),
            price: $product->price,
            compareAtPrice: $product->compare_at_price,
            stockStatus: $product->stock_status,
            coverMediaId: $product->cover_media_id,
            illustration: $product->illustration,
            badge: $product->badge,
            ratingAvg: $product->rating_avg,
            ratingCount: $product->rating_count,
            hasVariants: is_numeric($variants) && (int) $variants > 0,
            isDemo: $product->is_demo,
            shortDescription: $product->short_description,
        );
    }

    public function isPurchasable(): bool
    {
        return $this->stockStatus->isPurchasable();
    }

    public function discountPercent(): ?int
    {
        return $this->compareAtPrice === null ? null : $this->price->percentOff($this->compareAtPrice);
    }

    /** Aggregate of real reviews, or null when there are none (never show or emit an invented rating). */
    public function rating(): ?AggregateRatingData
    {
        return $this->ratingCount < 1 ? null : new AggregateRatingData($this->ratingAvg, $this->ratingCount, $this->ratingCount);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'title' => $this->title, 'slug' => $this->slug, 'brand' => $this->brand?->toArray(),
            'price' => $this->price->rial, 'compareAtPrice' => $this->compareAtPrice?->rial,
            'stockStatus' => $this->stockStatus->value, 'coverMediaId' => $this->coverMediaId,
            'illustration' => $this->illustration, 'badge' => $this->badge, 'ratingAvg' => $this->ratingAvg,
            'ratingCount' => $this->ratingCount, 'hasVariants' => $this->hasVariants, 'isDemo' => $this->isDemo,
            'shortDescription' => $this->shortDescription,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        /** @var array<string, mixed>|null $brand */
        $brand = $data['brand'] ?? null;

        return new self(
            id: (int) $data['id'],
            title: (string) $data['title'],
            slug: (string) $data['slug'],
            brand: $brand === null ? null : BrandData::fromArray($brand),
            price: Money::fromRial((int) $data['price']),
            compareAtPrice: isset($data['compareAtPrice']) ? Money::fromRial((int) $data['compareAtPrice']) : null,
            stockStatus: StockStatus::from((string) $data['stockStatus']),
            coverMediaId: isset($data['coverMediaId']) ? (int) $data['coverMediaId'] : null,
            illustration: isset($data['illustration']) ? (string) $data['illustration'] : null,
            badge: isset($data['badge']) ? (string) $data['badge'] : null,
            ratingAvg: (float) ($data['ratingAvg'] ?? 0),
            ratingCount: (int) ($data['ratingCount'] ?? 0),
            hasVariants: (bool) ($data['hasVariants'] ?? false),
            isDemo: (bool) ($data['isDemo'] ?? false),
            shortDescription: isset($data['shortDescription']) ? (string) $data['shortDescription'] : null,
        );
    }
}
