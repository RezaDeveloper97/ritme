<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Bookings board (personal data): directory managers and super-admins view requests, move them along the status flow
 * (`update`, ChangeBookingStatus) and export them. Requests come only from the place page; their content is never
 * edited and they are not deleted here (the parent's booked page reads them).
 */
final class BookingRequestPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::DirectoryManager];

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
