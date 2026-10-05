<?php

declare(strict_types=1);

namespace App\Filament\Http\Middleware;

use App\Models\User;
use Closure;
use Filament\Auth\MultiFactor\Http\Middleware\EnsureMultiFactorAuthenticationIsEnabled;
use Filament\Facades\Filament;
use Illuminate\Http\Request;

/**
 * Filament decides "MFA required" once, at route registration (no user yet), so the panel marks it required for
 * everyone and this middleware narrows it to `filament.admin.mfa_required_roles` (`ADMIN_MFA_ROLES`; default super-admin + the PII roles). Other roles may
 * still enable app MFA from their profile.
 */
final class RequireMultiFactorForRoles extends EnsureMultiFactorAuthenticationIsEnabled
{
    public function handle(Request $request, Closure $next): mixed
    {
        $user = Filament::auth()->user();
        $roles = config('filament.admin.mfa_required_roles', []);

        if (! $user instanceof User || ! is_array($roles) || $roles === [] || ! $user->hasAnyRole($roles)) {
            return $next($request);
        }

        return parent::handle($request, $next);
    }
}
