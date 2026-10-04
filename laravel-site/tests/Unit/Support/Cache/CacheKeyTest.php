<?php

declare(strict_types=1);

use App\Support\Cache\CacheKey;

it('joins parts with colons', function (): void {
    $key = CacheKey::make('blog', 'post', 42, 'سلام');

    expect($key->namespace)->toBe('blog')
        ->and($key->key)->toBe('post:42:سلام');
});

it('hashes keys longer than the limit', function (): void {
    $long = str_repeat('a', CacheKey::MAX_KEY_LENGTH + 1);

    expect(CacheKey::make('blog', $long)->key)->toBe('h:'.sha1($long))
        ->and(CacheKey::make('blog', str_repeat('a', CacheKey::MAX_KEY_LENGTH))->key)->toHaveLength(CacheKey::MAX_KEY_LENGTH);
});

it('rejects invalid namespaces and empty keys', function (string $namespace, array $parts): void {
    CacheKey::make($namespace, ...$parts);
})->throws(InvalidArgumentException::class)->with([
    'bad namespace' => ['Blog Posts', ['x']],
    'empty namespace' => ['', ['x']],
    'no parts' => ['blog', []],
    'empty part' => ['blog', ['']],
]);
