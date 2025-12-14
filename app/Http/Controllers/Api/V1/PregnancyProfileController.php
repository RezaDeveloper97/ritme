<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\AlertLevel;
use App\Enums\BloodType;
use App\Enums\ConfidenceLevel;
use App\Enums\PreExistingCondition;
use App\Enums\PregnancyAgeSource;
use App\Enums\RhFactor;
use App\Http\Controllers\Controller;
use App\Http\Requests\Api\V1\StorePregnancyProfileRequest;
use App\Http\Requests\Api\V1\UpdatePregnancyProfileRequest;
use App\Models\PregnancyProfile;
use App\Services\PregnancyEngine\PregnancyCalculationService;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class PregnancyProfileController extends Controller
{
    /**
     * @OA\Post(
     *     path="/pregnancy/activate",
     *     summary="Activate pregnancy mode",
     *     description="Switch from cycle mode to pregnancy mode. This initiates the pregnancy onboarding flow.",
     *     tags={"Pregnancy"},
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
     *         description="Pregnancy mode activated successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Pregnancy mode activated"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="pregnancy_mode", type="boolean", example=true),
     *                 @OA\Property(property="cycle_mode", type="boolean", example=false),
     *                 @OA\Property(property="onboarding_required", type="boolean", example=true)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function activate(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        // Create or update pregnancy profile
        $profile = PregnancyProfile::updateOrCreate(
            ['user_id' => $user->id],
            [
                'pregnancy_mode' => true,
                'cycle_mode' => false,
            ]
        );

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'حالت بارداری فعال شد' : 'Pregnancy mode activated',
            'data' => [
                'pregnancy_mode' => true,
                'cycle_mode' => false,
                'onboarding_required' => !$profile->onboarding_completed,
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/pregnancy/deactivate",
     *     summary="Deactivate pregnancy mode",
     *     description="Switch back from pregnancy mode to cycle mode.",
     *     tags={"Pregnancy"},
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
     *         description="Pregnancy mode deactivated successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Pregnancy mode deactivated"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="pregnancy_mode", type="boolean", example=false),
     *                 @OA\Property(property="cycle_mode", type="boolean", example=true)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function deactivate(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        $profile = $user->pregnancyProfile;

        if ($profile) {
            $profile->update([
                'pregnancy_mode' => false,
                'cycle_mode' => true,
            ]);
        }

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'حالت بارداری غیرفعال شد' : 'Pregnancy mode deactivated',
            'data' => [
                'pregnancy_mode' => false,
                'cycle_mode' => true,
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/pregnancy/onboarding",
     *     summary="Complete pregnancy onboarding",
     *     description="Submit pregnancy onboarding data including gestational age calculation source (LMP, ultrasound, or manual entry).",
     *     tags={"Pregnancy"},
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
     *             required={"age_source"},
     *             @OA\Property(property="age_source", type="string", enum={"lmp","ultrasound","manual"}, description="Source for calculating gestational age"),
     *             @OA\Property(property="lmp_date", type="string", format="date", description="Last Menstrual Period date (required if age_source is lmp)"),
     *             @OA\Property(property="ultrasound_date", type="string", format="date", description="Ultrasound date (required if age_source is ultrasound)"),
     *             @OA\Property(property="ultrasound_weeks", type="integer", minimum=1, maximum=42, description="Gestational weeks at ultrasound"),
     *             @OA\Property(property="ultrasound_days", type="integer", minimum=0, maximum=6, description="Additional days at ultrasound"),
     *             @OA\Property(property="manual_weeks", type="integer", minimum=1, maximum=42, description="Current pregnancy weeks (manual entry)"),
     *             @OA\Property(property="manual_days", type="integer", minimum=0, maximum=6, description="Current pregnancy days (manual entry)"),
     *             @OA\Property(property="has_miscarriage_history", type="boolean"),
     *             @OA\Property(property="has_high_risk_history", type="boolean"),
     *             @OA\Property(property="pre_existing_conditions", type="array", @OA\Items(type="string", enum={"chronic_hypertension","diabetes","hypothyroidism","hyperthyroidism","none"})),
     *             @OA\Property(property="blood_type", type="string", enum={"A","B","AB","O"}),
     *             @OA\Property(property="rh_factor", type="string", enum={"positive","negative"})
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=201,
     *         description="Onboarding completed successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Pregnancy onboarding completed"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="profile", ref="#/components/schemas/PregnancyProfile"),
     *                 @OA\Property(property="gestational_age", type="object",
     *                     @OA\Property(property="weeks", type="integer", example=12),
     *                     @OA\Property(property="days", type="integer", example=3),
     *                     @OA\Property(property="confidence_level", type="string", example="medium")
     *                 ),
     *                 @OA\Property(property="estimated_due_date", type="string", format="date")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=422, description="Validation error")
     * )
     */
    public function onboarding(StorePregnancyProfileRequest $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $validated = $request->validated();

        // Determine confidence level based on source
        $confidenceLevel = match ($validated['age_source']) {
            PregnancyAgeSource::ULTRASOUND->value => ConfidenceLevel::HIGH->value,
            PregnancyAgeSource::LMP->value => ConfidenceLevel::MEDIUM->value,
            PregnancyAgeSource::MANUAL->value => ConfidenceLevel::LOW->value,
            default => ConfidenceLevel::LOW->value,
        };

        // Set uncertainty days based on source
        $uncertaintyDays = match ($validated['age_source']) {
            PregnancyAgeSource::ULTRASOUND->value => 1,
            PregnancyAgeSource::LMP->value => 3,
            PregnancyAgeSource::MANUAL->value => 5,
            default => 5,
        };

        // If manual entry, set the entry date to today
        if ($validated['age_source'] === PregnancyAgeSource::MANUAL->value) {
            $validated['manual_entry_date'] = Carbon::today()->toDateString();
        }

        // Check RH factor for special care flag
        $rhNegativeCareFlag = isset($validated['rh_factor']) && $validated['rh_factor'] === RhFactor::NEGATIVE->value;

        // Create or update profile
        $profile = PregnancyProfile::updateOrCreate(
            ['user_id' => $user->id],
            array_merge($validated, [
                'pregnancy_mode' => true,
                'cycle_mode' => false,
                'confidence_level' => $confidenceLevel,
                'uncertainty_days' => $uncertaintyDays,
                'rh_negative_care_flag' => $rhNegativeCareFlag,
                'onboarding_completed' => true,
                'onboarding_completed_at' => now(),
            ])
        );

        // Calculate gestational age and EDD
        $calculationService = new PregnancyCalculationService($user->fresh(), $locale);
        $gestationalAge = $calculationService->calculateGestationalAge();
        $edd = $calculationService->calculateEDD();

        // Update calculated fields
        if ($edd) {
            $profile->update([
                'estimated_due_date' => $edd['date'],
            ]);
        }

        $profile->refresh();

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'آنبوردینگ بارداری تکمیل شد' : 'Pregnancy onboarding completed',
            'data' => [
                'profile' => $profile,
                'gestational_age' => $gestationalAge,
                'estimated_due_date' => $edd,
            ],
        ], 201);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/profile",
     *     summary="Get pregnancy profile",
     *     description="Retrieve the user's pregnancy profile with calculated gestational age and status.",
     *     tags={"Pregnancy"},
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
     *         description="Pregnancy profile retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="profile", ref="#/components/schemas/PregnancyProfile"),
     *                 @OA\Property(property="status", type="object")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Pregnancy profile not found")
     * )
     */
    public function show(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $profile = $user->pregnancyProfile;

        if (!$profile) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'پروفایل بارداری یافت نشد' : 'Pregnancy profile not found',
            ], 404);
        }

        $calculationService = new PregnancyCalculationService($user, $locale);
        $status = $calculationService->getPregnancyStatus();

        return response()->json([
            'success' => true,
            'data' => [
                'profile' => $profile,
                'status' => $status,
            ],
        ]);
    }

    /**
     * @OA\Put(
     *     path="/pregnancy/profile",
     *     summary="Update pregnancy profile",
     *     description="Update pregnancy profile data. Profile must be unlocked or have edit permission.",
     *     tags={"Pregnancy"},
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
     *             @OA\Property(property="age_source", type="string", enum={"lmp","ultrasound","manual"}),
     *             @OA\Property(property="lmp_date", type="string", format="date"),
     *             @OA\Property(property="ultrasound_date", type="string", format="date"),
     *             @OA\Property(property="ultrasound_weeks", type="integer", minimum=1, maximum=42),
     *             @OA\Property(property="ultrasound_days", type="integer", minimum=0, maximum=6),
     *             @OA\Property(property="manual_weeks", type="integer", minimum=1, maximum=42),
     *             @OA\Property(property="manual_days", type="integer", minimum=0, maximum=6),
     *             @OA\Property(property="has_miscarriage_history", type="boolean"),
     *             @OA\Property(property="has_high_risk_history", type="boolean"),
     *             @OA\Property(property="pre_existing_conditions", type="array", @OA\Items(type="string")),
     *             @OA\Property(property="blood_type", type="string", enum={"A","B","AB","O"}),
     *             @OA\Property(property="rh_factor", type="string", enum={"positive","negative"}),
     *             @OA\Property(property="fetal_movement_felt", type="boolean"),
     *             @OA\Property(property="first_fetal_movement_date", type="string", format="date")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Pregnancy profile updated successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Profile updated successfully"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="profile", ref="#/components/schemas/PregnancyProfile"),
     *                 @OA\Property(property="status", type="object")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Pregnancy profile not found"),
     *     @OA\Response(response=422, description="Validation error")
     * )
     */
    public function update(UpdatePregnancyProfileRequest $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $profile = $user->pregnancyProfile;

        if (!$profile) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'پروفایل بارداری یافت نشد' : 'Pregnancy profile not found',
            ], 404);
        }

        $validated = $request->validated();

        // Handle age source change
        if (isset($validated['age_source'])) {
            $validated['confidence_level'] = match ($validated['age_source']) {
                PregnancyAgeSource::ULTRASOUND->value => ConfidenceLevel::HIGH->value,
                PregnancyAgeSource::LMP->value => ConfidenceLevel::MEDIUM->value,
                PregnancyAgeSource::MANUAL->value => ConfidenceLevel::LOW->value,
                default => $profile->confidence_level,
            };

            $validated['uncertainty_days'] = match ($validated['age_source']) {
                PregnancyAgeSource::ULTRASOUND->value => 1,
                PregnancyAgeSource::LMP->value => 3,
                PregnancyAgeSource::MANUAL->value => 5,
                default => $profile->uncertainty_days,
            };

            if ($validated['age_source'] === PregnancyAgeSource::MANUAL->value && !isset($validated['manual_entry_date'])) {
                $validated['manual_entry_date'] = Carbon::today()->toDateString();
            }
        }

        // Check RH factor
        if (isset($validated['rh_factor'])) {
            $validated['rh_negative_care_flag'] = $validated['rh_factor'] === RhFactor::NEGATIVE->value;
        }

        $profile->update($validated);

        // Recalculate EDD if age source data changed
        $calculationService = new PregnancyCalculationService($user->fresh(), $locale);
        $edd = $calculationService->calculateEDD();

        if ($edd) {
            $profile->update(['estimated_due_date' => $edd['date']]);
        }

        $profile->refresh();
        $status = $calculationService->getPregnancyStatus();

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'پروفایل با موفقیت بروزرسانی شد' : 'Profile updated successfully',
            'data' => [
                'profile' => $profile,
                'status' => $status,
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/pregnancy/confirm",
     *     summary="Confirm and lock pregnancy profile",
     *     description="Confirm the pregnancy profile after review. This locks the profile but allows editing.",
     *     tags={"Pregnancy"},
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
     *         description="Pregnancy profile confirmed and locked",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Profile confirmed and locked"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="is_locked", type="boolean", example=true)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Pregnancy profile not found")
     * )
     */
    public function confirm(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');
        $profile = $user->pregnancyProfile;

        if (!$profile) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'پروفایل بارداری یافت نشد' : 'Pregnancy profile not found',
            ], 404);
        }

        $profile->update(['is_locked' => true]);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'پروفایل تأیید و قفل شد' : 'Profile confirmed and locked',
            'data' => [
                'is_locked' => true,
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/status",
     *     summary="Get pregnancy status",
     *     description="Get current pregnancy status including gestational age, trimester, and flags.",
     *     tags={"Pregnancy"},
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
     *         description="Pregnancy status retrieved successfully",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="is_active", type="boolean", example=true),
     *                 @OA\Property(property="gestational_age", type="object",
     *                     @OA\Property(property="weeks", type="integer", example=20),
     *                     @OA\Property(property="days", type="integer", example=3),
     *                     @OA\Property(property="trimester", type="integer", example=2)
     *                 ),
     *                 @OA\Property(property="estimated_due_date", type="object"),
     *                 @OA\Property(property="current_week", type="integer", example=21),
     *                 @OA\Property(property="is_high_risk", type="boolean", example=false),
     *                 @OA\Property(property="fetal_movement_tracking_active", type="boolean", example=true),
     *                 @OA\Property(property="fetal_movement_required", type="boolean", example=false),
     *                 @OA\Property(property="flags", type="object")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function status(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $request->header('Accept-Language', 'en');

        $calculationService = new PregnancyCalculationService($user, $locale);
        $status = $calculationService->getPregnancyStatus();

        return response()->json([
            'success' => true,
            'data' => $status,
        ]);
    }

    /**
     * @OA\Get(
     *     path="/pregnancy/enums",
     *     summary="Get pregnancy enum values",
     *     description="Retrieve all available enum values for pregnancy profile fields",
     *     tags={"Pregnancy"},
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

        $ageSources = collect(PregnancyAgeSource::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
            'description' => $s->description($locale),
        ])->values();

        $confidenceLevels = collect(ConfidenceLevel::cases())->map(fn($c) => [
            'value' => $c->value,
            'label' => $c->label($locale),
            'description' => $c->description($locale),
        ])->values();

        $bloodTypes = collect(BloodType::cases())->map(fn($b) => [
            'value' => $b->value,
            'label' => $b->label($locale),
        ])->values();

        $rhFactors = collect(RhFactor::cases())->map(fn($r) => [
            'value' => $r->value,
            'label' => $r->label($locale),
        ])->values();

        $conditions = collect(PreExistingCondition::cases())->map(fn($c) => [
            'value' => $c->value,
            'label' => $c->label($locale),
        ])->values();

        $alertLevels = collect(AlertLevel::cases())->map(fn($a) => [
            'value' => $a->value,
            'label' => $a->label($locale),
            'description' => $a->description($locale),
        ])->values();

        return response()->json([
            'success' => true,
            'data' => [
                'age_sources' => $ageSources,
                'confidence_levels' => $confidenceLevels,
                'blood_types' => $bloodTypes,
                'rh_factors' => $rhFactors,
                'pre_existing_conditions' => $conditions,
                'alert_levels' => $alertLevels,
            ],
        ]);
    }
}
