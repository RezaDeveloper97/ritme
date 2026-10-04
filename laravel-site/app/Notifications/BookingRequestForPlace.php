<?php

declare(strict_types=1);

namespace App\Notifications;

use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Booking\Support\ChildAge;
use App\Notifications\Channels\SmsChannel;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\Notification;
use Illuminate\Queue\SerializesModels;

/**
 * SMS to the place's own mobile (first mobile among its phones) about a new request, through the SmsSender contract
 * (log driver by default). Carries what the place needs to call back — exactly the data the booking panel promises
 * to share: the parent's name, mobile and the child's age, plus service and preferred day/window.
 */
final class BookingRequestForPlace extends Notification implements ShouldQueue
{
    use Queueable, SerializesModels;

    public bool $deleteWhenMissingModels = true;

    public int $tries = 3;

    public function __construct(public readonly BookingRequest $booking)
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
        $booking = $this->booking;

        return implode("\n", array_filter([
            'ریتمی — درخواست رزرو تازه',
            $booking->service_name,
            jdate($booking->preferred_date, 'l j F').' · '.$booking->time_window->describe(),
            $booking->parent_name.' · '.$booking->mobile,
            $booking->child_age_months === null ? null : 'سن کودک: '.ChildAge::label($booking->child_age_months),
            'کد: '.$booking->code,
        ]));
    }
}
