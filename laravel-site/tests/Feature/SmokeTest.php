<?php

declare(strict_types=1);

it('serves the home page', function (): void {
    $this->get('/')->assertOk();
});

it('runs in the Persian locale and Tehran timezone', function (): void {
    expect(app()->getLocale())->toBe('fa')
        ->and(config('app.timezone'))->toBe('Asia/Tehran');
});
