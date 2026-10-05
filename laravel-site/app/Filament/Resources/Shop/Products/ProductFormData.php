<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Products;

use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Filament\Resources\Shop\ShopAdmin;
use Illuminate\Validation\ValidationException;

/**
 * Splits the product form state for SaveProduct and builds the form state from a product. Prices are typed in TOMANS
 * (`price_toman`, `compare_at_toman`, per variant the same) and become Money in rials here; the size chart editor is
 * «columns» + one line per row with cells separated by `|`.
 */
final class ProductFormData
{
    public const CELL_SEPARATOR = '|';

    /**
     * @return array<string, mixed>
     */
    public static function fill(Product $product): array
    {
        $data = $product->attributesToArray();
        unset($data['price'], $data['compare_at_price']); // Money objects: the form edits tomans instead
        $data['price_toman'] = ShopAdmin::toToman($product->price);
        $data['compare_at_toman'] = ShopAdmin::toToman($product->compare_at_price);
        $data['stock_mode'] = in_array($product->stock_status, [StockStatus::PreOrder, StockStatus::BackOrder], true)
            ? $product->stock_status->value
            : 'auto';
        $data['life_stages'] = $product->life_stages ?? [];
        $data['specs'] = $product->specs ?? [];
        $data['size_chart_columns'] = $product->size_chart['columns'] ?? [];
        $data['size_chart_rows'] = array_map(
            static fn (array $row): array => ['cells' => implode(' '.self::CELL_SEPARATOR.' ', $row)],
            $product->size_chart['rows'] ?? [],
        );

        $variants = [];
        foreach ($product->variants()->get() as $variant) {
            $variants[] = [
                'id' => $variant->id,
                'size' => $variant->size,
                'color' => $variant->color,
                'color_hex' => $variant->color_hex,
                'sku' => $variant->sku,
                'price_toman' => ShopAdmin::toToman($variant->price),
                'compare_at_toman' => ShopAdmin::toToman($variant->compare_at_price),
                'stock_qty' => $variant->stock_qty,
                'is_active' => $variant->is_active,
            ];
        }
        $data['variants'] = $variants;

        $data['category_ids'] = $product->categories()->pluck('shop_categories.id')->map(intval(...))->all();
        $data['gallery_items'] = $product->gallery()->pluck('media.id')->map(intval(...))->values()->all();
        $data['cross_sell_ids'] = $product->crossSells()->pluck('shop_products.id')->map(intval(...))->values()->all();

        return $data;
    }

    /**
     * Model attributes for SaveProduct.
     *
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    public static function attributes(array $data): array
    {
        $attributes = [];
        foreach (['title', 'slug', 'sku', 'brand_id', 'short_description', 'description', 'badge', 'weight_grams', 'cover_media_id', 'sort_order'] as $key) {
            if (array_key_exists($key, $data)) {
                $attributes[$key] = $data[$key] === '' ? null : $data[$key];
            }
        }
        $attributes['slug'] = trim((string) ($attributes['slug'] ?? ''));
        $attributes['sort_order'] = (int) ($attributes['sort_order'] ?? 0);
        foreach (['is_published', 'is_featured', 'is_demo'] as $flag) {
            $attributes[$flag] = (bool) ($data[$flag] ?? false);
        }

        $attributes['price'] = ShopAdmin::fromToman($data['price_toman'] ?? null) ?? ShopAdmin::fromToman(0);
        $attributes['compare_at_price'] = ShopAdmin::fromToman($data['compare_at_toman'] ?? null);
        $attributes['stock_qty'] = max(0, (int) ($data['stock_qty'] ?? 0));
        $attributes['stock_status'] = (string) ($data['stock_mode'] ?? 'auto');
        $attributes['life_stages'] = array_values(array_filter((array) ($data['life_stages'] ?? []), is_string(...)));
        $attributes['specs'] = array_values(array_filter((array) ($data['specs'] ?? []), is_array(...)));
        $attributes['size_chart'] = self::sizeChart($data);

        return $attributes;
    }

    /**
     * Variant rows for SaveProduct (order = display order). Refuses two rows with the same size + colour or sku.
     *
     * @param  array<string, mixed>  $data
     * @return list<array<string, mixed>>
     *
     * @throws ValidationException
     */
    public static function variants(array $data): array
    {
        $variants = [];
        $combos = [];
        $skus = [];
        foreach (array_values(array_filter((array) ($data['variants'] ?? []), is_array(...))) as $row) {
            $size = trim((string) ($row['size'] ?? ''));
            $color = trim((string) ($row['color'] ?? ''));
            $sku = trim((string) ($row['sku'] ?? ''));

            $combo = mb_strtolower($size.'|'.$color);
            if (isset($combos[$combo])) {
                throw ValidationException::withMessages(['data.variants' => 'دو تنوع با سایز و رنگ یکسان تعریف شده است.']);
            }
            $combos[$combo] = true;
            if ($sku !== '') {
                if (isset($skus[$sku])) {
                    throw ValidationException::withMessages(['data.variants' => 'کد کالای (SKU) تنوع‌ها نباید تکراری باشد.']);
                }
                $skus[$sku] = true;
            }

            $variants[] = [
                'id' => is_numeric($row['id'] ?? null) ? (int) $row['id'] : null,
                'size' => $size,
                'color' => $color,
                'color_hex' => trim((string) ($row['color_hex'] ?? '')),
                'sku' => $sku,
                'price' => ShopAdmin::fromToman($row['price_toman'] ?? null),
                'compare_at_price' => ShopAdmin::fromToman($row['compare_at_toman'] ?? null),
                'stock_qty' => max(0, (int) ($row['stock_qty'] ?? 0)),
                'is_active' => (bool) ($row['is_active'] ?? true),
            ];
        }

        return $variants;
    }

    /**
     * @param  array<string, mixed>  $data
     * @return list<int>
     */
    public static function categoryIds(array $data): array
    {
        return self::ids($data['category_ids'] ?? []);
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function primaryCategoryId(array $data): ?int
    {
        return is_numeric($data['primary_category_id'] ?? null) ? (int) $data['primary_category_id'] : null;
    }

    /**
     * @param  array<string, mixed>  $data
     * @return list<int>
     */
    public static function galleryIds(array $data): array
    {
        return self::ids($data['gallery_items'] ?? []);
    }

    /**
     * @param  array<string, mixed>  $data
     * @return list<int>
     */
    public static function crossSellIds(array $data): array
    {
        return self::ids($data['cross_sell_ids'] ?? []);
    }

    /**
     * @param  array<string, mixed>  $data
     * @return array{columns: list<string>, rows: list<list<string>>}|null
     */
    private static function sizeChart(array $data): ?array
    {
        $columns = array_values(array_filter(array_map(
            static fn (mixed $c): string => is_scalar($c) ? trim((string) $c) : '',
            (array) ($data['size_chart_columns'] ?? []),
        ), static fn (string $c): bool => $c !== ''));
        if ($columns === []) {
            return null;
        }

        $rows = [];
        foreach ((array) ($data['size_chart_rows'] ?? []) as $row) {
            $line = is_array($row) ? (string) ($row['cells'] ?? '') : (is_scalar($row) ? (string) $row : '');
            if (trim($line) !== '') {
                $rows[] = array_map(trim(...), explode(self::CELL_SEPARATOR, $line));
            }
        }

        return ['columns' => $columns, 'rows' => $rows]; // ProductContent::sizeChart() pads, trims and caps it
    }

    /**
     * @return list<int>
     */
    private static function ids(mixed $values): array
    {
        return array_values(array_map(intval(...), array_filter((array) $values, is_numeric(...))));
    }
}
