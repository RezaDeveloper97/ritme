<?php

declare(strict_types=1);

namespace App\Support\Cache;

use InvalidArgumentException;

/**
 * A key inside a cache namespace, e.g. CacheKey::make('blog', 'post', $slug) => "post:{slug}".
 * Long keys are hashed so they fit the database store's key column.
 */
final class CacheKey
{
    public const MAX_KEY_LENGTH = 120;

    private function __construct(
        public readonly string $namespace,
        public readonly string $key,
    ) {}

    public static function make(string $namespace, string|int ...$parts): self
    {
        if (preg_match('/^[a-z][a-z0-9_-]*$/', $namespace) !== 1) {
            throw new InvalidArgumentException("Invalid cache namespace [{$namespace}].");
        }

        if ($parts === []) {
            throw new InvalidArgumentException('A cache key needs at least one part.');
        }

        $strings = [];
        foreach ($parts as $part) {
            $part = (string) $part;
            if ($part === '') {
                throw new InvalidArgumentException('A cache key part may not be empty.');
            }
            $strings[] = $part;
        }

        $key = implode(':', $strings);

        if (strlen($key) > self::MAX_KEY_LENGTH) {
            $key = 'h:'.sha1($key);
        }

        return new self($namespace, $key);
    }
}
