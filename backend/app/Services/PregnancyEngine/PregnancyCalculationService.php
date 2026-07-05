<?php

namespace App\Services\PregnancyEngine;

use App\Enums\ConfidenceLevel;
use App\Enums\PregnancyAgeSource;
use App\Models\PregnancyProfile;
use App\Models\User;
use Carbon\Carbon;

class PregnancyCalculationService
{
    private User $user;
    private ?PregnancyProfile $profile;
    private string $locale;

    public function __construct(User $user, string $locale = 'en')
    {
        $this->user = $user;
        $this->profile = $user->pregnancyProfile;
        $this->locale = $locale;
    }

    /**
     * Calculate gestational age based on source
     * Returns array with weeks, days, and total days
     */
    public function calculateGestationalAge(?Carbon $referenceDate = null): array
    {
        $referenceDate = $referenceDate ?? Carbon::today();

        if (!$this->profile) {
            return $this->getEmptyGestationalAge();
        }

        $ageSource = $this->profile->age_source;

        return match ($ageSource) {
            PregnancyAgeSource::LMP->value => $this->calculateFromLmp($referenceDate),
            PregnancyAgeSource::ULTRASOUND->value => $this->calculateFromUltrasound($referenceDate),
            PregnancyAgeSource::MANUAL->value => $this->calculateFromManual($referenceDate),
            default => $this->getEmptyGestationalAge(),
        };
    }

    /**
     * Calculate from LMP (Last Menstrual Period)
     * Gestational age = Today - LMP date
     * Uncertainty: ±3 days
     * Confidence: Medium
     */
    private function calculateFromLmp(Carbon $referenceDate): array
    {
        if (!$this->profile->lmp_date) {
            return $this->getEmptyGestationalAge();
        }

        $lmpDate = Carbon::parse($this->profile->lmp_date);
        $totalDays = $lmpDate->diffInDays($referenceDate);

        return $this->formatGestationalAge($totalDays, ConfidenceLevel::MEDIUM, 3);
    }

    /**
     * Calculate from ultrasound data
     * Uses ultrasound date and reported gestational age to calculate current age
     * Uncertainty: ±1 day
     * Confidence: High
     */
    private function calculateFromUltrasound(Carbon $referenceDate): array
    {
        if (!$this->profile->ultrasound_date ||
            $this->profile->ultrasound_weeks === null ||
            $this->profile->ultrasound_days === null) {
            return $this->getEmptyGestationalAge();
        }

        $ultrasoundDate = Carbon::parse($this->profile->ultrasound_date);
        $ultrasoundAgeDays = ($this->profile->ultrasound_weeks * 7) + $this->profile->ultrasound_days;

        // Days since ultrasound
        $daysSinceUltrasound = $ultrasoundDate->diffInDays($referenceDate);

        // Current gestational age
        $totalDays = $ultrasoundAgeDays + $daysSinceUltrasound;

        return $this->formatGestationalAge($totalDays, ConfidenceLevel::HIGH, 1);
    }

    /**
     * Calculate from manual entry
     * Uses manually entered week and day, adjusted from entry date
     * Uncertainty: varies based on user certainty
     * Confidence: Low
     */
    private function calculateFromManual(Carbon $referenceDate): array
    {
        if ($this->profile->manual_weeks === null ||
            $this->profile->manual_days === null ||
            !$this->profile->manual_entry_date) {
            return $this->getEmptyGestationalAge();
        }

        $entryDate = Carbon::parse($this->profile->manual_entry_date);
        $entryAgeDays = ($this->profile->manual_weeks * 7) + $this->profile->manual_days;

        // Days since manual entry
        $daysSinceEntry = $entryDate->diffInDays($referenceDate);

        // Current gestational age
        $totalDays = $entryAgeDays + $daysSinceEntry;

        return $this->formatGestationalAge($totalDays, ConfidenceLevel::LOW, 5);
    }

    /**
     * Format gestational age output
     */
    private function formatGestationalAge(int $totalDays, ConfidenceLevel $confidence, int $uncertainty): array
    {
        $weeks = (int) floor($totalDays / 7);
        $days = $totalDays % 7;

        // Trimester calculation
        $trimester = $this->calculateTrimester($weeks);

        return [
            'weeks' => $weeks,
            'days' => $days,
            'total_days' => $totalDays,
            'trimester' => $trimester,
            'confidence_level' => $confidence->value,
            'confidence_label' => $confidence->label($this->locale),
            'uncertainty_days' => $uncertainty,
            'formatted' => [
                'en' => "{$weeks} weeks, {$days} days",
                'fa' => "{$weeks} هفته و {$days} روز",
            ],
        ];
    }

    /**
     * Calculate trimester based on weeks
     */
    private function calculateTrimester(int $weeks): int
    {
        if ($weeks <= 12) {
            return 1;
        }
        if ($weeks <= 27) {
            return 2;
        }
        return 3;
    }

    /**
     * Calculate Estimated Due Date (EDD)
     * Standard calculation: LMP + 280 days
     */
    public function calculateEDD(): ?array
    {
        if (!$this->profile) {
            return null;
        }

        $ageSource = $this->profile->age_source;

        $edd = match ($ageSource) {
            PregnancyAgeSource::LMP->value => $this->calculateEDDFromLmp(),
            PregnancyAgeSource::ULTRASOUND->value => $this->calculateEDDFromUltrasound(),
            PregnancyAgeSource::MANUAL->value => $this->calculateEDDFromManual(),
            default => null,
        };

        if (!$edd) {
            return null;
        }

        $daysRemaining = Carbon::today()->diffInDays($edd, false);

        return [
            'date' => $edd->toDateString(),
            'days_remaining' => max(0, $daysRemaining),
            'weeks_remaining' => max(0, (int) floor($daysRemaining / 7)),
            'formatted' => [
                'en' => $edd->format('F j, Y'),
                'fa' => $this->formatPersianDate($edd),
            ],
        ];
    }

    /**
     * Calculate EDD from LMP: LMP + 280 days
     */
    private function calculateEDDFromLmp(): ?Carbon
    {
        if (!$this->profile->lmp_date) {
            return null;
        }

        return Carbon::parse($this->profile->lmp_date)->addDays(280);
    }

    /**
     * Calculate EDD from ultrasound
     * First calculate estimated LMP from ultrasound data, then add 280 days
     */
    private function calculateEDDFromUltrasound(): ?Carbon
    {
        if (!$this->profile->ultrasound_date ||
            $this->profile->ultrasound_weeks === null ||
            $this->profile->ultrasound_days === null) {
            return null;
        }

        $ultrasoundDate = Carbon::parse($this->profile->ultrasound_date);
        $ultrasoundAgeDays = ($this->profile->ultrasound_weeks * 7) + $this->profile->ultrasound_days;

        // Calculate estimated LMP (ultrasound date - gestational age at ultrasound)
        $estimatedLmp = $ultrasoundDate->copy()->subDays($ultrasoundAgeDays);

        // EDD = estimated LMP + 280 days
        return $estimatedLmp->addDays(280);
    }

    /**
     * Calculate EDD from manual entry
     */
    private function calculateEDDFromManual(): ?Carbon
    {
        if ($this->profile->manual_weeks === null ||
            $this->profile->manual_days === null ||
            !$this->profile->manual_entry_date) {
            return null;
        }

        $entryDate = Carbon::parse($this->profile->manual_entry_date);
        $entryAgeDays = ($this->profile->manual_weeks * 7) + $this->profile->manual_days;

        // Calculate estimated LMP
        $estimatedLmp = $entryDate->copy()->subDays($entryAgeDays);

        // EDD = estimated LMP + 280 days
        return $estimatedLmp->addDays(280);
    }

    /**
     * Calculate estimated conception date
     * Typically 2 weeks after LMP
     */
    public function calculateConceptionDate(): ?array
    {
        $gestationalAge = $this->calculateGestationalAge();

        if ($gestationalAge['total_days'] === null) {
            return null;
        }

        // Conception is approximately 2 weeks (14 days) after LMP
        $totalDays = $gestationalAge['total_days'];
        $conceptionDate = Carbon::today()->subDays($totalDays - 14);

        return [
            'date' => $conceptionDate->toDateString(),
            'formatted' => [
                'en' => $conceptionDate->format('F j, Y'),
                'fa' => $this->formatPersianDate($conceptionDate),
            ],
        ];
    }

    /**
     * Get current pregnancy week number (1-40)
     */
    public function getCurrentWeek(): int
    {
        $gestationalAge = $this->calculateGestationalAge();
        return min(40, max(1, $gestationalAge['weeks'] + 1)); // Week 1 starts at 0 weeks
    }

    /**
     * Check if fetal movement tracking should be active (after week 18)
     */
    public function isFetalMovementTrackingActive(): bool
    {
        return $this->getCurrentWeek() >= 18;
    }

    /**
     * Check if daily fetal movement logging is required (after week 24)
     */
    public function isFetalMovementRequired(): bool
    {
        return $this->getCurrentWeek() >= 24;
    }

    /**
     * Get pregnancy status summary
     */
    public function getPregnancyStatus(): array
    {
        if (!$this->profile) {
            return $this->getEmptyStatus();
        }

        $gestationalAge = $this->calculateGestationalAge();
        $edd = $this->calculateEDD();
        $currentWeek = $this->getCurrentWeek();

        return [
            'is_active' => $this->profile->pregnancy_mode,
            'gestational_age' => $gestationalAge,
            'estimated_due_date' => $edd,
            'current_week' => $currentWeek,
            'trimester' => $gestationalAge['trimester'] ?? null,
            'age_source' => $this->profile->age_source,
            'confidence_level' => $gestationalAge['confidence_level'] ?? null,
            'is_high_risk' => $this->isHighRisk(),
            'fetal_movement_tracking_active' => $this->isFetalMovementTrackingActive(),
            'fetal_movement_required' => $this->isFetalMovementRequired(),
            'flags' => $this->generateStatusFlags($gestationalAge, $currentWeek),
        ];
    }

    /**
     * Check if pregnancy is considered high risk
     */
    public function isHighRisk(): bool
    {
        if (!$this->profile) {
            return false;
        }

        // Check history
        if ($this->profile->has_miscarriage_history || $this->profile->has_high_risk_history) {
            return true;
        }

        // Check RH negative
        if ($this->profile->rh_factor === 'negative') {
            return true;
        }

        // Check pre-existing conditions
        $conditions = $this->profile->pre_existing_conditions ?? [];
        $highRiskConditions = ['chronic_hypertension', 'diabetes'];

        foreach ($conditions as $condition) {
            if (in_array($condition, $highRiskConditions)) {
                return true;
            }
        }

        return false;
    }

    /**
     * Generate status flags for display
     */
    private function generateStatusFlags(array $gestationalAge, int $currentWeek): array
    {
        $flags = [];
        $locale = $this->locale;

        // Trimester info
        $trimester = $gestationalAge['trimester'] ?? 1;
        $flags['trimester_info'] = [
            'en' => "You are in trimester {$trimester}.",
            'fa' => "شما در سه‌ماهه {$trimester} هستید.",
        ];

        // Week info
        $flags['week_info'] = [
            'en' => "Week {$currentWeek} of pregnancy.",
            'fa' => "هفته {$currentWeek} بارداری.",
        ];

        // Fetal movement reminder (after week 18)
        if ($currentWeek >= 18 && $currentWeek < 24) {
            $flags['fetal_movement'] = [
                'en' => "You may start feeling baby movements. Track them when you notice!",
                'fa' => "ممکن است حرکات جنین را احساس کنید. وقتی متوجه شدید ثبت کنید!",
            ];
        }

        // Required tracking (after week 24)
        if ($currentWeek >= 24) {
            $flags['daily_tracking'] = [
                'en' => "Daily fetal movement tracking is important. Please log movements daily.",
                'fa' => "ثبت روزانه حرکات جنین مهم است. لطفاً روزانه ثبت کنید.",
            ];
        }

        // RH negative warning
        if ($this->profile && $this->profile->rh_factor === 'negative') {
            $flags['rh_warning'] = [
                'en' => "RH negative detected. Special monitoring is recommended.",
                'fa' => "RH منفی شناسایی شد. مراقبت ویژه توصیه می‌شود.",
            ];
        }

        // High risk flag
        if ($this->isHighRisk()) {
            $flags['high_risk'] = [
                'en' => "Based on your profile, extra monitoring is recommended.",
                'fa' => "بر اساس پروفایل شما، مراقبت بیشتر توصیه می‌شود.",
            ];
        }

        return $flags;
    }

    /**
     * Format date in Persian (Jalali) - simplified version
     */
    private function formatPersianDate(Carbon $date): string
    {
        // Using Gregorian for now, can be enhanced with Jalali library
        $months = [
            'January' => 'ژانویه', 'February' => 'فوریه', 'March' => 'مارس',
            'April' => 'آوریل', 'May' => 'مه', 'June' => 'ژوئن',
            'July' => 'ژوئیه', 'August' => 'اوت', 'September' => 'سپتامبر',
            'October' => 'اکتبر', 'November' => 'نوامبر', 'December' => 'دسامبر',
        ];

        $month = $months[$date->format('F')] ?? $date->format('F');
        return "{$date->day} {$month} {$date->year}";
    }

    /**
     * Get empty gestational age response
     */
    private function getEmptyGestationalAge(): array
    {
        return [
            'weeks' => null,
            'days' => null,
            'total_days' => null,
            'trimester' => null,
            'confidence_level' => null,
            'confidence_label' => null,
            'uncertainty_days' => null,
            'formatted' => null,
        ];
    }

    /**
     * Get empty status response
     */
    private function getEmptyStatus(): array
    {
        return [
            'is_active' => false,
            'gestational_age' => $this->getEmptyGestationalAge(),
            'estimated_due_date' => null,
            'current_week' => null,
            'trimester' => null,
            'age_source' => null,
            'confidence_level' => null,
            'is_high_risk' => false,
            'fetal_movement_tracking_active' => false,
            'fetal_movement_required' => false,
            'flags' => [
                'no_profile' => [
                    'en' => 'Please complete pregnancy onboarding to get started.',
                    'fa' => 'لطفاً آنبوردینگ بارداری را تکمیل کنید.',
                ],
            ],
        ];
    }
}
