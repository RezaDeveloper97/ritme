<?php

namespace Tests\Feature;

use App\Models\OtpVerification;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\ClientRepository;
use Tests\TestCase;

/**
 * `verify-otp` hands out a token the moment a number is confirmed — long before
 * any registration question is answered. `profile_completed` is what tells the
 * client the difference, so a visitor who abandoned onboarding is sent back into
 * it instead of into a profile-less app.
 */
class OtpProfileCompletionTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        // verify-otp issues a real personal access token; the fresh test DB has
        // no client to sign it with.
        app(ClientRepository::class)->createPersonalAccessGrantClient('test', 'users');
    }

    private function verify(string $mobile): \Illuminate\Testing\TestResponse
    {
        OtpVerification::create([
            'mobile' => $mobile,
            'code' => '1111',
            'expires_at' => now()->addMinutes(2),
        ]);

        return $this->postJson('/api/v1/auth/verify-otp', [
            'mobile' => $mobile,
            'code' => '1111',
        ]);
    }

    public function test_brand_new_account_is_not_profile_completed(): void
    {
        $this->verify('09121110000')
            ->assertOk()
            ->assertJsonPath('data.new_user', true)
            ->assertJsonPath('data.profile_completed', false);
    }

    public function test_returning_account_that_abandoned_onboarding_is_not_completed(): void
    {
        // Verified before, never finished: no name, no health profile. This is
        // the case `new_user` alone gets wrong.
        User::factory()->create([
            'mobile' => '09121110001',
            'name' => null,
            'mobile_verified_at' => now(),
        ]);

        $this->verify('09121110001')
            ->assertOk()
            ->assertJsonPath('data.new_user', false)
            ->assertJsonPath('data.profile_completed', false);
    }

    public function test_name_without_a_health_profile_is_still_incomplete(): void
    {
        User::factory()->create(['mobile' => '09121110002', 'name' => 'Sara']);

        $this->verify('09121110002')
            ->assertOk()
            ->assertJsonPath('data.profile_completed', false);
    }

    public function test_fully_registered_account_is_completed(): void
    {
        $user = User::factory()->create(['mobile' => '09121110003', 'name' => 'Sara']);
        UserProfile::create([
            'user_id' => $user->id,
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => now()->subDays(10)->toDateString(),
        ]);

        $this->verify('09121110003')
            ->assertOk()
            ->assertJsonPath('data.profile_completed', true);
    }
}
