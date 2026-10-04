<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Support;

use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductSlug;
use App\Support\Html\TextSlug;

/**
 * Product slugs (`/shop/product/{slug}`): normalised (Persian kept, TextSlug), unique against current slugs AND other
 * products' history; a changed slug of a published product goes into history so the old URL 301s (L6-03). Re-using
 * one of the product's own old slugs removes it from history.
 */
final class ProductSlugger
{
    public const MAX_LENGTH = 120;

    public function assign(Product $product): void
    {
        $current = trim((string) $product->getAttribute('slug'));

        if ($product->exists && $current !== '' && ! $product->isDirty('slug')) {
            return;
        }

        $base = TextSlug::make($current !== '' ? $current : $product->title, self::MAX_LENGTH);
        $product->slug = $this->unique($base === '' ? 'product' : $base, $product->exists ? $product->id : null);
    }

    public function recordChange(Product $product): void
    {
        if (! $product->wasChanged('slug')) {
            return;
        }

        ProductSlug::query()->where('product_id', $product->id)->where('slug', $product->slug)->delete();

        $previous = (string) $product->getOriginal('slug');
        $wasPublished = (bool) $product->getOriginal('is_published') || $product->is_published;
        if ($previous === '' || ! $wasPublished) {
            return; // a draft that was never public has no URL worth redirecting
        }

        if (! ProductSlug::query()->where('slug', $previous)->exists()) {
            ProductSlug::query()->create(['product_id' => $product->id, 'slug' => $previous]);
        }
    }

    private function unique(string $base, ?int $ignoreId): string
    {
        $candidate = $base;
        for ($n = 2; $this->taken($candidate, $ignoreId); $n++) {
            $suffix = '-'.$n;
            $candidate = mb_substr($base, 0, self::MAX_LENGTH - strlen($suffix), 'UTF-8').$suffix;
        }

        return $candidate;
    }

    private function taken(string $slug, ?int $ignoreId): bool
    {
        $live = Product::query()->where('slug', $slug)->when($ignoreId !== null, static fn ($q) => $q->whereKeyNot($ignoreId))->exists();

        return $live || ProductSlug::query()->where('slug', $slug)->when($ignoreId !== null, static fn ($q) => $q->where('product_id', '!=', $ignoreId))->exists();
    }
}
