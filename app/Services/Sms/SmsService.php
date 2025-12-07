<?php

namespace App\Services\Sms;

use App\Services\Sms\Contracts\SmsProviderInterface;
use App\Services\Sms\Providers\KavenegarProvider;
use InvalidArgumentException;

class SmsService
{
    private SmsProviderInterface $provider;

    public function __construct(?string $providerName = null)
    {
        $providerName = $providerName ?? config('sms.default');
        $this->provider = $this->resolveProvider($providerName);
    }

    /**
     * Resolve the SMS provider
     */
    private function resolveProvider(string $name): SmsProviderInterface
    {
        return match ($name) {
            'kavenegar' => new KavenegarProvider,
            // Add more providers here as needed
            // 'melipayamak' => new MelipayamakProvider(),
            default => throw new InvalidArgumentException("Unknown SMS provider: {$name}"),
        };
    }

    /**
     * Send a simple SMS message
     */
    public function send(string $mobile, string $message): bool
    {
        return $this->provider->send($mobile, $message);
    }

    /**
     * Send OTP using template
     */
    public function sendOtp(string $mobile, string $code, string $template = 'login_otp'): bool
    {
        return $this->provider->sendOtp($mobile, $code, $template);
    }

    /**
     * Get current provider
     */
    public function getProvider(): SmsProviderInterface
    {
        return $this->provider;
    }

    /**
     * Switch to a different provider
     */
    public function useProvider(string $name): self
    {
        $this->provider = $this->resolveProvider($name);

        return $this;
    }

    /**
     * Generate OTP code
     */
    public static function generateOtp(int $length = 4): string
    {
        $length = $length ?: config('sms.otp.length', 4);
        $min = pow(10, $length - 1);
        $max = pow(10, $length) - 1;

        return (string) random_int($min, $max);
    }
}
