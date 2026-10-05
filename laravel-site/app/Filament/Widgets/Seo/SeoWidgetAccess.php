<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Seo;

use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use Filament\Facades\Filament;

/**
 * SEO dashboard widgets (L7-05) are for SEO managers and super-admins only (deny by default).
 */
trait SeoWidgetAccess
{
    public static function canView(): bool
    {
        return AdminAccess::allows(Filament::auth()->user(), [AdminRole::SeoManager]);
    }
}
