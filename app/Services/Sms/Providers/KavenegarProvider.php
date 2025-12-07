<?php

namespace App\Services\Sms\Providers;

use App\Services\Sms\Contracts\SmsProviderInterface;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class KavenegarProvider implements SmsProviderInterface
{
    private string $apiKey;

    private ?string $sender;

    private array $templates;

    private string $baseUrl = 'https://api.kavenegar.com/v1';

    public function __construct()
    {
        $config = config('sms.providers.kavenegar');
        $this->apiKey = $config['api_key'] ?? '';
        $this->sender = $config['sender'] ?? null;
        $this->templates = $config['templates'] ?? [];
    }

    /**
     * Send a simple SMS message
     */
    public function send(string $mobile, string $message): bool
    {
        try {
            $response = Http::get("{$this->baseUrl}/{$this->apiKey}/sms/send.json", [
                'receptor' => $mobile,
                'message' => $message,
                'sender' => $this->sender,
            ]);

            if ($response->successful()) {
                $data = $response->json();

                return isset($data['return']['status']) && $data['return']['status'] === 200;
            }

            Log::error('Kavenegar SMS send failed', [
                'mobile' => $mobile,
                'response' => $response->json(),
            ]);

            return false;
        } catch (\Exception $e) {
            Log::error('Kavenegar SMS exception', [
                'mobile' => $mobile,
                'error' => $e->getMessage(),
            ]);

            return false;
        }
    }

    /**
     * Send OTP using Kavenegar lookup (template) API
     */
    public function sendOtp(string $mobile, string $code, string $template): bool
    {
        try {
            $templateName = $this->templates[$template] ?? $template;

            $response = Http::get("{$this->baseUrl}/{$this->apiKey}/verify/lookup.json", [
                'receptor' => $mobile,
                'token' => $code,
                'template' => $templateName,
            ]);

            if ($response->successful()) {
                $data = $response->json();

                return isset($data['return']['status']) && $data['return']['status'] === 200;
            }

            Log::error('Kavenegar OTP send failed', [
                'mobile' => $mobile,
                'template' => $templateName,
                'response' => $response->json(),
            ]);

            return false;
        } catch (\Exception $e) {
            Log::error('Kavenegar OTP exception', [
                'mobile' => $mobile,
                'error' => $e->getMessage(),
            ]);

            return false;
        }
    }

    /**
     * Get provider name
     */
    public function getName(): string
    {
        return 'kavenegar';
    }
}
