<?php

declare(strict_types=1);

namespace App\Domain\Content;

use App\Domain\Content\Enums\SiteSection;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Settings\Contracts\SettingsRepository;

/**
 * Answers «is this part of the site switched on?» from the `pages` settings group (cached settings read, no query
 * when warm). Anything that is not a SiteSection — home, privacy, terms, feeds outside a section — is always on.
 */
final class SiteSections
{
    public function __construct(private readonly SettingsRepository $settings) {}

    public function enabled(SiteSection $section): bool
    {
        return $this->settings->all()->pages->enabled($section);
    }

    public function routeEnabled(?string $routeName): bool
    {
        $section = SiteSection::forRoute($routeName);

        return $section === null || $this->enabled($section);
    }

    public function pageEnabled(StaticPage $page): bool
    {
        return $this->routeEnabled($page->routeName());
    }

    public function sitemapEnabled(string $providerKey): bool
    {
        $section = SiteSection::forSitemap($providerKey);

        return $section === null || $this->enabled($section);
    }

    public function searchEnabled(string $providerKey): bool
    {
        $section = SiteSection::forSearchProvider($providerKey);

        return $section === null || $this->enabled($section);
    }
}
