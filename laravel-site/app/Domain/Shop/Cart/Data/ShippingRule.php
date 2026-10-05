<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

use App\Support\Money\Money;

/**
 * Shipping as shown in the cart (an estimate; checkout L6-05 has the final word): an optional flat fee and an optional
 * free-shipping threshold. Neither set → the fee is «calculated at the next step» and no free-shipping bar is shown.
 */
final readonly class ShippingRule
{
    public function __construct(public ?Money $flatFee = null, public ?Money $freeOver = null) {}

    public static function fromToman(mixed $flatFee, mixed $freeOver): self
    {
        $money = static fn (mixed $v): ?Money => is_numeric($v) && (int) $v >= 0 ? Money::fromToman((int) $v) : null;
        $free = $money($freeOver);

        return new self($money($flatFee), $free !== null && $free->isPositive() ? $free : null);
    }

    public function quote(Money $subtotal): ShippingQuote
    {
        if ($this->freeOver !== null && ! $subtotal->lessThan($this->freeOver)) {
            return new ShippingQuote(Money::zero(), true, $this->freeOver, Money::zero(), 100);
        }

        $remaining = $this->freeOver?->minus($subtotal);
        $progress = $this->freeOver === null || $subtotal->rial <= 0 ? 0 : intdiv($subtotal->rial * 100, $this->freeOver->rial);

        return new ShippingQuote($this->flatFee, false, $this->freeOver, $remaining, min(99, max(0, $progress)));
    }
}
