<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\SymptomSeverity;
use App\Http\Controllers\Controller;
use App\Http\Requests\Api\V1\StorePregnancySymptomLogRequest;
use App\Models\PregnancySymptomLog;
use App\Services\PregnancyEngine\PregnancyAlertService;
use App\Services\PregnancyEngine\PregnancyCalculationService;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class PregnancySymptomController extends Controller
{
    /**
     * @OA\Get(
     *     path="/pregnancy/symptoms",
     *     summary="Get pregnancy symptom logs",
     *     description="Retrieve all pregnancy symptom logs for the authenticated user",
     *     tags={"Pregnancy Symptoms"},
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
     *         description="Symptom logs retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="logs", type="array", @OA\Items(ref="#/components/schemas/PregnancySymptomLog")),
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
        $query = PregnancySymptomLog::where('user_id', $user->id)
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
     * @OA\Post(
     *     path="/pregnancy/symptoms",
     *     summary="Create or update pregnancy symptom log",
     *     description="Create or update a daily symptom log for pregnancy tracking. If a log already exists for the date, it will be updated.",
     *     tags={"Pregnancy Symptoms"},
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
     *             required={"log_date"},
     *             @OA\Property(property="log_date", type="string", format="date", example="2024-12-14"),
     *             @OA\Property(property="has_nausea", type="boolean"),
     *             @OA\Property(property="nausea_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_vomiting", type="boolean"),
     *             @OA\Property(property="vomiting_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_fatigue", type="boolean"),
     *             @OA\Property(property="fatigue_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_headache", type="boolean"),
     *             @OA\Property(property="headache_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_dizziness", type="boolean"),
     *             @OA\Property(property="dizziness_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_breast_pain", type="boolean"),
     *             @OA\Property(property="breast_pain_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_lower_abdominal_pain", type="boolean"),
     *             @OA\Property(property="lower_abdominal_pain_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_cramping", type="boolean"),
     *             @OA\Property(property="cramping_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_back_pain", type="boolean"),
     *             @OA\Property(property="back_pain_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_pelvic_pressure", type="boolean"),
     *             @OA\Property(property="pelvic_pressure_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_spotting", type="boolean"),
     *             @OA\Property(property="spotting_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_bleeding", type="boolean"),
     *             @OA\Property(property="bleeding_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_fluid_leakage", type="boolean"),
     *             @OA\Property(property="fluid_leakage_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="has_severe_sudden_pain", type="boolean"),
     *             @OA\Property(property="severe_sudden_pain_severity", type="string", enum={"mild","moderate","severe"}),
     *             @OA\Property(property="notes", type="string", maxLength=2000)
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=201,
     *         description="Symptom log created/updated successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Symptom log saved successfully"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="log", ref="#/components/schemas/PregnancySymptomLog"),
     *                 @OA\Property(property="alerts", type="array", @OA\Items(ref="#/components/schemas/PregnancyAlert"))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=422, description="Validation error")
     * )
     */
    public function store(StorePregnancySymptomLogRequest $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $validated = $request->validated();

        // Create or update log
        $log = PregnancySymptomLog::updateOrCreate(
            [
                'user_id' => $user->id,
                'log_date' => $validated['log_date'],
            ],
            $validated
        );

        // Process alerts
        $alertService = new PregnancyAlertService($user, $locale);
        $alerts = $alertService->processSymptomLog($log);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'علائم با موفقیت ذخیره شد' : 'Symptom log saved successfully',
            'data' => [
                'log' => $log,
                'alerts' => $alerts,
            ],
        ], 201);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/symptoms/{date}",
     *     summary="Get symptom log for specific date",
     *     description="Retrieve symptom log for a specific date",
     *     tags={"Pregnancy Symptoms"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="path",
     *         description="Date (YYYY-MM-DD)",
     *         required=true,
     *         @OA\Schema(type="string", format="date", example="2024-12-14")
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
     *         description="Symptom log retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="log", ref="#/components/schemas/PregnancySymptomLog"),
     *                 @OA\Property(property="pregnancy_week", type="integer", example=20)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Symptom log not found")
     * )
     */
    public function show(Request $request, string $date): JsonResponse
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

        $log = PregnancySymptomLog::where('user_id', $user->id)
            ->whereDate('log_date', $targetDate)
            ->first();

        if (!$log) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'گزارش علائم یافت نشد' : 'Symptom log not found',
            ], 404);
        }

        $calculationService = new PregnancyCalculationService($user, $locale);
        $currentWeek = $calculationService->getCurrentWeek();

        return response()->json([
            'success' => true,
            'data' => [
                'log' => $log,
                'pregnancy_week' => $currentWeek,
            ],
        ]);
    }

    /**
     * @OA\Delete(
     *     path="/pregnancy/symptoms/{date}",
     *     summary="Delete symptom log",
     *     description="Delete a symptom log for a specific date",
     *     tags={"Pregnancy Symptoms"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="path",
     *         description="Date (YYYY-MM-DD)",
     *         required=true,
     *         @OA\Schema(type="string", format="date", example="2024-12-14")
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
     *         description="Symptom log deleted successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Symptom log deleted successfully")
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Symptom log not found")
     * )
     */
    public function destroy(Request $request, string $date): JsonResponse
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

        $deleted = PregnancySymptomLog::where('user_id', $user->id)
            ->whereDate('log_date', $targetDate)
            ->delete();

        if (!$deleted) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'گزارش علائم یافت نشد' : 'Symptom log not found',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'گزارش علائم حذف شد' : 'Symptom log deleted successfully',
        ]);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/symptoms/enums",
     *     summary="Get symptom enum values",
     *     description="Retrieve all available enum values for symptom severities",
     *     tags={"Pregnancy Symptoms"},
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
     *                 @OA\Property(property="severity", type="array", @OA\Items(type="object",
     *                     @OA\Property(property="value", type="string"),
     *                     @OA\Property(property="label", type="string")
     *                 ))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function enums(Request $request): JsonResponse
    {
        $locale = $request->header('Accept-Language', 'en');

        $severity = collect(SymptomSeverity::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        return response()->json([
            'success' => true,
            'data' => [
                'severity' => $severity,
            ],
        ]);
    }
}
