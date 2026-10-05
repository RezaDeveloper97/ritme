<?php

declare(strict_types=1);

namespace App\Filament\Http\Middleware;

use Closure;
use Filament\Facades\Filament;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Logs an admin out after `filament.admin.session_timeout` minutes without a request to the panel, independent of
 * the global SESSION_LIFETIME (the session cookie may outlive the admin login). A login restored from the
 * remember-me cookie is logged out too, so it can't outlive the timeout.
 */
final class EnforceSessionTimeout
{
    public const SESSION_KEY = 'admin.last_activity_at';

    public function handle(Request $request, Closure $next): Response
    {
        $timeout = (int) config('filament.admin.session_timeout', 60);
        $guard = Filament::auth();

        if ($timeout > 0 && $guard->check()) {
            $session = $request->session();
            $last = $session->get(self::SESSION_KEY);

            // L9-04: a login restored from the "remember me" cookie (only ever happens in a new session) would restart the
            // inactivity clock and skip the MFA challenge forever. Such a login must go through the form again.
            $remembered = $guard->viaRemember();

            if ($remembered || (is_int($last) && now()->getTimestamp() - $last > $timeout * 60)) {
                $guard->logout();
                $session->invalidate();
                $session->regenerateToken();

                return redirect()->guest(Filament::getLoginUrl() ?? '/');
            }

            $session->put(self::SESSION_KEY, now()->getTimestamp());
        }

        return $next($request);
    }
}
