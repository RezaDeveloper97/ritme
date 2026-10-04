<?php

declare(strict_types=1);

namespace App\Domain\Blog\Rendering;

use App\Domain\Blog\Data\PostData;
use App\Domain\Blog\Support\PostContent;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use App\Support\Html\HeadingAnchors;
use App\Support\Html\HtmlFragment;
use DOMElement;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;

/**
 * Turns a post's stored (already sanitised) body into the article HTML:
 *
 *  1. sanitised again with the same allow-list as on save (defence in depth: rows written around the observer —
 *     imports, raw SQL — still can't inject markup); external links get rel="noopener", h2/h3 get ids;
 *  2. every `<img data-media-id>` becomes `<x-picture>` (lazy, sizes for the 760px content column); the first one
 *     is eager + fetchpriority=high when the page has no cover (then it may be the LCP). No preload is pushed, so
 *     the result can be cached; images whose media no longer exists are dropped;
 *  3. tables are wrapped in a focusable, labelled scroll region (`.rt-table-scroll`, resources/css/prose.css).
 *
 * Cache-aside in the `pages` namespace — it is bumped by Blog observers (content) and by the MediaObserver (variant
 * URLs inside the <picture> markup), so a cached body never points at deleted variant files.
 */
final class ArticleBodyRenderer
{
    public const SIZES = '(max-width: 1024px) calc(100vw - 40px), 760px';

    public const IMAGE_VIEW = 'components.blog.body-image';

    public const TABLE_LABEL = 'جدول؛ برای دیدن همه ستون‌ها افقی بکش';

    public function __construct(
        private readonly CacheAside $cache,
        private readonly ViewFactory $views,
        private readonly Config $config,
    ) {}

    public function render(PostData $post, bool $eagerFirstImage = false): ArticleBody
    {
        $key = CacheKey::make('pages', 'article-body', $post->id, sha1($post->body), $eagerFirstImage ? 'eager' : 'lazy');

        /** @var array<string, mixed> $data */
        $data = $this->cache->remember($key, null, fn (): array => $this->build($post->body, $eagerFirstImage)->toArray());

        return ArticleBody::fromArray($data);
    }

    public function build(string $body, bool $eagerFirstImage = false): ArticleBody
    {
        $html = PostContent::fromConfig($this->config)->body($body);
        if (trim($html) === '') {
            return new ArticleBody('', []);
        }

        $fragment = HtmlFragment::load($html);
        $pictures = [];

        foreach ($fragment->elements('img') as $img) {
            $mediaId = trim($img->getAttribute('data-media-id'));
            if ($mediaId === '' || ! ctype_digit($mediaId)) {
                $this->lazy($img); // own /media/ image without a library record (legacy): keep, never eager

                continue;
            }

            $token = 'rt-picture-'.count($pictures);
            $markup = trim($this->views->make(self::IMAGE_VIEW, [
                'media' => (int) $mediaId,
                'alt' => trim($img->getAttribute('alt')) !== '' ? trim($img->getAttribute('alt')) : null,
                'eager' => $eagerFirstImage && $pictures === [],
                'sizes' => self::SIZES,
            ])->render());

            $placeholder = $fragment->document->createElement('span');
            $placeholder->setAttribute('data-rt-picture', $token);
            $img->parentNode?->replaceChild($placeholder, $img);
            $pictures['<span data-rt-picture="'.$token.'"></span>'] = $markup;
        }

        foreach ($fragment->elements('table') as $table) {
            $wrapper = $fragment->document->createElement('div');
            $wrapper->setAttribute('class', 'rt-table-scroll');
            $wrapper->setAttribute('role', 'region');
            $wrapper->setAttribute('tabindex', '0');
            $wrapper->setAttribute('aria-label', self::TABLE_LABEL);
            $table->parentNode?->replaceChild($wrapper, $table);
            $wrapper->appendChild($table);
        }

        $html = strtr($fragment->html(), $pictures);
        $html = (string) preg_replace('#<figure>\s*</figure>#', '', $html); // figure whose media was deleted

        return new ArticleBody($html, HeadingAnchors::outline($html), count(array_filter($pictures, static fn (string $m): bool => $m !== '')));
    }

    private function lazy(DOMElement $img): void
    {
        $img->setAttribute('loading', 'lazy');
        $img->setAttribute('decoding', 'async');
    }
}
