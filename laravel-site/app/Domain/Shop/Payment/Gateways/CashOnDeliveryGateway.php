<?php

declare(strict_types=1);

namespace App\Domain\Shop\Payment\Gateways;

use App\Domain\Shop\Payment\Contracts\PaymentGateway;
use App\Domain\Shop\Payment\Data\PaymentStart;
use App\Domain\Shop\Payment\Enums\PaymentMethod;
use App\Domain\Shop\Payment\Enums\PaymentStatus;
use App\Support\Money\Money;

/**
 * Cash on delivery: nothing happens online. Totals above the COD cap (`ShopSettings.cod_max_amount`, from config
 * until L6-06) are refused; the order starts `unpaid`.
 */
final readonly class CashOnDeliveryGateway implements PaymentGateway
{
    public function __construct(private ?Money $maxAmount = null) {}

    /** Cap in tomans from config (null, non-numeric or ≤ 0 = no cap). */
    public static function fromToman(mixed $maxAmount): self
    {
        return new self(is_numeric($maxAmount) && (int) $maxAmount > 0 ? Money::fromToman((int) $maxAmount) : null);
    }

    public function method(): PaymentMethod
    {
        return PaymentMethod::CashOnDelivery;
    }

    public function limit(): ?Money
    {
        return $this->maxAmount;
    }

    public function accepts(Money $total): bool
    {
        return $this->maxAmount === null || ! $total->greaterThan($this->maxAmount);
    }

    public function start(string $orderCode, Money $total): PaymentStart
    {
        return new PaymentStart(PaymentStatus::Unpaid);
    }
}
