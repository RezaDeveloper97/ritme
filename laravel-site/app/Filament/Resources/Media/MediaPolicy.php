<?php

declare(strict_types=1);

namespace App\Filament\Resources\Media;

use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;

/**
 * Media library: every content-managing role (editors and up); support and users without a role are denied.
 * Deleting is further limited to unused items by DeleteUnusedMedia.
 */
final class MediaPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::Editor, AdminRole::SeoManager, AdminRole::ShopManager, AdminRole::DirectoryManager];
}
