<?php

declare(strict_types=1);

namespace App\Filament\Resources\Faq;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;

/**
 * FAQ groups and items (site copy): content editors and super-admins; every other role is denied.
 */
final class FaqPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::Editor];
}
