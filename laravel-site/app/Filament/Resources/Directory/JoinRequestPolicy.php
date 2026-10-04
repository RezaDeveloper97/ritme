<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * «ثبت مجموعه» requests (personal data of the contact person): directory managers and super-admins review them,
 * approve (→ draft place) or reject (`update`) and delete them. Requests come only from the public join form.
 */
final class JoinRequestPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::DirectoryManager];

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
