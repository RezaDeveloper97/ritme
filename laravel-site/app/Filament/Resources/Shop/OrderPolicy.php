<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Orders (personal data): shop managers and super-admins view them, move them along the lifecycle (`update`,
 * ChangeOrderStatus), add notes, print invoices and export. Orders come only from checkout: nobody creates, edits
 * the content of or deletes an order here (the customer's order page reads it).
 */
final class OrderPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::ShopManager];

    public function create(User $user): bool
    {
        return false;
    }

    public function delete(User $user, Model $record): bool
    {
        return false;
    }

    public function deleteAny(User $user): bool
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

    public function export(User $user): bool
    {
        return $this->allows($user);
    }
}
