<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Crawl;

use DOMDocument;
use DOMElement;
use DOMNode;
use DOMXPath;

/**
 * What the crawl-level checks need from one rendered HTML page, read in a single DOM pass: head tags, link targets,
 * element ids (fragment targets), images, JSON-LD blocks, the resources the browser would request and the <main>
 * content (for thin-content / analyser checks).
 */
final readonly class PageFacts
{
    /**
     * @param  list<string>  $canonicals
     * @param  list<string>  $links  raw href values of <a>/<area>
     * @param  array<string, true>  $ids  element ids and <a name> anchors
     * @param  list<array{src: string, alt: ?string, width: ?string, height: ?string, loading: string, priority: string, inMain: bool}>  $images
     * @param  list<string>  $jsonLd  raw JSON of every application/ld+json script
     * @param  list<string>  $resources  URLs the browser fetches while loading the page (scripts, styles, preloads, icons, images)
     */
    public function __construct(
        public ?string $title,
        public ?string $description,
        public string $robots,
        public array $canonicals,
        public ?string $heading,
        public ?string $ogImage,
        public array $links,
        public array $ids,
        public array $images,
        public array $jsonLd,
        public array $resources,
        public string $mainHtml,
        public int $mainWords,
        public int $htmlBytes,
    ) {}

    public static function fromHtml(string $html): self
    {
        $document = new DOMDocument;
        $previous = libxml_use_internal_errors(true);
        // The XML declaration makes libxml read the markup as UTF-8 (Persian text).
        $document->loadHTML('<?xml encoding="UTF-8">'.$html, LIBXML_NONET | LIBXML_NOERROR | LIBXML_NOWARNING);
        libxml_clear_errors();
        libxml_use_internal_errors($previous);
        $xpath = new DOMXPath($document);

        $main = self::first($xpath, '//main') ?? self::first($xpath, '//body');
        $mainHtml = '';
        if ($main !== null) {
            foreach ($main->childNodes as $child) {
                $mainHtml .= (string) $document->saveHTML($child);
            }
        }
        $mainText = $main === null ? '' : self::text($main);

        $ids = [];
        foreach (self::elements($xpath, '//*[@id] | //a[@name]') as $element) {
            $id = $element->hasAttribute('id') ? $element->getAttribute('id') : $element->getAttribute('name');
            if ($id !== '') {
                $ids[$id] = true;
            }
        }

        $links = [];
        foreach (self::elements($xpath, '//a[@href] | //area[@href]') as $element) {
            $links[] = trim($element->getAttribute('href'));
        }

        $images = [];
        $resources = [];
        foreach (self::elements($xpath, '//img') as $element) {
            $src = trim($element->getAttribute('src'));
            $images[] = [
                'src' => $src,
                'alt' => $element->hasAttribute('alt') ? $element->getAttribute('alt') : null,
                'width' => $element->hasAttribute('width') ? $element->getAttribute('width') : null,
                'height' => $element->hasAttribute('height') ? $element->getAttribute('height') : null,
                'loading' => strtolower(trim($element->getAttribute('loading'))),
                'priority' => strtolower(trim($element->getAttribute('fetchpriority'))),
                'inMain' => $main !== null && self::contains($main, $element),
            ];
            if ($src !== '' && strtolower(trim($element->getAttribute('loading'))) !== 'lazy') {
                $resources[] = $src;
            }
        }
        foreach (self::elements($xpath, '//script[@src] | //iframe[@src] | //embed[@src] | //video[@src] | //audio[@src]') as $element) {
            $resources[] = trim($element->getAttribute('src'));
        }
        foreach (self::elements($xpath, '//video[@poster]') as $element) {
            $resources[] = trim($element->getAttribute('poster'));
        }
        foreach (self::elements($xpath, '//link[@href]') as $element) {
            $rel = ' '.strtolower((string) preg_replace('/\s+/', ' ', $element->getAttribute('rel'))).' ';
            foreach ([' stylesheet ', ' preload ', ' modulepreload ', ' icon ', ' manifest ', ' apple-touch-icon '] as $fetched) {
                if (str_contains($rel, $fetched)) {
                    $resources[] = trim($element->getAttribute('href'));
                    break;
                }
            }
        }

        $jsonLd = [];
        foreach (self::elements($xpath, '//script[@type="application/ld+json"]') as $element) {
            $jsonLd[] = (string) $element->textContent;
        }

        $headings = self::elements($xpath, '//h1');

        return new self(
            title: ($t = self::first($xpath, '//head/title')) === null ? null : self::text($t),
            description: self::meta($xpath, 'name', 'description'),
            robots: strtolower(self::meta($xpath, 'name', 'robots') ?? ''),
            canonicals: array_map(static fn (DOMElement $e): string => trim($e->getAttribute('href')), self::elements($xpath, '//link[@rel="canonical"]')),
            heading: $headings === [] ? null : self::text($headings[0]),
            ogImage: self::meta($xpath, 'property', 'og:image'),
            links: $links,
            ids: $ids,
            images: $images,
            jsonLd: $jsonLd,
            resources: array_values(array_unique(array_filter($resources, static fn (string $u): bool => $u !== '' && ! str_starts_with($u, 'data:')))),
            mainHtml: $mainHtml,
            mainWords: $mainText === '' ? 0 : count(preg_split('/\s+/u', $mainText) ?: []),
            htmlBytes: strlen($html),
        );
    }

    public function indexable(): bool
    {
        return ! str_contains($this->robots, 'noindex') && ! str_contains($this->robots, 'none');
    }

    /**
     * @return list<DOMElement>
     */
    private static function elements(DOMXPath $xpath, string $query): array
    {
        $nodes = $xpath->query($query);
        $elements = [];
        foreach ($nodes === false ? [] : $nodes as $node) {
            if ($node instanceof DOMElement) {
                $elements[] = $node;
            }
        }

        return $elements;
    }

    private static function first(DOMXPath $xpath, string $query): ?DOMElement
    {
        return self::elements($xpath, $query)[0] ?? null;
    }

    private static function meta(DOMXPath $xpath, string $attribute, string $name): ?string
    {
        $element = self::first($xpath, "//meta[@{$attribute}=\"{$name}\"]");

        return $element === null ? null : trim($element->getAttribute('content'));
    }

    private static function text(DOMNode $node): string
    {
        return trim((string) preg_replace('/\s+/u', ' ', $node->textContent));
    }

    private static function contains(DOMNode $ancestor, DOMNode $node): bool
    {
        for ($current = $node->parentNode; $current !== null; $current = $current->parentNode) {
            if ($current->isSameNode($ancestor)) {
                return true;
            }
        }

        return false;
    }
}
