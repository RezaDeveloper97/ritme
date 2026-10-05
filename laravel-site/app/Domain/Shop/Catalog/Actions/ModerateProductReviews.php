<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\ProductReview;
use Illuminate\Database\Eloquent\Model;

/**
 * Approves / rejects product reviews (admin queue, L6-06). Saved one by one so ProductReviewObserver stamps
 * `approved_at`, recalculates each product's rating (approved, non-demo reviews only) and bumps the caches; every
 * change is written to the activity log (`shop`, `shop.review.status`). Returns the number of reviews changed.
 */
final class ModerateProductReviews
{
    /**
     * @param  list<int>  $reviewIds
     */
    public function handle(array $reviewIds, ReviewStatus $status, ?Model $causer = null): int
    {
        $changed = 0;
        foreach (ProductReview::query()->whereKey(array_values(array_unique($reviewIds)))->get() as $review) {
            if ($review->status === $status) {
                continue;
            }
            $from = $review->status;
            $review->status = $status;
            $review->save();
            $changed++;

            activity('shop')
                ->causedBy($causer)
                ->performedOn($review)
                ->event('updated')
                ->withProperties(['old' => ['status' => $from->value], 'attributes' => ['status' => $status->value]])
                ->log('shop.review.status');
        }

        return $changed;
    }
}
