<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\IndexNow;

use App\Domain\Blog\Models\Post;
use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Models\Place;
use App\Domain\Seo\Sitemap\SitemapUrl;
use App\Domain\Shop\Catalog\Models\Product;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Routing\Router;

/**
 * Which public URLs a content change affects, for IndexNow: the page of a post / product / place that is visible
 * now, or was visible before the change (unpublished, slug changed, deleted — so engines recrawl and drop it).
 * Demo / sample rows are never submitted (they are noindex and outside the sitemap).
 */
final class IndexNowUrls
{
    /** Model => public route (single `{slug}` parameter). */
    public const ROUTES = [
        Post::class => 'blog.show',
        Product::class => 'shop.product',
        Place::class => 'directory.place',
    ];

    public function __construct(
        private readonly Router $router,
        private readonly UrlGenerator $url,
        private readonly SitemapUrl $canonical,
    ) {}

    /**
     * @return list<string>
     */
    public function forChange(Model $model, bool $deleted = false): array
    {
        $route = self::ROUTES[$model::class] ?? null;
        if ($route === null || ! $this->router->has($route) || (bool) $model->getAttribute('is_demo')) {
            return [];
        }

        if (! $deleted && ! $model->wasRecentlyCreated && ! $model->wasChanged()) {
            return []; // a no-op save
        }

        $previous = $model->replicate();
        $previous->setRawAttributes(array_merge($model->getAttributes(), $model->getPrevious()));

        $urls = [];
        if (! $deleted && self::visible($model)) {
            $urls[] = $this->canonical->page($this->url->route($route, ['slug' => (string) $model->getAttribute('slug')]));
        }
        if (($deleted || $model->getPrevious() !== []) && self::visible($previous)) { // wasRecentlyCreated sticks to the instance
            $urls[] = $this->canonical->page($this->url->route($route, ['slug' => (string) $previous->getAttribute('slug')]));
        }

        return array_values(array_unique($urls));
    }

    private static function visible(Model $model): bool
    {
        return match (true) {
            $model instanceof Post => $model->isPublished(),
            $model instanceof Product => $model->is_published,
            $model instanceof Place => $model->status === PlaceStatus::Published,
            default => false,
        };
    }
}
