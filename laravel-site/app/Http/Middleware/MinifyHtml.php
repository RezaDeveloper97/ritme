<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use Closure;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\BinaryFileResponse;
use Symfony\Component\HttpFoundation\Response;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * Safe HTML minifier for public text/html responses (runs inside PageCache, so cached pages are stored minified).
 * Measured on the L1-02 layout pages: raw HTML −16 %, gzip −5 % (pagecache.minify, PAGE_CACHE_MINIFY).
 *
 * Keeps <pre>, <textarea>, <script> (incl. JSON-LD) and <style> blocks and conditional comments byte for byte;
 * removes other comments; collapses whitespace runs to one space and drops whitespace next to block-level tags
 * only (whitespace between inline elements is significant, so it is kept as a single space).
 */
final class MinifyHtml
{
    private const BLOCK = 'html|head|body|meta|link|title|base|main|header|footer|nav|section|article|aside|div|p|ul|ol|li|dl|dt|dd|h[1-6]|table|thead|tbody|tfoot|tr|th|td|form|fieldset|legend|figure|figcaption|source|blockquote|hr|br|details|summary|dialog|template|noscript|option';

    public function __construct(private readonly Config $config) {}

    public function handle(Request $request, Closure $next): Response
    {
        $response = $next($request);

        if (! $this->config->get('pagecache.minify', true)
            || AdminPaths::matches($request, $this->config) // Livewire snapshots live in attributes: never touch
            || $response instanceof StreamedResponse
            || $response instanceof BinaryFileResponse
            || ! str_starts_with(strtolower((string) $response->headers->get('Content-Type', '')), 'text/html')
        ) {
            return $response;
        }

        $content = $response->getContent();
        if (is_string($content) && $content !== '') {
            $response->setContent(self::minify($content));
        }

        return $response;
    }

    public static function minify(string $html): string
    {
        $kept = [];
        $protected = preg_replace_callback(
            '#<(pre|textarea|script|style)\b[^>]*>.*?</\1\s*>|<!--\[if\b.*?<!\[endif\]-->#is',
            static function (array $match) use (&$kept): string {
                $kept[] = $match[0];

                return "\x1A".(count($kept) - 1)."\x1A";
            },
            $html,
        );

        if (! is_string($protected)) {
            return $html; // PCRE backtrack limit: serve unminified rather than break the page
        }

        $block = self::BLOCK;
        $steps = [
            '#<!--(?!\[if\b).*?-->#s' => '',                       // comments (not conditional ones)
            '#[ \t\n\r\f]+#' => ' ',                             // ASCII whitespace runs → one space (keeps nbsp)
            "#[ ]?(</?(?:{$block})\\b[^>]*>)[ ]?#i" => '$1',       // no whitespace around block-level tags
        ];

        foreach ($steps as $pattern => $replacement) {
            $next = preg_replace($pattern, $replacement, $protected);
            if (! is_string($next)) {
                return $html;
            }
            $protected = $next;
        }

        return trim(preg_replace_callback("#\x1A(\\d+)\x1A#", static fn (array $m): string => $kept[(int) $m[1]], $protected) ?? $html);
    }
}
