<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\CalculationStatus;
use App\Enums\CyclePhase;
use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Services\HealthEngine\CycleDayViewBuilder;
use App\Services\HealthEngine\CycleEngineCache;
use App\Services\HealthEngine\DailyTipLocalizer;
use App\Services\HealthEngine\HealthDataEngine;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * The per-day engine result (`HealthDataEngine::calculateForDate()`), computed live on
 * every request. It used to be documented on the `CycleCalculation` model, which is
 * gone with its dead `cycle_calculations` storage (T-M1-13).
 *
 * @OA\Schema(
 *     schema="CycleCalculation",
 *     type="object",
 *
 *     @OA\Property(property="calculation_date", type="string", format="date", example="2024-12-07"),
 *     @OA\Property(property="cycle_day", type="integer", example=14),
 *     @OA\Property(property="phase", type="string", example="ovulation"),
 *     @OA\Property(property="subphase", type="string", example="ovulation_window"),
 *     @OA\Property(property="estimated_ovulation_day", type="integer", example=14),
 *     @OA\Property(property="cycle_length_used", type="integer", example=28),
 *     @OA\Property(property="is_fertile_window", type="boolean", example=true),
 *     @OA\Property(property="is_pms_window", type="boolean", example=false),
 *     @OA\Property(property="is_period_tomorrow", type="boolean", example=false),
 *     @OA\Property(property="is_luteal_spotting", type="boolean", example=false),
 *     @OA\Property(property="cycle_score", type="number", format="float", example=0.70),
 *     @OA\Property(property="age_factor", type="number", format="float", example=0.85),
 *     @OA\Property(property="base_probability", type="number", format="float", example=0.30),
 *     @OA\Property(property="symptom_score", type="number", format="float", example=1.30),
 *     @OA\Property(property="final_probability", type="number", format="float", example=19.89),
 *     @OA\Property(property="cycle_variability", type="string", example="regular"),
 *     @OA\Property(property="uncertainty_range", type="integer", example=1),
 *     @OA\Property(property="text_flags", type="object"),
 *     @OA\Property(property="daily_tips", type="array", description="Localized daily recommendations (admin-managed)",
 *
 *         @OA\Items(type="object",
 *
 *             @OA\Property(property="type", type="string", example="nutrition"),
 *             @OA\Property(property="title", type="string", example="تغذیه"),
 *             @OA\Property(property="icon", type="string", example="apple"),
 *             @OA\Property(property="text", type="string")
 *         )
 *     )
 * )
 */
class CycleCalculationController extends Controller
{
    use ResolvesLocale;

    /** `?view=` values accepted by `/cycle/month`. */
    private const MONTH_VIEWS = ['full', 'calendar'];

    public function __construct(private readonly CycleEngineCache $engineCache) {}

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
     *
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Today's cycle calculation retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="calculation", ref="#/components/schemas/CycleCalculation"),
     *                 @OA\Property(property="cycle_view", type="object", description="Render-ready daily payload (spec §19): daily_card, predictions, profile/calculated/effective_values, data_status, fertility_level, data_quality"),
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
        $locale = $this->resolveLocale($request);
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
     *
     *         @OA\Schema(type="string", format="date", example="2024-12-15")
     *     ),
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for text responses (en, fa)",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Cycle calculation retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="calculation", ref="#/components/schemas/CycleCalculation"),
     *                 @OA\Property(property="cycle_view", type="object", description="Render-ready daily payload (spec §19): daily_card, predictions, profile/calculated/effective_values, data_status, fertility_level, data_quality"),
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
        $locale = $this->resolveLocale($request);

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
     *
     *         @OA\Schema(type="integer", example=2024)
     *     ),
     *
     *     @OA\Parameter(
     *         name="month",
     *         in="path",
     *         description="Month (1-12)",
     *         required=true,
     *
     *         @OA\Schema(type="integer", example=12)
     *     ),
     *
     *     @OA\Parameter(
     *         name="view",
     *         in="query",
     *         description="Opt-in payload size. `full` (default, unchanged — the Android app reads it) returns every engine field per day, including the bilingual `text_flags`, `daily_tips` and `source_*` snapshots. `calendar` returns per day only calculation_date, cycle_day, phase, subphase, estimated_ovulation_day, cycle_length_used, is_fertile_window, is_pms_window, is_period_tomorrow, final_probability and cycle_variability (~93 % smaller) and skips the tip/text work. A day without cycle data keeps the same sentinel in both views: `cycle_day` null.",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="full", enum={"full","calendar"})
     *     ),
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for text responses (en, fa)",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Monthly cycle calculations retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="calculations", type="array", description="One entry per day of the month. With view=calendar each entry has only the calendar fields listed on the `view` parameter.", @OA\Items(ref="#/components/schemas/CycleCalculation")),
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
     *     ),
     *     @OA\Response(
     *         response=422,
     *         description="Invalid month or unknown view"
     *     )
     * )
     */
    public function month(Request $request, int $year, int $month): JsonResponse
    {
        $user = $request->user();
        $locale = $this->resolveLocale($request);
        $profile = $user->profile;

        if ($month < 1 || $month > 12) {
            return response()->json([
                'success' => false,
                'message' => 'Invalid month. Must be between 1 and 12.',
            ], 422);
        }

        $view = $request->query('view', 'full');
        if (! in_array($view, self::MONTH_VIEWS, true)) {
            return response()->json([
                'success' => false,
                'message' => 'Invalid view. Must be one of: '.implode(', ', self::MONTH_VIEWS).'.',
            ], 422);
        }
        $withContent = $view === 'full';

        $startDate = Carbon::createFromDate($year, $month, 1)->startOfMonth();
        $endDate = $startDate->copy()->endOfMonth();

        // Get status info
        $status = $profile?->calculation_status ?? CalculationStatus::PENDING->value;
        $isRecalculating = $status === CalculationStatus::PROCESSING->value;

        $engine = new HealthDataEngine($user, $locale);
        // Warm the daily-log cache for the whole month in one query so the loop
        // below doesn't fire a per-day SELECT (was 31 queries → now 1).
        $engine->preloadDailyLogs($startDate, $endDate);

        $monthData = $this->engineCache->remember(
            $engine,
            sprintf('month:%04d-%02d:%s', $year, $month, $view),
            $startDate,
            $endDate,
            $withContent,
            function () use ($engine, $startDate, $endDate, $withContent): array {
                $result = [];
                $currentDate = $startDate->copy();
                while ($currentDate <= $endDate) {
                    $result[] = $engine->calculateForDate($currentDate, $withContent);
                    $currentDate->addDay();
                }

                return [
                    'calculations' => $result,
                    'month_summary' => $this->calculateMonthSummary($result),
                ];
            },
        );

        return response()->json([
            'success' => true,
            'data' => [
                'calculations' => $monthData['calculations'],
                'calculation_status' => $status,
                'is_recalculating' => $isRecalculating,
                'month_summary' => $monthData['month_summary'],
            ],
        ], 200, [], JSON_UNESCAPED_UNICODE);
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
     *
     *         @OA\JsonContent(
     *
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
        $locale = $this->resolveLocale($request);
        $profile = $user->profile;

        if (! $profile) {
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
     *     description="Records a recalculation of the cycle data. Every cycle endpoint computes live from the user's inputs, so the recalculation completes synchronously: the version is bumped and the status is `completed` in the response (and in /cycle/status) right away. Kept for clients that call it after daily log updates.",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Recalculation completed",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Recalculation completed"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="version", type="integer", example=2),
     *                 @OA\Property(property="status", type="string", example="completed")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=400,
     *         description="Profile not complete",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="Please complete your profile first")
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
        $locale = $this->resolveLocale($request);
        $profile = $user->profile;

        if (! $profile) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'لطفاً اول پروفایل خود را تکمیل کنید'
                    : 'Please complete your profile first',
            ], 400);
        }

        $newVersion = $profile->markRecalculated();

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'محاسبه مجدد انجام شد' : 'Recalculation completed',
            'data' => [
                'version' => $newVersion,
                'status' => CalculationStatus::COMPLETED->value,
            ],
        ]);
    }

    /**
     * Get calculation for a specific date
     * Always calculates fresh to ensure correct cycle_day values
     */
    private function getCalculationForDate($user, Carbon $date, string $locale): JsonResponse
    {
        $profile = $user->profile;
        $status = $profile?->calculation_status ?? CalculationStatus::PENDING->value;
        $isRecalculating = $status === CalculationStatus::PROCESSING->value;

        // Reference "today" as the Tehran calendar day (the client sends its local date),
        // so a request near midnight doesn't render the user's real today as a future day.
        $today = Carbon::parse(Carbon::now('Asia/Tehran')->toDateString());

        // Always calculated from the live inputs (the cache key hashes them).
        $engine = new HealthDataEngine($user, $locale);

        $day = $this->engineCache->remember(
            $engine,
            'day:'.$date->toDateTimeString(),
            $date,
            $date,
            true,
            function () use ($engine, $user, $date, $today, $locale): array {
                // The engine stores bilingual `{en, fa}` blobs in `daily_tips`/`text_flags`;
                // collapse them to the request locale so clients get render-ready strings
                // (`daily_tips` becomes a list of `{type, title, icon, text}`) instead of raw
                // dictionaries.
                $calculationData = $this->localizeCalculation($engine->calculateForDate($date), $locale);

                // Render-ready spec §19 payload (daily card, predictions, three-layer values,
                // confidence) assembled alongside the raw calc. Kept under a separate key so
                // existing `calculation` consumers are unaffected. Reuses the engine's
                // cycle_histories load instead of reading the table a second time.
                $cycleView = (new CycleDayViewBuilder)
                    ->build($user, $date, $today, $locale, $calculationData, $engine->cycleHistories());

                return ['calculation' => $calculationData, 'cycle_view' => $cycleView];
            },
        );

        return response()->json([
            'success' => true,
            'data' => [
                'calculation' => $day['calculation'],
                'cycle_view' => $day['cycle_view'],
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
            $calculation['daily_tips'] = (new DailyTipLocalizer)->localize($calculation['daily_tips'], $locale);
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
