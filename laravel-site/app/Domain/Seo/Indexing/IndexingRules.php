<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing;

use App\Domain\Seo\Support\Robots;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\IndexingSettings;

/**
 * Read side of the indexing controls (settings `seo` group, cached): the per-type robots default SeoManager falls back
 * to, and the sitemap include / priority / changefreq overrides Sitemaps applies.
 */
final class IndexingRules
{
    public function __construct(private readonly SettingsRepository $settings) {}

    public function settings(): IndexingSettings
    {
        return $this->settings->all()->seo->indexing;
    }

    /** Robots default of the page type behind a route (null = the site default index,follow). */
    public function robotsForRoute(?string $routeName): ?Robots
    {
        $type = IndexingType::forRoute($routeName);

        return $type === null ? null : $this->robotsForType($type);
    }

    public function robotsForType(IndexingType $type): ?Robots
    {
        $value = $this->settings()->robotsTypes[$type->value] ?? null;

        return is_string($value) && array_key_exists($value, IndexingType::ROBOTS_OPTIONS) ? Robots::parse($value) : null;
    }

    /** A sitemap file family left out of the index: excluded explicitly, or its type defaults to noindex. */
    public function excludesSitemap(string $key): bool
    {
        if (in_array($key, $this->settings()->sitemapExclude, true)) {
            return true;
        }

        $type = IndexingType::tryFrom($key);
        $robots = $type === null ? null : $this->robotsForType($type);

        return $robots !== null && ! $robots->index;
    }

    public function sitemapPriority(string $key): ?float
    {
        return $this->settings()->sitemapPriorities[$key] ?? null;
    }

    public function sitemapChangefreq(string $key): ?string
    {
        $value = $this->settings()->sitemapChangefreq[$key] ?? null;

        return $value !== null && array_key_exists($value, IndexingType::CHANGEFREQ_OPTIONS) ? $value : null;
    }
}
