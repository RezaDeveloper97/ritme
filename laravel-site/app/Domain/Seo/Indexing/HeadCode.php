<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing;

use DOMDocument;
use DOMElement;
use DOMNode;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * "Custom head code" (super-admin only). The public site makes zero external requests and runs a strict CSP without
 * inline scripts (tasks/README.md, L1-07), so this is NOT a free-form HTML hole: only `<meta>` tags (name / property /
 * itemprop + content — e.g. extra verification codes) and same-origin `<link>` tags survive. Scripts, styles, iframes,
 * event handlers, `http-equiv`, external or `javascript:` URLs are rejected on save (errors()) and stripped again on
 * render (html()), which rebuilds the markup from the parsed allow-list — the stored string is never echoed as is.
 */
final class HeadCode
{
    public const MAX_BYTES = 8_192;

    public const MAX_TAGS = 30;

    private const META_ATTRIBUTES = ['name', 'property', 'itemprop', 'content'];

    private const LINK_ATTRIBUTES = ['rel', 'href', 'type', 'sizes', 'media', 'title', 'color'];

    private const FORBIDDEN_RELS = ['stylesheet', 'import', 'prerender'];

    public function __construct(
        private readonly IndexingRules $indexing,
        private readonly Config $config,
    ) {}

    /** The sanitised tags for `<x-seo.head/>` ('' when none). */
    public function html(): string
    {
        $code = $this->indexing->settings()->headCode;

        return $code === null ? '' : $this->sanitize($code);
    }

    /**
     * @return list<string> Persian validation errors (empty = valid)
     */
    public function errors(string $html): array
    {
        $html = trim($html);
        if ($html === '') {
            return [];
        }
        if (strlen($html) > self::MAX_BYTES) {
            return ['کد سفارشی نباید بیشتر از ۸ کیلوبایت باشد.'];
        }

        $errors = [];
        $tags = 0;
        foreach ($this->nodes($html) as $node) {
            if (! $node instanceof DOMElement) {
                if (trim((string) $node->textContent) !== '' && $node->nodeType !== XML_COMMENT_NODE) {
                    $errors[] = 'متن آزاد مجاز نیست؛ فقط تگ‌های <meta> و <link>.';
                }

                continue;
            }

            $tags++;
            $tag = strtolower($node->tagName);
            if (! in_array($tag, ['meta', 'link'], true)) {
                $errors[] = "تگ <{$tag}> مجاز نیست. اسکریپت، استایل و iframe بیرونی امتیاز GTmetrix را کم می‌کنند و CSP سخت‌گیرانه سایت آن‌ها را مسدود می‌کند؛ فقط <meta> و <link> هم‌دامنه.";

                continue;
            }

            $allowed = $tag === 'meta' ? self::META_ATTRIBUTES : self::LINK_ATTRIBUTES;
            foreach ($node->attributes ?? [] as $attribute) {
                $name = strtolower($attribute->nodeName);
                if (! in_array($name, $allowed, true)) {
                    $errors[] = "ویژگی «{$name}» روی <{$tag}> مجاز نیست.";
                }
            }

            if ($tag === 'link') {
                $rel = strtolower(trim($node->getAttribute('rel')));
                $href = trim($node->getAttribute('href'));
                if ($rel === '' || $href === '') {
                    $errors[] = 'تگ <link> باید rel و href داشته باشد.';
                } elseif (array_intersect(preg_split('/\s+/', $rel) ?: [], self::FORBIDDEN_RELS) !== []) {
                    $errors[] = "rel=\"{$rel}\" مجاز نیست (استایل‌ها از قالب سایت می‌آیند).";
                } elseif (! $this->sameOrigin($href)) {
                    $errors[] = "نشانی «{$href}» باید روی همین دامنه باشد؛ سایت هیچ درخواست بیرونی نمی‌فرستد.";
                }
            }
        }

        if ($tags > self::MAX_TAGS) {
            $errors[] = 'حداکثر '.self::MAX_TAGS.' تگ مجاز است.';
        }

        return array_values(array_unique($errors));
    }

    /** Rebuilds only the allowed tags and attributes, escaped. */
    public function sanitize(string $html): string
    {
        $out = [];
        foreach ($this->nodes(trim($html)) as $node) {
            if (! $node instanceof DOMElement || count($out) >= self::MAX_TAGS) {
                continue;
            }
            $tag = strtolower($node->tagName);
            if ($tag === 'meta') {
                $attributes = $this->attributes($node, self::META_ATTRIBUTES);
                if (! isset($attributes['content']) || array_intersect_key($attributes, array_flip(['name', 'property', 'itemprop'])) === []) {
                    continue;
                }
            } elseif ($tag === 'link') {
                $attributes = $this->attributes($node, self::LINK_ATTRIBUTES);
                $rel = strtolower($attributes['rel'] ?? '');
                $href = $attributes['href'] ?? '';
                if ($rel === '' || $href === '' || ! $this->sameOrigin($href)
                    || array_intersect(preg_split('/\s+/', $rel) ?: [], self::FORBIDDEN_RELS) !== []) {
                    continue;
                }
            } else {
                continue;
            }

            $parts = [];
            foreach ($attributes as $name => $value) {
                $parts[] = $name.'="'.htmlspecialchars($value, ENT_QUOTES | ENT_SUBSTITUTE | ENT_HTML5, 'UTF-8').'"';
            }
            $out[] = '<'.$tag.' '.implode(' ', $parts).'>';
        }

        return implode("\n", $out);
    }

    /**
     * @param  list<string>  $allowed
     * @return array<string, string>
     */
    private function attributes(DOMElement $node, array $allowed): array
    {
        $attributes = [];
        foreach ($allowed as $name) {
            if ($node->hasAttribute($name)) {
                $attributes[$name] = trim($node->getAttribute($name));
            }
        }

        return $attributes;
    }

    /** Relative path (`/…`, not `//host`) or an absolute http(s) URL on the app.url host. */
    private function sameOrigin(string $url): bool
    {
        if (preg_match('/^\/(?!\/)/', $url) === 1) {
            return true;
        }

        $parts = parse_url($url);
        $appHost = parse_url((string) $this->config->get('app.url'), PHP_URL_HOST);

        return is_array($parts)
            && in_array(strtolower((string) ($parts['scheme'] ?? '')), ['http', 'https'], true)
            && is_string($appHost)
            && strcasecmp((string) ($parts['host'] ?? ''), $appHost) === 0;
    }

    /**
     * Top-level nodes of the fragment as parsed by an HTML5-ish parser: everything libxml moved to <head> and <body>.
     *
     * @return list<DOMNode>
     */
    private function nodes(string $html): array
    {
        if ($html === '') {
            return [];
        }

        $document = new DOMDocument;
        $previous = libxml_use_internal_errors(true);
        $document->loadHTML('<!DOCTYPE html><html><head><meta charset="utf-8">'.$html.'</head><body></body></html>', LIBXML_NONET);
        libxml_clear_errors();
        libxml_use_internal_errors($previous);

        $nodes = [];
        foreach (['head', 'body'] as $section) {
            $parent = $document->getElementsByTagName($section)->item(0);
            foreach ($parent === null ? [] : $parent->childNodes as $index => $node) {
                if ($section === 'head' && $index === 0) {
                    continue; // our charset meta
                }
                $nodes[] = $node;
            }
        }

        return $nodes;
    }
}
