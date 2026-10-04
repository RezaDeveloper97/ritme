<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Support\BlogUrls;
use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Carbon\CarbonImmutable;
use Illuminate\Http\Response;

/**
 * `/blog/feed` (L4-04): RSS 2.0 of the newest LIMIT published posts — title, absolute permalink + guid, RFC 822
 * pubDate, category, full excerpt, cover as `<enclosure>` (OG variant, whose byte size is known). The rendered XML is
 * cached in the `blog` namespace, so publishing / editing a post (PostObserver bumps `blog`) refreshes it.
 */
final class FeedController
{
    public const LIMIT = 30;

    private const MIME = ['jpg' => 'image/jpeg', 'jpeg' => 'image/jpeg', 'png' => 'image/png', 'webp' => 'image/webp', 'avif' => 'image/avif', 'gif' => 'image/gif'];

    public function __construct(
        private readonly PostRepository $posts,
        private readonly MediaRepository $media,
        private readonly SettingsRepository $settings,
        private readonly BlogUrls $urls,
        private readonly CacheAside $cache,
    ) {}

    public function __invoke(): Response
    {
        $xml = $this->cache->remember(CacheKey::make('blog', 'feed', 'rss'), null, fn (): string => $this->render());

        return new Response($xml, 200, [
            'Content-Type' => 'application/rss+xml; charset=UTF-8',
            'Cache-Control' => 'public, max-age=900',
            'X-Robots-Tag' => 'noindex, follow',
        ]);
    }

    private function render(): string
    {
        $posts = $this->posts->latest(1, self::LIMIT)->items;
        $covers = $this->media->findMany(array_values(array_unique(array_filter(array_map(
            static fn (PostCardData $post): ?int => $post->coverMediaId,
            $posts,
        )))));
        $site = $this->settings->all()->general->siteName;
        $newest = $posts === [] ? CarbonImmutable::now() : max(array_map(static fn (PostCardData $post): CarbonImmutable => $post->updatedContentAt->max($post->publishedAt), $posts));

        // ltrim: the XML declaration must be the very first bytes (the view's doc comment leaves a newline).
        return ltrim(view('feed.rss', [
            'channel' => [
                'title' => __('search.feed.title', ['site' => $site]),
                'link' => $this->urls->absolute(route('blog.index')),
                'self' => $this->urls->absolute(route('blog.feed')),
                'description' => __('blog.index.seo_description'),
                'lastBuildDate' => $newest->toRssString(),
            ],
            'items' => array_map(fn (PostCardData $post): array => [
                'title' => $post->title,
                'link' => $this->urls->post($post->slug),
                'description' => $post->excerpt ?? '',
                'category' => $post->category?->name,
                'pubDate' => $post->publishedAt->toRssString(),
                'enclosure' => $post->coverMediaId === null ? null : $this->enclosure($covers[$post->coverMediaId] ?? null),
            ], $posts),
        ])->render());
    }

    /**
     * @return array{url: string, length: int, type: string}|null
     */
    private function enclosure(?MediaData $media): ?array
    {
        if ($media === null) {
            return null;
        }

        $format = $media->fallbackFormat();
        $file = $media->file('og', $format);
        $type = self::MIME[$format] ?? null;
        if ($file === null || $type === null || $file['size'] <= 0) {
            return null;
        }

        return ['url' => $this->urls->absolute($file['url']), 'length' => $file['size'], 'type' => $type];
    }
}
