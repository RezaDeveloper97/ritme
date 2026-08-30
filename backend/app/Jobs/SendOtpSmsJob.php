<?php

namespace App\Jobs;

use App\Services\Sms\SmsService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Log;
use RuntimeException;

/**
 * Delivers a login OTP over SMS in the background.
 *
 * The OTP row is already persisted by the time this runs, so the user can
 * enter the code the moment the SMS arrives; a delivery failure only costs a
 * retry, never the login attempt itself.
 */
class SendOtpSmsJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $tries = 3;

    public array $backoff = [5, 15];

    public function __construct(
        private string $mobile,
        private string $code,
        private string $template = 'login_otp',
    ) {}

    public function handle(): void
    {
        $sent = (new SmsService)->sendOtp($this->mobile, $this->code, $this->template);

        if (! $sent) {
            // Throwing hands the job back to the queue so `tries`/`backoff`
            // apply — SmsService itself only ever reports a boolean.
            throw new RuntimeException("OTP SMS delivery failed for {$this->mobile}");
        }
    }

    public function failed(\Throwable $e): void
    {
        Log::error('OTP SMS job exhausted all retries', [
            'mobile' => $this->mobile,
            'error' => $e->getMessage(),
        ]);
    }
}
