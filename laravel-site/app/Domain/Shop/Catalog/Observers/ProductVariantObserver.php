<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Observers;

use App\Domain\Shop\Catalog\Actions\RecalculateProductStock;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use Illuminate\Database\Eloquent\Model;

/**
 * Normalises a variant on save (trimmed sku/size/colour, `#RRGGBB` swatch or null, compare-at only when higher than
 * the variant's own price) and, after any change, recalculates the product's aggregate stock. Bumps `shop` (+ `pages`).
 */
final class ProductVariantObserver extends CacheBumpingObserver
{
    public function __construct(NamespaceBumper $bumper, private readonly RecalculateProductStock $recalculate)
    {
        parent::__construct($bumper);
    }

    protected function namespaces(Model $model): array
    {
        return ['shop'];
    }

    public function saving(ProductVariant $variant): void
    {
        foreach (['sku', 'size', 'color'] as $field) {
            $value = trim((string) $variant->getAttribute($field));
            $variant->setAttribute($field, $value === '' ? null : $value);
        }

        $hex = strtoupper(trim((string) $variant->color_hex));
        $variant->color_hex = preg_match('/^#[0-9A-F]{6}$/', $hex) === 1 ? $hex : null;

        $variant->stock_qty = max(0, $variant->stock_qty);

        if ($variant->compare_at_price !== null && ($variant->price === null || ! $variant->compare_at_price->greaterThan($variant->price))) {
            $variant->compare_at_price = null;
        }
    }

    public function saved(Model $model): void
    {
        $this->refresh($model);
        parent::saved($model);
    }

    public function deleted(Model $model): void
    {
        $this->refresh($model);
        parent::deleted($model);
    }

    private function refresh(Model $model): void
    {
        if ($model instanceof ProductVariant) {
            $this->recalculate->handle($model->product_id);
        }
    }
}
