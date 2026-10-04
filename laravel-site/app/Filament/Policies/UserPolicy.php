<?php

declare(strict_types=1);

namespace App\Filament\Policies;

use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Admin accounts: super-admin only. Nobody deletes their own account (no lock-out by accident).
 */
final class UserPolicy extends AdminPolicy
{
    public function delete(User $user, Model $record): bool
    {
        return $user->isNot($record) && parent::delete($user, $record);
    }
}
