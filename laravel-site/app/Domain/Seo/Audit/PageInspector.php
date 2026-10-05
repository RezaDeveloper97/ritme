<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Seo\Audit\Crawl\FetchResult;
use App\Domain\Seo\Audit\Crawl\PageFacts;
use App\Domain\Seo\Audit\Crawl\SiteUrls;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use JsonException;

/**
 * Single-page rules on top of SeoAuditor that need the request context: canonical on the site's host and pointing at
 * the page itself (indexable pages), JSON-LD parses and carries Google's required properties, LCP image not lazy and
 * lower images lazy, thin content (indexable posts / products / places), HTML weight, request count, third-party requests, response time and a local
 * og:image file. Cross-page rules (links, sitemap, orphans, duplicates) live in SeoAuditEngine.
 */
final class PageInspector
{
    public const HTML_BYTES_WARN = 200_000;

    public const REQUESTS_WARN = 40;

    public const SLOW_NOTICE_MS = 1000;

    public const SLOW_WARN_MS = 2500;

    /** Images in <main> after this many must be lazy (the first ones may be above the fold). */
    public const EAGER_IMAGES = 3;

    /** Google's required properties per @type (rich-result minimum, docs/SEO.md). */
    private const REQUIRED = [
        'BreadcrumbList' => ['itemListElement'],
        'ItemList' => ['itemListElement'],
        'FAQPage' => ['mainEntity'],
        'BlogPosting' => ['headline'],
        'Article' => ['headline'],
        'Organization' => ['name'],
        'WebSite' => ['name', 'url'],
        'MobileApplication' => ['name'],
        'LocalBusiness' => ['name', 'address'],
        'Person' => ['name'],
    ];

    public function __construct(private readonly SiteUrls $urls, private readonly string $publicPath) {}

    /**
     * @return list<AuditIssue>
     */
    public function inspect(FetchResult $fetch, PageFacts $facts, ?ContentType $type, ?int $contentWords = null, bool $productionCanonical = true): array
    {
        return [
            ...$this->canonical($fetch, $facts, $productionCanonical),
            ...$this->jsonLd($facts),
            ...$this->images($facts),
            ...$this->ogImage($facts),
            ...$this->weight($facts),
            ...($facts->indexable() ? $this->content($contentWords ?? $facts->mainWords, $type) : []),
            ...$this->speed($fetch),
        ];
    }

    /**
     * @return list<AuditIssue>
     */
    private function canonical(FetchResult $fetch, PageFacts $facts, bool $production): array
    {
        if (count($facts->canonicals) !== 1) {
            return []; // missing / multiple: SeoAuditor
        }
        $canonical = $facts->canonicals[0];
        $host = strtolower((string) parse_url($canonical, PHP_URL_HOST));
        if ($host === '') {
            return []; // relative: SeoAuditor
        }

        $issues = [];
        if ($host !== $this->urls->host()) {
            $issues[] = AuditIssue::error('canonical.host', "canonical روی دامنه دیگری است: {$canonical}");
        } elseif ($production && strtolower((string) parse_url($canonical, PHP_URL_SCHEME)) !== 'https') {
            $issues[] = AuditIssue::error('canonical.host', "canonical باید https باشد: {$canonical}");
        }

        $resolved = $this->urls->resolve($canonical);
        if ($facts->indexable() && $resolved !== null) {
            $self = $this->urls->resolve($fetch->path);
            $target = rtrim($resolved['path'], '/') ?: '/';
            $own = $self === null ? $fetch->path : (rtrim($self['path'], '/') ?: '/');
            if ($target !== $own || self::page($resolved['query']) !== self::page($self['query'] ?? '')) {
                $issues[] = AuditIssue::error('canonical.self', "صفحه قابل نمایه است ولی canonical آن به {$resolved['path']} اشاره می‌کند.");
            }
        }

        return $issues;
    }

    /**
     * @return list<AuditIssue>
     */
    private function jsonLd(PageFacts $facts): array
    {
        if ($facts->jsonLd === []) {
            return [AuditIssue::warning('jsonld.missing', 'داده ساختاریافته (JSON-LD) ندارد.')];
        }

        $issues = [];
        foreach ($facts->jsonLd as $index => $raw) {
            try {
                $data = json_decode($raw, true, 512, JSON_THROW_ON_ERROR);
            } catch (JsonException $e) {
                $issues[] = AuditIssue::error('jsonld.parse', 'بلوک JSON-LD شماره '.($index + 1).' قابل خواندن نیست: '.$e->getMessage());

                continue;
            }
            if (! is_array($data)) {
                $issues[] = AuditIssue::error('jsonld.parse', 'بلوک JSON-LD شماره '.($index + 1).' شیء نیست.');

                continue;
            }
            if (! isset($data['@context'])) {
                $issues[] = AuditIssue::error('jsonld.required', 'بلوک JSON-LD شماره '.($index + 1).' @context ندارد.');
            }
            $nodes = isset($data['@graph']) && is_array($data['@graph']) ? $data['@graph'] : [$data];
            foreach ($nodes as $node) {
                if (is_array($node)) {
                    $issues = [...$issues, ...$this->requiredProperties($node)];
                }
            }
        }

        return $issues;
    }

    /**
     * @param  array<mixed>  $node
     * @return list<AuditIssue>
     */
    private function requiredProperties(array $node): array
    {
        $types = array_values(array_filter((array) ($node['@type'] ?? []), 'is_string'));
        $issues = [];
        foreach ($types as $type) {
            $required = self::REQUIRED[$type] ?? (LocalBusinessType::tryFrom($type) !== null ? self::REQUIRED['LocalBusiness'] : null);
            if ($type === 'Product') {
                $required = ['name'];
                if (! isset($node['offers']) && ! isset($node['review']) && ! isset($node['aggregateRating'])) {
                    $issues[] = AuditIssue::error('jsonld.required', 'Product باید offers یا review یا aggregateRating داشته باشد.');
                }
            }
            foreach ($required ?? [] as $property) {
                $value = $node[$property] ?? null;
                if ($value === null || $value === '' || $value === []) {
                    $issues[] = AuditIssue::error('jsonld.required', "{$type} ویژگی لازم «{$property}» را ندارد.");
                }
            }
            if ($type === 'BreadcrumbList' && is_array($node['itemListElement'] ?? null)) {
                foreach ($node['itemListElement'] as $position => $item) {
                    if (! is_array($item) || ! isset($item['position'], $item['name'])) {
                        $issues[] = AuditIssue::error('jsonld.required', 'آیتم '.((int) $position + 1).' BreadcrumbList باید position و name داشته باشد.');
                    }
                }
            }
        }

        return $issues;
    }

    /**
     * @return list<AuditIssue>
     */
    private function images(PageFacts $facts): array
    {
        $issues = [];
        $inMain = array_values(array_filter($facts->images, static fn (array $image): bool => $image['inMain']));
        foreach ($facts->images as $image) {
            if ($image['priority'] === 'high' && $image['loading'] === 'lazy') {
                $issues[] = AuditIssue::error('img.lcp-lazy', "تصویر {$image['src']} هم fetchpriority=high دارد و هم loading=lazy.");
            }
        }
        if (isset($inMain[0]) && $inMain[0]['loading'] === 'lazy' && $inMain[0]['priority'] !== 'high') {
            $issues[] = AuditIssue::notice('img.lcp-lazy', "اولین تصویر محتوا ({$inMain[0]['src']}) lazy است؛ اگر بالای صفحه دیده می‌شود، lazy نباشد.");
        }
        $eager = 0;
        foreach (array_slice($inMain, self::EAGER_IMAGES) as $image) {
            if ($image['loading'] !== 'lazy' && $image['priority'] !== 'high') {
                $eager++;
            }
        }
        if ($eager > 0) {
            $issues[] = AuditIssue::notice('img.not-lazy', "{$eager} تصویر پایین صفحه loading=\"lazy\" ندارد.");
        }

        return $issues;
    }

    /**
     * @return list<AuditIssue>
     */
    private function ogImage(PageFacts $facts): array
    {
        if ($facts->ogImage === null || $facts->ogImage === '') {
            return []; // SeoAuditor: og.image
        }
        $resolved = $this->urls->resolve($facts->ogImage);
        if ($resolved === null) {
            return [AuditIssue::warning('og.image-external', "تصویر اشتراک روی دامنه دیگری است: {$facts->ogImage}")];
        }
        $file = $this->publicPath.$resolved['path'];
        if (! str_contains($resolved['path'], '..') && ! is_file($file)) {
            return [AuditIssue::error('og.image-missing', "فایل تصویر اشتراک پیدا نشد: {$resolved['path']}")];
        }

        return [];
    }

    /**
     * @return list<AuditIssue>
     */
    private function weight(PageFacts $facts): array
    {
        $issues = [];
        if ($facts->htmlBytes > self::HTML_BYTES_WARN) {
            $issues[] = AuditIssue::warning('weight.html', 'حجم HTML '.round($facts->htmlBytes / 1024).' کیلوبایت است (سقف '.(self::HTML_BYTES_WARN / 1000).').');
        }
        $external = [];
        foreach ($facts->resources as $resource) {
            if ($this->urls->resolve($resource) === null && preg_match('#^(https?:)?//#i', $resource) === 1) {
                $external[] = $resource;
            }
        }
        if ($external !== []) {
            $issues[] = AuditIssue::error('external.request', 'درخواست به منبع بیرونی: '.implode('، ', array_slice($external, 0, 3)));
        }
        if (count($facts->resources) > self::REQUESTS_WARN) {
            $issues[] = AuditIssue::warning('weight.requests', count($facts->resources).' منبع در HTML درخواست می‌شود (سقف '.self::REQUESTS_WARN.').');
        }

        return $issues;
    }

    /**
     * @return list<AuditIssue>
     */
    private function content(int $words, ?ContentType $type): array
    {
        if (! in_array($type, [ContentType::Post, ContentType::Product, ContentType::Place], true)) {
            return [];
        }
        $minimum = intdiv($type->minWords(), 2);
        if ($words < $minimum) {
            return [AuditIssue::warning('content.thin', "متن اصلی صفحه {$words} واژه است؛ برای «{$type->label()}» دست‌کم {$minimum} واژه پیشنهاد می‌شود.")];
        }

        return [];
    }

    /**
     * @return list<AuditIssue>
     */
    private function speed(FetchResult $fetch): array
    {
        return match (true) {
            $fetch->milliseconds >= self::SLOW_WARN_MS => [AuditIssue::warning('perf.slow', "ساخت صفحه {$fetch->milliseconds} میلی‌ثانیه طول کشید (بدون کش).")],
            $fetch->milliseconds >= self::SLOW_NOTICE_MS => [AuditIssue::notice('perf.slow', "ساخت صفحه {$fetch->milliseconds} میلی‌ثانیه طول کشید (بدون کش).")],
            default => [],
        };
    }

    private static function page(string $query): string
    {
        parse_str($query, $parameters);
        $page = $parameters['page'] ?? '';

        return is_string($page) && ctype_digit($page) && (int) $page > 1 ? $page : '';
    }
}
