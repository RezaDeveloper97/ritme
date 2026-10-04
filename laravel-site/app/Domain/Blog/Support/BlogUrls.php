<?php

declare(strict_types=1);

namespace App\Domain\Blog\Support;

use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;

/**
 * Absolute URLs of magazine pages for sitemaps/feeds. Uses the named routes once L4-02/L4-03 register them
 * (`blog.show`, `blog.category`) and the planned paths until then.
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

    public function absolute(string $pathOrUrl): string
    {
        return $this->url->to($pathOrUrl);
    }
}
