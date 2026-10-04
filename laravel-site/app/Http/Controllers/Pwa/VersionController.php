<?php

declare(strict_types=1);

namespace App\Http\Controllers\Pwa;

use App\Domain\Pwa\Version\AppVersion;
use Illuminate\Http\JsonResponse;

/**
 * `/pwa/version.json` (L8-02): deployed build, minimum allowed build and update message for the two-tier update.
 * Never cached anywhere (no-store; the service worker bypasses /pwa as well), no session or cookies.
 */
final class VersionController
{
    public function __invoke(AppVersion $version): JsonResponse
    {
        return new JsonResponse($version->current()->toArray(), 200, [
            'Cache-Control' => 'no-store, max-age=0',
            'X-Robots-Tag' => 'noindex',
        ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    }
}
