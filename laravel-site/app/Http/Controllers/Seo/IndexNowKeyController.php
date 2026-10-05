<?php

declare(strict_types=1);

namespace App\Http\Controllers\Seo;

use App\Domain\Seo\Indexing\IndexNow\IndexNow;
use Illuminate\Http\Response;

/**
 * `/{key}.txt` — the IndexNow key file (L7-04): the key as plain text while IndexNow is switched on and the requested
 * key is the configured one; 404 otherwise. Registered by SeoServiceProvider without session / cookies.
 */
final class IndexNowKeyController
{
    public const ROUTE = 'seo.indexnow-key';

    public function __invoke(string $key, IndexNow $indexNow): Response
    {
        $configured = $indexNow->key();
        abort_unless($configured !== null && hash_equals($configured, $key), 404);

        return new Response($configured, 200, [
            'Content-Type' => 'text/plain; charset=UTF-8',
            'Cache-Control' => 'public, max-age=3600',
            'X-Robots-Tag' => 'noindex',
        ]);
    }
}
