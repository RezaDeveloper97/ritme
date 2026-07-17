<?php

namespace App\Services\HealthEngine;

use App\Enums\ConfidenceLevel;
use App\Enums\CycleSubphase;
use App\Enums\RegularityStatus;
use App\Models\CycleHistory;
use App\Models\User;
use App\Models\UserProfile;
use Carbon\Carbon;

/**
 * Assembles the spec §19 daily payload for one date from all the engine pieces: the
 * three-layer metrics (§12), the Start-anchored prediction (§13), the open-period
 * state (§4), the confidence model (§18) and the render-ready daily card (§"card
 * content"). This is the single place the API stitches them together, keeping the
 * per-day {@see HealthDataEngine} calc (which also feeds persistence) untouched.
 */
class CycleDayViewBuilder
{
    private CycleMetricsCalculator $metrics;

    private CyclePredictionService $predictions;

    private OpenPeriodEvaluator $openPeriods;

    private CycleConfidenceCalculator $confidence;

    private DailyCardBuilder $cards;

    public function __construct()
    {
        $this->metrics = new CycleMetricsCalculator;
        $this->predictions = new CyclePredictionService;
        $this->openPeriods = new OpenPeriodEvaluator;
        $this->confidence = new CycleConfidenceCalculator;
        $this->cards = new DailyCardBuilder;
    }

    /**
     * @param  array<string, mixed>  $baseCalc  the per-day {@see HealthDataEngine::calculateForDate} result
     */
    public function build(User $user, Carbon $selectedDate, Carbon $today, string $locale, array $baseCalc): array
    {
        $profile = $user->profile;
        $histories = CycleHistory::where('user_id', $user->id)
            ->orderBy('period_start_date', 'desc')
            ->get();

        $metrics = $this->metrics->calculate($histories, $profile);

        // No anchor yet (incomplete profile) → serve just the three-layer values.
        if ($baseCalc['cycle_day'] === null) {
            return $this->emptyView($selectedDate, $metrics);
        }

        $selected = $selectedDate->copy()->startOfDay();
        $ref = $today->copy()->startOfDay();

        $lastConfirmedStart = $this->lastConfirmedStart($histories, $profile, $ref);

        // The period the user is "waiting for": the last confirmed start plus the
        // effective cycle length (not rolled forward), so overdue states fire (§16).
        $prediction = $this->predictions->predict(
            $lastConfirmedStart,
            $metrics->effectiveCycleLength,
            $metrics->effectivePeriodDuration,
            $lastConfirmedStart,
            $metrics->cycleLengthSource,
        );

        $openState = $this->openPeriods->evaluate(
            $this->openPeriodStart($histories, $ref),
            $metrics->effectivePeriodDuration,
            $profile?->period_duration,
            $ref,
        );

        [$loggedDay, $loggedClosed] = $this->loggedPeriodFor($histories, $selected, $ref);

        $subphase = CycleSubphase::from($baseCalc['subphase']);

        $card = $this->cards->build(
            $selected,
            $ref,
            (int) $baseCalc['cycle_day'],
            $subphase,
            $prediction->nextPeriodStart,
            $openState,
            $loggedDay,
            $loggedClosed,
            $locale,
        );

        $confidence = $this->confidence->forPrediction($metrics, $openState->pastHardCap);
        $layers = $metrics->toApiArray();

        return [
            'date' => $selected->toDateString(),
            'cycle_day' => (int) $baseCalc['cycle_day'],
            'phase' => $baseCalc['phase'],
            'subphase' => $baseCalc['subphase'],
            // Take the fertility level from the card so it always matches what the
            // user sees (the card forces "low" on real/ongoing period days, where the
            // raw sub-phase mapping would otherwise still read the predicted position).
            'fertility_level' => $card->fertilityLevel->value,
            'data_status' => $card->dataStatus->value,
            'predictions' => array_merge($prediction->toApiArray(), $confidence->toApiArray()),
            'profile_values' => $layers['profile_values'],
            'calculated_values' => $layers['calculated_values'],
            'effective_values' => $layers['effective_values'],
            'data_quality' => array_merge($confidence->toApiArray(), [
                'regularity_status' => $metrics->regularityStatus->value,
                'is_irregular_possible' => $metrics->regularityStatus === RegularityStatus::IRREGULAR_POSSIBLE,
                'missing_period_end' => $openState->endOverdue,
            ]),
            'daily_card' => $card->toApiArray(),
        ];
    }

    /**
     * The most recent user-confirmed period start on/before the reference date, or the
     * profile LMP as the onboarding anchor.
     */
    private function lastConfirmedStart(mixed $histories, ?UserProfile $profile, Carbon $ref): Carbon
    {
        $latest = $histories
            ->filter(fn ($history) => $history->is_confirmed && Carbon::parse($history->period_start_date)->lte($ref))
            ->sortByDesc('period_start_date')
            ->first();

        if ($latest) {
            return Carbon::parse($latest->period_start_date)->startOfDay();
        }

        return $profile?->last_period_start
            ? Carbon::parse($profile->last_period_start)->startOfDay()
            : $ref->copy();
    }

    /**
     * The start of the current open period — only the latest record, and only if it is
     * still unclosed (a stale older open record does not count).
     */
    private function openPeriodStart(mixed $histories, Carbon $ref): ?Carbon
    {
        $latest = $histories->sortByDesc('period_start_date')->first();

        if ($latest
            && $latest->is_confirmed
            && $latest->period_end_date === null
            && Carbon::parse($latest->period_start_date)->lte($ref)) {
            return Carbon::parse($latest->period_start_date)->startOfDay();
        }

        return null;
    }

    /**
     * If $selected falls inside a logged period, its day number and whether that period
     * is closed (→ actual) or still open (→ incomplete). An open period covers up to
     * the reference date.
     *
     * @return array{0: int|null, 1: bool}
     */
    private function loggedPeriodFor(mixed $histories, Carbon $selected, Carbon $ref): array
    {
        // Oldest→newest so an open (null-end) period can be capped at the next logged
        // period's start — a forgotten-open old period must not "cover" days that fall
        // after a newer period was logged.
        $confirmed = $histories
            ->filter(fn ($history) => (bool) $history->is_confirmed)
            ->sortBy(fn ($history) => Carbon::parse($history->period_start_date)->timestamp)
            ->values();

        foreach ($confirmed as $index => $history) {
            $start = Carbon::parse($history->period_start_date)->startOfDay();
            if ($selected->lt($start)) {
                continue;
            }

            if ($history->period_end_date !== null) {
                $end = Carbon::parse($history->period_end_date)->startOfDay();
                if ($selected->lte($end)) {
                    return [(int) $start->diffInDays($selected) + 1, true];
                }

                continue;
            }

            // Open period: covers up to the reference date, but never into the next one.
            $cap = $ref->copy();
            $next = $confirmed->get($index + 1);
            if ($next !== null) {
                $nextStart = Carbon::parse($next->period_start_date)->startOfDay()->subDay();
                if ($nextStart->lt($cap)) {
                    $cap = $nextStart;
                }
            }
            if ($selected->lte($cap)) {
                return [(int) $start->diffInDays($selected) + 1, false];
            }
        }

        return [null, false];
    }

    private function emptyView(Carbon $selectedDate, CycleMetrics $metrics): array
    {
        $layers = $metrics->toApiArray();

        return [
            'date' => $selectedDate->toDateString(),
            'cycle_day' => null,
            'phase' => null,
            'subphase' => null,
            'fertility_level' => null,
            'data_status' => null,
            'predictions' => null,
            'profile_values' => $layers['profile_values'],
            'calculated_values' => $layers['calculated_values'],
            'effective_values' => $layers['effective_values'],
            'data_quality' => [
                'confidence' => ConfidenceLevel::LOW->value,
                'confidence_reasons' => [CycleConfidenceCalculator::REASON_PROFILE_ONLY],
                'regularity_status' => $metrics->regularityStatus->value,
                'is_irregular_possible' => false,
                'missing_period_end' => false,
            ],
            'daily_card' => null,
        ];
    }
}
