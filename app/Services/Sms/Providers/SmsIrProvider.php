<?php

namespace App\Services\Sms\Providers;

use App\Services\Sms\Contracts\SmsProviderInterface;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class SmsIrProvider implements SmsProviderInterface
{
    private string $apiKey;

    private int $templateId;

    private string $baseUrl = 'https://api.sms.ir/v1';

    public function __construct()
    {
        $config = config('sms.providers.smsir');
        $this->apiKey = $config['api_key'] ?? '';
        $this->templateId = $config['template_id'] ?? 0;
    }

    /**
     * Send a simple SMS message
     */
    public function send(string $mobile, string $message): bool
    {
        try {
            $response = Http::withHeaders([
                'X-API-KEY' => $this->apiKey,
                'Content-Type' => 'application/json',
                'Accept' => 'application/json',
            ])->post("{$this->baseUrl}/send/bulk", [
                'lineNumber' => config('sms.providers.smsir.line_number'),
                'messageText' => $message,
                'mobiles' => [$mobile],
            ]);

            if ($response->successful()) {
                $data = $response->json();

                return isset($data['status']) && $data['status'] === 1;
            }

            Log::error('SMS.ir send failed', [
                'mobile' => $mobile,
                'response' => $response->json(),
            ]);

            return false;
        } catch (\Exception $e) {
            Log::error('SMS.ir exception', [
                'mobile' => $mobile,
                'error' => $e->getMessage(),
            ]);

            return false;
        }
    }

    /**
     * Send OTP using SMS.ir verify API
     */
    public function sendOtp(string $mobile, string $code, string $template): bool
    {
        try {
            $response = Http::withHeaders([
                'X-API-KEY' => $this->apiKey,
                'Content-Type' => 'application/json',
                'Accept' => 'application/json',
            ])->post("{$this->baseUrl}/send/verify", [
                'mobile' => $mobile,
                'templateId' => $this->templateId,
                'parameters' => [
                    [
                        'name' => 'OTPCODE',
                        'value' => $code,
                    ],
                ],
            ]);

            if ($response->successful()) {
                $data = $response->json();

                Log::info('SMS.ir OTP response', [
                    'mobile' => $mobile,
                    'response' => $data,
                ]);

                return isset($data['status']) && $data['status'] === 1;
            }

            Log::error('SMS.ir OTP send failed', [
                'mobile' => $mobile,
                'status' => $response->status(),
                'response' => $response->json(),
            ]);

            return false;
        } catch (\Exception $e) {
            Log::error('SMS.ir OTP exception', [
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
        return 'smsir';
    }
}
