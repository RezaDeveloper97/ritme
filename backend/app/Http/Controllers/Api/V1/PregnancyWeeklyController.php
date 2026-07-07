<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\FetalMovementStatus;
use App\Enums\MentalHealthStatus;
use App\Enums\SwellingLocation;
use App\Enums\SymptomSeverity;
use App\Http\Controllers\Controller;
use App\Http\Requests\Api\V1\StorePregnancyFetalMovementRequest;
use App\Http\Requests\Api\V1\StorePregnancyWeeklyLogRequest;
use App\Models\PregnancyFetalMovement;
use App\Models\PregnancyWeeklyContent;
use App\Models\PregnancyWeeklyLog;
use App\Services\PregnancyEngine\PregnancyAlertService;
use App\Services\PregnancyEngine\PregnancyCalculationService;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class PregnancyWeeklyController extends Controller
{
    /**
     * @OA\Get(
     *     path="/pregnancy/weekly",
     *     summary="Get weekly logs",
     *     description="Retrieve all weekly monitoring logs for the authenticated user",
     *     tags={"Pregnancy Weekly"},
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
     *         description="Weekly logs retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="logs", type="array", @OA\Items(ref="#/components/schemas/PregnancyWeeklyLog")),
     *                 @OA\Property(property="count", type="integer", example=10)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function index(Request $request): JsonResponse
    {
        $user = $request->user();
        $logs = PregnancyWeeklyLog::where('user_id', $user->id)
            ->orderBy('pregnancy_week', 'desc')
            ->get();

        return response()->json([
            'success' => true,
            'data' => [
                'logs' => $logs,
                'count' => $logs->count(),
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/pregnancy/weekly",
     *     summary="Create or update weekly log",
     *     description="Create or update a weekly monitoring log including weight, blood pressure, mental health, etc.",
     *     tags={"Pregnancy Weekly"},
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
     *     @OA\RequestBody(
     *         required=true,
     *         @OA\JsonContent(
     *             required={"log_date", "pregnancy_week"},
     *             @OA\Property(property="log_date", type="string", format="date", example="2024-12-14"),
     *             @OA\Property(property="pregnancy_week", type="integer", minimum=1, maximum=42, example=20),
     *             @OA\Property(property="weight", type="number", format="float", minimum=30, maximum=200),
     *             @OA\Property(property="has_swelling", type="boolean"),
     *             @OA\Property(property="swelling_locations", type="array", @OA\Items(type="string", enum={"feet","hands","face"})),
     *             @OA\Property(property="has_shortness_of_breath", type="boolean"),
     *             @OA\Property(property="has_blood_pressure_device", type="boolean"),
     *             @OA\Property(property="systolic_pressure", type="integer", minimum=60, maximum=250),
     *             @OA\Property(property="diastolic_pressure", type="integer", minimum=40, maximum=150),
     *             @OA\Property(property="fasting_blood_sugar", type="number", format="float"),
     *             @OA\Property(property="post_meal_blood_sugar", type="number", format="float"),
     *             @OA\Property(property="overall_mood", type="string", enum={"good","moderate","poor"}),
     *             @OA\Property(property="has_anxiety", type="boolean"),
     *             @OA\Property(property="anxiety_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_mood_swings", type="boolean"),
     *             @OA\Property(property="mood_swings_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_depression_feelings", type="boolean"),
     *             @OA\Property(property="depression_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="notes", type="string", maxLength=2000)
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=201,
     *         description="Weekly log created/updated successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Weekly log saved successfully"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="log", ref="#/components/schemas/PregnancyWeeklyLog"),
     *                 @OA\Property(property="alerts", type="array", @OA\Items(ref="#/components/schemas/PregnancyAlert"))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=422, description="Validation error")
     * )
     */
    public function store(StorePregnancyWeeklyLogRequest $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $validated = $request->validated();

        // Create or update log (one per week)
        $log = PregnancyWeeklyLog::updateOrCreate(
            [
                'user_id' => $user->id,
                'pregnancy_week' => $validated['pregnancy_week'],
            ],
            $validated
        );

        // Process alerts
        $alertService = new PregnancyAlertService($user, $locale);
        $alerts = $alertService->processWeeklyLog($log);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'گزارش هفتگی با موفقیت ذخیره شد' : 'Weekly log saved successfully',
            'data' => [
                'log' => $log,
                'alerts' => $alerts,
            ],
        ], 201);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/weekly/{week}",
     *     summary="Get weekly log for specific week",
     *     description="Retrieve weekly log for a specific pregnancy week",
     *     tags={"Pregnancy Weekly"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="week",
     *         in="path",
     *         description="Pregnancy week (1-42)",
     *         required=true,
     *         @OA\Schema(type="integer", minimum=1, maximum=42, example=20)
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
     *         description="Weekly log retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="log", ref="#/components/schemas/PregnancyWeeklyLog")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Weekly log not found")
     * )
     */
    public function show(Request $request, int $week): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        if ($week < 1 || $week > 42) {
            return response()->json([
                'success' => false,
                'message' => 'Invalid week number. Must be between 1 and 42.',
            ], 422);
        }

        $log = PregnancyWeeklyLog::where('user_id', $user->id)
            ->where('pregnancy_week', $week)
            ->first();

        if (!$log) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'گزارش هفتگی یافت نشد' : 'Weekly log not found',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'data' => [
                'log' => $log,
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/pregnancy/fetal-movement",
     *     summary="Log fetal movement",
     *     description="Create or update daily fetal movement log. Active from week 18, required from week 24.",
     *     tags={"Pregnancy Weekly"},
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
     *     @OA\RequestBody(
     *         required=true,
     *         @OA\JsonContent(
     *             required={"log_date", "pregnancy_week", "movement_status"},
     *             @OA\Property(property="log_date", type="string", format="date", example="2024-12-14"),
     *             @OA\Property(property="pregnancy_week", type="integer", minimum=1, maximum=42, example=24),
     *             @OA\Property(property="movement_status", type="string", enum={"not_felt_yet","felt","normal","reduced","increased","none"}),
     *             @OA\Property(property="movement_count", type="integer", minimum=0, maximum=100),
     *             @OA\Property(property="first_movement_time", type="string", format="time", example="09:30"),
     *             @OA\Property(property="last_movement_time", type="string", format="time", example="21:00"),
     *             @OA\Property(property="notes", type="string", maxLength=2000)
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=201,
     *         description="Fetal movement logged successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Fetal movement logged successfully"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="log", ref="#/components/schemas/PregnancyFetalMovement"),
     *                 @OA\Property(property="alerts", type="array", @OA\Items(ref="#/components/schemas/PregnancyAlert"))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=422, description="Validation error")
     * )
     */
    public function storeFetalMovement(StorePregnancyFetalMovementRequest $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $validated = $request->validated();

        // Create or update log
        $movement = PregnancyFetalMovement::updateOrCreate(
            [
                'user_id' => $user->id,
                'log_date' => $validated['log_date'],
            ],
            $validated
        );

        // Update pregnancy profile if first movement detected
        if (in_array($validated['movement_status'], ['felt', 'normal', 'increased'])) {
            $profile = $user->pregnancyProfile;
            if ($profile && !$profile->fetal_movement_felt) {
                $profile->update([
                    'fetal_movement_felt' => true,
                    'first_fetal_movement_date' => $profile->first_fetal_movement_date ?? $validated['log_date'],
                ]);
            }
        }

        // Process alerts
        $alertService = new PregnancyAlertService($user, $locale);
        $alerts = $alertService->processFetalMovement($movement);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'حرکات جنین ثبت شد' : 'Fetal movement logged successfully',
            'data' => [
                'log' => $movement,
                'alerts' => $alerts,
            ],
        ], 201);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/fetal-movement",
     *     summary="Get fetal movement logs",
     *     description="Retrieve all fetal movement logs",
     *     tags={"Pregnancy Weekly"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for text responses (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *     @OA\Parameter(
     *         name="from",
     *         in="query",
     *         description="Start date (YYYY-MM-DD)",
     *         required=false,
     *         @OA\Schema(type="string", format="date")
     *     ),
     *     @OA\Parameter(
     *         name="to",
     *         in="query",
     *         description="End date (YYYY-MM-DD)",
     *         required=false,
     *         @OA\Schema(type="string", format="date")
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Fetal movement logs retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="logs", type="array", @OA\Items(ref="#/components/schemas/PregnancyFetalMovement")),
     *                 @OA\Property(property="count", type="integer", example=10)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function indexFetalMovement(Request $request): JsonResponse
    {
        $user = $request->user();
        $query = PregnancyFetalMovement::where('user_id', $user->id)
            ->orderBy('log_date', 'desc');

        if ($request->has('from')) {
            $query->where('log_date', '>=', $request->get('from'));
        }
        if ($request->has('to')) {
            $query->where('log_date', '<=', $request->get('to'));
        }

        $logs = $query->get();

        return response()->json([
            'success' => true,
            'data' => [
                'logs' => $logs,
                'count' => $logs->count(),
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/content/{week}",
     *     summary="Get weekly content",
     *     description="Retrieve educational content for a specific pregnancy week",
     *     tags={"Pregnancy Weekly"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="week",
     *         in="path",
     *         description="Pregnancy week (1-40)",
     *         required=true,
     *         @OA\Schema(type="integer", minimum=1, maximum=40, example=12)
     *     ),
     *     @OA\Parameter(
     *         name="locale",
     *         in="query",
     *         description="Language for content (en, fa). Takes precedence over Accept-Language.",
     *         required=false,
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for content (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="en", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Weekly content retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="week", type="integer", example=12),
     *                 @OA\Property(property="content", ref="#/components/schemas/PregnancyWeeklyContent")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Content not found for this week")
     * )
     */
    public function content(Request $request, int $week): JsonResponse
    {
        // Browsers forbid JS from overriding the Accept-Language header, so the
        // SPA passes the active app locale as an explicit `?locale=` query param
        // which takes precedence over the header for content resolution.
        $locale = $request->query('locale', $request->header('Accept-Language', 'en'));
        $locale = in_array($locale, ['fa', 'en'], true) ? $locale : 'en';

        if ($week < 1 || $week > 40) {
            return response()->json([
                'success' => false,
                'message' => 'Invalid week number. Must be between 1 and 40.',
            ], 422);
        }

        $content = PregnancyWeeklyContent::getByWeek($week);

        if (!$content) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'محتوای این هفته یافت نشد' : 'Content not found for this week',
            ], 404);
        }

        // Localize content
        $localizedContent = [
            'week_number' => $content->week_number,
            'fetal_development' => $content->getLocalizedContent('fetal_development', $locale),
            'mother_body_changes' => $content->getLocalizedContent('mother_body_changes', $locale),
            'dos_and_donts' => $content->getLocalizedContent('dos_and_donts', $locale),
            'care_plan' => $content->getLocalizedContent('care_plan', $locale),
            'body_adaptation' => $content->getLocalizedContent('body_adaptation', $locale),
            'emotional_status' => $content->getLocalizedContent('emotional_status', $locale),
            'key_nutrition' => $content->getLocalizedContent('key_nutrition', $locale),
            'physical_activity' => $content->getLocalizedContent('physical_activity', $locale),
            'tests_and_checkups' => $content->getLocalizedContent('tests_and_checkups', $locale),
            'faq' => $content->getLocalizedContent('faq', $locale),
        ];

        return response()->json([
            'success' => true,
            'data' => [
                'week' => $week,
                'content' => $localizedContent,
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/weekly/enums",
     *     summary="Get weekly enum values",
     *     description="Retrieve all available enum values for weekly monitoring fields",
     *     tags={"Pregnancy Weekly"},
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
     *             @OA\Property(property="data", type="object")
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function enums(Request $request): JsonResponse
    {
        $locale = $request->header('Accept-Language', 'en');

        $swellingLocations = collect(SwellingLocation::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $moodStatus = collect(MentalHealthStatus::cases())->map(fn($m) => [
            'value' => $m->value,
            'label' => $m->label($locale),
        ])->values();

        $severity = collect(SymptomSeverity::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $fetalMovementStatus = collect(FetalMovementStatus::cases())->map(fn($f) => [
            'value' => $f->value,
            'label' => $f->label($locale),
        ])->values();

        return response()->json([
            'success' => true,
            'data' => [
                'swelling_locations' => $swellingLocations,
                'overall_mood' => $moodStatus,
                'severity' => $severity,
                'fetal_movement_status' => $fetalMovementStatus,
            ],
        ]);
    }
}
