<?php

declare(strict_types=1);

namespace App\Domain\Search\Support;

use App\Domain\Search\Contracts\SearchProvider;
use App\Domain\Search\Providers\FaqSearchProvider;
use App\Domain\Search\Providers\PostSearchProvider;
use Illuminate\Contracts\Container\Container;

/**
 * The providers `/search` asks, in result order for equal scores: the built-in magazine + FAQ providers, then every
 * provider another context tagged with TAG in its service provider, e.g.
 *
 *     $this->app->tag([PlaceSearchProvider::class], SearchRegistry::TAG);   // directory, L5-01
 */
final class SearchRegistry
{
    public const TAG = 'search.providers';

    /** @var list<class-string<SearchProvider>> */
    public const BUILT_IN = [PostSearchProvider::class, FaqSearchProvider::class];

    public function __construct(private readonly Container $container) {}

    /**
     * @return list<SearchProvider>
     */
    public function all(): array
    {
        $providers = [];
        foreach (self::BUILT_IN as $class) {
            $providers[$class] = $this->container->make($class);
        }

        foreach ($this->container->tagged(self::TAG) as $provider) {
            if ($provider instanceof SearchProvider) {
                $providers[$provider::class] = $provider;
            }
        }

        return array_values($providers);
    }
}
