<?php

declare(strict_types=1);

namespace App\Domain\Directory\Contracts;

use App\Domain\Directory\Data\PlaceData;
use App\Domain\Directory\Data\PlacePage;
use App\Domain\Directory\Data\PlaceSearchCriteria;
use App\Domain\Directory\Data\RatingSummaryData;
use App\Domain\Directory\Data\ReviewPage;

/**
 * Read side of the directory places (published places only). Cached in the `directory` namespace.
 */
interface PlaceRepository
{
    public function findPublishedBySlug(string $slug): ?PlaceData;

    /**
     * Current slug of the published place that used to live at $previousSlug (slug history → 301), or null.
     */
    public function currentSlugFor(string $previousSlug): ?string;

    public function search(PlaceSearchCriteria $criteria): PlacePage;

    /**
     * Approved reviews of a place, newest first (demo samples included and flagged).
     */
    public function reviews(int $placeId, int $page = 1, int $perPage = 10): ReviewPage;

    /**
     * Average + count (+ per-aspect averages) of the approved, non-demo reviews.
     */
    public function ratingSummary(int $placeId): RatingSummaryData;
}
