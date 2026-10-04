<?php

declare(strict_types=1);

namespace App\Filament\Resources\Newsletter;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Newsletter subscribers (personal data): content editors and super-admins may list and export them; every other role
 * is denied. Records come only from the public double opt-in form, so nobody creates or edits them here.
 */
final class SubscriberPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::Editor];

    public function create(User $user): bool
    {
        return false;
    }

    public function update(User $user, Model $record): bool
    {
        return false;
    }

    public function reorder(User $user): bool
    {
        return false;
    }

    public function export(User $user): bool
    {
        return $this->allows($user);
    }
}
