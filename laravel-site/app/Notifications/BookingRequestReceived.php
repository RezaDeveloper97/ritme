<?php

declare(strict_types=1);

namespace App\Notifications;

use App\Domain\Directory\Booking\Models\BookingRequest;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\Messages\MailMessage;
use Illuminate\Notifications\Notification;
use Illuminate\Queue\SerializesModels;

/**
 * "New booking request" mail to the team mailbox (CreateBookingRequest → support email). Queued after commit like
 * ContactMessageReceived. Data-minimal: place, service, preferred day/window and the code — the parent's name and
 * mobile stay in the admin panel (L5-06).
 */
final class BookingRequestReceived extends Notification implements ShouldQueue
{
    use Queueable, SerializesModels;

    /** The request may be deleted in the panel before the queue runs: then there is nothing to announce. */
    public bool $deleteWhenMissingModels = true;

    public int $tries = 3;

    public function __construct(public readonly BookingRequest $booking)
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
        $booking = $this->booking;

        return (new MailMessage)
            ->subject('درخواست رزرو تازه: '.$booking->place_name)
            ->greeting('درخواست رزرو تازه در فهرست مادر و کودک')
            ->line('مجموعه: '.$booking->place_name)
            ->line('خدمت: '.($booking->service_name ?? '—'))
            ->line('روز دلخواه: '.jdate($booking->preferred_date, 'l j F Y').' · '.$booking->time_window->describe())
            ->line('کد رزرو: '.$booking->code)
            ->line('نام و شماره تماس والد فقط در پنل مدیریت دیده می‌شود.')
            ->salutation('ریتمی');
    }
}
