<?php

declare(strict_types=1);

namespace App\Domain\Directory\Repositories;

use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\PlaceData;
use App\Domain\Directory\Data\PlacePage;
use App\Domain\Directory\Data\PlaceSearchCriteria;
use App\Domain\Directory\Data\RatingSummaryData;
use App\Domain\Directory\Data\ReviewData;
use App\Domain\Directory\Data\ReviewPage;
use App\Domain\Directory\Enums\ReviewAspect;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Directory\Models\PlaceSlug;
use App\Domain\Directory\Queries\SearchPlaces;

final class EloquentPlaceRepository implements PlaceRepository
{
    public function findPublishedBySlug(string $slug): ?PlaceData
    {
        $place = Place::query()
            ->published()
            ->where('slug', $slug)
            ->with(['category', 'city', 'district', 'amenities', 'services', 'gallery'])
            ->first();

        return $place === null ? null : PlaceData::fromModel($place);
    }

    public function currentSlugFor(string $previousSlug): ?string
    {
        $placeId = PlaceSlug::query()->where('slug', $previousSlug)->value('place_id');
        if ($placeId === null) {
            return null;
        }

        $slug = Place::query()->published()->whereKey((int) $placeId)->value('slug');

        return is_string($slug) ? $slug : null;
    }

    public function search(PlaceSearchCriteria $criteria): PlacePage
    {
        return (new SearchPlaces($criteria))->get();
    }

    public function reviews(int $placeId, int $page = 1, int $perPage = 10): ReviewPage
    {
        $page = max(1, $page);
        $perPage = max(1, min(50, $perPage));
        $query = PlaceReview::query()->approved()->where('place_id', $placeId);
        $total = (clone $query)->count();

        $items = array_values($query
            ->orderByDesc('approved_at')
            ->orderByDesc('id')
            ->forPage($page, $perPage)
            ->get()
            ->map(ReviewData::fromModel(...))
            ->all());

        return new ReviewPage($items, $total, $page, $perPage);
    }

    public function ratingSummary(int $placeId): RatingSummaryData
    {
        $reviews = PlaceReview::query()->counted()->where('place_id', $placeId)->get(['rating', 'aspects']);
        if ($reviews->isEmpty()) {
            return new RatingSummaryData;
        }

        $aspects = [];
        foreach (ReviewAspect::cases() as $aspect) {
            $scores = $reviews
                ->map(static fn (PlaceReview $r): ?int => isset($r->aspects[$aspect->value]) ? (int) $r->aspects[$aspect->value] : null)
                ->filter(static fn (?int $score): bool => $score !== null && $score >= 1 && $score <= 5);
            if ($scores->isNotEmpty()) {
                $aspects[$aspect->value] = round((float) $scores->avg(), 1);
            }
        }

        return new RatingSummaryData(round((float) $reviews->avg('rating'), 1), $reviews->count(), $aspects);
    }
}
