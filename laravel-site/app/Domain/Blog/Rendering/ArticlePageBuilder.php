<?php

declare(strict_types=1);

namespace App\Domain\Blog\Rendering;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\PostData;
use App\Domain\Blog\Support\BlogUrls;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Assembles the article page (L4-03) from cached reads only: the rendered body (ArticleBodyRenderer, `pages` ns),
 * related posts and prev/next in the category (PostRepository, `blog` ns), cover media (MediaRepository, `media`
 * ns) and the app links (settings). SEO head + JSON-LD are applied by the controller from this DTO.
 */
final class ArticlePageBuilder
{
    public const RELATED = 3;

    public function __construct(
        private readonly PostRepository $posts,
        private readonly ArticleBodyRenderer $renderer,
        private readonly MediaRepository $media,
        private readonly SettingsRepository $settings,
        private readonly SiteNavigation $navigation,
        private readonly BlogUrls $urls,
        private readonly Config $config,
    ) {}

    public function build(PostData $post): ArticlePage
    {
        $cover = $post->coverMediaId === null ? null : $this->media->find($post->coverMediaId);
        $mobileCover = $cover === null || $post->mobileCoverMediaId === null ? null : $this->media->find($post->mobileCoverMediaId);
        $url = CanonicalUrl::normalize($this->urls->post($post->slug), (string) $this->config->get('app.url'));
        $adjacent = $this->posts->adjacentInCategory($post->id);
        $appLinks = $this->settings->all()->appLinks;

        return new ArticlePage(
            post: $post,
            body: $this->renderer->render($post, eagerFirstImage: $cover === null),
            cover: $cover,
            mobileCover: $mobileCover,
            coverAlt: trim((string) $cover?->alt) !== '' ? trim((string) $cover?->alt) : $post->title,
            url: $url,
            breadcrumbs: $this->breadcrumbs($post, $url),
            related: $this->posts->related($post->id, self::RELATED),
            previous: $adjacent['previous'],
            next: $adjacent['next'],
            share: ShareLinks::for($url, $post->title),
            authorUrl: $post->author === null ? null : $this->urls->author($post->author->slug),
            reviewerUrl: $post->reviewer === null ? null : $this->urls->author($post->reviewer->slug),
            categoryUrl: $post->category === null ? null : $this->urls->category($post->category->slug),
            appLinks: $appLinks,
            downloadUrl: $this->navigation->url(StaticPage::Home, 'download'),
        );
    }

    /**
     * @return list<BreadcrumbItem>
     */
    private function breadcrumbs(PostData $post, string $url): array
    {
        $trail = [
            new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
            new BreadcrumbItem(StaticPage::Blog->label(), $this->navigation->url(StaticPage::Blog, absolute: true)),
        ];
        if ($post->category !== null) {
            $trail[] = new BreadcrumbItem($post->category->name, $this->urls->category($post->category->slug));
        }
        $trail[] = new BreadcrumbItem($post->title, $url);

        return $trail;
    }
}
