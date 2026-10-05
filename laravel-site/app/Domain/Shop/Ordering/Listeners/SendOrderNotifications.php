<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Listeners;

use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Support\ContactRecipients;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Shop\Ordering\Events\OrderPlaced;
use App\Notifications\OrderPlacedForCustomer;
use App\Notifications\OrderPlacedForTeam;
use Illuminate\Contracts\Notifications\Dispatcher;
use Illuminate\Notifications\AnonymousNotifiable;
use Psr\Log\LoggerInterface;
use Throwable;

/**
 * On OrderPlaced (after commit): queues a data-minimal mail to the team mailbox (support email) and an SMS to the
 * customer's mobile through the SmsSender contract (log driver until a provider is bound). Problems are logged and
 * reported, never shown — the order is already stored.
 */
final class SendOrderNotifications
{
    public function __construct(
        private readonly SettingsRepository $settings,
        private readonly Dispatcher $notifications,
        private readonly LoggerInterface $log,
    ) {}

    public function handle(OrderPlaced $event): void
    {
        $order = $event->order;

        try {
            $team = ContactRecipients::for(ContactTopic::Support, $this->settings->all());
            if ($team === null) {
                $this->log->warning('Order stored without team notification: no valid support email in settings.', ['id' => $order->id]);
            } else {
                $this->notifications->send((new AnonymousNotifiable)->route('mail', $team), new OrderPlacedForTeam($order));
            }

            $this->notifications->send((new AnonymousNotifiable)->route('sms', $order->mobile), new OrderPlacedForCustomer($order));
        } catch (Throwable $e) {
            report($e);
        }
    }
}
