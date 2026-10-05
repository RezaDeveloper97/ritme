<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use Illuminate\Routing\Middleware\ThrottleRequests;
use RuntimeException;

/**
 * `throttle:N,M` with one counter per ROUTE and client (L9-04). Laravel keys an unnamed limit by domain + IP only, so
 * every `throttle:N,M` route shared one counter: four blog view beacons (30/min) used up the place-review limit
 * (3 per 10 min), a busy cart blocked product reviews, and so on. Named limiters (`throttle:contact` …) keep their own
 * keys and are unaffected. Registered as the `throttle` alias in AppServiceProvider.
 */
final class ThrottlePerRoute extends ThrottleRequests
{
    protected function resolveRequestSignature($request)
    {
        $route = $request->route();
        if ($route === null) {
            throw new RuntimeException('Unable to generate the request signature. Route unavailable.');
        }

        $who = ($user = $request->user()) !== null ? 'user:'.$user->getAuthIdentifier() : 'ip:'.$request->ip();

        return sha1(($route->getName() ?? implode('|', $route->methods()).' '.$route->uri()).'|'.$route->getDomain().'|'.$who);
    }
}
