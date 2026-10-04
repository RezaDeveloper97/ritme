<?php

declare(strict_types=1);

namespace App\Filament\Policies;

use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Base policy for every admin resource: deny by default. A resource policy lists the roles that may manage it in
 * $roles; inactive users and users without one of those roles (or super-admin) are denied everything.
 * Subclasses hard-deny abilities by overriding a method to return false.
 */
abstract class AdminPolicy
{
    /**
     * Roles (besides super-admin) with full access to the resource.
     *
     * @var list<AdminRole>
     */
    protected array $roles = [];

    public function viewAny(User $user): bool
    {
        return $this->allows($user);
    }

    public function view(User $user, Model $record): bool
    {
        return $this->allows($user);
    }

    public function create(User $user): bool
    {
        return $this->allows($user);
    }

    public function update(User $user, Model $record): bool
    {
        return $this->allows($user);
    }

    public function delete(User $user, Model $record): bool
    {
        return $this->allows($user);
    }

    public function deleteAny(User $user): bool
    {
        return $this->allows($user);
    }

    public function restore(User $user, Model $record): bool
    {
        return $this->allows($user);
    }

    public function restoreAny(User $user): bool
    {
        return $this->allows($user);
    }

    public function forceDelete(User $user, Model $record): bool
    {
        return false;
    }

    public function forceDeleteAny(User $user): bool
    {
        return false;
    }

    public function replicate(User $user, Model $record): bool
    {
        return false;
    }

    public function reorder(User $user): bool
    {
        return $this->allows($user);
    }

    protected function allows(User $user): bool
    {
        return AdminAccess::allows($user, $this->roles);
    }
}
