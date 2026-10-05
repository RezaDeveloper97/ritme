<?php

declare(strict_types=1);

namespace App\Notifications;

use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Ordering\Support\IranProvinces;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\Messages\MailMessage;
use Illuminate\Notifications\Notification;
use Illuminate\Queue\SerializesModels;

/**
 * "New order" mail to the team mailbox (SendOrderNotifications → support email). Queued after commit. Data-minimal:
 * code, items count, amount due on delivery, province/city and the preferred slot — the recipient's name, mobile and
 * address stay in the admin panel (L6-06).
 */
final class OrderPlacedForTeam extends Notification implements ShouldQueue
{
    use Queueable, SerializesModels;

    public bool $deleteWhenMissingModels = true;

    public int $tries = 3;

    public function __construct(public readonly Order $order)
    {
        $this->afterCommit();
    }

    /**
     * @return list<string>
     */
    public function via(object $notifiable): array
    {
        return ['mail'];
    }

    public function toMail(object $notifiable): MailMessage
    {
        $order = $this->order;

        return (new MailMessage)
            ->subject('سفارش تازه در فروشگاه: '.$order->code)
            ->greeting('سفارش تازه (پرداخت در محل)')
            ->line('کد سفارش: '.$order->code.($order->is_demo ? ' (نمونه نمایشی)' : ''))
            ->line('تعداد کالا: '.fa_digits($order->items_count))
            ->line('مبلغ قابل دریافت هنگام تحویل: '.$order->total->format().($order->shipping_fee === null ? ' + هزینه ارسال' : ''))
            ->line('مقصد: '.IranProvinces::name($order->province).' · '.$order->city)
            ->line('زمان دلخواه تحویل: '.jdate($order->delivery_date, 'l j F').' · '.$order->delivery_window->hours())
            ->line($order->discreet_packaging ? 'بسته‌بندی ساده: بدون نام محصول روی جعبه و فاکتور بیرونی.' : 'بسته‌بندی معمولی.')
            ->line('نام، شماره و آدرس گیرنده فقط در پنل مدیریت دیده می‌شود.')
            ->salutation('ریتمی');
    }
}
