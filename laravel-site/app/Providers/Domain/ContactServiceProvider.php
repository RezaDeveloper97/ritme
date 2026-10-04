<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Providers\DomainServiceProvider;
use Illuminate\Cache\RateLimiting\Limit;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\RateLimiter;

final class ContactServiceProvider extends DomainServiceProvider
{
    /** Contact form posts per IP (route middleware `throttle:contact`); no captcha or external service. */
    public const PER_MINUTE = 3;

    public const PER_DAY = 10;

    public function boot(): void
    {
        parent::boot();

        RateLimiter::for('contact', static function (Request $request): array {
            $ip = (string) $request->ip();

            return [
                Limit::perMinute(self::PER_MINUTE)->by('contact:m:'.$ip),
                Limit::perDay(self::PER_DAY)->by('contact:d:'.$ip),
            ];
        });
    }
}
