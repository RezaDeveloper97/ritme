<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog;

use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use App\Filament\Policies\AdminPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * Magazine resources (posts, categories, tags, authors): editors write everything; SEO managers may open and update
 * records, but the forms only let them change the SEO tab (content fields are disabled and ignored on save, see
 * BlogPolicy::canEditContent()). Everyone else — support, shop/directory managers, users without a role — is denied.
 */
abstract class BlogPolicy extends AdminPolicy
{
    protected array $roles = [AdminRole::Editor];

    public function viewAny(User $user): bool
    {
        return self::canEditSeo($user);
    }

    public function view(User $user, Model $record): bool
    {
        return self::canEditSeo($user);
    }

    public function update(User $user, Model $record): bool
    {
        return self::canEditSeo($user);
    }

    /**
     * Content fields (everything but the SEO tab): editors and super-admins.
     */
    public static function canEditContent(mixed $user): bool
    {
        return AdminAccess::allows($user, [AdminRole::Editor]);
    }

    /**
     * The SEO tab: editors, SEO managers and super-admins.
     */
    public static function canEditSeo(mixed $user): bool
    {
        return AdminAccess::allows($user, [AdminRole::Editor, AdminRole::SeoManager]);
    }
}
