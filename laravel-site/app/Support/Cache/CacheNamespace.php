<?php

declare(strict_types=1);

namespace App\Support\Cache;

/**
 * A declared namespace (config/cacheaside.php) and its current version.
 */
final class CacheNamespace
{
    public function __construct(
        public readonly string $name,
        public readonly int $ttl,
        public readonly int $version,
    ) {}
}
