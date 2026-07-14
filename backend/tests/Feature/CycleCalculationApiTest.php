<?php

namespace Tests\Feature;

use App\Enums\CalculationStatus;
use App\Models\CycleHistory;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Guards the DivisionByZeroError regression: HealthDataEngine divides by the
 * effective cycle length everywhere, so any path that lets it reach 0 crashed
 * cycle/today, messages/daily and the recalculate job with a 500 / stuck
 * "processing" status. These tests lock the length to a positive fallback.
 */
class CycleCalculationApiTest extends TestCase
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

    /** cycle_duration of 0 must not crash — it used to divide by zero. */
    public function test_cycle_today_survives_zero_cycle_duration(): void
    {
        $this->actingUser(['cycle_duration' => 0]);

        $this->getJson('/api/v1/cycle/today')
            ->assertOk()
            ->assertJsonPath('success', true);
    }

    /** Histories whose cycle_length is all null must not crash (avg() → null → 0). */
    public function test_cycle_today_survives_null_length_histories(): void
    {
        $user = $this->actingUser();

        // A history exists but carries no usable cycle_length. This drove the
        // production crash at HealthDataEngine.php:225.
        CycleHistory::create([
            'user_id' => $user->id,
            'period_start_date' => now()->subDays(40)->toDateString(),
            'cycle_length' => null,
        ]);

        $this->getJson('/api/v1/cycle/today')->assertOk();
    }

    /** messages/daily shares the engine, so it must survive the same data. */
    public function test_daily_messages_survive_zero_cycle_duration(): void
    {
        $this->actingUser(['cycle_duration' => 0]);

        $this->getJson('/api/v1/messages/daily')->assertOk();
    }

    /**
     * A period length >= the cycle length is physiologically impossible and used
     * to paint every day of the month as menstruation (a solid period block on
     * the calendar). The engine now falls back to the default, so the month must
     * contain non-menstruation phases.
     */
    public function test_month_is_not_all_menstruation_when_period_exceeds_cycle(): void
    {
        $this->actingUser([
            'cycle_duration' => 15,
            'period_duration' => 15,
            'last_period_start' => now()->startOfMonth()->toDateString(),
        ]);

        $now = now();
        $response = $this->getJson("/api/v1/cycle/month/{$now->year}/{$now->month}")->assertOk();

        $phases = collect($response->json('data.calculations'))->pluck('phase')->unique();

        $this->assertGreaterThan(1, $phases->count(), 'Calendar month should not be a single solid phase.');
        $this->assertContains('follicular', $phases->all());
    }

    /**
     * The recalculate job must complete (not hang on "processing") even with the
     * degenerate data. Tests use the sync queue, so the job runs inline.
     */
    public function test_recalculate_completes_and_does_not_get_stuck(): void
    {
        $user = $this->actingUser(['cycle_duration' => 0]);

        $this->postJson('/api/v1/cycle/recalculate')
            ->assertOk()
            ->assertJsonPath('success', true);

        // Status settled to completed — not stuck on processing.
        $this->assertSame(
            CalculationStatus::COMPLETED->value,
            $user->profile->fresh()->calculation_status,
        );
    }
}
