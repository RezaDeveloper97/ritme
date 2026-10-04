<?php

declare(strict_types=1);

use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Cache\ArrayStore;
use Illuminate\Cache\Events\CacheMissed;
use Illuminate\Contracts\Cache\Store;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Event;
use Tests\TestCase;
use Tests\Unit\Support\Cache\CacheStores;

uses(TestCase::class, RefreshDatabase::class);

beforeEach(function (): void {
    config([
        'cacheaside.namespaces' => ['blog' => 100, 'faq' => 60],
        'cacheaside.jitter' => 0.1,
        'cacheaside.lock' => ['enabled' => true, 'seconds' => 10, 'wait' => 0],
    ]);
});

function cacheAside(): CacheAside
{
    return app(CacheAside::class);
}

/**
 * A loader that counts its calls.
 *
 * @return object{calls: int}
 */
function countingLoader(mixed $value): object
{
    return new class($value)
    {
        public int $calls = 0;

        public function __construct(private readonly mixed $value) {}

        public function __invoke(): mixed
        {
            $this->calls++;

            return $this->value;
        }
    };
}

it('misses once, then hits', function (string $driver): void {
    CacheStores::use($driver);
    $key = CacheKey::make('blog', 'post', 1);
    $loader = countingLoader(['title' => 'سلام']);

    $first = cacheAside()->remember($key, 60, $loader(...));
    $second = cacheAside()->remember($key, 60, $loader(...));

    expect($first)->toBe(['title' => 'سلام'])
        ->and($second)->toBe(['title' => 'سلام'])
        ->and($loader->calls)->toBe(1)
        ->and(app(NamespaceVersions::class)->store()->get('rt:blog:v1:post:1'))->toBe(['title' => 'سلام']);
})->with(CacheStores::drivers());

it('invalidates a namespace on bump and leaves others alone', function (string $driver): void {
    CacheStores::use($driver);
    $post = CacheKey::make('blog', 'post', 1);
    $faq = CacheKey::make('faq', 'all');

    cacheAside()->remember($post, 60, fn () => 'old');
    cacheAside()->remember($faq, 60, fn () => 'faq');
    app(NamespaceVersions::class)->bump('blog');

    expect(cacheAside()->remember($post, 60, fn () => 'new'))->toBe('new')
        ->and(cacheAside()->remember($faq, 60, fn () => 'changed'))->toBe('faq');
})->with(CacheStores::drivers());

it('forgets a single key', function (string $driver): void {
    CacheStores::use($driver);
    $key = CacheKey::make('blog', 'post', 1);
    cacheAside()->remember($key, 60, fn () => 'old');

    expect(cacheAside()->forget($key))->toBeTrue()
        ->and(cacheAside()->remember($key, 60, fn () => 'new'))->toBe('new');
})->with(CacheStores::drivers());

it('does not cache null unless asked', function (string $driver): void {
    CacheStores::use($driver);
    $key = CacheKey::make('blog', 'missing');
    $loader = countingLoader(null);

    cacheAside()->remember($key, 60, $loader(...));
    cacheAside()->remember($key, 60, $loader(...));
    expect($loader->calls)->toBe(2);

    $cached = CacheKey::make('blog', 'missing-cached');
    $nullLoader = countingLoader(null);
    $first = cacheAside()->remember($cached, 60, $nullLoader(...), cacheNull: true);
    $second = cacheAside()->remember($cached, 60, $nullLoader(...), cacheNull: true);

    expect($first)->toBeNull()
        ->and($second)->toBeNull()
        ->and($nullLoader->calls)->toBe(1);
})->with(CacheStores::drivers());

it('caches falsy non-null values', function (mixed $value): void {
    CacheStores::use('array');
    $key = CacheKey::make('blog', 'falsy');
    $loader = countingLoader($value);

    cacheAside()->remember($key, 60, $loader(...));

    expect(cacheAside()->remember($key, 60, $loader(...)))->toBe($value)
        ->and($loader->calls)->toBe(1);
})->with([[false], [0], [''], [[]]]);

it('remembers forever without expiry', function (string $driver): void {
    CacheStores::use($driver);
    $key = CacheKey::make('blog', 'forever');
    $loader = countingLoader('v');

    cacheAside()->rememberForever($key, $loader(...));
    $this->travel(5)->years();

    expect(cacheAside()->rememberForever($key, $loader(...)))->toBe('v')
        ->and($loader->calls)->toBe(1);

    $nullKey = CacheKey::make('blog', 'forever-null');
    $nullLoader = countingLoader(null);
    cacheAside()->rememberForever($nullKey, $nullLoader(...), cacheNull: true);
    cacheAside()->rememberForever($nullKey, $nullLoader(...), cacheNull: true);
    expect($nullLoader->calls)->toBe(1);
})->with(['array', 'file', 'database']);

it('expires within the jittered ttl', function (string $driver, int|DateInterval|null $ttl): void {
    CacheStores::use($driver);
    $key = CacheKey::make('blog', 'ttl');
    $loader = countingLoader('v');

    cacheAside()->remember($key, $ttl, $loader(...));
    $this->travel(89)->seconds();
    cacheAside()->remember($key, $ttl, $loader(...));
    expect($loader->calls)->toBe(1);

    $this->travel(22)->seconds(); // 111s > 100s + 10 %
    cacheAside()->remember($key, $ttl, $loader(...));
    expect($loader->calls)->toBe(2);
})->with(['array', 'file', 'database'])->with([
    'namespace default' => [null],
    'seconds' => [100],
    'interval' => [new DateInterval('PT100S')],
]);

it('does not cache with a non-positive ttl', function (): void {
    CacheStores::use('array');
    $key = CacheKey::make('blog', 'zero');
    $loader = countingLoader('v');

    cacheAside()->remember($key, 0, $loader(...));
    cacheAside()->remember($key, 0, $loader(...));

    expect($loader->calls)->toBe(2);
});

it('keeps jitter within ±10 %', function (): void {
    $seen = [];
    for ($i = 0; $i < 500; $i++) {
        $seen[] = cacheAside()->jitter(1000);
    }

    expect(min($seen))->toBeGreaterThanOrEqual(900)
        ->and(max($seen))->toBeLessThanOrEqual(1100)
        ->and(count(array_unique($seen)))->toBeGreaterThan(1);
});

it('does not jitter tiny ttls or when disabled', function (): void {
    expect(cacheAside()->jitter(5))->toBe(5)
        ->and(cacheAside()->jitter(0))->toBe(1);

    config(['cacheaside.jitter' => 0]);
    expect(cacheAside()->jitter(1000))->toBe(1000);

    config(['cacheaside.jitter' => 5]); // clamped to 100 %
    expect(cacheAside()->jitter(10))->toBeGreaterThanOrEqual(1)->toBeLessThanOrEqual(20);
});

it('runs the loader while holding the stampede lock and releases it', function (string $driver): void {
    CacheStores::use($driver);
    $key = CacheKey::make('blog', 'locked');
    $store = app(NamespaceVersions::class)->store();
    $lockName = 'rt:blog:v1:locked:lock';

    $value = cacheAside()->remember($key, 60, function () use ($store, $lockName): string {
        expect($store->lock($lockName, 5)->get())->toBeFalse(); // held by CacheAside

        return 'v';
    });

    expect($value)->toBe('v')
        ->and($store->lock($lockName, 5)->get())->toBeTrue(); // released
})->with(CacheStores::drivers());

it('loads without the lock when waiting times out', function (string $driver): void {
    CacheStores::use($driver);
    $store = app(NamespaceVersions::class)->store();
    $store->lock('rt:blog:v1:busy:lock', 30)->get(); // another process holds it

    expect(cacheAside()->remember(CacheKey::make('blog', 'busy'), 60, fn () => 'v'))->toBe('v')
        ->and($store->get('rt:blog:v1:busy'))->toBe('v');
})->with(CacheStores::drivers());

it('uses the value another process stored while it waited for the lock', function (string $driver): void {
    CacheStores::use($driver);
    $store = app(NamespaceVersions::class)->store();
    $filled = false;
    Event::listen(CacheMissed::class, function (CacheMissed $event) use ($store, &$filled): void {
        if (! $filled && $event->key === 'rt:blog:v1:race') {
            $filled = true;
            $store->put('rt:blog:v1:race', 'from-other-process', 60);
        }
    });
    $loader = countingLoader('mine');

    expect(cacheAside()->remember(CacheKey::make('blog', 'race'), 60, $loader(...)))->toBe('from-other-process')
        ->and($loader->calls)->toBe(0);
})->with(CacheStores::drivers());

it('works without locks when disabled or unsupported', function (): void {
    $plain = new class implements Store
    {
        private ArrayStore $inner;

        public function __construct()
        {
            $this->inner = new ArrayStore;
        }

        public function get($key): mixed
        {
            return $this->inner->get($key);
        }

        /** @param array<int, string> $keys */
        public function many(array $keys): array
        {
            return $this->inner->many($keys);
        }

        public function put($key, $value, $seconds): bool
        {
            return $this->inner->put($key, $value, $seconds);
        }

        /** @param array<string, mixed> $values */
        public function putMany(array $values, $seconds): bool
        {
            return $this->inner->putMany($values, $seconds);
        }

        public function increment($key, $value = 1): int|bool
        {
            return $this->inner->increment($key, $value);
        }

        public function decrement($key, $value = 1): int|bool
        {
            return $this->inner->decrement($key, $value);
        }

        public function forever($key, $value): bool
        {
            return $this->inner->forever($key, $value);
        }

        public function forget($key): bool
        {
            return $this->inner->forget($key);
        }

        public function flush(): bool
        {
            return $this->inner->flush();
        }

        public function getPrefix(): string
        {
            return '';
        }
    };
    Cache::extend('cacheaside_plain', fn () => Cache::repository($plain));
    config(['cache.stores.cacheaside_plain' => ['driver' => 'cacheaside_plain'], 'cacheaside.store' => 'cacheaside_plain']);
    $key = CacheKey::make('blog', 'plain');
    $loader = countingLoader('v');

    cacheAside()->remember($key, 60, $loader(...));
    cacheAside()->remember($key, 60, $loader(...));
    expect($loader->calls)->toBe(1);

    CacheStores::use('array');
    config(['cacheaside.lock.enabled' => false]);
    $store = app(NamespaceVersions::class)->store();
    $store->lock('rt:blog:v1:unlocked:lock', 30)->get();

    expect(cacheAside()->remember(CacheKey::make('blog', 'unlocked'), 60, function () use ($store): string {
        expect($store->lock('rt:blog:v1:unlocked:lock', 5)->get())->toBeFalse(); // still the foreign holder

        return 'v';
    }))->toBe('v');
});
