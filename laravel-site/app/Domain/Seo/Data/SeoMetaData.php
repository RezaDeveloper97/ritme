<?php

declare(strict_types=1);

namespace App\Domain\Seo\Data;

use App\Domain\Seo\Models\SeoMeta;

/**
 * One seo_meta row as a plain DTO (null = inherit). Cached as an array, see CachedSeoMetaRepository.
 */
final readonly class SeoMetaData
{
    /**
     * @param  array<string, mixed>  $schemaOverrides
     */
    public function __construct(
        public ?string $title = null,
        public ?string $description = null,
        public ?string $canonicalUrl = null,
        public ?string $robots = null,
        public ?string $ogTitle = null,
        public ?string $ogDescription = null,
        public ?int $ogMediaId = null,
        public ?string $ogType = null,
        public ?string $twitterCard = null,
        public ?string $focusKeyword = null,
        public array $schemaOverrides = [],
        public bool $sitemapInclude = true,
        public ?float $sitemapPriority = null,
        public ?string $sitemapChangefreq = null,
    ) {}

    public static function fromModel(SeoMeta $meta): self
    {
        return new self(
            title: self::text($meta->title),
            description: self::text($meta->description),
            canonicalUrl: self::text($meta->canonical_url),
            robots: self::text($meta->robots),
            ogTitle: self::text($meta->og_title),
            ogDescription: self::text($meta->og_description),
            ogMediaId: $meta->og_media_id,
            ogType: self::text($meta->og_type),
            twitterCard: self::text($meta->twitter_card),
            focusKeyword: self::text($meta->focus_keyword),
            schemaOverrides: $meta->schema_overrides ?? [],
            sitemapInclude: $meta->sitemap_include,
            sitemapPriority: $meta->sitemap_priority === null ? null : (float) $meta->sitemap_priority,
            sitemapChangefreq: self::text($meta->sitemap_changefreq),
        );
    }

    /**
     * @param  array<string, mixed>  $values
     */
    public static function fromArray(array $values): self
    {
        $string = static fn (string $key): ?string => is_string($values[$key] ?? null) ? self::text($values[$key]) : null;

        return new self(
            title: $string('title'),
            description: $string('description'),
            canonicalUrl: $string('canonical_url'),
            robots: $string('robots'),
            ogTitle: $string('og_title'),
            ogDescription: $string('og_description'),
            ogMediaId: is_int($values['og_media_id'] ?? null) ? $values['og_media_id'] : null,
            ogType: $string('og_type'),
            twitterCard: $string('twitter_card'),
            focusKeyword: $string('focus_keyword'),
            schemaOverrides: is_array($values['schema_overrides'] ?? null) ? $values['schema_overrides'] : [],
            sitemapInclude: (bool) ($values['sitemap_include'] ?? true),
            sitemapPriority: is_numeric($values['sitemap_priority'] ?? null) ? (float) $values['sitemap_priority'] : null,
            sitemapChangefreq: $string('sitemap_changefreq'),
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'title' => $this->title,
            'description' => $this->description,
            'canonical_url' => $this->canonicalUrl,
            'robots' => $this->robots,
            'og_title' => $this->ogTitle,
            'og_description' => $this->ogDescription,
            'og_media_id' => $this->ogMediaId,
            'og_type' => $this->ogType,
            'twitter_card' => $this->twitterCard,
            'focus_keyword' => $this->focusKeyword,
            'schema_overrides' => $this->schemaOverrides,
            'sitemap_include' => $this->sitemapInclude,
            'sitemap_priority' => $this->sitemapPriority,
            'sitemap_changefreq' => $this->sitemapChangefreq,
        ];
    }

    private static function text(?string $value): ?string
    {
        $value = $value === null ? '' : trim($value);

        return $value === '' ? null : $value;
    }
}
