<?php

declare(strict_types=1);

namespace App\Domain\Seo\Observers;

use App\Support\Cache\CacheBumpingObserver;
use Closure;
use Illuminate\Database\Eloquent\Model;

/**
 * A seo_meta change invalidates cached SEO records, sitemaps (sitemap_include / priority / robots) and
 * (via cacheaside.always_bump) the full-page cache.
 *
 * Bulk writes (the bulk SEO editor, L7-06) run inside batch(): every save still goes through this observer, but the
 * namespaces are bumped once when the outermost batch ends instead of once per row.
 */
final class SeoMetaObserver extends CacheBumpingObserver
{
    private static int $depth = 0;

    private static ?Model $pending = null;

    /**
     * @template T
     *
     * @param  Closure(): T  $callback
     * @return T
     */
    public static function batch(Closure $callback): mixed
    {
        self::$depth++;

        try {
            return $callback();
        } finally {
            self::$depth--;
            if (self::$depth === 0 && self::$pending !== null) {
                $model = self::$pending;
                self::$pending = null;
                app(self::class)->flush($model);
            }
        }
    }

    protected function namespaces(Model $model): array
    {
        return ['seo', 'sitemap'];
    }

    protected function bump(Model $model): void
    {
        if (self::$depth > 0) {
            self::$pending = $model;

            return;
        }

        parent::bump($model);
    }

    private function flush(Model $model): void
    {
        parent::bump($model);
    }
}
