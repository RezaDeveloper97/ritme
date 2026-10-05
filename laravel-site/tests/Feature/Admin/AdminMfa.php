<?php

declare(strict_types=1);

namespace Tests\Feature\Admin;

use App\Models\User;

/**
 * L9-04b: every role in `filament.admin.mfa_required_roles` (default: super-admin, shop-manager, directory-manager,
 * support — the roles that read personal data) must have app MFA before the panel lets it in. Test admins with such a
 * role are enrolled like a real admin would be; other roles stay without MFA (they may opt in from their profile).
 */
final class AdminMfa
{
    public const SECRET = 'JBSWY3DPEHPK3PXP';

    public static function requiredFor(User $user): bool
    {
        $roles = config('filament.admin.mfa_required_roles', []);

        return is_array($roles) && $roles !== [] && $user->hasAnyRole($roles);
    }

    /** Enrols TOTP when one of the user's roles requires it; returns the refreshed user. */
    public static function enrol(User $user): User
    {
        if (self::requiredFor($user)) {
            $user->saveAppAuthenticationSecret(self::SECRET);
        }

        return $user->refresh();
    }
}
