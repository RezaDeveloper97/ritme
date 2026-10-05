<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Database\Eloquent\Model;

/**
 * Creates or updates a product from the admin form (L6-06), in ONE transaction: attributes (never the computed
 * `rating_*` / `sales_count`), the variants (updated by id, new ones created, missing ones deleted — the observer
 * recalculates the aggregate stock), categories + primary category, ordered gallery and cross-sells through their
 * Sync* actions. Prices arrive as Money (the form converts tomans to rials). `stock_status` only keeps a manual
 * PreOrder / BackOrder; anything else follows the quantity.
 *
 * Activity log (`shop`): `shop.product.created`, `shop.product.status` (published ⇄ draft) and `shop.product.stock`
 * (old / new quantity of the product and of every changed variant).
 */
final class SaveProduct
{
    /** Columns the form may never write: maintained from reviews and orders. */
    private const GUARDED = ['rating_avg', 'rating_count', 'sales_count'];

    private const VARIANT_FIELDS = ['sku', 'size', 'color', 'color_hex', 'price', 'compare_at_price', 'stock_qty', 'is_active'];

    public function __construct(
        private readonly ConnectionInterface $db,
        private readonly SyncProductCategories $categories,
        private readonly SyncProductGallery $gallery,
        private readonly SyncCrossSells $crossSells,
    ) {}

    /**
     * @param  array<string, mixed>  $attributes
     * @param  list<array<string, mixed>>|null  $variants  null = leave as is; each item may carry its `id`
     * @param  list<int>|null  $categoryIds  null = leave as is
     * @param  list<int>|null  $galleryMediaIds  null = leave as is
     * @param  list<int>|null  $crossSellIds  null = leave as is
     */
    public function handle(
        Product $product,
        array $attributes,
        ?array $variants = null,
        ?array $categoryIds = null,
        ?int $primaryCategoryId = null,
        ?array $galleryMediaIds = null,
        ?array $crossSellIds = null,
        ?Model $causer = null,
    ): Product {
        $creating = ! $product->exists;
        $wasPublished = $creating ? null : $product->is_published;
        $stockBefore = $creating ? [] : $this->stockSnapshot($product);

        /** @var Product $product */
        $product = $this->db->transaction(function () use ($product, $attributes, $variants, $categoryIds, $primaryCategoryId, $galleryMediaIds, $crossSellIds): Product {
            $attributes = $this->attributes($attributes);

            // Existing product: variants first, so a product that loses its last variant keeps the quantity typed now.
            if ($product->exists && $variants !== null) {
                $this->syncVariants($product, $variants);
            }

            $product->fill($attributes);
            $product->save();

            if ($variants !== null && $product->wasRecentlyCreated) {
                $this->syncVariants($product, $variants);
            }
            if ($categoryIds !== null) {
                $this->categories->handle($product, $categoryIds, $primaryCategoryId);
            }
            if ($galleryMediaIds !== null) {
                $this->gallery->handle($product, $galleryMediaIds);
            }
            if ($crossSellIds !== null) {
                $this->crossSells->handle($product, $crossSellIds);
            }

            return $product->refresh();
        });

        $this->log($product, $creating, $wasPublished, $stockBefore, $causer);

        return $product;
    }

    /**
     * @param  array<string, mixed>  $attributes
     * @return array<string, mixed>
     */
    private function attributes(array $attributes): array
    {
        $attributes = array_diff_key($attributes, array_flip(self::GUARDED));

        if (array_key_exists('stock_status', $attributes)) {
            $status = $attributes['stock_status'] instanceof StockStatus
                ? $attributes['stock_status']
                : StockStatus::tryFrom((string) $attributes['stock_status']);
            // InStock / OutOfStock follow the quantity (ProductObserver::forQuantity); only PreOrder / BackOrder stick.
            $attributes['stock_status'] = $status === StockStatus::PreOrder || $status === StockStatus::BackOrder
                ? $status
                : StockStatus::OutOfStock;
        }

        return $attributes;
    }

    /**
     * @param  list<array<string, mixed>>  $variants
     */
    private function syncVariants(Product $product, array $variants): void
    {
        $existing = ProductVariant::query()->where('product_id', $product->id)->get()->keyBy('id');
        $kept = [];

        foreach ($variants as $position => $data) {
            $id = is_numeric($data['id'] ?? null) ? (int) $data['id'] : null;
            $variant = $id !== null ? $existing->get($id) : null;
            $variant ??= new ProductVariant(['product_id' => $product->id]);

            $variant->fill(array_intersect_key($data, array_flip(self::VARIANT_FIELDS)));
            $variant->sort_order = $position;
            $variant->save();
            $kept[] = $variant->id;
        }

        foreach ($existing as $variant) {
            if (! in_array($variant->id, $kept, true)) {
                $variant->delete(); // order lines keep their snapshot (variant_id → null)
            }
        }
    }

    /**
     * @return array<string, int>
     */
    private function stockSnapshot(Product $product): array
    {
        $snapshot = ['product' => (int) Product::query()->whereKey($product->id)->value('stock_qty')];
        foreach (ProductVariant::query()->where('product_id', $product->id)->get(['id', 'stock_qty']) as $variant) {
            $snapshot['variant:'.$variant->id] = $variant->stock_qty;
        }

        return $snapshot;
    }

    /**
     * @param  array<string, int>  $stockBefore
     */
    private function log(Product $product, bool $creating, ?bool $wasPublished, array $stockBefore, ?Model $causer): void
    {
        if ($creating) {
            activity('shop')->causedBy($causer)->performedOn($product)->event('created')
                ->withProperties(['attributes' => ['is_published' => $product->is_published, 'stock_qty' => $product->stock_qty]])
                ->log('shop.product.created');

            return;
        }

        if ($wasPublished !== null && $wasPublished !== $product->is_published) {
            activity('shop')->causedBy($causer)->performedOn($product)->event('updated')
                ->withProperties(['old' => ['is_published' => $wasPublished], 'attributes' => ['is_published' => $product->is_published]])
                ->log('shop.product.status');
        }

        $after = $this->stockSnapshot($product);
        $old = [];
        $new = [];
        foreach (array_unique([...array_keys($stockBefore), ...array_keys($after)]) as $key) {
            if (($stockBefore[$key] ?? null) !== ($after[$key] ?? null)) {
                $old[$key] = $stockBefore[$key] ?? null;
                $new[$key] = $after[$key] ?? null;
            }
        }
        if ($new !== []) {
            activity('shop')->causedBy($causer)->performedOn($product)->event('updated')
                ->withProperties(['old' => $old, 'attributes' => $new])
                ->log('shop.product.stock');
        }
    }
}
