<?php

namespace Tests\Feature;

use App\Jobs\SendOtpSmsJob;
use App\Models\OtpVerification;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

/**
 * `send-otp` has no bypass: no request flag, no config key, no environment
 * variable can fix the code at a known value or skip the SMS.
 *
 * It used to have two. An `is_test` flag in the request body meant anyone could
 * pin any account's code to 1111 and log in as that user; replacing it with a
 * server-side `OTP_TEST_MODE` only moved the risk to one mis-set variable on a
 * live server. These tests are what keep both from coming back.
 */
class OtpNoBypassTest extends TestCase
{
    use RefreshDatabase;

    private const MOBILE = '09123334444';

    public function test_request_body_cannot_ask_for_a_known_code(): void
    {
        Queue::fake();

        $response = $this->postJson('/api/v1/auth/send-otp', [
            'mobile' => self::MOBILE,
            'is_test' => true,
            'test_mode' => true,
        ]);

        // Asserting the code is *not* '1111' would be a 1-in-9000 flake against
        // random_int(1000, 9999). These say the same thing deterministically:
        // the real SMS path ran, which is the only path there is.
        $response->assertOk()->assertJsonPath('message', 'OTP sent successfully');
        Queue::assertPushed(SendOtpSmsJob::class);
    }

    public function test_no_test_mode_setting_exists_to_switch_on(): void
    {
        $this->assertNull(config('sms.otp.test_mode'));
    }

    public function test_the_code_is_randomly_generated_and_queued_for_delivery(): void
    {
        Queue::fake();

        $this->postJson('/api/v1/auth/send-otp', ['mobile' => self::MOBILE])->assertOk();

        $code = OtpVerification::where('mobile', self::MOBILE)->value('code');
        $this->assertMatchesRegularExpression('/^\d{4}$/', (string) $code);
        Queue::assertPushed(SendOtpSmsJob::class);
    }

    public function test_the_resend_window_is_enforced_with_no_way_around_it(): void
    {
        Queue::fake();

        $this->postJson('/api/v1/auth/send-otp', ['mobile' => self::MOBILE])->assertOk();
        $this->postJson('/api/v1/auth/send-otp', [
            'mobile' => self::MOBILE,
            'is_test' => true,
        ])->assertStatus(429);
    }
}
