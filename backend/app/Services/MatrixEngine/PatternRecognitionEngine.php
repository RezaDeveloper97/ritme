<?php

namespace App\Services\MatrixEngine;

use App\Enums\IntercourseType;
use App\Models\CycleHistory;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserProfile;
use Carbon\Carbon;
use Illuminate\Support\Collection;

/**
 * Pattern Recognition Engine - Layer 4 (Premium)
 *
 * Examines user history and identifies patterns for long-term health monitoring and medical alerts.
 * Logic: Scans user history. If phenomenon X occurs Y times in Z cycles, gives alert or analysis.
 * Goal: Identify hidden obstacles that aren't visible in a single day.
 */
class PatternRecognitionEngine
{
    private User $user;

    private ?UserProfile $profile;

    private string $locale;

    private const MINIMUM_CYCLES_FOR_PATTERN = 3;

    public function __construct(User $user, string $locale = 'fa')
    {
        $this->user = $user;
        $this->profile = $user->profile;
        $this->locale = $locale;
    }

    /**
     * Analyze patterns and return alerts
     */
    public function analyzePatterns(bool $isTTC = false): array
    {
        $patterns = [];
        $isPremium = $this->profile?->isPremium() ?? false;

        // Pattern recognition is primarily a premium feature
        if (! $isPremium) {
            return [];
        }

        $cycleHistories = $this->getCycleHistories();
        $healthLogs = $this->getHealthLogs();

        // General patterns (Non-TTC)
        if (! $isTTC) {
            $patterns = array_merge($patterns, $this->getGeneralPatterns($cycleHistories, $healthLogs));
        } else {
            // TTC-specific patterns
            $patterns = array_merge($patterns, $this->getTTCPatterns($cycleHistories, $healthLogs));
        }

        return $patterns;
    }

    /**
     * Get general patterns (Non-TTC mode)
     */
    private function getGeneralPatterns(Collection $cycleHistories, Collection $healthLogs): array
    {
        $patterns = [];
        $locale = $this->locale;
        $userName = $this->user->name ?? '';

        // Pattern 1: Always Painful Periods (Possible Dysmenorrhea/Endo)
        $painfulPeriodsPattern = $this->detectPainfulPeriodsPattern($healthLogs);
        if ($painfulPeriodsPattern) {
            $patterns[] = [
                'pattern_type' => 'painful_periods',
                'alert_level' => 'warning',
                'cycles_detected' => $painfulPeriodsPattern['count'],
                'condition' => $locale === 'fa'
                    ? 'ثبت درد شدید در روز ۱ و ۲ برای ۳ سیکل متوالی + مصرف مسکن'
                    : 'Recorded severe pain on days 1 and 2 for 3 consecutive cycles + painkiller use',
                'message' => $locale === 'fa'
                    ? "{$userName}، نمودارهات نشون میده ۳ ماه پشت سر همه که روزهای اول پریودت با درد شدید میگذره. درد پریود نباید زندگی رو مختل کنه. این الگو می‌تونه نشونه‌ای باشه که نیاز به بررسی پزشکی داره (مثل اندومتریوز یا کیست). این گزارش رو به دکترت نشون بده."
                    : "{$userName}, your charts show that for 3 consecutive months, your first period days have been with severe pain. Period pain shouldn't disrupt life. This pattern could be a sign that needs medical investigation (like endometriosis or cyst). Show this report to your doctor.",
                'recommended_action' => $locale === 'fa'
                    ? 'این گزارش رو به دکترت نشون بده.'
                    : 'Show this report to your doctor.',
            ];
        }

        // Pattern 2: Irregular Cycles
        $irregularCyclesPattern = $this->detectIrregularCyclesPattern($cycleHistories);
        if ($irregularCyclesPattern) {
            $patterns[] = [
                'pattern_type' => 'irregular_cycles',
                'alert_level' => 'info',
                'cycles_detected' => count($cycleHistories),
                'variation_days' => $irregularCyclesPattern['variation'],
                'condition' => $locale === 'fa'
                    ? 'تغییر طول سیکل بیش از ۷-۹ روز'
                    : 'Cycle length change more than 7-9 days',
                'message' => $locale === 'fa'
                    ? 'طول دوره‌هات نوسان زیادی داره. استرس، تغییر وزن یا تیروئید می‌تونن عاملش باشن. چون قصد بارداری نداری شاید مهم به نظر نیاد، اما پریود منظم نشونه سلامت کلی بدنه. اگر استرس خاصی نداشتی، بهتره یک چکاپ هورمونی بدی.'
                    : 'Your cycle length has a lot of variation. Stress, weight change or thyroid could be the cause. Since you\'re not trying to conceive it might not seem important, but regular period is a sign of overall body health. If you haven\'t had specific stress, it\'s better to get a hormonal checkup.',
                'recommended_action' => $locale === 'fa'
                    ? 'بهتره یک چکاپ هورمونی بدی.'
                    : 'Better to get a hormonal checkup.',
            ];
        }

        // Pattern 3: Severe PMS Mood (PMDD Watch)
        $pmddPattern = $this->detectPMDDPattern($healthLogs, $cycleHistories);
        if ($pmddPattern) {
            $patterns[] = [
                'pattern_type' => 'pmdd_watch',
                'alert_level' => 'info',
                'cycles_detected' => $pmddPattern['count'],
                'condition' => $locale === 'fa'
                    ? 'ثبت افسردگی/اضطراب شدید در ۵ روز قبل از پریود و بهبود ناگهانی با شروع خونریزی'
                    : 'Recorded severe depression/anxiety in 5 days before period and sudden improvement with bleeding start',
                'message' => $locale === 'fa'
                    ? 'الگوی مودت خیلی دقیقه: یک هفته قبل از پریود حال روحی‌ت به‌هم می‌ریزه و با شروع پریود خوب میشی. این احتمالاً PMS شدید یا PMDD هست. دونستن این موضوع کمکت می‌کنه اون هفته رو خالی‌تر بذاری و به خودت حق بدی که «الان واقعاً من نیستم».'
                    : 'Your mood pattern is very precise: a week before period your mental state gets disrupted and with period start you feel better. This is probably severe PMS or PMDD. Knowing this helps you keep that week lighter and give yourself the right to say "this isn\'t really me right now".',
                'recommended_action' => $locale === 'fa'
                    ? 'اون هفته رو خالی‌تر بذار و به خودت حق بده.'
                    : 'Keep that week lighter and give yourself the right.',
            ];
        }

        return $patterns;
    }

    /**
     * Get TTC-specific patterns
     */
    private function getTTCPatterns(Collection $cycleHistories, Collection $healthLogs): array
    {
        $patterns = [];
        $locale = $this->locale;
        $userName = $this->user->name ?? '';

        // TTC Pattern 1: Short Luteal Phase
        $shortLutealPattern = $this->detectShortLutealPhasePattern($cycleHistories);
        if ($shortLutealPattern) {
            $patterns[] = [
                'pattern_type' => 'short_luteal_phase',
                'alert_level' => 'warning',
                'cycles_detected' => $shortLutealPattern['count'],
                'average_luteal_length' => $shortLutealPattern['average'],
                'condition' => $locale === 'fa'
                    ? 'فاصله بین تخمک‌گذاری تا پریود بعدی کمتر از ۱۰ روز (برای ۳ سیکل متوالی)'
                    : 'Gap between ovulation and next period less than 10 days (for 3 consecutive cycles)',
                'analysis' => $locale === 'fa'
                    ? 'رحم زمان کافی برای ضخیم شدن و نگهداری جنین ندارد (نقص فاز لوتئال).'
                    : 'The uterus doesn\'t have enough time to thicken and hold the embryo (luteal phase defect).',
                'message' => $locale === 'fa'
                    ? "{$userName}، نمودارهای سه ماه گذشته نشون میدن فاصله بین تخمک‌گذاری تا پریودت کمی کوتاه‌تر از حد استاندارده. این یعنی شاید جنین زمان کافی برای چسبیدن به رحم رو پیدا نمی‌کنه. این موضوع با یک آزمایش ساده پروژسترون قابل بررسی و درمانه، پس حتماً در ویزیت بعدی به پزشکت اطلاع بده."
                    : "{$userName}, charts from the past three months show the gap between your ovulation and period is slightly shorter than standard. This means the embryo might not have enough time to attach to the uterus. This can be checked with a simple progesterone test and treated, so definitely inform your doctor at your next visit.",
                'recommended_action' => $locale === 'fa'
                    ? 'آزمایش پروژسترون بده و با پزشکت صحبت کن.'
                    : 'Get a progesterone test and talk to your doctor.',
            ];
        }

        // TTC Pattern 2: Pre-period Spotting
        $prePeriodSpottingPattern = $this->detectPrePeriodSpottingPattern($healthLogs, $cycleHistories);
        if ($prePeriodSpottingPattern) {
            $patterns[] = [
                'pattern_type' => 'pre_period_spotting',
                'alert_level' => 'info',
                'cycles_detected' => $prePeriodSpottingPattern['count'],
                'condition' => $locale === 'fa'
                    ? '۲ یا ۳ روز لکه‌بینی قهوه‌ای قبل از شروع خونریزی اصلی (تکرار شونده)'
                    : '2 or 3 days of brown spotting before main bleeding starts (recurring)',
                'analysis' => $locale === 'fa'
                    ? 'نشانه احتمالی افت زودهنگام پروژسترون یا آندومتریوز خفیف.'
                    : 'Possible sign of early progesterone drop or mild endometriosis.',
                'message' => $locale === 'fa'
                    ? 'متوجه شدم که اکثر ماه‌ها قبل از شروع کامل پریود، چند روز لکه‌بینی داری. در مسیر اقدام به بارداری، این می‌تونه نشونه‌ای باشه که سطح هورمون نگهدارنده‌ی بارداری زودتر از موعد افت می‌کنه. اگر این ماه هم بارداری رخ نداد، این نکته‌ی مهمی برای مشورت با پزشکه.'
                    : 'I noticed that most months you have a few days of spotting before your period fully starts. In the journey to conceive, this could be a sign that the pregnancy-supporting hormone drops earlier than it should. If pregnancy doesn\'t happen this month either, this is an important point to discuss with your doctor.',
                'recommended_action' => $locale === 'fa'
                    ? 'این نکته رو با پزشکت در میان بذار.'
                    : 'Share this point with your doctor.',
            ];
        }

        // TTC Pattern 3: Consistent One-sided Ovulation Pain
        $oneSidedPainPattern = $this->detectOneSidedOvulationPainPattern($healthLogs);
        if ($oneSidedPainPattern) {
            $patterns[] = [
                'pattern_type' => 'one_sided_ovulation_pain',
                'alert_level' => 'info',
                'cycles_detected' => $oneSidedPainPattern['count'],
                'dominant_side' => $oneSidedPainPattern['side'],
                'condition' => $locale === 'fa'
                    ? 'درد تخمک‌گذاری همیشه فقط در سمت راست یا فقط چپ (برای ۴ سیکل)'
                    : 'Ovulation pain always only on right or only left side (for 4 cycles)',
                'analysis' => $locale === 'fa'
                    ? 'شاید فقط یک تخمدان فعال است یا چسبندگی در یک سمت وجود دارد.'
                    : 'Maybe only one ovary is active or there\'s adhesion on one side.',
                'message' => $locale === 'fa'
                    ? "{$userName}، الگوی دردت نشون میده که تخمک‌گذاری‌ها اکثراً در یک سمت خاص حس میشن. اگرچه این می‌تونه طبیعی باشه، اما اگر زمان اقدام طولانی شده، بد نیست سونوگرافی کنی تا مطمئن بشیم هر دو تخمدان فعال و آزاد هستن."
                    : "{$userName}, your pain pattern shows ovulation is mostly felt on one specific side. Although this can be normal, if your trying period has been long, it wouldn\'t hurt to get an ultrasound to make sure both ovaries are active and free.",
                'recommended_action' => $locale === 'fa'
                    ? 'سونوگرافی برای بررسی فعالیت هر دو تخمدان.'
                    : 'Ultrasound to check activity of both ovaries.',
            ];
        }

        // TTC Pattern 4: Irregular Intercourse Timing
        $intercourseTimingPattern = $this->detectIntercourseTiming($healthLogs, $cycleHistories);
        if ($intercourseTimingPattern) {
            $patterns[] = [
                'pattern_type' => 'irregular_intercourse_timing',
                'alert_level' => 'tip',
                'cycles_detected' => $intercourseTimingPattern['count'],
                'condition' => $locale === 'fa'
                    ? 'ثبت رابطه جنسی فقط در روز تخمک‌گذاری (بدون روزهای قبل)'
                    : 'Intercourse recorded only on ovulation day (without days before)',
                'analysis' => $locale === 'fa'
                    ? 'از دست دادن پنجره باروری (اسپرم باید قبل از تخمک آنجا باشد).'
                    : 'Missing the fertile window (sperm should be there before the egg).',
                'message' => $locale === 'fa'
                    ? 'تحلیل سیکل‌های قبلی نشون میده تمرکزت برای رابطه دقیقاً روی روز آزادسازی تخمک بوده. برای افزایش شانس بارداری، بهتره رابطه رو از سه روز قبل شروع کنی تا اسپرم‌ها منتظر تخمک باشن، چون اسپرم تا ۵ روز زنده می‌مونه ولی تخمک فقط ۲۴ ساعت وقت داره.'
                    : 'Analysis of previous cycles shows your focus for intercourse has been exactly on the egg release day. To increase pregnancy chances, it\'s better to start intercourse from 3 days before so sperm are waiting for the egg, because sperm survive up to 5 days but the egg only has 24 hours.',
                'recommended_action' => $locale === 'fa'
                    ? 'رابطه رو از سه روز قبل از تخمک‌گذاری شروع کن.'
                    : 'Start intercourse from 3 days before ovulation.',
            ];
        }

        return $patterns;
    }

    /**
     * Detect painful periods pattern
     */
    private function detectPainfulPeriodsPattern(Collection $healthLogs): ?array
    {
        $painfulCycles = 0;
        $cycleDays1to3Logs = $healthLogs->filter(function ($log) {
            // This is simplified - in real implementation we'd need to check cycle day
            return $log->stomach_ache_intensity === 'high'
                || $log->pelvic_pain_intensity === 'high';
        });

        // Simplified check - count severe pain logs
        if ($cycleDays1to3Logs->count() >= self::MINIMUM_CYCLES_FOR_PATTERN) {
            return ['count' => $cycleDays1to3Logs->count()];
        }

        return null;
    }

    /**
     * Detect irregular cycles pattern
     */
    private function detectIrregularCyclesPattern(Collection $cycleHistories): ?array
    {
        if ($cycleHistories->count() < 2) {
            return null;
        }

        $lengths = $cycleHistories->pluck('cycle_length')->filter()->values();
        if ($lengths->count() < 2) {
            return null;
        }

        $min = $lengths->min();
        $max = $lengths->max();
        $variation = $max - $min;

        if ($variation >= 7) {
            return ['variation' => $variation];
        }

        return null;
    }

    /**
     * Detect PMDD pattern
     */
    private function detectPMDDPattern(Collection $healthLogs, Collection $cycleHistories): ?array
    {
        // Simplified - look for sad/anxious moods in logs
        $pmddCount = $healthLogs->filter(function ($log) {
            if (! is_array($log->moods)) {
                return false;
            }

            return in_array('sad', $log->moods) || in_array('anxious', $log->moods);
        })->count();

        if ($pmddCount >= self::MINIMUM_CYCLES_FOR_PATTERN) {
            return ['count' => $pmddCount];
        }

        return null;
    }

    /**
     * Detect short luteal phase pattern
     */
    private function detectShortLutealPhasePattern(Collection $cycleHistories): ?array
    {
        // In a real implementation, we'd calculate luteal phase from ovulation to period
        // For now, use a simplified approach based on cycle length
        $shortLutealCycles = $cycleHistories->filter(function ($history) {
            // Assuming ovulation is cycle_length - 14, luteal phase is 14 days normally
            // Short luteal = less than 10 days
            $cycleLength = $history->cycle_length;
            if (! $cycleLength) {
                return false;
            }
            $lutealPhase = 14; // Estimated

            return $lutealPhase < 10;
        });

        if ($shortLutealCycles->count() >= self::MINIMUM_CYCLES_FOR_PATTERN) {
            return [
                'count' => $shortLutealCycles->count(),
                'average' => 9, // Simplified
            ];
        }

        return null;
    }

    /**
     * Detect pre-period spotting pattern
     */
    private function detectPrePeriodSpottingPattern(Collection $healthLogs, Collection $cycleHistories): ?array
    {
        $spottingLogs = $healthLogs->filter(function ($log) {
            return $log->spotting === true;
        });

        if ($spottingLogs->count() >= 2) {
            return ['count' => $spottingLogs->count()];
        }

        return null;
    }

    /**
     * Detect one-sided ovulation pain pattern
     */
    private function detectOneSidedOvulationPainPattern(Collection $healthLogs): ?array
    {
        $ovarianPainLogs = $healthLogs->filter(function ($log) {
            return $log->ovarian_pain_intensity !== null;
        });

        // Simplified - we'd need to track which side the pain is on
        if ($ovarianPainLogs->count() >= 4) {
            return [
                'count' => $ovarianPainLogs->count(),
                'side' => 'right', // Placeholder
            ];
        }

        return null;
    }

    /**
     * Detect intercourse timing pattern
     */
    private function detectIntercourseTiming(Collection $healthLogs, Collection $cycleHistories): ?array
    {
        $intercourseL = $healthLogs->filter(function ($log) {
            // Current logs answer this with intercourse_type; older ones carry it
            // inside the sexual_activities multi-select.
            if ($log->intercourse_type === IntercourseType::UNPROTECTED->value) {
                return true;
            }
            if (! is_array($log->sexual_activities)) {
                return false;
            }

            return in_array('unprotected_intercourse', $log->sexual_activities);
        });

        // Simplified check
        if ($intercourseL->count() >= 2) {
            return ['count' => $intercourseL->count()];
        }

        return null;
    }

    /**
     * Get cycle histories
     */
    private function getCycleHistories(): Collection
    {
        return CycleHistory::where('user_id', $this->user->id)
            ->orderBy('period_start_date', 'desc')
            ->take(6)
            ->get();
    }

    /**
     * Get health logs for pattern analysis (last 3-6 months)
     */
    private function getHealthLogs(): Collection
    {
        return DailyHealthLog::where('user_id', $this->user->id)
            ->where('log_date', '>=', Carbon::now()->subMonths(6))
            ->orderBy('log_date', 'desc')
            ->get();
    }
}
