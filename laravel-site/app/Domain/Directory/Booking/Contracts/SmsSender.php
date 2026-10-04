<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Contracts;

/**
 * Sends one text message to an Iranian mobile (`09xxxxxxxxx`). The default binding is the `log` driver
 * (LogSmsSender, DirectoryServiceProvider) — no SMS provider is called until a real driver is bound.
 */
interface SmsSender
{
    public function send(string $mobile, string $text): void;
}
