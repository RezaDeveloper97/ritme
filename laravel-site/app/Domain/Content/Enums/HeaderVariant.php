<?php

declare(strict_types=1);

namespace App\Domain\Content\Enums;

/**
 * Site header look (docs/AUDIT.md §2.1): `Dark` sits on the night hero block (the hero shares its background),
 * `Light` is white with a bottom border.
 */
enum HeaderVariant: string
{
    case Dark = 'dark';
    case Light = 'light';

    public static function resolve(self|string|null $value, self $default = self::Light): self
    {
        if ($value instanceof self) {
            return $value;
        }

        return ($value === null ? null : self::tryFrom($value)) ?? $default;
    }

    public function isDark(): bool
    {
        return $this === self::Dark;
    }
}
