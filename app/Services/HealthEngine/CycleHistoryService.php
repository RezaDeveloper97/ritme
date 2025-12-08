<?php

namespace App\Services\HealthEngine;

use App\Models\CycleHistory;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserProfile;
use Carbon\Carbon;

class CycleHistoryService
{
    /**
     * Check if a new period has started and update cycle history
     */
    public static function checkAndUpdatePeriodStart(User $user, DailyHealthLog $log): void
    {
        // Only process if bleeding is reported
        if (!$log->bleeding_intensity) {
            return;
        }

        $logDate = Carbon::parse($log->log_date);

        // Check if yesterday had no bleeding (this could be period start)
        $yesterdayLog = $user->dailyHealthLogs()
            ->whereDate('log_date', $logDate->copy()->subDay())
            ->first();

        $isNewPeriodStart = !$yesterdayLog || !$yesterdayLog->bleeding_intensity;

        if (!$isNewPeriodStart) {
            return;
        }

        // Check if we already have this period recorded
        $existingPeriod = CycleHistory::where('user_id', $user->id)
            ->where('period_start_date', $logDate)
            ->first();

        if ($existingPeriod) {
            return;
        }

        // Get the previous period to calculate cycle length
        $previousPeriod = CycleHistory::where('user_id', $user->id)
            ->orderBy('period_start_date', 'desc')
            ->first();

        // Calculate cycle length if we have previous period
        $cycleLength = null;
        if ($previousPeriod) {
            $cycleLength = Carbon::parse($previousPeriod->period_start_date)
                ->diffInDays($logDate);

            // Also update the end date of previous period
            self::updatePreviousPeriodEndDate($user, $previousPeriod, $logDate);
        }

        // Create new period record
        CycleHistory::create([
            'user_id' => $user->id,
            'period_start_date' => $logDate,
            'cycle_length' => $cycleLength,
            'is_confirmed' => false, // User can confirm later
        ]);

        // Update user profile LMP (Last Menstrual Period)
        self::updateProfileLMP($user, $logDate);
    }

    /**
     * Update the end date of previous period
     */
    private static function updatePreviousPeriodEndDate(User $user, CycleHistory $previousPeriod, Carbon $newPeriodStart): void
    {
        // Find the last day of bleeding before the new period
        $lastBleedingDate = $user->dailyHealthLogs()
            ->whereDate('log_date', '>=', $previousPeriod->period_start_date)
            ->whereDate('log_date', '<', $newPeriodStart)
            ->whereNotNull('bleeding_intensity')
            ->orderBy('log_date', 'desc')
            ->value('log_date');

        if ($lastBleedingDate) {
            $bleedingLength = Carbon::parse($previousPeriod->period_start_date)
                ->diffInDays(Carbon::parse($lastBleedingDate)) + 1;

            $previousPeriod->update([
                'period_end_date' => $lastBleedingDate,
                'bleeding_length' => $bleedingLength,
            ]);
        }
    }

    /**
     * Update user profile with new LMP
     */
    private static function updateProfileLMP(User $user, Carbon $periodStart): void
    {
        $profile = $user->profile;

        if ($profile) {
            $profile->update([
                'last_period_start' => $periodStart,
            ]);
        }
    }

    /**
     * Detect if user is reporting spotting in luteal phase (not real period)
     * Returns true if this looks like luteal spotting, not a real period
     */
    public static function isLikelyLutealSpotting(User $user, DailyHealthLog $log): bool
    {
        if (!$log->spotting && $log->bleeding_intensity !== 'low') {
            return false;
        }

        $profile = $user->profile;
        if (!$profile || !$profile->last_period_start) {
            return false;
        }

        $cycleDay = Carbon::parse($profile->last_period_start)
            ->diffInDays(Carbon::parse($log->log_date)) + 1;

        $cycleLength = $profile->cycle_duration ?? 28;

        // If we're in day 15-28 (typical luteal phase) and it's light bleeding
        // it's more likely luteal spotting than a real period
        return $cycleDay >= 15 && $cycleDay <= ($cycleLength - 2);
    }

    /**
     * Get warning message if user might be logging spotting as period
     */
    public static function getSpottingWarning(User $user, DailyHealthLog $log, string $locale = 'en'): ?array
    {
        if (!self::isLikelyLutealSpotting($user, $log)) {
            return null;
        }

        return [
            'type' => 'luteal_spotting_warning',
            'message' => $locale === 'fa'
                ? 'این خون‌ریزی ممکن است لکه‌بینی لوتئال باشد نه شروع پریود. آیا مطمئن هستید؟'
                : 'This bleeding might be luteal spotting, not the start of your period. Are you sure?',
        ];
    }
}
