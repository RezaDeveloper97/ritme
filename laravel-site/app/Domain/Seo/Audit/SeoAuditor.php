<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

use DOMDocument;
use DOMElement;
use DOMXPath;

/**
 * seo:audit v1 rules on rendered HTML (extended into a full engine in L7-05):
 * title 30–60 chars and unique, description 70–160 and unique (uniqueness among indexable pages), exactly one
 * <h1>, canonical present + absolute, Open Graph complete, every <img> with alt + width + height, no href="#".
 * A missing og:image is a warning until the media library provides default OG images.
 */
final class SeoAuditor
{
    public const TITLE_MIN = 30;

    public const TITLE_MAX = 60;

    public const DESCRIPTION_MIN = 70;

    public const DESCRIPTION_MAX = 160;

    public const REQUIRED_OG = ['og:locale', 'og:site_name', 'og:type', 'og:url', 'og:title', 'og:description'];

    public const IMAGE_OG = ['og:image:width', 'og:image:height', 'og:image:alt'];

    public function auditPage(string $url, string $html): PageAudit
    {
        $xpath = $this->xpath($html);
        $issues = [];

        $titles = $this->texts($xpath, '//head/title');
        $title = $titles[0] ?? null;
        if ($title === null || $title === '') {
            $issues[] = AuditIssue::error('title.missing', 'No <title>.');
        } else {
            if (count($titles) > 1) {
                $issues[] = AuditIssue::error('title.multiple', count($titles).' <title> elements.');
            }
            $length = mb_strlen($title);
            if ($length < self::TITLE_MIN || $length > self::TITLE_MAX) {
                $issues[] = AuditIssue::error('title.length', "Title is {$length} chars (".self::TITLE_MIN.'–'.self::TITLE_MAX.').');
            }
        }

        $description = $this->metaContent($xpath, 'name', 'description');
        if ($description === null || $description === '') {
            $issues[] = AuditIssue::error('description.missing', 'No meta description.');
        } else {
            $length = mb_strlen($description);
            if ($length < self::DESCRIPTION_MIN || $length > self::DESCRIPTION_MAX) {
                $issues[] = AuditIssue::error('description.length', "Description is {$length} chars (".self::DESCRIPTION_MIN.'–'.self::DESCRIPTION_MAX.').');
            }
        }

        $h1 = $xpath->query('//h1');
        $h1Count = $h1 === false ? 0 : $h1->length;
        if ($h1Count !== 1) {
            $issues[] = AuditIssue::error('h1.count', "{$h1Count} <h1> elements (exactly one expected).");
        }

        $canonicals = $this->attributes($xpath, '//link[@rel="canonical"]', 'href');
        if ($canonicals === []) {
            $issues[] = AuditIssue::error('canonical.missing', 'No canonical link.');
        } elseif (count($canonicals) > 1) {
            $issues[] = AuditIssue::error('canonical.multiple', count($canonicals).' canonical links.');
        } elseif (preg_match('#^https?://[^/\s]+#i', $canonicals[0]) !== 1) {
            $issues[] = AuditIssue::error('canonical.relative', "Canonical is not absolute: {$canonicals[0]}");
        }

        foreach (self::REQUIRED_OG as $property) {
            if (($this->metaContent($xpath, 'property', $property) ?? '') === '') {
                $issues[] = AuditIssue::error('og.missing', "Missing {$property}.");
            }
        }
        if (($this->metaContent($xpath, 'property', 'og:image') ?? '') === '') {
            $issues[] = AuditIssue::warning('og.image', 'Missing og:image.');
        } else {
            foreach (self::IMAGE_OG as $property) {
                if (($this->metaContent($xpath, 'property', $property) ?? '') === '') {
                    $issues[] = AuditIssue::error('og.missing', "Missing {$property}.");
                }
            }
        }

        $images = $xpath->query('//img');
        foreach ($images === false ? [] : $images as $index => $image) {
            if (! $image instanceof DOMElement) {
                continue;
            }
            $missing = array_values(array_filter(['alt', 'width', 'height'], static fn (string $a): bool => ! $image->hasAttribute($a)));
            if ($missing !== []) {
                $src = $image->getAttribute('src') ?: '#'.($index + 1);
                $issues[] = AuditIssue::error('img.attributes', "<img {$src}> lacks ".implode(', ', $missing).'.');
            }
        }

        $hashLinks = $xpath->query('//a[normalize-space(@href)="#"]');
        $hashCount = $hashLinks === false ? 0 : $hashLinks->length;
        if ($hashCount > 0) {
            $issues[] = AuditIssue::error('link.hash', "{$hashCount} link(s) with href=\"#\".");
        }

        $robots = strtolower($this->metaContent($xpath, 'name', 'robots') ?? '');

        return new PageAudit(
            url: $url,
            title: $title,
            description: $description,
            indexable: ! str_contains($robots, 'noindex') && ! str_contains($robots, 'none'),
            issues: $issues,
        );
    }

    /**
     * Adds duplicate title / description errors across the indexable pages.
     *
     * @param  list<PageAudit>  $pages
     * @return list<PageAudit>
     */
    public function crossCheck(array $pages): array
    {
        foreach (['title' => 'title.duplicate', 'description' => 'description.duplicate'] as $field => $code) {
            $seen = [];
            foreach ($pages as $page) {
                $value = $page->{$field};
                if ($page->indexable && is_string($value) && $value !== '') {
                    $seen[$value][] = $page->url;
                }
            }
            foreach ($pages as $i => $page) {
                $value = $page->{$field};
                if (! $page->indexable || ! is_string($value) || count($seen[$value] ?? []) < 2) {
                    continue;
                }
                $others = array_values(array_diff($seen[$value], [$page->url]));
                $pages[$i] = $page->with(AuditIssue::error($code, ucfirst($field).' also used by '.implode(', ', $others).'.'));
            }
        }

        return $pages;
    }

    private function xpath(string $html): DOMXPath
    {
        $document = new DOMDocument;
        $previous = libxml_use_internal_errors(true);
        // The XML declaration makes libxml read the markup as UTF-8 (Persian text).
        $document->loadHTML('<?xml encoding="UTF-8">'.$html, LIBXML_NONET | LIBXML_NOERROR | LIBXML_NOWARNING);
        libxml_clear_errors();
        libxml_use_internal_errors($previous);

        return new DOMXPath($document);
    }

    /**
     * @return list<string>
     */
    private function texts(DOMXPath $xpath, string $query): array
    {
        $nodes = $xpath->query($query);
        $texts = [];
        foreach ($nodes === false ? [] : $nodes as $node) {
            $texts[] = trim((string) preg_replace('/\s+/u', ' ', $node->textContent));
        }

        return $texts;
    }

    /**
     * @return list<string>
     */
    private function attributes(DOMXPath $xpath, string $query, string $attribute): array
    {
        $nodes = $xpath->query($query);
        $values = [];
        foreach ($nodes === false ? [] : $nodes as $node) {
            if ($node instanceof DOMElement) {
                $values[] = trim($node->getAttribute($attribute));
            }
        }

        return $values;
    }

    private function metaContent(DOMXPath $xpath, string $attribute, string $name): ?string
    {
        return $this->attributes($xpath, "//meta[@{$attribute}=\"{$name}\"]", 'content')[0] ?? null;
    }
}
