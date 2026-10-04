<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Enums;

use App\Domain\Seo\Schema\Enums\ItemAvailability;

/**
 * Availability of a product. `InStock`/`OutOfStock` follow the stock quantity automatically (product observer /
 * RecalculateProductStock); `PreOrder` and `BackOrder` are set by hand and kept whatever the quantity.
 */
enum StockStatus: string
{
    case InStock = 'in_stock';
    case OutOfStock = 'out_of_stock';
    case PreOrder = 'preorder';
    case BackOrder = 'backorder';

    public static function forQuantity(int $quantity, ?self $current = null): self
    {
        if ($current === self::PreOrder || $current === self::BackOrder) {
            return $current;
        }

        return $quantity > 0 ? self::InStock : self::OutOfStock;
    }

    /** Whether the product can be added to the cart. */
    public function isPurchasable(): bool
    {
        return $this !== self::OutOfStock;
    }

    public function label(): string
    {
        return match ($this) {
            self::InStock => 'موجود',
            self::OutOfStock => 'ناموجود',
            self::PreOrder => 'پیش‌سفارش',
            self::BackOrder => 'تأمین پس از سفارش',
        };
    }

    public function availability(): ItemAvailability
    {
        return match ($this) {
            self::InStock => ItemAvailability::InStock,
            self::OutOfStock => ItemAvailability::OutOfStock,
            self::PreOrder => ItemAvailability::PreOrder,
            self::BackOrder => ItemAvailability::BackOrder,
        };
    }
}
