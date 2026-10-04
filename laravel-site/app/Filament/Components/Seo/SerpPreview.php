<?php

declare(strict_types=1);

namespace App\Filament\Components\Seo;

use Filament\Schemas\Components\Component;

/**
 * Google result preview (desktop + mobile) of the SEO tab. Data comes from SeoFields through viewData().
 */
final class SerpPreview extends Component
{
    protected string $view = SeoFields::VIEW_NAMESPACE.'::serp-preview';

    public static function make(): static
    {
        $static = app(self::class);
        $static->configure();

        return $static;
    }
}
