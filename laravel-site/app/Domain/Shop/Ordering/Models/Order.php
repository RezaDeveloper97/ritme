<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Models;

use App\Domain\Shop\Ordering\Enums\DeliveryWindow;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Payment\Enums\PaymentMethod;
use App\Domain\Shop\Payment\Enums\PaymentStatus;
use App\Support\Money\Money;
use App\Support\Money\MoneyCast;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * A shop order (L6-05), written by PlaceOrder only; managed in the admin (L6-06). Money columns are integer rials.
 *
 * @property int $id
 * @property string $code
 * @property string $idempotency_key
 * @property OrderStatus $status
 * @property PaymentMethod $payment_method
 * @property PaymentStatus $payment_status
 * @property Money $subtotal
 * @property Money|null $shipping_fee
 * @property Money $total
 * @property int $items_count
 * @property string $recipient_name
 * @property string $mobile
 * @property string $province
 * @property string $city
 * @property string $address
 * @property string|null $postal_code
 * @property string|null $note
 * @property Carbon $delivery_date
 * @property DeliveryWindow $delivery_window
 * @property bool $discreet_packaging
 * @property bool $is_demo
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Collection<int, OrderItem> $items
 */
final class Order extends Model
{
    protected $table = 'shop_orders';

    protected $fillable = [
        'code', 'idempotency_key', 'status', 'payment_method', 'payment_status', 'subtotal', 'shipping_fee', 'total',
        'items_count', 'recipient_name', 'mobile', 'province', 'city', 'address', 'postal_code', 'note',
        'delivery_date', 'delivery_window', 'discreet_packaging', 'is_demo',
    ];

    protected $attributes = [
        'status' => 'pending',
        'payment_status' => 'unpaid',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'status' => OrderStatus::class,
            'payment_method' => PaymentMethod::class,
            'payment_status' => PaymentStatus::class,
            'subtotal' => MoneyCast::class,
            'shipping_fee' => MoneyCast::class,
            'total' => MoneyCast::class,
            'items_count' => 'integer',
            'delivery_date' => 'date',
            'delivery_window' => DeliveryWindow::class,
            'discreet_packaging' => 'boolean',
            'is_demo' => 'boolean',
        ];
    }

    /**
     * @return HasMany<OrderItem, $this>
     */
    public function items(): HasMany
    {
        return $this->hasMany(OrderItem::class)->orderBy('id');
    }
}
