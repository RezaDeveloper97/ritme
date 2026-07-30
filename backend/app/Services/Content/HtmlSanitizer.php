<?php

namespace App\Services\Content;

use DOMComment;
use DOMDocument;
use DOMElement;
use DOMNode;

/**
 * Cleans the rich text an admin authors in CKEditor before it is handed to a
 * client that renders it as HTML (the React article page uses
 * dangerouslySetInnerHTML — there is no second line of defence there).
 *
 * The editor's own output is already tame; this exists so a compromised or
 * careless admin account cannot turn an article into stored XSS. Anything not
 * on the allowlist is unwrapped (its text survives, the tag does not), and the
 * few tags that carry executable payloads are dropped whole.
 */
class HtmlSanitizer
{
    /** Tags whose content is markup we keep, mapped to the attributes they may carry. */
    private const ALLOWED = [
        'p' => [], 'br' => [], 'strong' => [], 'b' => [], 'em' => [], 'i' => [],
        'u' => [], 's' => [], 'mark' => [], 'sub' => [], 'sup' => [], 'span' => [],
        'h1' => [], 'h2' => [], 'h3' => [], 'h4' => [], 'h5' => [], 'h6' => [],
        'ul' => [], 'ol' => [], 'li' => [], 'blockquote' => [], 'pre' => [], 'code' => [],
        'hr' => [], 'figure' => [], 'figcaption' => [],
        'table' => [], 'thead' => [], 'tbody' => [], 'tfoot' => [], 'tr' => [],
        'th' => ['colspan', 'rowspan'], 'td' => ['colspan', 'rowspan'],
        'a' => ['href', 'title', 'target', 'rel'],
        'img' => ['src', 'alt', 'width', 'height'],
    ];

    /** Tags removed with everything inside them — their content is never body copy. */
    private const DROPPED = ['script', 'style', 'iframe', 'object', 'embed', 'form', 'input', 'button', 'link', 'meta', 'svg'];

    /** URL schemes a link or image may point at. */
    private const ALLOWED_SCHEMES = ['http', 'https', 'mailto', 'tel'];

    /**
     * Sanitized HTML, or null when nothing renderable is left.
     */
    public static function clean(mixed $html): ?string
    {
        if (! is_string($html) || trim($html) === '') {
            return null;
        }

        $document = new DOMDocument('1.0', 'UTF-8');
        $previous = libxml_use_internal_errors(true);

        // The charset meta is what keeps Persian text from being mangled: libxml
        // assumes ISO-8859-1 for a fragment that doesn't declare an encoding.
        $document->loadHTML(
            '<html><head><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head><body>'.$html.'</body></html>'
        );

        libxml_clear_errors();
        libxml_use_internal_errors($previous);

        $body = $document->getElementsByTagName('body')->item(0);
        if ($body === null) {
            return null;
        }

        self::cleanChildren($body);

        $output = '';
        foreach ($body->childNodes as $child) {
            $output .= $document->saveHTML($child);
        }

        $output = trim($output);

        return $output === '' ? null : $output;
    }

    /**
     * Readable one-line text for a rich-text value — what a card summary or a
     * meta description needs. Clients would otherwise print the markup verbatim.
     */
    public static function toPlainText(mixed $html): ?string
    {
        if (! is_string($html) || $html === '') {
            return null;
        }

        // Block ends become spaces first, so "<p>a</p><p>b</p>" doesn't read "ab".
        $spaced = preg_replace('/<(br|\/p|\/div|\/li|\/h[1-6])[^>]*>/i', ' ', $html) ?? $html;
        $text = html_entity_decode(strip_tags($spaced), ENT_QUOTES | ENT_HTML5, 'UTF-8');
        $text = trim(preg_replace('/\s+/u', ' ', $text) ?? $text);

        return $text === '' ? null : $text;
    }

    private static function cleanChildren(DOMNode $node): void
    {
        // Snapshot the list: removing/unwrapping mutates it while we walk.
        foreach (iterator_to_array($node->childNodes) as $child) {
            if ($child instanceof DOMComment) {
                $child->parentNode?->removeChild($child);

                continue;
            }

            if (! $child instanceof DOMElement) {
                continue;
            }

            $tag = strtolower($child->nodeName);

            if (in_array($tag, self::DROPPED, true)) {
                $child->parentNode?->removeChild($child);

                continue;
            }

            if (! array_key_exists($tag, self::ALLOWED)) {
                self::unwrap($child);

                continue;
            }

            self::cleanAttributes($child, $tag);
            self::cleanChildren($child);
        }
    }

    /** Replace an element with its (already sanitized) children. */
    private static function unwrap(DOMElement $element): void
    {
        self::cleanChildren($element);

        $parent = $element->parentNode;
        if ($parent === null) {
            return;
        }

        while ($element->firstChild !== null) {
            $parent->insertBefore($element->firstChild, $element);
        }

        $parent->removeChild($element);
    }

    private static function cleanAttributes(DOMElement $element, string $tag): void
    {
        $allowed = self::ALLOWED[$tag];

        foreach (iterator_to_array($element->attributes) as $attribute) {
            $name = strtolower($attribute->nodeName);

            // Drops every on* handler and any styling/data attribute along with
            // it — the app's own stylesheet owns how article body copy looks.
            if (! in_array($name, $allowed, true)) {
                $element->removeAttribute($attribute->nodeName);

                continue;
            }

            if (($name === 'href' || $name === 'src') && ! self::isSafeUrl($attribute->nodeValue ?? '')) {
                $element->removeAttribute($attribute->nodeName);
            }
        }

        // A link that opens a new tab must not hand the opener to the target.
        if ($tag === 'a' && $element->getAttribute('target') !== '') {
            $element->setAttribute('rel', 'noopener noreferrer');
        }
    }

    /** Relative and anchor URLs pass; absolute ones must use a known scheme. */
    private static function isSafeUrl(string $url): bool
    {
        $url = trim($url);

        if ($url === '') {
            return false;
        }

        if (str_starts_with($url, '/') || str_starts_with($url, '#') || str_starts_with($url, '?')) {
            return true;
        }

        // parse_url returns false on a malformed URL ("java\nscript:…") — treat
        // that as "no scheme I can vouch for" rather than trusting it.
        $scheme = parse_url($url, PHP_URL_SCHEME) ?: null;

        return $scheme === null
            ? ! str_contains($url, ':')            // a bare relative path
            : in_array(strtolower($scheme), self::ALLOWED_SCHEMES, true);
    }
}
