<?php

declare(strict_types=1);

namespace App\Domain\Directory\Support;

use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\PlaceCardData;
use App\Domain\Directory\Data\PlaceSearchCriteria;

/**
 * Which directory lists may be indexed (L5-02, L7-05b) — one rule for ListPlacesController (robots) and
 * PagesSitemapProvider (`/directory` in `pages.xml`): a list that shows no real (non-demo) place is thin content →
 * `noindex,follow` and out of the sitemap.
 */
final class PlaceListIndexing
{
    /** Cards per listing page (ListPlacesController::PER_PAGE). */
    public const PER_PAGE = 12;

    public function __construct(private readonly PlaceRepository $places) {}

    /**
     * @param  list<PlaceCardData>  $items  the cards the page shows
     */
    public static function showsRealPlace(array $items): bool
    {
        foreach ($items as $card) {
            if (! $card->isDemo) {
                return true;
            }
        }

        return false;
    }

    /**
     * `/directory` without filters, page 1 — the same criteria (and cached search) the controller uses.
     */
    public function indexIndexable(): bool
    {
        return self::showsRealPlace($this->places->search(new PlaceSearchCriteria(perPage: self::PER_PAGE))->items);
    }
}
