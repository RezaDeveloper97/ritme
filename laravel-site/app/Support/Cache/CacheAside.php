<?php

declare(strict_types=1);

namespace App\Support\Cache;

use Closure;
use DateInterval;
use DateTimeImmutable;
use Illuminate\Contracts\Cache\LockProvider;
use Illuminate\Contracts\Cache\LockTimeoutException;
use Illuminate\Contracts\Cache\Repository;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Cache-aside reads over versioned namespaces. Inject it (no facade in domain code); cached repository
 * decorators usually extend CachedRepository instead of using it directly.
 *
 *  - TTL jitter (±cacheaside.jitter) so entries written together don't expire together.
 *  - Stampede protection with an atomic lock when the store supports it (fallback: no lock).
 *  - `null` results are not cached unless $cacheNull is true.
 */
final class CacheAside
{
    /** Stored in place of a cached null so a hit can be told apart from a miss. */
    public const NULL_VALUE = '__rt_cache_null__';

    private readonly Repository $store;

    private readonly float $jitter;

    private readonly bool $lockEnabled;

    private readonly int $lockSeconds;

    private readonly int $lockWait;

    public function __construct(private readonly NamespaceVersions $versions, Config $config)
    {
        $this->store = $versions->store();
        $this->jitter = min(1.0, max(0.0, (float) $config->get('cacheaside.jitter', 0.1)));
        $this->lockEnabled = (bool) $config->get('cacheaside.lock.enabled', true);
        $this->lockSeconds = max(1, (int) $config->get('cacheaside.lock.seconds', 10));
        $this->lockWait = max(0, (int) $config->get('cacheaside.lock.wait', 3));
    }

    /**
     * @template T
     *
     * @param  int|DateInterval|null  $ttl  null = the namespace's default TTL
     * @param  Closure(): T  $loader
     * @return T
     */
    public function remember(CacheKey $key, int|DateInterval|null $ttl, Closure $loader, bool $cacheNull = false): mixed
    {
        $seconds = $this->seconds($ttl ?? $this->versions->ttl($key->namespace));

        if ($seconds <= 0) {
            return $loader(); // a non-positive TTL means "don't cache"
        }

        return $this->load($key, $loader, $cacheNull, $this->jitter($seconds));
    }

    /**
     * @template T
     *
     * @param  Closure(): T  $loader
     * @return T
     */
    public function rememberForever(CacheKey $key, Closure $loader, bool $cacheNull = false): mixed
    {
        return $this->load($key, $loader, $cacheNull, null);
    }

    public function forget(CacheKey $key): bool
    {
        return $this->store->forget($this->versions->qualify($key));
    }

    /**
     * Applies ±jitter to a TTL in seconds (never below 1).
     */
    public function jitter(int $seconds): int
    {
        $spread = (int) floor($seconds * $this->jitter);

        if ($spread < 1) {
            return max(1, $seconds);
        }

        return max(1, $seconds + random_int(-$spread, $spread));
    }

    /**
     * @param  Closure(): mixed  $loader
     */
    private function load(CacheKey $key, Closure $loader, bool $cacheNull, ?int $seconds): mixed
    {
        $qualified = $this->versions->qualify($key);
        $found = false;
        $value = $this->read($qualified, $found);

        if ($found) {
            return $value;
        }

        $fill = function () use ($qualified, $loader, $cacheNull, $seconds): mixed {
            $found = false;
            $value = $this->read($qualified, $found);

            if ($found) {
                return $value; // another process filled it while we waited for the lock
            }

            $value = $loader();
            $this->write($qualified, $value, $cacheNull, $seconds);

            return $value;
        };

        $store = $this->store->getStore();

        if (! $this->lockEnabled || ! $store instanceof LockProvider) {
            return $fill();
        }

        try {
            return $store->lock("{$qualified}:lock", $this->lockSeconds)->block($this->lockWait, $fill);
        } catch (LockTimeoutException) {
            return $fill();
        }
    }

    private function read(string $qualified, bool &$found): mixed
    {
        $value = $this->store->get($qualified);
        $found = $value !== null;

        return $value === self::NULL_VALUE ? null : $value;
    }

    private function write(string $qualified, mixed $value, bool $cacheNull, ?int $seconds): void
    {
        if ($value === null && ! $cacheNull) {
            return;
        }

        $stored = $value ?? self::NULL_VALUE;

        if ($seconds === null) {
            $this->store->forever($qualified, $stored);
        } else {
            $this->store->put($qualified, $stored, $seconds);
        }
    }

    private function seconds(int|DateInterval $ttl): int
    {
        if ($ttl instanceof DateInterval) {
            $now = new DateTimeImmutable('@0');

            return $now->add($ttl)->getTimestamp() - $now->getTimestamp();
        }

        return $ttl;
    }
}
