<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

/**
 * Coercion helpers for SettingsGroupData::fromArray(). Stored values are admin input, so every read is defensive.
 */
trait CoercesSettingValues
{
    /**
     * @param  array<string, mixed>  $values
     */
    private static function string(array $values, string $key, string $default = ''): string
    {
        $value = $values[$key] ?? null;

        return is_scalar($value) ? trim((string) $value) : $default;
    }

    /**
     * @param  array<string, mixed>  $values
     */
    private static function nullableString(array $values, string $key, ?string $default = null): ?string
    {
        $value = array_key_exists($key, $values) ? self::string($values, $key) : $default;

        return $value === '' ? null : $value;
    }

    /**
     * @param  array<string, mixed>  $values
     */
    private static function bool(array $values, string $key, bool $default = false): bool
    {
        $value = $values[$key] ?? null;

        return match (true) {
            is_bool($value) => $value,
            is_int($value) => $value === 1,
            is_string($value) => in_array(strtolower(trim($value)), ['1', 'true', 'on', 'yes'], true),
            default => $default,
        };
    }

    /**
     * @param  array<string, mixed>  $values
     */
    private static function nullableInt(array $values, string $key): ?int
    {
        $value = $values[$key] ?? null;

        return is_numeric($value) && (int) $value > 0 ? (int) $value : null;
    }

    /**
     * @param  array<string, mixed>  $values
     * @param  list<string>  $default
     * @return list<string>
     */
    private static function stringList(array $values, string $key, array $default = []): array
    {
        if (! array_key_exists($key, $values)) {
            return $default;
        }

        $list = [];
        foreach (is_array($values[$key]) ? $values[$key] : [] as $item) {
            if (is_scalar($item) && trim((string) $item) !== '') {
                $list[] = trim((string) $item);
            }
        }

        return array_values(array_unique($list));
    }

    /**
     * @param  array<string, mixed>  $values
     * @return array<string, string>
     */
    private static function stringMap(array $values, string $key): array
    {
        $map = [];
        foreach (is_array($values[$key] ?? null) ? $values[$key] : [] as $name => $item) {
            if (is_string($name) && $name !== '' && is_scalar($item) && trim((string) $item) !== '') {
                $map[$name] = trim((string) $item);
            }
        }

        return $map;
    }
}
