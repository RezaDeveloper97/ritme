<?php

declare(strict_types=1);

use App\Console\Commands\AppDoctor;
use App\Domain\Seo\Sitemap\Sitemaps;
use Illuminate\Foundation\Inspiring;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Schedule;

Artisan::command('inspire', function () {
    $this->comment(Inspiring::quote());
})->purpose('Display an inspiring quote');

/*
|--------------------------------------------------------------------------
| Scheduler (L10-01) — driven by ONE cPanel cron line, every minute:
|   * * * * * /usr/local/bin/php /home/<user>/ritme/artisan schedule:run >> /dev/null 2>&1
|--------------------------------------------------------------------------
| Shared hosting has no Supervisor, so the database queue (notifications, media optimisation, IndexNow pings, SEO
| audits) is drained by a short-lived worker started every minute: it exits when the queue is empty or after 55 s, so
| two workers never overlap for long (withoutOverlapping keeps it at one). runInBackground lets the other tasks of
| the same minute start at once. Domain schedules live in their providers: blog:publish-scheduled (every minute),
| blog:flush-views + seo:flush-redirect-stats (every 5 min), seo:purge-404 (daily 03:40), weekly SEO audit (Sat 04:20).
| L10-02: each worker run that exits normally writes the queue heartbeat `php artisan app:doctor` checks; backups
| (spatie/laravel-backup, config/backup.php) run nightly. Full list: php artisan schedule:list. Guide: docs/GO-LIVE.md.
*/

Schedule::command('queue:work', ['--stop-when-empty', '--max-time=55', '--tries=3', '--timeout=50', '--sleep=3'])
    ->name('queue-worker')
    ->everyMinute()
    ->withoutOverlapping(2)
    ->runInBackground()
    // Runs via `schedule:finish` when the background worker exits 0: proof that cron + scheduler + queue all work.
    ->onSuccess(static function (): void {
        AppDoctor::recordHeartbeat();
    });

// Rebuild the sitemap documents after their namespace was bumped, so crawlers never wait for a cold build.
Schedule::call(static function (Sitemaps $sitemaps): void {
    $sitemaps->index();
})->name('sitemap-warm')->hourlyAt(7)->withoutOverlapping(10);

// Housekeeping: failed jobs older than a week, activity log entries older than config('activitylog.delete_records_older_than_days').
Schedule::command('queue:prune-failed', ['--hours=168'])->dailyAt('03:20');
Schedule::command('activitylog:clean', ['--force'])->dailyAt('03:30');

// Backups (L10-02): clean-up first (frees quota), then DB + media to the private disk, then the health check (mails
// OPS_ALERT_EMAIL only when the newest backup is too old or the backups use too much space). Quiet hours, Tehran time.
Schedule::command('backup:clean')->name('backup-clean')->dailyAt('01:50')->withoutOverlapping(60);
Schedule::command('backup:run')->name('backup-run')->dailyAt('02:10')->withoutOverlapping(120)->runInBackground();
Schedule::command('backup:monitor')->name('backup-monitor')->dailyAt('08:15');
