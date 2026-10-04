<?php

declare(strict_types=1);

namespace App\Domain\Newsletter\Actions;

use App\Domain\Newsletter\Models\Subscriber;

/**
 * Unsubscribe link (also the RFC 8058 one-click target). Idempotent; false for an unknown token.
 */
final class Unsubscribe
{
    public function handle(string $token): bool
    {
        $subscriber = $token === '' || strlen($token) > 64 ? null : Subscriber::query()->where('token', $token)->first();
        if ($subscriber === null) {
            return false;
        }

        if ($subscriber->unsubscribed_at === null) {
            $subscriber->forceFill(['unsubscribed_at' => now()])->save();
        }

        return true;
    }
}
