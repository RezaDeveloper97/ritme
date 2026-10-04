<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;

/**
 * Mother & child directory content (places, categories, cities + districts, amenities, landing texts): the
 * directory-manager role and super-admins manage everything; every other role is denied (deny by default).
 */
final class DirectoryPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::DirectoryManager];
}
