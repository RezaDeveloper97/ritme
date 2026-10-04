<?php

declare(strict_types=1);

namespace App\Domain\Seo\Repositories;

use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Data\SeoMetaData;
use App\Domain\Seo\Models\SeoMeta;

final class EloquentSeoMetaRepository implements SeoMetaRepository
{
    public function forRoute(string $routeName): ?SeoMetaData
    {
        $meta = SeoMeta::query()->where('route_name', $routeName)->first();

        return $meta === null ? null : SeoMetaData::fromModel($meta);
    }

    public function forModel(string $morphType, int|string $id): ?SeoMetaData
    {
        $meta = SeoMeta::query()->where('seoable_type', $morphType)->where('seoable_id', $id)->first();

        return $meta === null ? null : SeoMetaData::fromModel($meta);
    }
}
