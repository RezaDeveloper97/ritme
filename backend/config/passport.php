<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Passport Guard
    |--------------------------------------------------------------------------
    |
    | Here you may specify which authentication guard Passport will use when
    | authenticating users. This value should correspond with one of your
    | guards that is already present in your "auth" configuration file.
    |
    */

    'guard' => 'web',

    /*
    |--------------------------------------------------------------------------
    | Password Grant Client
    |--------------------------------------------------------------------------
    |
    | These are the client ID and secret for the Password Grant Client.
    | Create it with: php artisan passport:client --password
    |
    */

    'password_client' => [
        'id' => env('PASSPORT_PASSWORD_CLIENT_ID'),
        'secret' => env('PASSPORT_PASSWORD_CLIENT_SECRET'),
    ],

    /*
    |--------------------------------------------------------------------------
    | Encryption Keys
    |--------------------------------------------------------------------------
    |
    | Passport uses encryption keys while generating secure access tokens for
    | your application. By default, the keys are stored as local files but
    | can be set via environment variables when that is more convenient.
    |
    */

    'private_key' => env('PASSPORT_PRIVATE_KEY'),

    'public_key' => env('PASSPORT_PUBLIC_KEY'),

    /*
    |--------------------------------------------------------------------------
    | Passport Database Connection
    |--------------------------------------------------------------------------
    |
    | By default, Passport's models will utilize your application's default
    | database connection. If you wish to use a different connection you
    | may specify the configured name of the database connection here.
    |
    */

    'connection' => env('PASSPORT_CONNECTION'),

    /*
    |--------------------------------------------------------------------------
    | Session Token Lifetime
    |--------------------------------------------------------------------------
    |
    | Every token issued at OTP verification lives this many days. A fixed
    | day count (not "one year") keeps the lifetime the same across leap
    | years. A client whose token has fewer than `refresh_window_days` left
    | may trade it for a fresh one at POST /api/v1/auth/refresh-session
    | without entering another OTP.
    |
    */

    'token_lifetime_days' => (int) env('PASSPORT_TOKEN_LIFETIME_DAYS', 365),

    'refresh_window_days' => (int) env('PASSPORT_REFRESH_WINDOW_DAYS', 30),

];
