<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Seo\Schema\Data\AggregateRatingData;
use App\Domain\Seo\Schema\Data\OfferData;
use App\Domain\Seo\Schema\Data\ProductData as ProductSchemaData;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Html\HtmlText;
use App\Support\Money\Money;
use Illuminate\Support\Str;

/**
 * Everything the product page (L6-03) needs. `description` is sanitised HTML; `specs` a list of {label, value};
 * `sizeChart` {columns, rows}; `variants` the ACTIVE variants in display order. Prices are Money (integer rials).
 * `ratingCount` counts real approved reviews only.
 */
final readonly class ProductData
{
    /**
     * @param  list<CategoryData>  $categories
     * @param  list<array{label: string, value: string}>  $specs
     * @param  array{columns: list<string>, rows: list<list<string>>}|null  $sizeChart
     * @param  list<int>  $galleryMediaIds
     * @param  list<LifeStage>  $lifeStages
     * @param  list<VariantData>  $variants
     */
    public function __construct(
        public int $id,
        public string $title,
        public string $slug,
        public ?string $sku,
        public ?BrandData $brand,
        public ?CategoryData $primaryCategory,
        public array $categories,
        public ?string $shortDescription,
        public ?string $description,
        public array $specs,
        public ?array $sizeChart,
        public ?string $badge,
        public Money $price,
        public ?Money $compareAtPrice,
        public int $stockQty,
        public StockStatus $stockStatus,
        public ?int $weightGrams,
        public ?int $coverMediaId,
        public array $galleryMediaIds,
        public ?string $illustration,
        public array $lifeStages,
        public array $variants,
        public float $ratingAvg,
        public int $ratingCount,
        public int $salesCount,
        public bool $isFeatured,
        public bool $isDemo,
        public ?string $updatedAt,
    ) {}

    /**
     * Expects `brand`, `primaryCategory`, `categories`, `variants` and `gallery` loaded.
     */
    public static function fromModel(Product $product): self
    {
        $variants = array_values($product->variants
            ->filter(static fn (ProductVariant $v): bool => $v->is_active)
            ->map(static fn (ProductVariant $v): VariantData => VariantData::fromModel($v, $product))
            ->all());

        return new self(
            id: $product->id,
            title: $product->title,
            slug: $product->slug,
            sku: $product->sku,
            brand: $product->brand === null ? null : BrandData::fromModel($product->brand),
            primaryCategory: $product->primaryCategory === null ? null : CategoryData::fromModel($product->primaryCategory),
            categories: array_values($product->categories->sortBy('sort_order')->map(CategoryData::fromModel(...))->all()),
            shortDescription: $product->short_description,
            description: $product->description,
            specs: $product->specs ?? [],
            sizeChart: $product->size_chart,
            badge: $product->badge,
            price: $product->price,
            compareAtPrice: $product->compare_at_price,
            stockQty: $product->stock_qty,
            stockStatus: $product->stock_status,
            weightGrams: $product->weight_grams,
            coverMediaId: $product->cover_media_id,
            galleryMediaIds: array_values($product->gallery->map(static fn ($media): int => (int) $media->getKey())->all()),
            illustration: $product->illustration,
            lifeStages: $product->lifeStages(),
            variants: $variants,
            ratingAvg: $product->rating_avg,
            ratingCount: $product->rating_count,
            salesCount: $product->sales_count,
            isFeatured: $product->is_featured,
            isDemo: $product->is_demo,
            updatedAt: $product->updated_at?->toIso8601String(),
        );
    }

    public function hasVariants(): bool
    {
        return $this->variants !== [];
    }

    public function isPurchasable(): bool
    {
        return $this->stockStatus->isPurchasable();
    }

    public function discountPercent(): ?int
    {
        return $this->compareAtPrice === null ? null : $this->price->percentOff($this->compareAtPrice);
    }

    /**
     * Distinct sizes in variant order.
     *
     * @return list<string>
     */
    public function sizes(): array
    {
        return array_values(array_unique(array_filter(array_map(static fn (VariantData $v): ?string => $v->size, $this->variants), static fn (?string $s): bool => $s !== null)));
    }

    /**
     * Distinct colours in variant order with their swatch.
     *
     * @return list<array{name: string, hex: string|null}>
     */
    public function colors(): array
    {
        $colors = [];
        foreach ($this->variants as $variant) {
            if ($variant->color !== null && ! isset($colors[$variant->color])) {
                $colors[$variant->color] = ['name' => $variant->color, 'hex' => $variant->colorHex];
            }
        }

        return array_values($colors);
    }

    public function variantFor(?string $size, ?string $color = null): ?VariantData
    {
        foreach ($this->variants as $variant) {
            if ($variant->size === $size && $variant->color === $color) {
                return $variant;
            }
        }

        return null;
    }

    public function findVariant(int $id): ?VariantData
    {
        foreach ($this->variants as $variant) {
            if ($variant->id === $id) {
                return $variant;
            }
        }

        return null;
    }

    /**
     * Cover first, then the gallery (duplicates removed).
     *
     * @return list<int>
     */
    public function mediaIds(): array
    {
        return array_values(array_unique(array_filter([$this->coverMediaId, ...$this->galleryMediaIds], static fn (?int $id): bool => $id !== null)));
    }

    /** Aggregate of real reviews, or null when there are none. */
    public function rating(): ?AggregateRatingData
    {
        return $this->ratingCount < 1 ? null : new AggregateRatingData($this->ratingAvg, $this->ratingCount, $this->ratingCount);
    }

    public function toCard(): ProductCardData
    {
        return new ProductCardData(
            id: $this->id, title: $this->title, slug: $this->slug, brand: $this->brand, price: $this->price,
            compareAtPrice: $this->compareAtPrice, stockStatus: $this->stockStatus, coverMediaId: $this->coverMediaId,
            illustration: $this->illustration, badge: $this->badge, ratingAvg: $this->ratingAvg,
            ratingCount: $this->ratingCount, hasVariants: $this->hasVariants(), isDemo: $this->isDemo,
            shortDescription: $this->shortDescription,
        );
    }

    /**
     * schema.org Product for the JSON-LD graph: the price is already in rials (IRR), availability from the stock
     * status, a rating only from real reviews.
     *
     * @param  list<string>  $imageUrls  absolute image URLs
     */
    public function toSchema(string $url, array $imageUrls): ProductSchemaData
    {
        $description = $this->shortDescription ?? ($this->description === null ? null : Str::limit(HtmlText::plain($this->description), 300));

        return new ProductSchemaData(
            url: $url,
            name: $this->title,
            imageUrls: $imageUrls,
            offer: new OfferData(price: $this->price->rial, priceCurrency: Money::CURRENCY, availability: $this->stockStatus->availability(), url: $url),
            description: $description,
            sku: $this->sku,
            brand: $this->brand?->name,
            category: $this->primaryCategory?->name,
            rating: $this->rating(),
        );
    }

    /**
     * schema.org `offers` of the Product node (L6-03). Without variants: one Offer (product price + stock status).
     * With variants: an AggregateOffer (lowPrice / highPrice / offerCount, availability of the product) holding one
     * Offer per active variant — its sku, label, effective price and availability from the variant's own stock
     * (pre-order / back-order products keep that status). Prices in rials (IRR).
     *
     * @param  array<string, mixed>|null  $seller  e.g. a reference to the site Organization (single seller)
     * @return array<string, mixed>
     */
    public function offersNode(string $url, ?string $priceValidUntil = null, ?array $seller = null): array
    {
        $offer = static fn (Money $price, StockStatus|bool $stock, array $extra = []): array => array_filter([
            '@type' => 'Offer',
            ...$extra,
            'price' => $price->rial,
            'priceCurrency' => Money::CURRENCY,
            'availability' => ($stock instanceof StockStatus ? $stock : ($stock ? StockStatus::InStock : StockStatus::OutOfStock))->availability()->uri(),
            'itemCondition' => 'https://schema.org/NewCondition',
            'url' => $url,
            'priceValidUntil' => $priceValidUntil,
            'seller' => $seller,
        ], static fn (mixed $value): bool => $value !== null && $value !== '');

        if (! $this->hasVariants()) {
            return $offer($this->price, $this->stockStatus);
        }

        $manual = $this->stockStatus === StockStatus::PreOrder || $this->stockStatus === StockStatus::BackOrder;
        $offers = array_map(fn (VariantData $v): array => $offer(
            $v->price,
            $manual ? $this->stockStatus : $v->isInStock(),
            ['sku' => $v->sku, 'name' => trim($this->title.' — '.$v->label(), ' —')],
        ), $this->variants);
        $prices = array_map(static fn (VariantData $v): Money => $v->price, $this->variants);

        return array_filter([
            '@type' => 'AggregateOffer',
            'lowPrice' => Money::min(...$prices)->rial,
            'highPrice' => Money::max(...$prices)->rial,
            'offerCount' => count($offers),
            'priceCurrency' => Money::CURRENCY,
            'availability' => $this->stockStatus->availability()->uri(),
            'url' => $url,
            'seller' => $seller,
            'offers' => $offers,
        ], static fn (mixed $value): bool => $value !== null);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'title' => $this->title, 'slug' => $this->slug, 'sku' => $this->sku,
            'brand' => $this->brand?->toArray(), 'primaryCategory' => $this->primaryCategory?->toArray(),
            'categories' => array_map(static fn (CategoryData $c): array => $c->toArray(), $this->categories),
            'shortDescription' => $this->shortDescription, 'description' => $this->description, 'specs' => $this->specs,
            'sizeChart' => $this->sizeChart, 'badge' => $this->badge, 'price' => $this->price->rial,
            'compareAtPrice' => $this->compareAtPrice?->rial, 'stockQty' => $this->stockQty,
            'stockStatus' => $this->stockStatus->value, 'weightGrams' => $this->weightGrams,
            'coverMediaId' => $this->coverMediaId, 'galleryMediaIds' => $this->galleryMediaIds,
            'illustration' => $this->illustration,
            'lifeStages' => array_map(static fn (LifeStage $s): string => $s->value, $this->lifeStages),
            'variants' => array_map(static fn (VariantData $v): array => $v->toArray(), $this->variants),
            'ratingAvg' => $this->ratingAvg, 'ratingCount' => $this->ratingCount, 'salesCount' => $this->salesCount,
            'isFeatured' => $this->isFeatured, 'isDemo' => $this->isDemo, 'updatedAt' => $this->updatedAt,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $string = static fn (string $key): ?string => isset($data[$key]) ? (string) $data[$key] : null;
        $int = static fn (string $key): ?int => isset($data[$key]) ? (int) $data[$key] : null;
        /** @var array<string, mixed>|null $brand */
        $brand = $data['brand'] ?? null;
        /** @var array<string, mixed>|null $primary */
        $primary = $data['primaryCategory'] ?? null;
        /** @var list<array<string, mixed>> $categories */
        $categories = (array) ($data['categories'] ?? []);
        /** @var list<array<string, mixed>> $variants */
        $variants = (array) ($data['variants'] ?? []);
        /** @var list<array{label: string, value: string}> $specs */
        $specs = (array) ($data['specs'] ?? []);
        /** @var array{columns: list<string>, rows: list<list<string>>}|null $chart */
        $chart = $data['sizeChart'] ?? null;

        return new self(
            id: (int) $data['id'],
            title: (string) $data['title'],
            slug: (string) $data['slug'],
            sku: $string('sku'),
            brand: $brand === null ? null : BrandData::fromArray($brand),
            primaryCategory: $primary === null ? null : CategoryData::fromArray($primary),
            categories: array_map(CategoryData::fromArray(...), $categories),
            shortDescription: $string('shortDescription'),
            description: $string('description'),
            specs: $specs,
            sizeChart: $chart,
            badge: $string('badge'),
            price: Money::fromRial((int) $data['price']),
            compareAtPrice: isset($data['compareAtPrice']) ? Money::fromRial((int) $data['compareAtPrice']) : null,
            stockQty: (int) $data['stockQty'],
            stockStatus: StockStatus::from((string) $data['stockStatus']),
            weightGrams: $int('weightGrams'),
            coverMediaId: $int('coverMediaId'),
            galleryMediaIds: array_values(array_map(intval(...), (array) ($data['galleryMediaIds'] ?? []))),
            illustration: $string('illustration'),
            lifeStages: array_values(array_filter(array_map(static fn (mixed $s): ?LifeStage => LifeStage::tryFrom((string) $s), (array) ($data['lifeStages'] ?? [])))),
            variants: array_map(VariantData::fromArray(...), $variants),
            ratingAvg: (float) ($data['ratingAvg'] ?? 0),
            ratingCount: (int) ($data['ratingCount'] ?? 0),
            salesCount: (int) ($data['salesCount'] ?? 0),
            isFeatured: (bool) ($data['isFeatured'] ?? false),
            isDemo: (bool) ($data['isDemo'] ?? false),
            updatedAt: $string('updatedAt'),
        );
    }
}
