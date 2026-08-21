<?php

/*
 * The frontend (web.ritme.app) and the API (api.ritme.app) are separate
 * origins, so every browser call to /api/* is a cross-origin request and needs
 * CORS headers. Laravel's built-in default is `allowed_origins => ['*']`;
 * this file narrows that to the origins we actually ship, which is the whole
 * point of publishing it.
 *
 * Native clients (the Android app) are unaffected — CORS is a browser rule.
 *
 * Credentials stay off on purpose: auth is a Bearer token in the Authorization
 * header, never a cookie, so there is nothing for the browser to attach.
 */

$origins = array_values(array_filter(array_map(
    'trim',
    explode(',', (string) env('CORS_ALLOWED_ORIGINS', implode(',', [
        'https://web.ritme.app',
        // Local development against a remote or local API.
        'http://localhost:3000',
        'http://127.0.0.1:3000',
    ])))
)));

return [
    'paths' => ['api/*'],

    'allowed_methods' => ['*'],

    'allowed_origins' => $origins,

    'allowed_origins_patterns' => [],

    'allowed_headers' => ['*'],

    'exposed_headers' => [],

    'max_age' => 3600,

    'supports_credentials' => false,
];
