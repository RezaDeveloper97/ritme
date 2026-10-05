<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\NotFoundLogs;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * 404 monitor (L7-03): SEO managers and super-admins may list, resolve (create a redirect) and delete rows; rows are
 * only ever written by the monitor, so nobody creates or edits them here.
 */
final class NotFoundLogPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::SeoManager];

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
}
