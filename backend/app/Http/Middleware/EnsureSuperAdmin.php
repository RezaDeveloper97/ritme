<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Symfony\Component\HttpFoundation\Response;

/**
 * Restricts a route to super admins (e.g. managing other admin accounts).
 */
class EnsureSuperAdmin
{
    public function handle(Request $request, Closure $next): Response
    {
        $admin = Auth::guard('admin')->user();

        if (! $admin || ! $admin->isSuper()) {
            abort(403, 'دسترسی فقط برای ادمین ارشد مجاز است.');
        }

        return $next($request);
    }
}
