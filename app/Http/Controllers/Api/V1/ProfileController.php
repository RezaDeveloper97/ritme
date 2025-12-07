<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Models\UserProfile;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;

class ProfileController extends Controller
{
    /**
     * @OA\Get(
     *     path="/profile",
     *     summary="Get user profile",
     *     description="Retrieve the authenticated user's profile information",
     *     tags={"Profile"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Profile retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="user", type="object",
     *                     @OA\Property(property="id", type="integer", example=1),
     *                     @OA\Property(property="name", type="string", nullable=true, example="Sara"),
     *                     @OA\Property(property="mobile", type="string", example="09123456789")
     *                 ),
     *                 @OA\Property(property="profile", type="object", nullable=true,
     *                     @OA\Property(property="birthday", type="string", format="date", example="1995-05-15"),
     *                     @OA\Property(property="weight", type="number", format="float", example=60.5),
     *                     @OA\Property(property="height", type="integer", example=165),
     *                     @OA\Property(property="period_duration", type="integer", example=5),
     *                     @OA\Property(property="cycle_duration", type="integer", example=28),
     *                     @OA\Property(property="last_period_start", type="string", format="date", example="2024-12-01")
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
    public function show(Request $request): JsonResponse
    {
        $user = $request->user();
        $profile = $user->profile;

        return response()->json([
            'success' => true,
            'data' => [
                'user' => $user,
                'profile' => $profile,
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/profile",
     *     summary="Create or update user profile",
     *     description="Create or update the authenticated user's profile information including menstrual cycle data",
     *     tags={"Profile"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\RequestBody(
     *         required=true,
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="name", type="string", example="Sara", description="User's name"),
     *             @OA\Property(property="birthday", type="string", format="date", example="1995-05-15", description="Date of birth"),
     *             @OA\Property(property="weight", type="number", format="float", example=60.5, description="Weight in kg"),
     *             @OA\Property(property="height", type="integer", example=165, description="Height in cm"),
     *             @OA\Property(property="period_duration", type="integer", example=5, description="How long does your period usually last? (days)"),
     *             @OA\Property(property="cycle_duration", type="integer", example=28, description="How long does your cycle usually last? (days)"),
     *             @OA\Property(property="last_period_start", type="string", format="date", example="2024-12-01", description="When did your last period start?")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Profile updated successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Profile updated successfully"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="user", type="object",
     *                     @OA\Property(property="id", type="integer", example=1),
     *                     @OA\Property(property="name", type="string", example="Sara"),
     *                     @OA\Property(property="mobile", type="string", example="09123456789")
     *                 ),
     *                 @OA\Property(property="profile", type="object",
     *                     @OA\Property(property="birthday", type="string", format="date", example="1995-05-15"),
     *                     @OA\Property(property="weight", type="number", format="float", example=60.5),
     *                     @OA\Property(property="height", type="integer", example=165),
     *                     @OA\Property(property="period_duration", type="integer", example=5),
     *                     @OA\Property(property="cycle_duration", type="integer", example=28),
     *                     @OA\Property(property="last_period_start", type="string", format="date", example="2024-12-01")
     *                 )
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=422,
     *         description="Validation error",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="Validation failed"),
     *             @OA\Property(property="errors", type="object")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function store(Request $request): JsonResponse
    {
        $validator = Validator::make($request->all(), [
            'name' => 'nullable|string|max:255',
            'birthday' => 'nullable|date|before:today',
            'weight' => 'nullable|numeric|min:20|max:300',
            'height' => 'nullable|integer|min:50|max:250',
            'period_duration' => 'nullable|integer|min:1|max:15',
            'cycle_duration' => 'nullable|integer|min:15|max:60',
            'last_period_start' => 'nullable|date|before_or_equal:today',
        ]);

        if ($validator->fails()) {
            return response()->json([
                'success' => false,
                'message' => 'Validation failed',
                'errors' => $validator->errors(),
            ], 422);
        }

        $user = $request->user();

        // Update user name if provided
        if ($request->has('name')) {
            $user->update(['name' => $request->name]);
        }

        // Get or create profile
        $profile = $user->profile ?: new UserProfile(['user_id' => $user->id]);

        // Update profile fields
        $profileData = $request->only([
            'birthday',
            'weight',
            'height',
            'period_duration',
            'cycle_duration',
            'last_period_start',
        ]);

        $profile->fill($profileData);
        $profile->save();

        return response()->json([
            'success' => true,
            'message' => 'Profile updated successfully',
            'data' => [
                'user' => $user->fresh(),
                'profile' => $profile->fresh(),
            ],
        ]);
    }
}
