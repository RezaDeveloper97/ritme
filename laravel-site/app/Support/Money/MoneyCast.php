<?php

declare(strict_types=1);

namespace App\Support\Money;

use Illuminate\Contracts\Database\Eloquent\CastsAttributes;
use Illuminate\Database\Eloquent\Model;

/**
 * Eloquent cast for an integer rial column: reads as Money (or null), writes Money, an int (rials) or a numeric string
 * (rials). Use `'price' => MoneyCast::class` in a model's casts(). Queries still compare the raw integer column.
 *
 * @implements CastsAttributes<Money|null, Money|int|string|null>
 */
final class MoneyCast implements CastsAttributes
{
    /**
     * @param  array<string, mixed>  $attributes
     */
    public function get(Model $model, string $key, mixed $value, array $attributes): ?Money
    {
        if ($value === null || $value === '') {
            return null;
        }

        return Money::parseRial(is_int($value) ? $value : (string) $value);
    }

    /**
     * @param  array<string, mixed>  $attributes
     */
    public function set(Model $model, string $key, mixed $value, array $attributes): ?int
    {
        if ($value === null || $value === '') {
            return null;
        }
        if ($value instanceof Money) {
            return $value->rial;
        }

        return Money::parseRial($value)->rial;
    }
}
