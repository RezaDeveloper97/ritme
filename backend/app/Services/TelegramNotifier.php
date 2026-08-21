<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use Throwable;

/**
 * Sends operational notifications to a Telegram chat (admin/ops channel).
 *
 * Best-effort by design: a Telegram outage must never fail the request that
 * triggered the notification, so every failure is swallowed and logged.
 * Disabled (no-op) unless both TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are set.
 */
class TelegramNotifier
{
    public function isEnabled(): bool
    {
        return filled(config('services.telegram.token'))
            && filled(config('services.telegram.chat_id'));
    }

    /**
     * @param  string  $text  HTML-formatted message body.
     */
    public function send(string $text): bool
    {
        if (! $this->isEnabled()) {
            return false;
        }

        $token = config('services.telegram.token');

        try {
            $response = Http::timeout((int) config('services.telegram.timeout', 5))
                ->post("https://api.telegram.org/bot{$token}/sendMessage", [
                    'chat_id' => config('services.telegram.chat_id'),
                    'text' => $text,
                    'parse_mode' => 'HTML',
                    'disable_web_page_preview' => true,
                ]);

            if ($response->successful()) {
                return true;
            }

            Log::warning('Telegram notification failed', ['status' => $response->status()]);
        } catch (Throwable $e) {
            Log::warning('Telegram notification error', ['message' => $e->getMessage()]);
        }

        return false;
    }
}
