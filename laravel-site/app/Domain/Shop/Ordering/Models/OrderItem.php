<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Models;

use App\Support\Money\Money;
use App\Support\Money\MoneyCast;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * One line of an order: a snapshot of the product at checkout (title, variant label, unit price). `stock_tracked`
 * false = pre-order / back-order line whose stock was not decremented (nothing to return on cancellation).
 *
 * @property int $id
 * @property int $order_id
 * @property int|null $product_id
 * @property int|null $variant_id
 * @property string $title
 * @property string|null $variant_label
 * @property Money $unit_price
 * @property int $quantity
 * @property Money $line_total
 * @property int|null $cover_media_id
 * @property string|null $illustration
 * @property bool $stock_tracked
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Order $order
 */
final class OrderItem extends Model
{
    protected $table = 'shop_order_items';

    protected $fillable = [
        'order_id', 'product_id', 'variant_id', 'title', 'variant_label', 'unit_price', 'quantity', 'line_total',
        'cover_media_id', 'illustration', 'stock_tracked',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'unit_price' => MoneyCast::class,
            'line_total' => MoneyCast::class,
            'quantity' => 'integer',
            'cover_media_id' => 'integer',
            'stock_tracked' => 'boolean',
        ];
    }

    /**
     * @return BelongsTo<Order, $this>
     */
    public function order(): BelongsTo
    {
        return $this->belongsTo(Order::class);
    }
}
