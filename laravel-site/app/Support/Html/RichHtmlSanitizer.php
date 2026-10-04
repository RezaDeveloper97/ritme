<?php

declare(strict_types=1);

namespace App\Support\Html;

use DOMElement;
use Symfony\Component\HtmlSanitizer\HtmlSanitizer;
use Symfony\Component\HtmlSanitizer\HtmlSanitizerConfig;

/**
 * Allow-list sanitiser for editor-authored rich text (article bodies), built on symfony/html-sanitizer.
 *
 *  - Kept: h2–h4 (id), p, br, hr, lists, links (http/https/mailto/tel + relative; title, rel), tables
 *    (colspan/rowspan/scope), figure/figcaption, img, blockquote and inline emphasis. `h1` is demoted to `h2`
 *    (a page has exactly one h1: the title). Wrappers (div, span, section …) are unwrapped, their text survives.
 *    Everything else (script, style, iframe, form, on* and style attributes …) is removed.
 *  - Images only from our own media: a numeric `data-media-id` (resolved by the renderer) or a `src` under one of
 *    `$imagePathPrefixes` (relative, or absolute on one of `$ownHosts`). Other images are removed.
 *  - `rel` keeps only nofollow/sponsored/ugc/noopener/noreferrer; a link whose href was removed is unwrapped.
 */
final class RichHtmlSanitizer
{
    private const REL_TOKENS = ['nofollow', 'sponsored', 'ugc', 'noopener', 'noreferrer'];

    private const UNWRAP = ['div', 'span', 'section', 'article', 'main', 'header', 'footer', 'aside', 'font', 'center', 'nav'];

    private ?HtmlSanitizer $sanitizer = null;

    /**
     * @param  list<string>  $ownHosts  hosts whose absolute image URLs count as our own (e.g. ritme.ir)
     * @param  list<string>  $imagePathPrefixes  root-relative path prefixes images may come from (e.g. /media/)
     */
    public function __construct(private readonly array $ownHosts = [], private readonly array $imagePathPrefixes = ['/media/']) {}

    public function sanitize(string $html): string
    {
        $html = trim($html);
        if ($html === '') {
            return '';
        }

        $html = (string) preg_replace('/<(\/?)h1\b/i', '<$1h2', $html);
        $clean = $this->sanitizer()->sanitize($html);

        $fragment = HtmlFragment::load($clean);
        $this->filterImages($fragment);
        $this->filterLinks($fragment);

        return $fragment->html();
    }

    private function sanitizer(): HtmlSanitizer
    {
        if ($this->sanitizer !== null) {
            return $this->sanitizer;
        }

        $config = (new HtmlSanitizerConfig)
            ->allowLinkSchemes(['http', 'https', 'mailto', 'tel'])
            ->allowRelativeLinks()
            ->allowMediaSchemes(['http', 'https'])
            ->allowMediaHosts($this->ownHosts)
            ->allowRelativeMedias()
            ->withMaxInputLength(2_000_000);

        foreach (['p', 'br', 'hr', 'strong', 'b', 'em', 'i', 'u', 's', 'del', 'ins', 'sub', 'sup', 'mark', 'small', 'code', 'pre',
            'ul', 'li', 'blockquote', 'figure', 'figcaption', 'table', 'caption', 'thead', 'tbody', 'tfoot', 'tr'] as $element) {
            $config = $config->allowElement($element);
        }

        $config = $config
            ->allowElement('h2', ['id'])
            ->allowElement('h3', ['id'])
            ->allowElement('h4', ['id'])
            ->allowElement('ol', ['start'])
            ->allowElement('a', ['href', 'title', 'rel'])
            ->allowElement('abbr', ['title'])
            ->allowElement('img', ['src', 'alt', 'width', 'height', 'title', 'loading', 'data-media-id'])
            ->allowElement('th', ['colspan', 'rowspan', 'scope'])
            ->allowElement('td', ['colspan', 'rowspan']);

        foreach (self::UNWRAP as $element) {
            $config = $config->blockElement($element);
        }

        return $this->sanitizer = new HtmlSanitizer($config);
    }

    private function filterImages(HtmlFragment $fragment): void
    {
        foreach ($fragment->elements('img') as $img) {
            $mediaId = trim($img->getAttribute('data-media-id'));
            if ($mediaId !== '' && preg_match('/^[1-9]\d{0,18}$/', $mediaId) !== 1) {
                $img->removeAttribute('data-media-id');
                $mediaId = '';
            }

            if ($mediaId === '' && ! $this->isOwnImage($img->getAttribute('src'))) {
                $this->remove($img);
            }
        }

        foreach ($fragment->elements('figure') as $figure) {
            if ($figure->getElementsByTagName('img')->length === 0 && trim($figure->textContent) === '') {
                $this->remove($figure);
            }
        }
    }

    private function filterLinks(HtmlFragment $fragment): void
    {
        foreach ($fragment->elements('a') as $link) {
            if (trim($link->getAttribute('href')) === '') { // href dropped (javascript: …) → keep only the text
                while ($link->firstChild !== null) {
                    $link->parentNode?->insertBefore($link->firstChild, $link);
                }
                $this->remove($link);

                continue;
            }

            if (! $link->hasAttribute('rel')) {
                continue;
            }

            $tokens = array_values(array_intersect(
                array_unique(preg_split('/\s+/', strtolower(trim($link->getAttribute('rel')))) ?: []),
                self::REL_TOKENS,
            ));

            $tokens === [] ? $link->removeAttribute('rel') : $link->setAttribute('rel', implode(' ', $tokens));
        }
    }

    private function isOwnImage(string $src): bool
    {
        $src = trim($src);
        if ($src === '') {
            return false;
        }

        $parts = parse_url($src);
        if ($parts === false) {
            return false;
        }

        if (isset($parts['host'])) {
            $host = strtolower($parts['host']);
            $own = array_map(strtolower(...), $this->ownHosts);
            if (! in_array($host, $own, true)) {
                return false;
            }
        } elseif (! str_starts_with($src, '/') || str_starts_with($src, '//')) {
            return false; // only root-relative paths
        }

        $path = $parts['path'] ?? '';
        if (str_contains($path, '..')) {
            return false;
        }

        foreach ($this->imagePathPrefixes as $prefix) {
            if ($prefix !== '' && str_starts_with($path, $prefix)) {
                return true;
            }
        }

        return false;
    }

    private function remove(DOMElement $element): void
    {
        $element->parentNode?->removeChild($element);
    }
}
