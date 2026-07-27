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
            'difficulty' => 'easy',
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

    public function test_phase_specific_challenges_are_excluded_from_other_phases(): void
    {
        $lutealOnly = $this->challenge(['cycle_phase' => 'luteal']);
        $anyPhase = $this->challenge();

        $user = User::factory()->create();
        $date = Carbon::parse('2026-03-05');

        $this->assertSame($anyPhase->id, $this->service->challengeFor($user, $date, 'menstruation')->id);
        $this->assertContains(
            $this->service->challengeFor($user, $date, 'luteal')->id,
            [$lutealOnly->id, $anyPhase->id],
        );
    }

    public function test_hard_challenges_are_locked_until_the_streak_unlocks_them(): void
    {
        $hard = $this->challenge(['difficulty' => 'hard']);
        $easy = $this->challenge(['difficulty' => 'easy']);

        $user = User::factory()->create();
        $date = Carbon::parse('2026-03-05');

        // No streak → only `easy` is unlocked.
        $this->assertSame($easy->id, $this->service->challengeFor($user, $date)->id);

        // 8-day streak unlocks `hard`; `easy` was completed yesterday so it is
        // also out of the pool, leaving `hard` as the only candidate.
        for ($i = 1; $i <= 8; $i++) {
            $this->complete($user, $easy, $date->copy()->subDays($i)->toDateString());
        }

        $this->assertSame($hard->id, $this->service->challengeFor($user, $date)->id);
    }

    public function test_empty_pool_yields_no_challenge(): void
    {
        $this->challenge(['is_active' => false]);

        $this->assertNull(
            $this->service->challengeFor(User::factory()->create(), Carbon::parse('2026-03-05')),
        );
    }

    // ── Streak ──────────────────────────────────────────────────

    public function test_streak_counts_consecutive_days_and_survives_an_unfinished_today(): void
    {
        $challenge = $this->challenge();
        $user = User::factory()->create();
        $today = Carbon::parse('2026-03-05');

        foreach ([1, 2, 3] as $back) {
            $this->complete($user, $challenge, $today->copy()->subDays($back)->toDateString());
        }

        // Today is not done yet — yesterday still anchors the streak.
        $this->assertSame(3, $this->service->currentStreak($user, $today));

        $this->complete($user, $challenge, $today->toDateString());
        $this->assertSame(4, $this->service->currentStreak($user, $today));
    }

    public function test_a_missed_day_breaks_the_streak(): void
    {
        $challenge = $this->challenge();
        $user = User::factory()->create();
        $today = Carbon::parse('2026-03-05');

        foreach ([1, 3, 4, 5] as $back) {
            $this->complete($user, $challenge, $today->copy()->subDays($back)->toDateString());
        }

        $this->assertSame(1, $this->service->currentStreak($user, $today));
        $this->assertSame(3, $this->service->longestStreak($user));
    }

    public function test_week_days_marks_the_last_seven_days(): void
    {
        $challenge = $this->challenge();
        $user = User::factory()->create();
        $today = Carbon::parse('2026-03-05');

        $this->complete($user, $challenge, $today->copy()->subDays(2)->toDateString());

        $week = $this->service->weekDays($user, $today);

        $this->assertCount(7, $week);
        $this->assertTrue($week[6]['is_today']);
        $this->assertTrue($week[4]['is_completed']);
        $this->assertFalse($week[5]['is_completed']);
    }

    // ── API ─────────────────────────────────────────────────────

    public function test_challenge_section_exposes_streak_and_completion(): void
    {
        $challenge = $this->challenge();
        $user = User::factory()->create();
        Passport::actingAs($user);

        $response = $this->getJson('/api/v1/home/sections/challenge');

        $response->assertOk()
            ->assertJsonPath('data.section.data.id', $challenge->id)
            ->assertJsonPath('data.section.data.is_completed', false)
            ->assertJsonPath('data.section.data.streak', 0)
            ->assertJsonCount(7, 'data.section.data.week_days');
    }

    public function test_toggle_marks_completion_and_returns_the_new_streak(): void
    {
        $challenge = $this->challenge();
        $user = User::factory()->create();
        Passport::actingAs($user);

        $this->postJson("/api/v1/home/challenges/{$challenge->id}/toggle")
            ->assertOk()
            ->assertJsonPath('data.is_completed', true)
            ->assertJsonPath('data.streak', 1);

        $this->assertDatabaseHas('user_challenge_completions', [
            'user_id' => $user->id,
            'challenge_id' => $challenge->id,
        ]);

        $this->postJson("/api/v1/home/challenges/{$challenge->id}/toggle")
            ->assertOk()
            ->assertJsonPath('data.is_completed', false)
            ->assertJsonPath('data.streak', 0);

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
            'difficulty' => 'easy',
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
