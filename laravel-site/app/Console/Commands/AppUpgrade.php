<?php

declare(strict_types=1);

namespace App\Console\Commands;

use Illuminate\Console\Command;
use Illuminate\Support\Str;
use RuntimeException;
use Throwable;

/**
 * Release upgrade after the new package was extracted over the old one (L10-01, docs/DEPLOY-CPANEL.md): checks →
 * maintenance mode with a bypass secret → migrations → insert-missing production seeders → fresh caches (optimize,
 * Filament) → queue restart → every cache namespace bumped (pages carry the critical-CSS hash of the old build) →
 * sitemap warmed → up. On failure the site stays in maintenance mode and the bypass URL is printed.
 */
final class AppUpgrade extends Command
{
    protected $signature = 'app:upgrade
        {--secret= : Maintenance bypass secret (random when empty)}
        {--no-maintenance : Do not put the site into maintenance mode}
        {--no-cache : Do not cache config, routes, views and Filament components}
        {--skip-checks : Continue even when a required check fails}';

    protected $description = 'Upgrade to the extracted release: maintenance mode, migrate, seed defaults, rebuild caches, bump cache namespaces, up';

    public function handle(): int
    {
        if ($this->callSilently('app:install', ['--check' => true]) !== self::SUCCESS && ! $this->option('skip-checks')) {
            $this->error('Requirement checks failed — run php artisan app:install --check for details, or pass --skip-checks.');

            return self::FAILURE;
        }

        $maintenance = ! $this->option('no-maintenance');
        $secret = is_string($s = $this->option('secret')) && $s !== '' ? $s : Str::random(32);

        if ($maintenance) {
            $this->callSilently('down', ['--secret' => $secret, '--retry' => 60, '--refresh' => 30]);
            $this->warn('Maintenance mode on. Bypass (sets a cookie): '.url('/'.$secret));
        }

        try {
            $this->step('Migrations', 'migrate', ['--force' => true]);
            foreach (AppInstall::PRODUCTION_SEEDERS as $seeder) {
                $this->step('Seeder '.class_basename($seeder), 'db:seed', ['--class' => $seeder, '--force' => true]);
            }

            // Not optimize:clear: its cache:clear would also drop a cache-driver maintenance flag and the rate limits.
            foreach (['config:clear', 'route:clear', 'event:clear', 'view:clear'] as $clear) {
                $this->step('Old cache: '.$clear, $clear);
            }
            $this->step('Old Filament caches cleared', 'filament:optimize-clear');
            if (! $this->option('no-cache')) {
                AppInstall::warmCaches($this);
            } else {
                $this->step('Cache namespaces bumped', 'cache:ns', ['action' => 'bump-all']);
            }
            $this->step('Queue workers told to restart', 'queue:restart');
        } catch (Throwable $e) {
            report($e);
            $this->error('Upgrade failed: '.$e->getMessage());
            if ($maintenance) {
                $this->warn('The site stays in maintenance mode. Inspect it via '.url('/'.$secret).', fix the cause, run php artisan app:upgrade again (or php artisan up).');
            }

            return self::FAILURE;
        }

        if ($maintenance) {
            $this->callSilently('up');
        }
        $this->info('Upgrade finished'.($maintenance ? ' — the site is live again.' : '.'));

        return self::SUCCESS;
    }

    /**
     * @param  array<string, mixed>  $arguments
     */
    private function step(string $label, string $command, array $arguments = []): void
    {
        $this->components->task($label, function () use ($command, $arguments): bool {
            if ($this->callSilently($command, $arguments) !== self::SUCCESS) {
                throw new RuntimeException("php artisan {$command} failed.");
            }

            return true;
        });
    }
}
