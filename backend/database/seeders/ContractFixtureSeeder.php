<?php

namespace Database\Seeders;

use App\Models\Challenge;
use App\Models\Language;
use App\Models\TaskTemplate;
use App\Services\Language\LanguageRegistry;
use Carbon\Carbon;
use Carbon\CarbonImmutable;
use Illuminate\Database\Seeder;
use Illuminate\Support\Facades\DB;

/**
 * Deterministic fixture for recording Laravel contract goldens
 * (docs/go-migration/contract.md). Only ever run against the contract stack
 * (docker-compose.contract.yml) or an in-memory test database.
 *
 * - Idempotent: content seeders are themselves idempotent, and every persona is
 *   reset on each run (its child rows are deleted and re-inserted with the same
 *   fixed ids), so running it twice yields the same rows.
 * - Deterministic: fixed ids, fixed mobiles `0990000xxxx`, no Faker, and every
 *   relative date is computed from CONTRACT_TODAY rather than the real clock.
 *   Record with `X-Test-Now` on the same day so data and clock line up.
 *
 * Row ids: a persona's user id is 1001+; its child rows use `userId * 100 + n`.
 */
class ContractFixtureSeeder extends Seeder
{
    public const CONTRACT_TODAY = '2026-09-23';

    /** Wall-clock (Tehran) used for created_at/updated_at and the content seeders' now(). */
    public const CONTRACT_NOW = '2026-09-23 09:00:00';

    /**
     * key => [user id, mobile]. Keep in sync with docs/go-migration/contract.md.
     */
    public const PERSONAS = [
        'no_profile' => [1001, '09900000001'],
        'profile_no_history' => [1002, '09900000002'],
        'onboarding_declared' => [1003, '09900000003'],
        'regular' => [1004, '09900000004'],
        'irregular' => [1005, '09900000005'],
        'short_outlier' => [1006, '09900000006'],
        'open_period_day3' => [1007, '09900000007'],
        'open_period_day11' => [1008, '09900000008'],
        'open_period_day13' => [1009, '09900000009'],
        'overdue_10' => [1010, '09900000010'],
        'overdue_20' => [1011, '09900000011'],
        'ttc' => [1012, '09900000012'],
        'premium' => [1013, '09900000013'],
        'pregnant_lmp_w8' => [1014, '09900000014'],
        'pregnant_ultrasound_w26' => [1015, '09900000015'],
        'pregnant_manual_w14' => [1016, '09900000016'],
        'blocked' => [1017, '09900000017'],
        'engaged' => [1018, '09900000018'],
    ];

    /** Every per-user table a persona can own rows in (children first). */
    private const USER_TABLES = [
        'user_challenge_completions',
        'user_task_completions',
        'user_notifications',
        'reminders',
        'pregnancy_alerts',
        'pregnancy_fetal_movements',
        'pregnancy_weekly_logs',
        'pregnancy_symptom_logs',
        'pregnancy_profiles',
        'daily_health_logs',
        'cycle_histories',
        'user_profiles',
    ];

    private CarbonImmutable $today;

    public function run(): void
    {
        $previous = Carbon::getTestNow();
        Carbon::setTestNow(Carbon::parse(self::CONTRACT_NOW, config('app.timezone')));
        $this->today = CarbonImmutable::parse(self::CONTRACT_TODAY, config('app.timezone'))->startOfDay();

        try {
            $this->seedContent();
            DB::transaction(fn () => $this->seedPersonas());
        } finally {
            Carbon::setTestNow($previous);
        }
    }

    // ---------------------------------------------------------------- content

    private function seedContent(): void
    {
        $this->call([
            LanguageSeeder::class,
            TaskTemplateSeeder::class,
            ArticleSeeder::class,
            AffirmationSeeder::class,
            ChallengeSeeder::class,
            MessageContentSeeder::class,
            RecommendationSeeder::class,
            PregnancyWeeklyContentSeeder::class,
            PhaseContentSeeder::class,
            InfoSectionSeeder::class,
        ]);

        // A third active locale exercises the "not fa, not en" paths (enum
        // labels fall back to English, content falls back to the default).
        Language::firstOrCreate(['code' => 'ar'], [
            'name' => 'العربية',
            'english_name' => 'Arabic',
            'direction' => 'rtl',
            'is_active' => true,
            'is_default' => false,
            'sort_order' => 2,
        ]);
        app(LanguageRegistry::class)->flush();

        $banners = [
            [1, ['fa' => 'بنر بالای خانه', 'en' => 'Home top banner'], 'contract/banner-top.webp', 'home_top', '/articles', 'internal', $this->day(-30).' 00:00:00', $this->day(30).' 23:59:59', 1],
            [2, ['fa' => 'بنر میانی', 'en' => 'Middle banner'], 'contract/banner-middle.webp', 'home_middle', 'https://ritme.app', 'external', null, null, 2],
            [3, ['fa' => 'بنر منقضی', 'en' => 'Expired banner'], 'contract/banner-expired.webp', 'home_top', null, null, $this->day(-60).' 00:00:00', $this->day(-1).' 23:59:59', 3],
        ];
        foreach ($banners as [$id, $title, $path, $position, $url, $linkType, $starts, $ends, $sort]) {
            DB::table('banners')->updateOrInsert(['id' => $id], [
                'title' => json_encode($title),
                'image_path' => $path,
                'position' => $position,
                'link_url' => $url,
                'link_type' => $linkType,
                'starts_at' => $starts,
                'ends_at' => $ends,
                'is_active' => true,
                'sort_order' => $sort,
            ] + $this->stamps());
        }
    }

    // --------------------------------------------------------------- personas

    private function seedPersonas(): void
    {
        $ids = array_column(self::PERSONAS, 0);
        foreach (self::USER_TABLES as $table) {
            DB::table($table)->whereIn('user_id', $ids)->delete();
        }
        DB::table('otp_verifications')->whereIn('mobile', array_column(self::PERSONAS, 1))->delete();

        foreach (self::PERSONAS as $key => [$id, $mobile]) {
            DB::table('users')->updateOrInsert(['id' => $id], [
                'name' => $key === 'no_profile' ? null : 'Contract '.str_replace('_', ' ', $key),
                'email' => null,
                'mobile' => $mobile,
                'mobile_verified_at' => self::CONTRACT_NOW,
                'blocked_at' => $key === 'blocked' ? $this->day(-3).' 12:00:00' : null,
                'password' => null,
            ] + $this->stamps());
        }

        $p = fn (string $key): int => self::PERSONAS[$key][0];

        // No history at all: the resolver falls back to the profile LMP.
        $this->profile($p('profile_no_history'), ['last_period_start' => $this->day(-10)]);

        // Only the onboarding estimate row (pre-2026-08 shape: estimated, unconfirmed).
        $this->profile($p('onboarding_declared'), ['last_period_start' => $this->day(-12)]);
        $this->cycle($p('onboarding_declared'), 1, $this->day(-12), $this->day(-8), null, confirmed: false, estimated: true, source: 'onboarding_estimate');

        // Regular 28d, 6 confirmed cycles, last started 9 days ago (cycle day 10).
        $this->profile($p('regular'), ['last_period_start' => $this->day(-9)]);
        $this->cycles($p('regular'), -9, [28, 28, 28, 28, 28]);
        $this->log($p('regular'), 1, -9, ['bleeding_intensity' => 'high', 'blood_color' => 'red', 'stomach_ache_intensity' => 'medium', 'moods' => ['sensitive']]);
        $this->log($p('regular'), 2, -8, ['bleeding_intensity' => 'medium', 'moods' => ['calm']]);
        $this->log($p('regular'), 3, -2, ['energy_level' => 'high', 'sleep_quality' => 'good', 'sleep_duration' => '6_9']);
        $this->log($p('regular'), 4, 0, ['moods' => ['happy', 'calm'], 'weight' => '61.50', 'energy_level' => 'medium', 'notes' => 'contract']);

        // Irregular: last three lengths span > 7 days.
        $this->profile($p('irregular'), ['last_period_start' => $this->day(-14), 'cycle_duration' => 30]);
        $this->cycles($p('irregular'), -14, [26, 41, 29, 35, 24]);

        // A 17-day (outlier) cycle among regular ones.
        $this->profile($p('short_outlier'), ['last_period_start' => $this->day(-20)]);
        $this->cycles($p('short_outlier'), -20, [28, 17, 28, 29]);

        // Open periods: day 3, day 11 (past the warning cap), day 13 (past the hard cap of 12).
        foreach (['open_period_day3' => -2, 'open_period_day11' => -10, 'open_period_day13' => -12] as $key => $offset) {
            $this->profile($p($key), ['last_period_start' => $this->day($offset)]);
            $this->cycles($p($key), $offset, [28, 28, 28], openLatest: true);
        }

        // Overdue: the expected period (ECL 28) is 10 / 20 days late.
        $this->profile($p('overdue_10'), ['last_period_start' => $this->day(-38)]);
        $this->cycles($p('overdue_10'), -38, [28, 28, 28]);
        $this->profile($p('overdue_20'), ['last_period_start' => $this->day(-48)]);
        $this->cycles($p('overdue_20'), -48, [28, 28, 28]);

        // Trying to conceive.
        $this->profile($p('ttc'), ['last_period_start' => $this->day(-12), 'user_goal' => 'ttc', 'pregnancy_intention' => 'trying']);
        $this->cycles($p('ttc'), -12, [28, 29, 27]);

        // Premium: 20 low-energy days feed the premium pattern layer (chronic_fatigue ≥ 15).
        $this->profile($p('premium'), ['last_period_start' => $this->day(-16), 'subscription_type' => 'premium', 'chronic_conditions' => ['pcos']]);
        $this->cycles($p('premium'), -16, [30, 31, 29]);
        for ($i = 1; $i <= 20; $i++) {
            $this->log($p('premium'), $i, -$i, ['energy_level' => 'low', 'fatigue' => true, 'sleep_quality' => 'bad', 'headache_intensity' => $i % 3 === 0 ? 'high' : null]);
        }

        $this->seedPregnancies($p);

        // Blocked: full profile so only the block explains a 403.
        $this->profile($p('blocked'), ['last_period_start' => $this->day(-5)]);
        $this->cycles($p('blocked'), -5, [28, 28]);

        $this->seedEngaged($p('engaged'));
    }

    /** @param callable(string): int $p */
    private function seedPregnancies(callable $p): void
    {
        // LMP: 7w3d today → current week 8.
        $lmp = $this->today->subDays(7 * 7 + 3);
        $id = $p('pregnant_lmp_w8');
        $this->profile($id, ['last_period_start' => $lmp->toDateString(), 'pregnancy_intention' => 'pregnant']);
        $this->pregnancyProfile($id, [
            'age_source' => 'lmp',
            'confidence_level' => 'medium',
            'lmp_date' => $lmp->toDateString(),
            'estimated_due_date' => $lmp->addDays(280)->toDateString(),
            'estimated_conception_date' => $lmp->addDays(14)->toDateString(),
            'uncertainty_days' => 3,
        ]);
        $this->symptomLog($id, 1, -1, ['has_nausea' => true, 'nausea_severity' => 'moderate', 'has_fatigue' => true, 'fatigue_severity' => 'mild']);

        // Ultrasound: 23w2d two weeks ago → 25w2d today → current week 26; alerts, logs, movements.
        $id = $p('pregnant_ultrasound_w26');
        $usDate = $this->today->subDays(14);
        $usLmp = $usDate->subDays(23 * 7 + 2);
        $this->profile($id, ['last_period_start' => $usLmp->toDateString(), 'pregnancy_intention' => 'pregnant']);
        $this->pregnancyProfile($id, [
            'age_source' => 'ultrasound',
            'confidence_level' => 'high',
            'ultrasound_date' => $usDate->toDateString(),
            'ultrasound_weeks' => 23,
            'ultrasound_days' => 2,
            'estimated_due_date' => $usLmp->addDays(280)->toDateString(),
            'estimated_conception_date' => $usLmp->addDays(14)->toDateString(),
            'uncertainty_days' => 1,
            'has_miscarriage_history' => true,
            'has_high_risk_history' => false,
            'pre_existing_conditions' => json_encode(['hypertension']),
            'blood_type' => 'O',
            'rh_factor' => 'negative',
            'rh_negative_care_flag' => true,
            'first_fetal_movement_date' => $this->day(-40),
            'fetal_movement_felt' => true,
        ]);
        $this->symptomLog($id, 1, -2, ['has_back_pain' => true, 'back_pain_severity' => 'moderate']);
        $this->symptomLog($id, 2, -1, ['has_spotting' => true, 'spotting_severity' => 'mild', 'has_cramping' => true, 'cramping_severity' => 'mild']);
        $this->symptomLog($id, 3, 0, ['has_headache' => true, 'headache_severity' => 'severe', 'has_dizziness' => true, 'dizziness_severity' => 'severe', 'has_pelvic_pressure' => true, 'pelvic_pressure_severity' => 'severe', 'notes' => 'contract']);
        foreach ([24 => -14, 25 => -7, 26 => 0] as $week => $offset) {
            DB::table('pregnancy_weekly_logs')->insert([
                'id' => $id * 100 + $week,
                'user_id' => $id,
                'log_date' => $this->day($offset),
                'pregnancy_week' => $week,
                'weight' => $week === 26 ? '72.40' : '71.10',
                'has_swelling' => $week === 26,
                'swelling_locations' => $week === 26 ? json_encode(['feet', 'hands']) : null,
                'has_blood_pressure_device' => true,
                'systolic_pressure' => $week === 26 ? 142 : 118,
                'diastolic_pressure' => $week === 26 ? 92 : 76,
                'fasting_blood_sugar' => $week === 26 ? '97.50' : '88.00',
                'post_meal_blood_sugar' => '128.00',
                'overall_mood' => $week === 26 ? 'moderate' : 'good',
                'has_anxiety' => $week === 26,
                'anxiety_severity' => $week === 26 ? 'moderate' : null,
            ] + $this->stamps());
        }
        foreach ([1 => [-2, 'normal', 12], 2 => [-1, 'normal', 10], 3 => [0, 'reduced', 3]] as $n => [$offset, $status, $count]) {
            DB::table('pregnancy_fetal_movements')->insert([
                'id' => $id * 100 + $n,
                'user_id' => $id,
                'log_date' => $this->day($offset),
                'pregnancy_week' => $offset < -1 ? 25 : 26,
                'movement_status' => $status,
                'movement_count' => $count,
                'first_movement_time' => '08:15:00',
                'last_movement_time' => '21:40:00',
            ] + $this->stamps());
        }
        $alerts = [
            [1, 'emergency', 'symptom_based', 'کاهش حرکات جنین', 'حرکات جنین کمتر از معمول است. با پزشک تماس بگیرید.', ['movement_status' => 'reduced'], false, false],
            [2, 'warning', 'symptom_based', 'فشار خون بالا', 'فشار خون شما ۱۴۲/۹۲ ثبت شده است.', ['systolic' => 142, 'diastolic' => 92], true, false],
            [3, 'info', 'routine', 'یادآوری مراقبت Rh منفی', 'با پزشک درباره‌ی تزریق روگام صحبت کنید.', [], true, true],
        ];
        foreach ($alerts as [$n, $level, $type, $title, $message, $trigger, $read, $dismissed]) {
            DB::table('pregnancy_alerts')->insert([
                'id' => $id * 100 + $n,
                'user_id' => $id,
                'alert_level' => $level,
                'alert_type' => $type,
                'title' => $title,
                'message' => $message,
                'pregnancy_week' => 26,
                'trigger_symptoms' => json_encode($trigger),
                'medical_history_flags' => json_encode($n === 3 ? ['rh_negative' => true] : []),
                'is_read' => $read,
                'is_dismissed' => $dismissed,
                'read_at' => $read ? $this->day(-1).' 10:00:00' : null,
                'dismissed_at' => $dismissed ? $this->day(-1).' 10:05:00' : null,
                'recommended_actions' => json_encode(['contact_doctor']),
            ] + $this->stamps());
        }

        // Manual: 12w0d entered a week ago → 13w0d today → current week 14.
        $id = $p('pregnant_manual_w14');
        $manualLmp = $this->today->subDays(13 * 7);
        $this->profile($id, ['last_period_start' => $manualLmp->toDateString(), 'pregnancy_intention' => 'pregnant']);
        $this->pregnancyProfile($id, [
            'age_source' => 'manual',
            'confidence_level' => 'low',
            'manual_weeks' => 12,
            'manual_days' => 0,
            'manual_entry_date' => $this->day(-7),
            'estimated_due_date' => $manualLmp->addDays(280)->toDateString(),
            'estimated_conception_date' => $manualLmp->addDays(14)->toDateString(),
            'uncertainty_days' => 5,
        ]);
    }

    /** Reminders, notifications and task/challenge completions. */
    private function seedEngaged(int $id): void
    {
        $this->profile($id, ['last_period_start' => $this->day(-6)]);
        $this->cycles($id, -6, [28, 28, 27]);
        $this->log($id, 1, 0, ['moods' => ['happy'], 'energy_level' => 'high']);

        $reminders = [
            [1, 'doctor', 'دکتر رضایی', 'زنان و زایمان', $this->day(3).' 16:30:00', 'none', null, null, null, true, ['phone' => '02100000000']],
            [2, 'medication', 'آهن', '۱ قرص', null, 'daily', '09:00:00', $this->day(-10), $this->day(20), true, null],
            [3, 'appointment', 'سونوگرافی', null, $this->day(-5).' 10:00:00', 'none', null, null, null, true, null],
            [4, 'custom', 'نوشیدن آب', null, null, 'weekly', '12:00:00', $this->day(-30), null, false, null],
        ];
        foreach ($reminders as [$n, $type, $title, $subtitle, $scheduled, $recurrence, $time, $starts, $ends, $active, $meta]) {
            DB::table('reminders')->insert([
                'id' => $id * 100 + $n,
                'user_id' => $id,
                'type' => $type,
                'title' => $title,
                'subtitle' => $subtitle,
                'notes' => null,
                'scheduled_at' => $scheduled,
                'recurrence' => $recurrence,
                'recurrence_time' => $time,
                'starts_on' => $starts,
                'ends_on' => $ends,
                'is_active' => $active,
                'meta' => $meta === null ? null : json_encode($meta),
            ] + $this->stamps());
        }

        $notifications = [
            [1, 'reminder', ['fa' => 'یادآوری دارو', 'en' => 'Medication reminder'], ['fa' => 'وقت قرص آهن است.', 'en' => 'Time for your iron tablet.'], '/reminders', null],
            [2, 'tip', ['fa' => 'نکته‌ی امروز', 'en' => 'Tip of the day'], null, null, $this->day(-1).' 08:00:00'],
            [3, 'achievement', ['fa' => 'آفرین!', 'en' => 'Well done!'], ['fa' => '۷ روز پشت سر هم ثبت کردی.', 'en' => 'You logged 7 days in a row.'], null, null],
        ];
        foreach ($notifications as [$n, $type, $title, $body, $url, $readAt]) {
            DB::table('user_notifications')->insert([
                'id' => $id * 100 + $n,
                'user_id' => $id,
                'type' => $type,
                'title' => json_encode($title),
                'body' => $body === null ? null : json_encode($body),
                'action_url' => $url,
                'data' => json_encode(['source' => 'contract']),
                'read_at' => $readAt,
            ] + $this->stamps());
        }

        $tasks = TaskTemplate::query()->orderBy('id')->limit(2)->pluck('id')->all();
        foreach ($tasks as $n => $taskId) {
            DB::table('user_task_completions')->insert([
                'id' => $id * 100 + $n + 1,
                'user_id' => $id,
                'task_template_id' => $taskId,
                'completion_date' => $this->day(-$n),
                'completed_at' => $this->day(-$n).' 08:30:00',
            ] + $this->stamps());
        }

        $challenges = Challenge::query()->orderBy('id')->limit(2)->pluck('id')->all();
        foreach ($challenges as $n => $challengeId) {
            DB::table('user_challenge_completions')->insert([
                'id' => $id * 100 + $n + 1,
                'user_id' => $id,
                'challenge_id' => $challengeId,
                'completion_date' => $this->day(-1 - $n),
                'completed_at' => $this->day(-1 - $n).' 20:00:00',
            ] + $this->stamps());
        }
    }

    // ---------------------------------------------------------------- helpers

    private function day(int $offset): string
    {
        return $this->today->addDays($offset)->toDateString();
    }

    /** @return array{created_at: string, updated_at: string} */
    private function stamps(): array
    {
        return ['created_at' => self::CONTRACT_NOW, 'updated_at' => self::CONTRACT_NOW];
    }

    private function profile(int $userId, array $overrides = []): void
    {
        if (isset($overrides['chronic_conditions'])) {
            $overrides['chronic_conditions'] = json_encode($overrides['chronic_conditions']);
        }

        DB::table('user_profiles')->insert(array_merge([
            'id' => $userId,
            'user_id' => $userId,
            'birthday' => '1996-04-12',
            'weight' => 60.5,
            'height' => 165,
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => null,
            'user_goal' => 'non_ttc',
            'pregnancy_intention' => 'avoiding',
            'chronic_conditions' => null,
            'subscription_type' => 'free',
            'calculation_status' => 'completed',
            'calculation_started_at' => self::CONTRACT_NOW,
            'calculation_completed_at' => self::CONTRACT_NOW,
            'calculation_version' => 1,
        ], $overrides, $this->stamps()));
    }

    /**
     * Confirmed history: the latest period starts at $latestOffset, and each
     * earlier one starts the given number of days before the next (newest
     * first). All periods last 5 days; the latest can be left open.
     *
     * @param  list<int>  $lengths
     */
    private function cycles(int $userId, int $latestOffset, array $lengths, bool $openLatest = false): void
    {
        $starts = [$latestOffset];
        foreach ($lengths as $length) {
            $starts[] = end($starts) - $length;
        }
        $starts = array_reverse($starts); // oldest first

        foreach ($starts as $i => $start) {
            $isLatest = $i === count($starts) - 1;
            $open = $isLatest && $openLatest;
            $cycleLength = $isLatest ? null : $starts[$i + 1] - $start;
            $this->cycle($userId, $i + 1, $this->day($start), $open ? null : $this->day($start + 4), $cycleLength, bleeding: $open ? null : 5);
        }
    }

    private function cycle(
        int $userId,
        int $n,
        string $start,
        ?string $end,
        ?int $cycleLength,
        ?int $bleeding = 5,
        bool $confirmed = true,
        bool $estimated = false,
        string $source = 'user_logged',
    ): void {
        DB::table('cycle_histories')->insert([
            'id' => $userId * 100 + $n,
            'user_id' => $userId,
            'period_start_date' => $start,
            'period_end_date' => $end,
            'cycle_length' => $cycleLength,
            'bleeding_length' => $end === null ? null : $bleeding,
            'is_confirmed' => $confirmed,
            'is_estimated' => $estimated,
            'source' => $source,
            'data_quality_flags' => null,
        ] + $this->stamps());
    }

    private function log(int $userId, int $n, int $offset, array $fields): void
    {
        foreach (['moods', 'exercise_type', 'sexual_activities', 'medications'] as $json) {
            if (isset($fields[$json])) {
                $fields[$json] = json_encode($fields[$json]);
            }
        }

        DB::table('daily_health_logs')->insert([
            'id' => $userId * 100 + $n,
            'user_id' => $userId,
            'log_date' => $this->day($offset),
        ] + $fields + $this->stamps());
    }

    private function pregnancyProfile(int $userId, array $fields): void
    {
        DB::table('pregnancy_profiles')->insert([
            'id' => $userId,
            'user_id' => $userId,
            'pregnancy_mode' => true,
            'cycle_mode' => false,
            'is_locked' => false,
            'onboarding_completed' => true,
            'onboarding_completed_at' => $this->day(-20).' 11:00:00',
        ] + $fields + $this->stamps());
    }

    private function symptomLog(int $userId, int $n, int $offset, array $fields): void
    {
        DB::table('pregnancy_symptom_logs')->insert([
            'id' => $userId * 100 + $n,
            'user_id' => $userId,
            'log_date' => $this->day($offset),
        ] + $fields + $this->stamps());
    }
}
