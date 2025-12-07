<?php

namespace App\Services\Sms;

use App\Services\Sms\Contracts\SmsProviderInterface;
use App\Services\Sms\Providers\KavenegarProvider;
use App\Services\Sms\Providers\SmsIrProvider;
use Illuminate\Support\Facades\Log;
use InvalidArgumentException;

class SmsService
{
    private SmsProviderInterface $provider;

    private array $fallbackProviders = [];

    public function __construct(?string $providerName = null)
    {
        $providerName = $providerName ?? config('sms.default');
        $this->provider = $this->resolveProvider($providerName);

        // Setup fallback providers
        $fallbacks = config('sms.fallback', []);
        foreach ($fallbacks as $fallback) {
            if ($fallback !== $providerName) {
                $this->fallbackProviders[] = $fallback;
            }
        }
    }

    /**
     * Resolve the SMS provider
     */
    private function resolveProvider(string $name): SmsProviderInterface
    {
        return match ($name) {
            'smsir' => new SmsIrProvider,
            'kavenegar' => new KavenegarProvider,
            default => throw new InvalidArgumentException("Unknown SMS provider: {$name}"),
        };
    }

    /**
     * Send a simple SMS message with fallback support
     */
    public function send(string $mobile, string $message): bool
    {
        // Try primary provider
        if ($this->provider->send($mobile, $message)) {
            Log::info('SMS sent successfully', [
                'provider' => $this->provider->getName(),
                'mobile' => $mobile,
            ]);

            return true;
        }

        // Try fallback providers
        return $this->tryFallbacks('send', $mobile, $message);
    }

    /**
     * Send OTP using template with fallback support
     */
    public function sendOtp(string $mobile, string $code, string $template = 'login_otp'): bool
    {
        // Try primary provider
        if ($this->provider->sendOtp($mobile, $code, $template)) {
            Log::info('OTP sent successfully', [
                'provider' => $this->provider->getName(),
                'mobile' => $mobile,
            ]);

            return true;
        }

        // Try fallback providers
        return $this->tryFallbacksOtp($mobile, $code, $template);
    }

    /**
     * Try fallback providers for simple SMS
     */
    private function tryFallbacks(string $method, string $mobile, string $message): bool
    {
        foreach ($this->fallbackProviders as $providerName) {
            try {
                $fallbackProvider = $this->resolveProvider($providerName);

                Log::info('Trying fallback SMS provider', [
                    'provider' => $providerName,
                    'mobile' => $mobile,
                ]);

                if ($fallbackProvider->send($mobile, $message)) {
                    Log::info('SMS sent via fallback provider', [
                        'provider' => $providerName,
                        'mobile' => $mobile,
                    ]);

                    return true;
                }
            } catch (\Exception $e) {
                Log::error('Fallback provider failed', [
                    'provider' => $providerName,
                    'error' => $e->getMessage(),
                ]);
            }
        }

        Log::error('All SMS providers failed', ['mobile' => $mobile]);

        return false;
    }

    /**
     * Try fallback providers for OTP
     */
    private function tryFallbacksOtp(string $mobile, string $code, string $template): bool
    {
        foreach ($this->fallbackProviders as $providerName) {
            try {
                $fallbackProvider = $this->resolveProvider($providerName);

                Log::info('Trying fallback OTP provider', [
                    'provider' => $providerName,
                    'mobile' => $mobile,
                ]);

                if ($fallbackProvider->sendOtp($mobile, $code, $template)) {
                    Log::info('OTP sent via fallback provider', [
                        'provider' => $providerName,
                        'mobile' => $mobile,
                    ]);

                    return true;
                }
            } catch (\Exception $e) {
                Log::error('Fallback OTP provider failed', [
                    'provider' => $providerName,
                    'error' => $e->getMessage(),
                ]);
            }
        }

        Log::error('All OTP providers failed', ['mobile' => $mobile]);

        return false;
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
