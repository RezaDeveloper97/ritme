<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceReview;

/**
 * Stores the rating aggregate of a place from its approved, NON-demo reviews (`rating_avg`, `rating_count`). Written
 * without model events or touching `updated_at` (the caller — the review observer — bumps the caches); a place with
 * no real reviews has 0 / 0 and shows no rating.
 */
final class RecalculatePlaceRating
{
    public function handle(int $placeId): void
    {
        $stats = PlaceReview::query()->counted()->where('place_id', $placeId)
            ->toBase()
            ->selectRaw('COUNT(*) as aggregate_count, AVG(rating) as aggregate_avg')
            ->first();

        $count = (int) ($stats->aggregate_count ?? 0);
        $average = $count > 0 ? round((float) $stats->aggregate_avg, 2) : 0.0;

        Place::query()->whereKey($placeId)->toBase()->update(['rating_avg' => $average, 'rating_count' => $count]);
    }
}
