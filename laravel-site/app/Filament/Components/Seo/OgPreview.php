<?php

declare(strict_types=1);

namespace App\Filament\Components\Seo;

use Filament\Schemas\Components\Component;

/**
 * Share-card (Open Graph / Twitter) preview of the SEO tab. Data comes from SeoFields through viewData().
 */
final class OgPreview extends Component
{
    protected string $view = SeoFields::VIEW_NAMESPACE.'::og-preview';

    public static function make(): static
    {
        $static = app(self::class);
        $static->configure();

        return $static;
    }
}
