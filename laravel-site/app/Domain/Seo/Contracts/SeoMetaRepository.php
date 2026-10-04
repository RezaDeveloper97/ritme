<?php

declare(strict_types=1);

namespace App\Domain\Seo\Contracts;

use App\Domain\Seo\Data\SeoMetaData;

interface SeoMetaRepository
{
    /**
     * SEO overrides of a static page, keyed by its route name.
     */
    public function forRoute(string $routeName): ?SeoMetaData;

    /**
     * SEO overrides of a model page (seo_meta morph row).
     */
    public function forModel(string $morphType, int|string $id): ?SeoMetaData;
}
