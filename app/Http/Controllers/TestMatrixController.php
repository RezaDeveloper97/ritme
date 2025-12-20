<?php

namespace App\Http\Controllers;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\OverrideType;
use App\Enums\SubscriptionType;
use App\Enums\UserGoal;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserProfile;
use App\Services\HealthDataEngine;
use App\Services\MatrixEngine\CorrelationEngine;
use App\Services\MatrixEngine\MatrixMessageEngine;
use App\Services\MatrixEngine\NutritionSleepModule;
use App\Services\MatrixEngine\PatternRecognitionEngine;
use App\Services\MatrixEngine\TTCMatrixEngine;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class TestMatrixController extends Controller
{
    // Test mobile number for default user
    private const TEST_MOBILE = '09123456789';

    /**
     * Display matrix test page
     */
    public function index()
    {
        $user = $this->getOrCreateTestUser();
        $profile = $user->profile;

        return view('test-matrix', [
            'user' => $user,
            'profile' => $profile,
        ]);
    }

    /**
     * Get matrix enums
     */
    public function getEnums(Request $request): JsonResponse
    {
        $locale = $request->input('locale', 'fa');

        $phases = collect(CyclePhase::cases())->map(fn($p) => [
            'value' => $p->value,
            'label' => $p->label($locale),
            'description' => $p->description($locale),
        ])->values();

        $subphases = collect(CycleSubphase::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $userGoals = collect(UserGoal::cases())->map(fn($g) => [
            'value' => $g->value,
            'label' => $g->label($locale),
        ])->values();

        $subscriptionTypes = collect(SubscriptionType::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $overrideTypes = collect(OverrideType::cases())->map(fn($o) => [
            'value' => $o->value,
            'label' => $o->label($locale),
        ])->values();

        return response()->json([
            'success' => true,
            'data' => [
                'phases' => $phases,
                'subphases' => $subphases,
                'user_goals' => $userGoals,
                'subscription_types' => $subscriptionTypes,
                'override_types' => $overrideTypes,
            ],
        ]);
    }

    /**
     * Get matrix messages for a specific date
     */
    public function getMatrixMessages(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $profile = $user->profile;

        if (!$profile || !$profile->last_period_start) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'لطفاً ابتدا پروفایل خود را تکمیل کنید (تاریخ آخرین پریود را در صفحه تست اصلی وارد کنید)'
                    : 'Please complete your profile first (enter last period date in the main test page)',
            ], 400);
        }

        // Get target date
        $date = $request->input('date')
            ? Carbon::parse($request->input('date'))
            : Carbon::today();

        // Get cycle calculation data
        $healthEngine = new HealthDataEngine($user, $locale);
        $cycleData = $healthEngine->calculateForDate($date);

        // Check if we have valid cycle data
        if (!isset($cycleData['phase']) || !$cycleData['phase']) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'اطلاعات سیکل موجود نیست'
                    : 'Cycle information not available',
            ], 400);
        }

        // Get phase and subphase enums
        $phase = CyclePhase::from($cycleData['phase']);
        $subphase = CycleSubphase::from($cycleData['subphase']);
        $cycleDay = $cycleData['cycle_day'];
        $isFertileWindow = $cycleData['is_fertile_window'] ?? false;
        $isPmsWindow = $cycleData['is_pms_window'] ?? false;

        // Get daily health log
        $dailyLog = DailyHealthLog::where('user_id', $user->id)
            ->whereDate('log_date', $date)
            ->first();

        // Determine user goal and subscription
        $isTTC = $profile->isTTC();
        $isPremium = $profile->isPremium();
        $userGoal = $profile->user_goal ?? UserGoal::NON_TTC->value;
        $subscriptionType = $profile->subscription_type ?? SubscriptionType::FREE->value;

        // Get matrix message based on user goal
        if ($isTTC) {
            $matrixEngine = new TTCMatrixEngine($user, $locale);
            $matrixMessage = $matrixEngine->getMatrixMessage($phase, $subphase, $cycleDay, $dailyLog, $isFertileWindow);
        } else {
            $matrixEngine = new MatrixMessageEngine($user, $locale);
            $matrixMessage = $matrixEngine->getMatrixMessage($phase, $subphase, $cycleDay, $dailyLog);
        }

        // Get correlations (Layer 3)
        $correlationEngine = new CorrelationEngine($user, $locale);
        $correlations = $correlationEngine->analyzeCorrelations($phase, $subphase, $dailyLog, $isTTC);

        // Filter correlations based on subscription
        if (!$isPremium) {
            $correlations = array_filter($correlations, fn($c) => !($c['is_premium_only'] ?? false));
            $correlations = array_values($correlations);
        }

        // Get patterns (Layer 4 - Premium only)
        $patterns = [];
        if ($isPremium) {
            $patternEngine = new PatternRecognitionEngine($user, $locale);
            $patterns = $patternEngine->analyzePatterns($isTTC);
        }

        // Get nutrition and sleep tips
        $nutritionSleepModule = new NutritionSleepModule($user, $locale);
        $nutritionSleepTips = $nutritionSleepModule->getTips($phase, $subphase, $dailyLog, $isTTC);

        return response()->json([
            'success' => true,
            'data' => [
                'date' => $date->toDateString(),
                'user_goal' => $userGoal,
                'user_goal_label' => UserGoal::tryFrom($userGoal)?->label($locale) ?? $userGoal,
                'subscription_type' => $subscriptionType,
                'subscription_type_label' => SubscriptionType::tryFrom($subscriptionType)?->label($locale) ?? $subscriptionType,
                'cycle_info' => [
                    'phase' => $phase->value,
                    'phase_label' => $phase->label($locale),
                    'subphase' => $subphase->value,
                    'subphase_label' => $subphase->label($locale),
                    'cycle_day' => $cycleDay,
                    'is_fertile_window' => $isFertileWindow,
                    'is_pms_window' => $isPmsWindow,
                    'estimated_ovulation_day' => $cycleData['estimated_ovulation_day'] ?? null,
                    'cycle_length_used' => $cycleData['cycle_length_used'] ?? null,
                ],
                'matrix_message' => $matrixMessage,
                'correlations' => $correlations,
                'patterns' => $patterns,
                'nutrition_sleep_tips' => $nutritionSleepTips,
            ],
        ]);
    }

    /**
     * Update user profile (user_goal and subscription_type)
     */
    public function updateProfile(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');

        $profile = $user->profile;
        if (!$profile) {
            $profile = UserProfile::create([
                'user_id' => $user->id,
            ]);
        }

        $data = $request->only(['user_goal', 'subscription_type']);
        $profile->update($data);
        $profile->refresh();

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'پروفایل با موفقیت بروزرسانی شد' : 'Profile updated successfully',
            'data' => [
                'user_goal' => $profile->user_goal,
                'user_goal_label' => $profile->user_goal ? UserGoal::tryFrom($profile->user_goal)?->label($locale) : null,
                'subscription_type' => $profile->subscription_type,
                'subscription_type_label' => $profile->subscription_type ? SubscriptionType::tryFrom($profile->subscription_type)?->label($locale) : null,
                'is_ttc' => $profile->isTTC(),
                'is_premium' => $profile->isPremium(),
            ],
        ]);
    }

    /**
     * Get current profile status
     */
    public function getProfile(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $profile = $user->profile;

        if (!$profile) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'پروفایل یافت نشد' : 'Profile not found',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'data' => [
                'user_goal' => $profile->user_goal,
                'user_goal_label' => $profile->user_goal ? UserGoal::tryFrom($profile->user_goal)?->label($locale) : null,
                'subscription_type' => $profile->subscription_type,
                'subscription_type_label' => $profile->subscription_type ? SubscriptionType::tryFrom($profile->subscription_type)?->label($locale) : null,
                'is_ttc' => $profile->isTTC(),
                'is_premium' => $profile->isPremium(),
                'last_period_start' => $profile->last_period_start,
                'cycle_length' => $profile->cycle_length,
            ],
        ]);
    }

    /**
     * Create or get test user
     */
    private function getOrCreateTestUser(): User
    {
        $user = User::where('mobile', self::TEST_MOBILE)->first();

        if (!$user) {
            $user = User::create([
                'name' => 'Test User',
                'mobile' => self::TEST_MOBILE,
            ]);
        }

        return $user;
    }
}
