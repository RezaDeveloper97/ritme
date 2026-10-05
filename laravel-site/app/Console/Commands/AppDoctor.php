<?php

declare(strict_types=1);

namespace App\Console\Commands;

use App\View\Components\Layout\Assets;
use Illuminate\Console\Command;
use Illuminate\Contracts\Http\Kernel as HttpKernel;
use Illuminate\Database\Migrations\Migrator;
use Illuminate\Foundation\Vite;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Facade;
use Illuminate\Support\Facades\Mail;
use Illuminate\Support\Facades\Schema;
use Illuminate\Support\Str;
use Spatie\Backup\BackupDestination\BackupDestination;
use Spatie\Backup\Tasks\Monitor\HealthChecks\MaximumAgeInDays;
use Throwable;

/**
 * Production health check (L10-02, docs/GO-LIVE.md): run it on the host after every deploy and whenever something
 * looks off. Every check prints ok / warn / FAIL with a short reason; any FAIL → exit code 1. Read-only except a
 * cache round trip and the optional `--mail` test message. HTTP checks run in-process through the HTTP kernel (no
 * network): `/up`, `/robots.txt`, `/sitemap.xml`, the home page.
 *
 * Queue liveness: the scheduler's queue worker (routes/console.php) touches `storage/framework/cache/queue-heartbeat`
 * each time it finishes; the doctor wants that file younger than `--minutes` AND no ready job waiting longer.
 */
final class AppDoctor extends Command
{
    /** Outside the file cache's data folder, so cache:clear keeps it; git-ignored. */
    public const HEARTBEAT = 'framework/cache/queue-heartbeat';

    /** Roles that read personal data and must use TOTP (docs/SECURITY.md §5). */
    private const PII_ROLES = ['super-admin', 'shop-manager', 'directory-manager', 'support'];

    protected $signature = 'app:doctor
        {--minutes=5 : Max age of the queue heartbeat and of the oldest waiting job}
        {--mail= : Also send a test e-mail to this address through the configured mailer}';

    protected $description = 'Production health check: debug, key, https, drivers, queue + cron, paths, caches, robots, sitemap, assets, backups, mail';

    /** @var list<array{0: string, 1: 'ok'|'warn'|'fail', 2: string}> */
    private array $results = [];

    /** Called by the scheduler when the minute's queue worker exits normally. */
    public static function recordHeartbeat(): void
    {
        $file = storage_path(self::HEARTBEAT);
        @file_put_contents($file, (string) time());
    }

    /** Seconds since the last heartbeat, null when there is none. */
    public static function heartbeatAge(): ?int
    {
        $file = storage_path(self::HEARTBEAT);
        if (! is_file($file)) {
            return null;
        }
        $stamp = (int) trim((string) @file_get_contents($file));
        $stamp = $stamp > 0 ? $stamp : (int) filemtime($file);

        return max(0, time() - $stamp);
    }

    public function handle(): int
    {
        $minutes = max(1, (int) $this->option('minutes'));

        $this->checkEnvironment();
        $this->checkDatabase();
        $this->checkDrivers();
        $this->checkQueue($minutes);
        $this->checkWritablePaths();
        $this->checkOptimizeCaches();
        $this->checkLogging();
        $this->checkBuild();
        $this->checkHttp();
        $this->checkBackups();
        $this->checkMail();
        $this->checkAdmin();

        $this->table(['Check', 'Status', 'Detail'], array_map(static fn (array $row): array => [
            $row[0],
            match ($row[1]) {
                'ok' => '<info>ok</info>',
                'warn' => '<comment>warn</comment>',
                'fail' => '<error>FAIL</error>',
            },
            $row[2],
        ], $this->results));

        $counts = array_count_values(array_column($this->results, 1));
        $fails = $counts['fail'] ?? 0;
        $summary = sprintf('%d ok, %d warn, %d FAIL', $counts['ok'] ?? 0, $counts['warn'] ?? 0, $fails);
        $fails === 0 ? $this->info('Healthy — '.$summary) : $this->error('Not healthy — '.$summary);

        return $fails === 0 ? self::SUCCESS : self::FAILURE;
    }

    /**
     * @param  'ok'|'warn'|'fail'  $status
     */
    private function result(string $check, string $status, string $detail = ''): void
    {
        $this->results[] = [$check, $status, $detail];
    }

    private function pass(string $check, bool $ok, string $detail = '', bool $required = true): void
    {
        $this->result($check, $ok ? 'ok' : ($required ? 'fail' : 'warn'), $detail);
    }

    private function checkEnvironment(): void
    {
        $env = (string) config('app.env');
        $this->pass('APP_ENV=production', $env === 'production', $env);
        $this->pass('APP_DEBUG off', ! config('app.debug'), config('app.debug') ? 'debug pages leak secrets — set APP_DEBUG=false' : 'false');

        $key = (string) config('app.key');
        $raw = str_starts_with($key, 'base64:') ? base64_decode(substr($key, 7), true) : $key;
        $keyOk = is_string($raw) && strlen($raw) === (str_contains(strtolower((string) config('app.cipher')), '128') ? 16 : 32);
        $this->pass('APP_KEY set and valid', $keyOk, $keyOk ? (string) config('app.cipher') : 'missing or wrong length — php artisan key:generate (only on a new install)');

        $url = (string) config('app.url');
        $host = (string) parse_url($url, PHP_URL_HOST);
        $this->pass('APP_URL https', str_starts_with($url, 'https://') && $host !== '', $url);
        $this->pass('APP_CANONICAL_REDIRECT on', (bool) config('app.canonical_redirect'), config('app.canonical_redirect') ? 'one-hop 301 to '.$url : 'http / www variants are not redirected', required: false);
        $this->pass('Not in maintenance mode', ! $this->laravel->isDownForMaintenance(), $this->laravel->isDownForMaintenance() ? 'php artisan up' : '', required: false);
    }

    private function checkDatabase(): void
    {
        try {
            DB::connection()->getPdo();
            /** @var Migrator $migrator */
            $migrator = $this->laravel->make('migrator');
            if (! $migrator->repositoryExists()) {
                $this->result('Database + migrations', 'fail', 'no migrations table — php artisan app:install');

                return;
            }
            $files = $migrator->getMigrationFiles([database_path('migrations'), ...$migrator->paths()]);
            $pending = array_diff(array_keys($files), $migrator->getRepository()->getRan());
            $driver = (string) DB::connection()->getDriverName();
            $this->pass('Database + migrations', $pending === [], $pending === []
                ? $driver.', '.count($files).' migrations ran'
                : count($pending).' pending — php artisan migrate --force (app:upgrade does it)');
        } catch (Throwable $e) {
            $this->result('Database + migrations', 'fail', Str::limit($e->getMessage(), 120));
        }
    }

    private function checkDrivers(): void
    {
        $store = (string) config('cache.default');
        $storeDriver = (string) config("cache.stores.{$store}.driver");
        try {
            $key = 'app-doctor:'.Str::random(8);
            Cache::put($key, 'ok', 60);
            $works = Cache::get($key) === 'ok';
            Cache::forget($key);
        } catch (Throwable) {
            $works = false;
        }
        $this->pass('Cache store', $works && ! in_array($storeDriver, ['array', 'null'], true), "{$store} ({$storeDriver})".($works ? '' : ' — read/write failed'));

        $session = (string) config('session.driver');
        $this->result('Session driver', match (true) {
            in_array($session, ['file', 'database', 'redis', 'memcached'], true) => 'ok',
            $session === 'cookie' => 'warn',
            default => 'fail',
        }, $session.(config('session.secure') ? ', secure cookie' : ', cookie not marked secure'));
    }

    private function checkQueue(int $minutes): void
    {
        $connection = (string) config('queue.default');
        $driver = (string) config("queue.connections.{$connection}.driver");
        $this->result('Queue connection', match ($driver) {
            'database', 'redis', 'beanstalkd', 'sqs' => 'ok',
            'sync' => 'warn',
            default => 'fail',
        }, "{$connection} ({$driver})".($driver === 'sync' ? ' — mails and image optimisation run inside the request' : ''));

        $age = self::heartbeatAge();
        $this->pass("Queue worker ran (≤ {$minutes} min)", $age !== null && $age <= $minutes * 60, $age === null
            ? 'no heartbeat yet — is the cron line (schedule:run every minute) installed?'
            : 'last run '.self::ago($age).' ago (cron → schedule:run → queue:work)');

        if ($driver === 'database') {
            try {
                $table = (string) config("queue.connections.{$connection}.table", 'jobs');
                $oldest = DB::table($table)->whereNull('reserved_at')->where('available_at', '<=', time())->min('available_at');
                $wait = is_numeric($oldest) ? time() - (int) $oldest : 0;
                $this->pass("Queue backlog (≤ {$minutes} min)", $wait <= $minutes * 60, $oldest === null
                    ? 'no waiting jobs'
                    : 'oldest ready job waits '.self::ago($wait));
            } catch (Throwable $e) {
                $this->result("Queue backlog (≤ {$minutes} min)", 'fail', Str::limit($e->getMessage(), 120));
            }
        }

        try {
            $failedTable = (string) config('queue.failed.table', 'failed_jobs');
            $failed = Schema::hasTable($failedTable)
                ? DB::table($failedTable)->where('failed_at', '>=', now()->subDay())->count()
                : 0;
            $this->pass('Failed jobs (24 h)', $failed === 0, $failed === 0 ? 'none' : $failed.' — php artisan queue:failed / queue:retry all', required: false);
        } catch (Throwable $e) {
            $this->result('Failed jobs (24 h)', 'warn', Str::limit($e->getMessage(), 120));
        }
    }

    private function checkWritablePaths(): void
    {
        $paths = [
            storage_path('app'), storage_path('app/private'), storage_path('app/pending'), storage_path('framework/cache'),
            storage_path('framework/sessions'), storage_path('framework/views'), storage_path('logs'),
            base_path('bootstrap/cache'), (string) config('filesystems.disks.public.root', public_path('media')),
            (string) config('backup.backup.temporary_directory', storage_path('app/backup-temp')),
        ];
        $bad = [];
        foreach ($paths as $path) {
            // The backup temp folder is created on demand: its parent must be writable.
            $probe = is_dir($path) ? $path : dirname($path);
            if (! is_dir($probe) || ! is_writable($probe)) {
                $bad[] = self::short($path);
            }
        }
        $this->pass('Writable paths', $bad === [], $bad === [] ? count($paths).' folders (storage, bootstrap/cache, media, pending, backups)' : 'not writable: '.implode(', ', $bad));
    }

    private function checkOptimizeCaches(): void
    {
        $missing = [];
        if (! $this->laravel->configurationIsCached()) {
            $missing[] = 'config';
        }
        if (! $this->laravel->routesAreCached()) {
            $missing[] = 'routes';
        }
        if (! $this->laravel->eventsAreCached()) {
            $missing[] = 'events';
        }
        $filament = (string) config('filament.cache_path', base_path('bootstrap/cache/filament'));
        if (! is_dir($filament) || (glob($filament.'/*') ?: []) === []) {
            $missing[] = 'filament';
        }
        $this->pass('Optimize caches present', $missing === [], $missing === []
            ? 'config, routes, events, filament'
            : 'missing: '.implode(', ', $missing).' — php artisan optimize && php artisan filament:optimize');
    }

    private function checkLogging(): void
    {
        $channel = (string) config('logging.default');
        $channels = $channel === 'stack' ? (array) config('logging.channels.stack.channels', []) : [$channel];
        $daily = in_array('daily', $channels, true);
        $level = (string) config('logging.channels.'.($daily ? 'daily' : $channel).'.level', 'debug');
        $this->pass('Log rotation + level', $daily && $level !== 'debug', ($daily ? 'daily, '.(int) config('logging.channels.daily.days').' days' : $channel.' (not rotated — LOG_CHANNEL=daily)').", level {$level}", required: false);
    }

    private function checkBuild(): void
    {
        $missing = [];
        foreach (['build/manifest.json', 'build/build-id.json', Assets::CRITICAL_DIR.'/manifest.json', 'sw.js'] as $file) {
            if (! is_file(public_path($file))) {
                $missing[] = $file;
            }
        }
        $buildId = json_decode((string) @file_get_contents(public_path('build/build-id.json')), true);
        $id = is_array($buildId) && is_string($buildId['build_id'] ?? null) ? $buildId['build_id'] : '?';
        $this->pass('Build assets (Vite, build-id, sw.js)', $missing === [], $missing === [] ? 'build '.$id : 'missing: '.implode(', ', $missing));

        $manifest = Assets::criticalManifest($this->laravel->make(Vite::class));
        $this->pass('Critical CSS manifest matches build', $manifest !== null, $manifest !== null
            ? count($manifest['templates']).' templates'
            : 'missing or cut from another build — pages fall back to the blocking stylesheet; rebuild the package');
    }

    private function checkHttp(): void
    {
        $root = rtrim((string) config('app.url'), '/');

        $up = $this->fetch($root.'/up');
        $this->pass('Health route /up', $up[0] === 200, 'HTTP '.$up[0]);

        [$status, $type, $body] = $this->fetch($root.'/robots.txt');
        $blanket = preg_match('/^\s*Disallow:\s*\/\s*$/mi', $body) === 1;
        $sitemapLine = str_contains($body, 'Sitemap: '.$root.'/sitemap.xml');
        $static = is_file(public_path('robots.txt'));
        $this->pass('robots.txt (production rules)', $status === 200 && ! $blanket && $sitemapLine && ! $static, match (true) {
            $status !== 200 => 'HTTP '.$status,
            $static => 'a static public/robots.txt shadows the managed one — delete it',
            $blanket => '«Disallow: /» blocks the whole site (APP_ENV not production or admin indexing switch off)',
            ! $sitemapLine => 'no «Sitemap: '.$root.'/sitemap.xml» line',
            default => 'allows crawling, lists the sitemap',
        });

        [$status, $type, $body] = $this->fetch($root.'/sitemap.xml');
        $count = preg_match_all('#<loc>([^<]+)</loc>#', $body, $locs);
        $foreign = array_filter($locs[1], static fn (string $loc): bool => ! str_starts_with(html_entity_decode($loc), $root.'/'));
        $this->pass('sitemap.xml reachable', $status === 200 && str_contains($type, 'xml') && $count > 0 && $foreign === [], match (true) {
            $status !== 200 => 'HTTP '.$status,
            $count === 0 => 'no <loc> entries',
            $foreign !== [] => 'URLs outside APP_URL, e.g. '.reset($foreign).' — cache:ns bump sitemap',
            default => $count.' entries under '.$root,
        });

        [$status, , $body] = $this->fetch($root.'/');
        $this->pass('Home page renders', $status === 200 && str_contains($body, '<h1'), 'HTTP '.$status.($status === 200 && ! str_contains($body, '<style') ? ' — no inline critical CSS (stale page cache? cache:ns bump pages)' : ''));
    }

    /**
     * @return array{0: int, 1: string, 2: string} status, content type, body
     */
    private function fetch(string $url): array
    {
        $original = $this->laravel->bound('request') ? $this->laravel->make('request') : null;
        try {
            $request = Request::create($url, 'GET', server: [
                'HTTP_USER_AGENT' => 'RitmeDoctor/1.0 (in-process)',
                'HTTP_ACCEPT' => 'text/html,application/xml,text/plain,*/*',
                'HTTP_CACHE_CONTROL' => 'no-cache',
            ]);
            $kernel = $this->laravel->make(HttpKernel::class);
            $response = $kernel->handle($request);
            $kernel->terminate($request, $response);
            $content = $response->getContent();

            return [$response->getStatusCode(), strtolower((string) $response->headers->get('Content-Type', '')), is_string($content) ? $content : ''];
        } catch (Throwable $e) {
            return [0, '', $e->getMessage()];
        } finally {
            if ($original !== null) {
                $this->laravel->instance('request', $original);
            }
            Facade::clearResolvedInstance('request');
        }
    }

    private function checkBackups(): void
    {
        /** @var list<array{name?: string, disks?: list<string>, health_checks?: array<string, int>}> $monitors */
        $monitors = (array) config('backup.monitor_backups', []);
        foreach ($monitors as $monitor) {
            $name = (string) ($monitor['name'] ?? config('backup.backup.name'));
            $maxDays = (int) ($monitor['health_checks'][MaximumAgeInDays::class] ?? 1);
            foreach ((array) ($monitor['disks'] ?? []) as $disk) {
                $label = "Backup recent ({$disk}, ≤ {$maxDays} d)";
                try {
                    $destination = BackupDestination::create((string) $disk, $name);
                    if (! $destination->isReachable()) {
                        $this->result($label, 'fail', 'destination not reachable');

                        continue;
                    }
                    $newest = $destination->newestBackup();
                    if ($newest === null) {
                        $this->result($label, 'fail', 'no backup yet — php artisan backup:run (the scheduler runs it daily at 02:10)');

                        continue;
                    }
                    $age = (int) $newest->date()->diffInSeconds(now(), true);
                    $this->pass($label, $age <= $maxDays * 86400, basename($newest->path()).', '.self::ago($age).' old, '
                        .self::megabytes($newest->sizeInBytes()).'; '.$destination->backups()->count().' kept, '.self::megabytes($destination->usedStorage()).' in total');
                } catch (Throwable $e) {
                    $this->result($label, 'fail', Str::limit($e->getMessage(), 120));
                }
            }
        }
        if ($monitors === []) {
            $this->result('Backup recent', 'fail', 'config/backup.php monitor_backups is empty');
        }
    }

    private function checkMail(): void
    {
        $mailer = (string) config('mail.default');
        $transport = (string) config("mail.mailers.{$mailer}.transport");
        $from = (string) config('mail.from.address');
        $problems = [];
        if (in_array($transport, ['log', 'array', ''], true)) {
            $problems[] = "MAIL_MAILER={$mailer} sends nothing";
        }
        if ($transport === 'smtp' && in_array((string) config("mail.mailers.{$mailer}.host"), ['', '127.0.0.1', 'localhost'], true)) {
            $problems[] = 'MAIL_HOST is local';
        }
        if ($from === '' || filter_var($from, FILTER_VALIDATE_EMAIL) === false || str_ends_with($from, '@example.com') || str_contains($from, '<')) {
            $problems[] = 'MAIL_FROM_ADDRESS is a placeholder';
        }
        $this->pass('Mail configured', $problems === [], $problems === [] ? "{$mailer} ({$transport}), from {$from}" : implode('; ', $problems));

        $alert = config('logging.alerts.mail_to');
        $alertOk = is_string($alert) && filter_var($alert, FILTER_VALIDATE_EMAIL) !== false;
        $this->pass('Operator alerts (OPS_ALERT_EMAIL)', $alertOk, $alertOk ? 'failed jobs + failed/unhealthy backups → '.$alert : 'not set — failures are only logged', required: false);

        $to = $this->option('mail');
        if (is_string($to) && $to !== '') {
            try {
                Mail::raw('Ritme app:doctor test message — mail delivery works ('.now()->toDateTimeString().').', static function ($message) use ($to): void {
                    $message->to($to)->subject('['.config('app.name').'] app:doctor test mail');
                });
                $this->result('Test mail sent', 'ok', $to.' — check the inbox (and spam folder)');
            } catch (Throwable $e) {
                $this->result('Test mail sent', 'fail', Str::limit($e->getMessage(), 160));
            }
        }
    }

    private function checkAdmin(): void
    {
        $roles = (array) config('filament.admin.mfa_required_roles', []);
        $missing = array_values(array_diff(self::PII_ROLES, $roles));
        $this->pass('ADMIN_MFA_ROLES covers PII roles', $missing === [], $missing === [] ? implode(',', $roles) : 'missing: '.implode(',', $missing));
    }

    private static function ago(int $seconds): string
    {
        return match (true) {
            $seconds < 120 => $seconds.' s',
            $seconds < 7200 => intdiv($seconds, 60).' min',
            $seconds < 172800 => intdiv($seconds, 3600).' h',
            default => intdiv($seconds, 86400).' d',
        };
    }

    private static function megabytes(float $bytes): string
    {
        return number_format($bytes / 1048576, 1).' MB';
    }

    private static function short(string $path): string
    {
        return str_replace(dirname(base_path()).'/', '', $path);
    }
}
