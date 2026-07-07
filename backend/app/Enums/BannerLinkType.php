<?php

namespace App\Enums;

/**
 * How a banner's link should be opened by the client:
 *  - INTERNAL: an in-app route (e.g. "/calendar") opened with the SPA router.
 *  - EXTERNAL: an absolute URL opened in a new browser tab.
 */
enum BannerLinkType: string
{
    case INTERNAL = 'internal';
    case EXTERNAL = 'external';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::INTERNAL => 'داخلی (درون برنامه)',
                self::EXTERNAL => 'خارجی (تب جدید)',
            },
            default => match ($this) {
                self::INTERNAL => 'Internal (in-app)',
                self::EXTERNAL => 'External (new tab)',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
