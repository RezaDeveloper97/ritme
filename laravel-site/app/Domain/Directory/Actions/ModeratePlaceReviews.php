<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\PlaceReview;

/**
 * Approves / rejects reviews (admin queue, L5-06). Saved one by one so the observer recalculates each place's rating
 * and bumps the caches. Returns the number of reviews whose status changed.
 */
final class ModeratePlaceReviews
{
    /**
     * @param  list<int>  $reviewIds
     */
    public function handle(array $reviewIds, ReviewStatus $status): int
    {
        $changed = 0;
        foreach (PlaceReview::query()->whereKey(array_values(array_unique($reviewIds)))->get() as $review) {
            if ($review->status === $status) {
                continue;
            }
            $review->status = $status;
            $review->save();
            $changed++;
        }

        return $changed;
    }
}
