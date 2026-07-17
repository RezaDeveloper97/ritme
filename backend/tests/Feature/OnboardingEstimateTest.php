<?php

namespace Tests\Feature;

use App\Models\CycleHistory;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * The onboarding-estimate cycle seed (spec §1): building the profile creates an
 * estimate record tagged as such — never treated as a real logged period, excluded
 * from the user-facing history and the prediction medians, and promoted to a real
 * log if the user later confirms a period on that day.
 */
class OnboardingEstimateTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        Passport::actingAs($user);

        return $user;
    }

    public function test_building_the_profile_seeds_a_tagged_estimate(): void
    {
        $user = $this->actingUser();
        $lmp = now()->subDays(3)->toDateString();

        $this->postJson('/api/v1/profile', [
            'birthday' => '1995-05-15',
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => $lmp,
            'pregnancy_intention' => 'avoiding',
        ])->assertOk();

        $seed = CycleHistory::where('user_id', $user->id)->first();
        $this->assertNotNull($seed);
        $this->assertSame('onboarding_estimate', $seed->source);
        $this->assertTrue((bool) $seed->is_estimated);
        $this->assertFalse((bool) $seed->is_confirmed);
        $this->assertSame($lmp, $seed->period_start_date->toDateString());
    }

    public function test_the_estimate_is_hidden_from_the_logged_period_history(): void
    {
        $this->actingUser();
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28,
            'last_period_start' => now()->subDays(3)->toDateString(),
        ])->assertOk();

        $periods = $this->getJson('/api/v1/cycle/period/history')->assertOk()->json('data.periods');
        $this->assertCount(0, $periods); // the estimate is a seed, not a logged period
    }

    public function test_the_estimate_does_not_feed_prediction_medians(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28,
            'last_period_start' => now()->subDays(3)->toDateString(),
        ])->assertOk();

        // Re-authenticate with a fresh instance so the profile relation isn't the
        // stale null cached on the actingAs instance (a test-only Passport gotcha;
        // production loads a fresh user per request).
        Passport::actingAs($user->fresh());

        $view = $this->getJson('/api/v1/cycle/today')->assertOk()->json('data.cycle_view');
        // Only an estimate exists → no valid cycles → effective falls back to the profile.
        $this->assertNull($view['calculated_values']['cycle_length']);
        $this->assertSame('profile', $view['effective_values']['source']);
    }

    public function test_logging_a_real_start_promotes_the_estimate(): void
    {
        $user = $this->actingUser();
        $lmp = now()->subDays(3)->toDateString();
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28, 'last_period_start' => $lmp,
        ])->assertOk();

        // Confirm the period actually started on the estimated day.
        $this->postJson('/api/v1/cycle/period/start', ['date' => $lmp])->assertOk();

        $record = CycleHistory::where('user_id', $user->id)->whereDate('period_start_date', $lmp)->first();
        $this->assertSame('user_logged', $record->source);
        $this->assertFalse((bool) $record->is_estimated);
        $this->assertTrue((bool) $record->is_confirmed);

        // Now it IS a real logged period, so it shows in history.
        $periods = $this->getJson('/api/v1/cycle/period/history')->assertOk()->json('data.periods');
        $this->assertCount(1, $periods);
    }

    public function test_logging_a_start_on_a_different_day_clears_the_estimate(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28,
            'last_period_start' => now()->subDays(40)->toDateString(),
        ])->assertOk();

        // A real period logged elsewhere supersedes the onboarding seed entirely.
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(2)->toDateString()])->assertOk();

        $this->assertSame(0, CycleHistory::where('user_id', $user->id)->where('is_estimated', true)->count());
    }
}
