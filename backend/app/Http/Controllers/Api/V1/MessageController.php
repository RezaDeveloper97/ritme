<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Services\MessageSystem\Core\MessageManager;
use App\Services\MessageSystem\Enums\MessageMode;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * @OA\Tag(
 *     name="Messages",
 *     description="Personalized message system for cycle and pregnancy modes"
 * )
 */
class MessageController extends Controller
{
    /**
     * @OA\Get(
     *     path="/messages/daily",
     *     summary="Get daily personalized messages",
     *     description="Automatically detects user's current mode (cycle/pregnancy) and returns personalized messages including base message, correlations, patterns (premium), and supplement tips (nutrition, sleep, exercise).",
     *     tags={"Messages"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="query",
     *         description="Date for messages (YYYY-MM-DD). Defaults to today.",
     *         required=false,
     *         @OA\Schema(type="string", format="date", example="2024-12-20")
     *     ),
     *     @OA\Parameter(
     *         name="mode",
     *         in="query",
     *         description="Force specific mode (cycle, pregnancy). If not provided, auto-detects.",
     *         required=false,
     *         @OA\Schema(type="string", enum={"cycle", "pregnancy"})
     *     ),
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for messages (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="fa", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Messages retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="mode", type="string", example="cycle"),
     *                 @OA\Property(property="date", type="string", format="date", example="2024-12-20"),
     *                 @OA\Property(property="user_goal", type="string", example="non_ttc"),
     *                 @OA\Property(property="subscription_type", type="string", example="free"),
     *                 @OA\Property(property="context_info", type="object",
     *                     @OA\Property(property="phase", type="string", example="follicular"),
     *                     @OA\Property(property="phase_label", type="string", example="Follicular"),
     *                     @OA\Property(property="cycle_day", type="integer", example=8)
     *                 ),
     *                 @OA\Property(property="primary_message", type="object",
     *                     @OA\Property(property="short_message", type="string"),
     *                     @OA\Property(property="long_message", type="string"),
     *                     @OA\Property(property="action_suggestion", type="string"),
     *                     @OA\Property(property="dos", type="array", @OA\Items(type="string")),
     *                     @OA\Property(property="donts", type="array", @OA\Items(type="string"))
     *                 ),
     *                 @OA\Property(property="correlations", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="patterns", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="supplements", type="object",
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
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function daily(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'fa');

        // Get date parameter
        $date = $request->query('date')
            ? Carbon::parse($request->query('date'))
            : Carbon::today();

        // Get optional mode override
        $forceMode = null;
        if ($request->query('mode')) {
            $forceMode = MessageMode::tryFrom($request->query('mode'));
        }

        // Create message manager
        $manager = new MessageManager($user, $locale);

        // Check if user has required profile data
        $detectedMode = $forceMode ?? $manager->detectMode();

        if ($detectedMode === MessageMode::CYCLE) {
            $profile = $user->profile;
            if (!$profile || !$profile->last_period_start) {
                return response()->json([
                    'success' => false,
                    'message' => $locale === 'fa'
                        ? 'لطفاً ابتدا پروفایل خود را تکمیل کنید (تاریخ آخرین پریود)'
                        : 'Please complete your profile first (last period date)',
                ], 400);
            }
        }

        if ($detectedMode === MessageMode::PREGNANCY) {
            $pregnancyProfile = $user->pregnancyProfile;
            if (!$pregnancyProfile || !$pregnancyProfile->onboarding_completed) {
                return response()->json([
                    'success' => false,
                    'message' => $locale === 'fa'
                        ? 'لطفاً ابتدا آنبوردینگ بارداری را تکمیل کنید'
                        : 'Please complete pregnancy onboarding first',
                ], 400);
            }
        }

        // Generate messages
        $result = $manager->generateMessages($date, $forceMode);

        return response()->json([
            'success' => true,
            'data' => $result->toArray(),
        ]);
    }

    /**
     * @OA\Get(
     *     path="/messages/enums",
     *     summary="Get message system enums",
     *     description="Retrieve all available enum values for the message system including modes, user goals, subscription types, and mode-specific enums.",
     *     tags={"Messages"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for labels (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="fa", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Enums retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="modes", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="user_goals", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="subscription_types", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="cycle", type="object"),
     *                 @OA\Property(property="pregnancy", type="object")
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
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'fa');

        $manager = new MessageManager($user, $locale);
        $enums = $manager->getEnums();

        return response()->json([
            'success' => true,
            'data' => $enums,
        ]);
    }

    /**
     * @OA\Get(
     *     path="/messages/mode",
     *     summary="Get current user mode",
     *     description="Detect and return the current mode for the user (cycle or pregnancy).",
     *     tags={"Messages"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for labels (en, fa)",
     *         required=false,
     *         @OA\Schema(type="string", default="fa", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Mode detected successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="mode", type="string", example="cycle"),
     *                 @OA\Property(property="mode_label", type="string", example="Menstrual Cycle"),
     *                 @OA\Property(property="user_goal", type="string", example="non_ttc"),
     *                 @OA\Property(property="subscription_type", type="string", example="free"),
     *                 @OA\Property(property="is_ttc", type="boolean", example=false),
     *                 @OA\Property(property="is_premium", type="boolean", example=false)
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
    public function mode(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'fa');
        $profile = $user->profile;

        $manager = new MessageManager($user, $locale);
        $mode = $manager->detectMode();

        return response()->json([
            'success' => true,
            'data' => [
                'mode' => $mode->value,
                'mode_label' => $mode->label($locale),
                'user_goal' => $profile?->user_goal ?? 'non_ttc',
                'subscription_type' => $profile?->subscription_type ?? 'free',
                'is_ttc' => $profile?->isTTC() ?? false,
                'is_premium' => $profile?->isPremium() ?? false,
            ],
        ]);
    }
}
