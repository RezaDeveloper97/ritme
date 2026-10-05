<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Product review moderation: shop managers and super-admins approve / reject (`update`) and delete reviews. Reviews
 * come only from the product page form — nobody writes or edits a review text here.
 */
final class ProductReviewPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::ShopManager];

    public function create(User $user): bool
    {
        return false;
    }

    public function reorder(User $user): bool
    {
        return false;
    }

    public function restore(User $user, Model $record): bool
    {
        return false;
    }

    public function restoreAny(User $user): bool
    {
        return false;
    }
}
