<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Models;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Media\Models\Media;
use App\Domain\Seo\Concerns\HasSeo;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Support\Money\Money;
use App\Support\Money\MoneyCast;
use Database\Factories\Shop\ProductFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * A shop product. Read through ProductRepository (DTOs). Prices are Money (integer rials). On save (ProductObserver):
 * slug + history, sanitised description, normalised specs / size chart / life stages, compare-at price dropped unless
 * higher than the price, stock status from the quantity, primary category attached. With variants, `stock_qty` /
 * `stock_status` are maintained from the active variants (RecalculateProductStock). `rating_*` (approved, non-demo
 * reviews) and `sales_count` (orders, L6-05) are maintained elsewhere — never set them by hand.
 *
 * @property int $id
 * @property string $title
 * @property string $slug
 * @property string|null $sku
 * @property int|null $brand_id
 * @property int|null $primary_category_id
 * @property string|null $short_description
 * @property string|null $description
 * @property list<array{label: string, value: string}>|null $specs
 * @property array{columns: list<string>, rows: list<list<string>>}|null $size_chart
 * @property string|null $badge
 * @property Money $price
 * @property Money|null $compare_at_price
 * @property int $stock_qty
 * @property StockStatus $stock_status
 * @property int|null $weight_grams
 * @property int|null $cover_media_id
 * @property string|null $illustration
 * @property list<string>|null $life_stages
 * @property float $rating_avg
 * @property int $rating_count
 * @property int $sales_count
 * @property bool $is_published
 * @property bool $is_featured
 * @property bool $is_demo
 * @property int $sort_order
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Brand|null $brand
 * @property-read Category|null $primaryCategory
 * @property-read Collection<int, Category> $categories
 * @property-read Collection<int, ProductVariant> $variants
 * @property-read Collection<int, Media> $gallery
 * @property-read Collection<int, ProductReview> $reviews
 * @property-read Collection<int, Product> $crossSells
 */
final class Product extends Model
{
    /** @use HasFactory<ProductFactory> */
    use HasFactory;

    use HasSeo;

    protected $table = 'shop_products';

    protected $fillable = [
        'title', 'slug', 'sku', 'brand_id', 'primary_category_id', 'short_description', 'description', 'specs',
        'size_chart', 'badge', 'price', 'compare_at_price', 'stock_qty', 'stock_status', 'weight_grams',
        'cover_media_id', 'illustration', 'life_stages', 'is_published', 'is_featured', 'is_demo', 'sort_order',
    ];

    protected $attributes = [
        'stock_qty' => 0,
        'stock_status' => 'out_of_stock',
        'rating_avg' => 0,
        'rating_count' => 0,
        'sales_count' => 0,
        'is_published' => false,
        'is_featured' => false,
        'is_demo' => false,
        'sort_order' => 0,
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'brand_id' => 'integer',
            'primary_category_id' => 'integer',
            'specs' => 'array',
            'size_chart' => 'array',
            'price' => MoneyCast::class,
            'compare_at_price' => MoneyCast::class,
            'stock_qty' => 'integer',
            'stock_status' => StockStatus::class,
            'weight_grams' => 'integer',
            'cover_media_id' => 'integer',
            'life_stages' => 'array',
            'rating_avg' => 'float',
            'rating_count' => 'integer',
            'sales_count' => 'integer',
            'is_published' => 'boolean',
            'is_featured' => 'boolean',
            'is_demo' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    protected static function newFactory(): ProductFactory
    {
        return ProductFactory::new();
    }

    /**
     * @param  Builder<self>  $query
     */
    public function scopePublished(Builder $query): void
    {
        $query->where($this->qualifyColumn('is_published'), true);
    }

    /**
     * @return list<LifeStage>
     */
    public function lifeStages(): array
    {
        return array_values(array_filter(array_map(
            static fn (string $value): ?LifeStage => LifeStage::tryFrom($value),
            $this->life_stages ?? [],
        )));
    }

    /**
     * @return BelongsTo<Brand, $this>
     */
    public function brand(): BelongsTo
    {
        return $this->belongsTo(Brand::class);
    }

    /**
     * @return BelongsTo<Category, $this>
     */
    public function primaryCategory(): BelongsTo
    {
        return $this->belongsTo(Category::class, 'primary_category_id');
    }

    /**
     * @return BelongsToMany<Category, $this>
     */
    public function categories(): BelongsToMany
    {
        return $this->belongsToMany(Category::class, 'shop_category_product', 'product_id', 'category_id');
    }

    /**
     * Variants in display order (active and inactive; DTOs keep the active ones).
     *
     * @return HasMany<ProductVariant, $this>
     */
    public function variants(): HasMany
    {
        return $this->hasMany(ProductVariant::class)->orderBy('sort_order')->orderBy('id');
    }

    /**
     * Ordered gallery images (the cover is `cover_media_id`).
     *
     * @return BelongsToMany<Media, $this>
     */
    public function gallery(): BelongsToMany
    {
        return $this->belongsToMany(Media::class, 'shop_product_media', 'product_id', 'media_id')
            ->withPivot('sort_order')
            ->orderByPivot('sort_order');
    }

    /**
     * Hand-picked «معمولاً با این می‌خرند» products, in order.
     *
     * @return BelongsToMany<Product, $this>
     */
    public function crossSells(): BelongsToMany
    {
        return $this->belongsToMany(self::class, 'shop_product_cross_sells', 'product_id', 'related_id')
            ->withPivot('sort_order')
            ->orderByPivot('sort_order');
    }

    /**
     * @return HasMany<ProductReview, $this>
     */
    public function reviews(): HasMany
    {
        return $this->hasMany(ProductReview::class);
    }

    /**
     * @return HasMany<ProductSlug, $this>
     */
    public function previousSlugs(): HasMany
    {
        return $this->hasMany(ProductSlug::class);
    }
}
