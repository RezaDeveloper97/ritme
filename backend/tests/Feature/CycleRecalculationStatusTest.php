<?php

namespace Tests\Feature;

use App\Enums\CalculationStatus;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Schema;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * T-M1-13: the dead-storage job (CalculateCycleDataJob → cycle_calculations) is
 * gone. Every engine-input write now bumps `calculation_version` synchronously
 * and reports `completed`, so `/cycle/status` never says "processing" and client
 * polling (frontend useCycleStatus, Android pollUntilDone) stops on the first poll.
 */
class CycleRecalculationStatusTest extends TestCase
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
            'last_period_start' => now()->subDays(10)->toDateString(),
            'calculation_version' => 3,
        ], $profileOverrides));

        Passport::actingAs($user);

        return $user;
    }

    private function profileOf(User $user): UserProfile
    {
        return UserProfile::where('user_id', $user->id)->firstOrFail();
    }

    private function assertBumpedAndCompleted(User $user, int $expectedVersion): void
    {
        $profile = $this->profileOf($user);
        $this->assertSame($expectedVersion, $profile->calculation_version);
        $this->assertSame(CalculationStatus::COMPLETED->value, $profile->calculation_status);
        $this->assertNotNull($profile->calculation_completed_at);
    }

    public function test_recalculate_completes_synchronously_and_status_is_not_processing(): void
    {
        $user = $this->actingUser();

        $this->postJson('/api/v1/cycle/recalculate')
            ->assertOk()
            ->assertJsonPath('success', true)
            ->assertJsonPath('data.version', 4)
            ->assertJsonPath('data.status', CalculationStatus::COMPLETED->value);

        $this->getJson('/api/v1/cycle/status')
            ->assertOk()
            ->assertJsonPath('data.status', CalculationStatus::COMPLETED->value)
            ->assertJsonPath('data.is_processing', false)
            ->assertJsonPath('data.version', 4);

        $this->assertBumpedAndCompleted($user, 4);
    }

    /** The old "already processing → 400 / skip" guard is gone: a stale status never blocks. */
    public function test_recalculate_is_not_blocked_by_a_stale_processing_status(): void
    {
        $user = $this->actingUser(['calculation_status' => CalculationStatus::PROCESSING->value]);

        $this->postJson('/api/v1/cycle/recalculate')
            ->assertOk()
            ->assertJsonPath('data.version', 4);

        $this->assertBumpedAndCompleted($user, 4);
    }

    public function test_every_recalculate_bumps_the_version(): void
    {
        $user = $this->actingUser();

        $this->postJson('/api/v1/cycle/recalculate')->assertOk();
        $this->postJson('/api/v1/cycle/recalculate')->assertOk()->assertJsonPath('data.version', 5);

        $this->assertBumpedAndCompleted($user, 5);
    }

    public function test_recalculate_without_profile_is_rejected(): void
    {
        Passport::actingAs(User::factory()->create(['mobile' => '09121234567']));

        $this->postJson('/api/v1/cycle/recalculate')
            ->assertStatus(400)
            ->assertJsonPath('success', false);
    }

    public function test_health_log_write_bumps_version(): void
    {
        $user = $this->actingUser(['calculation_status' => CalculationStatus::PROCESSING->value]);

        $this->postJson('/api/v1/health-logs', ['log_date' => now()->toDateString(), 'moods' => ['happy']])
            ->assertStatus(201);

        $this->assertBumpedAndCompleted($user, 4);
    }

    public function test_period_start_bumps_version(): void
    {
        $user = $this->actingUser();

        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->toDateString()])->assertOk();

        $this->assertBumpedAndCompleted($user, 4);
    }

    public function test_period_delete_bumps_version(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->toDateString()])->assertOk();
        $id = $this->getJson('/api/v1/cycle/period/history')->json('data.periods.0.id');

        $this->deleteJson("/api/v1/cycle/period/{$id}")->assertOk();

        $this->assertBumpedAndCompleted($user, 5);
    }

    public function test_profile_cycle_change_bumps_version_and_reports_completed(): void
    {
        $user = $this->actingUser();

        $this->postJson('/api/v1/profile', ['cycle_duration' => 30])
            ->assertOk()
            ->assertJsonPath('data.calculation_status', CalculationStatus::COMPLETED->value);

        $this->assertBumpedAndCompleted($user, 4);
    }

    public function test_day_and_month_views_do_not_report_recalculating_after_a_write(): void
    {
        $this->actingUser();
        $this->postJson('/api/v1/cycle/recalculate')->assertOk();

        $this->getJson('/api/v1/cycle/today')->assertOk()->assertJsonPath('data.is_recalculating', false);
        $this->getJson('/api/v1/cycle/month/'.now()->year.'/'.now()->month)
            ->assertOk()
            ->assertJsonPath('data.is_recalculating', false);
    }

    public function test_the_dead_storage_table_is_dropped(): void
    {
        $this->assertFalse(Schema::hasTable('cycle_calculations'));
    }

    /** Removed in T-M1-13: no client (frontend, Android, admin) ever called them. */
    public function test_removed_unused_routes_are_gone(): void
    {
        $this->actingUser();

        foreach (['/api/v1/cycle/enums', '/api/v1/cycle/matrix-messages', '/api/v1/cycle/matrix-enums', '/api/v1/messages/enums'] as $uri) {
            $this->getJson($uri)->assertNotFound();
        }
    }
}
