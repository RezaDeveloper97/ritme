<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Symfony\Component\HttpFoundation\Response;

/**
 * Logs out and rejects an admin whose account has been deactivated mid-session.
 * Applied after `auth:admin`, so a valid session is already guaranteed here.
 */
class EnsureAdminActive
{
    public function handle(Request $request, Closure $next): Response
    {
        $admin = Auth::guard('admin')->user();

        if (! $admin || ! $admin->is_active) {
            Auth::guard('admin')->logout();
            $request->session()->invalidate();
            $request->session()->regenerateToken();

            return redirect()->route('admin.login')
                ->withErrors(['email' => 'حساب شما غیرفعال شده است.']);
        }

        return $next($request);
    }
}
