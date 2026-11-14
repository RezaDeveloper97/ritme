<?php

declare(strict_types=1);

namespace App\Domain\Settings\Observers;

use App\Support\Cache\CacheBumpingObserver;
use Illuminate\Database\Eloquent\Model;

/**
 * Any settings change invalidates cached settings, SEO fragments built from SEO defaults / Organization data,
 * the sitemap documents (a section switched off in «صفحه‌ها» leaves them) and (via cacheaside.always_bump) the full-page cache.
 */
final class SettingObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return ['settings', 'seo', 'sitemap'];
    }
}
