<?php

namespace App\Services\MessageSystem\Core;

use App\Models\User;
use App\Services\MessageSystem\Enums\MessageMode;
use Carbon\Carbon;

/**
 * Data Transfer Object for message context
 * Contains all information needed to generate personalized messages
 */
class MessageContext
{
    public function __construct(
        // User & Settings
        public readonly User $user,
        public readonly string $locale,
        public readonly Carbon $date,

        // Mode Detection
        public readonly MessageMode $mode,

        // User Preferences
        public readonly string $userGoal,        // ttc, non_ttc
        public readonly string $subscriptionType, // free, premium

        // Cycle Context (for CYCLE mode)
        public readonly ?string $cyclePhase = null,
        public readonly ?string $cycleSubphase = null,
        public readonly ?int $cycleDay = null,
        public readonly ?int $cycleLength = null,
        public readonly ?bool $isFertileWindow = null,
        public readonly ?bool $isPmsWindow = null,
        public readonly ?int $estimatedOvulationDay = null,

        // Pregnancy Context (for PREGNANCY mode)
        public readonly ?int $pregnancyWeek = null,
        public readonly ?int $pregnancyDay = null,
        public readonly ?int $trimester = null,
        public readonly ?Carbon $dueDate = null,

        // Health Data (shared across modes)
        public readonly ?object $dailyLog = null,
        public readonly ?array $symptoms = null,
        public readonly ?array $recentLogs = null, // For pattern detection
    ) {}

    /**
     * Check if user is trying to conceive
     */
    public function isTTC(): bool
    {
        return $this->userGoal === 'ttc';
    }

    /**
     * Check if user has premium subscription
     */
    public function isPremium(): bool
    {
        return $this->subscriptionType === 'premium';
    }

    /**
     * Check if in cycle mode
     */
    public function isCycleMode(): bool
    {
        return $this->mode === MessageMode::CYCLE;
    }

    /**
     * Check if in pregnancy mode
     */
    public function isPregnancyMode(): bool
    {
        return $this->mode === MessageMode::PREGNANCY;
    }

    /**
     * Get formatted gestational age string
     */
    public function getGestationalAgeString(): ?string
    {
        if (!$this->isPregnancyMode() || $this->pregnancyWeek === null) {
            return null;
        }

        $day = $this->pregnancyDay ?? 0;

        if ($this->locale === 'fa') {
            return "هفته {$this->pregnancyWeek} و روز {$day}";
        }

        return "Week {$this->pregnancyWeek}, Day {$day}";
    }

    /**
     * Convert to array for debugging/logging
     */
    public function toArray(): array
    {
        return [
            'user_id' => $this->user->id,
            'locale' => $this->locale,
            'date' => $this->date->toDateString(),
            'mode' => $this->mode->value,
            'user_goal' => $this->userGoal,
            'subscription_type' => $this->subscriptionType,
            'cycle_phase' => $this->cyclePhase,
            'cycle_subphase' => $this->cycleSubphase,
            'cycle_day' => $this->cycleDay,
            'is_fertile_window' => $this->isFertileWindow,
            'is_pms_window' => $this->isPmsWindow,
            'pregnancy_week' => $this->pregnancyWeek,
            'pregnancy_day' => $this->pregnancyDay,
            'trimester' => $this->trimester,
            'has_daily_log' => $this->dailyLog !== null,
            'symptoms_count' => $this->symptoms ? count($this->symptoms) : 0,
        ];
    }
}
