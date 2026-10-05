<?php

declare(strict_types=1);

use App\Console\Commands\AppInstall;
use App\Filament\Auth\AdminRole;
use App\Models\User;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Console\Scheduling\CallbackEvent;
use Illuminate\Console\Scheduling\Event;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Support\Facades\DB;
use Spatie\Permission\Models\Role;

/**
 * L10-01: web-less installer / upgrader for cPanel and the scheduler that replaces Supervisor. Caching commands
 * (optimize writes bootstrap/cache) are skipped with --no-cache; maintenance mode uses the array cache so parallel
 * test processes never see a "down" file; view:clear (inside optimize:clear) is pointed at a private folder.
 */
beforeEach(function (): void {
    config([
        'app.maintenance.driver' => 'cache',
        'app.maintenance.store' => 'deploy-test',
        'cache.stores.deploy-test' => ['driver' => 'array'],
        'view.compiled' => sys_get_temp_dir().'/ritme-deploy-test-views-'.getmypid(),
    ]);
    @mkdir((string) config('view.compiled'), 0755, true);
});

it('runs the requirement checks only with --check', function (): void {
    $this->artisan('app:install', ['--check' => true])
        ->expectsOutputToContain('PHP >= '.AppInstall::MIN_PHP)
        ->expectsOutputToContain('ext-intl')
        ->expectsOutputToContain('ext-gd or ext-imagick')
        ->expectsOutputToContain('proc_open enabled')
        ->expectsOutputToContain('APP_DEBUG=false in production')
        ->doesntExpectOutputToContain('Migrations')
        ->assertSuccessful();

    expect(DB::table('settings')->count())->toBe(0);
});

it('installs: migrations, production defaults, no admin without a terminal, cron line', function (): void {
    $this->artisan('app:install', ['--no-cache' => true, '--no-interaction' => true])
        ->expectsOutputToContain('php artisan admin:create')
        ->expectsOutputToContain('schedule:run >> /dev/null 2>&1')
        ->assertSuccessful();

    expect(DB::table('settings')->count())->toBeGreaterThan(0)
        ->and(Role::query()->pluck('name')->all())->toContain(AdminRole::SuperAdmin->value)
        ->and(User::query()->count())->toBe(0)
        ->and(AppInstall::cronLine())->toBe('* * * * * '.PHP_BINARY.' '.base_path('artisan').' schedule:run >> /dev/null 2>&1');
});

it('offers to create the first admin in a terminal', function (): void {
    $this->artisan('app:install', ['--no-cache' => true])
        ->expectsConfirmation('Create the first admin user now?', 'no')
        ->assertSuccessful();
});

it('refuses to run again on an installed site unless forced', function (): void {
    User::query()->create(['name' => 'Admin', 'email' => 'admin@example.test', 'password' => 'Str0ngPassw0rd99', 'is_active' => true]);

    $this->artisan('app:install', ['--no-cache' => true, '--no-interaction' => true])
        ->expectsOutputToContain('php artisan app:upgrade')
        ->assertFailed();

    $this->artisan('app:install', ['--no-cache' => true, '--no-interaction' => true, '--force' => true])->assertSuccessful();
});

it('refuses a public path without the front controller', function (): void {
    $this->artisan('app:install', ['--check' => true, '--public-path' => '/definitely/not/here'])
        ->expectsOutputToContain('is not a folder with index.php')
        ->assertFailed();

    expect(base_path('bootstrap/public-path.php'))->not->toBeFile();
});

it('upgrades behind maintenance mode with a bypass secret and bumps every cache namespace', function (): void {
    $versions = app(NamespaceVersions::class);
    $before = $versions->version('pages');

    $this->artisan('app:upgrade', ['--no-cache' => true, '--secret' => 'release-bypass-123'])
        ->expectsOutputToContain(url('/release-bypass-123'))
        ->expectsOutputToContain('Upgrade finished')
        ->assertSuccessful();

    expect(app()->maintenanceMode()->active())->toBeFalse()
        ->and(storage_path('framework/maintenance.php'))->not->toBeFile()
        ->and(DB::table('settings')->count())->toBeGreaterThan(0)
        ->and($versions->version('pages'))->toBeGreaterThan($before);
});

it('can upgrade without maintenance mode', function (): void {
    $this->artisan('app:upgrade', ['--no-cache' => true, '--no-maintenance' => true])
        ->doesntExpectOutputToContain('Maintenance mode on')
        ->assertSuccessful();
});

it('drains the database queue from the scheduler every minute without overlapping workers', function (): void {
    /** @var list<Event> $events */
    $events = app(Schedule::class)->events();
    $worker = collect($events)->first(fn (Event $e): bool => str_contains((string) $e->command, 'queue:work'));

    expect($worker)->not->toBeNull()
        ->and((string) $worker->command)->toContain('--stop-when-empty')->toContain('--max-time=55')
        ->and($worker->expression)->toBe('* * * * *')
        ->and($worker->withoutOverlapping)->toBeTrue()
        ->and($worker->runInBackground)->toBeTrue();

    $names = collect($events)->map(fn (Event $e): string => $e instanceof CallbackEvent ? (string) $e->description : (string) $e->command)->implode("\n");
    expect($names)
        ->toContain('sitemap-warm')
        ->toContain('blog:publish-scheduled')
        ->toContain('seo:purge-404')
        ->toContain('seo-audit')
        ->toContain('queue:prune-failed')
        ->toContain('activitylog:clean');
});
