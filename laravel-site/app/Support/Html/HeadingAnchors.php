<?php

declare(strict_types=1);

namespace App\Support\Html;

/**
 * Gives every h2/h3 a stable, unique `id` (Persian-friendly slug of its text) so the article TOC can link to it, and
 * reads the outline back for the TOC. An existing id is kept (normalised) unless it collides.
 *
 * @phpstan-type Heading array{level: int, id: string, text: string}
 */
final class HeadingAnchors
{
    /**
     * @param  list<string>  $tags
     */
    public static function apply(string $html, array $tags = ['h2', 'h3']): string
    {
        if (trim($html) === '' || preg_match('/<h[2-6]\b/i', $html) !== 1) {
            return $html;
        }

        $fragment = HtmlFragment::load($html);
        $used = [];

        foreach ($fragment->elements(...$tags) as $heading) {
            $base = TextSlug::make($heading->getAttribute('id'), 80);
            if ($base === '') {
                $base = TextSlug::make($heading->textContent, 80);
            }
            if ($base === '' || ctype_digit($base[0])) {
                $base = 'section'.($base === '' ? '' : '-'.$base);
            }

            $id = $base;
            for ($n = 2; isset($used[$id]); $n++) {
                $id = "{$base}-{$n}";
            }
            $used[$id] = true;
            $heading->setAttribute('id', $id);
        }

        return $fragment->html();
    }

    /**
     * @param  list<string>  $tags
     * @return list<Heading>
     */
    public static function outline(string $html, array $tags = ['h2', 'h3']): array
    {
        if (trim($html) === '') {
            return [];
        }

        $outline = [];
        foreach (HtmlFragment::load($html)->elements(...$tags) as $heading) {
            $id = $heading->getAttribute('id');
            $text = trim((string) preg_replace('/\s+/u', ' ', $heading->textContent));
            if ($id !== '' && $text !== '') {
                $outline[] = ['level' => (int) substr($heading->tagName, 1), 'id' => $id, 'text' => $text];
            }
        }

        return $outline;
    }
}
