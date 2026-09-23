<?php

namespace App\Http\Middleware;

use Carbon\Carbon;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;
use Throwable;

/**
 * Freezes "now" for one request from the `X-Test-Now` header, so the contract
 * recorder (docs/go-migration/contract.md) gets the same response every run.
 *
 * Fails closed: it only acts when APP_ENV is one of ENVIRONMENTS *and*
 * TEST_CLOCK_ENABLED=true. Production can never match the first condition, so
 * a stray flag or header there is ignored. When it is active, an unparseable
 * header is a 400 rather than silently falling back to the real clock — a typo
 * in the recorder must not produce a golden that depends on the wall clock.
 */
class TestClock
{
    public const HEADER = 'X-Test-Now';

    /** @var list<string> */
    public const ENVIRONMENTS = ['local', 'testing', 'contract'];

    public static function isEnabled(): bool
    {
        return app()->environment(self::ENVIRONMENTS)
            && filter_var(env('TEST_CLOCK_ENABLED', false), FILTER_VALIDATE_BOOLEAN);
    }

    public function handle(Request $request, Closure $next): Response
    {
        $value = $request->header(self::HEADER);

        if ($value === null || $value === '' || ! self::isEnabled()) {
            return $next($request);
        }

        try {
            // ISO-8601; a value without an offset is read as Tehran wall-clock.
            $now = Carbon::parse($value, 'Asia/Tehran')->setTimezone(config('app.timezone'));
        } catch (Throwable) {
            return response()->json(['message' => 'Invalid '.self::HEADER.' header.'], 400);
        }

        $previous = Carbon::getTestNow();
        Carbon::setTestNow($now);

        try {
            return $next($request);
        } finally {
            Carbon::setTestNow($previous);
        }
    }
}
