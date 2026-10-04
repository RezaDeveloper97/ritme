<?php

declare(strict_types=1);

namespace App\Notifications;

use App\Domain\Contact\Models\ContactMessage;
use App\Filament\Resources\ContactMessages\ContactMessageResource;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\Messages\MailMessage;
use Illuminate\Notifications\Notification;
use Illuminate\Queue\SerializesModels;

/**
 * "New contact message" mail to the team mailbox (SubmitContactMessage → ContactRecipients). Queued on the database
 * queue (cPanel cron drives `schedule:run`/the worker; mail driver `log` by default). Data-minimal: topic, sender name
 * and a link to the admin inbox — the message text and the sender's email/phone stay in the panel, not in mailboxes.
 */
final class ContactMessageReceived extends Notification implements ShouldQueue
{
    use Queueable, SerializesModels;

    /** The message may be deleted in the panel before the queue runs: then there is nothing to announce. */
    public bool $deleteWhenMissingModels = true;

    public int $tries = 3;

    public function __construct(public readonly ContactMessage $contactMessage)
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
        $topic = $this->contactMessage->topic->label();

        return (new MailMessage)
            ->subject('پیام تازه از فرم تماس: '.$topic)
            ->greeting('پیام تازه در صندوق تماس')
            ->line('موضوع: '.$topic)
            ->line('فرستنده: '.$this->contactMessage->name)
            ->line('متن پیام و راه پاسخ فقط در پنل مدیریت دیده می‌شود.')
            ->action('دیدن پیام', ContactMessageResource::getUrl('view', ['record' => $this->contactMessage], panel: 'admin'))
            ->salutation('ریتمی');
    }
}
