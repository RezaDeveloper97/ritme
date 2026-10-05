<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Crawl;

use App\Domain\Blog\Models\Author as BlogAuthor;
use App\Domain\Blog\Models\Category as BlogCategory;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag as BlogTag;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Directory\Models\Place;
use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Shop\Catalog\Models\Category as ShopCategory;
use App\Domain\Shop\Catalog\Models\Product;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Routing\Router;
use Throwable;

/**
 * Maps a rendered route (name + parameters) to its admin edit page — the "fix" link of every finding on that URL —
 * plus the analyser content type and focus keyword. Admin routes are referenced by name only (the domain does not
 * know the Filament classes); a missing admin route just means no fix link.
 */
final class ContentTargets
{
    /**
     * public route => [model, slug parameter, admin edit route, content type, body column]
     *
     * @var array<string, array{0: class-string<Model>, 1: string, 2: string, 3: ContentType, 4: string|null}>
     */
    private const MODELS = [
        'blog.show' => [Post::class, 'slug', 'filament.admin.resources.blog.posts.edit', ContentType::Post, 'body'],
        'shop.product' => [Product::class, 'slug', 'filament.admin.resources.shop.products.edit', ContentType::Product, 'description'],
        'directory.place' => [Place::class, 'slug', 'filament.admin.resources.directory.places.edit', ContentType::Place, 'description'],
        'blog.category' => [BlogCategory::class, 'slug', 'filament.admin.resources.blog.categories.edit', ContentType::Archive, null],
        'blog.tag' => [BlogTag::class, 'slug', 'filament.admin.resources.blog.tags.edit', ContentType::Archive, null],
        'blog.author' => [BlogAuthor::class, 'slug', 'filament.admin.resources.blog.authors.edit', ContentType::Archive, null],
        'shop.category' => [ShopCategory::class, 'slug', 'filament.admin.resources.shop.categories.edit', ContentType::Archive, null],
    ];

    public const STATIC_PAGE_ROUTE = 'filament.admin.resources.seo.static-pages.edit';

    public const REDIRECTS_ROUTE = 'filament.admin.resources.seo.redirects.create';

    public const INDEXING_ROUTE = 'filament.admin.pages.seo.indexing';

    public const LANDINGS_ROUTE = 'filament.admin.resources.directory.landings.index';

    public function __construct(private readonly Router $router, private readonly UrlGenerator $url) {}

    /**
     * @param  array<string, string>  $parameters
     */
    public function for(?string $routeName, array $parameters): ContentTarget
    {
        if ($routeName === null) {
            return new ContentTarget;
        }

        $static = StaticPage::tryFrom($routeName);
        if ($static !== null) {
            return new ContentTarget(
                type: null,
                editUrl: $this->admin(self::STATIC_PAGE_ROUTE, ['page' => str_replace('.', '-', $routeName)]),
            );
        }

        if (in_array($routeName, ['directory.city', 'directory.category'], true)) {
            return new ContentTarget(type: ContentType::Archive, editUrl: $this->admin(self::LANDINGS_ROUTE));
        }

        $map = self::MODELS[$routeName] ?? null;
        $slug = $map === null ? null : ($parameters[$map[1]] ?? null);
        if ($map === null || $slug === null) {
            return new ContentTarget;
        }

        [$class, , $adminRoute, $type, $body] = $map;
        try {
            /** @var Model|null $model */
            $model = $class::query()->with('seoMeta')->where('slug', rawurldecode($slug))->first();
        } catch (Throwable) {
            $model = null;
        }
        if ($model === null) {
            return new ContentTarget(type: $type);
        }

        $meta = $model->getRelation('seoMeta');

        return new ContentTarget(
            type: $type,
            editUrl: $this->admin($adminRoute, ['record' => $model->getKey()]),
            focusKeyword: $meta instanceof Model ? trim((string) $meta->getAttribute('focus_keyword')) : '',
            contentHtml: $body === null ? null : (string) $model->getAttribute($body),
        );
    }

    public function redirectsUrl(): ?string
    {
        return $this->admin(self::REDIRECTS_ROUTE);
    }

    public function indexingUrl(): ?string
    {
        return $this->admin(self::INDEXING_ROUTE);
    }

    /**
     * @param  array<string, mixed>  $parameters
     */
    private function admin(string $name, array $parameters = []): ?string
    {
        if (! $this->router->has($name)) {
            return null;
        }

        try {
            return $this->url->route($name, $parameters, false);
        } catch (Throwable) {
            return null;
        }
    }
}
