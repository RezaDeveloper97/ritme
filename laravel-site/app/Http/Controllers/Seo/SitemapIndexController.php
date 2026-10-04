<?php

declare(strict_types=1);

namespace App\Http\Controllers\Seo;

use App\Domain\Seo\Sitemap\Sitemaps;
use Illuminate\Http\Response;

/** `/sitemap.xml`: the sitemap index (cached document, see Sitemaps). */
final class SitemapIndexController
{
    public function __invoke(Sitemaps $sitemaps): Response
    {
        return SitemapController::xml($sitemaps->index());
    }
}
