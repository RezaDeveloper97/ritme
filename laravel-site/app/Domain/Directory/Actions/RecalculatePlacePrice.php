<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceService;

/**
 * Stores the cheapest announced service price of a place (`price_from`, «از ۳۲۰ هزار تومان»; null without prices).
 * Written without model events; the service observer bumps the caches.
 */
final class RecalculatePlacePrice
{
    public function handle(int $placeId): void
    {
        $min = PlaceService::query()->where('place_id', $placeId)->whereNotNull('price')->where('price', '>', 0)->min('price');

        Place::query()->whereKey($placeId)->toBase()->update(['price_from' => $min === null ? null : (int) $min]);
    }
}
