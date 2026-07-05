<?php

namespace App\Jobs;

use App\Enums\CalculationStatus;
use App\Models\CycleCalculation;
use App\Models\User;
use App\Services\HealthEngine\HealthDataEngine;
use Carbon\Carbon;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

class CalculateCycleDataJob implements ShouldQueue
{
    use Queueable;

    public int $tries = 3;
    public int $backoff = 60;

    private int $userId;
    private int $version;
    private ?string $locale;

    /**
     * Create a new job instance.
     */
    public function __construct(int $userId, int $version, ?string $locale = 'en')
    {
        $this->userId = $userId;
        $this->version = $version;
        $this->locale = $locale;
    }

    /**
     * Execute the job.
     */
    public function handle(): void
    {
        $user = User::with('profile')->find($this->userId);

        if (!$user || !$user->profile) {
            Log::warning("CalculateCycleDataJob: User {$this->userId} not found or has no profile");
            return;
        }

        $profile = $user->profile;

        // Mark as processing
        $profile->update([
            'calculation_status' => CalculationStatus::PROCESSING->value,
            'calculation_started_at' => now(),
        ]);

        try {
            $this->processCalculations($user);

            // Mark as completed
            $profile->update([
                'calculation_status' => CalculationStatus::COMPLETED->value,
                'calculation_completed_at' => now(),
                'calculation_version' => $this->version,
            ]);

            Log::info("CalculateCycleDataJob: Completed calculations for user {$this->userId}, version {$this->version}");
        } catch (\Exception $e) {
            // Mark as failed
            $profile->update([
                'calculation_status' => CalculationStatus::FAILED->value,
            ]);

            Log::error("CalculateCycleDataJob: Failed for user {$this->userId}", [
                'error' => $e->getMessage(),
                'trace' => $e->getTraceAsString(),
            ]);

            throw $e;
        }
    }

    /**
     * Process calculations for one year
     */
    private function processCalculations(User $user): void
    {
        $engine = new HealthDataEngine($user, $this->locale ?? 'en');
        $today = Carbon::today();

        // Lock past days (before today) that are already calculated
        $this->lockPastCalculations($user->id, $today);

        // Calculate for one year from today
        $startDate = $today;
        $endDate = $today->copy()->addYear();

        $calculations = [];
        $currentDate = $startDate->copy();

        while ($currentDate <= $endDate) {
            $calculationData = $engine->calculateForDate($currentDate);

            $calculations[] = [
                'user_id' => $user->id,
                'version' => $this->version,
                'is_locked' => false, // Future dates are not locked
                ...$calculationData,
                'text_flags' => json_encode($calculationData['text_flags']),
                'daily_tips' => json_encode($calculationData['daily_tips']),
                'source_profile_data' => json_encode($calculationData['source_profile_data']),
                'source_daily_log_data' => json_encode($calculationData['source_daily_log_data']),
                'created_at' => now(),
                'updated_at' => now(),
            ];

            // Batch insert every 100 records
            if (count($calculations) >= 100) {
                $this->upsertCalculations($calculations);
                $calculations = [];
            }

            $currentDate->addDay();
        }

        // Insert remaining calculations
        if (!empty($calculations)) {
            $this->upsertCalculations($calculations);
        }
    }

    /**
     * Lock past calculations so they won't be modified
     */
    private function lockPastCalculations(int $userId, Carbon $today): void
    {
        CycleCalculation::where('user_id', $userId)
            ->where('calculation_date', '<', $today)
            ->where('is_locked', false)
            ->update(['is_locked' => true]);
    }

    /**
     * Upsert calculations (update if exists for same date/version, insert otherwise)
     */
    private function upsertCalculations(array $calculations): void
    {
        DB::table('cycle_calculations')->upsert(
            $calculations,
            ['user_id', 'calculation_date', 'version'],
            [
                'cycle_day',
                'phase',
                'subphase',
                'estimated_ovulation_day',
                'cycle_length_used',
                'is_fertile_window',
                'is_pms_window',
                'is_period_tomorrow',
                'is_luteal_spotting',
                'cycle_score',
                'age_factor',
                'base_probability',
                'symptom_score',
                'final_probability',
                'cycle_variability',
                'uncertainty_range',
                'text_flags',
                'daily_tips',
                'source_profile_data',
                'source_daily_log_data',
                'updated_at',
            ]
        );
    }
}
