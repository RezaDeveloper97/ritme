<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

/**
 * Raises PHP's memory_limit for image work (never lowers it; -1 stays unlimited).
 */
final class MemoryLimit
{
    public static function raise(string $limit): void
    {
        $current = self::bytes((string) ini_get('memory_limit'));
        $wanted = self::bytes($limit);

        if ($current !== -1 && ($wanted === -1 || $wanted > $current)) {
            ini_set('memory_limit', $limit);
        }
    }

    public static function bytes(string $value): int
    {
        $value = trim($value);
        if ($value === '' || $value === '-1') {
            return -1;
        }

        $number = (int) $value;

        return match (strtolower(substr($value, -1))) {
            'g' => $number * 1024 ** 3,
            'm' => $number * 1024 ** 2,
            'k' => $number * 1024,
            default => $number,
        };
    }
}
