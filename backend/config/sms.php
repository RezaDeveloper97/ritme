<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Default SMS Provider
    |--------------------------------------------------------------------------
    |
    | This option controls the default SMS provider that gets used when
    | sending SMS messages. The name specified in this option should match
    | one of the providers defined in the "providers" configuration array.
    |
    */

    'default' => env('SMS_PROVIDER', 'kavenegar'),

    /*
    |--------------------------------------------------------------------------
    | Fallback Providers
    |--------------------------------------------------------------------------
    |
    | If the default provider fails, these providers will be tried in order.
    |
    */

    'fallback' => ['smsir'],

    /*
    |--------------------------------------------------------------------------
    | SMS Providers
    |--------------------------------------------------------------------------
    |
    | Here you may configure all of the SMS providers used by your application.
    | Several examples have been configured for you and you are free to add
    | your own as your application requires.
    |
    */

    'providers' => [

        'smsir' => [
            'driver' => 'smsir',
            'api_key' => env('SMSIR_API_KEY'),
            'template_id' => env('SMSIR_TEMPLATE_ID', 511293),
            'line_number' => env('SMSIR_LINE_NUMBER'),
        ],

        'kavenegar' => [
            'driver' => 'kavenegar',
            'api_key' => env('KAVENEGAR_API_KEY'),
            'sender' => env('KAVENEGAR_SENDER'),
            'templates' => [
                // Kavenegar lookup template: «کد ورود %token ریتمی»
                'login_otp' => env('KAVENEGAR_TEMPLATE_LOGIN_OTP', '1507703'),
            ],
        ],

    ],

    /*
    |--------------------------------------------------------------------------
    | OTP Settings
    |--------------------------------------------------------------------------
    */

    'otp' => [
        'length' => 4,
        'expires_in' => 2, // minutes
        'max_attempts' => 5,
        'resend_after' => 60, // seconds

        // There is deliberately no test/bypass mode here. Any switch that
        // fixes the code at a known value is a way in: from the request body
        // it lets anyone pin any account's OTP, and from the environment it is
        // one mis-set variable away from doing the same on a live server.
        // Local dev reads the generated code out of the database instead —
        // see .claude/skills/local-dev.
    ],

];
