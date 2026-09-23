<?php
// Records Laravel's validation behaviour for the Go rule engine's golden tests.
// Run from the repo root (local php + backend/vendor; nothing on disk is touched,
// the database is an in-memory sqlite):
//
//   php backend-go/internal/platform/validation/testdata/capture.php
//
// Inputs (hand-written, shared with the Go tests):
//   engine_cases.json  rule sets + data run through Validator::make directly
//   http_cases.json    real requests through the HTTP kernel (TrimStrings,
//                      ConvertEmptyStringsToNull, SetLocale, FormRequests, controllers)
// Outputs:
//   engine_golden.json {name: {passes, errors, first, validated}}
//   http_golden.json   {name: {status, body}} (body = the raw response bytes)
//   rulesets.json      the FormRequest rules()/messages() (Rule::in stringified)
//                      and the enum value lists the inline controller rules use
//
// Clock: 2026-09-23 10:00 Asia/Tehran (CONTRACT_TODAY). Languages: fa (default), en, ar.
// Inputs avoid what docs/go-migration/deviations.md D-09 excludes from the port
// (strtotime relative phrases and single-letter military zones such as "x").
$dir = __DIR__;
$base = dirname($dir, 5).'/backend';
foreach ([
    'APP_ENV' => 'production', 'APP_DEBUG' => 'false', 'DB_CONNECTION' => 'sqlite', 'DB_DATABASE' => ':memory:',
    'CACHE_STORE' => 'array', 'SESSION_DRIVER' => 'array', 'QUEUE_CONNECTION' => 'sync', 'SMS_PROVIDER' => 'log',
    'ADMIN_PANEL_ENABLED' => 'false', 'TELEGRAM_BOT_TOKEN' => '', 'APP_URL' => 'https://api.ritme.app',
] as $k => $v) {
    putenv("$k=$v");
    $_ENV[$k] = $v;
    $_SERVER[$k] = $v;
}
require $base.'/vendor/autoload.php';
$app = require $base.'/bootstrap/app.php';
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();

use App\Models\Language;
use App\Models\User;
use App\Services\Language\LanguageRegistry;
use Carbon\Carbon;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Validator;
use Laravel\Passport\Passport;

Artisan::call('migrate', ['--force' => true]);
Carbon::setTestNow(Carbon::parse('2026-09-23 10:00:00', 'Asia/Tehran'));
Language::create(['code' => 'ar', 'name' => 'العربية', 'english_name' => 'Arabic', 'direction' => 'rtl',
    'is_active' => true, 'is_default' => false, 'sort_order' => 3]);
app(LanguageRegistry::class)->flush();

$user = User::create(['name' => 'Capture', 'mobile' => '09120000000']);
$user->reminders()->create(['type' => 'custom', 'title' => 'x', 'recurrence' => 'none', 'is_active' => true]);

$flags = JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES;

// ---------------------------------------------------------------------------
// Engine cases: Validator::make with the case's locale.
$engine = [];
foreach (json_decode(file_get_contents("$dir/engine_cases.json"), true) as $case) {
    app()->setLocale($case['locale']);
    $v = Validator::make($case['data'], $case['rules'], $case['messages'] ?? [], $case['attributes'] ?? []);
    $passes = $v->passes();
    $engine[$case['name']] = [
        'passes' => $passes,
        'errors' => $v->errors()->toArray() ?: new stdClass,
        'first' => $v->errors()->first(),
        'validated' => $passes ? ($v->validated() ?: new stdClass) : null,
    ];
    echo "engine {$case['name']}: ".($passes ? 'passes' : 'fails')."\n";
}
file_put_contents("$dir/engine_golden.json", json_encode($engine, $flags)."\n");

// ---------------------------------------------------------------------------
// Rule sets the Go tests rebuild (FormRequests verbatim, enum lists for inline rules).
$formRequests = [];
foreach ([
    'healthlog_store' => App\Http\Requests\Api\V1\StoreDailyHealthLogRequest::class,
    'pregnancy_onboarding' => App\Http\Requests\Api\V1\StorePregnancyProfileRequest::class,
    'pregnancy_update' => App\Http\Requests\Api\V1\UpdatePregnancyProfileRequest::class,
    'pregnancy_symptoms' => App\Http\Requests\Api\V1\StorePregnancySymptomLogRequest::class,
    'pregnancy_weekly' => App\Http\Requests\Api\V1\StorePregnancyWeeklyLogRequest::class,
    'pregnancy_fetal' => App\Http\Requests\Api\V1\StorePregnancyFetalMovementRequest::class,
] as $set => $class) {
    $req = new $class;
    $rules = [];
    foreach ($req->rules() as $attr => $list) {
        $rules[$attr] = array_map(fn ($r) => is_object($r) ? (string) $r : $r, (array) $list);
    }
    $formRequests[$set] = ['rules' => $rules, 'messages' => $req->messages() ?: new stdClass];
}
file_put_contents("$dir/rulesets.json", json_encode([
    'form_requests' => $formRequests,
    'enums' => [
        'ReminderType' => App\Enums\ReminderType::values(),
        'UserGoal' => App\Enums\UserGoal::values(),
        'SubscriptionType' => App\Enums\SubscriptionType::values(),
        'PregnancyIntention' => App\Enums\PregnancyIntention::values(),
        'ChronicCondition' => App\Enums\ChronicCondition::values(),
        'MessageMode' => array_column(App\Services\MessageSystem\Enums\MessageMode::cases(), 'value'),
    ],
], $flags)."\n");

// ---------------------------------------------------------------------------
// HTTP cases through the real kernel, authenticated as $user.
$kernel = $app->make(Illuminate\Contracts\Http\Kernel::class);
$http = [];
$ip = 1;
foreach (json_decode(file_get_contents("$dir/http_cases.json"), true) as $case) {
    $server = ['HTTP_ACCEPT' => 'application/json', 'HTTP_HOST' => 'api.ritme.app', 'HTTPS' => 'on',
        'REMOTE_ADDR' => '10.0.0.'.($ip++)];
    if ($case['lang'] !== null) {
        $server['HTTP_ACCEPT_LANGUAGE'] = $case['lang'];
    }
    $uri = $case['path'].(isset($case['query']) ? '?'.http_build_query($case['query']) : '');
    $content = null;
    if (isset($case['body'])) {
        $server['CONTENT_TYPE'] = 'application/json';
        $content = json_encode($case['body'], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        if ($content === '[]') {
            $content = '{}';
        }
    }
    $req = Illuminate\Http\Request::create($uri, $case['method'], [], [], [], $server, $content);
    if ($case['lang'] === null) { // Request::create() defaults Accept-Language to "en-us,en;q=0.5"
        $req->headers->remove('Accept-Language');
        $req->server->remove('HTTP_ACCEPT_LANGUAGE');
    }
    Passport::actingAs($user);
    $resp = $kernel->handle($req);
    $http[$case['name']] = ['status' => $resp->getStatusCode(), 'body' => $resp->getContent()];
    $kernel->terminate($req, $resp);
    echo "http {$case['name']}: {$resp->getStatusCode()}\n";
}
file_put_contents("$dir/http_golden.json", json_encode($http, $flags)."\n");
