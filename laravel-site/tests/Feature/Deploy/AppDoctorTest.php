<?php

declare(strict_types=1);

use App\Console\Commands\AppDoctor;
use App\Listeners\MailFailedJob;
use Illuminate\Console\Scheduling\Event;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Contracts\Queue\Job;
use Illuminate\Queue\Events\JobFailed;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Mail;
use Illuminate\Support\Facades\Storage;

/**
 * L10-02: production health check, queue heartbeat, backup schedule, job-failure alert mail. The full green run needs
 * cached config/routes/events + a real build and is done on a production-like environment (docs/GO-LIVE.md); here
 * each check's verdict is asserted on its own row.
 */

/** Status of one row in the doctor's (last) table ('ok' | 'warn' | 'FAIL'), null when the row is missing. */
function doctorRow(string $output, string $check): ?string
{
    $status = null;
    foreach (explode("\n", $output) as $line) {
        $cells = array_map('trim', explode('|', $line));
        if (count($cells) >= 4 && $cells[1] === $check) {
            $status = $cells[2]; // Artisan::output() can hold several runs of one test: keep the last
        }
    }

    return $status;
}

function runDoctor(array $options = []): array
{
    $code = Artisan::call('app:doctor', $options);

    return [$code, Artisan::output()];
}

function productionLike(): void
{
    config([
        'app.env' => 'production',
        'app.debug' => false,
        'app.url' => 'https://ritme.test',
        'app.canonical_redirect' => true,
        'session.driver' => 'file',
        'queue.default' => 'database',
        'mail.default' => 'smtp',
        'mail.mailers.smtp.host' => 'mail.ritme.test',
        'mail.from.address' => 'no-reply@ritme.test',
        'logging.default' => 'daily',
        'logging.channels.daily.level' => 'warning',
        'logging.alerts.mail_to' => 'ops@ritme.test',
    ]);
}

beforeEach(function (): void {
    $this->heartbeat = storage_path(AppDoctor::HEARTBEAT);
    $this->heartbeatBackup = is_file($this->heartbeat) ? file_get_contents($this->heartbeat) : null;
    Storage::fake('local');
});

afterEach(function (): void {
    $this->heartbeatBackup === null ? @unlink($this->heartbeat) : file_put_contents($this->heartbeat, $this->heartbeatBackup);
});

it('fails loudly on a development configuration and names every check', function (): void {
    [$code, $output] = runDoctor();

    expect($code)->toBe(1)
        ->and($output)->toContain('Not healthy')
        ->and(doctorRow($output, 'APP_ENV=production'))->toBe('FAIL')
        ->and(doctorRow($output, 'APP_URL https'))->toBe('FAIL')
        ->and(doctorRow($output, 'Session driver'))->toBe('FAIL')   // array in tests
        ->and(doctorRow($output, 'Queue connection'))->toBe('warn') // sync in tests
        ->and(doctorRow($output, 'Mail configured'))->toBe('FAIL')
        ->and(doctorRow($output, 'robots.txt (production rules)'))->toBe('FAIL'); // Disallow: / outside production

    foreach ([
        'APP_DEBUG off', 'APP_KEY set and valid', 'APP_CANONICAL_REDIRECT on', 'Database + migrations', 'Cache store',
        'Queue worker ran (≤ 5 min)', 'Writable paths', 'Optimize caches present', 'Log rotation + level',
        'Build assets (Vite, build-id, sw.js)', 'Critical CSS manifest matches build', 'Health route /up',
        'sitemap.xml reachable', 'Home page renders', 'Backup recent (local, ≤ 1 d)', 'Operator alerts (OPS_ALERT_EMAIL)',
        'ADMIN_MFA_ROLES covers PII roles',
    ] as $check) {
        expect(doctorRow($output, $check))->not->toBeNull("row «{$check}» missing");
    }
});

it('passes the configuration, http, queue, backup and mail checks on a production-like setup', function (): void {
    productionLike();
    AppDoctor::recordHeartbeat();
    Storage::disk('local')->put('ritme-site/'.now()->subHours(3)->format('Y-m-d-H-i-s').'.zip', 'zip');

    [, $output] = runDoctor();

    foreach ([
        'APP_ENV=production', 'APP_DEBUG off', 'APP_KEY set and valid', 'APP_URL https', 'APP_CANONICAL_REDIRECT on',
        'Database + migrations', 'Session driver', 'Queue connection', 'Queue worker ran (≤ 5 min)',
        'Queue backlog (≤ 5 min)', 'Writable paths', 'Log rotation + level', 'Health route /up',
        'robots.txt (production rules)', 'sitemap.xml reachable', 'Backup recent (local, ≤ 1 d)', 'Mail configured',
        'Operator alerts (OPS_ALERT_EMAIL)', 'ADMIN_MFA_ROLES covers PII roles',
    ] as $check) {
        expect(doctorRow($output, $check))->toBe('ok', "«{$check}»:\n{$output}");
    }
    // Tests never run with cached config/routes, so this row can only be green on a real host.
    expect(doctorRow($output, 'Optimize caches present'))->toBe('FAIL')
        ->and($output)->toContain('php artisan optimize');
});

it('fails the queue checks without a fresh heartbeat or with a stuck job', function (): void {
    productionLike();
    @unlink(storage_path(AppDoctor::HEARTBEAT));

    [, $output] = runDoctor();
    expect(doctorRow($output, 'Queue worker ran (≤ 5 min)'))->toBe('FAIL')
        ->and($output)->toContain('schedule:run every minute');

    file_put_contents(storage_path(AppDoctor::HEARTBEAT), (string) (time() - 600));
    DB::table('jobs')->insert(['queue' => 'default', 'payload' => '{}', 'attempts' => 0, 'reserved_at' => null, 'available_at' => time() - 900, 'created_at' => time() - 900]);

    [, $output] = runDoctor(['--minutes' => 5]);
    expect(doctorRow($output, 'Queue worker ran (≤ 5 min)'))->toBe('FAIL')
        ->and(doctorRow($output, 'Queue backlog (≤ 5 min)'))->toBe('FAIL');

    [, $output] = runDoctor(['--minutes' => 20]);
    expect(doctorRow($output, 'Queue worker ran (≤ 20 min)'))->toBe('ok')
        ->and(doctorRow($output, 'Queue backlog (≤ 20 min)'))->toBe('ok');
});

it('fails the backup check when the newest backup is too old or missing', function (): void {
    productionLike();

    [, $output] = runDoctor();
    expect(doctorRow($output, 'Backup recent (local, ≤ 1 d)'))->toBe('FAIL')->and($output)->toContain('backup:run');

    Storage::disk('local')->put('ritme-site/'.now()->subDays(3)->format('Y-m-d-H-i-s').'.zip', 'zip');
    [, $output] = runDoctor();
    expect(doctorRow($output, 'Backup recent (local, ≤ 1 d)'))->toBe('FAIL');
});

it('flags debug, a missing key and a narrowed MFA role list', function (): void {
    productionLike();
    config(['app.debug' => true, 'app.key' => '', 'filament.admin.mfa_required_roles' => ['super-admin']]);

    [$code, $output] = runDoctor();

    expect($code)->toBe(1)
        ->and(doctorRow($output, 'APP_DEBUG off'))->toBe('FAIL')
        ->and(doctorRow($output, 'APP_KEY set and valid'))->toBe('FAIL')
        ->and(doctorRow($output, 'ADMIN_MFA_ROLES covers PII roles'))->toBe('FAIL');
});

it('sends a test mail on request', function (): void {
    productionLike();
    config(['mail.default' => 'array']);

    [, $output] = runDoctor(['--mail' => 'owner@ritme.test']);

    expect(doctorRow($output, 'Test mail sent'))->toBe('ok')
        ->and(app('mailer')->getSymfonyTransport()->messages())->toHaveCount(1);
});

it('schedules the heartbeat on the queue worker and the nightly backups', function (): void {
    $events = collect(app(Schedule::class)->events());
    $worker = $events->first(static fn (Event $event): bool => str_contains((string) $event->command, 'queue:work'));
    $command = static fn (string $name): ?Event => $events->first(static fn (Event $event): bool => str_contains((string) $event->command, $name));

    expect($worker)->not->toBeNull()
        ->and($command('backup:clean')?->expression)->toBe('50 1 * * *')
        ->and($command('backup:run')?->expression)->toBe('10 2 * * *')
        ->and($command('backup:monitor')?->expression)->toBe('15 8 * * *');

    @unlink(storage_path(AppDoctor::HEARTBEAT));
    $worker->finish(app(), 0);
    expect(AppDoctor::heartbeatAge())->not->toBeNull()->toBeLessThan(5);

    @unlink(storage_path(AppDoctor::HEARTBEAT));
    $worker->finish(app(), 1);
    expect(AppDoctor::heartbeatAge())->toBeNull();
});

it('backs up the database and the media folders to the private disk', function (): void {
    expect(config('backup.backup.source.databases'))->toBe([config('database.default')])
        ->and(config('backup.backup.source.files.include'))->toContain(public_path('media'), storage_path('app/pending'))
        ->and(config('backup.backup.destination.disks'))->toBe(['local'])
        ->and(config('filesystems.disks.local.root'))->toBe(storage_path('app/private'))
        // No alert address in tests: no channel, so nothing is mailed.
        ->and(array_filter(config('backup.notifications.notifications')))->toBe([]);
});

function failedJobEvent(string $message): JobFailed
{
    $job = Mockery::mock(Job::class);
    $job->shouldReceive('resolveName')->andReturn('App\\Jobs\\SendThing');
    $job->shouldReceive('getQueue')->andReturn('default');
    $job->shouldReceive('attempts')->andReturn(3);

    return new JobFailed('database', $job, new RuntimeException($message));
}

it('mails the operator once per job class and hour when a job fails, without personal data', function (): void {
    config(['logging.alerts.mail_to' => 'ops@ritme.test', 'mail.default' => 'array', 'app.url' => 'https://ritme.test']);
    Cache::flush();

    event(failedJobEvent('SMTP 550 for sara@example.org, mobile 0912 345 6789'));
    event(failedJobEvent('again'));

    $messages = app('mailer')->getSymfonyTransport()->messages();
    expect($messages)->toHaveCount(1);
    $email = $messages->first()->getOriginalMessage();
    expect($email->getSubject())->toContain('Job failed on ritme.test: SendThing')
        ->and($email->getTo()[0]->getAddress())->toBe('ops@ritme.test')
        ->and($email->getTextBody())->toContain('[email]')->toContain('[number]')
        ->not->toContain('sara@example.org')->not->toContain('345');
});

it('stays silent without an alert address', function (): void {
    config(['logging.alerts.mail_to' => null, 'mail.default' => 'array']);
    Mail::fake();

    event(failedJobEvent('boom'));

    Mail::assertNothingOutgoing();
    expect(MailFailedJob::mask('x'))->toBe('x');
});
