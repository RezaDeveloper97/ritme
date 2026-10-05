<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Observers;

use App\Support\Cache\CacheBumpingObserver;
use Illuminate\Database\Eloquent\Model;

/**
 * A redirect change invalidates the cached redirect map (`seo`) and, via cacheaside.always_bump, `pages`. Hit counters
 * are written through the query builder and never reach this observer.
 */
final class RedirectObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return ['seo'];
    }
}
