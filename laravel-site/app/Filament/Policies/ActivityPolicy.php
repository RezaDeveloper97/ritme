<?php

declare(strict_types=1);

namespace App\Filament\Policies;

use App\Filament\Auth\AdminRole;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * The activity log is read-only for everyone (super-admin + support may read it).
 */
final class ActivityPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::Support];

    public function create(User $user): bool
    {
        return false;
    }

    public function update(User $user, Model $record): bool
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

    public function restore(User $user, Model $record): bool
    {
        return false;
    }

    public function restoreAny(User $user): bool
    {
        return false;
    }

    public function reorder(User $user): bool
    {
        return false;
    }
}
