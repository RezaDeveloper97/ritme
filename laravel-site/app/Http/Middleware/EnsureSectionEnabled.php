<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use App\Domain\Content\SiteSections;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * 404 for every route of a site section switched off in the admin «صفحه‌ها» settings (SiteSection::forRoute).
 * Runs in the `web` group before PageCache, so a page cached while the section was on is never served after it was
 * switched off (the settings write bumps `pages` anyway).
 */
final class EnsureSectionEnabled
{
    public function __construct(private readonly SiteSections $sections) {}

    public function handle(Request $request, Closure $next): Response
    {
        abort_unless($this->sections->routeEnabled($request->route()?->getName()), 404);

        return $next($request);
    }
}
