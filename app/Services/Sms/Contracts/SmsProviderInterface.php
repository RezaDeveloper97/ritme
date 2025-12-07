<?php

namespace App\Services\Sms\Contracts;

interface SmsProviderInterface
{
    /**
     * Send a simple SMS message
     */
    public function send(string $mobile, string $message): bool;

    /**
     * Send OTP using template (for providers that support it)
     */
    public function sendOtp(string $mobile, string $code, string $template): bool;

    /**
     * Get provider name
     */
    public function getName(): string;
}
