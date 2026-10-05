<?php

declare(strict_types=1);

return [

    /*
    |--------------------------------------------------------------------------
    | Broadcasting
    |--------------------------------------------------------------------------
    |
    | By uncommenting the Laravel Echo configuration, you may connect Filament
    | to any Pusher-compatible websockets server.
    |
    | This will allow your users to receive real-time notifications.
    |
    */

    'broadcasting' => [

        // 'echo' => [
        //     'broadcaster' => 'pusher',
        //     'key' => env('VITE_PUSHER_APP_KEY'),
        //     'cluster' => env('VITE_PUSHER_APP_CLUSTER'),
        //     'wsHost' => env('VITE_PUSHER_HOST'),
        //     'wsPort' => env('VITE_PUSHER_PORT'),
        //     'wssPort' => env('VITE_PUSHER_PORT'),
        //     'authEndpoint' => '/broadcasting/auth',
        //     'disableStats' => true,
        //     'encrypted' => true,
        //     'forceTLS' => env('VITE_PUSHER_SCHEME', 'https') === 'https',
        // ],

    ],

    /*
    |--------------------------------------------------------------------------
    | Default Filesystem Disk
    |--------------------------------------------------------------------------
    |
    | This is the storage disk Filament will use to store files. You may use
    | any of the disks defined in the `config/filesystems.php`.
    |
    */

    'default_filesystem_disk' => env('FILESYSTEM_DISK', 'local'),

    /*
    |--------------------------------------------------------------------------
    | Temporary File URL Expiry
    |--------------------------------------------------------------------------
    |
    | When Filament generates temporary URLs for previewing private files
    | (file uploads, image columns, image entries, rich editor attachments,
    | etc.), this value controls how many minutes those URLs remain valid.
    |
    | The generated URL's expiry is rounded up to the end of the hour it
    | falls in, so the effective lifetime will be between this value and
    | this value plus up to 60 minutes.
    |
    */

    'temporary_file_url_expiry_minutes' => 30,

    /*
    |--------------------------------------------------------------------------
    | Assets Path
    |--------------------------------------------------------------------------
    |
    | This is the directory where Filament's assets will be published to. It
    | is relative to the `public` directory of your Laravel application.
    |
    | After changing the path, you should run `php artisan filament:assets`.
    |
    */

    'assets_path' => null,

    /*
    |--------------------------------------------------------------------------
    | Cache Path
    |--------------------------------------------------------------------------
    |
    | This is the directory that Filament will use to store cache files that
    | are used to optimize the registration of components.
    |
    | After changing the path, you should run `php artisan filament:cache-components`.
    |
    */

    'cache_path' => base_path('bootstrap/cache/filament'),

    /*
    |--------------------------------------------------------------------------
    | Livewire Loading Delay
    |--------------------------------------------------------------------------
    |
    | This sets the delay before loading indicators appear.
    |
    | Setting this to 'none' makes indicators appear immediately, which can be
    | desirable for high-latency connections. Setting it to 'default' applies
    | Livewire's standard 200ms delay.
    |
    */

    'livewire_loading_delay' => 'default',

    /*
    |--------------------------------------------------------------------------
    | File Generation
    |--------------------------------------------------------------------------
    |
    | Artisan commands that generate files can be configured here by setting
    | configuration flags that will impact their location or content.
    |
    | Often, this is useful to preserve file generation behavior from a
    | previous version of Filament, to ensure consistency between older and
    | newer generated files. These flags are often documented in the upgrade
    | guide for the version of Filament you are upgrading to.
    |
    */

    'file_generation' => [
        'flags' => [],
    ],

    /*
    |--------------------------------------------------------------------------
    | System Route Prefix
    |--------------------------------------------------------------------------
    |
    | This is the prefix used for the system routes that Filament registers,
    | such as the routes for downloading exports and failed import rows.
    |
    */

    'system_route_prefix' => 'filament',

    /*
    |--------------------------------------------------------------------------
    | Ritme admin panel (L1-08)
    |--------------------------------------------------------------------------
    |
    | path:              URL prefix of the panel (ADMIN_PATH, default "admin").
    | session_timeout:   minutes of inactivity after which an admin is logged out (independent of SESSION_LIFETIME).
    |                    Login throttling is Filament's built-in limit (5 attempts / minute / IP).
    | brand_color:       mirrors `--color-primary` in resources/css/app.css @theme (asserted by a test).
    | mfa_required_roles: roles that must set up app (TOTP) multi-factor authentication before using the panel.
    |                    ADMIN_MFA_ROLES (comma-separated). Go-live (docs/SECURITY.md): every role that can read
    |                    personal data — super-admin,shop-manager,directory-manager,support (or all six roles).
    |
    */

    'admin' => [
        'path' => env('ADMIN_PATH', 'admin'),
        'session_timeout' => (int) env('ADMIN_SESSION_TIMEOUT', 60),
        'brand_color' => '#6e54f0',
        'mfa_required_roles' => array_values(array_filter(array_map('trim', explode(',', (string) env('ADMIN_MFA_ROLES', 'super-admin'))))),
    ],

];
