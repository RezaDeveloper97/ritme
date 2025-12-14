<?php

namespace App\Http\Controllers;

use App\Models\User;
use App\Models\UserProfile;
use App\Services\HealthEngine\HealthDataEngine;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class TestPageController extends Controller
{
    // شماره تست برای کاربر پیش‌فرض
    private const TEST_MOBILE = '09123456789';

    /**
     * نمایش صفحه تست
     */
    public function index()
    {
        $user = $this->getOrCreateTestUser();
        $profile = $user->profile;

        return view('test-page', [
            'user' => $user,
            'profile' => $profile,
        ]);
    }

    /**
     * دریافت پروفایل کاربر تست
     */
    public function getProfile(): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
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
     * بروزرسانی پروفایل کاربر تست
     */
    public function updateProfile(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();

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

        // Default last_period_start to today if not provided and profile doesn't have one
        if (!isset($profileData['last_period_start']) && !$profile->last_period_start) {
            $profileData['last_period_start'] = now()->toDateString();
        }

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

    /**
     * دریافت داده‌های سایکل برای ماه‌های انتخابی
     */
    public function getCycleData(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');

        // ماه‌های انتخاب شده (آرایه‌ای از year-month)
        $months = $request->input('months', []);

        if (empty($months)) {
            // اگر ماهی انتخاب نشده، ماه جاری و 2 ماه بعد را بگیر
            $now = Carbon::now();
            $months = [
                $now->format('Y-m'),
                $now->copy()->addMonth()->format('Y-m'),
                $now->copy()->addMonths(2)->format('Y-m'),
            ];
        }

        $result = [];
        $engine = new HealthDataEngine($user, $locale);

        foreach ($months as $monthStr) {
            [$year, $month] = explode('-', $monthStr);
            $year = (int) $year;
            $month = (int) $month;

            $startDate = Carbon::createFromDate($year, $month, 1)->startOfMonth();
            $endDate = $startDate->copy()->endOfMonth();

            // Always calculate fresh to ensure correct cycle_day values
            $monthData = [];
            $currentDate = $startDate->copy();

            while ($currentDate <= $endDate) {
                $dateStr = $currentDate->toDateString();
                $monthData[$dateStr] = $engine->calculateForDate($currentDate);
                $currentDate->addDay();
            }

            $result[$monthStr] = [
                'year' => $year,
                'month' => $month,
                'month_name' => $this->getMonthName($month, $locale),
                'days' => $monthData,
            ];
        }

        return response()->json([
            'success' => true,
            'data' => $result,
        ]);
    }

    /**
     * ایجاد یا دریافت کاربر تست
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

    /**
     * Localize calculation text fields
     */
    private function localizeCalculation(array $calculation, string $locale): array
    {
        if (isset($calculation['text_flags']) && is_array($calculation['text_flags'])) {
            $localizedFlags = [];
            foreach ($calculation['text_flags'] as $key => $value) {
                if (is_array($value) && isset($value[$locale])) {
                    $localizedFlags[$key] = $value[$locale];
                } else {
                    $localizedFlags[$key] = $value;
                }
            }
            $calculation['text_flags'] = $localizedFlags;
        }

        if (isset($calculation['daily_tips']) && is_array($calculation['daily_tips'])) {
            $localizedTips = [];
            foreach ($calculation['daily_tips'] as $tip) {
                if (is_array($tip) && isset($tip[$locale])) {
                    $localizedTips[] = [
                        'type' => $tip['type'] ?? 'general',
                        'text' => $tip[$locale],
                    ];
                } elseif (is_array($tip) && isset($tip['en'])) {
                    $localizedTips[] = [
                        'type' => $tip['type'] ?? 'general',
                        'text' => $tip['en'],
                    ];
                }
            }
            $calculation['daily_tips'] = $localizedTips;
        }

        return $calculation;
    }

    /**
     * دریافت نام ماه بر اساس زبان
     */
    private function getMonthName(int $month, string $locale): string
    {
        $persianMonths = [
            1 => 'ژانویه', 2 => 'فوریه', 3 => 'مارس', 4 => 'آوریل',
            5 => 'مه', 6 => 'ژوئن', 7 => 'ژوئیه', 8 => 'آگوست',
            9 => 'سپتامبر', 10 => 'اکتبر', 11 => 'نوامبر', 12 => 'دسامبر',
        ];

        $englishMonths = [
            1 => 'January', 2 => 'February', 3 => 'March', 4 => 'April',
            5 => 'May', 6 => 'June', 7 => 'July', 8 => 'August',
            9 => 'September', 10 => 'October', 11 => 'November', 12 => 'December',
        ];

        return $locale === 'fa' ? $persianMonths[$month] : $englishMonths[$month];
    }
}
