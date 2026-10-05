<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Data;

use App\Domain\Directory\Booking\Support\MobileMask;
use App\Domain\Shop\Ordering\Enums\DeliveryWindow;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Ordering\Models\OrderItem;
use App\Domain\Shop\Ordering\Support\IranProvinces;
use App\Domain\Shop\Payment\Enums\PaymentMethod;
use App\Domain\Shop\Payment\Enums\PaymentStatus;
use App\Support\Money\Money;
use Carbon\CarbonImmutable;

/**
 * The order as its public page shows it — already reduced to what a shared link may reveal: masked mobile, province +
 * city only (no street address, postal code or note), recipient name.
 */
final readonly class OrderData
{
    /**
     * @param  list<OrderItemData>  $items
     */
    public function __construct(
        public string $code,
        public OrderStatus $status,
        public PaymentMethod $paymentMethod,
        public PaymentStatus $paymentStatus,
        public Money $subtotal,
        public ?Money $shippingFee,
        public Money $total,
        public int $itemsCount,
        public string $recipientName,
        public string $maskedMobile,
        public string $place,
        public CarbonImmutable $deliveryDate,
        public DeliveryWindow $deliveryWindow,
        public bool $discreetPackaging,
        public bool $isDemo,
        public CarbonImmutable $placedAt,
        public array $items,
    ) {}

    /** Expects `items` loaded. */
    public static function fromModel(Order $order): self
    {
        return new self(
            code: $order->code,
            status: $order->status,
            paymentMethod: $order->payment_method,
            paymentStatus: $order->payment_status,
            subtotal: $order->subtotal,
            shippingFee: $order->shipping_fee,
            total: $order->total,
            itemsCount: $order->items_count,
            recipientName: $order->recipient_name,
            maskedMobile: MobileMask::mask($order->mobile),
            place: IranProvinces::name($order->province).' · '.$order->city,
            deliveryDate: CarbonImmutable::parse($order->delivery_date->format('Y-m-d'), 'Asia/Tehran'),
            deliveryWindow: $order->delivery_window,
            discreetPackaging: $order->discreet_packaging,
            isDemo: $order->is_demo,
            placedAt: CarbonImmutable::instance($order->created_at ?? now()),
            items: $order->items->map(static fn (OrderItem $item): OrderItemData => OrderItemData::fromModel($item))->values()->all(),
        );
    }
}
