<?php

declare(strict_types=1);

namespace App\Domain\Newsletter\Actions;

use App\Domain\Newsletter\Enums\SubscribeOutcome;
use App\Domain\Newsletter\Mail\ConfirmSubscriptionMail;
use App\Domain\Newsletter\Models\Subscriber;
use Illuminate\Contracts\Mail\Mailer;
use Illuminate\Contracts\Routing\UrlGenerator;
use Throwable;

/**
 * Double opt-in sign-up: stores the address as pending (consent time + source page) and mails the confirm link.
 * A pending address gets at most one mail per RESEND_AFTER_MINUTES; an active one gets nothing. Mail failures are
 * reported, never shown (the visitor still sees the neutral "check your inbox" message).
 */
final class Subscribe
{
    public const RESEND_AFTER_MINUTES = 10;

    public function __construct(private readonly Mailer $mailer, private readonly UrlGenerator $url) {}

    public function handle(string $email, ?string $source = null): SubscribeOutcome
    {
        $email = Subscriber::normalizeEmail($email);
        $now = now();

        $subscriber = Subscriber::query()->createOrFirst(['email' => $email], [
            'source' => $source,
            'token' => Subscriber::newToken(),
            'consent_at' => $now,
        ]);

        if ($subscriber->wasRecentlyCreated) {
            $outcome = SubscribeOutcome::Created;
        } elseif ($subscriber->unsubscribed_at !== null) {
            $subscriber->forceFill([
                'source' => $source, 'token' => Subscriber::newToken(), 'consent_at' => $now,
                'confirmed_at' => null, 'unsubscribed_at' => null, 'confirmation_sent_at' => null,
            ])->save();
            $outcome = SubscribeOutcome::Resubscribed;
        } elseif ($subscriber->confirmed_at !== null) {
            return SubscribeOutcome::AlreadyActive;
        } elseif ($subscriber->confirmation_sent_at !== null && $subscriber->confirmation_sent_at->gt($now->copy()->subMinutes(self::RESEND_AFTER_MINUTES))) {
            return SubscribeOutcome::Throttled;
        } else {
            $subscriber->forceFill(['consent_at' => $now, 'source' => $source ?? $subscriber->source])->save();
            $outcome = SubscribeOutcome::Resent;
        }

        $this->sendConfirmation($subscriber);

        return $outcome;
    }

    private function sendConfirmation(Subscriber $subscriber): void
    {
        try {
            $this->mailer->to($subscriber->email)->send(new ConfirmSubscriptionMail(
                confirmUrl: $this->url->route('newsletter.confirm', [$subscriber->token]),
                unsubscribeUrl: $this->url->route('newsletter.unsubscribe', [$subscriber->token]),
            ));
            $subscriber->forceFill(['confirmation_sent_at' => now()])->save();
        } catch (Throwable $e) {
            report($e);
        }
    }
}
