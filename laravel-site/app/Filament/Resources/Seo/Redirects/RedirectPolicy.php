<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\Redirects;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;

/**
 * Redirects (L7-03): SEO managers and super-admins; every other role is denied.
 */
final class RedirectPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::SeoManager];

    public function import(User $user): bool
    {
        return $this->allows($user);
    }

    public function export(User $user): bool
    {
        return $this->allows($user);
    }
}
