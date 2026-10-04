<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Places;

use App\Domain\Directory\Models\Place;
use App\Filament\Resources\Directory\DirectoryAdmin;

/**
 * Splits the place form state for SavePlace: model attributes (with the hours editor turned into `opening_hours`),
 * amenity ids and ordered gallery media ids — and builds the form state from a place.
 */
final class PlaceFormData
{
    /**
     * @return array<string, mixed>
     */
    public static function fill(Place $place): array
    {
        $data = $place->attributesToArray();
        $data['booking_mode'] = $place->booking_mode->value;
        $data['hours'] = DirectoryAdmin::hoursState($place->opening_hours);
        $data['amenity_ids'] = $place->amenities()->pluck('directory_amenities.id')->map(intval(...))->all();
        $data['gallery_items'] = $place->gallery()->pluck('media.id')->map(static fn (mixed $id): int => (int) $id)->values()->all();

        return $data;
    }

    /**
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    public static function attributes(array $data): array
    {
        $data['opening_hours'] = DirectoryAdmin::toOpeningHours($data['hours'] ?? null);
        $data['phones'] = array_values(array_filter(array_map(strval(...), (array) ($data['phones'] ?? [])), static fn (string $p): bool => trim($p) !== ''));
        $data['phones'] = $data['phones'] === [] ? null : $data['phones'];
        unset($data['hours'], $data['amenity_ids'], $data['gallery_items'], $data['services'], $data['seoMeta']);

        return $data;
    }

    /**
     * @param  array<string, mixed>  $data
     * @return list<int>
     */
    public static function amenityIds(array $data): array
    {
        return array_values(array_map(intval(...), array_filter((array) ($data['amenity_ids'] ?? []), is_numeric(...))));
    }

    /**
     * @param  array<string, mixed>  $data
     * @return list<int>
     */
    public static function galleryIds(array $data): array
    {
        return array_values(array_map(intval(...), array_filter((array) ($data['gallery_items'] ?? []), is_numeric(...))));
    }
}
