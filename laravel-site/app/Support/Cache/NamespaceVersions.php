<?php

declare(strict_types=1);

namespace App\Support\Cache;

use Illuminate\Contracts\Cache\Factory as CacheFactory;
use Illuminate\Contracts\Cache\Repository;
use Illuminate\Contracts\Config\Repository as Config;
use InvalidArgumentException;

/**
 * Versioned cache namespaces: keys are rt:{ns}:v{version}:{key}; bump() increments the version so every
 * existing entry of the namespace becomes unreachable and expires on its own. Needs no cache tags, so it
 * works on the file and database drivers. Versions are stored forever in the same store as the values.
 */
final class NamespaceVersions
{
    private readonly Repository $store;

    private readonly string $prefix;

    /** @var array<string, int> */
    private readonly array $namespaces;

    public function __construct(CacheFactory $cache, Config $config)
    {
        $store = $config->get('cacheaside.store');
        $this->store = $cache->store(is_string($store) && $store !== '' ? $store : null);
        $this->prefix = (string) $config->get('cacheaside.prefix', 'rt');

        $namespaces = [];
        foreach ((array) $config->get('cacheaside.namespaces', []) as $name => $ttl) {
            $namespaces[(string) $name] = (int) $ttl;
        }
        $this->namespaces = $namespaces;
    }

    public function store(): Repository
    {
        return $this->store;
    }

    public function prefix(): string
    {
        return $this->prefix;
    }

    public function has(string $namespace): bool
    {
        return array_key_exists($namespace, $this->namespaces);
    }

    /**
     * Default TTL (seconds) of a declared namespace.
     */
    public function ttl(string $namespace): int
    {
        $this->assertDeclared($namespace);

        return $this->namespaces[$namespace];
    }

    public function version(string $namespace): int
    {
        $this->assertDeclared($namespace);

        $version = $this->store->get($this->versionKey($namespace));

        return is_numeric($version) ? max(1, (int) $version) : 1;
    }

    /**
     * Invalidate one or more namespaces. Returns the new version of each.
     *
     * @return array<string, int>
     */
    public function bump(string ...$namespaces): array
    {
        $versions = [];
        foreach (array_unique($namespaces) as $namespace) {
            $this->assertDeclared($namespace);
            $key = $this->versionKey($namespace);

            // A missing version means 1; materialise it so increment() works on every driver
            // (the database store cannot increment a missing row).
            $this->store->add($key, 1);
            $new = $this->store->increment($key);

            $versions[$namespace] = is_int($new) ? $new : $this->version($namespace);
        }

        return $versions;
    }

    /**
     * @return array<string, int>
     */
    public function bumpAll(): array
    {
        return $this->bump(...array_keys($this->namespaces));
    }

    /**
     * @return list<CacheNamespace>
     */
    public function all(): array
    {
        $all = [];
        foreach ($this->namespaces as $name => $ttl) {
            $all[] = new CacheNamespace($name, $ttl, $this->version($name));
        }

        return $all;
    }

    /**
     * The physical key: rt:{ns}:v{version}:{key}.
     */
    public function qualify(CacheKey $key): string
    {
        return "{$this->prefix}:{$key->namespace}:v{$this->version($key->namespace)}:{$key->key}";
    }

    private function versionKey(string $namespace): string
    {
        return "{$this->prefix}:nsv:{$namespace}";
    }

    private function assertDeclared(string $namespace): void
    {
        if (! $this->has($namespace)) {
            throw new InvalidArgumentException("Cache namespace [{$namespace}] is not declared in config/cacheaside.php.");
        }
    }
}
