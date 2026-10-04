<?php

declare(strict_types=1);

namespace App\Domain\Blog\Support;

use App\Domain\Blog\Models\Post;
use App\Support\Html\ExternalLinks;
use App\Support\Html\HeadingAnchors;
use App\Support\Html\HtmlText;
use App\Support\Html\RichHtmlSanitizer;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Prepares a post's content on save: body and sources go through the allow-list sanitiser (own-media images only),
 * external links get rel="noopener", body h2/h3 get ids for the TOC; the excerpt becomes plain text; word count and
 * reading time (Persian word count, 200 wpm) are recomputed.
 */
final class PostContent
{
    public const WORDS_PER_MINUTE = 200;

    /**
     * @param  list<string>  $ownHosts
     */
    public function __construct(private readonly RichHtmlSanitizer $sanitizer, private readonly array $ownHosts) {}

    public static function fromConfig(Config $config): self
    {
        $host = parse_url((string) $config->get('app.url'), PHP_URL_HOST);
        $hosts = is_string($host) && $host !== '' ? [strtolower($host)] : [];
        if ($hosts !== [] && ! str_starts_with($hosts[0], 'www.')) {
            $hosts[] = 'www.'.$hosts[0];
        }

        $disk = (string) $config->get('media.disk', 'public');
        $mediaUrl = (string) $config->get("filesystems.disks.{$disk}.url", '/media');
        $mediaPath = parse_url($mediaUrl, PHP_URL_PATH);
        $prefix = rtrim(is_string($mediaPath) && $mediaPath !== '' ? $mediaPath : '/media', '/').'/';

        return new self(new RichHtmlSanitizer($hosts, [$prefix]), $hosts);
    }

    public function body(string $html): string
    {
        return HeadingAnchors::apply(ExternalLinks::apply($this->sanitizer->sanitize($html), $this->ownHosts));
    }

    public function sources(?string $html): ?string
    {
        $clean = ExternalLinks::apply($this->sanitizer->sanitize((string) $html), $this->ownHosts);

        return HtmlText::plain($clean) === '' ? null : $clean;
    }

    public function prepare(Post $post): void
    {
        if (! $post->exists || $post->isDirty('body')) {
            $post->body = $this->body((string) $post->getAttribute('body'));
            $post->setAttribute('word_count', HtmlText::wordCount($post->body));
            $post->setAttribute('reading_time', HtmlText::readingMinutes($post->word_count, self::WORDS_PER_MINUTE));
        }

        if (! $post->exists || $post->isDirty('sources')) {
            $post->sources = $this->sources($post->sources);
        }

        if (! $post->exists || $post->isDirty('excerpt')) {
            $excerpt = HtmlText::plain((string) $post->excerpt);
            $post->excerpt = $excerpt === '' ? null : $excerpt;
        }
    }
}
