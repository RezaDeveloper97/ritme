<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing;

use App\Domain\Seo\Sitemap\DefaultRobotsRules;
use App\Domain\Seo\Sitemap\RobotsRules;

/**
 * The production crawl rules: the admin-edited robots.txt (settings `seo.robots_txt`, validated on save by
 * RobotsTxtValidator) or, when empty / reset, the built-in DefaultRobotsRules. Non-production environments never get
 * here (RobotsTxt serves `Disallow: /`). Settings writes bump `sitemap` (IndexingSettingObserver), where RobotsTxt
 * caches the document.
 */
final class SettingsRobotsRules implements RobotsRules
{
    public function __construct(
        private readonly IndexingRules $indexing,
        private readonly DefaultRobotsRules $defaults,
    ) {}

    public function rules(): string
    {
        return $this->indexing->settings()->robotsTxt ?? $this->defaults->rules();
    }

    public function defaults(): string
    {
        return $this->defaults->rules();
    }
}
