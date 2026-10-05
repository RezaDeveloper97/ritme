<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\Redirects;

use App\Domain\Seo\Redirects\Actions\ExportRedirects;
use App\Domain\Seo\Redirects\Models\Redirect;
use Filament\Facades\Filament;
use Illuminate\Support\Facades\Gate;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * GET {admin}/seo/redirects/export — streamed CSV of every redirect (registered by RedirectResource::registerRoutes()
 * inside the panel's auth middleware). Access: RedirectPolicy::export().
 */
final class ExportRedirectsController
{
    public function __invoke(ExportRedirects $export): StreamedResponse
    {
        $user = Filament::auth()->user();
        Gate::forUser($user)->authorize('export', Redirect::class);

        return response()->streamDownload(static function () use ($export, $user): void {
            $out = fopen('php://output', 'wb');
            if ($out === false) {
                return;
            }

            $export->handle($out, $user);
            fclose($out);
        }, ExportRedirects::filename(), [
            'Content-Type' => 'text/csv; charset=UTF-8',
            'Cache-Control' => 'private, no-store',
            'X-Robots-Tag' => 'noindex, nofollow',
        ]);
    }
}
