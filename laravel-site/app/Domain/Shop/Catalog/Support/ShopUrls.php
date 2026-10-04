<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Support;

use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;

/**
 * Absolute URLs of shop pages for sitemaps and search. Uses the named routes (`shop.product` L6-03, `shop.category`
 * L6-02) and the planned paths while a route is not registered.
 */
final class ShopUrls
{
    public function __construct(private readonly UrlGenerator $url, private readonly Router $router) {}

    public function product(string $slug): string
    {
        return $this->router->has('shop.product')
            ? $this->url->route('shop.product', [$slug])
            : $this->url->to('/shop/product/'.rawurlencode($slug));
    }

    public function category(string $slug): string
    {
        return $this->router->has('shop.category')
            ? $this->url->route('shop.category', [$slug])
            : $this->url->to('/shop/category/'.rawurlencode($slug));
    }

    public function absolute(string $pathOrUrl): string
    {
        return $this->url->to($pathOrUrl);
    }
}
