<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use Illuminate\Contracts\Container\Container;
use InvalidArgumentException;

/**
 * Every SitemapProvider tagged with self::TAG, keyed by SitemapProvider::key(), in registration order.
 */
final class SitemapRegistry
{
    public const TAG = 'seo.sitemap.providers';

    private const KEY_PATTERN = '/^[a-z](?:[a-z0-9-]*[a-z])?$/';

    /** @var array<string, SitemapProvider>|null */
    private ?array $providers = null;

    public function __construct(private readonly Container $container) {}

    /**
     * @return array<string, SitemapProvider>
     */
    public function all(): array
    {
        if ($this->providers !== null) {
            return $this->providers;
        }

        $providers = [];
        foreach ($this->container->tagged(self::TAG) as $provider) {
            if (! $provider instanceof SitemapProvider) {
                throw new InvalidArgumentException(get_debug_type($provider).' is tagged '.self::TAG.' but does not implement '.SitemapProvider::class.'.');
            }
            $key = $provider->key();
            if (preg_match(self::KEY_PATTERN, $key) !== 1) {
                throw new InvalidArgumentException("Invalid sitemap provider key [{$key}].");
            }
            if (isset($providers[$key])) {
                throw new InvalidArgumentException("Duplicate sitemap provider key [{$key}].");
            }
            $providers[$key] = $provider;
        }

        return $this->providers = $providers;
    }

    public function find(string $key): ?SitemapProvider
    {
        return $this->all()[$key] ?? null;
    }
}
