<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Queries;

use App\Domain\Shop\Ordering\Data\OrderData;
use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Ordering\Support\CheckoutSession;
use App\Domain\Shop\Ordering\Support\OrderCode;

/**
 * Order lookups: by public code (case-insensitive, overlong input cut by OrderCode::normalize) for the order page, and
 * by checkout token for a repeated submit. Uncached — orders are private, transactional data.
 */
final class FindOrder
{
    public function byCode(string $code): ?OrderData
    {
        $order = Order::query()->where('code', OrderCode::normalize($code))->with('items')->first();

        return $order === null ? null : OrderData::fromModel($order);
    }

    public function byToken(string $token): ?Order
    {
        return $token === '' ? null : Order::query()->where('idempotency_key', CheckoutSession::key($token))->first();
    }
}
