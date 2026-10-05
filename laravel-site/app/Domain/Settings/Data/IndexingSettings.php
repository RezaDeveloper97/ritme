<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

/**
 * Indexing controls (L7-04), stored as flat keys of the `seo` settings group and exposed as SeoDefaults::$indexing.
 * Values are only coerced here; the admin form validates them (App\Domain\Seo\Indexing) before they are saved.
 *
 *  - robots_txt          custom production crawl rules (null = the built-in DefaultRobotsRules)
 *  - robots_types        sitemap key => robots meta default of that content type ('noindex,follow' …)
 *  - sitemap_exclude     sitemap keys left out of /sitemap.xml
 *  - sitemap_priorities  sitemap key => priority 0.0–1.0 overriding the provider's value
 *  - sitemap_changefreq  sitemap key => changefreq overriding the provider's value
 *  - sitemap_ping_urls   endpoints pinged (server side, queued) after "regenerate sitemap"; `{sitemap}` = index URL
 *  - indexnow_enabled    submit published/updated URLs to IndexNow (server side, queued); off by default
 *  - indexnow_key        the IndexNow key, served at /{key}.txt while enabled
 *  - head_code           extra <meta>/<link> tags for the public <head> (super-admin only, sanitised; never scripts)
 */
final readonly class IndexingSettings
{
    use CoercesSettingValues;

    public const KEYS = [
        'robots_txt', 'robots_types', 'sitemap_exclude', 'sitemap_priorities', 'sitemap_changefreq',
        'sitemap_ping_urls', 'indexnow_enabled', 'indexnow_key', 'head_code',
    ];

    /**
     * @param  array<string, string>  $robotsTypes
     * @param  list<string>  $sitemapExclude
     * @param  array<string, float>  $sitemapPriorities
     * @param  array<string, string>  $sitemapChangefreq
     * @param  list<string>  $sitemapPingUrls
     */
    public function __construct(
        public ?string $robotsTxt = null,
        public array $robotsTypes = [],
        public array $sitemapExclude = [],
        public array $sitemapPriorities = [],
        public array $sitemapChangefreq = [],
        public array $sitemapPingUrls = [],
        public bool $indexNowEnabled = false,
        public ?string $indexNowKey = null,
        public ?string $headCode = null,
    ) {}

    /**
     * @param  array<string, mixed>  $values
     */
    public static function fromArray(array $values): self
    {
        $priorities = [];
        foreach (self::stringMap($values, 'sitemap_priorities') as $key => $priority) {
            if (is_numeric($priority)) {
                $priorities[$key] = round(min(1.0, max(0.0, (float) $priority)), 1);
            }
        }

        $robotsTxt = $values['robots_txt'] ?? null;
        $robotsTxt = is_string($robotsTxt) ? str_replace("\r\n", "\n", trim($robotsTxt)) : '';
        $headCode = $values['head_code'] ?? null;
        $headCode = is_string($headCode) ? trim($headCode) : '';

        return new self(
            robotsTxt: $robotsTxt === '' ? null : $robotsTxt,
            robotsTypes: self::stringMap($values, 'robots_types'),
            sitemapExclude: self::stringList($values, 'sitemap_exclude'),
            sitemapPriorities: $priorities,
            sitemapChangefreq: self::stringMap($values, 'sitemap_changefreq'),
            sitemapPingUrls: self::stringList($values, 'sitemap_ping_urls'),
            indexNowEnabled: self::bool($values, 'indexnow_enabled'),
            indexNowKey: self::nullableString($values, 'indexnow_key'),
            headCode: $headCode === '' ? null : $headCode,
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'robots_txt' => $this->robotsTxt,
            'robots_types' => $this->robotsTypes,
            'sitemap_exclude' => $this->sitemapExclude,
            'sitemap_priorities' => $this->sitemapPriorities,
            'sitemap_changefreq' => $this->sitemapChangefreq,
            'sitemap_ping_urls' => $this->sitemapPingUrls,
            'indexnow_enabled' => $this->indexNowEnabled,
            'indexnow_key' => $this->indexNowKey,
            'head_code' => $this->headCode,
        ];
    }
}
