<?php

namespace App\Services\PregnancyEngine;

use App\Enums\AlertLevel;
use App\Models\PregnancyAlert;
use App\Models\PregnancyFetalMovement;
use App\Models\PregnancyProfile;
use App\Models\PregnancySymptomLog;
use App\Models\PregnancyWeeklyLog;
use App\Models\User;
use Carbon\Carbon;

class PregnancyAlertService
{
    private User $user;
    private ?PregnancyProfile $profile;
    private string $locale;

    public function __construct(User $user, string $locale = 'en')
    {
        $this->user = $user;
        $this->profile = $user->pregnancyProfile;
        $this->locale = $locale;
    }

    /**
     * Process symptom log and generate alerts
     */
    public function processSymptomLog(PregnancySymptomLog $log): array
    {
        $alerts = [];
        $calculationService = new PregnancyCalculationService($this->user, $this->locale);
        $currentWeek = $calculationService->getCurrentWeek();

        // Check for critical symptoms
        if ($log->hasCriticalSymptoms()) {
            $criticalSymptoms = $log->getCriticalSymptoms();

            foreach ($criticalSymptoms as $symptom => $severity) {
                $alert = $this->createSymptomAlert($symptom, $severity, $currentWeek);
                if ($alert) {
                    $alerts[] = $alert;
                }
            }
        }

        // Check for combined severe symptoms
        $severeSymptomCount = $this->countSevereSymptoms($log);
        if ($severeSymptomCount >= 3) {
            $alerts[] = $this->createMultipleSymptomsAlert($currentWeek);
        }

        return $alerts;
    }

    /**
     * Create alert for specific critical symptom
     */
    private function createSymptomAlert(string $symptom, ?string $severity, int $currentWeek): ?PregnancyAlert
    {
        $alertData = $this->getSymptomAlertData($symptom, $severity, $currentWeek);

        if (!$alertData) {
            return null;
        }

        return PregnancyAlert::create([
            'user_id' => $this->user->id,
            'alert_level' => $alertData['level'],
            'alert_type' => 'symptom_based',
            'title' => $alertData['title'],
            'message' => $alertData['message'],
            'pregnancy_week' => $currentWeek,
            'trigger_symptoms' => [$symptom => $severity],
            'medical_history_flags' => $this->getMedicalHistoryFlags(),
            'recommended_actions' => $alertData['actions'],
        ]);
    }

    /**
     * Get alert data based on symptom type and severity
     */
    private function getSymptomAlertData(string $symptom, ?string $severity, int $currentWeek): ?array
    {
        $isHighRisk = $this->isHighRisk();
        $locale = $this->locale;

        // Bleeding - Always serious
        if ($symptom === 'bleeding') {
            if ($severity === 'severe' || $isHighRisk) {
                return [
                    'level' => AlertLevel::EMERGENCY->value,
                    'title' => $locale === 'fa' ? 'اورژانس: خونریزی شدید' : 'Emergency: Heavy Bleeding',
                    'message' => $locale === 'fa'
                        ? 'خونریزی شدید در دوران بارداری نیاز به توجه فوری پزشکی دارد. لطفاً فوراً به اورژانس مراجعه کنید.'
                        : 'Heavy bleeding during pregnancy requires immediate medical attention. Please go to emergency immediately.',
                    'actions' => $locale === 'fa'
                        ? ['فوراً به اورژانس مراجعه کنید', 'با پزشک تماس بگیرید']
                        : ['Go to emergency immediately', 'Call your doctor'],
                ];
            }

            return [
                'level' => AlertLevel::WARNING->value,
                'title' => $locale === 'fa' ? 'هشدار: خونریزی' : 'Warning: Bleeding',
                'message' => $locale === 'fa'
                    ? 'خونریزی گزارش شده است. لطفاً با پزشک خود تماس بگیرید.'
                    : 'Bleeding has been reported. Please contact your doctor.',
                'actions' => $locale === 'fa'
                    ? ['با پزشک تماس بگیرید', 'استراحت کنید']
                    : ['Contact your doctor', 'Rest'],
            ];
        }

        // Spotting
        if ($symptom === 'spotting') {
            // First trimester spotting is more concerning
            if ($currentWeek <= 12 || $severity === 'severe') {
                return [
                    'level' => AlertLevel::WARNING->value,
                    'title' => $locale === 'fa' ? 'هشدار: لکه‌بینی' : 'Warning: Spotting',
                    'message' => $locale === 'fa'
                        ? 'لکه‌بینی گزارش شده است. لطفاً وضعیت را پیگیری کنید و در صورت تداوم با پزشک مشورت کنید.'
                        : 'Spotting has been reported. Please monitor and consult your doctor if it continues.',
                    'actions' => $locale === 'fa'
                        ? ['وضعیت را پیگیری کنید', 'استراحت کنید', 'در صورت تداوم با پزشک تماس بگیرید']
                        : ['Monitor the situation', 'Rest', 'Contact doctor if it continues'],
                ];
            }

            return [
                'level' => AlertLevel::INFO->value,
                'title' => $locale === 'fa' ? 'توجه: لکه‌بینی خفیف' : 'Note: Light Spotting',
                'message' => $locale === 'fa'
                    ? 'لکه‌بینی خفیف گزارش شده است. این می‌تواند طبیعی باشد اما پیگیری کنید.'
                    : 'Light spotting has been reported. This can be normal but please monitor.',
                'actions' => $locale === 'fa'
                    ? ['وضعیت را پیگیری کنید', 'در صورت افزایش با پزشک تماس بگیرید']
                    : ['Monitor the situation', 'Contact doctor if it increases'],
            ];
        }

        // Fluid leakage - Always serious
        if ($symptom === 'fluid_leakage') {
            return [
                'level' => AlertLevel::EMERGENCY->value,
                'title' => $locale === 'fa' ? 'اورژانس: خروج مایع' : 'Emergency: Fluid Leakage',
                'message' => $locale === 'fa'
                    ? 'خروج مایع ممکن است نشانه پارگی کیسه آب باشد. لطفاً فوراً به بیمارستان مراجعه کنید.'
                    : 'Fluid leakage may indicate ruptured membranes. Please go to hospital immediately.',
                'actions' => $locale === 'fa'
                    ? ['فوراً به بیمارستان مراجعه کنید', 'حرکت نکنید و دراز بکشید']
                    : ['Go to hospital immediately', 'Lie down and avoid movement'],
            ];
        }

        // Severe/sudden pain
        if ($symptom === 'severe_sudden_pain') {
            return [
                'level' => AlertLevel::EMERGENCY->value,
                'title' => $locale === 'fa' ? 'اورژانس: درد شدید' : 'Emergency: Severe Pain',
                'message' => $locale === 'fa'
                    ? 'درد شدید و ناگهانی نیاز به بررسی فوری پزشکی دارد.'
                    : 'Severe sudden pain requires immediate medical evaluation.',
                'actions' => $locale === 'fa'
                    ? ['فوراً به اورژانس مراجعه کنید', 'با اورژانس ۱۱۵ تماس بگیرید']
                    : ['Go to emergency immediately', 'Call emergency services'],
            ];
        }

        return null;
    }

    /**
     * Count severe symptoms in a log
     */
    private function countSevereSymptoms(PregnancySymptomLog $log): int
    {
        $count = 0;
        $severityFields = [
            'nausea_severity', 'vomiting_severity', 'fatigue_severity',
            'headache_severity', 'dizziness_severity', 'breast_pain_severity',
            'lower_abdominal_pain_severity', 'cramping_severity',
            'back_pain_severity', 'pelvic_pressure_severity',
        ];

        foreach ($severityFields as $field) {
            if ($log->{$field} === 'severe') {
                $count++;
            }
        }

        return $count;
    }

    /**
     * Create alert for multiple severe symptoms
     */
    private function createMultipleSymptomsAlert(int $currentWeek): PregnancyAlert
    {
        $locale = $this->locale;

        return PregnancyAlert::create([
            'user_id' => $this->user->id,
            'alert_level' => AlertLevel::WARNING->value,
            'alert_type' => 'symptom_based',
            'title' => $locale === 'fa' ? 'هشدار: علائم متعدد شدید' : 'Warning: Multiple Severe Symptoms',
            'message' => $locale === 'fa'
                ? 'چندین علامت شدید گزارش شده است. لطفاً با پزشک خود مشورت کنید.'
                : 'Multiple severe symptoms reported. Please consult your doctor.',
            'pregnancy_week' => $currentWeek,
            'trigger_symptoms' => ['multiple_severe' => true],
            'medical_history_flags' => $this->getMedicalHistoryFlags(),
            'recommended_actions' => $locale === 'fa'
                ? ['با پزشک تماس بگیرید', 'استراحت کنید', 'علائم را پیگیری کنید']
                : ['Contact your doctor', 'Rest', 'Monitor symptoms'],
        ]);
    }

    /**
     * Process weekly log and generate alerts
     */
    public function processWeeklyLog(PregnancyWeeklyLog $log): array
    {
        $alerts = [];
        $locale = $this->locale;
        $calculationService = new PregnancyCalculationService($this->user, $this->locale);
        $currentWeek = $calculationService->getCurrentWeek();

        // Check blood pressure
        if ($log->hasHighBloodPressure()) {
            $alerts[] = PregnancyAlert::create([
                'user_id' => $this->user->id,
                'alert_level' => AlertLevel::WARNING->value,
                'alert_type' => 'symptom_based',
                'title' => $locale === 'fa' ? 'هشدار: فشار خون بالا' : 'Warning: High Blood Pressure',
                'message' => $locale === 'fa'
                    ? "فشار خون شما ({$log->systolic_pressure}/{$log->diastolic_pressure}) بالاتر از حد طبیعی است. لطفاً با پزشک مشورت کنید."
                    : "Your blood pressure ({$log->systolic_pressure}/{$log->diastolic_pressure}) is above normal. Please consult your doctor.",
                'pregnancy_week' => $currentWeek,
                'trigger_symptoms' => [
                    'systolic' => $log->systolic_pressure,
                    'diastolic' => $log->diastolic_pressure,
                ],
                'recommended_actions' => $locale === 'fa'
                    ? ['با پزشک تماس بگیرید', 'استراحت کنید', 'مصرف نمک را کاهش دهید']
                    : ['Contact your doctor', 'Rest', 'Reduce salt intake'],
            ]);
        }

        // Check blood sugar
        if ($log->hasHighBloodSugar()) {
            $alerts[] = PregnancyAlert::create([
                'user_id' => $this->user->id,
                'alert_level' => AlertLevel::WARNING->value,
                'alert_type' => 'symptom_based',
                'title' => $locale === 'fa' ? 'هشدار: قند خون بالا' : 'Warning: High Blood Sugar',
                'message' => $locale === 'fa'
                    ? 'سطح قند خون شما بالاتر از حد طبیعی است. لطفاً با پزشک مشورت کنید.'
                    : 'Your blood sugar level is above normal. Please consult your doctor.',
                'pregnancy_week' => $currentWeek,
                'trigger_symptoms' => [
                    'fasting' => $log->fasting_blood_sugar,
                    'post_meal' => $log->post_meal_blood_sugar,
                ],
                'recommended_actions' => $locale === 'fa'
                    ? ['با پزشک تماس بگیرید', 'رژیم غذایی را بررسی کنید']
                    : ['Contact your doctor', 'Review your diet'],
            ]);
        }

        // Check for severe depression
        if ($log->has_depression_feelings && $log->depression_severity === 'severe') {
            $alerts[] = PregnancyAlert::create([
                'user_id' => $this->user->id,
                'alert_level' => AlertLevel::WARNING->value,
                'alert_type' => 'symptom_based',
                'title' => $locale === 'fa' ? 'توجه: وضعیت روحی' : 'Note: Mental Health',
                'message' => $locale === 'fa'
                    ? 'احساسات افسردگی شدید گزارش شده است. صحبت با پزشک یا مشاور می‌تواند کمک‌کننده باشد.'
                    : 'Severe depression feelings reported. Talking to a doctor or counselor can be helpful.',
                'pregnancy_week' => $currentWeek,
                'recommended_actions' => $locale === 'fa'
                    ? ['با پزشک یا مشاور صحبت کنید', 'از حمایت خانواده بهره بگیرید']
                    : ['Talk to a doctor or counselor', 'Seek family support'],
            ]);
        }

        return $alerts;
    }

    /**
     * Process fetal movement and generate alerts
     */
    public function processFetalMovement(PregnancyFetalMovement $movement): array
    {
        $alerts = [];
        $locale = $this->locale;
        $calculationService = new PregnancyCalculationService($this->user, $this->locale);
        $currentWeek = $calculationService->getCurrentWeek();

        // After week 24, reduced or no movement is concerning
        if ($currentWeek >= 24 && $movement->requiresAttention()) {
            $level = $movement->movement_status === 'none'
                ? AlertLevel::EMERGENCY->value
                : AlertLevel::WARNING->value;

            $title = $movement->movement_status === 'none'
                ? ($locale === 'fa' ? 'اورژانس: عدم حرکت جنین' : 'Emergency: No Fetal Movement')
                : ($locale === 'fa' ? 'هشدار: کاهش حرکت جنین' : 'Warning: Reduced Fetal Movement');

            $message = $movement->movement_status === 'none'
                ? ($locale === 'fa'
                    ? 'عدم حرکت جنین گزارش شده است. لطفاً فوراً با پزشک تماس بگیرید یا به بیمارستان مراجعه کنید.'
                    : 'No fetal movement reported. Please contact your doctor immediately or go to hospital.')
                : ($locale === 'fa'
                    ? 'کاهش حرکات جنین گزارش شده است. لطفاً با پزشک خود مشورت کنید.'
                    : 'Reduced fetal movement reported. Please consult your doctor.');

            $alerts[] = PregnancyAlert::create([
                'user_id' => $this->user->id,
                'alert_level' => $level,
                'alert_type' => 'symptom_based',
                'title' => $title,
                'message' => $message,
                'pregnancy_week' => $currentWeek,
                'trigger_symptoms' => ['fetal_movement' => $movement->movement_status],
                'recommended_actions' => $movement->movement_status === 'none'
                    ? ($locale === 'fa'
                        ? ['فوراً به بیمارستان مراجعه کنید', 'با پزشک تماس بگیرید']
                        : ['Go to hospital immediately', 'Contact your doctor'])
                    : ($locale === 'fa'
                        ? ['با پزشک تماس بگیرید', 'دراز بکشید و حرکات را پیگیری کنید']
                        : ['Contact your doctor', 'Lie down and monitor movements']),
            ]);
        }

        return $alerts;
    }

    /**
     * Create missing data reminder alert
     */
    public function createMissingDataAlert(string $dataType, int $daysMissing): PregnancyAlert
    {
        $locale = $this->locale;
        $calculationService = new PregnancyCalculationService($this->user, $this->locale);
        $currentWeek = $calculationService->getCurrentWeek();

        $title = match ($dataType) {
            'symptoms' => $locale === 'fa' ? 'یادآوری: ثبت علائم' : 'Reminder: Log Symptoms',
            'fetal_movement' => $locale === 'fa' ? 'یادآوری: ثبت حرکات جنین' : 'Reminder: Log Fetal Movement',
            'weekly' => $locale === 'fa' ? 'یادآوری: ثبت اطلاعات هفتگی' : 'Reminder: Log Weekly Data',
            default => $locale === 'fa' ? 'یادآوری' : 'Reminder',
        };

        $message = $locale === 'fa'
            ? "شما {$daysMissing} روز است که داده‌ای ثبت نکرده‌اید. ثبت منظم به پیگیری بهتر سلامت شما کمک می‌کند."
            : "You haven't logged data for {$daysMissing} days. Regular logging helps track your health better.";

        return PregnancyAlert::create([
            'user_id' => $this->user->id,
            'alert_level' => AlertLevel::INFO->value,
            'alert_type' => 'missing_data',
            'title' => $title,
            'message' => $message,
            'pregnancy_week' => $currentWeek,
            'recommended_actions' => $locale === 'fa'
                ? ['علائم امروز را ثبت کنید']
                : ['Log today\'s symptoms'],
        ]);
    }

    /**
     * Get medical history flags for alert context
     */
    private function getMedicalHistoryFlags(): array
    {
        if (!$this->profile) {
            return [];
        }

        $flags = [];

        if ($this->profile->has_miscarriage_history) {
            $flags['miscarriage_history'] = true;
        }
        if ($this->profile->has_high_risk_history) {
            $flags['high_risk_history'] = true;
        }
        if ($this->profile->rh_factor === 'negative') {
            $flags['rh_negative'] = true;
        }
        if (!empty($this->profile->pre_existing_conditions)) {
            $flags['pre_existing_conditions'] = $this->profile->pre_existing_conditions;
        }

        return $flags;
    }

    /**
     * Check if pregnancy is high risk
     */
    private function isHighRisk(): bool
    {
        $calculationService = new PregnancyCalculationService($this->user, $this->locale);
        return $calculationService->isHighRisk();
    }

    /**
     * Get unread alerts for user
     */
    public function getUnreadAlerts(): \Illuminate\Database\Eloquent\Collection
    {
        return PregnancyAlert::where('user_id', $this->user->id)
            ->unread()
            ->active()
            ->orderBy('created_at', 'desc')
            ->get();
    }

    /**
     * Get all active alerts for user
     */
    public function getActiveAlerts(): \Illuminate\Database\Eloquent\Collection
    {
        return PregnancyAlert::where('user_id', $this->user->id)
            ->active()
            ->orderBy('created_at', 'desc')
            ->get();
    }

    /**
     * Get emergency alerts count
     */
    public function getEmergencyAlertsCount(): int
    {
        return PregnancyAlert::where('user_id', $this->user->id)
            ->level(AlertLevel::EMERGENCY->value)
            ->unread()
            ->active()
            ->count();
    }
}
