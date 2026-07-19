<?php

namespace App\Services\HealthEngine;

use App\Enums\ConfidenceLevel;
use App\Enums\CycleSubphase;
use App\Enums\CycleWarning;
use App\Enums\DataQualityLevel;
use App\Enums\EffectiveSource;
use App\Enums\FertilityLevel;
use App\Enums\MainPhase;
use App\Enums\PeriodEndSource;
use App\Enums\PeriodStartSource;
use App\Enums\ResolutionSource;
use App\Models\UserProfile;
use Carbon\Carbon;
use Illuminate\Support\Collection;

/**
 * The v1.1 cycle engine (task.md): resolves the full cycle status of one
 * target date from the user's logged periods, onboarding profile and the
 * effective metrics. Implements the resolve priority of §20 — actual/assumed
 * menstrual, period_expected, fertile display zone, luteal, follicular,
 * unknown — with the §12 soft/warning/hard caps for a missing period end, the
 * §22 overdue model, and the deterministic §28/§29 data-quality and confidence
 * assignment.
 *
 * Everything here is an estimate for a calendar day — never a diagnosis, and
 * never written back into the user's actual period logs (§2, §39–40).
 */
class CycleStatusResolver
{
    /** MVP luteal phase length (§17). */
    private const LUTEAL_LENGTH = 14;

    /** Fertile biological window: O-5 .. O+1 (§18). */
    private const FERTILE_DAYS_BEFORE_OVULATION = 5;

    /** An unclosed period is never treated as active bleeding beyond this (§12). */
    private const PERIOD_HARD_CAP = 12;

    /** period_warning_cap = max(effective_period_length + 3, 8) (§12). */
    private const WARNING_CAP_EXTRA_DAYS = 3;

    private const WARNING_CAP_MIN_DAYS = 8;

    /** period_expected gives up after this many late days and goes unknown (§22.4). */
    private const MAX_LATE_DAYS = 14;

    /** Confidence-reason codes (§29.6). */
    public const REASON_LOW_HISTORY = 'low_history';

    public const REASON_HIGH_CYCLE_VARIABILITY = 'high_cycle_variability';

    public const REASON_MISSING_PERIOD_END = 'missing_period_end';

    public const REASON_MISSING_END_BEYOND_WARNING_CAP = 'missing_period_end_beyond_warning_cap';

    public const REASON_MISSING_END_BEYOND_HARD_CAP = 'missing_period_end_beyond_hard_cap';

    public const REASON_ONBOARDING_DATA_USED = 'onboarding_data_used';

    public const REASON_DEFAULT_VALUES_USED = 'default_values_used';

    public const REASON_PREDICTION_BASED_OUTPUT = 'prediction_based_output';

    public const REASON_PERIOD_OVERDUE = 'period_overdue';

    public const REASON_UNRESOLVED_CYCLE = 'unresolved_cycle';

    public const REASON_POOR_DATA_QUALITY = 'poor_data_quality';

    public const REASON_INSUFFICIENT_ANCHOR_DATA = 'insufficient_anchor_data';

    /**
     * Resolve the status of $targetDate. $today separates the two prediction
     * regimes (§10, §22): for dates up to today an awaited period that was never
     * logged goes through the period_expected / overdue model, while for future
     * dates the anchor is rolled forward over predicted cycles
     * (predicted_reference) so the calendar keeps rendering phases.
     */
    public function resolve(
        Collection $histories,
        ?UserProfile $profile,
        Carbon $targetDate,
        Carbon $today,
        CycleMetrics $metrics,
    ): CycleStatus {
        $target = $targetDate->copy()->startOfDay();
        $ref = $today->copy()->startOfDay();

        $anchor = $this->anchorFor($histories, $profile, $target);
        if ($anchor === null) {
            return $this->unresolved($target, $metrics);
        }

        [$start, $startSource, $loggedEnd] = $anchor;

        $ecl = max(1, $metrics->effectiveCycleLength);
        $epl = max(1, $metrics->effectivePeriodDuration);

        // §10 predicted_reference: only future dates ride predicted cycles; for
        // target dates up to today the anchor stays real so §22 overdue fires.
        if ($target->gt($ref)) {
            while ($start->copy()->addDays($ecl)->lte($target)) {
                $start = $start->copy()->addDays($ecl);
                $startSource = PeriodStartSource::PREDICTED_REFERENCE;
                $loggedEnd = null;
            }
        }

        // §11: a logged end is authoritative; otherwise the end is assumed from
        // the effective period length and must be flagged unconfirmed.
        $endConfirmed = $loggedEnd !== null && $loggedEnd->gte($start);
        $end = $endConfirmed ? $loggedEnd->copy() : $start->copy()->addDays($epl - 1);
        $endSource = $endConfirmed
            ? PeriodEndSource::USER_LOGGED
            : PeriodEndSource::ASSUMED_FROM_EFFECTIVE_DURATION;

        // §16–19 anchors.
        $predictedNext = $start->copy()->addDays($ecl);
        $ovulation = $predictedNext->copy()->subDays(self::LUTEAL_LENGTH);
        $biologicalFertileStart = $ovulation->copy()->subDays(self::FERTILE_DAYS_BEFORE_OVULATION);
        $dayAfterPeriodEnd = $end->copy()->addDay();
        $displayFertileStart = $biologicalFertileStart->gt($dayAfterPeriodEnd)
            ? $biologicalFertileStart
            : $dayAfterPeriodEnd;
        $displayFertileEnd = $ovulation->copy();
        $fertileWindowEmpty = $displayFertileEnd->lt($displayFertileStart);

        // §5, §22.1, §34 day counters.
        $cycleDay = $this->daysBetween($start, $target) + 1;
        $rawDaysToPeriod = $this->daysBetween($target, $predictedNext);
        $daysToPeriod = max(0, $rawDaysToPeriod);
        $daysLate = max(0, -$rawDaysToPeriod);
        $daysToOvulation = $this->daysBetween($target, $ovulation);

        // §12 caps — measured against the *effective* period length.
        $softCap = $epl;
        $warningCap = max($epl + self::WARNING_CAP_EXTRA_DAYS, self::WARNING_CAP_MIN_DAYS);
        $isRealOpenPeriod = $startSource === PeriodStartSource::USER_LOGGED && $loggedEnd === null;
        $pastWarningCapOpen = $isRealOpenPeriod && $cycleDay > $warningCap;

        $warnings = [];
        $requiresInput = false;
        $forcedPoor = false;
        $forcedLowConfidence = false;
        $anchorContradiction = false;

        $main = null;
        $sub = null;
        $resolution = null;

        // ── 1. actual / assumed-active menstrual (§20, §21) ──────────────────
        if ($endConfirmed && $target->betweenIncluded($start, $end)) {
            $main = MainPhase::MENSTRUAL;
            $sub = CycleSubphase::MENSTRUAL;
            $resolution = ResolutionSource::USER_LOGGED;
        } elseif (! $endConfirmed && $target->gte($start)) {
            if ($isRealOpenPeriod) {
                if ($cycleDay <= $softCap) {
                    $main = MainPhase::MENSTRUAL;
                    $sub = CycleSubphase::MENSTRUAL;
                    $resolution = ResolutionSource::USER_LOGGED_WITH_ASSUMED_END;
                } elseif ($cycleDay <= $warningCap) {
                    $main = MainPhase::MENSTRUAL;
                    $sub = CycleSubphase::MENSTRUAL_POSSIBLE;
                    $resolution = ResolutionSource::USER_LOGGED_WITH_ASSUMED_END;
                    $warnings[] = CycleWarning::PERIOD_END_MISSING;
                } elseif ($cycleDay <= self::PERIOD_HARD_CAP) {
                    $main = MainPhase::MENSTRUAL;
                    $sub = CycleSubphase::MENSTRUAL_POSSIBLE;
                    $resolution = ResolutionSource::USER_LOGGED_WITH_ASSUMED_END;
                    $warnings[] = CycleWarning::PERIOD_END_MISSING;
                    $warnings[] = CycleWarning::PERIOD_END_MISSING_WARNING_CAP_EXCEEDED;
                    $forcedPoor = true;
                    $forcedLowConfidence = true;
                    $requiresInput = true;
                } else {
                    // §21.5: past the hard cap the bleed is no longer assumed
                    // active; the assumed end stays an internal anchor and the
                    // rest of the cycle resolves normally below.
                    $warnings[] = CycleWarning::PERIOD_END_MISSING_HARD_CAP_EXCEEDED;
                    $forcedPoor = true;
                    $requiresInput = true;
                }
            } elseif ($cycleDay <= $softCap) {
                // Onboarding-declared or predicted-reference period: the assumed
                // bleed window renders as menstrual; the source is decided below.
                $main = MainPhase::MENSTRUAL;
                $sub = CycleSubphase::MENSTRUAL;
                if ($startSource === PeriodStartSource::USER_LOGGED) {
                    // Real start whose logged end predates it (inconsistent
                    // record): treat like an assumed-end resolve.
                    $resolution = ResolutionSource::USER_LOGGED_WITH_ASSUMED_END;
                }
            }
        }

        // ── 2. period_expected (§22) ─────────────────────────────────────────
        if ($main === null && $target->gte($predictedNext)) {
            if ($daysLate <= 7) {
                $main = MainPhase::PERIOD_EXPECTED;
                $sub = CycleSubphase::PERIOD_EXPECTED;
                $resolution = ResolutionSource::PREDICTION;
            } elseif ($daysLate <= self::MAX_LATE_DAYS) {
                $main = MainPhase::PERIOD_EXPECTED;
                $sub = CycleSubphase::PERIOD_EXPECTED;
                $resolution = ResolutionSource::PREDICTION;
                $warnings[] = CycleWarning::PREDICTED_PERIOD_OVERDUE;
                $forcedLowConfidence = true;
            } else {
                // §22.4: unresolved — but the engine output stays usable:
                // anchors and days_late are still returned.
                $main = MainPhase::UNKNOWN;
                $sub = CycleSubphase::UNKNOWN;
                $resolution = ResolutionSource::PREDICTION;
                $warnings[] = CycleWarning::PREDICTED_PERIOD_OVERDUE;
                $warnings[] = CycleWarning::CYCLE_UNRESOLVED_AFTER_EXPECTED_PERIOD;
                $forcedLowConfidence = true;
                $forcedPoor = true;
                $requiresInput = true;
            }
        }

        // ── 3. fertile display zone (§19, §24) ───────────────────────────────
        if ($main === null
            && ! $fertileWindowEmpty
            && $target->betweenIncluded($displayFertileStart, $displayFertileEnd)) {
            $main = MainPhase::FERTILE;
            $sub = match (true) {
                $daysToOvulation >= 3 => CycleSubphase::FERTILE_RISING,
                $daysToOvulation >= 1 => CycleSubphase::HIGH_FERTILITY,
                default => CycleSubphase::OVULATION_LIKELY,
            };
            $resolution = ResolutionSource::PREDICTION;
        }

        // ── 4. luteal (§25) ──────────────────────────────────────────────────
        if ($main === null && $target->gt($ovulation) && $target->lt($predictedNext)) {
            $daysSinceOvulation = $this->daysBetween($ovulation, $target);
            $daysUntilPeriod = $this->daysBetween($target, $predictedNext);
            $main = MainPhase::LUTEAL;
            $sub = match (true) {
                $daysSinceOvulation === 1 => CycleSubphase::POST_OVULATION,
                $daysUntilPeriod >= 1 && $daysUntilPeriod <= 3 => CycleSubphase::PMS_POSSIBLE,
                $daysUntilPeriod >= 4 && $daysUntilPeriod <= 6 => CycleSubphase::LATE_LUTEAL,
                default => $this->earlyOrMidLuteal($target, $ovulation, $predictedNext),
            };
            $resolution = ResolutionSource::PREDICTION;
        }

        // ── 5. follicular (§23) ──────────────────────────────────────────────
        if ($main === null) {
            $follicularStart = $end->copy()->addDay();
            $follicularEnd = $displayFertileStart->copy()->subDay();
            if (! $follicularEnd->lt($follicularStart)
                && $target->betweenIncluded($follicularStart, $follicularEnd)) {
                $main = MainPhase::FOLLICULAR;
                $sub = $this->follicularSubphase($target, $follicularStart, $follicularEnd);
                $resolution = ResolutionSource::PREDICTION;
            }
        }

        // ── 6. unknown fallback (§20): anchors incomplete or contradictory ───
        if ($main === null) {
            $main = MainPhase::UNKNOWN;
            $sub = CycleSubphase::UNKNOWN;
            $resolution = ResolutionSource::UNKNOWN;
            $warnings[] = CycleWarning::INSUFFICIENT_ANCHOR_DATA;
            $requiresInput = true;
            $anchorContradiction = true;
        }

        // §30: without any real log for the current cycle, the resolve is
        // onboarding- or default-based — except period_expected/late-unknown,
        // which the spec pins to prediction (§22).
        if ($startSource === PeriodStartSource::ONBOARDING_DECLARED
            && $resolution !== ResolutionSource::UNKNOWN
            && $main !== MainPhase::PERIOD_EXPECTED
            && ! ($main === MainPhase::UNKNOWN && $daysLate > self::MAX_LATE_DAYS)) {
            $resolution = $this->onboardingResolution($metrics);
        } elseif ($resolution === null) {
            $resolution = ResolutionSource::PREDICTION;
        }

        // ── §32 data-signal warnings ─────────────────────────────────────────
        $range = $metrics->cycleVariabilityRange;
        if ($metrics->validCyclesCount < 2) {
            $warnings[] = CycleWarning::LOW_HISTORY;
        }
        if ($range !== null && $range > 5) {
            $warnings[] = CycleWarning::HIGH_CYCLE_VARIABILITY;
        }
        if ($metrics->hasShortCycleOutlier) {
            $warnings[] = CycleWarning::SHORT_CYCLE_OUTLIER_DETECTED;
        }
        if ($metrics->hasLongCycleOutlier) {
            $warnings[] = CycleWarning::LONG_CYCLE_OUTLIER_DETECTED;
        }
        $onboardingUsed = $startSource === PeriodStartSource::ONBOARDING_DECLARED
            || $metrics->cycleLengthSource === EffectiveSource::PROFILE
            || $metrics->periodDurationSource === EffectiveSource::PROFILE;
        $defaultsUsed = $metrics->cycleLengthSource === EffectiveSource::DEFAULT
            || $metrics->periodDurationSource === EffectiveSource::DEFAULT;
        if ($onboardingUsed) {
            $warnings[] = CycleWarning::ONBOARDING_DATA_USED;
        }
        if ($defaultsUsed) {
            $warnings[] = CycleWarning::DEFAULT_VALUES_USED;
        }
        if ($resolution === ResolutionSource::PREDICTION
            || $startSource === PeriodStartSource::PREDICTED_REFERENCE) {
            $warnings[] = CycleWarning::PREDICTION_BASED_OUTPUT;
        }

        $warnings = array_values(array_unique(array_map(fn (CycleWarning $w) => $w->value, $warnings)));
        foreach ($warnings as $warning) {
            if (CycleWarning::from($warning)->requiresUserInput()) {
                $requiresInput = true;
            }
        }

        // ── data_quality (§28): insufficient → poor → good → partial ─────────
        $endInvolved = $main === MainPhase::MENSTRUAL
            || $main === MainPhase::FOLLICULAR
            || ($main === MainPhase::FERTILE && $displayFertileStart->eq($end->copy()->addDay()));
        $dataQuality = match (true) {
            $forcedPoor || $pastWarningCapOpen || $daysLate > self::MAX_LATE_DAYS
                || $anchorContradiction || $metrics->onlyOutlierHistory => DataQualityLevel::POOR,
            $startSource === PeriodStartSource::USER_LOGGED
                && $metrics->validCyclesCount >= 2
                && $main !== MainPhase::UNKNOWN
                && $daysLate <= 7
                && $metrics->cycleLengthSource === EffectiveSource::RECENT_VALID_CYCLES
                && $metrics->periodDurationSource === EffectiveSource::RECENT_VALID_CYCLES
                && ! ($endInvolved && ! $endConfirmed) => DataQualityLevel::GOOD,
            default => DataQualityLevel::PARTIAL,
        };

        // ── confidence (§29): unknown → low → high → medium ──────────────────
        $confidence = match (true) {
            $resolution === ResolutionSource::UNKNOWN => ConfidenceLevel::UNKNOWN,
            $forcedLowConfidence
                || $main === MainPhase::UNKNOWN
                || $daysLate > 7
                || $dataQuality === DataQualityLevel::POOR
                || $pastWarningCapOpen
                || $metrics->validCyclesCount < 2
                || ($range !== null && $range > 5)
                || in_array($resolution, [ResolutionSource::DEFAULT_BASED, ResolutionSource::ONBOARDING_BASED], true) => ConfidenceLevel::LOW,
            $dataQuality === DataQualityLevel::GOOD
                && $metrics->validCyclesCount >= 2
                && $metrics->validPeriodDurationsCount >= 2
                && $startSource === PeriodStartSource::USER_LOGGED
                && (! $endInvolved || $endConfirmed)
                && $range !== null && $range <= 5
                && ! in_array($main, [MainPhase::PERIOD_EXPECTED, MainPhase::UNKNOWN], true) => ConfidenceLevel::HIGH,
            default => ConfidenceLevel::MEDIUM,
        };

        $reasons = $this->confidenceReasons(
            $metrics,
            $range,
            $isRealOpenPeriod,
            $cycleDay,
            $warningCap,
            $onboardingUsed,
            $defaultsUsed,
            $resolution,
            $startSource,
            $daysLate,
            $main,
            $dataQuality,
            $anchorContradiction,
        );

        return new CycleStatus(
            date: $target,
            cycleDay: $cycleDay,
            mainPhase: $main,
            subphase: $sub,
            fertilityLevel: $main === MainPhase::UNKNOWN ? FertilityLevel::UNKNOWN : $sub->fertilityLevelV11(),
            daysToOvulation: $daysLate > self::MAX_LATE_DAYS ? null : $daysToOvulation,
            daysToPeriod: $daysToPeriod,
            daysLate: $daysLate,
            currentPeriodStart: $start,
            currentPeriodStartSource: $startSource,
            currentPeriodEnd: $end,
            currentPeriodEndSource: $endSource,
            currentPeriodEndIsConfirmed: $endConfirmed,
            predictedNextPeriodStart: $predictedNext,
            estimatedOvulationDate: $ovulation,
            effectiveCycleLength: $ecl,
            effectivePeriodLength: $epl,
            cycleVariability: $range,
            confidence: $confidence,
            confidenceReasons: $reasons,
            dataQuality: $dataQuality,
            resolutionSource: $resolution,
            requiresUserInput: $requiresInput,
            warnings: $warnings,
        );
    }

    /**
     * The current-cycle start anchor for the target date (§10.1): the latest
     * user-confirmed start on/before it, else the onboarding-declared start
     * (estimated seed row or profile LMP), else null.
     *
     * @return array{0: Carbon, 1: PeriodStartSource, 2: ?Carbon}|null [start, source, logged end]
     */
    private function anchorFor(Collection $histories, ?UserProfile $profile, Carbon $target): ?array
    {
        $latestReal = $histories
            ->filter(fn ($history) => (bool) $history->is_confirmed
                && Carbon::parse($history->period_start_date)->startOfDay()->lte($target))
            ->sortByDesc(fn ($history) => Carbon::parse($history->period_start_date)->timestamp)
            ->first();

        if ($latestReal) {
            return [
                Carbon::parse($latestReal->period_start_date)->startOfDay(),
                PeriodStartSource::USER_LOGGED,
                $latestReal->period_end_date !== null
                    ? Carbon::parse($latestReal->period_end_date)->startOfDay()
                    : null,
            ];
        }

        $onboardingRow = $histories
            ->filter(fn ($history) => (bool) ($history->is_estimated ?? false)
                && Carbon::parse($history->period_start_date)->startOfDay()->lte($target))
            ->sortByDesc(fn ($history) => Carbon::parse($history->period_start_date)->timestamp)
            ->first();

        if ($onboardingRow) {
            // A declared period is not an actual log (§6.2): its end is always
            // re-assumed from the effective duration, never treated as logged.
            return [
                Carbon::parse($onboardingRow->period_start_date)->startOfDay(),
                PeriodStartSource::ONBOARDING_DECLARED,
                null,
            ];
        }

        if ($profile?->last_period_start) {
            $lmp = Carbon::parse($profile->last_period_start)->startOfDay();
            if ($lmp->lte($target)) {
                return [$lmp, PeriodStartSource::ONBOARDING_DECLARED, null];
            }
        }

        return null;
    }

    /** §28.2 / §29.2 / §30: no anchor at all — engine output is still well-formed. */
    private function unresolved(Carbon $target, CycleMetrics $metrics): CycleStatus
    {
        return new CycleStatus(
            date: $target,
            cycleDay: null,
            mainPhase: MainPhase::UNKNOWN,
            subphase: CycleSubphase::UNKNOWN,
            fertilityLevel: FertilityLevel::UNKNOWN,
            daysToOvulation: null,
            daysToPeriod: null,
            daysLate: null,
            currentPeriodStart: null,
            currentPeriodStartSource: PeriodStartSource::UNKNOWN,
            currentPeriodEnd: null,
            currentPeriodEndSource: PeriodEndSource::UNKNOWN,
            currentPeriodEndIsConfirmed: false,
            predictedNextPeriodStart: null,
            estimatedOvulationDate: null,
            effectiveCycleLength: max(1, $metrics->effectiveCycleLength),
            effectivePeriodLength: max(1, $metrics->effectivePeriodDuration),
            cycleVariability: $metrics->cycleVariabilityRange,
            confidence: ConfidenceLevel::UNKNOWN,
            confidenceReasons: [self::REASON_INSUFFICIENT_ANCHOR_DATA],
            dataQuality: DataQualityLevel::INSUFFICIENT,
            resolutionSource: ResolutionSource::UNKNOWN,
            requiresUserInput: true,
            warnings: [CycleWarning::INSUFFICIENT_ANCHOR_DATA->value],
        );
    }

    /**
     * §23: split the follicular display range 40/40/20 (early/mid/late
     * transition), with the 1-day and 2-day special cases and the floor+minimum
     * rounding rules.
     */
    private function follicularSubphase(Carbon $target, Carbon $rangeStart, Carbon $rangeEnd): CycleSubphase
    {
        $length = $this->daysBetween($rangeStart, $rangeEnd) + 1;
        $position = $this->daysBetween($rangeStart, $target) + 1;

        if ($length === 1) {
            return CycleSubphase::MID_FOLLICULAR;
        }

        if ($length === 2) {
            return $position === 1
                ? CycleSubphase::EARLY_FOLLICULAR
                : CycleSubphase::LATE_FOLLICULAR_TRANSITION;
        }

        $early = max(1, (int) floor($length * 0.4));
        $mid = max(1, (int) floor($length * 0.4));

        return match (true) {
            $position <= $early => CycleSubphase::EARLY_FOLLICULAR,
            $position <= $early + $mid => CycleSubphase::MID_FOLLICULAR,
            default => CycleSubphase::LATE_FOLLICULAR_TRANSITION,
        };
    }

    /**
     * §25.4: split the assignable luteal range (O+2 .. P−7) into an early and a
     * mid half; on an odd length the extra day belongs to early_luteal.
     */
    private function earlyOrMidLuteal(Carbon $target, Carbon $ovulation, Carbon $predictedNext): CycleSubphase
    {
        $assignableStart = $ovulation->copy()->addDays(2);
        $assignableEnd = $predictedNext->copy()->subDays(7);

        if ($assignableEnd->lt($assignableStart)
            || ! $target->betweenIncluded($assignableStart, $assignableEnd)) {
            return CycleSubphase::MID_LUTEAL;
        }

        $length = $this->daysBetween($assignableStart, $assignableEnd) + 1;
        $earlyLength = (int) ceil($length / 2);

        return $this->daysBetween($assignableStart, $target) < $earlyLength
            ? CycleSubphase::EARLY_LUTEAL
            : CycleSubphase::MID_LUTEAL;
    }

    /** §30.1: onboarding anchor without declared lengths degrades to default_based. */
    private function onboardingResolution(CycleMetrics $metrics): ResolutionSource
    {
        $noDeclaredValues = $metrics->cycleLengthSource === EffectiveSource::DEFAULT
            && $metrics->periodDurationSource === EffectiveSource::DEFAULT;

        return $noDeclaredValues ? ResolutionSource::DEFAULT_BASED : ResolutionSource::ONBOARDING_BASED;
    }

    /**
     * §29.6: every active cause behind the confidence grade, as stable codes.
     *
     * @return array<string>
     */
    private function confidenceReasons(
        CycleMetrics $metrics,
        ?int $range,
        bool $isRealOpenPeriod,
        int $cycleDay,
        int $warningCap,
        bool $onboardingUsed,
        bool $defaultsUsed,
        ResolutionSource $resolution,
        PeriodStartSource $startSource,
        int $daysLate,
        MainPhase $main,
        DataQualityLevel $dataQuality,
        bool $anchorContradiction,
    ): array {
        $reasons = [];

        if ($metrics->validCyclesCount < 2) {
            $reasons[] = self::REASON_LOW_HISTORY;
        }
        if ($range !== null && $range > 5) {
            $reasons[] = self::REASON_HIGH_CYCLE_VARIABILITY;
        }
        if ($isRealOpenPeriod) {
            $reasons[] = self::REASON_MISSING_PERIOD_END;
            if ($cycleDay > $warningCap && $cycleDay <= self::PERIOD_HARD_CAP) {
                $reasons[] = self::REASON_MISSING_END_BEYOND_WARNING_CAP;
            } elseif ($cycleDay > self::PERIOD_HARD_CAP) {
                $reasons[] = self::REASON_MISSING_END_BEYOND_HARD_CAP;
            }
        }
        if ($onboardingUsed) {
            $reasons[] = self::REASON_ONBOARDING_DATA_USED;
        }
        if ($defaultsUsed) {
            $reasons[] = self::REASON_DEFAULT_VALUES_USED;
        }
        if ($resolution === ResolutionSource::PREDICTION
            || $startSource === PeriodStartSource::PREDICTED_REFERENCE) {
            $reasons[] = self::REASON_PREDICTION_BASED_OUTPUT;
        }
        if ($daysLate > 0) {
            $reasons[] = self::REASON_PERIOD_OVERDUE;
        }
        if ($main === MainPhase::UNKNOWN && $daysLate > self::MAX_LATE_DAYS) {
            $reasons[] = self::REASON_UNRESOLVED_CYCLE;
        }
        if ($dataQuality === DataQualityLevel::POOR) {
            $reasons[] = self::REASON_POOR_DATA_QUALITY;
        }
        if ($anchorContradiction) {
            $reasons[] = self::REASON_INSUFFICIENT_ANCHOR_DATA;
        }

        return array_values(array_unique($reasons));
    }

    /** Signed whole-day difference: $to − $from (task.md §4). */
    private function daysBetween(Carbon $from, Carbon $to): int
    {
        return (int) round($from->diffInDays($to));
    }
}
