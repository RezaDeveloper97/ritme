<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Observers;

use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Domain\Shop\Catalog\Support\ProductContent;
use App\Domain\Shop\Catalog\Support\ProductSlugger;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use Illuminate\Database\Eloquent\Model;

/**
 * Prepares a product on save (slug + history, sanitised content, compare-at price only when higher than the price,
 * stock status from the quantity unless manually PreOrder/BackOrder; with variants the quantity is owned by
 * RecalculateProductStock and hand edits are ignored) and keeps
 * the primary category in the category pivot. Invalidates `shop`, `sitemap` and (cacheaside.always_bump) `pages`.
 */
final class ProductObserver extends CacheBumpingObserver
{
    public function __construct(
        NamespaceBumper $bumper,
        private readonly ProductSlugger $slugger,
        private readonly ProductContent $content,
    ) {
        parent::__construct($bumper);
    }

    protected function namespaces(Model $model): array
    {
        return ['shop', 'sitemap'];
    }

    public function saving(Product $product): void
    {
        $this->slugger->assign($product);
        $this->content->prepare($product);

        $sku = trim((string) $product->sku);
        $product->sku = $sku === '' ? null : $sku;

        if ($product->compare_at_price !== null && ! $product->compare_at_price->greaterThan($product->price)) {
            $product->compare_at_price = null;
        }

        // With variants the quantity is their aggregate (RecalculateProductStock): a hand-edited value is ignored.
        if ($product->exists && $product->isDirty('stock_qty') && ProductVariant::query()->where('product_id', $product->id)->exists()) {
            $product->stock_qty = (int) $product->getOriginal('stock_qty');
        }
        $product->stock_qty = max(0, $product->stock_qty);
        $product->stock_status = StockStatus::forQuantity($product->stock_qty, $product->stock_status);
    }

    public function saved(Model $model): void
    {
        if ($model instanceof Product && $model->primary_category_id !== null && ($model->wasRecentlyCreated || $model->wasChanged('primary_category_id'))) {
            $model->categories()->syncWithoutDetaching([$model->primary_category_id]);
        }

        parent::saved($model);
    }

    public function updated(Product $product): void
    {
        $this->slugger->recordChange($product);
    }
}
