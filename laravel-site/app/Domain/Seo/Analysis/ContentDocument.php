<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

use App\Support\Html\HtmlFragment;
use App\Support\Html\HtmlText;

/**
 * The body parsed once for every check: plain text, paragraphs, headings, images and links. Accepts HTML (rich
 * editors) or plain text (a textarea description — paragraphs are its lines).
 */
final readonly class ContentDocument
{
    /**
     * @param  list<string>  $paragraphs
     * @param  list<array{level: int, text: string}>  $headings
     * @param  list<string|null>  $imageAlts  null = alt missing or empty
     * @param  list<array{href: string, rel: list<string>, blank: bool}>  $links
     */
    private function __construct(
        public string $text,
        public int $wordCount,
        public array $paragraphs,
        public array $headings,
        public array $imageAlts,
        public array $links,
    ) {}

    public static function parse(string $content): self
    {
        if (trim($content) === '') {
            return new self('', 0, [], [], [], []);
        }

        if (preg_match('/<[a-z][^>]*>/i', $content) !== 1) {
            return self::fromPlain($content);
        }

        $fragment = HtmlFragment::load($content);

        $paragraphs = [];
        foreach ($fragment->elements('p', 'li', 'blockquote') as $block) {
            if (strtolower($block->tagName) !== 'p' && $block->getElementsByTagName('p')->length > 0) {
                continue; // its <p> children (rich-editor list items, quotes) are counted themselves
            }
            $text = self::squish($block->textContent);
            if ($text !== '') {
                $paragraphs[] = $text;
            }
        }

        $headings = [];
        foreach ($fragment->elements('h1', 'h2', 'h3', 'h4', 'h5', 'h6') as $heading) {
            $headings[] = ['level' => (int) substr(strtolower($heading->tagName), 1), 'text' => self::squish($heading->textContent)];
        }

        $alts = [];
        foreach ($fragment->elements('img') as $image) {
            $alt = trim($image->getAttribute('alt'));
            $alts[] = $alt === '' ? null : $alt;
        }

        $links = [];
        foreach ($fragment->elements('a') as $link) {
            $href = trim($link->getAttribute('href'));
            if ($href === '') {
                continue;
            }
            $links[] = [
                'href' => $href,
                'rel' => preg_split('/\s+/', strtolower(trim($link->getAttribute('rel'))), -1, PREG_SPLIT_NO_EMPTY) ?: [],
                'blank' => strtolower(trim($link->getAttribute('target'))) === '_blank',
            ];
        }

        $text = HtmlText::plain($content);
        if ($paragraphs === [] && $text !== '') {
            $paragraphs = [$text];
        }

        return new self($text, PersianText::wordCount($text), $paragraphs, $headings, $alts, $links);
    }

    private static function fromPlain(string $content): self
    {
        $content = html_entity_decode($content, ENT_QUOTES | ENT_HTML5, 'UTF-8');
        $paragraphs = array_values(array_filter(array_map(self::squish(...), preg_split('/\R+/u', $content) ?: [])));
        $text = implode(' ', $paragraphs);

        return new self($text, PersianText::wordCount($text), $paragraphs, [], [], []);
    }

    private static function squish(string $text): string
    {
        return trim((string) preg_replace('/[\s\x{00A0}]+/u', ' ', $text));
    }
}
