<?php

declare(strict_types=1);

namespace App\Domain\Directory\Support;

use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;

/**
 * Absolute URLs of directory pages for sitemaps and search. Uses the named routes (`directory.place` L5-03,
 * `directory.city` / `directory.category` L5-02) and the planned paths while a route is not registered.
 */
final class DirectoryUrls
{
    public function __construct(private readonly UrlGenerator $url, private readonly Router $router) {}

    public function place(string $slug): string
    {
        return $this->router->has('directory.place')
            ? $this->url->route('directory.place', [$slug])
            : $this->url->to('/directory/place/'.rawurlencode($slug));
    }

    public function city(string $citySlug): string
    {
        return $this->router->has('directory.city')
            ? $this->url->route('directory.city', [$citySlug])
            : $this->url->to('/directory/'.rawurlencode($citySlug));
    }

    public function category(string $citySlug, string $categorySlug): string
    {
        return $this->router->has('directory.category')
            ? $this->url->route('directory.category', [$citySlug, $categorySlug])
            : $this->url->to('/directory/'.rawurlencode($citySlug).'/'.rawurlencode($categorySlug));
    }

    public function absolute(string $pathOrUrl): string
    {
        return $this->url->to($pathOrUrl);
    }
}
