<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Models\SeoMeta;

/**
 * Drops every admin SEO override of a static page (deletes its seo_meta row), so it renders its lang defaults again.
 * Deleting through the model fires SeoMetaObserver (bumps `seo`, `sitemap`, `pages`). Returns false when there was
 * nothing to reset.
 */
final class ResetStaticPageSeo
{
    public function handle(StaticPage $page): bool
    {
        $meta = SeoMeta::query()->where('route_name', $page->routeName())->first();

        return $meta !== null && (bool) $meta->delete();
    }
}
