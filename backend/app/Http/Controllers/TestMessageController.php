<?php

namespace App\Http\Controllers;

use App\Enums\SubscriptionType;
use App\Enums\UserGoal;
use App\Models\User;
use App\Models\UserProfile;
use App\Services\MessageSystem\Core\MessageManager;
use App\Services\MessageSystem\Enums\MessageMode;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class TestMessageController extends Controller
{
    // Test mobile number for default user
    private const TEST_MOBILE = '09123456789';

    /**
     * Display message test page
     */
    public function index()
    {
        $user = $this->getOrCreateTestUser();
        $profile = $user->profile;
        $pregnancyProfile = $user->pregnancyProfile;

        return view('test-message', [
            'user' => $user,
            'profile' => $profile,
            'pregnancyProfile' => $pregnancyProfile,
        ]);
    }

    /**
     * Get message system enums
     */
    public function getEnums(Request $request): JsonResponse
    {
        $locale = $request->input('locale', 'fa');
        $user = $this->getOrCreateTestUser();

        $manager = new MessageManager($user, $locale);
        $enums = $manager->getEnums();

        return response()->json([
            'success' => true,
            'data' => $enums,
        ]);
    }

    /**
     * Get daily messages
     */
    public function getDailyMessages(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $profile = $user->profile;

        // Get date parameter
        $date = $request->input('date')
            ? Carbon::parse($request->input('date'))
            : Carbon::today();

        // Get optional mode override
        $forceMode = null;
        if ($request->input('mode')) {
            $forceMode = MessageMode::tryFrom($request->input('mode'));
        }

        // Create message manager
        $manager = new MessageManager($user, $locale);

        // Check if user has required profile data
        $detectedMode = $forceMode ?? $manager->detectMode();

        if ($detectedMode === MessageMode::CYCLE) {
            if (!$profile || !$profile->last_period_start) {
                return response()->json([
                    'success' => false,
                    'message' => $locale === 'fa'
                        ? 'لطفاً ابتدا پروفایل خود را تکمیل کنید (تاریخ آخرین پریود را در صفحه تست اصلی وارد کنید)'
                        : 'Please complete your profile first (enter last period date in the main test page)',
                ], 400);
            }
        }

        if ($detectedMode === MessageMode::PREGNANCY) {
            $pregnancyProfile = $user->pregnancyProfile;
            if (!$pregnancyProfile || !$pregnancyProfile->onboarding_completed) {
                return response()->json([
                    'success' => false,
                    'message' => $locale === 'fa'
                        ? 'لطفاً ابتدا آنبوردینگ بارداری را تکمیل کنید (در صفحه تست بارداری)'
                        : 'Please complete pregnancy onboarding first (in pregnancy test page)',
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
     * Get current mode
     */
    public function getMode(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $profile = $user->profile;

        $manager = new MessageManager($user, $locale);
        $mode = $manager->detectMode();

        return response()->json([
            'success' => true,
            'data' => [
                'mode' => $mode->value,
                'mode_label' => $mode->label($locale),
                'user_goal' => $profile?->user_goal ?? 'non_ttc',
                'user_goal_label' => $profile?->user_goal
                    ? UserGoal::tryFrom($profile->user_goal)?->label($locale)
                    : null,
                'subscription_type' => $profile?->subscription_type ?? 'free',
                'subscription_type_label' => $profile?->subscription_type
                    ? SubscriptionType::tryFrom($profile->subscription_type)?->label($locale)
                    : null,
                'is_ttc' => $profile?->isTTC() ?? false,
                'is_premium' => $profile?->isPremium() ?? false,
                'last_period_start' => $profile?->last_period_start,
                'cycle_length' => $profile?->cycle_length,
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
                'user_goal_label' => $profile->user_goal
                    ? UserGoal::tryFrom($profile->user_goal)?->label($locale)
                    : null,
                'subscription_type' => $profile->subscription_type,
                'subscription_type_label' => $profile->subscription_type
                    ? SubscriptionType::tryFrom($profile->subscription_type)?->label($locale)
                    : null,
                'is_ttc' => $profile->isTTC(),
                'is_premium' => $profile->isPremium(),
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
