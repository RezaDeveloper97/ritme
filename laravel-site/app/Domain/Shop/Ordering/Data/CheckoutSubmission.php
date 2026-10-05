<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Data;

use App\Domain\Shop\Ordering\Enums\DeliveryWindow;
use Carbon\CarbonImmutable;

/**
 * A validated checkout form (CheckoutRequest). Only the recipient's minimal data + delivery preferences; prices and
 * items never come from the form — they are re-read from the live cart. `token` = the form's one-time idempotency
 * token, `signature` = the CartSignature of the cart the shopper saw (refuse if it changed meanwhile).
 */
final readonly class CheckoutSubmission
{
    public function __construct(
        public string $recipientName,
        public string $mobile,
        public string $province,
        public string $city,
        public string $address,
        public ?string $postalCode,
        public ?string $note,
        public CarbonImmutable $deliveryDate,
        public DeliveryWindow $deliveryWindow,
        public bool $discreetPackaging,
        public string $token,
        public string $signature,
    ) {}
}
