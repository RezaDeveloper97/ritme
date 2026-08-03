<?php

namespace Tests\Feature;

use App\Models\CycleHistory;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * The period declared during onboarding (LMP + period duration) is stored as a real,
 * confirmed period log, so the calendar paints those days exactly as if the user had
 * logged them. The seed re-syncs while it is the whole history and is never touched
 * again once the user logs anything themselves.
 */
class OnboardingPeriodLogTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        Passport::actingAs($user);

        return $user;
    }

    public function test_building_the_profile_logs_the_declared_period(): void
    {
        $user = $this->actingUser();
        $lmp = now()->subDays(10)->toDateString();

        $this->postJson('/api/v1/profile', [
            'birthday' => '1995-05-15',
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => $lmp,
            'pregnancy_intention' => 'avoiding',
        ])->assertOk();

        $seed = CycleHistory::where('user_id', $user->id)->first();
        $this->assertNotNull($seed);
        $this->assertSame('user_profile_confirmed', $seed->source);
        $this->assertTrue((bool) $seed->is_confirmed);
        $this->assertFalse((bool) $seed->is_estimated);
        $this->assertSame($lmp, $seed->period_start_date->toDateString());
        $this->assertSame(now()->subDays(6)->toDateString(), $seed->period_end_date->toDateString());
        $this->assertSame(5, (int) $seed->bleeding_length);
    }

    public function test_the_declared_period_shows_in_the_logged_history(): void
    {
        $lmp = now()->subDays(10)->toDateString();
        $this->actingUser();
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28, 'last_period_start' => $lmp,
        ])->assertOk();

        $periods = $this->getJson('/api/v1/cycle/period/history')->assertOk()->json('data.periods');
        $this->assertCount(1, $periods);
        $this->assertSame($lmp, $periods[0]['period_start_date']);
    }

    public function test_a_period_still_running_today_is_left_open(): void
    {
        $user = $this->actingUser();

        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28,
            'last_period_start' => now()->subDays(2)->toDateString(),
        ])->assertOk();

        $seed = CycleHistory::where('user_id', $user->id)->first();
        $this->assertNull($seed->period_end_date);
        $this->assertNull($seed->bleeding_length);
        $this->assertTrue((bool) $seed->is_confirmed);
    }

    public function test_editing_the_profile_resyncs_the_same_record(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28,
            'last_period_start' => now()->subDays(20)->toDateString(),
        ])->assertOk();

        $corrected = now()->subDays(12)->toDateString();
        Passport::actingAs($user->fresh());
        $this->postJson('/api/v1/profile', [
            'period_duration' => 6, 'cycle_duration' => 28, 'last_period_start' => $corrected,
        ])->assertOk();

        $records = CycleHistory::where('user_id', $user->id)->get();
        $this->assertCount(1, $records);
        $this->assertSame($corrected, $records[0]->period_start_date->toDateString());
        $this->assertSame(6, (int) $records[0]->bleeding_length);
    }

    public function test_a_real_logged_period_is_never_overwritten_by_a_profile_edit(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28,
            'last_period_start' => now()->subDays(40)->toDateString(),
        ])->assertOk();

        $logged = now()->subDays(3)->toDateString();
        Passport::actingAs($user->fresh());
        $this->postJson('/api/v1/cycle/period', ['start_date' => $logged])->assertOk();

        Passport::actingAs($user->fresh());
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 30,
            'last_period_start' => now()->subDays(45)->toDateString(),
        ])->assertOk();

        $starts = CycleHistory::where('user_id', $user->id)
            ->orderBy('period_start_date')
            ->pluck('period_start_date')
            ->map(fn ($d) => $d->toDateString())
            ->all();

        $this->assertSame([now()->subDays(40)->toDateString(), $logged], $starts);
    }

    public function test_logging_the_same_day_again_stays_a_single_record(): void
    {
        $user = $this->actingUser();
        $lmp = now()->subDays(10)->toDateString();
        $this->postJson('/api/v1/profile', [
            'period_duration' => 5, 'cycle_duration' => 28, 'last_period_start' => $lmp,
        ])->assertOk();

        Passport::actingAs($user->fresh());
        $this->postJson('/api/v1/cycle/period/start', ['date' => $lmp])->assertOk();

        $records = CycleHistory::where('user_id', $user->id)->get();
        $this->assertCount(1, $records);
        $this->assertSame('user_logged', $records[0]->source);
        $this->assertTrue((bool) $records[0]->is_confirmed);
    }
}
