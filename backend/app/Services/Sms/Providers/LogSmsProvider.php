<?php

namespace App\Services\Sms\Providers;

use App\Services\Sms\Contracts\SmsProviderInterface;
use Illuminate\Support\Facades\Log;
use RuntimeException;

/**
 * Writes messages to the log instead of sending them (`SMS_PROVIDER=log`).
 *
 * For the contract stack and local development only: it refuses to exist in
 * production, so a mis-set SMS_PROVIDER there fails loudly instead of silently
 * dropping every login code. It is not a bypass — the OTP is still random and
 * is still read from the database (see .claude/skills/local-dev).
 */
class LogSmsProvider implements SmsProviderInterface
{
    public function __construct()
    {
        if (app()->environment('production')) {
            throw new RuntimeException('The log SMS provider cannot be used in production.');
        }
    }

    public function send(string $mobile, string $message): bool
    {
        Log::info('SMS (log provider, not sent)', ['mobile' => $mobile, 'message' => $message]);

        return true;
    }

    public function sendOtp(string $mobile, string $code, string $template): bool
    {
        // The code itself is deliberately not logged: it is readable from
        // otp_verifications where this provider is allowed to run anyway.
        Log::info('OTP SMS (log provider, not sent)', ['mobile' => $mobile, 'template' => $template]);

        return true;
    }

    public function getName(): string
    {
        return 'log';
    }
}
