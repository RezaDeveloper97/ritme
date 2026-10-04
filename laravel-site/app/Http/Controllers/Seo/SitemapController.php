<?php

declare(strict_types=1);

namespace App\Http\Controllers\Seo;

use App\Domain\Seo\Sitemap\Sitemaps;
use Illuminate\Http\Response;

/** `/sitemaps/{key}.xml` and `/sitemaps/{key}-{page}.xml` (cached documents, see Sitemaps). */
final class SitemapController
{
    public function __invoke(Sitemaps $sitemaps, string $file): Response
    {
        [$key, $page] = Sitemaps::parse($file) ?? abort(404);

        return self::xml($sitemaps->file($key, $page) ?? abort(404));
    }

    /** gzip is left to the web server (mod_deflate / .htaccess). */
    public static function xml(string $document): Response
    {
        return new Response($document, 200, [
            'Content-Type' => 'application/xml; charset=UTF-8',
            'Cache-Control' => 'public, max-age=3600',
            'X-Robots-Tag' => 'noindex',
        ]);
    }
}
