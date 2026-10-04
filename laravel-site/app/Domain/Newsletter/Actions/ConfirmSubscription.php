<?php

declare(strict_types=1);

namespace App\Domain\Newsletter\Actions;

use App\Domain\Newsletter\Enums\SubscriptionStatus;
use App\Domain\Newsletter\Models\Subscriber;

/**
 * The confirm link of the double opt-in mail. Returns the resulting status, or null for an unknown token. An
 * unsubscribed address is never re-activated by an old link (a new sign-up rotates the token).
 */
final class ConfirmSubscription
{
    public function handle(string $token): ?SubscriptionStatus
    {
        $subscriber = $this->find($token);
        if ($subscriber === null) {
            return null;
        }

        if ($subscriber->status() === SubscriptionStatus::Pending) {
            $subscriber->forceFill(['confirmed_at' => now()])->save();
        }

        return $subscriber->status();
    }

    private function find(string $token): ?Subscriber
    {
        return $token === '' || strlen($token) > 64 ? null : Subscriber::query()->where('token', $token)->first();
    }
}
