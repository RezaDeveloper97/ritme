<?php

namespace App\Http\Controllers;

use App\Enums\AlertLevel;
use App\Enums\BloodType;
use App\Enums\ConfidenceLevel;
use App\Enums\FetalMovementStatus;
use App\Enums\MentalHealthStatus;
use App\Enums\PreExistingCondition;
use App\Enums\PregnancyAgeSource;
use App\Enums\RhFactor;
use App\Enums\SwellingLocation;
use App\Enums\SymptomSeverity;
use App\Models\PregnancyAlert;
use App\Models\PregnancyFetalMovement;
use App\Models\PregnancyProfile;
use App\Models\PregnancySymptomLog;
use App\Models\PregnancyWeeklyContent;
use App\Models\PregnancyWeeklyLog;
use App\Models\User;
use App\Services\PregnancyEngine\PregnancyAlertService;
use App\Services\PregnancyEngine\PregnancyCalculationService;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class TestPregnancyController extends Controller
{
    // شماره تست برای کاربر پیش‌فرض
    private const TEST_MOBILE = '09123456789';

    /**
     * نمایش صفحه تست بارداری
     */
    public function index()
    {
        $user = $this->getOrCreateTestUser();
        $profile = $user->pregnancyProfile;

        return view('test-pregnancy', [
            'user' => $user,
            'profile' => $profile,
        ]);
    }

    /**
     * دریافت enum های بارداری
     */
    public function getEnums(Request $request): JsonResponse
    {
        $locale = $request->input('locale', 'fa');

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

        $severity = collect(SymptomSeverity::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $swellingLocations = collect(SwellingLocation::cases())->map(fn($s) => [
            'value' => $s->value,
            'label' => $s->label($locale),
        ])->values();

        $moodStatus = collect(MentalHealthStatus::cases())->map(fn($m) => [
            'value' => $m->value,
            'label' => $m->label($locale),
        ])->values();

        $fetalMovementStatus = collect(FetalMovementStatus::cases())->map(fn($f) => [
            'value' => $f->value,
            'label' => $f->label($locale),
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
                'severity' => $severity,
                'swelling_locations' => $swellingLocations,
                'overall_mood' => $moodStatus,
                'fetal_movement_status' => $fetalMovementStatus,
            ],
        ]);
    }

    // ==================== Pregnancy Profile ====================

    /**
     * فعال‌سازی حالت بارداری
     */
    public function activatePregnancy(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');

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
     * غیرفعال‌سازی حالت بارداری
     */
    public function deactivatePregnancy(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');

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
     * تکمیل آنبوردینگ بارداری
     */
    public function onboarding(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $data = $request->all();

        // Determine confidence level based on source
        $confidenceLevel = match ($data['age_source'] ?? '') {
            PregnancyAgeSource::ULTRASOUND->value => ConfidenceLevel::HIGH->value,
            PregnancyAgeSource::LMP->value => ConfidenceLevel::MEDIUM->value,
            PregnancyAgeSource::MANUAL->value => ConfidenceLevel::LOW->value,
            default => ConfidenceLevel::LOW->value,
        };

        // Set uncertainty days based on source
        $uncertaintyDays = match ($data['age_source'] ?? '') {
            PregnancyAgeSource::ULTRASOUND->value => 1,
            PregnancyAgeSource::LMP->value => 3,
            PregnancyAgeSource::MANUAL->value => 5,
            default => 5,
        };

        // If manual entry, set the entry date to today
        if (($data['age_source'] ?? '') === PregnancyAgeSource::MANUAL->value) {
            $data['manual_entry_date'] = Carbon::today()->toDateString();
        }

        // Check RH factor for special care flag
        $rhNegativeCareFlag = isset($data['rh_factor']) && $data['rh_factor'] === RhFactor::NEGATIVE->value;

        // Create or update profile
        $profile = PregnancyProfile::updateOrCreate(
            ['user_id' => $user->id],
            array_merge($data, [
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
        ]);
    }

    /**
     * دریافت پروفایل بارداری
     */
    public function getProfile(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
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
     * بروزرسانی پروفایل بارداری
     */
    public function updateProfile(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $profile = $user->pregnancyProfile;

        if (!$profile) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'پروفایل بارداری یافت نشد' : 'Pregnancy profile not found',
            ], 404);
        }

        $data = $request->except(['locale']);

        // Handle age source change
        if (isset($data['age_source'])) {
            $data['confidence_level'] = match ($data['age_source']) {
                PregnancyAgeSource::ULTRASOUND->value => ConfidenceLevel::HIGH->value,
                PregnancyAgeSource::LMP->value => ConfidenceLevel::MEDIUM->value,
                PregnancyAgeSource::MANUAL->value => ConfidenceLevel::LOW->value,
                default => $profile->confidence_level,
            };

            $data['uncertainty_days'] = match ($data['age_source']) {
                PregnancyAgeSource::ULTRASOUND->value => 1,
                PregnancyAgeSource::LMP->value => 3,
                PregnancyAgeSource::MANUAL->value => 5,
                default => $profile->uncertainty_days,
            };

            if ($data['age_source'] === PregnancyAgeSource::MANUAL->value && !isset($data['manual_entry_date'])) {
                $data['manual_entry_date'] = Carbon::today()->toDateString();
            }
        }

        // Check RH factor
        if (isset($data['rh_factor'])) {
            $data['rh_negative_care_flag'] = $data['rh_factor'] === RhFactor::NEGATIVE->value;
        }

        $profile->update($data);

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
     * دریافت وضعیت بارداری
     */
    public function getStatus(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');

        $calculationService = new PregnancyCalculationService($user, $locale);
        $status = $calculationService->getPregnancyStatus();

        return response()->json([
            'success' => true,
            'data' => $status,
        ]);
    }

    // ==================== Pregnancy Symptoms ====================

    /**
     * دریافت لیست علائم
     */
    public function getSymptoms(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $from = $request->input('from');
        $to = $request->input('to');

        $query = PregnancySymptomLog::where('user_id', $user->id)
            ->orderBy('log_date', 'desc');

        if ($from) {
            $query->where('log_date', '>=', $from);
        }
        if ($to) {
            $query->where('log_date', '<=', $to);
        }

        $logs = $query->get();

        return response()->json([
            'success' => true,
            'data' => [
                'logs' => $logs,
                'count' => $logs->count(),
            ],
        ]);
    }

    /**
     * ذخیره علائم روزانه
     */
    public function storeSymptom(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $data = $request->except(['locale']);

        $log = PregnancySymptomLog::updateOrCreate(
            [
                'user_id' => $user->id,
                'log_date' => $data['log_date'],
            ],
            $data
        );

        // Process alerts
        $alertService = new PregnancyAlertService($user, $locale);
        $alerts = $alertService->processSymptomLog($log);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'علائم با موفقیت ذخیره شد' : 'Symptom log saved successfully',
            'data' => [
                'log' => $log,
                'alerts' => $alerts,
            ],
        ]);
    }

    /**
     * دریافت علائم یک تاریخ
     */
    public function getSymptomByDate(string $date): JsonResponse
    {
        $user = $this->getOrCreateTestUser();

        $log = PregnancySymptomLog::where('user_id', $user->id)
            ->whereDate('log_date', $date)
            ->first();

        if (!$log) {
            return response()->json([
                'success' => false,
                'message' => 'گزارش علائم یافت نشد',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'data' => [
                'log' => $log,
            ],
        ]);
    }

    /**
     * حذف علائم یک تاریخ
     */
    public function deleteSymptom(string $date): JsonResponse
    {
        $user = $this->getOrCreateTestUser();

        $deleted = PregnancySymptomLog::where('user_id', $user->id)
            ->whereDate('log_date', $date)
            ->delete();

        if (!$deleted) {
            return response()->json([
                'success' => false,
                'message' => 'گزارش علائم یافت نشد',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'message' => 'گزارش علائم حذف شد',
        ]);
    }

    // ==================== Pregnancy Weekly ====================

    /**
     * دریافت لیست گزارشات هفتگی
     */
    public function getWeeklyLogs(): JsonResponse
    {
        $user = $this->getOrCreateTestUser();

        $logs = PregnancyWeeklyLog::where('user_id', $user->id)
            ->orderBy('pregnancy_week', 'desc')
            ->get();

        return response()->json([
            'success' => true,
            'data' => [
                'logs' => $logs,
                'count' => $logs->count(),
            ],
        ]);
    }

    /**
     * ذخیره گزارش هفتگی
     */
    public function storeWeeklyLog(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $data = $request->except(['locale']);

        $log = PregnancyWeeklyLog::updateOrCreate(
            [
                'user_id' => $user->id,
                'pregnancy_week' => $data['pregnancy_week'],
            ],
            $data
        );

        // Process alerts
        $alertService = new PregnancyAlertService($user, $locale);
        $alerts = $alertService->processWeeklyLog($log);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'گزارش هفتگی با موفقیت ذخیره شد' : 'Weekly log saved successfully',
            'data' => [
                'log' => $log,
                'alerts' => $alerts,
            ],
        ]);
    }

    /**
     * دریافت گزارش هفتگی
     */
    public function getWeeklyLog(int $week): JsonResponse
    {
        $user = $this->getOrCreateTestUser();

        $log = PregnancyWeeklyLog::where('user_id', $user->id)
            ->where('pregnancy_week', $week)
            ->first();

        if (!$log) {
            return response()->json([
                'success' => false,
                'message' => 'گزارش هفتگی یافت نشد',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'data' => [
                'log' => $log,
            ],
        ]);
    }

    /**
     * دریافت محتوای هفتگی
     */
    public function getWeeklyContent(Request $request, int $week): JsonResponse
    {
        $locale = $request->input('locale', 'fa');

        if ($week < 1 || $week > 40) {
            return response()->json([
                'success' => false,
                'message' => 'Invalid week number. Must be between 1 and 40.',
            ], 422);
        }

        $content = PregnancyWeeklyContent::getByWeek($week);

        if (!$content) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'محتوای این هفته یافت نشد' : 'Content not found for this week',
            ], 404);
        }

        $localizedContent = [
            'week_number' => $content->week_number,
            'fetal_development' => $content->getLocalizedContent('fetal_development', $locale),
            'mother_body_changes' => $content->getLocalizedContent('mother_body_changes', $locale),
            'dos_and_donts' => $content->getLocalizedContent('dos_and_donts', $locale),
            'care_plan' => $content->getLocalizedContent('care_plan', $locale),
            'body_adaptation' => $content->getLocalizedContent('body_adaptation', $locale),
            'emotional_status' => $content->getLocalizedContent('emotional_status', $locale),
            'key_nutrition' => $content->getLocalizedContent('key_nutrition', $locale),
            'physical_activity' => $content->getLocalizedContent('physical_activity', $locale),
            'tests_and_checkups' => $content->getLocalizedContent('tests_and_checkups', $locale),
            'faq' => $content->getLocalizedContent('faq', $locale),
        ];

        return response()->json([
            'success' => true,
            'data' => [
                'week' => $week,
                'content' => $localizedContent,
            ],
        ]);
    }

    // ==================== Fetal Movement ====================

    /**
     * دریافت لیست حرکات جنین
     */
    public function getFetalMovements(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $from = $request->input('from');
        $to = $request->input('to');

        $query = PregnancyFetalMovement::where('user_id', $user->id)
            ->orderBy('log_date', 'desc');

        if ($from) {
            $query->where('log_date', '>=', $from);
        }
        if ($to) {
            $query->where('log_date', '<=', $to);
        }

        $logs = $query->get();

        return response()->json([
            'success' => true,
            'data' => [
                'logs' => $logs,
                'count' => $logs->count(),
            ],
        ]);
    }

    /**
     * ذخیره حرکات جنین
     */
    public function storeFetalMovement(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');
        $data = $request->except(['locale']);

        $movement = PregnancyFetalMovement::updateOrCreate(
            [
                'user_id' => $user->id,
                'log_date' => $data['log_date'],
            ],
            $data
        );

        // Update pregnancy profile if first movement detected
        if (in_array($data['movement_status'] ?? '', ['felt', 'normal', 'increased'])) {
            $profile = $user->pregnancyProfile;
            if ($profile && !$profile->fetal_movement_felt) {
                $profile->update([
                    'fetal_movement_felt' => true,
                    'first_fetal_movement_date' => $profile->first_fetal_movement_date ?? $data['log_date'],
                ]);
            }
        }

        // Process alerts
        $alertService = new PregnancyAlertService($user, $locale);
        $alerts = $alertService->processFetalMovement($movement);

        return response()->json([
            'success' => true,
            'message' => $locale === 'fa' ? 'حرکات جنین ثبت شد' : 'Fetal movement logged successfully',
            'data' => [
                'log' => $movement,
                'alerts' => $alerts,
            ],
        ]);
    }

    // ==================== Alerts ====================

    /**
     * دریافت لیست هشدارها
     */
    public function getAlerts(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $level = $request->input('level');
        $unreadOnly = $request->boolean('unread_only');

        $query = PregnancyAlert::where('user_id', $user->id)
            ->active()
            ->orderBy('created_at', 'desc');

        if ($level && in_array($level, AlertLevel::values())) {
            $query->level($level);
        }

        if ($unreadOnly) {
            $query->unread();
        }

        $alerts = $query->get();

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
     * دریافت خلاصه هشدارها
     */
    public function getAlertsSummary(Request $request): JsonResponse
    {
        $user = $this->getOrCreateTestUser();
        $locale = $request->input('locale', 'fa');

        $alertService = new PregnancyAlertService($user, $locale);
        $emergencyCount = $alertService->getEmergencyAlertsCount();
        $unreadAlerts = $alertService->getUnreadAlerts();

        $counts = [
            'total' => PregnancyAlert::where('user_id', $user->id)->active()->count(),
            'unread' => $unreadAlerts->count(),
            'emergency' => $emergencyCount,
            'warning' => PregnancyAlert::where('user_id', $user->id)->active()->level(AlertLevel::WARNING->value)->count(),
            'info' => PregnancyAlert::where('user_id', $user->id)->active()->level(AlertLevel::INFO->value)->count(),
        ];

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

    /**
     * علامت‌گذاری هشدار به عنوان خوانده شده
     */
    public function markAlertAsRead(int $id): JsonResponse
    {
        $user = $this->getOrCreateTestUser();

        $alert = PregnancyAlert::where('user_id', $user->id)
            ->where('id', $id)
            ->first();

        if (!$alert) {
            return response()->json([
                'success' => false,
                'message' => 'هشدار یافت نشد',
            ], 404);
        }

        $alert->markAsRead();

        return response()->json([
            'success' => true,
            'message' => 'هشدار خوانده شد',
        ]);
    }

    /**
     * علامت‌گذاری همه هشدارها به عنوان خوانده شده
     */
    public function markAllAlertsAsRead(): JsonResponse
    {
        $user = $this->getOrCreateTestUser();

        $count = PregnancyAlert::where('user_id', $user->id)
            ->unread()
            ->update([
                'is_read' => true,
                'read_at' => now(),
            ]);

        return response()->json([
            'success' => true,
            'message' => 'همه هشدارها خوانده شدند',
            'data' => [
                'count' => $count,
            ],
        ]);
    }

    /**
     * رد کردن هشدار
     */
    public function dismissAlert(int $id): JsonResponse
    {
        $user = $this->getOrCreateTestUser();

        $alert = PregnancyAlert::where('user_id', $user->id)
            ->where('id', $id)
            ->first();

        if (!$alert) {
            return response()->json([
                'success' => false,
                'message' => 'هشدار یافت نشد',
            ], 404);
        }

        $alert->dismiss();

        return response()->json([
            'success' => true,
            'message' => 'هشدار رد شد',
        ]);
    }

    // ==================== Helper Methods ====================

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
}
