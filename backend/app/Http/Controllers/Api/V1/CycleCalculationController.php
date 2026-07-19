<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\CalculationStatus;
use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\CycleVariability;
use App\Enums\OverrideType;
use App\Enums\SubscriptionType;
use App\Enums\UserGoal;
use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Jobs\CalculateCycleDataJob;
use App\Models\DailyHealthLog;
use App\Services\HealthEngine\HealthDataEngine;
use App\Services\MatrixEngine\CorrelationEngine;
use App\Services\MatrixEngine\MatrixMessageEngine;
use App\Services\MatrixEngine\NutritionSleepModule;
use App\Services\MatrixEngine\PatternRecognitionEngine;
use App\Services\MatrixEngine\TTCMatrixEngine;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class CycleCalculationController extends Controller
{
    use ResolvesLocale;

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
        $locale = $this->resolveLocale($request);
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

        // Always calculate fresh to ensure correct cycle_day values
        $engine = new HealthDataEngine($user, $locale);
        $result = [];

        $currentDate = $startDate->copy();
        while ($currentDate <= $endDate) {
            $result[] = $engine->calculateForDate($currentDate);
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
     *     description="Manually trigger a recalculation of cycle data (useful after daily log updates)",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Recalculation triggered successfully",
     *
     *         @OA\JsonContent(
     *
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
     *
     *         @OA\JsonContent(
     *
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
     *
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Enum values retrieved successfully",
     *
     *         @OA\JsonContent(
     *
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
        $locale = $this->resolveLocale($request);

        $phases = collect(CyclePhase::cases())->map(fn ($p) => [
            'value' => $p->value,
            'label' => $p->label($locale),
            'description' => $p->description($locale),
        ])->values();

        $subphases = collect(CycleSubphase::cases())->map(fn ($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $variability = collect(CycleVariability::cases())->map(fn ($v) => [
            'value' => $v->value,
            'label' => $v->label($locale),
            'uncertainty_range' => $v->uncertaintyRange(),
        ])->values();

        $status = collect(CalculationStatus::cases())->map(fn ($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $mainPhases = collect(\App\Enums\MainPhase::cases())->map(fn ($p) => [
            'value' => $p->value,
            'label' => $p->label($locale),
        ])->values();

        return response()->json([
            'success' => true,
            'data' => [
                'phases' => $phases,
                'subphases' => $subphases,
                'variability' => $variability,
                'calculation_status' => $status,
                // v1.1 engine enums (task.md §13, §26, §28–§32).
                'main_phases' => $mainPhases,
                'fertility_levels' => \App\Enums\FertilityLevel::values(),
                'resolution_sources' => \App\Enums\ResolutionSource::values(),
                'data_quality_levels' => \App\Enums\DataQualityLevel::values(),
                'warnings' => \App\Enums\CycleWarning::values(),
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/cycle/matrix-messages",
     *     summary="Get personalized matrix messages",
     *     description="Retrieve personalized messages based on cycle phase, symptoms, and user goal (TTC/Non-TTC). Includes Layer 1&2 (Base Matrix), Layer 3 (Correlation Engine), Layer 4 (Pattern Recognition), and supplementary modules (Nutrition & Sleep).",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="query",
     *         description="Date for messages (YYYY-MM-DD). Defaults to today.",
     *         required=false,
     *
     *         @OA\Schema(type="string", format="date", example="2024-12-15")
     *     ),
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for messages (en, fa)",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="fa", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Matrix messages retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="date", type="string", format="date", example="2024-12-15"),
     *                 @OA\Property(property="user_goal", type="string", example="non_ttc"),
     *                 @OA\Property(property="subscription_type", type="string", example="free"),
     *                 @OA\Property(property="cycle_info", type="object",
     *                     @OA\Property(property="phase", type="string", example="follicular"),
     *                     @OA\Property(property="subphase", type="string", example="early_follicular"),
     *                     @OA\Property(property="cycle_day", type="integer", example=8),
     *                     @OA\Property(property="is_fertile_window", type="boolean", example=false),
     *                     @OA\Property(property="is_pms_window", type="boolean", example=false)
     *                 ),
     *                 @OA\Property(property="matrix_message", type="object",
     *                     @OA\Property(property="phase", type="string", example="follicular"),
     *                     @OA\Property(property="override_type", type="string", example="normal"),
     *                     @OA\Property(property="short_message", type="string"),
     *                     @OA\Property(property="long_message", type="string"),
     *                     @OA\Property(property="action_suggestion", type="string"),
     *                     @OA\Property(property="dos", type="array", @OA\Items(type="string")),
     *                     @OA\Property(property="donts", type="array", @OA\Items(type="string"))
     *                 ),
     *                 @OA\Property(property="correlations", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="type", type="string"),
     *                     @OA\Property(property="insight_message", type="string"),
     *                     @OA\Property(property="action", type="string")
     *                 )),
     *                 @OA\Property(property="patterns", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="pattern_type", type="string"),
     *                     @OA\Property(property="alert_level", type="string"),
     *                     @OA\Property(property="message", type="string")
     *                 )),
     *                 @OA\Property(property="nutrition_sleep_tips", type="object",
     *                     @OA\Property(property="nutrition", type="object"),
     *                     @OA\Property(property="sleep", type="object"),
     *                     @OA\Property(property="exercise", type="object")
     *                 )
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
    public function matrixMessages(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'fa');
        $profile = $user->profile;

        if (! $profile || ! $profile->last_period_start) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'لطفاً ابتدا پروفایل خود را تکمیل کنید'
                    : 'Please complete your profile first',
            ], 400);
        }

        // Get target date
        $date = $request->query('date')
            ? Carbon::parse($request->query('date'))
            : Carbon::today();

        // Get cycle calculation data
        $healthEngine = new HealthDataEngine($user, $locale);
        $cycleData = $healthEngine->calculateForDate($date);

        // Check if we have valid cycle data
        if (! isset($cycleData['phase']) || ! $cycleData['phase']) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'اطلاعات سیکل موجود نیست'
                    : 'Cycle information not available',
            ], 400);
        }

        // Get phase and subphase enums
        $phase = CyclePhase::from($cycleData['phase']);
        $subphase = CycleSubphase::from($cycleData['subphase']);
        $cycleDay = $cycleData['cycle_day'];
        $isFertileWindow = $cycleData['is_fertile_window'] ?? false;
        $isPmsWindow = $cycleData['is_pms_window'] ?? false;

        // Get daily health log
        $dailyLog = DailyHealthLog::where('user_id', $user->id)
            ->whereDate('log_date', $date)
            ->first();

        // Determine user goal and subscription
        $isTTC = $profile->isTTC();
        $isPremium = $profile->isPremium();
        $userGoal = $profile->user_goal ?? UserGoal::NON_TTC->value;
        $subscriptionType = $profile->subscription_type ?? SubscriptionType::FREE->value;

        // Get matrix message based on user goal
        if ($isTTC) {
            $matrixEngine = new TTCMatrixEngine($user, $locale);
            $matrixMessage = $matrixEngine->getMatrixMessage($phase, $subphase, $cycleDay, $dailyLog, $isFertileWindow);
        } else {
            $matrixEngine = new MatrixMessageEngine($user, $locale);
            $matrixMessage = $matrixEngine->getMatrixMessage($phase, $subphase, $cycleDay, $dailyLog);
        }

        // Get correlations (Layer 3)
        $correlationEngine = new CorrelationEngine($user, $locale);
        $correlations = $correlationEngine->analyzeCorrelations($phase, $subphase, $dailyLog, $isTTC);

        // Filter correlations based on subscription
        if (! $isPremium) {
            $correlations = array_filter($correlations, fn ($c) => ! ($c['is_premium_only'] ?? false));
            $correlations = array_values($correlations);
        }

        // Get patterns (Layer 4 - Premium only)
        $patterns = [];
        if ($isPremium) {
            $patternEngine = new PatternRecognitionEngine($user, $locale);
            $patterns = $patternEngine->analyzePatterns($isTTC);
        }

        // Get nutrition and sleep tips
        $nutritionSleepModule = new NutritionSleepModule($user, $locale);
        $nutritionSleepTips = $nutritionSleepModule->getTips($phase, $subphase, $dailyLog, $isTTC);

        return response()->json([
            'success' => true,
            'data' => [
                'date' => $date->toDateString(),
                'user_goal' => $userGoal,
                'subscription_type' => $subscriptionType,
                'cycle_info' => [
                    'phase' => $phase->value,
                    'phase_label' => $phase->label($locale),
                    'subphase' => $subphase->value,
                    'subphase_label' => $subphase->label($locale),
                    'cycle_day' => $cycleDay,
                    'is_fertile_window' => $isFertileWindow,
                    'is_pms_window' => $isPmsWindow,
                    'estimated_ovulation_day' => $cycleData['estimated_ovulation_day'] ?? null,
                    'cycle_length_used' => $cycleData['cycle_length_used'] ?? null,
                ],
                'matrix_message' => $matrixMessage,
                'correlations' => $correlations,
                'patterns' => $patterns,
                'nutrition_sleep_tips' => $nutritionSleepTips,
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/cycle/matrix-enums",
     *     summary="Get matrix enum values",
     *     description="Retrieve all available enum values for user goals, subscription types, and override types",
     *     tags={"Cycle Calculation"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for labels (en, fa)",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="fa", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Matrix enum values retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="user_goals", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="value", type="string"),
     *                     @OA\Property(property="label", type="string")
     *                 )),
     *                 @OA\Property(property="subscription_types", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="value", type="string"),
     *                     @OA\Property(property="label", type="string")
     *                 )),
     *                 @OA\Property(property="override_types", type="array", @OA\Items(type="object",
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
    public function matrixEnums(Request $request): JsonResponse
    {
        $locale = $request->header('Accept-Language', 'fa');

        $userGoals = collect(UserGoal::cases())->map(fn ($g) => [
            'value' => $g->value,
            'label' => $g->label($locale),
        ])->values();

        $subscriptionTypes = collect(SubscriptionType::cases())->map(fn ($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $overrideTypes = collect(OverrideType::cases())->map(fn ($o) => [
            'value' => $o->value,
            'label' => $o->label($locale),
        ])->values();

        return response()->json([
            'success' => true,
            'data' => [
                'user_goals' => $userGoals,
                'subscription_types' => $subscriptionTypes,
                'override_types' => $overrideTypes,
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

        // Always calculate fresh to ensure correct cycle_day values
        // The stored calculations may have stale data from old calculation logic
        $engine = new HealthDataEngine($user, $locale);
        $calculationData = $engine->calculateForDate($date);

        // Render-ready spec §19 payload (daily card, predictions, three-layer values,
        // confidence) assembled alongside the raw calc. Kept under a separate key so
        // existing `calculation` consumers are unaffected.
        // Reference "today" as the Tehran calendar day (the client sends its local date),
        // so a request near midnight doesn't render the user's real today as a future day.
        $today = Carbon::parse(Carbon::now('Asia/Tehran')->toDateString());
        $cycleView = (new \App\Services\HealthEngine\CycleDayViewBuilder)
            ->build($user, $date, $today, $locale, $calculationData);

        return response()->json([
            'success' => true,
            'data' => [
                'calculation' => $calculationData,
                'cycle_view' => $cycleView,
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
