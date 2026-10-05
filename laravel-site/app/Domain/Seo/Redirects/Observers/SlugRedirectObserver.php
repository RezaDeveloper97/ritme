<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Observers;

use App\Domain\Blog\Models\Author as BlogAuthor;
use App\Domain\Blog\Models\Category as BlogCategory;
use App\Domain\Blog\Models\Tag as BlogTag;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Seo\Redirects\Actions\CreateSlugRedirect;
use App\Domain\Seo\Redirects\Support\RedirectPath;
use App\Domain\Shop\Catalog\Models\Category as ShopCategory;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Routing\Router;

/**
 * Auto 301s when the slug of a taxonomy page changes (L7-03): blog categories / tags / authors and shop categories
 * get an exact redirect old URL → new URL; a directory city or category appears in many landing URLs, so it gets
 * one regex redirect (`/directory/{old-city}[/{category}]`, `/directory/{city}/{old-category}`).
 *
 * Posts, products and places are NOT handled here: their slug history (blog_post_slugs, directory_place_slugs,
 * shop_product_slugs) already 301s inside their controllers, before any 404 — a redirect row would never be used.
 */
final class SlugRedirectObserver
{
    /** Model => route name of its page (first route parameter = the slug). */
    private const EXACT = [
        BlogCategory::class => 'blog.category',
        BlogTag::class => 'blog.tag',
        BlogAuthor::class => 'blog.author',
        ShopCategory::class => 'shop.category',
    ];

    private const PLACEHOLDER = 'rtslugplaceholder';

    public function __construct(
        private readonly CreateSlugRedirect $create,
        private readonly UrlGenerator $url,
        private readonly Router $router,
    ) {}

    public function updated(Model $model): void
    {
        if (! $model->wasChanged('slug')) {
            return;
        }

        $old = trim((string) $model->getOriginal('slug'));
        $new = trim((string) $model->getAttribute('slug'));
        if ($old === '' || $new === '' || $old === $new) {
            return;
        }

        $note = 'تغییر نامک: '.$old.' ← '.$new;
        $route = self::EXACT[$model::class] ?? null;

        if ($route !== null) {
            if ($this->router->has($route)) {
                $newPath = $this->path($route, [$new]);
                $this->create->handle($this->path($route, [$old]), $newPath, $newPath, false, $note);
            }

            return;
        }

        if ($model instanceof City && $this->router->has('directory.city')) {
            $oldPath = $this->path('directory.city', [$old]);
            $newPath = $this->path('directory.city', [$new]);
            $this->create->handle(preg_quote($oldPath, '~').'(/[^/]+)?', $this->literal($newPath).'$1', $newPath, true, $note);

            return;
        }

        if ($model instanceof PlaceCategory && $this->router->has('directory.category')) {
            $oldPath = preg_quote($this->path('directory.category', [self::PLACEHOLDER, $old]), '~');
            $newPath = $this->path('directory.category', [self::PLACEHOLDER, $new]);
            $this->create->handle(
                str_replace(self::PLACEHOLDER, '([^/]+)', $oldPath),
                str_replace(self::PLACEHOLDER, '$1', $this->literal($newPath)),
                str_replace(self::PLACEHOLDER, 'x', $newPath),
                true,
                $note,
            );
        }
    }

    /**
     * @param  list<string>  $parameters
     */
    private function path(string $route, array $parameters): string
    {
        return RedirectPath::normalize((string) parse_url($this->url->route($route, $parameters, false), PHP_URL_PATH));
    }

    /** A literal path inside a preg_replace replacement. */
    private function literal(string $path): string
    {
        return str_replace(['\\', '$'], ['\\\\', '\\$'], $path);
    }
}
