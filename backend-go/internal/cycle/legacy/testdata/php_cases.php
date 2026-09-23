<?php

// Regenerates php_cases.json: expected values computed by the PHP classes themselves
// (HealthDataEngine, DailyTipLocalizer, Recommendation), for the Go unit tests of the edge cases the
// cycle-sweep goldens don't reach (probability/rounding, text flags, code-fallback tips, backward
// cycle-day extrapolation, recommendation localisation).
//
//   php backend-go/internal/cycle/legacy/testdata/php_cases.php > backend-go/internal/cycle/legacy/testdata/php_cases.json
//
// Boots the Laravel app for the model casts; no database is touched.

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\CycleVariability;
use App\Models\DailyHealthLog;
use App\Models\Recommendation;
use App\Models\User;
use App\Models\UserProfile;
use App\Services\HealthEngine\DailyTipLocalizer;
use App\Services\HealthEngine\HealthDataEngine;
use App\Services\HealthEngine\RecommendationRepository;
use Carbon\Carbon;
use Illuminate\Contracts\Console\Kernel;

$backend = realpath(__DIR__.'/../../../../../backend');
require $backend.'/vendor/autoload.php';
$app = require $backend.'/bootstrap/app.php';
$app->make(Kernel::class)->bootstrap();

$noDbTips = new class extends RecommendationRepository
{
    public function hasContent(): bool
    {
        return false;
    }
};

function engine(?UserProfile $profile, RecommendationRepository $repo): HealthDataEngine
{
    $user = new User;
    $user->setRelation('profile', $profile);

    return new HealthDataEngine($user, 'en', $repo);
}

function profile(array $attrs): UserProfile
{
    $p = new UserProfile;
    $p->setRawAttributes($attrs);

    return $p;
}

/** @param array<string, mixed> $attrs raw column values */
function dailyLog(array $attrs): DailyHealthLog
{
    $log = new DailyHealthLog;
    $log->setRawAttributes($attrs);

    return $log;
}

$engine = engine(null, $noDbTips);
$out = [];

// --- Log variants shared by the symptom / cycle-score / tips cases ------------------------------
$logs = [];
foreach ([null, 'egg_white', 'watery', 'creamy'] as $texture) {
    foreach ([null, 'mild'] as $ovarian) {
        foreach (['none', 'desire', 'activities'] as $libido) {
            foreach ([null, 'severe'] as $bloating) {
                foreach ([null, 1, 0] as $dryness) {
                    $logs[] = array_filter([
                        'discharge_texture' => $texture,
                        'ovarian_pain_intensity' => $ovarian,
                        'sexual_desire' => $libido === 'desire' ? 'higher' : null,
                        'sexual_activities' => $libido === 'activities' ? '["protected","high_desire"]' : null,
                        'bloating_intensity' => $bloating,
                        'vaginal_dryness' => $dryness,
                    ], fn ($v) => $v !== null);
                }
            }
        }
    }
}

// --- Symptom score ------------------------------------------------------------------------------
$symptomValues = [];
foreach (array_merge([null], $logs) as $attrs) {
    foreach ([false, true] as $pms) {
        foreach ([false, true] as $spotting) {
            $s = $engine->calculateSymptomScore($attrs === null ? null : dailyLog($attrs), $pms, $spotting);
            $symptomValues[(string) $s] = $s;
            $out['symptom_score'][] = [
                'log' => $attrs === null ? null : (object) $attrs, 'pms' => $pms, 'luteal_spotting' => $spotting,
                'score' => $s, 'rounded' => round($s, 4),
            ];
        }
    }
}

// --- Cycle score --------------------------------------------------------------------------------
$cycleValues = [];
foreach (CycleVariability::cases() as $variability) {
    foreach (array_merge([null], $logs) as $attrs) {
        $s = $engine->calculateCycleScore($variability, $attrs === null ? null : dailyLog($attrs));
        $cycleValues[(string) $s] = $s;
        $out['cycle_score'][] = [
            'variability' => $variability->value, 'log' => $attrs === null ? null : (object) $attrs,
            'score' => $s, 'rounded' => round($s, 4),
        ];
    }
}

// --- Age factor (Carbon ->age against a frozen now) ---------------------------------------------
$ageValues = [];
$today = '2026-09-23';
Carbon::setTestNow(Carbon::parse($today.' 10:00:00'));
foreach ([
    '2026-09-24', '2026-09-23', '2007-09-24', '2006-09-23', '1997-09-23', '1997-09-24', '1996-09-23',
    '1992-09-24', '1992-09-23', '1989-09-23', '1988-09-24', '1986-09-23', '1985-09-23', '1926-09-23',
    '1925-09-23', '1996-02-29', '2004-02-29', '2006-02-28',
] as $birthday) {
    $f = engine(profile(['birthday' => $birthday]), $noDbTips)->calculateAgeFactor();
    $ageValues[(string) $f] = $f;
    $out['age_factor'][] = ['birthday' => $birthday, 'today' => $today, 'factor' => $f];
}
$out['age_factor'][] = ['birthday' => null, 'today' => $today, 'factor' => engine(profile([]), $noDbTips)->calculateAgeFactor()];

// Leap-day "today": ->age on 2028-02-29 / 2027-02-28 / 2027-03-01.
foreach (['2027-02-28', '2027-03-01', '2028-02-29'] as $now) {
    Carbon::setTestNow(Carbon::parse($now.' 10:00:00'));
    foreach (['2008-02-29', '2007-03-01', '1998-02-28'] as $birthday) {
        $out['age_factor'][] = [
            'birthday' => $birthday, 'today' => $now,
            'factor' => engine(profile(['birthday' => $birthday]), $noDbTips)->calculateAgeFactor(),
        ];
    }
}
Carbon::setTestNow();

// --- Final probability: every achievable base × age × cycle × symptom ----------------------------
foreach (range(-6, 2) as $day) {
    $base = $engine->getBaseProbability($day);
    foreach ($ageValues as $age) {
        foreach ($cycleValues as $cycle) {
            foreach ($symptomValues as $symptom) {
                $p = $engine->calculateFinalProbability($base, $age, $cycle, $symptom);
                $percent = round($p * 100, 1);
                // [day, age, cycle, symptom, final, round(final*100, 2), "{round(final*100, 1)}"]
                $out['final_probability'][] = [$day, $age, $cycle, $symptom, $p, round($p * 100, 2), "{$percent}"];
            }
        }
    }
}

// --- Text flags ---------------------------------------------------------------------------------
$probabilities = [0.0, 0.0595, 0.1, 0.125, 0.12345, 0.19635, 0.2244, 0.33 * 0.85 * 0.7 * 1.3, 0.35, 0.000049, 0.00005, 1 / 3];
$i = 0;
foreach (CycleSubphase::cases() as $subphase) {
    foreach (CyclePhase::cases() as $phase) {
        $variability = CycleVariability::cases()[$i % 3];
        $p = $probabilities[$i % count($probabilities)];
        $flags = [$i % 2 === 0, $i % 3 === 0, $i % 5 < 2];
        $out['text_flags'][] = [
            'phase' => $phase->value, 'subphase' => $subphase->value, 'fertile' => $flags[0],
            'pms' => $flags[1], 'period_tomorrow' => $flags[2], 'probability' => $p,
            'variability' => $variability->value,
            'flags' => $engine->generateTextFlags($phase, $subphase, $flags[0], $flags[1], $flags[2], $p, $variability),
        ];
        $i++;
    }
}

// --- Code-fallback daily tips -------------------------------------------------------------------
$tipLogs = [null, [], ['headache_intensity' => 'mild'], ['pelvic_pain_intensity' => 'mild'],
    ['stomach_ache_intensity' => 'severe', 'sleep_quality' => 'bad'], ['sleep_quality' => 'good'],
    ['moods' => '["calm","sad"]'], ['moods' => '["anxious"]'], ['moods' => '"sad"'], ['bloating_intensity' => 'mild'],
    ['fatigue' => 1], ['fatigue' => 0], ['headache_intensity' => 'x', 'pelvic_pain_intensity' => 'x',
        'sleep_quality' => 'bad', 'moods' => '["sad"]', 'bloating_intensity' => 'x', 'fatigue' => 1]];
$tipCase = fn (CyclePhase $phase, CycleSubphase $subphase, ?array $attrs): array => [
    'phase' => $phase->value, 'subphase' => $subphase->value, 'log' => $attrs === null ? null : (object) $attrs,
    'tips' => $engine->generateDailyTips($phase, $subphase, $attrs === null ? null : dailyLog($attrs)),
];
foreach (CyclePhase::cases() as $phase) {
    foreach (CycleSubphase::cases() as $subphase) {
        $out['fallback_tips'][] = $tipCase($phase, $subphase, null);
    }
}
foreach ($tipLogs as $attrs) {
    $out['fallback_tips'][] = $tipCase(CyclePhase::LUTEAL, CycleSubphase::MID_LUTEAL, $attrs);
}

// --- Cycle day: LMP anchor, no history (backward extrapolation quirk) ----------------------------
foreach ([21, 26, 28, 30, 35, 45] as $length) {
    $lmp = '2026-03-15';
    $e = engine(profile(['last_period_start' => $lmp]), $noDbTips);
    foreach (range(-100, 100, 1) as $offset) {
        $date = Carbon::parse($lmp)->addDays($offset);
        $out['cycle_day'][] = [
            'lmp' => $lmp, 'length' => $length, 'date' => $date->toDateString(),
            'cycle_day' => $e->calculateCycleDay($date, $length, collect()),
        ];
    }
}

// --- Recommendation::toTip + DailyTipLocalizer --------------------------------------------------
$rows = [
    ['type' => 'hydration', 'title' => null, 'text' => '{"fa":"آب","en":"Water"}'],
    ['type' => '', 'title' => '{"fa":"عنوان"}', 'text' => '{"en":"Only en"}'],
    ['type' => '0', 'title' => '{"fa":"","en":""}', 'text' => '{"fa":"فقط فا"}'],
    ['type' => 'unknown_kind', 'title' => '{"en":"Custom","fa":null}', 'text' => '{"fa":"","en":"x"}'],
    ['type' => 'pms', 'title' => '{"en":12}', 'text' => '{"fa":5,"en":true}'],
    ['type' => 'sleep', 'title' => '["list"]', 'text' => '{"fa":null,"en":null}'],
];
$localizer = new DailyTipLocalizer;
foreach ($rows as $attrs) {
    $rec = new Recommendation;
    $rec->setRawAttributes($attrs);
    $tip = $rec->toTip();
    $localized = [];
    foreach (['fa', 'en', 'ar'] as $locale) {
        $localized[$locale] = $localizer->localize([$tip], $locale);
    }
    $out['to_tip'][] = ['row' => $attrs, 'tip' => $tip, 'localized' => $localized];
}
// Code-fallback tips (no title) through the localizer.
$fallback = $engine->generateDailyTips(CyclePhase::MENSTRUATION, CycleSubphase::MENSTRUATION, dailyLog(['fatigue' => 1]));
foreach (['fa', 'en', 'ar'] as $locale) {
    $out['localize_fallback'][$locale] = $localizer->localize($fallback, $locale);
}

echo json_encode($out, JSON_UNESCAPED_UNICODE | JSON_PRESERVE_ZERO_FRACTION), "\n";
