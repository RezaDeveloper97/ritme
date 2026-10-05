<?php

declare(strict_types=1);

namespace Tests\Unit\Support\Cache;

use Illuminate\Filesystem\Filesystem;
use Illuminate\Support\Facades\Cache;

/**
 * Points the cache-aside kernel at a fresh store of the given driver (array, file, database).
 */
final class CacheStores
{
    /** @return array<string, array{0: string}> */
    public static function drivers(): array
    {
        return ['array' => ['array'], 'file' => ['file'], 'database' => ['database']];
    }

    public static function use(string $driver): void
    {
        $name = "cacheaside_{$driver}";
        $config = ['driver' => $driver];

        if ($driver === 'file') {
            // Per process: a concurrent run (or parallel worker) must not flush this one's files.
            $dir = storage_path('framework/testing/cache-aside-'.getmypid());
            if (! is_dir($dir)) {
                register_shutdown_function(static fn () => (new Filesystem)->deleteDirectory($dir));
            }
            (new Filesystem)->deleteDirectory($dir);
            $config += ['path' => $dir, 'lock_path' => $dir];
        }

        if ($driver === 'database') {
            $config += ['connection' => null, 'table' => 'cache', 'lock_connection' => null, 'lock_table' => 'cache_locks'];
        }

        config(["cache.stores.{$name}" => $config, 'cacheaside.store' => $name]);
        Cache::forgetDriver($name);
        Cache::store($name)->flush();
    }
}
