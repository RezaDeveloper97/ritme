<?php

declare(strict_types=1);

namespace App\Http\Controllers\Blog;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\AuthorData;
use App\Domain\Blog\Data\TagData;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Rendering\ArticlePage;
use App\Domain\Blog\Rendering\ArticlePageBuilder;
use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\Data\BlogPostingData;
use App\Domain\Seo\Schema\Data\PersonData;
use App\Domain\Seo\Schema\Nodes\BlogPostingNode;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Support\Html\HtmlText;
use App\View\Components\Picture;
use Carbon\CarbonImmutable;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\View;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

/**
 * `/blog/{slug}` — the article page (L4-03, design/html/article.html).
 *
 * Unknown slug: an old slug from the slug history 301s to the current URL (query string kept), anything else 404s.
 * SEO: admin seo_meta of the post wins (title, description, OG …), else the post title / excerpt; og:type article,
 * og:image = the cover's 1200×630 OG crop; `article:*` tags are pushed by the view. JSON-LD: WebPage (dates,
 * reviewedBy + lastReviewed), BlogPosting (author Person, publisher, articleSection, keywords), BreadcrumbList
 * (registered by <x-ui.breadcrumbs> in the view).
 *
 * Views are not counted here (a page-cache HIT never reaches the controller): the page's view beacon
 * (`data-module="view-beacon"` on <x-blog.meta>) posts to PostViewController once per visit (L4-03b).
 */
final class ShowPostController
{
    public function __construct(
        private readonly PostRepository $posts,
        private readonly ArticlePageBuilder $builder,
        private readonly SeoMetaRepository $seoMeta,
        private readonly OgImageResolver $ogImages,
        private readonly Config $config,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function __invoke(Request $request, string $slug, SeoManager $seo, SchemaGraph $graph): View|RedirectResponse
    {
        $post = $this->posts->findPublishedBySlug($slug);

        if ($post === null) {
            $current = $this->posts->currentSlugFor($slug);
            if ($current === null || $current === $slug) {
                abort(404);
            }

            $query = $request->getQueryString();

            return new RedirectResponse(route('blog.show', [$current]).($query !== null && $query !== '' ? '?'.$query : ''), 301);
        }

        $page = $this->builder->build($post);
        $this->describe($page, $seo, $graph);

        return view('pages.blog.show', ['page' => $page, 'navRoute' => 'blog.index', 'appCta' => true]);
    }

    private function describe(ArticlePage $page, SeoManager $seo, SchemaGraph $graph): void
    {
        $post = $page->post;
        $meta = $this->seoMeta->forModel((new Post)->getMorphClass(), $post->id);

        $seo->for((new Post)->forceFill(['id' => $post->id]));
        if (($meta->title ?? '') === '') {
            $seo->title($post->title);
        }
        $seo->excerpt($post->excerpt ?? HtmlText::plain($post->body));
        if (($meta->ogType ?? '') === '') {
            $seo->type('article');
        }

        $cover = $post->coverMediaId === null ? null : $this->ogImages->resolve($post->coverMediaId, $page->coverAlt);
        if ($cover !== null && $meta?->ogMediaId === null) {
            $seo->image($cover);
        }

        $head = $seo->resolve();
        $siteUrl = SchemaIds::root((string) $this->config->get('app.url'));
        $published = self::iso($post->publishedAt);
        $modified = self::iso($post->updatedContentAt);

        $graph->pageName($post->title)->dates($published, $modified);
        if ($post->reviewer !== null) {
            $graph->reviewedBy($this->person($post->reviewer, $page->reviewerUrl), $post->reviewedAt?->setTimezone(self::tz())->toDateString());
        }

        $graph->add(BlogPostingNode::make(new BlogPostingData(
            url: $head->canonical,
            headline: $post->title,
            datePublished: $published,
            authors: $post->author === null ? [] : [$this->person($post->author, $page->authorUrl)],
            dateModified: $modified,
            description: $head->description,
            image: $cover ?? $head->image,
            section: $post->category?->name,
            keywords: array_map(static fn (TagData $tag): string => $tag->name, $post->tags),
            wordCount: $post->wordCount > 0 ? $post->wordCount : null,
        ), $siteUrl));
    }

    /**
     * Person with URLs on the canonical origin (stable `@id` whatever host the request came in on).
     */
    private function person(AuthorData $author, ?string $url): PersonData
    {
        $avatar = $author->avatarMediaId === null ? null : Picture::mediaUrl($author->avatarMediaId, 'thumb');

        return $author->toPerson(
            $url === null ? null : CanonicalUrl::normalize($url, (string) $this->config->get('app.url')),
            $avatar === null ? null : CanonicalUrl::normalize($avatar, (string) $this->config->get('app.url')),
        );
    }

    /**
     * ISO 8601 with the Tehran offset (`+03:30`), as docs/SEO.md asks.
     */
    private static function iso(CarbonImmutable $date): string
    {
        return $date->setTimezone(self::tz())->toIso8601String();
    }

    private static function tz(): string
    {
        $tz = config('app.timezone');

        return is_string($tz) && $tz !== '' ? $tz : 'Asia/Tehran';
    }
}
