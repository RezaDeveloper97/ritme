<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

/**
 * Shop business settings (L6-06), edited on the admin «تنظیمات فروشگاه» page. Amounts are in TOMANS (as typed by the
 * shop manager); Money converts them to rials where they are used. Null = not set → the `config('shop.*')` value
 * (ShopServiceProvider), and when that is unset too: no flat fee / no free-shipping bar / no COD cap.
 *
 * - `shippingFlatFee`: flat shipping fee (0 = always free shipping)
 * - `freeShippingOver`: subtotal from which shipping is free
 * - `codMaxAmount`: highest order total accepted for cash on delivery
 * - `lowStockThreshold`: the admin low-stock widget lists products / variants with at most this many units left
 */
final readonly class ShopSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    public const DEFAULT_LOW_STOCK = 3;

    public const MAX_AMOUNT = 4_000_000_000;

    public function __construct(
        public ?int $shippingFlatFee = null,
        public ?int $freeShippingOver = null,
        public ?int $codMaxAmount = null,
        public int $lowStockThreshold = self::DEFAULT_LOW_STOCK,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Shop;
    }

    public static function fromArray(array $values): static
    {
        $threshold = $values['low_stock_threshold'] ?? null;

        return new self(
            shippingFlatFee: self::amount($values, 'shipping_flat_fee', allowZero: true),
            freeShippingOver: self::amount($values, 'free_shipping_over'),
            codMaxAmount: self::amount($values, 'cod_max_amount'),
            lowStockThreshold: is_numeric($threshold) && (int) $threshold >= 0 && (int) $threshold <= 1000
                ? (int) $threshold
                : self::DEFAULT_LOW_STOCK,
        );
    }

    public function toArray(): array
    {
        return [
            'shipping_flat_fee' => $this->shippingFlatFee,
            'free_shipping_over' => $this->freeShippingOver,
            'cod_max_amount' => $this->codMaxAmount,
            'low_stock_threshold' => $this->lowStockThreshold,
        ];
    }

    /**
     * A toman amount or null (missing, empty, non-numeric, negative, above the cap; 0 only where it means something).
     *
     * @param  array<string, mixed>  $values
     */
    private static function amount(array $values, string $key, bool $allowZero = false): ?int
    {
        $value = $values[$key] ?? null;
        if (! is_numeric($value)) {
            return null;
        }

        $amount = (int) $value;

        return ($amount > 0 || ($allowZero && $amount === 0)) && $amount <= self::MAX_AMOUNT ? $amount : null;
    }
}
