<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use Illuminate\Contracts\Config\Repository as Config;

/**
 * Built-in production rules: everything is crawlable except the admin panel and the transactional pages
 * (cart, checkout, order and booking confirmations), which are also noindex.
 */
final class DefaultRobotsRules implements RobotsRules
{
    public function __construct(private readonly Config $config) {}

    public function rules(): string
    {
        $admin = trim((string) $this->config->get('filament.admin.path', 'admin'), '/');
        $disallow = array_filter([
            $admin === '' ? null : '/'.$admin,
            '/livewire/',
            '/shop/cart',
            '/shop/checkout',
            '/shop/order/',
            '/directory/booked/',
        ]);

        return "User-agent: *\n".implode("\n", array_map(static fn (string $path): string => 'Disallow: '.$path, $disallow));
    }
}
