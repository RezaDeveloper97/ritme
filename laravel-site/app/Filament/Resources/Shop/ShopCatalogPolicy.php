<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Products and shop categories: shop managers and super-admins manage everything; SEO managers may open and update
 * records but only their SEO tab is saved (content fields disabled and ignored on save — ShopAdmin::canEditContent()).
 * Everyone else is denied.
 */
final class ShopCatalogPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::ShopManager];

    public function viewAny(User $user): bool
    {
        return ShopAdmin::canEditSeo($user);
    }

    public function view(User $user, Model $record): bool
    {
        return ShopAdmin::canEditSeo($user);
    }

    public function update(User $user, Model $record): bool
    {
        return ShopAdmin::canEditSeo($user);
    }
}
