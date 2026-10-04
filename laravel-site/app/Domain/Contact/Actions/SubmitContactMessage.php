<?php

declare(strict_types=1);

namespace App\Domain\Contact\Actions;

use App\Domain\Contact\Data\ContactMessageData;
use App\Domain\Contact\Models\ContactMessage;
use App\Domain\Contact\Support\ContactRecipients;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Notifications\ContactMessageReceived;
use Illuminate\Contracts\Notifications\Dispatcher;
use Illuminate\Notifications\AnonymousNotifiable;
use Psr\Log\LoggerInterface;
use Throwable;

/**
 * Stores a contact form message in the admin inbox (unread) and queues the "new message" mail to the mailbox the
 * topic belongs to (ContactRecipients, from settings). Notification problems — no valid mailbox configured, a queue
 * or mail failure — are logged/reported, never shown: the visitor's message is already safe in the inbox.
 */
final class SubmitContactMessage
{
    public function __construct(
        private readonly SettingsRepository $settings,
        private readonly Dispatcher $notifications,
        private readonly LoggerInterface $log,
    ) {}

    public function handle(ContactMessageData $data): ContactMessage
    {
        $message = ContactMessage::query()->create([
            'topic' => $data->topic,
            'name' => $data->name,
            'email' => $data->channel->email,
            'phone' => $data->channel->phone,
            'message' => $data->message,
        ]);

        $this->notify($message);

        return $message;
    }

    private function notify(ContactMessage $message): void
    {
        $recipient = ContactRecipients::for($message->topic, $this->settings->all());
        if ($recipient === null) {
            $this->log->warning('Contact message stored without notification: no valid support email in settings.', ['id' => $message->id]);

            return;
        }

        try {
            $this->notifications->send((new AnonymousNotifiable)->route('mail', $recipient), new ContactMessageReceived($message));
        } catch (Throwable $e) {
            report($e);
        }
    }
}
