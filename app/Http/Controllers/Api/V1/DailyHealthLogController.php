<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Http\Requests\Api\V1\StoreDailyHealthLogRequest;
use App\Models\DailyHealthLog;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * @OA\Tag(
 *     name="Daily Health Log",
 *     description="پایش سلامت روزانه"
 * )
 */
class DailyHealthLogController extends Controller
{
    /**
     * @OA\Get(
     *     path="/api/v1/health-logs",
     *     tags={"Daily Health Log"},
     *     summary="دریافت لیست لاگ‌های سلامت",
     *     security={{"bearerAuth":{}}},
     *     @OA\Parameter(
     *         name="from_date",
     *         in="query",
     *         description="از تاریخ (YYYY-MM-DD)",
     *         required=false,
     *         @OA\Schema(type="string", format="date")
     *     ),
     *     @OA\Parameter(
     *         name="to_date",
     *         in="query",
     *         description="تا تاریخ (YYYY-MM-DD)",
     *         required=false,
     *         @OA\Schema(type="string", format="date")
     *     ),
     *     @OA\Response(
     *         response=200,
     *         description="لیست لاگ‌ها",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="array", @OA\Items(type="object"))
     *         )
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
     *     path="/api/v1/health-logs",
     *     tags={"Daily Health Log"},
     *     summary="ثبت یا ویرایش لاگ سلامت روزانه",
     *     security={{"bearerAuth":{}}},
     *     @OA\RequestBody(
     *         required=true,
     *         @OA\JsonContent(
     *             required={"log_date"},
     *             @OA\Property(property="log_date", type="string", format="date", example="2024-12-07"),
     *             @OA\Property(property="bleeding_intensity", type="string", enum={"low","medium","high","very_high"}),
     *             @OA\Property(property="blood_color", type="string", enum={"bright_red","red","dark_red","brown"}),
     *             @OA\Property(property="has_clots", type="boolean"),
     *             @OA\Property(property="spotting", type="boolean"),
     *             @OA\Property(property="bleeding_smell", type="string", enum={"normal","slightly_unusual","strong_unpleasant"}),
     *             @OA\Property(property="headache_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="stomach_ache_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="pelvic_pain_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="breast_pain_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="back_pain_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="ovarian_pain_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="nausea_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="bloating_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="diarrhea", type="boolean"),
     *             @OA\Property(property="constipation", type="boolean"),
     *             @OA\Property(property="appetite_change", type="string", enum={"loss","gain","normal"}),
     *             @OA\Property(property="food_craving", type="boolean"),
     *             @OA\Property(property="breast_sensitivity_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="vaginal_dryness", type="boolean"),
     *             @OA\Property(property="vaginal_burning", type="boolean"),
     *             @OA\Property(property="vaginal_itching", type="boolean"),
     *             @OA\Property(property="vaginal_smell_change", type="boolean"),
     *             @OA\Property(property="urination_change", type="string", enum={"increase","decrease","normal"}),
     *             @OA\Property(property="urination_burning_intensity", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="acne", type="boolean"),
     *             @OA\Property(property="oily_skin", type="boolean"),
     *             @OA\Property(property="hair_loss", type="boolean"),
     *             @OA\Property(property="swelling", type="boolean"),
     *             @OA\Property(property="fatigue", type="boolean"),
     *             @OA\Property(property="dizziness", type="boolean"),
     *             @OA\Property(property="hot_flashes", type="boolean"),
     *             @OA\Property(property="chills", type="boolean"),
     *             @OA\Property(property="moods", type="array", @OA\Items(type="string", enum={"happy","calm","angry","anxious","sad","frustrated","sensitive","bored"})),
     *             @OA\Property(property="sleep_duration", type="string", enum={"0_3","3_6","6_9","9_plus"}),
     *             @OA\Property(property="sleep_quality", type="string", enum={"good","medium","bad"}),
     *             @OA\Property(property="sexual_activity_level", type="string", enum={"low","medium","high","very_high"}),
     *             @OA\Property(property="weight", type="number", format="float", example=65.5),
     *             @OA\Property(property="basal_body_temperature", type="number", format="float", example=36.5),
     *             @OA\Property(property="discharge_color", type="string", example="سفید"),
     *             @OA\Property(property="discharge_texture", type="string", enum={"watery","creamy","egg_white","thick"}),
     *             @OA\Property(property="discharge_amount", type="string", enum={"low","medium","high"}),
     *             @OA\Property(property="discharge_smell", type="string", enum={"normal","slightly_unusual","strong_unpleasant"}),
     *             @OA\Property(property="discharge_itching", type="boolean"),
     *             @OA\Property(property="discharge_burning", type="boolean"),
     *             @OA\Property(property="frequent_urination", type="boolean"),
     *             @OA\Property(property="medications", type="object",
     *                 @OA\Property(property="painkillers", type="string", example="ایبوپروفن"),
     *                 @OA\Property(property="hormonal_pills", type="string", example="LD"),
     *                 @OA\Property(property="antibiotics", type="string"),
     *                 @OA\Property(property="supplements", type="string", example="آهن")
     *             ),
     *             @OA\Property(property="notes", type="string", example="امروز احساس خوبی داشتم")
     *         )
     *     ),
     *     @OA\Response(
     *         response=200,
     *         description="لاگ با موفقیت ثبت/ویرایش شد",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string"),
     *             @OA\Property(property="data", type="object")
     *         )
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
            'message' => $isNew ? 'لاگ سلامت با موفقیت ثبت شد' : 'لاگ سلامت با موفقیت ویرایش شد',
            'data' => $log,
        ], $isNew ? 201 : 200);
    }

    /**
     * @OA\Get(
     *     path="/api/v1/health-logs/{date}",
     *     tags={"Daily Health Log"},
     *     summary="دریافت لاگ یک روز خاص",
     *     security={{"bearerAuth":{}}},
     *     @OA\Parameter(
     *         name="date",
     *         in="path",
     *         description="تاریخ (YYYY-MM-DD)",
     *         required=true,
     *         @OA\Schema(type="string", format="date")
     *     ),
     *     @OA\Response(
     *         response=200,
     *         description="اطلاعات لاگ"
     *     ),
     *     @OA\Response(
     *         response=404,
     *         description="لاگی یافت نشد"
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
                'message' => 'لاگی برای این تاریخ یافت نشد',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'data' => $log,
        ]);
    }

    /**
     * @OA\Delete(
     *     path="/api/v1/health-logs/{date}",
     *     tags={"Daily Health Log"},
     *     summary="حذف لاگ یک روز",
     *     security={{"bearerAuth":{}}},
     *     @OA\Parameter(
     *         name="date",
     *         in="path",
     *         description="تاریخ (YYYY-MM-DD)",
     *         required=true,
     *         @OA\Schema(type="string", format="date")
     *     ),
     *     @OA\Response(
     *         response=200,
     *         description="لاگ با موفقیت حذف شد"
     *     ),
     *     @OA\Response(
     *         response=404,
     *         description="لاگی یافت نشد"
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
                'message' => 'لاگی برای این تاریخ یافت نشد',
            ], 404);
        }

        $log->delete();

        return response()->json([
            'success' => true,
            'message' => 'لاگ با موفقیت حذف شد',
        ]);
    }

    /**
     * @OA\Get(
     *     path="/api/v1/health-logs/enums",
     *     tags={"Daily Health Log"},
     *     summary="دریافت مقادیر enum برای فرم",
     *     security={{"bearerAuth":{}}},
     *     @OA\Response(
     *         response=200,
     *         description="لیست مقادیر enum"
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
