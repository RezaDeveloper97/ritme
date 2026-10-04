<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use App\Domain\Media\Exceptions\InvalidMediaException;
use DOMAttr;
use DOMDocument;
use DOMElement;
use DOMXPath;
use enshrined\svgSanitize\Sanitizer;

/**
 * Strips scripts, event handlers, foreignObject, external references etc. (enshrined/svg-sanitize). An SVG that
 * cannot be parsed or sanitised is rejected; the sanitised markup is what gets stored.
 */
final class SvgSanitizer
{
    /**
     * @return array{svg: string, width: int|null, height: int|null}
     */
    public function sanitize(string $markup): array
    {
        $sanitizer = new Sanitizer;
        $sanitizer->removeRemoteReferences(true);
        $sanitizer->removeXMLTag(true);
        $sanitizer->minify(true);

        $clean = $sanitizer->sanitize($markup);

        if (! is_string($clean) || trim($clean) === '' || preg_match('/<svg[\s>]/i', $clean) !== 1) {
            throw InvalidMediaException::unsafeSvg();
        }

        $clean = $this->dropExternalReferences($clean);

        return ['svg' => $clean, ...$this->dimensions($clean)];
    }

    /**
     * The library keeps some remote hrefs (e.g. on <image>); public pages make no external requests, so every
     * href that is not a same-document fragment and every url(...) pointing off-document is removed.
     */
    private function dropExternalReferences(string $svg): string
    {
        $dom = new DOMDocument;
        if (! @$dom->loadXML($svg, LIBXML_NONET)) {
            throw InvalidMediaException::unsafeSvg();
        }

        $xpath = new DOMXPath($dom);
        foreach ($xpath->query('//@*') ?: [] as $attribute) {
            if (! $attribute instanceof DOMAttr || ! $attribute->ownerElement instanceof DOMElement) {
                continue;
            }
            $value = trim($attribute->value);
            $isHref = in_array(strtolower($attribute->localName), ['href', 'src'], true);
            $external = $isHref
                ? ! str_starts_with($value, '#')
                : preg_match('/url\(\s*[\'"]?\s*(?!#)/i', $value) === 1;

            if ($external) {
                $attribute->ownerElement->removeAttributeNode($attribute);
            }
        }

        $root = $dom->documentElement;
        if ($root === null) {
            throw InvalidMediaException::unsafeSvg();
        }

        return (string) $dom->saveXML($root);
    }

    /**
     * @return array{width: int|null, height: int|null}
     */
    private function dimensions(string $svg): array
    {
        $root = @simplexml_load_string($svg);
        if ($root === false) {
            return ['width' => null, 'height' => null];
        }

        $attributes = $root->attributes();
        $width = $this->length((string) ($attributes['width'] ?? ''));
        $height = $this->length((string) ($attributes['height'] ?? ''));

        if (($width === null || $height === null) && preg_match('/^\s*[-\d.]+[\s,]+[-\d.]+[\s,]+([\d.]+)[\s,]+([\d.]+)\s*$/', (string) ($attributes['viewBox'] ?? ''), $m) === 1) {
            $width = (int) round((float) $m[1]);
            $height = (int) round((float) $m[2]);
        }

        return ['width' => $width ?: null, 'height' => $height ?: null];
    }

    private function length(string $value): ?int
    {
        return preg_match('/^\s*([\d.]+)\s*(px)?\s*$/', $value, $m) === 1 ? (int) round((float) $m[1]) : null;
    }
}
