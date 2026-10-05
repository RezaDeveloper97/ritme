<?php

declare(strict_types=1);

namespace App\Domain\Seo\StaticPages;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\SeoManager;

/**
 * Applies a static page's default title / description (StaticPageSeoDefaults — the one place they live) to the
 * request's SeoManager, unless the admin's seo_meta row for the page's route overrides them.
 *
 * Holds no request state: the request-scoped SeoManager is passed in per call, so controllers (and this service)
 * may outlive a request in tests and long-running workers.
 */
final class StaticPageSeo
{
    public function __construct(
        private readonly SeoMetaRepository $meta,
        private readonly StaticPageSeoDefaults $defaults,
    ) {}

    public function apply(SeoManager $seo, StaticPage $page): void
    {
        $meta = $this->meta->forRoute($page->routeName());
        $defaults = $this->defaults->for($page);

        if (($meta->title ?? '') === '' && $defaults->titleIsComplete) {
            $seo->rawTitle($defaults->title);
        } elseif (($meta->title ?? '') === '') {
            $seo->title($defaults->title); // through the site title template
        }
        if (($meta->description ?? '') === '') {
            $seo->description($defaults->description);
        }
    }
}
