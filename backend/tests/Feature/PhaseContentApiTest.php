<?php

namespace Tests\Feature;

use App\Enums\CycleSubphase;
use App\Models\PhaseContent;
use App\Models\User;
use Database\Seeders\PhaseContentSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * GET /api/v1/cycle/phase-content/{phase} — the DB-driven educational content
 * for the Phase Details screen. Content is keyed by CycleSubphase value and
 * localized off the ?locale param.
 */
class PhaseContentApiTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        Passport::actingAs($user);

        return $user;
    }

    private function seedHighFertility(): PhaseContent
    {
        return PhaseContent::create([
            'phase' => 'high_fertility',
            'symptom_prediction' => ['fa' => 'علائم فارسی', 'en' => 'English symptoms'],
            'vaginal_discharge' => ['fa' => 'ترشحات فارسی', 'en' => null],
            'fertility' => ['fa' => 'باروری فارسی', 'en' => 'Fertility EN'],
            // sleep intentionally left empty to prove empty sections are hidden.
            'sleep' => ['fa' => '', 'en' => ''],
        ]);
    }

    public function test_returns_localized_sections_for_a_phase(): void
    {
        $this->actingUser();
        $this->seedHighFertility();

        $data = $this->getJson('/api/v1/cycle/phase-content/high_fertility?locale=fa')
            ->assertOk()
            ->json('data');

        $this->assertSame('high_fertility', $data['phase']);
        $this->assertSame('باروری بالا', $data['phase_label']);
        $this->assertSame('علائم فارسی', $data['sections']['symptom_prediction']);
        $this->assertSame('ترشحات فارسی', $data['sections']['vaginal_discharge']);
        // Empty sections are omitted so the client can hide them gracefully.
        $this->assertArrayNotHasKey('sleep', $data['sections']);
        // Sections with no copy for this phase are absent, never null keys.
        $this->assertArrayNotHasKey('exercise', $data['sections']);
    }

    public function test_english_locale_returns_english_and_falls_back_to_fa(): void
    {
        $this->actingUser();
        $this->seedHighFertility();

        $data = $this->getJson('/api/v1/cycle/phase-content/high_fertility?locale=en')
            ->assertOk()
            ->json('data');

        $this->assertSame('High Fertility', $data['phase_label']);
        $this->assertSame('English symptoms', $data['sections']['symptom_prediction']);
        // en is null → falls back to the fa copy rather than disappearing.
        $this->assertSame('ترشحات فارسی', $data['sections']['vaginal_discharge']);
    }

    public function test_missing_content_returns_404(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/cycle/phase-content/mid_luteal')
            ->assertNotFound()
            ->assertJson(['success' => false]);
    }

    public function test_invalid_phase_key_returns_422(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/cycle/phase-content/not_a_real_phase')
            ->assertStatus(422)
            ->assertJson(['success' => false]);
    }

    public function test_requires_authentication(): void
    {
        $this->getJson('/api/v1/cycle/phase-content/high_fertility')
            ->assertUnauthorized();
    }

    public function test_seeder_populates_every_phase_with_all_nine_sections(): void
    {
        $this->seed(PhaseContentSeeder::class);

        $this->assertCount(count(CycleSubphase::cases()), PhaseContent::all());

        foreach (CycleSubphase::values() as $phase) {
            $row = PhaseContent::getByPhase($phase);
            $this->assertNotNull($row, "missing seeded content for {$phase}");
            foreach (PhaseContent::SECTIONS as $section) {
                $fa = $row->getLocalizedContent($section, 'fa');
                $this->assertIsString($fa);
                $this->assertNotSame('', trim($fa), "empty {$section} for {$phase}");
            }
        }
    }

    public function test_seeded_content_is_served_through_the_endpoint(): void
    {
        $this->actingUser();
        $this->seed(PhaseContentSeeder::class);

        $sections = $this->getJson('/api/v1/cycle/phase-content/menstruation?locale=fa')
            ->assertOk()
            ->json('data.sections');

        // All nine sections come back, and the copy is the verbatim spec text.
        $this->assertCount(9, $sections);
        $this->assertStringContainsString('شروع پریود', $sections['symptom_prediction']);
    }
}
