<?php

namespace Tests\Feature;

use App\Models\Admin;
use App\Models\Challenge;
use App\Models\User;
use App\Models\UserChallengeCompletion;
use App\Services\Challenges\DailyChallengeService;
use Carbon\Carbon;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

class DailyChallengeTest extends TestCase
{
    use RefreshDatabase;

    private DailyChallengeService $service;

    protected function setUp(): void
    {
        parent::setUp();
        $this->service = new DailyChallengeService;
    }

    private function challenge(array $overrides = []): Challenge
    {
        static $n = 0;
        $n++;

        return Challenge::create(array_merge([
            'slug' => "challenge-{$n}",
            'title' => ['fa' => "چالش {$n}", 'en' => "Challenge {$n}"],
            'description' => ['fa' => 'توضیح', 'en' => 'Description'],
            'category' => 'tracking',
            'is_active' => true,
            'sort_order' => $n,
        ], $overrides));
    }

    private function complete(User $user, Challenge $challenge, string $date): void
    {
        UserChallengeCompletion::create([
            'user_id' => $user->id,
            'challenge_id' => $challenge->id,
            'completion_date' => $date,
            'completed_at' => Carbon::parse($date),
        ]);
    }

    // ── Selection ───────────────────────────────────────────────

    public function test_pick_is_stable_within_a_day_and_differs_between_users(): void
    {
        for ($i = 0; $i < 8; $i++) {
            $this->challenge();
        }

        $date = Carbon::parse('2026-03-05');
        $alice = User::factory()->create();

        $first = $this->service->challengeFor($alice, $date);
        $this->assertSame($first->id, $this->service->challengeFor($alice, $date)->id);

        // Two given users may legitimately collide; the point is that the seed
        // includes the user, so a group of users must not all see one challenge.
        $picks = [];
        for ($i = 0; $i < 10; $i++) {
            $picks[] = $this->service->challengeFor(User::factory()->create(), $date)->id;
        }

        $this->assertGreaterThan(
            1,
            count(array_unique($picks)),
            'The daily pick should vary between users',
        );
    }

    public function test_completing_today_does_not_change_todays_pick(): void
    {
        for ($i = 0; $i < 5; $i++) {
            $this->challenge();
        }

        $user = User::factory()->create();
        $date = Carbon::parse('2026-03-05');

        $picked = $this->service->challengeFor($user, $date);
        $this->service->toggle($user, $picked, $date);

        $this->assertSame($picked->id, $this->service->challengeFor($user, $date)->id);
    }

    public function test_recently_completed_challenges_are_skipped(): void
    {
        $a = $this->challenge();
        $b = $this->challenge();

        $user = User::factory()->create();
        $date = Carbon::parse('2026-03-05');

        $picked = $this->service->challengeFor($user, $date);
        $this->complete($user, $picked, $date->copy()->subDay()->toDateString());

        $other = $picked->id === $a->id ? $b : $a;
        $this->assertSame($other->id, $this->service->challengeFor($user, $date)->id);
    }

    public function test_challenges_outside_the_cycle_day_range_are_excluded(): void
    {
        $this->challenge(['cycle_day_from' => 20, 'cycle_day_to' => 25]);
        $anyDay = $this->challenge();

        $user = User::factory()->create();
        $date = Carbon::parse('2026-03-05');

        $this->assertSame($anyDay->id, $this->service->challengeFor($user, $date, 3)->id);
    }

    public function test_a_challenge_targeting_todays_cycle_day_wins_over_untargeted_ones(): void
    {
        for ($i = 0; $i < 6; $i++) {
            $this->challenge();
        }
        $targeted = $this->challenge(['cycle_day_from' => 6, 'cycle_day_to' => 12]);

        $user = User::factory()->create();
        $date = Carbon::parse('2026-03-05');

        $this->assertSame($targeted->id, $this->service->challengeFor($user, $date, 7)->id);
        // …and only on its own days.
        $this->assertNotSame($targeted->id, $this->service->challengeFor($user, $date, 13)->id);
    }

    public function test_open_ended_ranges_are_honoured_on_both_sides(): void
    {
        $fromDay20 = $this->challenge(['cycle_day_from' => 20, 'cycle_day_to' => null]);
        $upToDay5 = $this->challenge(['cycle_day_from' => null, 'cycle_day_to' => 5]);
        $this->challenge();

        $user = User::factory()->create();
        $date = Carbon::parse('2026-03-05');

        $this->assertSame($upToDay5->id, $this->service->challengeFor($user, $date, 2)->id);
        $this->assertSame($fromDay20->id, $this->service->challengeFor($user, $date, 30)->id);
    }

    public function test_users_without_cycle_data_only_see_untargeted_challenges(): void
    {
        $this->challenge(['cycle_day_from' => 1, 'cycle_day_to' => 5]);
        $anyDay = $this->challenge();

        $user = User::factory()->create();

        $this->assertSame(
            $anyDay->id,
            $this->service->challengeFor($user, Carbon::parse('2026-03-05'), null)->id,
        );
    }

    public function test_day_targeting_falls_back_rather_than_leaving_the_user_empty_handed(): void
    {
        // The whole pool is targeted at days the user isn't on — showing an
        // off-range challenge beats showing no card at all.
        $only = $this->challenge(['cycle_day_from' => 20, 'cycle_day_to' => 25]);

        $this->assertSame(
            $only->id,
            $this->service->challengeFor(User::factory()->create(), Carbon::parse('2026-03-05'), 3)->id,
        );
    }

    public function test_empty_pool_yields_no_challenge(): void
    {
        $this->challenge(['is_active' => false]);

        $this->assertNull(
            $this->service->challengeFor(User::factory()->create(), Carbon::parse('2026-03-05')),
        );
    }

    // ── API ─────────────────────────────────────────────────────

    public function test_challenge_section_exposes_the_challenge_and_its_completion(): void
    {
        $challenge = $this->challenge();
        $user = User::factory()->create();
        Passport::actingAs($user);

        $response = $this->getJson('/api/v1/home/sections/challenge');

        $response->assertOk()
            ->assertJsonPath('data.section.data.id', $challenge->id)
            ->assertJsonPath('data.section.data.is_completed', false)
            // The card is a task with a tick and nothing else — no streak,
            // no history strip, no encouragement copy.
            ->assertJsonMissingPath('data.section.data.streak')
            ->assertJsonMissingPath('data.section.data.week_days')
            ->assertJsonMissingPath('data.section.data.status_message');
    }

    public function test_toggle_marks_and_unmarks_todays_completion(): void
    {
        $challenge = $this->challenge();
        $user = User::factory()->create();
        Passport::actingAs($user);

        $this->postJson("/api/v1/home/challenges/{$challenge->id}/toggle")
            ->assertOk()
            ->assertJsonPath('data.is_completed', true)
            ->assertJsonMissingPath('data.streak');

        $this->assertDatabaseHas('user_challenge_completions', [
            'user_id' => $user->id,
            'challenge_id' => $challenge->id,
        ]);

        $this->postJson("/api/v1/home/challenges/{$challenge->id}/toggle")
            ->assertOk()
            ->assertJsonPath('data.is_completed', false);

        $this->assertDatabaseCount('user_challenge_completions', 0);
    }

    public function test_toggle_requires_authentication(): void
    {
        $challenge = $this->challenge();

        $this->postJson("/api/v1/home/challenges/{$challenge->id}/toggle")->assertUnauthorized();
    }

    // ── Seeding ─────────────────────────────────────────────────

    public function test_seeder_adopts_pre_slug_rows_instead_of_duplicating_them(): void
    {
        // A row as it exists on installs seeded before `slug` was introduced.
        $legacy = Challenge::create([
            'title' => ['fa' => 'ثبت دمای بدن', 'en' => 'Log your body temperature'],
            'description' => ['fa' => 'قدیمی', 'en' => 'old'],
            'category' => 'tracking',
            'is_active' => true,
            'sort_order' => 1,
        ]);

        $user = User::factory()->create();
        $this->complete($user, $legacy, '2026-03-01');

        $this->seed(\Database\Seeders\ChallengeSeeder::class);
        $this->seed(\Database\Seeders\ChallengeSeeder::class);

        $legacy->refresh();

        $this->assertSame('log-basal-temperature', $legacy->slug);
        $this->assertSame(
            1,
            Challenge::query()->where('slug', 'log-basal-temperature')->count(),
            'Re-seeding must not duplicate an adopted challenge',
        );
        // The user's history survives adoption.
        $this->assertDatabaseHas('user_challenge_completions', [
            'user_id' => $user->id,
            'challenge_id' => $legacy->id,
        ]);
    }

    // ── Admin CRUD ──────────────────────────────────────────────

    private function admin(string $email): Admin
    {
        return Admin::create([
            'name' => 'Test Admin',
            'email' => $email,
            'password' => 'secret123',
            'role' => Admin::ROLE_SUPER,
            'is_active' => true,
        ]);
    }

    public function test_admin_forms_render_the_cycle_day_range_fields(): void
    {
        $admin = $this->admin('forms@ritme.test');
        $challenge = $this->challenge(['cycle_day_from' => 6, 'cycle_day_to' => 12]);

        $this->actingAs($admin, 'admin')
            ->get(route('admin.challenges.create'))
            ->assertOk()
            ->assertSee('name="cycle_day_from"', false)
            ->assertSee('name="cycle_day_to"', false);

        $this->actingAs($admin, 'admin')
            ->get(route('admin.challenges.edit', $challenge))
            ->assertOk()
            ->assertSee('value="6"', false)
            ->assertSee('value="12"', false);
    }

    public function test_admin_can_author_a_challenge_for_a_cycle_day_range(): void
    {
        $this->actingAs($this->admin('crud@ritme.test'), 'admin')
            ->post(route('admin.challenges.store'), [
                'title' => ['fa' => 'کشش ملایم', 'en' => 'Gentle stretch'],
                'description' => ['fa' => 'توضیح', 'en' => 'Description'],
                'cycle_day_from' => 6,
                'cycle_day_to' => 12,
                'category' => 'exercise',
                'sort_order' => 3,
                'is_active' => '1',
            ])
            ->assertRedirect(route('admin.challenges.index'));

        $this->assertDatabaseHas('challenges', [
            'cycle_day_from' => 6,
            'cycle_day_to' => 12,
        ]);
    }

    public function test_admin_cannot_save_an_inverted_or_out_of_bounds_range(): void
    {
        $admin = $this->admin('validation@ritme.test');

        $payload = [
            'title' => ['fa' => 'تست'],
            'category' => 'exercise',
        ];

        // End before start.
        $this->actingAs($admin, 'admin')
            ->post(route('admin.challenges.store'), $payload + ['cycle_day_from' => 12, 'cycle_day_to' => 6])
            ->assertSessionHasErrors('cycle_day_to');

        // Past the 35-day cap.
        $this->actingAs($admin, 'admin')
            ->post(route('admin.challenges.store'), $payload + ['cycle_day_from' => 1, 'cycle_day_to' => 40])
            ->assertSessionHasErrors('cycle_day_to');

        $this->assertDatabaseCount('challenges', 0);
    }

    public function test_admin_list_can_be_filtered_by_cycle_day(): void
    {
        $early = $this->challenge(['cycle_day_from' => 1, 'cycle_day_to' => 5]);
        $late = $this->challenge(['cycle_day_from' => 20, 'cycle_day_to' => 25]);
        $anyDay = $this->challenge();

        // Day 3 shows the matching range plus the untargeted fallback, and
        // hides the range that doesn't cover it.
        $this->actingAs($this->admin('filter@ritme.test'), 'admin')
            ->get(route('admin.challenges.index', ['cycle_day' => 3]))
            ->assertOk()
            ->assertSeeText($early->title['fa'])
            ->assertSeeText($anyDay->title['fa'])
            ->assertDontSeeText($late->title['fa']);
    }

    // ── Admin report ────────────────────────────────────────────

    public function test_admin_can_see_who_completed_challenges(): void
    {
        $challenge = $this->challenge();
        $user = User::factory()->create(['name' => 'زهرا']);
        $this->complete($user, $challenge, '2026-03-04');

        $admin = Admin::create([
            'name' => 'Test Admin',
            'email' => 'super@ritme.test',
            'password' => 'secret123',
            'role' => Admin::ROLE_SUPER,
            'is_active' => true,
        ]);

        $this->actingAs($admin, 'admin')
            ->get(route('admin.challenge-completions.index'))
            ->assertOk()
            ->assertSee('زهرا')
            ->assertSee($challenge->title['fa']);
    }

    public function test_admin_report_can_be_filtered_by_challenge(): void
    {
        $seen = $this->challenge();
        $hidden = $this->challenge();
        $user = User::factory()->create(['name' => 'زهرا']);
        $this->complete($user, $seen, '2026-03-04');
        $this->complete($user, $hidden, '2026-03-03');

        $admin = Admin::create([
            'name' => 'Test Admin',
            'email' => 'super2@ritme.test',
            'password' => 'secret123',
            'role' => Admin::ROLE_SUPER,
            'is_active' => true,
        ]);

        $this->actingAs($admin, 'admin')
            ->get(route('admin.challenge-completions.index', ['challenge_id' => $seen->id]))
            ->assertOk()
            // Only the filtered challenge's completion is counted (the other
            // title still appears in the filter dropdown, so assert the stats).
            ->assertSeeText('مجموع انجام‌ها: 1')
            ->assertSeeText($seen->title['fa']);
    }
}
