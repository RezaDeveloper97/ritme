<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Models\Place;
use App\Support\Cache\NamespaceBumper;

/**
 * Replaces a place's gallery with the given media ids in this order (sort_order = position). Bumps `directory` and
 * `sitemap` (+ `pages`).
 */
final class SyncPlaceGallery
{
    public function __construct(private readonly NamespaceBumper $bumper) {}

    /**
     * @param  list<int>  $mediaIds
     */
    public function handle(Place $place, array $mediaIds): void
    {
        $ordered = [];
        foreach (array_values(array_unique(array_map(intval(...), $mediaIds))) as $position => $mediaId) {
            $ordered[$mediaId] = ['sort_order' => $position];
        }

        $place->gallery()->sync($ordered);
        $this->bumper->bumpFor($place, ['directory', 'sitemap']);
    }
}
