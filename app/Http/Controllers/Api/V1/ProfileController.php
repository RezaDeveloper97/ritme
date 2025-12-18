<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\CalculationStatus;
use App\Enums\SubscriptionType;
use App\Enums\UserGoal;
use App\Http\Controllers\Controller;
use App\Jobs\CalculateCycleDataJob;
use App\Models\UserProfile;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;
use Illuminate\Validation\Rule;

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
     *             @OA\Property(property="last_period_start", type="string", format="date", example="2024-12-01", description="When did your last period start? Defaults to today if not provided"),
     *             @OA\Property(property="user_goal", type="string", example="non_ttc", enum={"ttc","non_ttc"}, description="User goal: ttc (Trying to Conceive) or non_ttc"),
     *             @OA\Property(property="subscription_type", type="string", example="free", enum={"free","premium"}, description="Subscription type")
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
            'user_goal' => ['nullable', Rule::in(UserGoal::values())],
            'subscription_type' => ['nullable', Rule::in(SubscriptionType::values())],
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
            'user_goal',
            'subscription_type',
        ]);

        // Default last_period_start to today if not provided and profile doesn't have one
        if (!isset($profileData['last_period_start']) && !$profile->last_period_start) {
            $profileData['last_period_start'] = now()->toDateString();
        }

        // Check if cycle-related data changed
        $cycleFieldsChanged = $this->hasCycleFieldsChanged($profile, $profileData);

        $profile->fill($profileData);
        $profile->save();

        // Trigger recalculation if cycle data changed and profile has required data
        if ($cycleFieldsChanged && $profile->last_period_start) {
            $this->triggerRecalculation($user, $profile, $request->header('Accept-Language', 'en'));
        }

        return response()->json([
            'success' => true,
            'message' => 'Profile updated successfully',
            'data' => [
                'user' => $user->fresh(),
                'profile' => $profile->fresh(),
                'calculation_status' => $profile->calculation_status ?? CalculationStatus::PENDING->value,
            ],
        ]);
    }

    /**
     * Check if cycle-related fields have changed
     */
    private function hasCycleFieldsChanged(UserProfile $profile, array $newData): bool
    {
        $cycleFields = ['birthday', 'period_duration', 'cycle_duration', 'last_period_start'];

        foreach ($cycleFields as $field) {
            if (isset($newData[$field])) {
                $oldValue = $profile->getOriginal($field);
                $newValue = $newData[$field];

                // Convert dates to comparable format
                if ($oldValue instanceof \Carbon\Carbon) {
                    $oldValue = $oldValue->toDateString();
                }

                if ($oldValue != $newValue) {
                    return true;
                }
            }
        }

        return false;
    }

    /**
     * Trigger background recalculation of cycle data
     */
    private function triggerRecalculation($user, UserProfile $profile, string $locale): void
    {
        // Don't trigger if already processing
        if ($profile->calculation_status === CalculationStatus::PROCESSING->value) {
            return;
        }

        // Increment version and dispatch job
        $newVersion = ($profile->calculation_version ?? 0) + 1;

        $profile->update([
            'calculation_status' => CalculationStatus::PROCESSING->value,
        ]);

        CalculateCycleDataJob::dispatch($user->id, $newVersion, $locale);
    }
}
