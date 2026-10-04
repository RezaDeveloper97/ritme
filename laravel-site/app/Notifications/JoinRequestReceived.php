<?php

declare(strict_types=1);

namespace App\Notifications;

use App\Domain\Directory\Join\Models\JoinRequest;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\Messages\MailMessage;
use Illuminate\Notifications\Notification;
use Illuminate\Queue\SerializesModels;

/**
 * "New join request" mail to the partnership mailbox (SubmitJoinRequest). Queued after commit like
 * ContactMessageReceived. Data-minimal: place name, tracking code and photo count — the contact person's name,
 * mobile and email stay in the admin panel (review: L5-06).
 */
final class JoinRequestReceived extends Notification implements ShouldQueue
{
    use Queueable, SerializesModels;

    /** The request may be deleted in the panel before the queue runs: then there is nothing to announce. */
    public bool $deleteWhenMissingModels = true;

    public int $tries = 3;

    public function __construct(public readonly JoinRequest $joinRequest)
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
        return (new MailMessage)
            ->subject('درخواست تازه ثبت مجموعه: '.$this->joinRequest->name)
            ->greeting('درخواست تازه برای فهرست مادر و کودک')
            ->line('مجموعه: '.$this->joinRequest->name)
            ->line('کد پیگیری: '.$this->joinRequest->code)
            ->line('تعداد عکس: '.$this->joinRequest->photos()->count())
            ->line('اطلاعات تماس و جزئیات درخواست فقط در پنل مدیریت دیده می‌شود.')
            ->salutation('ریتمی');
    }
}
