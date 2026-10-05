<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Actions;

use App\Domain\Shop\Ordering\Models\Order;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Collection;
use InvalidArgumentException;
use Spatie\Activitylog\Models\Activity;

/**
 * Internal team notes on an order (admin, L6-06): «تماس گرفته شد، فردا عصر تحویل» … Never shown to the customer. Stored
 * as entries of the order's activity log (`shop`, `shop.order.note`) — append-only, with the author and the time, so
 * no extra table is needed. Plain text, trimmed, ≤ 1000 characters.
 */
final class AddOrderNote
{
    public const MAX_LENGTH = 1000;

    public const DESCRIPTION = 'shop.order.note';

    /**
     * @throws InvalidArgumentException on an empty note
     */
    public function handle(Order $order, string $note, ?Model $causer = null): Activity
    {
        $text = mb_substr(trim(strip_tags($note)), 0, self::MAX_LENGTH);
        if ($text === '') {
            throw new InvalidArgumentException('The note is empty.');
        }

        /** @var Activity $activity */
        $activity = activity(ChangeOrderStatus::LOG)
            ->causedBy($causer)
            ->performedOn($order)
            ->event('note')
            ->withProperties(['note' => $text])
            ->log(self::DESCRIPTION);

        return $activity;
    }

    /**
     * The order's notes and status changes, newest first (for the order page).
     *
     * @return Collection<int, Activity>
     */
    public static function history(Order $order): Collection
    {
        return Activity::query()
            ->with('causer')
            ->where('log_name', ChangeOrderStatus::LOG)
            ->where('subject_type', $order->getMorphClass())
            ->where('subject_id', $order->getKey())
            ->whereIn('description', [self::DESCRIPTION, 'shop.order.status'])
            ->latest('id')
            ->limit(100)
            ->get();
    }
}
