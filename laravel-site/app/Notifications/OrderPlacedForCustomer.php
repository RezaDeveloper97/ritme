<?php

declare(strict_types=1);

namespace App\Notifications;

use App\Domain\Shop\Ordering\Models\Order;
use App\Notifications\Channels\SmsChannel;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\Notification;
use Illuminate\Queue\SerializesModels;

/**
 * SMS to the customer after an order (SmsSender contract, log driver by default): code, amount due on delivery and
 * the order page link. Never says "paid" — cash on delivery. No product names (discreet packaging, shared phones).
 */
final class OrderPlacedForCustomer extends Notification implements ShouldQueue
{
    use Queueable, SerializesModels;

    public bool $deleteWhenMissingModels = true;

    public int $tries = 3;

    public function __construct(public readonly Order $order)
    {
        $this->afterCommit();
    }

    /**
     * @return list<class-string>
     */
    public function via(object $notifiable): array
    {
        return [SmsChannel::class];
    }

    public function toSms(object $notifiable): string
    {
        $order = $this->order;

        return implode("\n", [
            'ریتمی — سفارشت ثبت شد',
            'کد سفارش: '.$order->code,
            'مبلغ هنگام تحویل: '.$order->total->format().($order->shipping_fee === null ? ' + هزینه ارسال' : ''),
            'برای تأیید زمان تحویل با تو تماس می‌گیریم.',
            route('shop.order', [$order->code]),
        ]);
    }
}
