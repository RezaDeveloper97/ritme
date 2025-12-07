<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Http\Requests\Api\V1\StoreDailyHealthLogRequest;
use App\Models\DailyHealthLog;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class DailyHealthLogController extends Controller
{
    /**
     * @OA\Get(
     *     path="/health-logs",
     *     summary="Get health logs list",
     *     description="Retrieve paginated list of user's daily health logs with optional date filtering",
     *     tags={"Daily Health Log"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="from_date",
     *         in="query",
     *         description="Filter logs from this date (YYYY-MM-DD)",
     *         required=false,
     *         @OA\Schema(type="string", format="date", example="2024-12-01")
     *     ),
     *     @OA\Parameter(
     *         name="to_date",
     *         in="query",
     *         description="Filter logs until this date (YYYY-MM-DD)",
     *         required=false,
     *         @OA\Schema(type="string", format="date", example="2024-12-31")
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Health logs retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="current_page", type="integer", example=1),
     *                 @OA\Property(property="data", type="array", @OA\Items(ref="#/components/schemas/DailyHealthLog")),
     *                 @OA\Property(property="last_page", type="integer", example=1),
     *                 @OA\Property(property="per_page", type="integer", example=30),
     *                 @OA\Property(property="total", type="integer", example=5)
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
    public function index(Request $request): JsonResponse
    {
        $query = $request->user()->dailyHealthLogs()->orderBy('log_date', 'desc');

        if ($request->has('from_date')) {
            $query->whereDate('log_date', '>=', $request->from_date);
        }

        if ($request->has('to_date')) {
            $query->whereDate('log_date', '<=', $request->to_date);
        }

        $logs = $query->paginate(30);

        return response()->json([
            'success' => true,
            'data' => $logs,
        ]);
    }

    /**
     * @OA\Post(
     *     path="/health-logs",
     *     summary="Create or update daily health log",
     *     description="Create a new health log or update existing one for the specified date. Each user can have only one log per day.",
     *     tags={"Daily Health Log"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\RequestBody(
     *         required=true,
     *         @OA\JsonContent(
     *             required={"log_date"},
     *             @OA\Property(property="log_date", type="string", format="date", example="2024-12-07", description="Date of the log entry"),
     *
     *             @OA\Property(property="bleeding_intensity", type="string", enum={"low","medium","high","very_high"}, description="Bleeding intensity level"),
     *             @OA\Property(property="blood_color", type="string", enum={"bright_red","red","dark_red","brown"}, description="Blood color"),
     *             @OA\Property(property="has_clots", type="boolean", description="Presence of blood clots"),
     *             @OA\Property(property="spotting", type="boolean", description="Spotting occurrence"),
     *             @OA\Property(property="bleeding_smell", type="string", enum={"normal","slightly_unusual","strong_unpleasant"}, description="Bleeding odor"),
     *
     *             @OA\Property(property="headache_intensity", type="string", enum={"low","medium","high"}, description="Headache intensity"),
     *             @OA\Property(property="stomach_ache_intensity", type="string", enum={"low","medium","high"}, description="Stomach ache intensity"),
     *             @OA\Property(property="pelvic_pain_intensity", type="string", enum={"low","medium","high"}, description="Pelvic pain intensity"),
     *             @OA\Property(property="breast_pain_intensity", type="string", enum={"low","medium","high"}, description="Breast pain intensity"),
     *             @OA\Property(property="back_pain_intensity", type="string", enum={"low","medium","high"}, description="Back pain intensity"),
     *             @OA\Property(property="ovarian_pain_intensity", type="string", enum={"low","medium","high"}, description="Ovarian pain intensity"),
     *
     *             @OA\Property(property="nausea_intensity", type="string", enum={"low","medium","high"}, description="Nausea/vomiting intensity"),
     *             @OA\Property(property="bloating_intensity", type="string", enum={"low","medium","high"}, description="Bloating intensity"),
     *             @OA\Property(property="diarrhea", type="boolean", description="Diarrhea occurrence"),
     *             @OA\Property(property="constipation", type="boolean", description="Constipation occurrence"),
     *             @OA\Property(property="appetite_change", type="string", enum={"loss","gain","normal"}, description="Appetite change"),
     *             @OA\Property(property="food_craving", type="boolean", description="Food craving"),
     *
     *             @OA\Property(property="breast_sensitivity_intensity", type="string", enum={"low","medium","high"}, description="Breast sensitivity intensity"),
     *             @OA\Property(property="vaginal_dryness", type="boolean", description="Vaginal dryness"),
     *             @OA\Property(property="vaginal_burning", type="boolean", description="Vaginal burning sensation"),
     *             @OA\Property(property="vaginal_itching", type="boolean", description="Vaginal itching"),
     *             @OA\Property(property="vaginal_smell_change", type="boolean", description="Vaginal smell change"),
     *             @OA\Property(property="urination_change", type="string", enum={"increase","decrease","normal"}, description="Urination frequency change"),
     *             @OA\Property(property="urination_burning_intensity", type="string", enum={"low","medium","high"}, description="Urination burning intensity"),
     *
     *             @OA\Property(property="acne", type="boolean", description="Acne occurrence"),
     *             @OA\Property(property="oily_skin", type="boolean", description="Oily skin"),
     *             @OA\Property(property="hair_loss", type="boolean", description="Hair loss"),
     *             @OA\Property(property="swelling", type="boolean", description="Swelling/puffiness"),
     *
     *             @OA\Property(property="fatigue", type="boolean", description="Fatigue"),
     *             @OA\Property(property="dizziness", type="boolean", description="Dizziness"),
     *             @OA\Property(property="hot_flashes", type="boolean", description="Hot flashes"),
     *             @OA\Property(property="chills", type="boolean", description="Chills"),
     *
     *             @OA\Property(property="moods", type="array", @OA\Items(type="string", enum={"happy","calm","angry","anxious","sad","frustrated","sensitive","bored"}), description="Mood states (can select multiple)"),
     *
     *             @OA\Property(property="sleep_duration", type="string", enum={"0_3","3_6","6_9","9_plus"}, description="Sleep duration in hours"),
     *             @OA\Property(property="sleep_quality", type="string", enum={"good","medium","bad"}, description="Sleep quality"),
     *
     *             @OA\Property(property="sexual_activities", type="array", @OA\Items(type="string", enum={"high_desire","protected_intercourse","unprotected_intercourse","no_desire","dryness","burning","pain_during_intercourse","bleeding_after_intercourse","lubricant_use"}), description="Sexual activity indicators (can select multiple)"),
     *
     *             @OA\Property(property="weight", type="number", format="float", example=65.5, description="Body weight in kg"),
     *             @OA\Property(property="basal_body_temperature", type="number", format="float", example=36.5, description="Basal body temperature (BBT) in Celsius"),
     *
     *             @OA\Property(property="discharge_color", type="string", example="white", description="Vaginal discharge color"),
     *             @OA\Property(property="discharge_texture", type="string", enum={"watery","creamy","egg_white","thick"}, description="Vaginal discharge texture"),
     *             @OA\Property(property="discharge_amount", type="string", enum={"low","medium","high"}, description="Vaginal discharge amount"),
     *             @OA\Property(property="discharge_smell", type="string", enum={"normal","slightly_unusual","strong_unpleasant"}, description="Vaginal discharge smell"),
     *             @OA\Property(property="discharge_itching", type="boolean", description="Discharge-related itching"),
     *             @OA\Property(property="discharge_burning", type="boolean", description="Discharge-related burning"),
     *
     *             @OA\Property(property="frequent_urination", type="boolean", description="Frequent urination"),
     *
     *             @OA\Property(property="medications", type="object", description="Medications and supplements taken",
     *                 @OA\Property(property="painkillers", type="string", example="Ibuprofen"),
     *                 @OA\Property(property="hormonal_pills", type="string", example="LD"),
     *                 @OA\Property(property="antibiotics", type="string", example="Amoxicillin"),
     *                 @OA\Property(property="supplements", type="string", example="Iron, Vitamin D")
     *             ),
     *
     *             @OA\Property(property="notes", type="string", example="Feeling good today", description="Personal notes")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=201,
     *         description="Health log created successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Health log created successfully"),
     *             @OA\Property(property="data", ref="#/components/schemas/DailyHealthLog")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Health log updated successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Health log updated successfully"),
     *             @OA\Property(property="data", ref="#/components/schemas/DailyHealthLog")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=422,
     *         description="Validation error"
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function store(StoreDailyHealthLogRequest $request): JsonResponse
    {
        $validated = $request->validated();

        $log = DailyHealthLog::updateOrCreate(
            [
                'user_id' => $request->user()->id,
                'log_date' => $validated['log_date'],
            ],
            $validated
        );

        $isNew = $log->wasRecentlyCreated;

        return response()->json([
            'success' => true,
            'message' => $isNew ? 'Health log created successfully' : 'Health log updated successfully',
            'data' => $log,
        ], $isNew ? 201 : 200);
    }

    /**
     * @OA\Get(
     *     path="/health-logs/{date}",
     *     summary="Get health log by date",
     *     description="Retrieve a specific health log entry for the given date",
     *     tags={"Daily Health Log"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="path",
     *         description="Date of the log (YYYY-MM-DD)",
     *         required=true,
     *         @OA\Schema(type="string", format="date", example="2024-12-07")
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Health log retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", ref="#/components/schemas/DailyHealthLog")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=404,
     *         description="Health log not found",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="No health log found for this date")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function show(Request $request, string $date): JsonResponse
    {
        $log = $request->user()->dailyHealthLogs()
            ->whereDate('log_date', $date)
            ->first();

        if (!$log) {
            return response()->json([
                'success' => false,
                'message' => 'No health log found for this date',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'data' => $log,
        ]);
    }

    /**
     * @OA\Delete(
     *     path="/health-logs/{date}",
     *     summary="Delete health log",
     *     description="Delete a health log entry for the specified date",
     *     tags={"Daily Health Log"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="path",
     *         description="Date of the log to delete (YYYY-MM-DD)",
     *         required=true,
     *         @OA\Schema(type="string", format="date", example="2024-12-07")
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Health log deleted successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Health log deleted successfully")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=404,
     *         description="Health log not found",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="No health log found for this date")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function destroy(Request $request, string $date): JsonResponse
    {
        $log = $request->user()->dailyHealthLogs()
            ->whereDate('log_date', $date)
            ->first();

        if (!$log) {
            return response()->json([
                'success' => false,
                'message' => 'No health log found for this date',
            ], 404);
        }

        $log->delete();

        return response()->json([
            'success' => true,
            'message' => 'Health log deleted successfully',
        ]);
    }

    /**
     * @OA\Get(
     *     path="/health-logs/enums",
     *     summary="Get enum values for forms",
     *     description="Retrieve all available enum values for dropdown fields in the health log form",
     *     tags={"Daily Health Log"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Enum values retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="bleeding_intensity", type="array", @OA\Items(type="string"), example={"low","medium","high","very_high"}),
     *                 @OA\Property(property="blood_color", type="array", @OA\Items(type="string"), example={"bright_red","red","dark_red","brown"}),
     *                 @OA\Property(property="bleeding_smell", type="array", @OA\Items(type="string"), example={"normal","slightly_unusual","strong_unpleasant"}),
     *                 @OA\Property(property="pain_intensity", type="array", @OA\Items(type="string"), example={"low","medium","high"}),
     *                 @OA\Property(property="appetite_change", type="array", @OA\Items(type="string"), example={"loss","gain","normal"}),
     *                 @OA\Property(property="urination_change", type="array", @OA\Items(type="string"), example={"increase","decrease","normal"}),
     *                 @OA\Property(property="moods", type="array", @OA\Items(type="string"), example={"happy","calm","angry","anxious","sad","frustrated","sensitive","bored"}),
     *                 @OA\Property(property="sleep_duration", type="array", @OA\Items(type="string"), example={"0_3","3_6","6_9","9_plus"}),
     *                 @OA\Property(property="sleep_quality", type="array", @OA\Items(type="string"), example={"good","medium","bad"}),
     *                 @OA\Property(property="sexual_activities", type="array", @OA\Items(type="string"), example={"high_desire","protected_intercourse","unprotected_intercourse","no_desire","dryness","burning","pain_during_intercourse","bleeding_after_intercourse","lubricant_use"}),
     *                 @OA\Property(property="discharge_texture", type="array", @OA\Items(type="string"), example={"watery","creamy","egg_white","thick"}),
     *                 @OA\Property(property="discharge_amount", type="array", @OA\Items(type="string"), example={"low","medium","high"}),
     *                 @OA\Property(property="discharge_smell", type="array", @OA\Items(type="string"), example={"normal","slightly_unusual","strong_unpleasant"})
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
    public function enums(): JsonResponse
    {
        return response()->json([
            'success' => true,
            'data' => DailyHealthLog::getEnumValues(),
        ]);
    }
}
