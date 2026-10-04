<?php

declare(strict_types=1);

namespace App\Domain\Directory\Support;

/**
 * "Open in maps" deep links for a place (docs: tasks/README.md Maps row — no embedded maps, nothing is loaded from
 * these hosts; they are plain links the visitor may follow): `geo:` URI (Android/iOS map apps), Neshan, Balad and
 * Google Maps.
 */
final readonly class MapLinks
{
    public function __construct(
        public string $geo,
        public string $neshan,
        public string $balad,
        public string $google,
    ) {}

    public static function at(float $latitude, float $longitude, ?string $label = null): self
    {
        $lat = self::coordinate($latitude);
        $lng = self::coordinate($longitude);
        $query = $label === null || trim($label) === '' ? "{$lat},{$lng}" : "{$lat},{$lng}(".rawurlencode(trim($label)).')';

        return new self(
            geo: "geo:{$lat},{$lng}?q={$query}",
            neshan: "https://neshan.org/maps/@{$lat},{$lng},16z,0p",
            balad: "https://balad.ir/location?latitude={$lat}&longitude={$lng}&zoom=16",
            google: "https://www.google.com/maps/search/?api=1&query={$lat},{$lng}",
        );
    }

    private static function coordinate(float $value): string
    {
        return rtrim(rtrim(number_format($value, 6, '.', ''), '0'), '.');
    }
}
