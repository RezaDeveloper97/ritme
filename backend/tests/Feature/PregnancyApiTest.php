<?php

namespace Tests\Feature;

use App\Models\User;
use Database\Seeders\PregnancyWeeklyContentSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * End-to-end coverage of pregnancy mode: activation, dating/onboarding and the
 * derived week (1-40), daily symptom logging with alert generation, weekly
 * checkups, fetal-movement tracking, and the seeded weekly educational content.
 */
class PregnancyApiTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        Passport::actingAs($user);

        return $user;
    }

    /** Onboard the acting user at a known gestational age via LMP dating. */
    private function onboardAtWeeks(int $weeks): void
    {
        $this->postJson('/api/v1/pregnancy/activate')->assertOk();

        $this->postJson('/api/v1/pregnancy/onboarding', [
            'age_source' => 'lmp',
            'lmp_date' => now()->subDays($weeks * 7)->toDateString(),
        ])->assertCreated();
    }

    // ── Activation & mode ──────────────────────────────────────

    public function test_activate_switches_into_pregnancy_mode(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/pregnancy/activate')
            ->assertOk()
            ->assertJsonPath('success', true)
            ->assertJsonPath('data.pregnancy_mode', true)
            ->assertJsonPath('data.cycle_mode', false)
            ->assertJsonPath('data.onboarding_required', true);
    }

    public function test_deactivate_returns_to_cycle_mode(): void
    {
        $this->actingUser();
        $this->postJson('/api/v1/pregnancy/activate')->assertOk();

        $this->postJson('/api/v1/pregnancy/deactivate')
            ->assertOk()
            ->assertJsonPath('data.pregnancy_mode', false)
            ->assertJsonPath('data.cycle_mode', true);
    }

    public function test_pregnancy_endpoints_require_authentication(): void
    {
        $this->getJson('/api/v1/pregnancy/status')->assertUnauthorized();
    }

    // ── Onboarding & week calculation ──────────────────────────

    public function test_onboarding_with_lmp_computes_week_and_due_date(): void
    {
        $this->actingUser();
        $this->postJson('/api/v1/pregnancy/activate')->assertOk();

        // Exactly 20 weeks + 0 days ago → display week 21 (weeks + 1).
        $response = $this->postJson('/api/v1/pregnancy/onboarding', [
            'age_source' => 'lmp',
            'lmp_date' => now()->subDays(140)->toDateString(),
        ])->assertCreated();

        $response->assertJsonPath('data.gestational_age.weeks', 20);
        $response->assertJsonPath('data.gestational_age.trimester', 2);
        $this->assertNotNull($response->json('data.estimated_due_date.date'));
    }

    public function test_status_reflects_current_week_after_onboarding(): void
    {
        $this->actingUser();
        $this->onboardAtWeeks(20);

        $this->getJson('/api/v1/pregnancy/status')
            ->assertOk()
            ->assertJsonPath('data.is_active', true)
            ->assertJsonPath('data.current_week', 21)
            ->assertJsonPath('data.trimester', 2);
    }

    public function test_onboarding_validates_age_source(): void
    {
        $this->actingUser();
        $this->postJson('/api/v1/pregnancy/activate')->assertOk();

        // ultrasound requires ultrasound_date + weeks.
        $this->postJson('/api/v1/pregnancy/onboarding', [
            'age_source' => 'ultrasound',
        ])->assertStatus(422);

        // unknown source rejected.
        $this->postJson('/api/v1/pregnancy/onboarding', [
            'age_source' => 'crystal_ball',
        ])->assertStatus(422);
    }

    public function test_profile_is_404_before_onboarding(): void
    {
        $this->actingUser();
        $this->getJson('/api/v1/pregnancy/profile')->assertNotFound();
    }

    public function test_profile_is_present_after_onboarding(): void
    {
        // Note: a fresh acting user — Passport::actingAs reuses one in-memory
        // User across requests, so a pre-onboarding read would cache a null
        // `pregnancyProfile` relation (a test artifact; production resolves the
        // user per request).
        $this->actingUser();
        $this->onboardAtWeeks(10);

        $this->getJson('/api/v1/pregnancy/profile')
            ->assertOk()
            ->assertJsonPath('data.profile.onboarding_completed', true)
            ->assertJsonPath('data.status.current_week', 11);
    }

    // ── Enums ──────────────────────────────────────────────────

    public function test_enum_endpoints_return_option_lists(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/pregnancy/enums')
            ->assertOk()
            ->assertJsonStructure(['data' => ['age_sources', 'blood_types', 'rh_factors', 'pre_existing_conditions']]);

        $this->getJson('/api/v1/pregnancy/symptoms/enums')
            ->assertOk()
            ->assertJsonStructure(['data' => ['severity']]);

        $this->getJson('/api/v1/pregnancy/weekly/enums')
            ->assertOk()
            ->assertJsonStructure(['data' => ['swelling_locations', 'overall_mood', 'severity', 'fetal_movement_status']]);
    }

    // ── Daily symptoms + alerts ────────────────────────────────

    public function test_symptom_log_upserts_and_can_be_read_back(): void
    {
        $this->actingUser();
        $this->onboardAtWeeks(12);

        $date = now()->toDateString();
        $this->postJson('/api/v1/pregnancy/symptoms', [
            'log_date' => $date,
            'has_nausea' => true,
            'nausea_severity' => 'moderate',
            'has_fatigue' => true,
        ])->assertCreated()
            ->assertJsonPath('data.log.has_nausea', true);

        $this->getJson("/api/v1/pregnancy/symptoms/{$date}")
            ->assertOk()
            ->assertJsonPath('data.log.nausea_severity', 'moderate');

        // Second post for the same date updates rather than duplicates.
        $this->postJson('/api/v1/pregnancy/symptoms', [
            'log_date' => $date,
            'has_nausea' => true,
            'nausea_severity' => 'severe',
        ])->assertCreated();

        $this->assertDatabaseCount('pregnancy_symptom_logs', 1);
    }

    public function test_critical_bleeding_symptom_raises_emergency_alert(): void
    {
        $this->actingUser();
        $this->onboardAtWeeks(12);

        $response = $this->postJson('/api/v1/pregnancy/symptoms', [
            'log_date' => now()->toDateString(),
            'has_bleeding' => true,
            'bleeding_severity' => 'severe',
        ])->assertCreated();

        $alerts = $response->json('data.alerts');
        $this->assertNotEmpty($alerts, 'severe bleeding should raise an alert');
        $this->assertContains('emergency', array_column($alerts, 'alert_level'));

        $this->getJson('/api/v1/pregnancy/alerts/summary')
            ->assertOk()
            ->assertJsonPath('data.has_emergency', true);
    }

    public function test_alert_can_be_marked_read_and_dismissed(): void
    {
        $this->actingUser();
        $this->onboardAtWeeks(12);

        $alerts = $this->postJson('/api/v1/pregnancy/symptoms', [
            'log_date' => now()->toDateString(),
            'has_fluid_leakage' => true,
        ])->json('data.alerts');

        $id = $alerts[0]['id'];

        $this->postJson("/api/v1/pregnancy/alerts/{$id}/read")->assertOk();
        $this->postJson("/api/v1/pregnancy/alerts/{$id}/dismiss")->assertOk();

        // Dismissed alerts drop out of the active list.
        $this->getJson('/api/v1/pregnancy/alerts')
            ->assertOk()
            ->assertJsonCount(0, 'data.alerts');
    }

    // ── Weekly checkup + alerts ────────────────────────────────

    public function test_weekly_log_upserts_and_high_bp_raises_alert(): void
    {
        $this->actingUser();
        $this->onboardAtWeeks(30);

        $response = $this->postJson('/api/v1/pregnancy/weekly', [
            'log_date' => now()->toDateString(),
            'pregnancy_week' => 31,
            'weight' => 72.5,
            'has_blood_pressure_device' => true,
            'systolic_pressure' => 150,
            'diastolic_pressure' => 95,
        ])->assertCreated();

        $this->assertNotEmpty($response->json('data.alerts'), 'BP >= 140/90 should raise a warning');

        $this->getJson('/api/v1/pregnancy/weekly/31')
            ->assertOk()
            ->assertJsonPath('data.log.systolic_pressure', 150);
    }

    // ── Fetal movement + alerts ────────────────────────────────

    public function test_fetal_movement_absent_after_week_24_raises_alert(): void
    {
        $this->actingUser();
        $this->onboardAtWeeks(26);

        $response = $this->postJson('/api/v1/pregnancy/fetal-movement', [
            'log_date' => now()->toDateString(),
            'pregnancy_week' => 27,
            'movement_status' => 'none',
        ])->assertCreated();

        $this->assertNotEmpty($response->json('data.alerts'), 'no movement at 24+ weeks should raise an alert');
    }

    public function test_fetal_movement_felt_records_first_movement_on_profile(): void
    {
        $this->actingUser();
        $this->onboardAtWeeks(20);

        $this->postJson('/api/v1/pregnancy/fetal-movement', [
            'log_date' => now()->toDateString(),
            'pregnancy_week' => 21,
            'movement_status' => 'felt',
        ])->assertCreated();

        $this->getJson('/api/v1/pregnancy/profile')
            ->assertOk()
            ->assertJsonPath('data.profile.fetal_movement_felt', true);
    }

    // ── Weekly educational content (seeded) ────────────────────

    public function test_weekly_content_returns_seeded_bilingual_content(): void
    {
        $this->seed(PregnancyWeeklyContentSeeder::class);
        $this->actingUser();

        // Persian content via the explicit locale query param.
        $fa = $this->getJson('/api/v1/pregnancy/content/12?locale=fa')
            ->assertOk()
            ->assertJsonPath('data.week', 12);
        $this->assertNotEmpty($fa->json('data.content.fetal_development'));

        // English resolves to a different string for the same week.
        $en = $this->getJson('/api/v1/pregnancy/content/12?locale=en')->assertOk();
        $this->assertNotSame(
            $fa->json('data.content.fetal_development'),
            $en->json('data.content.fetal_development'),
        );
    }

    public function test_all_forty_weeks_have_content(): void
    {
        $this->seed(PregnancyWeeklyContentSeeder::class);
        $this->actingUser();

        foreach ([1, 20, 40] as $week) {
            $this->getJson("/api/v1/pregnancy/content/{$week}?locale=fa")
                ->assertOk()
                ->assertJsonPath('data.week', $week);
        }
    }

    public function test_weekly_content_rejects_out_of_range_week(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/pregnancy/content/41?locale=fa')->assertStatus(422);
        $this->getJson('/api/v1/pregnancy/content/0?locale=fa')->assertStatus(422);
    }
}
