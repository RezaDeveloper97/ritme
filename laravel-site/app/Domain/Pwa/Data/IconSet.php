<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Data;

/**
 * Every icon URL the site advertises: manifest icons (any / maskable / monochrome), the shortcut icon and the head
 * icons (favicons, apple-touch-icon, Safari mask icon). Either the committed defaults in public/icons or the set
 * generated from the admin-uploaded logo (App\Domain\Pwa\Actions\GeneratePwaIcons).
 */
final readonly class IconSet
{
    /**
     * @param  list<ManifestIcon>  $icons
     */
    public function __construct(
        public array $icons,
        public ManifestIcon $shortcutIcon,
        public string $appleTouchIcon,
        public string $faviconIco,
        public ?string $faviconSvg,
        public ?string $faviconPng,
        public string $maskIcon,
    ) {}

}
