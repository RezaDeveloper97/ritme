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
    | SMS Providers
    |--------------------------------------------------------------------------
    |
    | Here you may configure all of the SMS providers used by your application.
    | Several examples have been configured for you and you are free to add
    | your own as your application requires.
    |
    */

    'providers' => [

        'kavenegar' => [
            'driver' => 'kavenegar',
            'api_key' => env('KAVENEGAR_API_KEY'),
            'sender' => env('KAVENEGAR_SENDER'),
            'templates' => [
                'login_otp' => env('KAVENEGAR_TEMPLATE_LOGIN_OTP', 'login-otp'),
            ],
        ],

        // Future providers can be added here
        // 'melipayamak' => [
        //     'driver' => 'melipayamak',
        //     'username' => env('MELIPAYAMAK_USERNAME'),
        //     'password' => env('MELIPAYAMAK_PASSWORD'),
        //     'sender' => env('MELIPAYAMAK_SENDER'),
        // ],

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
    ],

];
