<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\CalculationStatus;
use App\Enums\ChronicCondition;
use App\Enums\PregnancyIntention;
use App\Enums\SubscriptionType;
use App\Enums\UserGoal;
use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Jobs\CalculateCycleDataJob;
use App\Models\UserProfile;
use App\Services\BmiService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;
use Illuminate\Validation\Rule;

class ProfileController extends Controller
{
    use ResolvesLocale;

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
     *                 ),
     *                 @OA\Property(property="bmi", type="object", nullable=true, description="Computed BMI; null when height/weight are missing",
     *                     @OA\Property(property="value", type="number", format="float", example=22.1),
     *                     @OA\Property(property="category", type="string", enum={"underweight","normal","overweight","obese"}, example="normal"),
     *                     @OA\Property(property="category_label", type="string", example="طبیعی"),
     *                     @OA\Property(property="message", type="string", example="بر اساس قد و وزن وارد شده، در محدوده‌ی وزنی طبیعی قرار می‌گیری...")
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
    public function show(Request $request, BmiService $bmi): JsonResponse
    {
        $locale = $this->resolveLocale($request);
        $user = $request->user();
        $profile = $user->profile;

        return response()->json([
            'success' => true,
            'data' => [
                'user' => $user,
                'profile' => $profile,
                // Computed body-mass index + supportive message; null when the
                // profile has no height/weight yet so the client omits the card.
                'bmi' => $profile ? $bmi->forProfile($profile, $locale) : null,
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
     *             @OA\Property(property="subscription_type", type="string", example="free", enum={"free","premium"}, description="Subscription type"),
     *             @OA\Property(property="pregnancy_intention", type="string", example="avoiding", enum={"avoiding","pregnant","trying","unsure"}, description="Pregnancy intention captured at onboarding; derives user_goal (trying=ttc)"),
     *             @OA\Property(property="chronic_conditions", type="array", @OA\Items(type="string", enum={"pcos","hypothyroidism","hyperthyroidism","hypertension","heart_disease","diabetes"}), description="Optional self-reported chronic conditions")
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
        $locale = $this->resolveLocale($request);
        app()->setLocale($locale);

        try {
            $user = $request->user();
            if (! $user) {
                return response()->json([
                    'success' => false,
                    'message' => __('profile.unauthenticated'),
                ], 401);
            }

            $allowedFields = [
                'name', 'birthday', 'weight', 'height', 'period_duration',
                'cycle_duration', 'last_period_start', 'user_goal', 'subscription_type',
                'pregnancy_intention', 'chronic_conditions',
            ];
            $unknownFields = array_diff(array_keys($request->all()), $allowedFields);
            if (! empty($unknownFields)) {
                return response()->json([
                    'success' => false,
                    'message' => __('profile.unknown_fields', ['fields' => implode(', ', $unknownFields)]),
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
                'pregnancy_intention' => ['nullable', Rule::in(PregnancyIntention::values())],
                'chronic_conditions' => ['nullable', 'array'],
                'chronic_conditions.*' => [Rule::in(ChronicCondition::values())],
            ], [
                'name.string' => __('profile.errors.name_string'),
                'name.max' => __('profile.errors.name_max'),
                'birthday.date' => __('profile.errors.birthday_date'),
                'birthday.before' => __('profile.errors.birthday_before'),
                'weight.numeric' => __('profile.errors.weight_numeric'),
                'weight.min' => __('profile.errors.weight_min'),
                'weight.max' => __('profile.errors.weight_max'),
                'height.integer' => __('profile.errors.height_integer'),
                'height.min' => __('profile.errors.height_min'),
                'height.max' => __('profile.errors.height_max'),
                'period_duration.integer' => __('profile.errors.period_duration_integer'),
                'period_duration.min' => __('profile.errors.period_duration_min'),
                'period_duration.max' => __('profile.errors.period_duration_max'),
                'cycle_duration.integer' => __('profile.errors.cycle_duration_integer'),
                'cycle_duration.min' => __('profile.errors.cycle_duration_min'),
                'cycle_duration.max' => __('profile.errors.cycle_duration_max'),
                'last_period_start.date' => __('profile.errors.last_period_start_date'),
                'last_period_start.before_or_equal' => __('profile.errors.last_period_start_before_or_equal'),
                'user_goal.in' => __('profile.errors.user_goal_in', ['values' => implode(', ', UserGoal::values())]),
                'subscription_type.in' => __('profile.errors.subscription_type_in', ['values' => implode(', ', SubscriptionType::values())]),
                'pregnancy_intention.in' => __('profile.errors.pregnancy_intention_in', ['values' => implode(', ', PregnancyIntention::values())]),
                'chronic_conditions.array' => __('profile.errors.chronic_conditions_array'),
                'chronic_conditions.*.in' => __('profile.errors.chronic_conditions_in', ['values' => implode(', ', ChronicCondition::values())]),
            ]);

            if ($validator->fails()) {
                return response()->json([
                    'success' => false,
                    'message' => __('profile.validation_failed', ['first' => $validator->errors()->first()]),
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
                'pregnancy_intention',
                'chronic_conditions',
            ]);

            // Keep the cycle engine's user_goal aligned with the stated intention,
            // unless the caller set user_goal explicitly. Only "trying" is TTC.
            if ($request->has('pregnancy_intention') && ! $request->has('user_goal')) {
                $profileData['user_goal'] = $request->pregnancy_intention === PregnancyIntention::TRYING->value
                    ? UserGoal::TTC->value
                    : UserGoal::NON_TTC->value;
            }

            // Don't seed a bogus period start for pregnant users — period tracking
            // is disabled in pregnancy mode.
            $isPregnant = ($profileData['pregnancy_intention'] ?? $profile->pregnancy_intention) === PregnancyIntention::PREGNANT->value;
            if (! $isPregnant && ! isset($profileData['last_period_start']) && ! $profile->last_period_start) {
                $profileData['last_period_start'] = now()->toDateString();
            }

            $cycleFieldsChanged = $this->hasCycleFieldsChanged($profile, $profileData);

            $profile->fill($profileData);
            $profile->save();

            if ($cycleFieldsChanged && $profile->last_period_start) {
                $this->triggerRecalculation($user, $profile, $locale);
            }

            return response()->json([
                'success' => true,
                'message' => __('profile.updated'),
                'data' => [
                    'user' => $user->fresh(),
                    'profile' => $profile->fresh(),
                    'calculation_status' => $profile->calculation_status ?? CalculationStatus::PENDING->value,
                ],
            ]);
        } catch (\Illuminate\Validation\ValidationException $e) {
            return response()->json([
                'success' => false,
                'message' => __('profile.validation_failed', ['first' => collect($e->errors())->flatten()->first()]),
                'errors' => $e->errors(),
            ], 422);
        } catch (\Illuminate\Database\QueryException $e) {
            \Log::error('Profile store DB error', ['error' => $e->getMessage(), 'user_id' => $request->user()?->id]);

            return response()->json([
                'success' => false,
                'message' => __('profile.db_error'),
                'error' => config('app.debug') ? $e->getMessage() : __('profile.try_again'),
            ], 500);
        } catch (\Throwable $e) {
            \Log::error('Profile store error', ['error' => $e->getMessage(), 'trace' => $e->getTraceAsString()]);

            return response()->json([
                'success' => false,
                'message' => __('profile.unexpected_error'),
                'error' => config('app.debug') ? $e->getMessage() : __('profile.try_again'),
            ], 500);
        }
    }

    /**
     * @OA\Get(
     *     path="/profile/export",
     *     summary="Export all personal data",
     *     description="Returns a full JSON export of the authenticated user's data (account, profile, health logs, cycle history, pregnancy data, reminders) for data-portability.",
     *     tags={"Profile"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Export generated successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="exported_at", type="string", format="date-time"),
     *                 @OA\Property(property="account", type="object"),
     *                 @OA\Property(property="profile", type="object", nullable=true),
     *                 @OA\Property(property="health_logs", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="cycle_histories", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="pregnancy", type="object"),
     *                 @OA\Property(property="reminders", type="array", @OA\Items(type="object"))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function export(Request $request): JsonResponse
    {
        $user = $request->user();

        return response()->json([
            'success' => true,
            'data' => [
                'exported_at' => now()->toIso8601String(),
                'account' => [
                    'id' => $user->id,
                    'name' => $user->name,
                    'mobile' => $user->mobile,
                    'created_at' => $user->created_at?->toIso8601String(),
                ],
                'profile' => $user->profile,
                'health_logs' => $user->dailyHealthLogs()->orderBy('log_date')->get(),
                'cycle_histories' => $user->cycleHistories()->orderBy('period_start_date')->get(),
                'pregnancy' => [
                    'profile' => $user->pregnancyProfile,
                    'symptom_logs' => $user->pregnancySymptomLogs()->orderBy('log_date')->get(),
                    'weekly_logs' => $user->pregnancyWeeklyLogs()->orderBy('pregnancy_week')->get(),
                    'fetal_movements' => $user->pregnancyFetalMovements()->get(),
                ],
                'reminders' => $user->reminders()->get(),
            ],
        ]);
    }

    /**
     * @OA\Delete(
     *     path="/account",
     *     summary="Delete account and all personal data",
     *     description="Permanently deletes the authenticated user's account together with every related record (profile, health logs, cycle data, pregnancy data, reminders, notifications) and revokes all access tokens. Irreversible.",
     *     tags={"Profile"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Account deleted",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string")
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function destroyAccount(Request $request): JsonResponse
    {
        $locale = $this->resolveLocale($request);
        $user = $request->user();

        // Revoke every issued token before removing the account so no orphaned
        // credentials survive the delete.
        $user->tokens()->each(fn ($token) => $token->revoke());

        $user->delete();

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa'
                ? 'حساب کاربری و همه داده‌های شما حذف شد'
                : 'Your account and all associated data have been deleted',
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
