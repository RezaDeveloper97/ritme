<?php

namespace Tests\Unit;

use App\Enums\CycleSubphase;
use App\Enums\DataStatus;
use App\Enums\FertilityLevel;
use App\Services\HealthEngine\DailyCard;
use App\Services\HealthEngine\DailyCardBuilder;
use App\Services\HealthEngine\OpenPeriodState;
use Carbon\Carbon;
use PHPUnit\Framework\TestCase;

/**
 * The rule-based daily card (spec §"final title logic"). Each scenario exercises one
 * branch of the priority chain: actual > incomplete > needs_confirmation > predicted.
 */
class DailyCardBuilderTest extends TestCase
{
    private DailyCardBuilder $builder;

    protected function setUp(): void
    {
        parent::setUp();
        $this->builder = new DailyCardBuilder;
    }

    /**
     * @param  array<string, mixed>  $o
     */
    private function card(array $o): DailyCard
    {
        return $this->builder->build(
            Carbon::parse($o['selected'] ?? '2026-07-15'),
            Carbon::parse($o['today'] ?? '2026-07-15'),
            $o['cycleDay'] ?? 8,
            $o['subphase'] ?? CycleSubphase::MID_FOLLICULAR,
            Carbon::parse($o['nextStart'] ?? '2026-08-04'),
            $o['open'] ?? OpenPeriodState::none(),
            $o['loggedDay'] ?? null,
            $o['loggedClosed'] ?? false,
            $o['locale'] ?? 'en',
        );
    }

    private function openState(bool $endOverdue): OpenPeriodState
    {
        return new OpenPeriodState(
            hasOpenPeriod: true,
            startDate: Carbon::parse('2026-07-13'),
            menstrualDay: $endOverdue ? 10 : 3,
            displayAsActiveMenstrual: ! $endOverdue,
            endOverdue: $endOverdue,
            pastHardCap: false,
            assumedEndDate: Carbon::parse('2026-07-17'),
            dataQualityFlags: [],
        );
    }

    public function test_inside_a_closed_logged_period_is_actual(): void
    {
        $card = $this->card(['loggedDay' => 2, 'loggedClosed' => true]);

        $this->assertSame(DataStatus::ACTUAL, $card->dataStatus);
        $this->assertSame('Day 2 of your period', $card->title);
        $this->assertSame(FertilityLevel::LOW, $card->fertilityLevel);
    }

    public function test_open_period_within_expected_length_is_incomplete(): void
    {
        $card = $this->card(['loggedDay' => 3, 'loggedClosed' => false, 'open' => $this->openState(false)]);

        $this->assertSame(DataStatus::INCOMPLETE, $card->dataStatus);
        $this->assertSame('Day 3 of your period', $card->title);
        $this->assertSame(self::actionTypes($card), ['log_symptoms', 'log_period_end', 'view_details']);
    }

    public function test_open_period_past_expected_length_asks_for_the_end_today(): void
    {
        $card = $this->card([
            'loggedDay' => 10,
            'loggedClosed' => false,
            'open' => $this->openState(true),
            'today' => '2026-07-15',
            'selected' => '2026-07-15',
        ]);

        $this->assertSame(DataStatus::INCOMPLETE, $card->dataStatus);
        $this->assertSame('Period end not logged yet', $card->title);
        $this->assertSame('log_period_end', $card->primaryAction['type']);
    }

    public function test_predicted_period_due_needs_confirmation(): void
    {
        $card = $this->card(['selected' => '2026-07-15', 'today' => '2026-07-15', 'nextStart' => '2026-07-15']);

        $this->assertSame(DataStatus::NEEDS_CONFIRMATION, $card->dataStatus);
        $this->assertSame('Your predicted period is due', $card->title);
        $this->assertSame('confirm_period_start', $card->primaryAction['type']);
    }

    public function test_one_day_overdue_reports_the_gap(): void
    {
        $card = $this->card(['selected' => '2026-07-15', 'today' => '2026-07-15', 'nextStart' => '2026-07-14']);

        $this->assertSame(DataStatus::NEEDS_CONFIRMATION, $card->dataStatus);
        $this->assertSame('1 day(s) past the predicted date', $card->title);
    }

    public function test_several_days_overdue_switches_message(): void
    {
        $card = $this->card(['selected' => '2026-07-15', 'today' => '2026-07-15', 'nextStart' => '2026-07-11']);

        $this->assertSame('Period start not logged yet', $card->title);
        $this->assertSame('log_period_start', $card->primaryAction['type']);
    }

    public function test_countdown_to_the_next_period(): void
    {
        $card = $this->card([
            'selected' => '2026-07-20', 'today' => '2026-07-20',
            'nextStart' => '2026-07-25', 'ovulation' => '2026-07-11',
            'subphase' => CycleSubphase::LATE_LUTEAL,
        ]);

        $this->assertSame(DataStatus::PREDICTED, $card->dataStatus);
        $this->assertSame('About 5 days to your next period', $card->title);
    }

    public function test_fertile_window_uses_the_subphase_fertility(): void
    {
        $card = $this->card([
            'selected' => '2026-07-12', 'today' => '2026-07-12',
            'ovulation' => '2026-07-14', 'nextStart' => '2026-07-28',
            'subphase' => CycleSubphase::HIGH_FERTILITY,
        ]);

        $this->assertSame("You're likely in your fertile window", $card->title);
        $this->assertSame(FertilityLevel::HIGH, $card->fertilityLevel);
    }

    public function test_ovulation_day(): void
    {
        $card = $this->card([
            'selected' => '2026-07-14', 'today' => '2026-07-14',
            'ovulation' => '2026-07-14', 'nextStart' => '2026-07-28',
            'subphase' => CycleSubphase::OVULATION_LIKELY,
        ]);

        $this->assertSame('Ovulation is likely near', $card->title);
        $this->assertSame(FertilityLevel::VERY_HIGH, $card->fertilityLevel);
    }

    public function test_plain_cycle_day_is_the_default(): void
    {
        $card = $this->card([
            'selected' => '2026-07-08', 'today' => '2026-07-08', 'cycleDay' => 8,
            'ovulation' => '2026-07-14', 'nextStart' => '2026-07-28',
        ]);

        $this->assertSame('Day 8 of your cycle', $card->title);
        $this->assertSame(DataStatus::PREDICTED, $card->dataStatus);
    }

    public function test_predicted_period_day_offers_confirming_the_start(): void
    {
        // A rolled-forward predicted bleeding day in the past: the prediction the
        // user is looking at is the one thing they can correct, so the card must
        // offer "it started here" instead of a generic complete-this-day.
        $card = $this->card([
            'selected' => '2026-06-01', 'today' => '2026-07-15', 'cycleDay' => 1,
            'nextStart' => '2026-07-28', 'subphase' => CycleSubphase::MENSTRUAL,
        ]);

        $this->assertSame(DataStatus::NEEDS_CONFIRMATION, $card->dataStatus);
        $this->assertSame('Day 1 of your predicted period', $card->title);
        $this->assertSame(self::actionTypes($card), ['confirm_period_start', 'period_not_started', 'view_details']);
    }

    public function test_predicted_period_day_in_the_future_stays_a_prediction(): void
    {
        // Future days can't have started yet — no confirm CTA there (§"future days").
        $card = $this->card([
            'selected' => '2026-08-25', 'today' => '2026-07-15', 'cycleDay' => 2,
            'nextStart' => '2026-07-28', 'subphase' => CycleSubphase::MENSTRUAL,
        ]);

        $this->assertSame(DataStatus::PREDICTED, $card->dataStatus);
        $this->assertSame('set_reminder', $card->primaryAction['type']);
    }

    public function test_future_day_never_says_not_started_and_offers_a_reminder(): void
    {
        $card = $this->card([
            'selected' => '2026-08-15', 'today' => '2026-07-15', 'cycleDay' => 12,
            'nextStart' => '2026-07-28', 'ovulation' => '2026-07-14',
        ]);

        $this->assertSame(DataStatus::PREDICTED, $card->dataStatus);
        $this->assertSame('Day 12 of the predicted cycle', $card->title);
        $this->assertSame('set_reminder', $card->primaryAction['type']);
    }

    public function test_persian_locale_localises_title_and_digits(): void
    {
        $card = $this->card(['loggedDay' => 2, 'loggedClosed' => true, 'locale' => 'fa']);

        $this->assertSame('روز ۲ پریود', $card->title);
        $this->assertSame('ثبت علائم امروز', $card->primaryAction['label']);
    }

    /**
     * @return array<string>
     */
    private static function actionTypes(DailyCard $card): array
    {
        $types = [];
        if ($card->primaryAction) {
            $types[] = $card->primaryAction['type'];
        }
        foreach ($card->secondaryActions as $action) {
            $types[] = $action['type'];
        }

        return $types;
    }
}
