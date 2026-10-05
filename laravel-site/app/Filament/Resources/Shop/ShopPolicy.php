<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;

/**
 * Brands (no SEO tab): shop managers and super-admins; every other role is denied (deny by default).
 */
final class ShopPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::ShopManager];
}
