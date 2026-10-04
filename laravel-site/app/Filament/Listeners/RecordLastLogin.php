<?php

declare(strict_types=1);

namespace App\Filament\Listeners;

use App\Filament\Http\Middleware\EnforceSessionTimeout;
use App\Models\User;
use Illuminate\Auth\Events\Login;

/**
 * Stamps `last_login_at` (quietly: no model events, so no activity-log entry) and starts the admin inactivity clock.
 */
final class RecordLastLogin
{
    public function handle(Login $event): void
    {
        if (! $event->user instanceof User) {
            return;
        }

        $event->user->forceFill(['last_login_at' => now()])->saveQuietly();

        if (app()->bound('session.store')) {
            session()->put(EnforceSessionTimeout::SESSION_KEY, now()->getTimestamp());
        }
    }
}
