<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Gates the Swagger UI and the raw OpenAPI JSON/YAML behind HTTP Basic auth.
 *
 * The API docs enumerate every endpoint, parameter and example payload; they
 * must not be world-readable in production. Credentials come from the
 * environment (SWAGGER_USER / SWAGGER_PASSWORD) so nothing is committed.
 *
 * Behaviour:
 *  - If no password is configured AND the app is in local/testing, access is
 *    open (developer convenience — the docs aren't exposed there anyway).
 *  - If no password is configured in any other environment, access is DENIED
 *    (fail closed) rather than silently public.
 *  - Otherwise a valid user/password pair is required.
 */
class SwaggerBasicAuth
{
    public function handle(Request $request, Closure $next): Response
    {
        $expectedUser = (string) env('SWAGGER_USER', 'ritme');
        $expectedPass = (string) env('SWAGGER_PASSWORD', '');

        if ($expectedPass === '') {
            // No credentials set: open only in local/testing, closed elsewhere.
            if (app()->environment('local', 'testing')) {
                return $next($request);
            }

            return $this->deny();
        }

        $user = (string) $request->getUser();
        $pass = (string) $request->getPassword();

        $ok = hash_equals($expectedUser, $user) & hash_equals($expectedPass, $pass);

        if (! $ok) {
            return $this->deny();
        }

        return $next($request);
    }

    private function deny(): Response
    {
        return response('Authentication required.', 401, [
            'WWW-Authenticate' => 'Basic realm="Ritme API Docs"',
        ]);
    }
}
