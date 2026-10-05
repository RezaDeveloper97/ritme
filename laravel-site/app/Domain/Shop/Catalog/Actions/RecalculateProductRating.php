<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;

/**
 * Stores the rating aggregate of a product from its approved, NON-demo reviews (`rating_avg`, `rating_count`). Written
 * without model events or touching `updated_at` (the review observer bumps the caches); a product with no real reviews
 * has 0 / 0 and shows no rating.
 */
final class RecalculateProductRating
{
    public function handle(int $productId): void
    {
        $stats = ProductReview::query()->counted()->where('product_id', $productId)
            ->toBase()
            ->selectRaw('COUNT(*) as aggregate_count, AVG(rating) as aggregate_avg')
            ->first();

        $count = (int) ($stats->aggregate_count ?? 0);
        $average = $count > 0 ? round((float) $stats?->aggregate_avg, 2) : 0.0;

        Product::query()->whereKey($productId)->toBase()->update(['rating_avg' => $average, 'rating_count' => $count]);
    }
}
