<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\IndexNow;

/**
 * IndexNow keys: 8–128 characters of a-z, A-Z, 0-9 and "-" (spec). Ours are 32 random hex characters.
 */
final class IndexNowKey
{
    public const PATTERN = '[A-Za-z0-9-]{8,128}';

    public static function generate(): string
    {
        return bin2hex(random_bytes(16));
    }

    public static function isValid(?string $key): bool
    {
        return $key !== null && preg_match('/^'.self::PATTERN.'$/', $key) === 1;
    }
}
