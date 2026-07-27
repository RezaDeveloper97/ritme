<?php

namespace App\Services\HealthEngine;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\CycleVariability;
use App\Models\CycleHistory;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserProfile;
use Carbon\Carbon;
use Illuminate\Support\Collection;

class HealthDataEngine
{
    /**
     * Base probability by day relative to ovulation
     */
    private const BASE_PROBABILITY_MAP = [
        -5 => 0.10,
        -4 => 0.15,
        -3 => 0.18,
        -2 => 0.25,
        -1 => 0.30,
        0 => 0.33,
        1 => 0.12,
    ];

    /**
     * Age factor ranges
     */
    private const AGE_FACTORS = [
        [20, 29, 1.0],
        [30, 34, 0.85],
        [35, 37, 0.70],
        [38, 40, 0.55],
        [41, 100, 0.35],
    ];

    /**
     * Positive symptom multipliers
     */
    private const POSITIVE_SYMPTOMS = [
        'ewcm' => 1.30,          // Egg White Cervical Mucus
        'ovulation_cramps' => 1.20,
        'high_libido' => 1.15,
        'watery_discharge' => 1.10,
        'abdominal_heaviness' => 1.05,
    ];

    /**
     * Negative symptom multipliers
     */
    private const NEGATIVE_SYMPTOMS = [
        'pms' => 0.8,
        'dry_mucus' => 0.75,
        'luteal_spotting' => 0.7,
    ];

    /** Fixed luteal-phase length (days) — the MVP ovulation model (§2). */
    private const LUTEAL_LENGTH = 14;

    /**
     * Ovulation never falls before this cycle day. Without a floor a short cycle
     * (e.g. a 15-day one) would put ovulation on day 1, collapsing the follicular
     * phase; cycles in the valid 21–45 range are unaffected (their O is already ≥7).
     */
    private const MIN_OVULATION_DAY = 7;

    private User $user;

    private ?UserProfile $profile;

    private string $locale;

    private CycleMetricsCalculator $metricsCalculator;

    private CyclePhaseMapper $phaseMapper;

    /**
     * Effective bleeding length (days) for the cycle being calculated — set from the
     * three-layer metrics at the top of calculateForDate so phase/subphase detection
     * uses the learned/median period duration rather than only the raw profile value.
     */
    private int $effectivePeriodDuration = 5;

    public function __construct(User $user, string $locale = 'en')
    {
        $this->user = $user;
        $this->profile = $user->profile;
        $this->locale = $locale;
        $this->metricsCalculator = new CycleMetricsCalculator;
        $this->phaseMapper = new CyclePhaseMapper;
    }

    /**
     * Calculate data for a specific date
     */
    public function calculateForDate(Carbon $date): array
    {
        if (! $this->profile || ! $this->profile->last_period_start) {
            return $this->getEmptyCalculation($date);
        }

        $dailyLog = $this->getDailyLog($date);
        $cycleHistories = $this->getCycleHistories();

        // Core calculations. Effective cycle length, period duration and variability
        // all come from the same three-layer metrics (§6–8, §12): the median of the
        // last three valid cycles when available, else the profile baseline, else a
        // default. This is what replaced the old "average of 1–2 cycles" behaviour.
        $metrics = $this->metricsCalculator->calculate($cycleHistories, $this->profile);
        $this->effectivePeriodDuration = $metrics->effectivePeriodDuration;

        $cycleLength = $metrics->effectiveCycleLength;
        $cycleDay = $this->calculateCycleDay($date, $cycleLength, $cycleHistories);
        $variability = $metrics->variability;
        $ovulationDay = $this->calculateOvulationDay($cycleLength);

        // Phase detection
        // Prefer the bleeding length the user actually logged for this cycle's
        // period (start→end) over the profile default, so an edited end date
        // is reflected on the calendar immediately.
        $loggedBleeding = $this->loggedBleedingLength($date, $cycleLength, $cycleHistories);
        $phase = $this->determinePhase($cycleDay, $ovulationDay, $cycleLength, $loggedBleeding);
        $subphase = $this->determineSubphase($cycleDay, $ovulationDay, $cycleLength, $loggedBleeding);

        // Windows detection.
        // A bleeding day is only that — it can never also be a PMS day (the
        // pre-menstrual tail of the cycle) or a fertile day. On a short cycle the
        // fertile window (around ovulation = length−14) can otherwise reach back
        // into the bleeding days, and the PMS window (length−6) can abut them.
        // Menstruation always wins, so a day is period OR one of these windows,
        // never both — on the calendar and in every downstream read of the flags.
        $isPeriodDay = $phase === CyclePhase::MENSTRUATION;
        $isFertileWindow = $this->isInFertileWindow($cycleDay, $ovulationDay) && ! $isPeriodDay;
        $isPmsWindow = $this->isInPmsWindow($cycleDay, $cycleLength) && ! $isPeriodDay;
        $isPeriodTomorrow = $this->isPeriodTomorrow($cycleDay, $cycleLength, $variability);
        $isLutealSpotting = $this->detectLutealSpotting($cycleDay, $dailyLog);

        // Score calculations
        $cycleScore = $this->calculateCycleScore($variability, $dailyLog);
        $ageFactor = $this->calculateAgeFactor();
        $dayFromOvulation = $cycleDay - $ovulationDay;
        $baseProbability = $this->getBaseProbability($dayFromOvulation);
        $symptomScore = $this->calculateSymptomScore($dailyLog, $isPmsWindow, $isLutealSpotting);
        $finalProbability = $this->calculateFinalProbability(
            $baseProbability,
            $ageFactor,
            $cycleScore,
            $symptomScore
        );

        // Generate text flags and daily tips
        $textFlags = $this->generateTextFlags(
            $phase,
            $subphase,
            $isFertileWindow,
            $isPmsWindow,
            $isPeriodTomorrow,
            $finalProbability,
            $variability
        );
        $dailyTips = $this->generateDailyTips($phase, $subphase, $dailyLog);

        return [
            'calculation_date' => $date->toDateString(),
            'cycle_day' => $cycleDay,
            'phase' => $phase->value,
            'subphase' => $subphase->value,
            'estimated_ovulation_day' => $ovulationDay,
            'cycle_length_used' => $cycleLength,
            'is_fertile_window' => $isFertileWindow,
            'is_pms_window' => $isPmsWindow,
            'is_period_tomorrow' => $isPeriodTomorrow,
            'is_luteal_spotting' => $isLutealSpotting,
            'cycle_score' => round($cycleScore, 4),
            'age_factor' => round($ageFactor, 4),
            'base_probability' => round($baseProbability, 4),
            'symptom_score' => round($symptomScore, 4),
            'final_probability' => round($finalProbability * 100, 2),
            'cycle_variability' => $variability->value,
            'uncertainty_range' => $variability->uncertaintyRange(),
            'text_flags' => $textFlags,
            'daily_tips' => $dailyTips,
            'source_profile_data' => $this->getProfileSnapshot(),
            'source_daily_log_data' => $dailyLog?->toArray(),
        ];
    }

    /**
     * Calculate cycle day with proper wrapping based on cycle length
     *
     * The cycle day is always between 1 and cycle_length.
     * When cycle_length days pass, a new cycle starts at day 1.
     *
     * Priority for determining cycle start:
     * 1. Most recent confirmed period in cycle history
     * 2. Profile's last_period_start date
     *
     * For dates before last_period_start: extrapolate backwards based on cycle length
     */
    public function calculateCycleDay(Carbon $date, int $cycleLength, Collection $cycleHistories): int
    {
        // Find the most relevant cycle start date
        $cycleStartDate = $this->findRelevantCycleStart($date, $cycleLength, $cycleHistories);

        // Calculate days since cycle start (signed: negative if date is before cycle start)
        $daysSinceCycleStart = $cycleStartDate->diffInDays($date, false);

        // If the date is before the cycle start date, extrapolate backwards
        if ($daysSinceCycleStart < 0) {
            // Calculate how many days before the cycle start
            $daysBefore = abs($daysSinceCycleStart);
            // Calculate which day of the previous cycle(s) we're in
            $cycleDay = $cycleLength - ($daysBefore % $cycleLength);

            // If we land exactly on cycle length, it means day 1 of the current cycle start
            return $cycleDay === $cycleLength ? $cycleLength : $cycleDay;
        }

        // Calculate cycle day with wrapping (1 to cycleLength)
        // Day 1 is the first day, when days = 0, cycle day = 1
        // When days = cycleLength, new cycle starts at day 1
        $cycleDay = ($daysSinceCycleStart % $cycleLength) + 1;

        return $cycleDay;
    }

    /**
     * Find the most relevant cycle start date for a given date
     *
     * This finds the actual period start that applies to the given date,
     * or calculates the predicted cycle start based on LMP and cycle length.
     */
    private function findRelevantCycleStart(Carbon $date, int $cycleLength, Collection $cycleHistories): Carbon
    {
        $lmp = Carbon::parse($this->profile->last_period_start);

        // If we have cycle histories, find the most recent period start before or on the date
        if ($cycleHistories->isNotEmpty()) {
            $relevantPeriod = $cycleHistories
                ->filter(function ($history) use ($date) {
                    $periodStart = Carbon::parse($history->period_start_date);

                    return $periodStart->lte($date);
                })
                ->sortByDesc('period_start_date')
                ->first();

            if ($relevantPeriod) {
                $periodStart = Carbon::parse($relevantPeriod->period_start_date);

                // Check if this period is still the "current" cycle
                // Using signed difference to handle edge cases
                $daysSincePeriod = $periodStart->diffInDays($date, false);

                if ($daysSincePeriod >= 0 && $daysSincePeriod < $cycleLength) {
                    // We're still in the cycle that started with this period
                    return $periodStart;
                }

                if ($daysSincePeriod >= $cycleLength) {
                    // More than one cycle has passed - calculate the predicted cycle start
                    $completeCycles = (int) floor($daysSincePeriod / $cycleLength);

                    return $periodStart->copy()->addDays($completeCycles * $cycleLength);
                }
            }
        }

        // No relevant history or date is before all history, use LMP
        // Using signed difference
        $daysSinceLmp = $lmp->diffInDays($date, false);

        if ($daysSinceLmp < 0) {
            // Date is before LMP - return LMP as the reference point
            // The cycle day calculation will handle backwards extrapolation
            return $lmp;
        }

        if ($daysSinceLmp < $cycleLength) {
            // Still in first cycle
            return $lmp;
        }

        // Calculate which cycle we're in and when it started
        $completeCycles = (int) floor($daysSinceLmp / $cycleLength);

        return $lmp->copy()->addDays($completeCycles * $cycleLength);
    }

    /**
     * Effective cycle length via the shared three-layer metrics (§7): the median of
     * the last three valid cycles when available, else the profile baseline, else 28.
     * Kept as a public entry point for callers that only need the length. Never
     * returns 0 — every downstream calculation (cycle-day wrapping, ovulation, PMS
     * window) divides by it, so a 0 would raise DivisionByZeroError.
     */
    public function getEffectiveCycleLength(Collection $histories): int
    {
        return $this->metricsCalculator->calculate($histories, $this->profile)->effectiveCycleLength;
    }

    /**
     * Calculate variability based on standard deviation
     */
    public function calculateVariability(array $cycleLengths): CycleVariability
    {
        if (count($cycleLengths) < 2) {
            return CycleVariability::REGULAR;
        }

        $mean = array_sum($cycleLengths) / count($cycleLengths);
        $squaredDiffs = array_map(fn ($x) => pow($x - $mean, 2), $cycleLengths);
        $stdDev = sqrt(array_sum($squaredDiffs) / count($cycleLengths));

        return CycleVariability::fromStdDev($stdDev);
    }

    /**
     * Calculate ovulation day (cycle_length - 14)
     */
    public function calculateOvulationDay(int $cycleLength): int
    {
        // +1 so the 1-based ovulation cycle day matches the date-based ovulation used by
        // CyclePredictionService (next_period_start − 14 = cycle day cycleLength−13),
        // keeping the phase engine, predictions and the calendar fertile window in step (§2).
        return max($cycleLength - self::LUTEAL_LENGTH + 1, self::MIN_OVULATION_DAY);
    }

    /**
     * Effective bleeding (menstruation) length in days, guarded so it can never
     * span the whole cycle. A period longer than the cycle is physiologically
     * impossible and — before this guard — painted every day of the month as
     * "menstruation" (a solid period block on the calendar). When the stored
     * value is implausible (>= cycle length, e.g. from a bad client/legacy
     * onboarding), fall back to the 5-day default instead of trusting it.
     */
    private function effectiveBleedingLength(int $cycleLength, ?int $logged = null): int
    {
        $bleeding = $logged ?? $this->effectivePeriodDuration;

        if ($bleeding >= $cycleLength) {
            return max(1, min(5, $cycleLength - 1));
        }

        return max(1, $bleeding);
    }

    /**
     * The bleeding length the user actually logged for the cycle containing
     * `$date` (the record anchoring that cycle), or null when the period is
     * still open / unlogged — then the profile default applies.
     */
    private function loggedBleedingLength(Carbon $date, int $cycleLength, Collection $cycleHistories): ?int
    {
        $cycleStart = $this->findRelevantCycleStart($date, $cycleLength, $cycleHistories);

        $record = $cycleHistories->first(
            fn ($history) => Carbon::parse($history->period_start_date)->isSameDay($cycleStart)
        );

        return $record?->bleeding_length;
    }

    /**
     * Determine current phase
     */
    public function determinePhase(int $cycleDay, int $ovulationDay, int $cycleLength, ?int $loggedBleeding = null): CyclePhase
    {
        $bleedingLength = $this->effectiveBleedingLength($cycleLength, $loggedBleeding);

        return $this->phaseMapper->phaseFor($cycleDay, $ovulationDay, $bleedingLength);
    }

    /**
     * Determine detailed subphase
     */
    public function determineSubphase(int $cycleDay, int $ovulationDay, int $cycleLength, ?int $loggedBleeding = null): CycleSubphase
    {
        $bleedingLength = $this->effectiveBleedingLength($cycleLength, $loggedBleeding);

        return $this->phaseMapper->subphaseFor($cycleDay, $ovulationDay, $cycleLength, $bleedingLength);
    }

    /**
     * Check if in fertile window (ovulation - 5 to ovulation + 1)
     */
    public function isInFertileWindow(int $cycleDay, int $ovulationDay): bool
    {
        return $cycleDay >= ($ovulationDay - 5) && $cycleDay <= ($ovulationDay + 1);
    }

    /**
     * Check if in PMS window (cycle_length - 6 to end)
     */
    public function isInPmsWindow(int $cycleDay, int $cycleLength): bool
    {
        return $cycleDay >= ($cycleLength - 6);
    }

    /**
     * Check if period is tomorrow
     */
    public function isPeriodTomorrow(int $cycleDay, int $cycleLength, CycleVariability $variability): bool
    {
        $uncertainty = $variability->uncertaintyRange();
        $expectedPeriodDay = $cycleLength;

        return $cycleDay >= ($expectedPeriodDay - $uncertainty - 1)
            && $cycleDay <= ($expectedPeriodDay + $uncertainty - 1);
    }

    /**
     * Detect luteal phase spotting
     */
    public function detectLutealSpotting(int $cycleDay, ?DailyHealthLog $log): bool
    {
        if (! $log) {
            return false;
        }

        // Spotting in luteal phase (day 15-28)
        return $cycleDay >= 15 && $cycleDay <= 28 && $log->spotting === true;
    }

    /**
     * Calculate CycleScore
     */
    public function calculateCycleScore(CycleVariability $variability, ?DailyHealthLog $log): float
    {
        // Base: cycle-model-only
        $score = 0.60;

        // Add based on variability
        $score += match ($variability) {
            CycleVariability::REGULAR => 0.10,
            CycleVariability::SEMI_IRREGULAR => 0.05,
            CycleVariability::IRREGULAR => 0,
        };

        // Add for strong fertile signals
        if ($log && $this->hasStrongFertileSignals($log)) {
            $score += 0.10;
        }

        // Cap for irregular cycles
        if ($variability === CycleVariability::IRREGULAR) {
            $score = min($score, 0.40);
        }

        // Maximum cap
        return min($score, 0.85);
    }

    /**
     * Check for strong fertile signals (EWCM, ovulation cramps, etc.)
     */
    private function hasStrongFertileSignals(?DailyHealthLog $log): bool
    {
        if (! $log) {
            return false;
        }

        // EWCM (egg white cervical mucus)
        if ($log->discharge_texture === 'egg_white') {
            return true;
        }

        // Ovulation cramps (ovarian pain)
        if ($log->ovarian_pain_intensity !== null) {
            return true;
        }

        // High libido
        if (is_array($log->sexual_activities) && in_array('high_desire', $log->sexual_activities)) {
            return true;
        }

        // Watery discharge
        if ($log->discharge_texture === 'watery') {
            return true;
        }

        return false;
    }

    /**
     * Calculate age factor
     */
    public function calculateAgeFactor(): float
    {
        if (! $this->profile || ! $this->profile->birthday) {
            return 1.0;
        }

        $age = Carbon::parse($this->profile->birthday)->age;

        foreach (self::AGE_FACTORS as [$minAge, $maxAge, $factor]) {
            if ($age >= $minAge && $age <= $maxAge) {
                return $factor;
            }
        }

        return 0.35;
    }

    /**
     * Get base probability by day from ovulation
     */
    public function getBaseProbability(int $dayFromOvulation): float
    {
        return self::BASE_PROBABILITY_MAP[$dayFromOvulation] ?? 0.0;
    }

    /**
     * Calculate symptom score
     */
    public function calculateSymptomScore(?DailyHealthLog $log, bool $isPmsWindow, bool $isLutealSpotting): float
    {
        if (! $log) {
            return 1.0;
        }

        $positiveScore = 1.0;
        $negativeMultiplier = 1.0;

        // Positive symptoms
        if ($log->discharge_texture === 'egg_white') {
            $positiveScore += (self::POSITIVE_SYMPTOMS['ewcm'] - 1);
        }

        if ($log->ovarian_pain_intensity !== null) {
            $positiveScore += (self::POSITIVE_SYMPTOMS['ovulation_cramps'] - 1);
        }

        if (is_array($log->sexual_activities) && in_array('high_desire', $log->sexual_activities)) {
            $positiveScore += (self::POSITIVE_SYMPTOMS['high_libido'] - 1);
        }

        if ($log->discharge_texture === 'watery') {
            $positiveScore += (self::POSITIVE_SYMPTOMS['watery_discharge'] - 1);
        }

        if ($log->bloating_intensity !== null) {
            $positiveScore += (self::POSITIVE_SYMPTOMS['abdominal_heaviness'] - 1);
        }

        // Cap positive score at 1.4
        $positiveScore = min($positiveScore, 1.4);

        // Negative symptoms
        if ($isPmsWindow) {
            $negativeMultiplier = min($negativeMultiplier, self::NEGATIVE_SYMPTOMS['pms']);
        }

        if ($log->vaginal_dryness === true) {
            $negativeMultiplier = min($negativeMultiplier, self::NEGATIVE_SYMPTOMS['dry_mucus']);
        }

        if ($isLutealSpotting) {
            $negativeMultiplier = min($negativeMultiplier, self::NEGATIVE_SYMPTOMS['luteal_spotting']);
        }

        // Return the higher impact (positive boost or negative reduction)
        return max($positiveScore, $negativeMultiplier);
    }

    /**
     * Calculate final pregnancy probability
     */
    public function calculateFinalProbability(
        float $baseProbability,
        float $ageFactor,
        float $cycleScore,
        float $symptomScore
    ): float {
        $result = $baseProbability * $ageFactor * $cycleScore * $symptomScore;

        return min($result, 0.35); // Cap at 35%
    }

    /**
     * Generate text flags for today page
     */
    public function generateTextFlags(
        CyclePhase $phase,
        CycleSubphase $subphase,
        bool $isFertileWindow,
        bool $isPmsWindow,
        bool $isPeriodTomorrow,
        float $finalProbability,
        CycleVariability $variability
    ): array {
        $flags = [];
        $locale = $this->locale;

        // Fertility status
        if ($isFertileWindow) {
            $flags['fertility_status'] = [
                'en' => 'You are in your fertile window. Pregnancy chance is higher.',
                'fa' => 'شما در پنجره باروری هستید. احتمال بارداری بیشتر است.',
            ];
        }

        // Probability message
        $probPercent = round($finalProbability * 100, 1);
        $flags['probability_message'] = [
            'en' => "Today's pregnancy probability is {$probPercent}%.",
            'fa' => "احتمال بارداری امروز {$probPercent}٪ است.",
        ];

        // PMS warning
        if ($isPmsWindow) {
            $flags['pms_warning'] = [
                'en' => 'You are in the PMS window. You may experience mood changes and physical symptoms.',
                'fa' => 'شما در دوره PMS هستید. ممکن است تغییرات خلقی و علائم جسمی را تجربه کنید.',
            ];
        }

        // Period prediction
        if ($isPeriodTomorrow) {
            $uncertainty = $variability->uncertaintyRange();
            $flags['period_prediction'] = [
                'en' => "Your period may start soon (±{$uncertainty} days based on your cycle regularity).",
                'fa' => "پریود شما ممکن است به زودی شروع شود (±{$uncertainty} روز بر اساس نظم سیکل شما).",
            ];
        }

        // Phase info
        $flags['phase_info'] = [
            'en' => "Current phase: {$phase->label('en')} ({$subphase->label('en')})",
            'fa' => "فاز فعلی: {$phase->label('fa')} ({$subphase->label('fa')})",
        ];

        return $flags;
    }

    /**
     * Generate daily tips based on phase and symptoms
     */
    public function generateDailyTips(CyclePhase $phase, CycleSubphase $subphase, ?DailyHealthLog $log): array
    {
        $tips = [];

        // Phase-specific tips
        $tips = array_merge($tips, $this->getPhaseBasedTips($phase, $subphase));

        // Symptom-specific tips
        if ($log) {
            $tips = array_merge($tips, $this->getSymptomBasedTips($log));
        }

        return $tips;
    }

    /**
     * Get tips based on current phase
     */
    private function getPhaseBasedTips(CyclePhase $phase, CycleSubphase $subphase): array
    {
        $tips = [];

        switch ($phase) {
            case CyclePhase::MENSTRUATION:
                $tips[] = [
                    'type' => 'hydration',
                    'en' => 'Stay hydrated and drink plenty of water.',
                    'fa' => 'آب کافی بنوشید و هیدراته بمانید.',
                ];
                $tips[] = [
                    'type' => 'warmth',
                    'en' => 'Apply a warm compress to your lower abdomen to relieve cramps.',
                    'fa' => 'کمپرس گرم روی شکم قرار دهید تا دردها کاهش یابد.',
                ];
                $tips[] = [
                    'type' => 'rest',
                    'en' => 'Get enough rest and avoid strenuous activities.',
                    'fa' => 'استراحت کافی داشته باشید و از فعالیت‌های سنگین اجتناب کنید.',
                ];
                $tips[] = [
                    'type' => 'nutrition',
                    'en' => 'Eat iron-rich foods like spinach and red meat.',
                    'fa' => 'غذاهای غنی از آهن مثل اسفناج و گوشت قرمز بخورید.',
                ];
                break;

            case CyclePhase::FOLLICULAR:
                $tips[] = [
                    'type' => 'energy',
                    'en' => 'Your energy levels are rising. Great time for new projects!',
                    'fa' => 'سطح انرژی شما در حال افزایش است. زمان مناسبی برای پروژه‌های جدید!',
                ];
                $tips[] = [
                    'type' => 'exercise',
                    'en' => 'Perfect time for high-intensity workouts.',
                    'fa' => 'زمان مناسبی برای ورزش‌های سنگین است.',
                ];
                $tips[] = [
                    'type' => 'nutrition',
                    'en' => 'Add protein and fresh vegetables to rebuild your energy stores.',
                    'fa' => 'پروتئین و سبزیجات تازه بخورید تا ذخیره انرژی‌تان دوباره پر شود.',
                ];
                $tips[] = [
                    'type' => 'mental_health',
                    'en' => 'A good window for planning and learning something new.',
                    'fa' => 'فرصت خوبی برای برنامه‌ریزی و یادگیری چیزهای تازه است.',
                ];
                break;

            case CyclePhase::OVULATION:
                $tips[] = [
                    'type' => 'fertility',
                    'en' => 'Peak fertility days. If trying to conceive, this is the best time.',
                    'fa' => 'روزهای اوج باروری. اگر قصد بارداری دارید، بهترین زمان است.',
                ];
                $tips[] = [
                    'type' => 'energy',
                    'en' => 'You may feel more confident and social.',
                    'fa' => 'ممکن است احساس اعتماد به نفس و اجتماعی بودن بیشتری داشته باشید.',
                ];
                $tips[] = [
                    'type' => 'hydration',
                    'en' => 'Drink water regularly — hydration supports cervical fluid.',
                    'fa' => 'مرتب آب بنوشید؛ هیدراته بودن به کیفیت ترشحات کمک می‌کند.',
                ];
                $tips[] = [
                    'type' => 'sleep',
                    'en' => 'Keep a steady sleep schedule to support hormone balance.',
                    'fa' => 'برنامه خواب منظمی داشته باشید تا تعادل هورمونی حفظ شود.',
                ];
                break;

            case CyclePhase::LUTEAL:
                if ($subphase === CycleSubphase::LATE_LUTEAL || $subphase === CycleSubphase::PMS_POSSIBLE) {
                    $tips[] = [
                        'type' => 'pms',
                        'en' => 'PMS symptoms may appear. Practice self-care and relaxation.',
                        'fa' => 'علائم PMS ممکن است ظاهر شوند. مراقبت از خود و آرامش را تمرین کنید.',
                    ];
                    $tips[] = [
                        'type' => 'nutrition',
                        'en' => 'Reduce salt and caffeine intake to minimize bloating.',
                        'fa' => 'مصرف نمک و کافئین را کاهش دهید تا نفخ کم شود.',
                    ];
                    $tips[] = [
                        'type' => 'mood',
                        'en' => 'Take breaks and practice deep breathing if feeling irritable.',
                        'fa' => 'استراحت کنید و اگر احساس تحریک‌پذیری دارید تنفس عمیق تمرین کنید.',
                    ];
                } else {
                    $tips[] = [
                        'type' => 'nutrition',
                        'en' => 'Focus on foods rich in magnesium and vitamin B6.',
                        'fa' => 'روی غذاهای غنی از منیزیم و ویتامین B6 تمرکز کنید.',
                    ];
                    $tips[] = [
                        'type' => 'exercise',
                        'en' => 'Switch to gentler movement like yoga, pilates or walking.',
                        'fa' => 'به حرکات ملایم‌تر مثل یوگا، پیلاتس یا پیاده‌روی رو بیاورید.',
                    ];
                    $tips[] = [
                        'type' => 'sleep',
                        'en' => 'Go to bed a little earlier — energy dips in this phase.',
                        'fa' => 'کمی زودتر بخوابید؛ در این فاز انرژی کم‌کم افت می‌کند.',
                    ];
                    $tips[] = [
                        'type' => 'mental_health',
                        'en' => 'A good time to finish what you started rather than begin new things.',
                        'fa' => 'زمان خوبی برای تمام‌کردن کارهای نیمه‌تمام است تا شروع کارهای جدید.',
                    ];
                }
                break;
        }

        return $tips;
    }

    /**
     * Get tips based on reported symptoms
     */
    private function getSymptomBasedTips(?DailyHealthLog $log): array
    {
        $tips = [];

        if (! $log) {
            return $tips;
        }

        // Headache tips
        if ($log->headache_intensity !== null) {
            $tips[] = [
                'type' => 'pain_relief',
                'en' => 'For headaches: rest in a dark room and stay hydrated.',
                'fa' => 'برای سردرد: در اتاق تاریک استراحت کنید و آب بنوشید.',
            ];
        }

        // Cramps tips
        if ($log->pelvic_pain_intensity !== null || $log->stomach_ache_intensity !== null) {
            $tips[] = [
                'type' => 'pain_relief',
                'en' => 'For cramps: try a warm bath or heating pad.',
                'fa' => 'برای کرامپ: حمام گرم یا پد گرم‌کننده امتحان کنید.',
            ];
        }

        // Sleep issues
        if ($log->sleep_quality === 'bad') {
            $tips[] = [
                'type' => 'sleep',
                'en' => 'Try to maintain a regular sleep schedule and avoid screens before bed.',
                'fa' => 'سعی کنید برنامه خواب منظم داشته باشید و قبل از خواب از صفحه نمایش استفاده نکنید.',
            ];
        }

        // Mood support
        if (is_array($log->moods) && (in_array('anxious', $log->moods) || in_array('sad', $log->moods))) {
            $tips[] = [
                'type' => 'mental_health',
                'en' => 'Take time for activities you enjoy. Consider light exercise or meditation.',
                'fa' => 'وقتی برای فعالیت‌های مورد علاقه‌تان بگذارید. ورزش سبک یا مدیتیشن را در نظر بگیرید.',
            ];
        }

        // Bloating
        if ($log->bloating_intensity !== null) {
            $tips[] = [
                'type' => 'digestion',
                'en' => 'Reduce salt intake and eat smaller, frequent meals.',
                'fa' => 'مصرف نمک را کم کنید و وعده‌های کوچک و مکرر بخورید.',
            ];
        }

        // Fatigue
        if ($log->fatigue === true) {
            $tips[] = [
                'type' => 'energy',
                'en' => 'Listen to your body. Take short breaks and prioritize rest.',
                'fa' => 'به بدنتان گوش دهید. استراحت‌های کوتاه داشته باشید و استراحت را اولویت قرار دهید.',
            ];
        }

        return $tips;
    }

    /**
     * Get daily log for a specific date
     */
    private function getDailyLog(Carbon $date): ?DailyHealthLog
    {
        return $this->user->dailyHealthLogs()
            ->whereDate('log_date', $date)
            ->first();
    }

    /**
     * Get cycle histories for the user
     */
    private function getCycleHistories(): Collection
    {
        // Full history (no ->take cap): CycleMetricsCalculator internally limits itself to
        // the last 3 valid cycles, and CycleDayViewBuilder loads the full history too — a
        // cap here made the effective values in `calculation` and `cycle_view` disagree.
        return CycleHistory::where('user_id', $this->user->id)
            ->orderBy('period_start_date', 'desc')
            ->get();
    }

    /**
     * Get profile data snapshot
     */
    private function getProfileSnapshot(): array
    {
        if (! $this->profile) {
            return [];
        }

        return [
            'birthday' => $this->profile->birthday?->toDateString(),
            'period_duration' => $this->profile->period_duration,
            'cycle_duration' => $this->profile->cycle_duration,
            'last_period_start' => $this->profile->last_period_start?->toDateString(),
        ];
    }

    /**
     * Return empty calculation when no profile data available
     */
    private function getEmptyCalculation(Carbon $date): array
    {
        return [
            'calculation_date' => $date->toDateString(),
            'cycle_day' => null,
            'phase' => null,
            'subphase' => null,
            'estimated_ovulation_day' => null,
            'cycle_length_used' => null,
            'is_fertile_window' => false,
            'is_pms_window' => false,
            'is_period_tomorrow' => false,
            'is_luteal_spotting' => false,
            'cycle_score' => null,
            'age_factor' => null,
            'base_probability' => null,
            'symptom_score' => null,
            'final_probability' => null,
            'cycle_variability' => null,
            'uncertainty_range' => null,
            'text_flags' => [
                'incomplete_profile' => [
                    'en' => 'Please complete your profile to get cycle predictions.',
                    'fa' => 'لطفاً پروفایل خود را کامل کنید تا پیش‌بینی سیکل را دریافت کنید.',
                ],
            ],
            'daily_tips' => [],
            'source_profile_data' => null,
            'source_daily_log_data' => null,
        ];
    }
}
