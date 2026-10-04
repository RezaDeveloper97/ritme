<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Review moderation queue: directory managers and super-admins approve / reject (`update`) and delete reviews.
 * Reviews come only from the public form — nobody writes or edits a review text here.
 */
final class PlaceReviewPolicy extends AdminPolicy
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
