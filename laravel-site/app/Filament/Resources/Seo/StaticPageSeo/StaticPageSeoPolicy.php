<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\StaticPageSeo;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;

/**
 * `seo_meta` rows edited directly (the static-pages SEO manager, L7-01): SEO managers and super-admins; every other
 * role is denied. Model SEO tabs (posts, categories …) save through their parent record and its own policy.
 */
final class StaticPageSeoPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::SeoManager];
}
