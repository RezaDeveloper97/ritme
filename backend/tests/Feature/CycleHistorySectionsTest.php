<?php

namespace Tests\Feature;

use App\Models\CycleHistory;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Locks the contract the home "سیکل‌های من" and "خلاصه سیکل" cards render from:
 * every cycle length is derived from the logged timeline (start → next start),
 * an unfinished period never reports a partial duration, and aggregate rows only
 * appear once they summarize more than one cycle.
 */
class CycleHistorySectionsTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(string $lastPeriodStart): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        UserProfile::create([
            'user_id' => $user->id,
            'last_period_start' => $lastPeriodStart,
            'cycle_duration' => 28,
            'period_duration' => 5,
        ]);
        Passport::actingAs($user);

        return $user;
    }

    private function period(User $user, string $start, ?string $end): void
    {
        CycleHistory::create([
            'user_id' => $user->id,
            'period_start_date' => $start,
            'period_end_date' => $end,
            'is_confirmed' => true,
            'is_estimated' => false,
            'source' => 'user',
        ]);
    }

    private function section(string $key): array
    {
        return $this->getJson("/api/v1/home/sections/{$key}")->assertOk()->json('data.section');
    }

    private function items(array $section): array
    {
        return collect($section['data']['items'])->keyBy('key')->all();
    }

    /**
     * Three logged periods 30 and 27 days apart: each previous cycle reports the
     * gap to the period that followed it, and the running one reports none.
     */
    public function test_previous_cycles_report_the_gap_to_the_next_period(): void
    {
        $start = now()->subDays(57)->toDateString();
        $user = $this->actingUser($start);

        $this->period($user, $start, now()->subDays(53)->toDateString());               // 5-day bleed
        $this->period($user, now()->subDays(27)->toDateString(), now()->subDays(23)->toDateString());
        $this->period($user, now()->toDateString(), null);                              // ongoing

        $data = $this->section('my_cycles')['data'];

        $this->assertSame(2, $data['previous_count']);
        $this->assertCount(2, $data['previous']);

        // Newest first: the cycle that started 27 days ago ran until today (27),
        // and the one before it ran 30 days.
        $this->assertSame(27, $data['previous'][0]['cycle_length']);
        $this->assertSame(5, $data['previous'][0]['period_length']);
        $this->assertSame(30, $data['previous'][1]['cycle_length']);

        // The current cycle is still open — no end date, no measured length.
        $this->assertTrue($data['current']['is_ongoing']);
        $this->assertNull($data['current']['period_end_date']);
        $this->assertSame(now()->toDateString(), $data['current']['started_at']);
        $this->assertSame(1, $data['current']['cycle_day']);

        // Averages cover both completed cycles: (27 + 30) / 2.
        $this->assertSame(29, $data['averages']['cycle_length']);
        $this->assertSame(2, $data['averages']['based_on_cycles']);
    }

    /** A single logged period has nothing behind it — the list is empty, not fabricated. */
    public function test_a_lone_period_yields_no_previous_cycles(): void
    {
        $start = now()->subDays(3)->toDateString();
        $user = $this->actingUser($start);
        $this->period($user, $start, null);

        $data = $this->section('my_cycles')['data'];

        $this->assertSame(0, $data['previous_count']);
        $this->assertSame([], $data['previous']);
        $this->assertNull($data['averages']['cycle_length']);
    }

    /** The summary reads the last completed cycle, not the one in progress. */
    public function test_summary_reports_the_last_completed_cycle_and_period(): void
    {
        $start = now()->subDays(30)->toDateString();
        $user = $this->actingUser($start);

        $this->period($user, $start, now()->subDays(26)->toDateString()); // 5-day bleed
        $this->period($user, now()->toDateString(), null);                // ongoing

        $items = $this->items($this->section('cycle_summary'));

        $this->assertSame(30, $items['last_cycle_length']['value']);
        $this->assertSame('normal', $items['last_cycle_length']['status']);
        $this->assertSame('days', $items['last_cycle_length']['unit']);

        // The open period contributes nothing; the finished one before it does.
        $this->assertSame(5, $items['last_period_duration']['value']);
        $this->assertSame('normal', $items['last_period_duration']['status']);
    }

    /** With no history at all the rows exist but read as "not known yet". */
    public function test_summary_rows_are_unknown_without_history(): void
    {
        $this->actingUser(now()->subDays(5)->toDateString());

        $section = $this->section('cycle_summary');
        $items = $this->items($section);

        $this->assertFalse($section['data']['has_history']);
        $this->assertNull($items['last_cycle_length']['value']);
        $this->assertSame('unknown', $items['last_cycle_length']['status']);
        $this->assertNotNull($items['last_cycle_length']['hint']);
        $this->assertSame('unknown', $items['cycle_variability']['status']);

        // Aggregates need more than one cycle — they must not appear.
        $this->assertArrayNotHasKey('average_cycle_length', $items);
        $this->assertArrayNotHasKey('cycle_length_range', $items);
    }

    /** A cycle outside 21–35 days is flagged descriptively, never as "abnormal". */
    public function test_out_of_range_cycle_is_reported_as_outside_the_usual_range(): void
    {
        $start = now()->subDays(50)->toDateString();
        $user = $this->actingUser($start);

        $this->period($user, $start, now()->subDays(46)->toDateString());
        $this->period($user, now()->subDays(12)->toDateString(), now()->subDays(11)->toDateString());

        $items = $this->items($this->section('cycle_summary'));

        $this->assertSame(38, $items['last_cycle_length']['value']);
        $this->assertSame('outside_range', $items['last_cycle_length']['status']);
        // 2-day bleed is short but still within the usual 2–8 range.
        $this->assertSame(2, $items['last_period_duration']['value']);
        $this->assertSame('normal', $items['last_period_duration']['status']);
    }

    /** Aggregate rows appear once two valid cycles exist, and summarize both. */
    public function test_aggregate_rows_appear_with_two_valid_cycles(): void
    {
        $start = now()->subDays(57)->toDateString();
        $user = $this->actingUser($start);

        $this->period($user, $start, now()->subDays(53)->toDateString());
        $this->period($user, now()->subDays(27)->toDateString(), now()->subDays(23)->toDateString());
        $this->period($user, now()->toDateString(), null);

        $section = $this->section('cycle_summary');
        $items = $this->items($section);

        $this->assertSame(2, $section['data']['based_on_cycles']);
        $this->assertSame(29, $items['average_cycle_length']['value']);
        $this->assertSame(27, $items['cycle_length_range']['value_min']);
        $this->assertSame(30, $items['cycle_length_range']['value_max']);
    }
}
