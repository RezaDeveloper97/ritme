<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\CalculationStatus;
use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\CycleVariability;
use App\Http\Controllers\Controller;
use App\Jobs\CalculateCycleDataJob;
use App\Models\CycleCalculation;
use App\Services\HealthEngine\HealthDataEngine;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class CycleCalculationController extends Controller
{
    /**
     * @OA\Get(
     *     path="/cycle/today",
     *     summary="Get today's cycle calculation",
     *     description="Retrieve calculated cycle data for today including phase, fertility window, and pregnancy probability",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for text responses (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Today's cycle calculation retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="calculation", ref="#/components/schemas/CycleCalculation"),
     *                 @OA\Property(property="calculation_status", type="string", example="completed"),
     *                 @OA\Property(property="is_recalculating", type="boolean", example=false)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function today(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $today = Carbon::today();

        return $this->getCalculationForDate($user, $today, $locale);
    }

    /**
     * @OA\Get(
     *     path="/cycle/date/{date}",
     *     summary="Get cycle calculation for specific date",
     *     description="Retrieve calculated cycle data for a specific date",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="path",
     *         description="Date (YYYY-MM-DD)",
     *         required=true,
     *         @OA\Schema(type="string", format="date", example="2024-12-15")
     *     ),
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for text responses (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Cycle calculation retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="calculation", ref="#/components/schemas/CycleCalculation"),
     *                 @OA\Property(property="calculation_status", type="string", example="completed"),
     *                 @OA\Property(property="is_recalculating", type="boolean", example=false)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function forDate(Request $request, string $date): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        try {
            $targetDate = Carbon::parse($date);
        } catch (\Exception $e) {
            return response()->json([
                'success' => false,
                'message' => 'Invalid date format. Use YYYY-MM-DD.',
            ], 422);
        }

        return $this->getCalculationForDate($user, $targetDate, $locale);
    }

    /**
     * @OA\Get(
     *     path="/cycle/month/{year}/{month}",
     *     summary="Get cycle calculations for a month",
     *     description="Retrieve all calculated cycle data for a specific month (calendar view)",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="year",
     *         in="path",
     *         description="Year",
     *         required=true,
     *         @OA\Schema(type="integer", example=2024)
     *     ),
     *     @OA\Parameter(
     *         name="month",
     *         in="path",
     *         description="Month (1-12)",
     *         required=true,
     *         @OA\Schema(type="integer", example=12)
     *     ),
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for text responses (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Monthly cycle calculations retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="calculations", type="array", @OA\Items(ref="#/components/schemas/CycleCalculation")),
     *                 @OA\Property(property="calculation_status", type="string", example="completed"),
     *                 @OA\Property(property="is_recalculating", type="boolean", example=false),
     *                 @OA\Property(property="month_summary", type="object",
     *                     @OA\Property(property="fertile_days", type="integer", example=6),
     *                     @OA\Property(property="period_days", type="integer", example=5),
     *                     @OA\Property(property="pms_days", type="integer", example=6)
     *                 )
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function month(Request $request, int $year, int $month): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $profile = $user->profile;

        if ($month < 1 || $month > 12) {
            return response()->json([
                'success' => false,
                'message' => 'Invalid month. Must be between 1 and 12.',
            ], 422);
        }

        $startDate = Carbon::createFromDate($year, $month, 1)->startOfMonth();
        $endDate = $startDate->copy()->endOfMonth();

        // Get status info
        $status = $profile?->calculation_status ?? CalculationStatus::PENDING->value;
        $isRecalculating = $status === CalculationStatus::PROCESSING->value;

        // Get stored calculations
        $calculations = CycleCalculation::where('user_id', $user->id)
            ->whereBetween('calculation_date', [$startDate, $endDate])
            ->orderBy('calculation_date')
            ->get()
            ->keyBy(fn($c) => $c->calculation_date->toDateString());

        // If no calculations exist or recalculating, calculate on-the-fly
        $engine = new HealthDataEngine($user, $locale);
        $result = [];

        $currentDate = $startDate->copy();
        while ($currentDate <= $endDate) {
            $dateStr = $currentDate->toDateString();

            if ($calculations->has($dateStr)) {
                $calc = $calculations[$dateStr];
                // Localize text_flags based on locale
                $result[] = $this->localizeCalculation($calc->toArray(), $locale);
            } else {
                // Calculate on-the-fly
                $result[] = $engine->calculateForDate($currentDate);
            }

            $currentDate->addDay();
        }

        // Calculate month summary
        $summary = $this->calculateMonthSummary($result);

        return response()->json([
            'success' => true,
            'data' => [
                'calculations' => $result,
                'calculation_status' => $status,
                'is_recalculating' => $isRecalculating,
                'month_summary' => $summary,
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/cycle/status",
     *     summary="Get calculation status",
     *     description="Check if cycle calculations are being processed",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Status retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="status", type="string", example="completed"),
     *                 @OA\Property(property="status_label", type="string", example="Completed"),
     *                 @OA\Property(property="is_processing", type="boolean", example=false),
     *                 @OA\Property(property="version", type="integer", example=1),
     *                 @OA\Property(property="started_at", type="string", format="date-time", nullable=true),
     *                 @OA\Property(property="completed_at", type="string", format="date-time", nullable=true)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function status(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $profile = $user->profile;

        if (!$profile) {
            return response()->json([
                'success' => true,
                'data' => [
                    'status' => CalculationStatus::PENDING->value,
                    'status_label' => CalculationStatus::PENDING->label($locale),
                    'is_processing' => false,
                    'version' => 0,
                    'started_at' => null,
                    'completed_at' => null,
                ],
            ]);
        }

        $statusEnum = CalculationStatus::tryFrom($profile->calculation_status) ?? CalculationStatus::PENDING;

        return response()->json([
            'success' => true,
            'data' => [
                'status' => $statusEnum->value,
                'status_label' => $statusEnum->label($locale),
                'is_processing' => $statusEnum === CalculationStatus::PROCESSING,
                'version' => $profile->calculation_version ?? 0,
                'started_at' => $profile->calculation_started_at,
                'completed_at' => $profile->calculation_completed_at,
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/cycle/recalculate",
     *     summary="Trigger recalculation",
     *     description="Manually trigger a recalculation of cycle data (useful after daily log updates)",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Recalculation triggered successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Recalculation started"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="version", type="integer", example=2),
     *                 @OA\Property(property="status", type="string", example="processing")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=400,
     *         description="Recalculation already in progress",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="Calculation already in progress")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function recalculate(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $profile = $user->profile;

        if (!$profile) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'لطفاً اول پروفایل خود را تکمیل کنید'
                    : 'Please complete your profile first',
            ], 400);
        }

        // Check if already processing
        if ($profile->calculation_status === CalculationStatus::PROCESSING->value) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'محاسبه در حال انجام است'
                    : 'Calculation already in progress',
            ], 400);
        }

        // Increment version and dispatch job
        $newVersion = ($profile->calculation_version ?? 0) + 1;

        CalculateCycleDataJob::dispatch($user->id, $newVersion, $locale);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'محاسبه مجدد شروع شد' : 'Recalculation started',
            'data' => [
                'version' => $newVersion,
                'status' => CalculationStatus::PROCESSING->value,
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/cycle/enums",
     *     summary="Get cycle enum values",
     *     description="Retrieve all available enum values for cycle phases, subphases, and variability",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for labels (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Enum values retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="phases", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="value", type="string"),
     *                     @OA\Property(property="label", type="string"),
     *                     @OA\Property(property="description", type="string")
     *                 )),
     *                 @OA\Property(property="subphases", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="value", type="string"),
     *                     @OA\Property(property="label", type="string")
     *                 )),
     *                 @OA\Property(property="variability", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="value", type="string"),
     *                     @OA\Property(property="label", type="string")
     *                 )),
     *                 @OA\Property(property="calculation_status", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="value", type="string"),
     *                     @OA\Property(property="label", type="string")
     *                 ))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function enums(Request $request): JsonResponse
    {
        $locale = $request->header('Accept-Language', 'en');

        $phases = collect(CyclePhase::cases())->map(fn($p) => [
            'value' => $p->value,
            'label' => $p->label($locale),
            'description' => $p->description($locale),
        ])->values();

        $subphases = collect(CycleSubphase::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $variability = collect(CycleVariability::cases())->map(fn($v) => [
            'value' => $v->value,
            'label' => $v->label($locale),
            'uncertainty_range' => $v->uncertaintyRange(),
        ])->values();

        $status = collect(CalculationStatus::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        return response()->json([
            'success' => true,
            'data' => [
                'phases' => $phases,
                'subphases' => $subphases,
                'variability' => $variability,
                'calculation_status' => $status,
            ],
        ]);
    }

    /**
     * Get calculation for a specific date
     */
    private function getCalculationForDate($user, Carbon $date, string $locale): JsonResponse
    {
        $profile = $user->profile;
        $status = $profile?->calculation_status ?? CalculationStatus::PENDING->value;
        $isRecalculating = $status === CalculationStatus::PROCESSING->value;

        // Try to get from stored calculations first
        $calculation = CycleCalculation::where('user_id', $user->id)
            ->where('calculation_date', $date)
            ->orderBy('version', 'desc')
            ->first();

        if ($calculation) {
            return response()->json([
                'success' => true,
                'data' => [
                    'calculation' => $this->localizeCalculation($calculation->toArray(), $locale),
                    'calculation_status' => $status,
                    'is_recalculating' => $isRecalculating,
                ],
            ]);
        }

        // Calculate on-the-fly if not stored
        $engine = new HealthDataEngine($user, $locale);
        $calculationData = $engine->calculateForDate($date);

        return response()->json([
            'success' => true,
            'data' => [
                'calculation' => $calculationData,
                'calculation_status' => $status,
                'is_recalculating' => $isRecalculating,
            ],
        ]);
    }

    /**
     * Localize calculation text fields based on locale
     */
    private function localizeCalculation(array $calculation, string $locale): array
    {
        // Extract localized text from text_flags
        if (isset($calculation['text_flags']) && is_array($calculation['text_flags'])) {
            $localizedFlags = [];
            foreach ($calculation['text_flags'] as $key => $value) {
                if (is_array($value) && isset($value[$locale])) {
                    $localizedFlags[$key] = $value[$locale];
                } else {
                    $localizedFlags[$key] = $value;
                }
            }
            $calculation['text_flags'] = $localizedFlags;
        }

        // Extract localized text from daily_tips
        if (isset($calculation['daily_tips']) && is_array($calculation['daily_tips'])) {
            $localizedTips = [];
            foreach ($calculation['daily_tips'] as $tip) {
                if (is_array($tip) && isset($tip[$locale])) {
                    $localizedTips[] = [
                        'type' => $tip['type'] ?? 'general',
                        'text' => $tip[$locale],
                    ];
                } elseif (is_array($tip) && isset($tip['en'])) {
                    $localizedTips[] = [
                        'type' => $tip['type'] ?? 'general',
                        'text' => $tip['en'],
                    ];
                }
            }
            $calculation['daily_tips'] = $localizedTips;
        }

        return $calculation;
    }

    /**
     * Calculate summary for a month
     */
    private function calculateMonthSummary(array $calculations): array
    {
        $fertileDays = 0;
        $periodDays = 0;
        $pmsDays = 0;

        foreach ($calculations as $calc) {
            if ($calc['is_fertile_window'] ?? false) {
                $fertileDays++;
            }
            if (($calc['phase'] ?? '') === CyclePhase::MENSTRUATION->value) {
                $periodDays++;
            }
            if ($calc['is_pms_window'] ?? false) {
                $pmsDays++;
            }
        }

        return [
            'fertile_days' => $fertileDays,
            'period_days' => $periodDays,
            'pms_days' => $pmsDays,
        ];
    }
}
