<?php

namespace Tests\Feature;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\RecommendationTrigger;
use App\Enums\RecommendationType;
use App\Models\Admin;
use App\Models\DailyHealthLog;
use App\Models\Recommendation;
use App\Models\User;
use App\Models\UserProfile;
use App\Services\HealthEngine\CyclePhaseMapper;
use App\Services\HealthEngine\HealthDataEngine;
use App\Services\HealthEngine\RecommendationRepository;
use Database\Seeders\RecommendationSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * The home page's "توصیه‌های امروز" card is admin-managed content: the engine
 * resolves it from the recommendations table, keeping its built-in copy only as
 * the fallback for an install that has never been seeded.
 */
class RecommendationTest extends TestCase
{
    use RefreshDatabase;

    private function admin(): Admin
    {
        return Admin::create([
            'name' => 'Test Admin',
            'email' => 'super@ritme.test',
            'password' => 'secret123',
            'role' => Admin::ROLE_SUPER,
            'is_active' => true,
        ]);
    }

    private function recommendation(array $overrides = []): Recommendation
    {
        return Recommendation::create(array_merge([
            'type' => RecommendationType::NUTRITION->value,
            'text' => ['fa' => 'متن فارسی', 'en' => 'English text'],
            'cycle_phase' => CyclePhase::MENSTRUATION->value,
            'is_active' => true,
            'sort_order' => 0,
        ], $overrides));
    }

    private function log(array $attributes): DailyHealthLog
    {
        return DailyHealthLog::create(array_merge([
            'user_id' => User::factory()->create()->id,
            'log_date' => now()->toDateString(),
        ], $attributes));
    }

    // ── Resolution ──────────────────────────────────────────────

    public function test_repository_returns_rows_for_the_phase_and_the_phase_agnostic_ones(): void
    {
        $this->recommendation(['text' => ['fa' => 'قاعدگی', 'en' => 'Menstruation']]);
        $this->recommendation(['cycle_phase' => null, 'text' => ['fa' => 'عمومی', 'en' => 'Any phase']]);
        $this->recommendation([
            'cycle_phase' => CyclePhase::LUTEAL->value,
            'text' => ['fa' => 'لوتئال', 'en' => 'Luteal'],
        ]);

        $tips = (new RecommendationRepository)->forDay(
            CyclePhase::MENSTRUATION,
            CycleSubphase::MENSTRUATION,
            null,
        );

        $this->assertEqualsCanonicalizing(
            ['Menstruation', 'Any phase'],
            array_column($tips, 'en'),
        );
    }

    public function test_inactive_rows_are_never_served(): void
    {
        $this->recommendation(['is_active' => false]);

        $tips = (new RecommendationRepository)->forDay(
            CyclePhase::MENSTRUATION,
            CycleSubphase::MENSTRUATION,
            null,
        );

        $this->assertSame([], $tips);
    }

    public function test_subphase_list_narrows_a_phase_row(): void
    {
        $this->recommendation([
            'cycle_phase' => CyclePhase::LUTEAL->value,
            'cycle_subphases' => [CycleSubphase::LATE_LUTEAL->value, CycleSubphase::PMS_POSSIBLE->value],
            'text' => ['fa' => 'پی‌ام‌اس', 'en' => 'PMS'],
        ]);
        $this->recommendation([
            'cycle_phase' => CyclePhase::LUTEAL->value,
            'cycle_subphases' => null, // every luteal sub-phase
            'text' => ['fa' => 'همیشه', 'en' => 'Always'],
        ]);

        $repository = new RecommendationRepository;

        $early = $repository->forDay(CyclePhase::LUTEAL, CycleSubphase::EARLY_LUTEAL, null);
        $this->assertSame(['Always'], array_column($early, 'en'));

        $late = $repository->forDay(CyclePhase::LUTEAL, CycleSubphase::PMS_POSSIBLE, null);
        $this->assertEqualsCanonicalizing(['PMS', 'Always'], array_column($late, 'en'));
    }

    /** v1.1 sub-phase aliases collapse onto the canonical key content is tagged with. */
    public function test_alias_subphase_matches_its_canonical_row(): void
    {
        $this->recommendation([
            'cycle_subphases' => [CycleSubphase::MENSTRUATION->value],
            'text' => ['fa' => 'قاعدگی', 'en' => 'Menstruation'],
        ]);

        $tips = (new RecommendationRepository)->forDay(
            CyclePhase::MENSTRUATION,
            CycleSubphase::MENSTRUAL, // alias of MENSTRUATION
            null,
        );

        $this->assertSame(['Menstruation'], array_column($tips, 'en'));
    }

    public function test_symptom_triggered_rows_need_the_symptom_to_be_logged(): void
    {
        $this->recommendation([
            'cycle_phase' => null,
            'symptom_trigger' => RecommendationTrigger::HEADACHE->value,
            'text' => ['fa' => 'سردرد', 'en' => 'Headache advice'],
        ]);

        $repository = new RecommendationRepository;

        $this->assertSame(
            [],
            $repository->forDay(CyclePhase::MENSTRUATION, CycleSubphase::MENSTRUATION, null),
        );

        $withSymptom = $repository->forDay(
            CyclePhase::MENSTRUATION,
            CycleSubphase::MENSTRUATION,
            $this->log(['headache_intensity' => 'medium']),
        );

        $this->assertSame(['Headache advice'], array_column($withSymptom, 'en'));
    }

    public function test_symptom_triggered_rows_come_before_plain_phase_rows(): void
    {
        $this->recommendation(['sort_order' => 1, 'text' => ['fa' => 'فاز', 'en' => 'Phase']]);
        $this->recommendation([
            'sort_order' => 99,
            'symptom_trigger' => RecommendationTrigger::FATIGUE->value,
            'text' => ['fa' => 'خستگی', 'en' => 'Fatigue'],
        ]);

        $tips = (new RecommendationRepository)->forDay(
            CyclePhase::MENSTRUATION,
            CycleSubphase::MENSTRUATION,
            $this->log(['fatigue' => true]),
        );

        $this->assertSame(['Fatigue', 'Phase'], array_column($tips, 'en'));
    }

    public function test_title_defaults_to_the_category_label_and_an_override_wins(): void
    {
        $plain = $this->recommendation();
        $this->assertSame(
            RecommendationType::NUTRITION->label('fa'),
            $plain->toTip()['title']['fa'],
        );

        $custom = $this->recommendation(['title' => ['fa' => 'عنوان دلخواه', 'en' => 'Custom']]);
        $this->assertSame('عنوان دلخواه', $custom->toTip()['title']['fa']);
        $this->assertSame('Custom', $custom->toTip()['title']['en']);
    }

    // ── Engine wiring ───────────────────────────────────────────

    private function engineUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);

        UserProfile::create([
            'user_id' => $user->id,
            'birthday' => '1995-05-15',
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => now()->toDateString(),
        ]);

        return $user;
    }

    /** An install whose table was never seeded still shows the built-in copy. */
    public function test_engine_falls_back_to_built_in_tips_when_nothing_is_defined(): void
    {
        $tips = (new HealthDataEngine($this->engineUser(), 'fa'))
            ->calculateForDate(now())['daily_tips'];

        $this->assertNotEmpty($tips);
        $this->assertContains('hydration', array_column($tips, 'type'));
    }

    /** Once any recommendation exists, the database is the only source. */
    public function test_engine_serves_admin_recommendations_once_they_exist(): void
    {
        $this->recommendation([
            'type' => RecommendationType::REST->value,
            'text' => ['fa' => 'فقط این', 'en' => 'Only this'],
        ]);

        $tips = (new HealthDataEngine($this->engineUser(), 'fa'))
            ->calculateForDate(now())['daily_tips'];

        $this->assertSame(['Only this'], array_column($tips, 'en'));
    }

    /** Deactivating every row genuinely empties the card — no zombie fallback. */
    public function test_engine_returns_nothing_when_all_recommendations_are_inactive(): void
    {
        $this->recommendation(['is_active' => false]);

        $tips = (new HealthDataEngine($this->engineUser(), 'fa'))
            ->calculateForDate(now())['daily_tips'];

        $this->assertSame([], $tips);
    }

    public function test_cycle_today_returns_localized_recommendations(): void
    {
        $user = $this->engineUser();
        $this->recommendation([
            'type' => RecommendationType::NUTRITION->value,
            'text' => ['fa' => 'آهن بخورید', 'en' => 'Eat iron'],
        ]);

        Passport::actingAs($user);

        $tip = $this->getJson('/api/v1/cycle/today', ['Accept-Language' => 'fa'])
            ->assertOk()
            ->json('data.calculation.daily_tips.0');

        $this->assertSame([
            'type' => 'nutrition',
            'title' => RecommendationType::NUTRITION->label('fa'),
            'icon' => RecommendationType::NUTRITION->icon(),
            'text' => 'آهن بخورید',
        ], $tip);
    }

    // ── Seeder ──────────────────────────────────────────────────

    public function test_seeder_is_idempotent_and_preserves_admin_edits(): void
    {
        $this->seed(RecommendationSeeder::class);
        $count = Recommendation::count();
        $this->assertGreaterThan(0, $count);

        $edited = Recommendation::where('key', 'menstruation.hydration')->firstOrFail();
        $edited->update(['text' => ['fa' => 'متن ویرایش‌شده', 'en' => 'Edited'], 'is_active' => false]);

        $this->seed(RecommendationSeeder::class);

        $this->assertSame($count, Recommendation::count());
        $edited->refresh();
        $this->assertSame('Edited', $edited->text['en']);
        $this->assertFalse($edited->is_active);
    }

    // ── Admin CRUD ──────────────────────────────────────────────

    public function test_admin_panel_requires_authentication(): void
    {
        $this->get('/admin/recommendations')->assertRedirect('/admin/login');
    }

    public function test_admin_can_create_a_recommendation(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $this->post('/admin/recommendations', [
            'type' => RecommendationType::SLEEP->value,
            'text' => ['fa' => 'زودتر بخوابید', 'en' => 'Sleep earlier'],
            'title' => ['fa' => '', 'en' => ''],
            'cycle_phase' => CyclePhase::LUTEAL->value,
            'cycle_subphases' => [CycleSubphase::LATE_LUTEAL->value],
            'symptom_trigger' => '',
            'sort_order' => 5,
            'is_active' => '1',
        ])->assertRedirect(route('admin.recommendations.index'));

        $row = Recommendation::firstOrFail();
        $this->assertSame('Sleep earlier', $row->text['en']);
        $this->assertSame([CycleSubphase::LATE_LUTEAL->value], $row->cycle_subphases);
        $this->assertNull($row->symptom_trigger);
        // An all-blank title is stored as null so the category label is used.
        $this->assertNull($row->title);
        $this->assertTrue($row->is_active);
    }

    public function test_admin_create_rejects_an_unknown_category(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $this->post('/admin/recommendations', [
            'type' => 'astrology',
            'text' => ['fa' => 'متن'],
        ])->assertSessionHasErrors('type');

        $this->assertSame(0, Recommendation::count());
    }

    /** A sub-phase outside the chosen phase could never fire, so it is rejected. */
    public function test_admin_create_rejects_a_subphase_from_another_phase(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $this->post('/admin/recommendations', [
            'type' => RecommendationType::REST->value,
            'text' => ['fa' => 'متن'],
            'cycle_phase' => CyclePhase::MENSTRUATION->value,
            'cycle_subphases' => [CycleSubphase::LATE_LUTEAL->value],
        ])->assertSessionHasErrors('cycle_subphases.0');

        $this->assertSame(0, Recommendation::count());
    }

    /** With no phase chosen the row is phase-agnostic, so any sub-phase is legal. */
    public function test_admin_can_target_any_subphase_on_a_phase_agnostic_row(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $this->post('/admin/recommendations', [
            'type' => RecommendationType::REST->value,
            'text' => ['fa' => 'متن'],
            'cycle_phase' => '',
            'cycle_subphases' => [CycleSubphase::LATE_LUTEAL->value, CycleSubphase::MENSTRUATION->value],
        ])->assertSessionHasNoErrors();

        $this->assertSame(1, Recommendation::count());
    }

    /**
     * The picker's option list has to be exactly what a calculated day can
     * report: no sub-phase claimed by two phases, none the mapper emits but the
     * picker omits, and none offered that the mapper never returns (a
     * recommendation narrowed to it would save cleanly and never appear).
     */
    public function test_offered_subphases_are_exactly_the_ones_the_mapper_emits(): void
    {
        $mapped = [];
        foreach (CyclePhase::cases() as $phase) {
            foreach ($phase->subphases() as $subphase) {
                $this->assertArrayNotHasKey(
                    $subphase->value,
                    $mapped,
                    "{$subphase->value} is claimed by more than one phase",
                );
                $mapped[$subphase->value] = $phase->value;
            }
        }

        // Walk a whole 28-day cycle and collect what the mapper actually emits.
        $mapper = new CyclePhaseMapper;
        $emitted = [];
        for ($day = 1; $day <= 28; $day++) {
            $emitted[$mapper->subphaseFor($day, 14, 28, 5)->value] = true;
        }

        $this->assertEqualsCanonicalizing(array_keys($emitted), array_keys($mapped));

        // period_expected is content-backed elsewhere (the phase-content page)
        // but is decided by the daily card, never by subphaseFor().
        $this->assertArrayNotHasKey(CycleSubphase::PERIOD_EXPECTED->value, $mapped);
        $this->assertNotContains(
            CycleSubphase::PERIOD_EXPECTED->value,
            CyclePhase::subphaseValuesFor(null),
        );
    }

    public function test_admin_create_rejects_a_subphase_the_engine_never_reports(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $this->post('/admin/recommendations', [
            'type' => RecommendationType::REST->value,
            'text' => ['fa' => 'متن'],
            'cycle_phase' => CyclePhase::LUTEAL->value,
            'cycle_subphases' => [CycleSubphase::PERIOD_EXPECTED->value],
        ])->assertSessionHasErrors('cycle_subphases.0');
    }

    /**
     * The repository is a singleton, so the engines a single request builds
     * (the home page makes at least two) share one load of the table.
     *
     * Worth asserting rather than trusting: a mis-registered binding costs only
     * an extra query, so nothing else in the suite would go red.
     */
    public function test_every_engine_in_a_request_shares_one_load_of_the_table(): void
    {
        $this->seed(RecommendationSeeder::class);
        $user = $this->engineUser();

        $this->assertSame(
            app(RecommendationRepository::class),
            app(RecommendationRepository::class),
            'RecommendationRepository must be bound as a singleton',
        );

        $queries = 0;
        DB::listen(function ($query) use (&$queries) {
            if (str_contains($query->sql, 'recommendations')) {
                $queries++;
            }
        });

        (new HealthDataEngine($user, 'fa'))->calculateForDate(now());
        (new HealthDataEngine($user, 'fa'))->calculateForDate(now());

        $this->assertSame(1, $queries);
    }

    /** One load of the table serves a whole month, not one query per day. */
    public function test_resolving_a_month_of_days_hits_the_table_once(): void
    {
        $this->seed(RecommendationSeeder::class);
        $repository = new RecommendationRepository;

        $queries = 0;
        DB::listen(function ($query) use (&$queries) {
            if (str_contains($query->sql, 'recommendations')) {
                $queries++;
            }
        });

        $mapper = new CyclePhaseMapper;
        for ($day = 1; $day <= 28; $day++) {
            $repository->forDay(
                $mapper->phaseFor($day, 14, 5),
                $mapper->subphaseFor($day, 14, 28, 5),
                // A different symptom each day — the case that defeated the old
                // per-day query cache.
                $this->log(['fatigue' => $day % 2 === 0, 'bloating_intensity' => $day % 3 === 0 ? 'low' : null]),
            );
        }

        $this->assertSame(1, $queries);
    }

    public function test_admin_can_update_toggle_and_delete(): void
    {
        $this->actingAs($this->admin(), 'admin');
        $row = $this->recommendation();

        $this->put("/admin/recommendations/{$row->id}", [
            'type' => RecommendationType::MOOD->value,
            'text' => ['fa' => 'به‌روزشده', 'en' => 'Updated'],
            'title' => ['fa' => 'عنوان', 'en' => 'Title'],
            'cycle_phase' => '',
            'sort_order' => 2,
        ])->assertRedirect(route('admin.recommendations.index'));

        $row->refresh();
        $this->assertSame('Updated', $row->text['en']);
        $this->assertSame('Title', $row->title['en']);
        $this->assertNull($row->cycle_phase);
        // The checkbox was absent from the request, so the row is now hidden.
        $this->assertFalse($row->is_active);

        $this->post("/admin/recommendations/{$row->id}/toggle");
        $this->assertTrue($row->refresh()->is_active);

        $this->delete("/admin/recommendations/{$row->id}");
        $this->assertSame(0, Recommendation::count());
    }

    public function test_admin_index_filters_by_phase_and_category(): void
    {
        $this->actingAs($this->admin(), 'admin');

        // Distinctive copy: the phase and category names themselves also appear
        // in the page's own filter dropdowns, so asserting on them proves nothing.
        $this->recommendation(['text' => ['fa' => 'ردیف اول برای آزمون', 'en' => 'First row']]);
        $this->recommendation([
            'cycle_phase' => CyclePhase::LUTEAL->value,
            'type' => RecommendationType::MOOD->value,
            'text' => ['fa' => 'ردیف دوم برای آزمون', 'en' => 'Second row'],
        ]);

        $this->get('/admin/recommendations?phase='.CyclePhase::LUTEAL->value)
            ->assertOk()
            ->assertSee('ردیف دوم برای آزمون')
            ->assertDontSee('ردیف اول برای آزمون');

        $this->get('/admin/recommendations?type='.RecommendationType::NUTRITION->value)
            ->assertOk()
            ->assertSee('ردیف اول برای آزمون')
            ->assertDontSee('ردیف دوم برای آزمون');
    }
}
