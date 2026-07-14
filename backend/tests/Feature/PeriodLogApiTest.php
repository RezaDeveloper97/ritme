<?php

namespace Tests\Feature;

use App\Models\CycleHistory;
use App\Models\User;
use App\Models\UserProfile;
use Carbon\Carbon;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Covers the period logging endpoints (start / end / status): starting a period
 * creates a CycleHistory record and re-anchors the profile LMP, and ending it
 * closes that record with an end date and bleeding length.
 */
class PeriodLogApiTest extends TestCase
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

    public function test_status_reports_inactive_when_no_open_period(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/cycle/period/status')
            ->assertOk()
            ->assertJsonPath('data.active', false);
    }

    public function test_start_period_creates_history_and_reanchors_lmp(): void
    {
        $user = $this->actingUser();
        $today = now()->toDateString();

        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])
            ->assertOk()
            ->assertJsonPath('success', true)
            ->assertJsonPath('data.active', true)
            ->assertJsonPath('data.period_start_date', $today);

        $this->assertDatabaseHas('cycle_histories', [
            'user_id' => $user->id,
            'period_end_date' => null,
        ]);
        $this->assertSame($today, $user->profile->fresh()->last_period_start->toDateString());

        // Status now reports an ongoing period.
        $this->getJson('/api/v1/cycle/period/status')
            ->assertOk()
            ->assertJsonPath('data.active', true);
    }

    public function test_start_period_is_idempotent_for_same_day(): void
    {
        $user = $this->actingUser();
        $today = now()->toDateString();

        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])->assertOk();
        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])->assertOk();

        $this->assertSame(1, CycleHistory::where('user_id', $user->id)
            ->whereDate('period_start_date', $today)->count());
    }

    public function test_end_period_closes_open_record_with_bleeding_length(): void
    {
        $user = $this->actingUser();
        $start = now()->subDays(4)->toDateString();
        $end = now()->toDateString();

        $this->postJson('/api/v1/cycle/period/start', ['date' => $start])->assertOk();

        $this->postJson('/api/v1/cycle/period/end', ['date' => $end])
            ->assertOk()
            ->assertJsonPath('data.active', false)
            ->assertJsonPath('data.period_end_date', $end);

        $record = CycleHistory::where('user_id', $user->id)->latest('period_start_date')->first();
        $this->assertSame($end, $record->period_end_date->toDateString());
        $this->assertSame(5, $record->bleeding_length); // 4 days later, inclusive
    }

    public function test_restarting_an_ended_period_reopens_it_so_it_can_be_ended_again(): void
    {
        $this->actingUser();
        $today = now()->toDateString();

        // Start, end, then start again on the same day. The restart must clear
        // the prior end date so the period is ongoing and End works again
        // (regression: End used to report "no ongoing period").
        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])->assertOk();
        $this->postJson('/api/v1/cycle/period/end', ['date' => $today])->assertOk();

        $this->postJson('/api/v1/cycle/period/start', ['date' => $today])
            ->assertOk()
            ->assertJsonPath('data.active', true)
            ->assertJsonPath('data.period_end_date', null);

        $this->getJson('/api/v1/cycle/period/status')->assertJsonPath('data.active', true);

        $this->postJson('/api/v1/cycle/period/end', ['date' => $today])
            ->assertOk()
            ->assertJsonPath('data.active', false)
            ->assertJsonPath('data.period_end_date', $today);
    }

    public function test_end_period_without_ongoing_period_fails(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/cycle/period/end')
            ->assertStatus(422)
            ->assertJsonPath('success', false);
    }

    public function test_end_before_start_is_rejected(): void
    {
        $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(2)->toDateString()])->assertOk();

        $this->postJson('/api/v1/cycle/period/end', ['date' => now()->subDays(5)->toDateString()])
            ->assertStatus(422);
    }

    public function test_history_lists_logged_periods_newest_first(): void
    {
        $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(30)->toDateString()])->assertOk();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(2)->toDateString()])->assertOk();

        $response = $this->getJson('/api/v1/cycle/period/history')->assertOk();
        $periods = $response->json('data.periods');

        $this->assertCount(2, $periods);
        $this->assertSame(now()->subDays(2)->toDateString(), $periods[0]['period_start_date']);
        $this->assertSame(now()->subDays(30)->toDateString(), $periods[1]['period_start_date']);
    }

    public function test_update_moves_period_and_reanchors_lmp(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(3)->toDateString()])->assertOk();
        $id = CycleHistory::where('user_id', $user->id)->value('id');

        $newStart = now()->subDays(5)->toDateString();
        $newEnd = now()->subDays(1)->toDateString();

        $this->putJson("/api/v1/cycle/period/{$id}", ['start_date' => $newStart, 'end_date' => $newEnd])
            ->assertOk()
            ->assertJsonPath('data.period_start_date', $newStart)
            ->assertJsonPath('data.period_end_date', $newEnd);

        $record = CycleHistory::find($id);
        $this->assertSame(5, $record->bleeding_length);
        $this->assertSame($newStart, $user->profile->fresh()->last_period_start->toDateString());
    }

    public function test_update_recomputes_cycle_lengths(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(40)->toDateString()])->assertOk();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(10)->toDateString()])->assertOk();

        $second = CycleHistory::where('user_id', $user->id)->orderByDesc('period_start_date')->first();
        $this->assertSame(30, (int) $second->cycle_length);

        // Move the second period 2 days earlier → the gap shrinks to 28.
        $this->putJson("/api/v1/cycle/period/{$second->id}", [
            'start_date' => now()->subDays(12)->toDateString(),
        ])->assertOk();

        $this->assertSame(28, (int) $second->fresh()->cycle_length);
    }

    public function test_delete_removes_period_and_reanchors_to_previous(): void
    {
        $user = $this->actingUser();
        $older = now()->subDays(40)->toDateString();
        $this->postJson('/api/v1/cycle/period/start', ['date' => $older])->assertOk();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(5)->toDateString()])->assertOk();

        $latest = CycleHistory::where('user_id', $user->id)->orderByDesc('period_start_date')->first();

        $this->deleteJson("/api/v1/cycle/period/{$latest->id}")->assertOk();

        $this->assertDatabaseMissing('cycle_histories', ['id' => $latest->id]);
        // The anchor falls back to the remaining (older) period.
        $this->assertSame($older, $user->profile->fresh()->last_period_start->toDateString());
    }

    public function test_delete_last_period_clears_anchor(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(5)->toDateString()])->assertOk();
        $id = CycleHistory::where('user_id', $user->id)->value('id');

        $this->deleteJson("/api/v1/cycle/period/{$id}")->assertOk();

        $this->assertNull($user->profile->fresh()->last_period_start);
    }

    public function test_update_rejects_end_before_start_and_foreign_records(): void
    {
        $user = $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(3)->toDateString()])->assertOk();
        $id = CycleHistory::where('user_id', $user->id)->value('id');

        $this->putJson("/api/v1/cycle/period/{$id}", [
            'start_date' => now()->subDays(2)->toDateString(),
            'end_date' => now()->subDays(4)->toDateString(),
        ])->assertStatus(422);

        // Another user's record is invisible → 404, never editable.
        $other = User::factory()->create(['mobile' => '09120000001']);
        $foreign = CycleHistory::create([
            'user_id' => $other->id,
            'period_start_date' => now()->subDays(3)->toDateString(),
            'is_confirmed' => true,
        ]);

        $this->putJson("/api/v1/cycle/period/{$foreign->id}", [
            'start_date' => now()->subDays(2)->toDateString(),
        ])->assertStatus(404);
        $this->deleteJson("/api/v1/cycle/period/{$foreign->id}")->assertStatus(404);
    }

    public function test_closely_logged_periods_do_not_collapse_the_cycle(): void
    {
        // Regression: periods logged a few days apart (correcting mistakes)
        // created cycle lengths of 4/8 days; the engine averaged them into a
        // ~6-day cycle and painted the entire month as menstruation.
        $this->actingUser();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(20)->toDateString()])->assertOk();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(16)->toDateString()])->assertOk();
        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->subDays(8)->toDateString()])->assertOk();

        $calc = $this->getJson('/api/v1/cycle/today')->assertOk()->json('data.calculation');

        // Implausible 4/8-day gaps are ignored → the profile's 28-day cycle wins.
        $this->assertSame(28, $calc['cycle_length_used']);
        $this->assertNotSame('menstruation', $calc['phase']);
    }

    public function test_edited_end_date_changes_rendered_bleeding_length(): void
    {
        // The calendar must reflect the *logged* bleeding range, not the
        // profile's default duration (regression: editing the end date had no
        // visible effect because determinePhase always used period_duration).
        $this->actingUser(); // profile default: 5 days
        $start = now()->subDays(9);
        $this->postJson('/api/v1/cycle/period/start', ['date' => $start->toDateString()])->assertOk();
        $this->postJson('/api/v1/cycle/period/end', ['date' => $start->copy()->addDays(6)->toDateString()])->assertOk();

        // Day 7 of the cycle is inside the logged 7-day bleed → menstruation.
        $day7 = $start->copy()->addDays(6)->toDateString();
        $calc = $this->getJson("/api/v1/cycle/date/{$day7}")->assertOk()->json('data.calculation');
        $this->assertSame('menstruation', $calc['phase']);

        // Shorten the period to 3 days → the same day is no longer bleeding.
        $id = CycleHistory::whereDate('period_start_date', $start->toDateString())->value('id');
        $this->putJson("/api/v1/cycle/period/{$id}", [
            'start_date' => $start->toDateString(),
            'end_date' => $start->copy()->addDays(2)->toDateString(),
        ])->assertOk();

        $calc = $this->getJson("/api/v1/cycle/date/{$day7}")->assertOk()->json('data.calculation');
        $this->assertNotSame('menstruation', $calc['phase']);
    }

    public function test_next_cycle_is_predicted_automatically_from_a_logged_period(): void
    {
        // The whole product promise: log days 1–5 of a 28-day cycle and the
        // engine must, with no further input, predict the NEXT period exactly
        // 28 days after the logged start, plus ovulation (day 14) and PMS in
        // the late luteal phase — for any future date the calendar asks about.
        $this->actingUser(); // cycle_duration 28, period_duration 5
        $start = now()->subDays(6); // logged period: days 1–5 fully in the past
        $this->postJson('/api/v1/cycle/period/start', ['date' => $start->toDateString()])->assertOk();
        $this->postJson('/api/v1/cycle/period/end', ['date' => $start->copy()->addDays(4)->toDateString()])->assertOk();

        $phaseOn = function (Carbon $date): array {
            $calc = $this->getJson("/api/v1/cycle/date/{$date->toDateString()}")
                ->assertOk()->json('data.calculation');

            return [$calc['phase'], $calc['is_pms_window'] ?? false, $calc['cycle_day']];
        };

        // Day 28 (start+27) is the last day of this cycle — not menstruation.
        [$phase] = $phaseOn($start->copy()->addDays(27));
        $this->assertNotSame('menstruation', $phase);

        // Day 29 (start+28) is day 1 of the predicted next cycle → menstruation.
        [$phase, , $cycleDay] = $phaseOn($start->copy()->addDays(28));
        $this->assertSame('menstruation', $phase);
        $this->assertSame(1, $cycleDay);

        // Ovulation lands on cycle day 14 (28 − 14) of the current cycle…
        [$phase] = $phaseOn($start->copy()->addDays(13));
        $this->assertSame('ovulation', $phase);

        // …and recurs in the predicted next cycle too (day 14 there).
        [$phase] = $phaseOn($start->copy()->addDays(28 + 13));
        $this->assertSame('ovulation', $phase);

        // The late luteal days before the next period read as PMS (the
        // calendar paints `is_pms_window` days purple).
        [, $isPms] = $phaseOn($start->copy()->addDays(25));
        $this->assertTrue($isPms);
    }

    public function test_future_start_date_is_rejected(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/cycle/period/start', ['date' => now()->addDay()->toDateString()])
            ->assertStatus(422);
    }
}
