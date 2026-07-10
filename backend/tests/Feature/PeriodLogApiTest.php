<?php

namespace Tests\Feature;

use App\Models\CycleHistory;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Covers the period logging endpoints (start / end / status): starting a period
 * creates a CycleHistory record and re-anchors the profile LMP, and ending it
 * closes that record with an end date and bleeding length.
 */
class PeriodLogApiTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(array $profileOverrides = []): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);

        UserProfile::create(array_merge([
            'user_id' => $user->id,
            'birthday' => '1995-05-15',
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => now()->subDays(40)->toDateString(),
        ], $profileOverrides));

        Passport::actingAs($user);

        return $user;
    }

    public function test_status_reports_inactive_when_no_open_period(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/cycle/period/status')
            ->assertOk()
            ->assertJsonPath('data.active', false);
    }

    public function test_start_period_creates_history_and_reanchors_lmp(): void
    {
        $user = $this->actingUser();
        $today = now()->toDateString();

        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])
            ->assertOk()
            ->assertJsonPath('success', true)
            ->assertJsonPath('data.active', true)
            ->assertJsonPath('data.period_start_date', $today);

        $this->assertDatabaseHas('cycle_histories', [
            'user_id' => $user->id,
            'period_end_date' => null,
        ]);
        $this->assertSame($today, $user->profile->fresh()->last_period_start->toDateString());

        // Status now reports an ongoing period.
        $this->getJson('/api/v1/cycle/period/status')
            ->assertOk()
            ->assertJsonPath('data.active', true);
    }

    public function test_start_period_is_idempotent_for_same_day(): void
    {
        $user = $this->actingUser();
        $today = now()->toDateString();

        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])->assertOk();
        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])->assertOk();

        $this->assertSame(1, CycleHistory::where('user_id', $user->id)
            ->whereDate('period_start_date', $today)->count());
    }

    public function test_end_period_closes_open_record_with_bleeding_length(): void
    {
        $user = $this->actingUser();
        $start = now()->subDays(4)->toDateString();
        $end = now()->toDateString();

        $this->postJson('/api/v1/cycle/period/start', ['date' => $start])->assertOk();

        $this->postJson('/api/v1/cycle/period/end', ['date' => $end])
            ->assertOk()
            ->assertJsonPath('data.active', false)
            ->assertJsonPath('data.period_end_date', $end);

        $record = CycleHistory::where('user_id', $user->id)->latest('period_start_date')->first();
        $this->assertSame($end, $record->period_end_date->toDateString());
        $this->assertSame(5, $record->bleeding_length); // 4 days later, inclusive
    }

    public function test_restarting_an_ended_period_reopens_it_so_it_can_be_ended_again(): void
    {
        $this->actingUser();
        $today = now()->toDateString();

        // Start, end, then start again on the same day. The restart must clear
        // the prior end date so the period is ongoing and End works again
        // (regression: End used to report "no ongoing period").
        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])->assertOk();
        $this->postJson('/api/v1/cycle/period/end', ['date' => $today])->assertOk();

        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])
            ->assertOk()
            ->assertJsonPath('data.active', true)
            ->assertJsonPath('data.period_end_date', null);

        $this->getJson('/api/v1/cycle/period/status')->assertJsonPath('data.active', true);

        $this->postJson('/api/v1/cycle/period/end', ['date' => $today])
            ->assertOk()
            ->assertJsonPath('data.active', false)
            ->assertJsonPath('data.period_end_date', $today);
    }

    public function test_end_period_without_ongoing_period_fails(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/cycle/period/end')
            ->assertStatus(422)
            ->assertJsonPath('success', false);
    }

    public function test_end_before_start_is_rejected(): void
    {
        $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(2)->toDateString()])->assertOk();

        $this->postJson('/api/v1/cycle/period/end', ['date' => now()->subDays(5)->toDateString()])
            ->assertStatus(422);
    }

    public function test_future_start_date_is_rejected(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->addDay()->toDateString()])
            ->assertStatus(422);
    }
}
