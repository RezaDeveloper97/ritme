<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Models\Place;
use App\Support\Cache\NamespaceBumper;

/**
 * Replaces a place's amenities. Pivot writes fire no model events, so the caches are bumped here when anything changed.
 */
final class SyncPlaceAmenities
{
    public function __construct(private readonly NamespaceBumper $bumper) {}

    /**
     * @param  list<int>  $amenityIds
     */
    public function handle(Place $place, array $amenityIds): void
    {
        $changes = $place->amenities()->sync(array_values(array_unique(array_map(intval(...), $amenityIds))));

        if (array_filter($changes) !== []) {
            $this->bumper->bumpFor($place, ['directory']);
        }
    }
}
