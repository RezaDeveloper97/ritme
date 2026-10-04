<?php

declare(strict_types=1);

namespace App\Notifications\Channels;

use App\Domain\Directory\Booking\Contracts\SmsSender;
use Illuminate\Notifications\Notification;

/**
 * Notification channel over the SmsSender contract: the notifiable's `sms` route is the mobile, the notification's
 * `toSms()` returns the text. Notifications use it as `via() → [SmsChannel::class]`.
 */
final readonly class SmsChannel
{
    public function __construct(private SmsSender $sender) {}

    public function send(object $notifiable, Notification $notification): void
    {
        $mobile = method_exists($notifiable, 'routeNotificationFor') ? $notifiable->routeNotificationFor('sms', $notification) : null;
        if (! is_string($mobile) || $mobile === '' || ! method_exists($notification, 'toSms')) {
            return;
        }

        $text = $notification->toSms($notifiable);
        if (is_string($text) && $text !== '') {
            $this->sender->send($mobile, $text);
        }
    }
}
