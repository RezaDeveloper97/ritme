<?php

declare(strict_types=1);

namespace App\Filament\Resources\ContactMessages;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Contact inbox (personal data): the support role and super-admins read, triage (read/unread/archive), delete and
 * export messages; every other role is denied. Messages come only from the public form, so nobody creates them here,
 * and their content is never edited — `update` covers status changes only.
 */
final class ContactMessagePolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::Support];

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

    public function export(User $user): bool
    {
        return $this->allows($user);
    }
}
