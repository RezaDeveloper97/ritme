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

    /** The ورزش section round-trips: types (multi), duration (minutes) and intensity. */
    public function test_exercise_fields_round_trip(): void
    {
        $this->actingUser();
        $date = now()->subDay()->toDateString();

        $this->postJson('/api/v1/health-logs', [
            'log_date' => $date,
            'exercise_type' => ['yoga', 'walking'],
            'exercise_duration' => 45,
            'exercise_intensity' => 'medium',
        ])
            ->assertStatus(201)
            ->assertJsonPath('data.exercise_type', ['yoga', 'walking'])
            ->assertJsonPath('data.exercise_duration', 45)
            ->assertJsonPath('data.exercise_intensity', 'medium');
    }

    /** Clients still on the single-value build keep working — the string is wrapped. */
    public function test_exercise_type_accepts_a_legacy_single_value(): void
    {
        $this->actingUser();
        $date = now()->subDay()->toDateString();

        $this->postJson('/api/v1/health-logs', [
            'log_date' => $date,
            'exercise_type' => 'yoga',
        ])
            ->assertStatus(201)
            ->assertJsonPath('data.exercise_type', ['yoga']);
    }

    /** Unknown activity kinds and out-of-range durations are rejected. */
    public function test_exercise_fields_are_validated(): void
    {
        $this->actingUser();
        $date = now()->subDay()->toDateString();

        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'exercise_type' => ['skydiving']])
            ->assertStatus(422);
        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'exercise_type' => 'skydiving'])
            ->assertStatus(422);
        $this->postJson('/api/v1/health-logs', ['log_date' => $date, 'exercise_duration' => 601])
            ->assertStatus(422);
    }

    /** The form's option lists are served from /health-logs/enums (never hardcoded client-side). */
    public function test_enums_expose_exercise_options(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/health-logs/enums')
            ->assertStatus(200)
            ->assertJsonPath('data.exercise_type.0', 'walking')
            ->assertJsonPath('data.exercise_intensity', ['low', 'medium', 'high']);
    }

    /** The graded / single-choice answers that replaced the old yes-no flags round-trip. */
    public function test_reshaped_symptom_fields_round_trip(): void
    {
        $this->actingUser();
        $date = now()->subDay()->toDateString();

        $this->postJson('/api/v1/health-logs', [
            'log_date' => $date,
            'clots_amount' => 'medium',
            'discharge_color' => 'pink_bloody',
            'vaginal_burning_intensity' => 'low',
            'vaginal_itching_intensity' => 'high',
            'sexual_desire' => 'higher',
            'intercourse_type' => 'unprotected',
            'sexual_activities' => ['lubricant_use'],
        ])
            ->assertStatus(201)
            ->assertJsonPath('data.clots_amount', 'medium')
            ->assertJsonPath('data.discharge_color', 'pink_bloody')
            ->assertJsonPath('data.vaginal_burning_intensity', 'low')
            ->assertJsonPath('data.vaginal_itching_intensity', 'high')
            ->assertJsonPath('data.sexual_desire', 'higher')
            ->assertJsonPath('data.intercourse_type', 'unprotected');
    }

    /** Discharge color is a closed list now — free text is rejected. */
    public function test_discharge_color_rejects_free_text(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/health-logs', [
            'log_date' => now()->subDay()->toDateString(),
            'discharge_color' => 'turquoise',
        ])->assertStatus(422);
    }

    /** Options the form offers reflect the reshaped questions. */
    public function test_enums_reflect_reshaped_questions(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/health-logs/enums')
            ->assertStatus(200)
            ->assertJsonPath('data.clots_amount', ['none', 'low', 'medium', 'high'])
            ->assertJsonPath('data.appetite_change', ['loss', 'normal', 'gain'])
            ->assertJsonPath('data.urination_change', ['decrease', 'increase'])
            ->assertJsonPath('data.sexual_desire', ['lower', 'normal', 'higher'])
            ->assertJsonPath('data.intercourse_type', ['protected', 'unprotected'])
            // Desire and protection are their own questions, so they're gone from here.
            ->assertJsonPath('data.sexual_activities', [
                'dryness', 'burning', 'pain_during_intercourse', 'bleeding_after_intercourse', 'lubricant_use',
            ]);
    }

    /** Logs written before the split still count as unprotected intercourse / high libido. */
    public function test_legacy_sexual_activity_values_are_still_accepted(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/health-logs', [
            'log_date' => now()->subDay()->toDateString(),
            'sexual_activities' => ['high_desire', 'unprotected_intercourse'],
        ])->assertStatus(201);
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
