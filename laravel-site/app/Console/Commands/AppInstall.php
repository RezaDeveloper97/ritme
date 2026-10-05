<?php

declare(strict_types=1);

namespace App\Console\Commands;

use App\Domain\Seo\Sitemap\Sitemaps;
use App\Models\User;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;
use Throwable;

/**
 * First-time installation on shared hosting (L10-01, docs/DEPLOY-CPANEL.md). Web-less: run it from cPanel Terminal
 * (or once from a cron entry): requirement checks → writable folders → .env + APP_KEY → database → migrations →
 * production seeders → first admin → caches → the cron line to add. `--check` only runs the checks (app:upgrade uses
 * it before taking the site down).
 */
final class AppInstall extends Command
{
    public const MIN_PHP = '8.2.0';

    /** Seeders that are safe on a live database (insert-missing only); app:upgrade runs them too. */
    public const PRODUCTION_SEEDERS = ['Database\\Seeders\\SettingsSeeder', 'Database\\Seeders\\AdminRolesSeeder'];

    protected $signature = 'app:install
        {--check : Only run the requirement and environment checks}
        {--public-path= : Layout public_html: the document-root folder that holds index.php (e.g. ../public_html)}
        {--skip-checks : Continue even when a required check fails}
        {--no-admin : Do not offer to create the first admin user}
        {--no-cache : Do not cache config, routes, views and Filament components}
        {--force : Run even when the site is installed already (use app:upgrade for releases)}';

    protected $description = 'Install the site on a new host: checks, .env + key, migrations, seeders, first admin, caches, cron line';

    public function handle(): int
    {
        if (is_string($path = $this->option('public-path')) && $path !== '' && ! $this->savePublicPath($path)) {
            return self::FAILURE;
        }

        $this->ensureDirectories();
        $failures = $this->runChecks();

        if ($this->option('check')) {
            return $failures === 0 ? self::SUCCESS : self::FAILURE;
        }
        if ($failures > 0 && ! $this->option('skip-checks')) {
            $this->error("{$failures} required check(s) failed — fix them (cPanel → Select PHP Version / MultiPHP INI Editor) and run again, or pass --skip-checks.");

            return self::FAILURE;
        }

        if (! $this->ensureEnvironment() || ! $this->databaseAnswers()) {
            return self::FAILURE;
        }

        if ($this->alreadyInstalled() && ! $this->option('force')) {
            $this->warn('The site is installed already (migrations ran and an admin exists). For a new release run: php artisan app:upgrade');

            return self::FAILURE;
        }

        $this->components->task('Migrations', fn (): bool => $this->callSilently('migrate', ['--force' => true]) === self::SUCCESS);
        $this->components->task('Production defaults (settings, FAQ, admin roles)', fn (): bool => $this->callSilently('db:seed', ['--force' => true]) === self::SUCCESS);

        $this->offerAdmin();

        if (! $this->option('no-cache')) {
            self::warmCaches($this);
        }

        $this->newLine();
        $this->info('Installed. Add this cron job (cPanel → Cron Jobs → «Once Per Minute»):');
        $this->line('  '.self::cronLine());
        $this->newLine();
        $this->line('Then: AutoSSL certificate → APP_URL=https://… + APP_CANONICAL_REDIRECT=true → php artisan config:cache.');

        return self::SUCCESS;
    }

    public static function cronLine(): string
    {
        return '* * * * * '.PHP_BINARY.' '.base_path('artisan').' schedule:run >> /dev/null 2>&1';
    }

    /** optimize (config, events, routes, views) + Filament components/icons, fresh cache namespaces, warm sitemap. */
    public static function warmCaches(Command $command): void
    {
        $command->components->task('Caches (config, routes, views, Filament)', static function () use ($command): bool {
            return $command->callSilently('optimize') === self::SUCCESS
                && $command->callSilently('filament:optimize') === self::SUCCESS;
        });
        $command->components->task('Cache namespaces bumped (pages, sitemap, …)', static fn (): bool => $command->callSilently('cache:ns', ['action' => 'bump-all']) === self::SUCCESS);
        $command->components->task('Sitemap warmed', static function (): bool {
            try {
                app(Sitemaps::class)->index();

                return true;
            } catch (Throwable $e) {
                report($e);

                return false;
            }
        });
    }

    /** Requirement + environment checks as a table; returns the number of failed required checks. */
    private function runChecks(): int
    {
        $rows = [];
        $failures = 0;
        foreach ($this->checks() as [$label, $ok, $required, $note]) {
            if (! $ok && $required) {
                $failures++;
            }
            $rows[] = [$label, $ok ? '<info>ok</info>' : ($required ? '<error>FAIL</error>' : '<comment>warn</comment>'), $note];
        }
        $this->table(['Check', '', 'Note'], $rows);

        return $failures;
    }

    /**
     * @return list<array{0: string, 1: bool, 2: bool, 3: string}> [label, ok, required, note]
     */
    private function checks(): array
    {
        $checks = [['PHP >= '.self::MIN_PHP, version_compare(PHP_VERSION, self::MIN_PHP, '>='), true, PHP_VERSION.' ('.PHP_BINARY.')']];

        foreach (['ctype', 'curl', 'dom', 'exif', 'fileinfo', 'filter', 'intl', 'mbstring', 'openssl', 'pdo', 'tokenizer', 'xml'] as $ext) {
            $checks[] = ["ext-{$ext}", extension_loaded($ext), true, ''];
        }

        $driver = (string) config('database.connections.'.config('database.default').'.driver');
        $pdo = 'pdo_'.(in_array($driver, ['mysql', 'mariadb'], true) ? 'mysql' : $driver);
        $checks[] = ["ext-{$pdo} (DB_CONNECTION={$driver})", extension_loaded($pdo), true, ''];

        $gd = extension_loaded('gd');
        $imagick = extension_loaded('imagick');
        $checks[] = ['ext-gd or ext-imagick', $gd || $imagick, true, $gd ? 'gd' : ($imagick ? 'imagick' : 'image optimisation needs one of them')];
        $checks[] = ['AVIF encoder', $gd && function_exists('imageavif'), false, 'optional — WebP + JPEG/PNG are served without it'];
        $checks[] = ['ext-zip', extension_loaded('zip'), false, 'admin XLSX exports'];
        $checks[] = ['OPcache', extension_loaded('Zend OPcache'), false, 'strongly recommended for speed'];

        $disabled = array_map('trim', explode(',', (string) ini_get('disable_functions')));
        $checks[] = ['proc_open enabled', function_exists('proc_open') && ! in_array('proc_open', $disabled, true), true, 'the scheduler starts queue/maintenance commands with it'];

        $upload = self::bytes((string) ini_get('upload_max_filesize'));
        $post = self::bytes((string) ini_get('post_max_size'));
        $media = (int) config('media.max_bytes', 15 * 1024 * 1024);
        $checks[] = ['upload_max_filesize / post_max_size', min($upload, $post) >= $media, false, ini_get('upload_max_filesize').' / '.ini_get('post_max_size').' (media library accepts '.intdiv($media, 1024 * 1024).' MB; web PHP may differ from CLI)'];
        $memory = (string) ini_get('memory_limit');
        $checks[] = ['memory_limit >= 256M', $memory === '-1' || self::bytes($memory) >= 256 * 1024 * 1024, false, $memory.' (image optimisation)'];
        $basedir = (string) ini_get('open_basedir');
        $checks[] = ['open_basedir', $basedir === '' || str_contains($basedir, dirname(base_path())), false, $basedir === '' ? 'not set' : $basedir];

        foreach ($this->writablePaths() as $path) {
            $checks[] = ['writable '.str_replace(dirname(base_path()).'/', '', $path), is_dir($path) && is_writable($path), true, ''];
        }

        $production = app()->environment('production');
        $checks[] = ['APP_DEBUG=false in production', ! ($production && config('app.debug')), true, 'debug pages leak secrets'];
        if ($production) {
            $https = str_starts_with((string) config('app.url'), 'https://');
            $checks[] = ['APP_URL is https', $https, false, (string) config('app.url')];
            $checks[] = ['APP_CANONICAL_REDIRECT=true', (bool) config('app.canonical_redirect'), false, 'one-hop 301 to https + canonical host'];
            $roles = (array) config('filament.admin.mfa_required_roles', []);
            $checks[] = ['ADMIN_MFA_ROLES covers PII roles', array_diff(['super-admin', 'shop-manager', 'directory-manager', 'support'], $roles) === [], false, implode(',', $roles)];
            $checks[] = ['MAIL_MAILER is not log/array', ! in_array(config('mail.default'), ['log', 'array'], true), false, (string) config('mail.default')];
        }

        return $checks;
    }

    /** @return list<string> */
    private function writablePaths(): array
    {
        return [storage_path('app'), storage_path('framework/cache'), storage_path('framework/sessions'), storage_path('framework/views'), storage_path('logs'), base_path('bootstrap/cache'), public_path('media')];
    }

    /** Folders a zip extract may lose (empty) or that never existed; created quietly, permissions 0755. */
    private function ensureDirectories(): void
    {
        foreach ([
            storage_path('app/private'), storage_path('app/public'), storage_path('framework/cache/data'),
            storage_path('framework/sessions'), storage_path('framework/views'), storage_path('logs'),
            base_path('bootstrap/cache'), public_path('media'),
        ] as $dir) {
            if (! is_dir($dir)) {
                @mkdir($dir, 0755, true);
            }
        }
    }

    /** .env from .env.cpanel.example when missing, then APP_KEY. Returns false when the operator must edit .env first. */
    private function ensureEnvironment(): bool
    {
        $env = app()->environmentFilePath();
        if (! is_file($env)) {
            $example = base_path('.env.cpanel.example');
            if (! is_file($example) || ! copy($example, $env)) {
                $this->error("No .env and no .env.cpanel.example to copy from ({$env}).");

                return false;
            }
            @chmod($env, 0600);
            $this->callSilently('key:generate', ['--force' => true]);
            $this->warn("Created {$env} from .env.cpanel.example with a new APP_KEY.");
            $this->warn('Fill in APP_URL, DB_DATABASE / DB_USERNAME / DB_PASSWORD and MAIL_*, then run php artisan app:install again.');

            return false;
        }

        if ((string) config('app.key') === '') {
            $this->components->task('APP_KEY generated', fn (): bool => $this->callSilently('key:generate', ['--force' => true]) === self::SUCCESS);
        }

        return true;
    }

    private function databaseAnswers(): bool
    {
        try {
            DB::connection()->getPdo();

            return true;
        } catch (Throwable $e) {
            $this->error('Database connection failed: '.$e->getMessage());
            $this->line('Check DB_HOST (usually localhost), DB_DATABASE and DB_USERNAME (both carry the cPanel account prefix, e.g. user_ritme), DB_PASSWORD, and that the user was added to the database with ALL PRIVILEGES.');

            return false;
        }
    }

    private function alreadyInstalled(): bool
    {
        return Schema::hasTable('migrations') && Schema::hasTable('users') && User::query()->exists();
    }

    private function offerAdmin(): void
    {
        if ($this->option('no-admin') || User::query()->exists()) {
            return;
        }
        if (! $this->input->isInteractive()) {
            $this->line('No admin user yet — create one with: php artisan admin:create');

            return;
        }
        if ($this->confirm('Create the first admin user now?', true)) {
            $this->call('admin:create');
        }
    }

    /** Writes bootstrap/public-path.php (read by bootstrap/app.php) after checking the folder holds the front controller. */
    private function savePublicPath(string $path): bool
    {
        $absolute = str_starts_with($path, '/') ? $path : base_path($path);
        $real = realpath($absolute);
        if ($real === false || ! is_file($real.'/index.php')) {
            $this->error("{$absolute} is not a folder with index.php — upload the public_html part of the package first.");

            return false;
        }

        $code = "<?php\n\n// Written by `php artisan app:install --public-path` (L10-01): the document root that holds index.php.\nreturn ".var_export($real, true).";\n";
        if (file_put_contents(base_path('bootstrap/public-path.php'), $code) === false) {
            $this->error('Could not write bootstrap/public-path.php.');

            return false;
        }

        $old = public_path('media');
        $this->laravel->usePublicPath($real);
        if (config('filesystems.disks.public.root') === $old) {
            config(['filesystems.disks.public.root' => public_path('media')]);
        }
        $this->info("Public folder set to {$real}.");

        return true;
    }

    /** php.ini shorthand (2M, 512K, 1G) → bytes. */
    private static function bytes(string $value): int
    {
        $value = trim($value);
        if ($value === '' || $value === '-1') {
            return PHP_INT_MAX;
        }
        $number = (int) $value;

        return match (strtolower(substr($value, -1))) {
            'g' => $number * 1024 ** 3,
            'm' => $number * 1024 ** 2,
            'k' => $number * 1024,
            default => $number,
        };
    }
}
