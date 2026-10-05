<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Sms;

use App\Domain\Directory\Booking\Contracts\SmsSender;
use Psr\Log\LoggerInterface;

/**
 * The `log` SMS driver (default): writes the message to the application log instead of calling a provider, with
 * the number masked except its last four digits and order/booking codes redacted (L9-04: a code opens its status
 * page, so the log must not hand it out).
 */
final readonly class LogSmsSender implements SmsSender
{
    public function __construct(private LoggerInterface $log) {}

    public function send(string $mobile, string $text): void
    {
        $this->log->info('SMS (log driver)', [
            'to' => str_repeat('*', max(0, strlen($mobile) - 4)).substr($mobile, -4),
            'text' => (string) preg_replace('/\b[2-9A-HJKMNP-Z]{4}-[2-9A-HJKMNP-Z]{4}-[2-9A-HJKMNP-Z]{4}\b/i', '****-****-****', $text),
        ]);
    }
}
