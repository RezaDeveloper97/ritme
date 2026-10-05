<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\Observers;

use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Settings\Models\Setting;
use App\Support\Cache\CacheBumpingObserver;
use Illuminate\Database\Eloquent\Model;

/**
 * SEO settings feed the crawler files (robots rules, sitemap include / priorities, per-type robots), which are cached
 * in the `sitemap` namespace: any write to the `seo` group bumps it (SettingObserver already bumps settings/seo/pages).
 */
final class IndexingSettingObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return $model instanceof Setting && $model->getAttribute('group') === SettingGroup::Seo->value ? ['sitemap'] : [];
    }
}
