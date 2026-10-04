<?php

declare(strict_types=1);

namespace App\Domain\Seo\Observers;

use App\Support\Cache\CacheBumpingObserver;
use Illuminate\Database\Eloquent\Model;

/**
 * A seo_meta change invalidates cached SEO records, sitemaps (sitemap_include / priority / robots) and
 * (via cacheaside.always_bump) the full-page cache.
 */
final class SeoMetaObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return ['seo', 'sitemap'];
    }
}
