<?php

declare(strict_types=1);

namespace App\Filament\Auth;

use App\Models\User;

/**
 * The one place that decides "may this user manage X": active, and super-admin or one of the given roles.
 * Used by resource policies and by pages (settings, dashboard widgets) that have no model.
 */
final class AdminAccess
{
    /**
     * @param  list<AdminRole>  $roles  roles besides super-admin
     */
    public static function allows(mixed $user, array $roles = []): bool
    {
        if (! $user instanceof User || ! $user->is_active) {
            return false;
        }

        $names = array_map(static fn (AdminRole $role): string => $role->value, $roles);
        $names[] = AdminRole::SuperAdmin->value;

        return $user->hasAnyRole($names);
    }
}
