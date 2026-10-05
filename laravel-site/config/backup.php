<?php

declare(strict_types=1);

use Spatie\Backup\Notifications\Notifiable;
use Spatie\Backup\Notifications\Notifications\BackupHasFailedNotification;
use Spatie\Backup\Notifications\Notifications\BackupWasSuccessfulNotification;
use Spatie\Backup\Notifications\Notifications\CleanupHasFailedNotification;
use Spatie\Backup\Notifications\Notifications\CleanupWasSuccessfulNotification;
use Spatie\Backup\Notifications\Notifications\HealthyBackupWasFoundNotification;
use Spatie\Backup\Notifications\Notifications\UnhealthyBackupWasFoundNotification;
use Spatie\Backup\Tasks\Cleanup\Strategies\DefaultStrategy;
use Spatie\Backup\Tasks\Monitor\HealthChecks\MaximumAgeInDays;
use Spatie\Backup\Tasks\Monitor\HealthChecks\MaximumStorageInMegabytes;

/*
|--------------------------------------------------------------------------
| Backups (L10-02) — spatie/laravel-backup, driven by the cron scheduler (routes/console.php)
|--------------------------------------------------------------------------
| What: a dump of the default database + the uploaded media (<public>/media, wherever the cPanel layout put it) +
| unapproved join photos (storage/app/pending). Not included on purpose: code (the release zip is the code backup),
| `.env` and APP_KEY (secrets — keep them in the password manager; without APP_KEY the admins' encrypted TOTP secrets
| cannot be read after a restore).
| Where: the private `local` disk (storage/app/private/<BACKUP_NAME>/<date>.zip) — never under the web root. Admins
| download them from the panel (super-admin only, «سیستم ← پشتیبان‌ها»). An off-site copy is a manual step for now
| (docs/GO-LIVE.md); add a remote disk to BACKUP_DISKS later (e.g. an FTP/SFTP disk in config/filesystems.php).
| When: daily 02:10 (backup:run), clean-up 01:50, health check 08:15 — mails only on failure / unhealthy backups,
| and only when OPS_ALERT_EMAIL is set.
| MySQL needs the `mysqldump` binary (present on cPanel hosts) and proc_open; SQLite needs `sqlite3`.
*/

// Invalid / placeholder addresses count as "not set": the package refuses to run with an invalid mail address.
$alertMail = filter_var((string) env('OPS_ALERT_EMAIL', ''), FILTER_VALIDATE_EMAIL) !== false ? (string) env('OPS_ALERT_EMAIL') : '';
$fromMail = filter_var((string) env('MAIL_FROM_ADDRESS', ''), FILTER_VALIDATE_EMAIL) !== false ? (string) env('MAIL_FROM_ADDRESS') : 'hello@example.com';
$alertChannels = $alertMail !== '' ? ['mail'] : [];
$disks = array_values(array_filter(array_map('trim', explode(',', (string) env('BACKUP_DISKS', 'local')))));
$name = (string) env('BACKUP_NAME', 'ritme-site');
$maxAgeDays = (int) env('BACKUP_MAX_AGE_DAYS', 1);
$maxStorageMb = (int) env('BACKUP_MAX_STORAGE_MB', 4000);

return [

    'backup' => [
        // Folder name on the destination disk and the name shown in notifications.
        'name' => $name,

        'source' => [
            'files' => [
                'include' => [
                    // Same root as the `public` disk in config/filesystems.php (media library, all variants).
                    (string) env('MEDIA_PUBLIC_ROOT', public_path('media')),
                    // Join-request photos waiting for moderation (private `pending` disk).
                    storage_path('app/pending'),
                ],

                'exclude' => [],

                'follow_links' => false,

                // An unreadable folder must not abort the whole backup on shared hosting; it is reported in the log.
                'ignore_unreadable_directories' => true,

                // Paths inside the zip relative to the account folder: `ritme/public/media/…` (layout A) or
                // `public_html/media/…` (layout B) — readable and restorable by hand.
                'relative_path' => dirname(base_path()),
            ],

            'databases' => [
                env('DB_CONNECTION', 'sqlite'),
            ],
        ],

        // The zip compresses the dump already; a gzip compressor would need the `gzip` binary on the host.
        'database_dump_compressor' => null,

        'database_dump_file_timestamp_format' => null,

        'database_dump_filename_base' => 'connection',

        'database_dump_file_extension' => '',

        'destination' => [
            'compression_method' => ZipArchive::CM_DEFAULT,

            // 6 instead of 9: almost the same size for SQL + already-compressed images, much less CPU on shared hosts.
            'compression_level' => 6,

            'filename_prefix' => '',

            'disks' => $disks !== [] ? $disks : ['local'],
        ],

        'temporary_directory' => storage_path('app/backup-temp'),

        // Optional AES-256 zip password (keep it next to APP_KEY in the password manager).
        'password' => env('BACKUP_ARCHIVE_PASSWORD'),

        'encryption' => 'default',

        'tries' => 1,

        'retry_delay' => 0,
    ],

    // Mail only what needs a human: failures and unhealthy backups. Success mails would just be daily noise.
    'notifications' => [
        'notifications' => [
            BackupHasFailedNotification::class => $alertChannels,
            UnhealthyBackupWasFoundNotification::class => $alertChannels,
            CleanupHasFailedNotification::class => $alertChannels,
            BackupWasSuccessfulNotification::class => [],
            HealthyBackupWasFoundNotification::class => [],
            CleanupWasSuccessfulNotification::class => [],
        ],

        'notifiable' => Notifiable::class,

        'mail' => [
            // The package validates this address even when no channel uses it, hence the harmless fallback.
            'to' => $alertMail !== '' ? $alertMail : $fromMail,

            'from' => [
                'address' => $fromMail,
                'name' => env('MAIL_FROM_NAME', 'Ritme'),
            ],
        ],

        'slack' => [
            'webhook_url' => '',
            'channel' => null,
            'username' => null,
            'icon' => null,
        ],

        'discord' => [
            'webhook_url' => '',
            'username' => '',
            'avatar_url' => '',
        ],
    ],

    // backup:monitor + `php artisan app:doctor` + the admin backups page read these limits.
    'monitor_backups' => [
        [
            'name' => $name,
            'disks' => $disks !== [] ? $disks : ['local'],
            'health_checks' => [
                MaximumAgeInDays::class => $maxAgeDays,
                MaximumStorageInMegabytes::class => $maxStorageMb,
            ],
        ],
    ],

    // Shared hosting quota: two weeks of dailies, then weeklies for two months, monthlies for half a year; the newest
    // backup is never deleted. The size cap must stay below the account's free space.
    'cleanup' => [
        'strategy' => DefaultStrategy::class,

        'default_strategy' => [
            'keep_all_backups_for_days' => 3,
            'keep_daily_backups_for_days' => 14,
            'keep_weekly_backups_for_weeks' => 8,
            'keep_monthly_backups_for_months' => 6,
            'keep_yearly_backups_for_years' => 1,
            'delete_oldest_backups_when_using_more_megabytes_than' => $maxStorageMb,
        ],

        'tries' => 1,

        'retry_delay' => 0,
    ],

];
