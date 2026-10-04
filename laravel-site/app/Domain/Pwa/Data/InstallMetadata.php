<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Data;

/**
 * What <x-pwa.head> prints: manifest link, theme colours (light / dark scheme), app names and the icon links.
 */
final readonly class InstallMetadata
{
    public function __construct(
        public string $manifestUrl,
        public string $applicationName,
        public string $themeColorLight,
        public string $themeColorDark,
        public string $faviconIco,
        public ?string $faviconSvg,
        public ?string $faviconPng,
        public string $appleTouchIcon,
        public string $maskIcon,
        public string $maskIconColor,
    ) {}

    /**
     * @return array<string, string|null>
     */
    public function toArray(): array
    {
        return get_object_vars($this);
    }

    /**
     * @param  array<string, mixed>  $values
     */
    public static function fromArray(array $values): self
    {
        $string = static fn (string $key): string => is_string($values[$key] ?? null) ? $values[$key] : '';
        $nullable = static fn (string $key): ?string => is_string($values[$key] ?? null) ? $values[$key] : null;

        return new self(
            manifestUrl: $string('manifestUrl'),
            applicationName: $string('applicationName'),
            themeColorLight: $string('themeColorLight'),
            themeColorDark: $string('themeColorDark'),
            faviconIco: $string('faviconIco'),
            faviconSvg: $nullable('faviconSvg'),
            faviconPng: $nullable('faviconPng'),
            appleTouchIcon: $string('appleTouchIcon'),
            maskIcon: $string('maskIcon'),
            maskIconColor: $string('maskIconColor'),
        );
    }
}
