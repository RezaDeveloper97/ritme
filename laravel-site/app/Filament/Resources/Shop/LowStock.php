<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop;

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\ShopSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use Illuminate\Database\Eloquent\Builder;

/**
 * «Low stock» of the shop admin (L6-06): the threshold comes from the shop settings (`low_stock_threshold`, default 3).
 * A published or draft product is low when it has no variants and at most that many units, or when one of its ACTIVE
 * variants has at most that many. Pre-order / back-order products are sold beyond stock and never listed.
 */
final class LowStock
{
    public static function threshold(): int
    {
        $shop = app(SettingsRepository::class)->group(SettingGroup::Shop);

        return $shop instanceof ShopSettings ? $shop->lowStockThreshold : ShopSettings::DEFAULT_LOW_STOCK;
    }

    /**
     * @return Builder<Product>
     */
    public static function products(?int $threshold = null): Builder
    {
        $threshold ??= self::threshold();
        $variants = (new ProductVariant)->getTable();

        return Product::query()
            ->whereNotIn('stock_status', [StockStatus::PreOrder->value, StockStatus::BackOrder->value])
            ->where(static function (Builder $query) use ($threshold, $variants): void {
                $query
                    ->where(static function (Builder $q) use ($threshold, $variants): void {
                        $q->where('stock_qty', '<=', $threshold)
                            ->whereNotExists(static fn ($sub) => $sub->from($variants)->whereColumn($variants.'.product_id', 'shop_products.id'));
                    })
                    ->orWhereExists(static fn ($sub) => $sub->from($variants)
                        ->whereColumn($variants.'.product_id', 'shop_products.id')
                        ->where($variants.'.is_active', true)
                        ->where($variants.'.stock_qty', '<=', $threshold));
            });
    }

    /**
     * «۰-۳ ماه / صورتی: ۱» … for the low variants of a product.
     */
    public static function variantSummary(Product $product, ?int $threshold = null): ?string
    {
        $threshold ??= self::threshold();
        $parts = [];
        foreach ($product->variants as $variant) {
            if ($variant->is_active && $variant->stock_qty <= $threshold) {
                $label = trim(implode(' / ', array_filter([$variant->size, $variant->color])));
                $parts[] = ($label !== '' ? $label : '#'.$variant->id).': '.fa_digits($variant->stock_qty);
            }
        }

        return $parts === [] ? null : implode('، ', $parts);
    }
}
