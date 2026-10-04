<?php

declare(strict_types=1);

namespace App\Domain\Blog\Support;

use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;

/**
 * Absolute URLs of magazine pages for sitemaps/feeds. Uses the named routes (`blog.show` L4-03, `blog.category`,
 * `blog.tag`, `blog.author` L4-02) and the planned paths while a route is not registered.
 */
final class BlogUrls
{
    public function __construct(private readonly UrlGenerator $url, private readonly Router $router) {}

    public function post(string $slug): string
    {
        return $this->router->has('blog.show')
            ? $this->url->route('blog.show', [$slug])
            : $this->url->to('/blog/'.rawurlencode($slug));
    }

    public function category(string $slug): string
    {
        return $this->router->has('blog.category')
            ? $this->url->route('blog.category', [$slug])
            : $this->url->to('/blog/category/'.rawurlencode($slug));
    }

    public function tag(string $slug): string
    {
        return $this->router->has('blog.tag')
            ? $this->url->route('blog.tag', [$slug])
            : $this->url->to('/blog/tag/'.rawurlencode($slug));
    }

    public function author(string $slug): string
    {
        return $this->router->has('blog.author')
            ? $this->url->route('blog.author', [$slug])
            : $this->url->to('/blog/author/'.rawurlencode($slug));
    }

    public function absolute(string $pathOrUrl): string
    {
        return $this->url->to($pathOrUrl);
    }
}
