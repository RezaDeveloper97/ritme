<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Models;

use App\Support\Money\Money;
use App\Support\Money\MoneyCast;
use Database\Factories\Shop\ProductVariantFactory;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * A simple variant (size and/or colour) with its own sku, optional own price (null = the product price) and stock.
 * Any change recalculates the product's aggregate stock (ProductVariantObserver → RecalculateProductStock).
 *
 * @property int $id
 * @property int $product_id
 * @property string|null $sku
 * @property string|null $size
 * @property string|null $color
 * @property string|null $color_hex
 * @property Money|null $price
 * @property Money|null $compare_at_price
 * @property int $stock_qty
 * @property int $sort_order
 * @property bool $is_active
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Product $product
 */
final class ProductVariant extends Model
{
    /** @use HasFactory<ProductVariantFactory> */
    use HasFactory;

    protected $table = 'shop_product_variants';

    protected $fillable = ['product_id', 'sku', 'size', 'color', 'color_hex', 'price', 'compare_at_price', 'stock_qty', 'sort_order', 'is_active'];

    protected $attributes = ['stock_qty' => 0, 'sort_order' => 0, 'is_active' => true];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'product_id' => 'integer',
            'price' => MoneyCast::class,
            'compare_at_price' => MoneyCast::class,
            'stock_qty' => 'integer',
            'sort_order' => 'integer',
            'is_active' => 'boolean',
        ];
    }

    protected static function newFactory(): ProductVariantFactory
    {
        return ProductVariantFactory::new();
    }

    /**
     * @return BelongsTo<Product, $this>
     */
    public function product(): BelongsTo
    {
        return $this->belongsTo(Product::class);
    }
}
