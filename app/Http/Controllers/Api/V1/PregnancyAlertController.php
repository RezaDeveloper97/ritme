<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\AlertLevel;
use App\Http\Controllers\Controller;
use App\Models\PregnancyAlert;
use App\Services\PregnancyEngine\PregnancyAlertService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class PregnancyAlertController extends Controller
{
    /**
     * @OA\Get(
     *     path="/pregnancy/alerts",
     *     summary="Get pregnancy alerts",
     *     description="Retrieve all active pregnancy alerts for the authenticated user",
     *     tags={"Pregnancy Alerts"},
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
     *         name="level",
     *         in="query",
     *         description="Filter by alert level",
     *         required=false,
     *         @OA\Schema(type="string", enum={"info","warning","emergency"})
     *     ),
     *     @OA\Parameter(
     *         name="unread_only",
     *         in="query",
     *         description="Only show unread alerts",
     *         required=false,
     *         @OA\Schema(type="boolean")
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Alerts retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="alerts", type="array", @OA\Items(ref="#/components/schemas/PregnancyAlert")),
     *                 @OA\Property(property="counts", type="object",
     *                     @OA\Property(property="total", type="integer", example=5),
     *                     @OA\Property(property="unread", type="integer", example=2),
     *                     @OA\Property(property="emergency", type="integer", example=0),
     *                     @OA\Property(property="warning", type="integer", example=1),
     *                     @OA\Property(property="info", type="integer", example=4)
     *                 )
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
        $query = PregnancyAlert::where('user_id', $user->id)
            ->active()
            ->orderBy('created_at', 'desc');

        // Filter by level
        if ($request->has('level') && in_array($request->get('level'), AlertLevel::values())) {
            $query->level($request->get('level'));
        }

        // Filter by unread
        if ($request->boolean('unread_only')) {
            $query->unread();
        }

        $alerts = $query->get();

        // Get counts
        $counts = [
            'total' => PregnancyAlert::where('user_id', $user->id)->active()->count(),
            'unread' => PregnancyAlert::where('user_id', $user->id)->active()->unread()->count(),
            'emergency' => PregnancyAlert::where('user_id', $user->id)->active()->level(AlertLevel::EMERGENCY->value)->count(),
            'warning' => PregnancyAlert::where('user_id', $user->id)->active()->level(AlertLevel::WARNING->value)->count(),
            'info' => PregnancyAlert::where('user_id', $user->id)->active()->level(AlertLevel::INFO->value)->count(),
        ];

        return response()->json([
            'success' => true,
            'data' => [
                'alerts' => $alerts,
                'counts' => $counts,
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/alerts/{id}",
     *     summary="Get specific alert",
     *     description="Retrieve a specific pregnancy alert by ID",
     *     tags={"Pregnancy Alerts"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="id",
     *         in="path",
     *         description="Alert ID",
     *         required=true,
     *         @OA\Schema(type="integer", example=1)
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
     *         description="Alert retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="alert", ref="#/components/schemas/PregnancyAlert")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Alert not found")
     * )
     */
    public function show(Request $request, int $id): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        $alert = PregnancyAlert::where('user_id', $user->id)
            ->where('id', $id)
            ->first();

        if (!$alert) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'هشدار یافت نشد' : 'Alert not found',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'data' => [
                'alert' => $alert,
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/pregnancy/alerts/{id}/read",
     *     summary="Mark alert as read",
     *     description="Mark a specific pregnancy alert as read",
     *     tags={"Pregnancy Alerts"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="id",
     *         in="path",
     *         description="Alert ID",
     *         required=true,
     *         @OA\Schema(type="integer", example=1)
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
     *         description="Alert marked as read",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Alert marked as read")
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Alert not found")
     * )
     */
    public function markAsRead(Request $request, int $id): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        $alert = PregnancyAlert::where('user_id', $user->id)
            ->where('id', $id)
            ->first();

        if (!$alert) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'هشدار یافت نشد' : 'Alert not found',
            ], 404);
        }

        $alert->markAsRead();

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'هشدار خوانده شد' : 'Alert marked as read',
        ]);
    }

    /**
     * @OA\Post(
     *     path="/pregnancy/alerts/read-all",
     *     summary="Mark all alerts as read",
     *     description="Mark all pregnancy alerts as read",
     *     tags={"Pregnancy Alerts"},
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
     *         description="All alerts marked as read",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="All alerts marked as read"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="count", type="integer", example=5)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function markAllAsRead(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        $count = PregnancyAlert::where('user_id', $user->id)
            ->unread()
            ->update([
                'is_read' => true,
                'read_at' => now(),
            ]);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'همه هشدارها خوانده شدند' : 'All alerts marked as read',
            'data' => [
                'count' => $count,
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/pregnancy/alerts/{id}/dismiss",
     *     summary="Dismiss alert",
     *     description="Dismiss a specific pregnancy alert",
     *     tags={"Pregnancy Alerts"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="id",
     *         in="path",
     *         description="Alert ID",
     *         required=true,
     *         @OA\Schema(type="integer", example=1)
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
     *         description="Alert dismissed",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Alert dismissed")
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Alert not found")
     * )
     */
    public function dismiss(Request $request, int $id): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        $alert = PregnancyAlert::where('user_id', $user->id)
            ->where('id', $id)
            ->first();

        if (!$alert) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'هشدار یافت نشد' : 'Alert not found',
            ], 404);
        }

        $alert->dismiss();

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'هشدار رد شد' : 'Alert dismissed',
        ]);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/alerts/summary",
     *     summary="Get alerts summary",
     *     description="Get a summary of pregnancy alerts including counts and emergency status",
     *     tags={"Pregnancy Alerts"},
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
     *         description="Alerts summary retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="has_emergency", type="boolean", example=false),
     *                 @OA\Property(property="has_unread", type="boolean", example=true),
     *                 @OA\Property(property="counts", type="object",
     *                     @OA\Property(property="total", type="integer", example=5),
     *                     @OA\Property(property="unread", type="integer", example=2),
     *                     @OA\Property(property="emergency", type="integer", example=0),
     *                     @OA\Property(property="warning", type="integer", example=1),
     *                     @OA\Property(property="info", type="integer", example=4)
     *                 ),
     *                 @OA\Property(property="latest_emergency", ref="#/components/schemas/PregnancyAlert")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function summary(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        $alertService = new PregnancyAlertService($user, $locale);
        $emergencyCount = $alertService->getEmergencyAlertsCount();
        $unreadAlerts = $alertService->getUnreadAlerts();

        // Get counts
        $counts = [
            'total' => PregnancyAlert::where('user_id', $user->id)->active()->count(),
            'unread' => $unreadAlerts->count(),
            'emergency' => $emergencyCount,
            'warning' => PregnancyAlert::where('user_id', $user->id)->active()->level(AlertLevel::WARNING->value)->count(),
            'info' => PregnancyAlert::where('user_id', $user->id)->active()->level(AlertLevel::INFO->value)->count(),
        ];

        // Get latest emergency if any
        $latestEmergency = null;
        if ($emergencyCount > 0) {
            $latestEmergency = PregnancyAlert::where('user_id', $user->id)
                ->level(AlertLevel::EMERGENCY->value)
                ->unread()
                ->active()
                ->orderBy('created_at', 'desc')
                ->first();
        }

        return response()->json([
            'success' => true,
            'data' => [
                'has_emergency' => $emergencyCount > 0,
                'has_unread' => $unreadAlerts->count() > 0,
                'counts' => $counts,
                'latest_emergency' => $latestEmergency,
            ],
        ]);
    }
}
