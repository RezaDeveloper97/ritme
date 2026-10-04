<?php

declare(strict_types=1);

namespace App\Domain\Content\Support;

use DOMDocument;
use DOMElement;
use DOMNode;
use DOMText;

/**
 * Reduces admin-pasted HTML (the enamad trust-seal snippet) to an allow-list of tags with safe attributes.
 *
 *  - Tags outside the list are unwrapped (their text survives); script/style/iframe/object… are dropped whole.
 *  - Attributes: a → href (http/https only), title, target=_blank (+ rel noopener); img → src, alt, width,
 *    height. Every other attribute (on*, style, class …) is removed.
 *  - Images are kept only when same-origin (relative path): public pages make no external requests, so a
 *    third-party seal image (trustseal.enamad.ir) is dropped and a link without content gets `$fallbackText`.
 */
final class AllowedHtml
{
    private const DROP_WITH_CONTENT = ['script', 'style', 'iframe', 'object', 'embed', 'noscript', 'template', 'svg', 'math', 'form'];

    /**
     * @param  list<string>  $allowedTags
     */
    public static function clean(?string $html, array $allowedTags, string $fallbackText = ''): string
    {
        $html = trim((string) $html);
        if ($html === '' || $allowedTags === []) {
            return '';
        }

        $allowed = array_map(strtolower(...), $allowedTags);
        $dom = new DOMDocument('1.0', 'UTF-8');
        $previous = libxml_use_internal_errors(true);
        $dom->loadHTML('<?xml encoding="UTF-8"><div id="__root">'.$html.'</div>', LIBXML_HTML_NOIMPLIED | LIBXML_HTML_NODEFDTD | LIBXML_NONET);
        libxml_clear_errors();
        libxml_use_internal_errors($previous);

        $root = $dom->getElementById('__root');
        if ($root === null) {
            return '';
        }

        self::walk($root, $allowed, $fallbackText);

        $out = '';
        foreach (iterator_to_array($root->childNodes) as $child) {
            $out .= $dom->saveHTML($child);
        }

        return trim($out);
    }

    /**
     * @param  list<string>  $allowed
     */
    private static function walk(DOMNode $node, array $allowed, string $fallbackText): void
    {
        foreach (iterator_to_array($node->childNodes) as $child) {
            if ($child->nodeType === XML_COMMENT_NODE || $child->nodeType === XML_PI_NODE || $child->nodeType === XML_CDATA_SECTION_NODE) {
                $node->removeChild($child);

                continue;
            }

            if (! $child instanceof DOMElement) {
                continue;
            }

            $tag = strtolower($child->tagName);

            if (in_array($tag, self::DROP_WITH_CONTENT, true)) {
                $node->removeChild($child);

                continue;
            }

            self::walk($child, $allowed, $fallbackText);

            if (! in_array($tag, $allowed, true) || ! self::keepAttributes($child, $tag)) {
                while ($child->firstChild !== null) {
                    $node->insertBefore($child->firstChild, $child);
                }
                $node->removeChild($child);

                continue;
            }

            if ($tag === 'a' && trim($child->textContent) === '' && $child->getElementsByTagName('img')->length === 0 && $fallbackText !== '') {
                $child->appendChild(new DOMText($fallbackText));
            }
        }
    }

    /**
     * Strips the element to its safe attributes; false when the element must go (unsafe link, external image).
     */
    private static function keepAttributes(DOMElement $element, string $tag): bool
    {
        $keep = match ($tag) {
            'a' => ['href', 'title', 'target'],
            'img' => ['src', 'alt', 'width', 'height'],
            default => [],
        };

        foreach (iterator_to_array($element->attributes ?? []) as $attribute) {
            if (! in_array(strtolower($attribute->nodeName), $keep, true)) {
                $element->removeAttribute($attribute->nodeName);
            }
        }

        if ($tag === 'a') {
            $href = trim($element->getAttribute('href'));
            if (preg_match('#^https?://#i', $href) !== 1) {
                return false;
            }
            if ($element->hasAttribute('target')) {
                $element->setAttribute('target', '_blank');
            }
            $element->setAttribute('rel', 'noopener nofollow');
        }

        if ($tag === 'img') {
            $src = trim($element->getAttribute('src'));
            if ($src === '' || preg_match('#^(/(?!/)|[a-z0-9._-]+/)#i', $src) !== 1) {
                return false; // external, protocol-relative, data: or javascript: sources
            }
            foreach (['width', 'height'] as $dimension) {
                if ($element->hasAttribute($dimension) && preg_match('/^\d{1,4}$/', $element->getAttribute($dimension)) !== 1) {
                    $element->removeAttribute($dimension);
                }
            }
            if (! $element->hasAttribute('alt')) {
                $element->setAttribute('alt', '');
            }
        }

        return true;
    }
}
