<?php

namespace Tests\Feature;

use App\Models\DailyHealthLog;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Locks the per-field auto-save contract the daily-log screen depends on: each
 * tap sends a tiny patch to POST /health-logs, so the endpoint must upsert the
 * SAME day's row on every call (never re-insert), merge only the sent fields,
 * and clear a field when the patch sends it as null.
 */
class DailyHealthLogApiTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        Passport::actingAs($user);

        return $user;
    }

    /** Repeated patches to the same date update one row — the regression the date:Y-m-d cast guards. */
    public function test_repeated_same_day_patches_upsert_one_row(): void
    {
        $user = $this->actingUser();
        $date = now()->subDay()->toDateString();

        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'moods' => ['happy']])
            ->assertStatus(201);
        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'sleep_quality' => 'good'])
            ->assertStatus(200);

        $this->assertSame(1, DailyHealthLog::where('user_id', $user->id)->count());
    }

    /** A field patch merges into the existing row without wiping previously-saved fields. */
    public function test_partial_patch_preserves_other_fields(): void
    {
        $this->actingUser();
        $date = now()->subDay()->toDateString();

        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'moods' => ['happy']]);
        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'sleep_quality' => 'good'])
            ->assertJsonPath('data.moods', ['happy'])
            ->assertJsonPath('data.sleep_quality', 'good');
    }

    /** Deselecting an option sends null, which clears just that field. */
    public function test_null_patch_clears_a_field(): void
    {
        $this->actingUser();
        $date = now()->subDay()->toDateString();

        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'sleep_quality' => 'good', 'moods' => ['happy']]);
        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'sleep_quality' => null])
            ->assertJsonPath('data.sleep_quality', null)
            ->assertJsonPath('data.moods', ['happy']);
    }

    /** The saved date round-trips as a clean Y-m-d string (frontend cache key). */
    public function test_log_date_serializes_as_ymd(): void
    {
        $this->actingUser();
        $date = now()->subDay()->toDateString();

        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'moods' => ['calm']])
            ->assertJsonPath('data.log_date', $date);
    }
}
