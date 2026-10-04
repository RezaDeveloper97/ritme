<?php

declare(strict_types=1);

namespace App\Filament\Components\Seo;

use Filament\Schemas\Components\Component;

/**
 * Live content-analysis checklist of the SEO tab (score + checks). Data comes from SeoFields through viewData().
 */
final class SeoAnalysisPanel extends Component
{
    protected string $view = SeoFields::VIEW_NAMESPACE.'::seo-analysis';

    public static function make(): static
    {
        $static = app(self::class);
        $static->configure();

        return $static;
    }
}
