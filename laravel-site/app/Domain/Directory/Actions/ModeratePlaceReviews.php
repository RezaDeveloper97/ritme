<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Directory\Support\DirectoryActivity;
use Illuminate\Database\Eloquent\Model;

/**
 * Approves / rejects reviews (admin queue, L5-06). Saved one by one so the observer recalculates each place's rating
 * and bumps the caches; every change is written to the activity log (`directory`, `directory.review.status`).
 * Returns the number of reviews whose status changed.
 */
final class ModeratePlaceReviews
{
    /**
     * @param  list<int>  $reviewIds
     */
    public function handle(array $reviewIds, ReviewStatus $status, ?Model $causer = null): int
    {
        $changed = 0;
        foreach (PlaceReview::query()->whereKey(array_values(array_unique($reviewIds)))->get() as $review) {
            if ($review->status === $status) {
                continue;
            }
            $from = $review->status;
            $review->status = $status;
            $review->save();
            $changed++;

            DirectoryActivity::status($review, 'directory.review.status', $from->value, $status->value, $causer);
        }

        return $changed;
    }
}
