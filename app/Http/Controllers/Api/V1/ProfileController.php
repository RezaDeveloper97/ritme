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
        try {
            $user = $request->user();
            if (!$user) {
                return response()->json([
                    'success' => false,
                    'message' => 'Unauthenticated. Please provide a valid Bearer token.',
                ], 401);
            }

            $allowedFields = [
                'name', 'birthday', 'weight', 'height', 'period_duration',
                'cycle_duration', 'last_period_start', 'user_goal', 'subscription_type',
            ];
            $unknownFields = array_diff(array_keys($request->all()), $allowedFields);
            if (!empty($unknownFields)) {
                return response()->json([
                    'success' => false,
                    'message' => 'Unknown field(s) in request: ' . implode(', ', $unknownFields),
                    'allowed_fields' => $allowedFields,
                ], 422);
            }

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
            ], [
                'birthday.before' => 'Birthday must be a date before today.',
                'weight.min' => 'Weight must be at least 20 kg.',
                'weight.max' => 'Weight must not exceed 300 kg.',
                'height.min' => 'Height must be at least 50 cm.',
                'height.max' => 'Height must not exceed 250 cm.',
                'period_duration.min' => 'Period duration must be at least 1 day.',
                'period_duration.max' => 'Period duration must not exceed 15 days.',
                'cycle_duration.min' => 'Cycle duration must be at least 15 days.',
                'cycle_duration.max' => 'Cycle duration must not exceed 60 days.',
                'last_period_start.before_or_equal' => 'Last period start cannot be in the future.',
                'user_goal.in' => 'user_goal must be one of: ' . implode(', ', UserGoal::values()),
                'subscription_type.in' => 'subscription_type must be one of: ' . implode(', ', SubscriptionType::values()),
            ]);

            if ($validator->fails()) {
                return response()->json([
                    'success' => false,
                    'message' => 'Validation failed: ' . $validator->errors()->first(),
                    'errors' => $validator->errors(),
                ], 422);
            }

            if ($request->has('name')) {
                $user->update(['name' => $request->name]);
            }

            $profile = $user->profile ?: new UserProfile(['user_id' => $user->id]);

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

            if (!isset($profileData['last_period_start']) && !$profile->last_period_start) {
                $profileData['last_period_start'] = now()->toDateString();
            }

            $cycleFieldsChanged = $this->hasCycleFieldsChanged($profile, $profileData);

            $profile->fill($profileData);
            $profile->save();

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
        } catch (\Illuminate\Validation\ValidationException $e) {
            return response()->json([
                'success' => false,
                'message' => 'Validation failed: ' . collect($e->errors())->flatten()->first(),
                'errors' => $e->errors(),
            ], 422);
        } catch (\Illuminate\Database\QueryException $e) {
            \Log::error('Profile store DB error', ['error' => $e->getMessage(), 'user_id' => $request->user()?->id]);
            return response()->json([
                'success' => false,
                'message' => 'Database error while saving profile.',
                'error' => config('app.debug') ? $e->getMessage() : 'Please try again later.',
            ], 500);
        } catch (\Throwable $e) {
            \Log::error('Profile store error', ['error' => $e->getMessage(), 'trace' => $e->getTraceAsString()]);
            return response()->json([
                'success' => false,
                'message' => 'An unexpected error occurred while updating your profile.',
                'error' => config('app.debug') ? $e->getMessage() : 'Please try again later.',
            ], 500);
        }
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
