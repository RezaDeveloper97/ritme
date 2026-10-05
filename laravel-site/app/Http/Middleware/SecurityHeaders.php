<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use App\View\Components\Layout\Assets;
use Closure;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Foundation\Application;
use Illuminate\Foundation\Vite;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Security headers on every response (global, outermost, so redirects and error pages get them too).
 *
 * Public site — strict CSP with no external origin: everything is `'self'` (+ `data:` images), no inline styles, no
 * inline scripts (JSON-LD data blocks are not executed and need nothing) except the build-generated critical CSS
 * <style> blocks, allowed by their sha256 from public/build/critical/manifest.json (L9-01); an opt-in per-request nonce
 * (pagecache.security.nonce) allows an inline bootstrap. Admin (filament.admin.path, Livewire, Filament endpoints) — Livewire/Alpine need inline scripts, eval and
 * inline styles, so `'unsafe-inline' 'unsafe-eval'` there, still without any external origin and without a nonce
 * (a nonce would disable 'unsafe-inline').
 *
 * While `npm run dev` runs (public/hot, non-production only) the Vite dev-server origin is allowed. Laravel's debug
 * exception page (5xx with app.debug) gets no CSP. HSTS only over https, by default only in production.
 * A CSP set by the controller itself is kept.
 */
final class SecurityHeaders
{
    public function __construct(
        private readonly Config $config,
        private readonly Application $app,
        private readonly Vite $vite,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        if (! $this->config->get('pagecache.security.enabled', true)) {
            return $next($request);
        }

        $admin = AdminPaths::matches($request, $this->config);
        $nonce = null;

        if (! $admin && $this->config->get('pagecache.security.nonce', false)) {
            $nonce = $this->vite->useCspNonce(); // fresh for every request
        }

        $response = $next($request);
        $headers = $response->headers;

        $debugError = $response->getStatusCode() >= 500 && $this->config->get('app.debug');
        if (! $debugError && ! $headers->has('Content-Security-Policy')) {
            $headers->set('Content-Security-Policy', $this->policy($admin, $nonce, $request->isSecure()));
        }

        $defaults = [
            'X-Content-Type-Options' => 'nosniff',
            'X-Frame-Options' => 'SAMEORIGIN',
            'Referrer-Policy' => 'strict-origin-when-cross-origin',
            'Permissions-Policy' => (string) $this->config->get('pagecache.security.permissions_policy', ''),
            'Cross-Origin-Opener-Policy' => 'same-origin',
        ];
        foreach ($defaults as $name => $value) {
            if ($value !== '' && ! $headers->has($name)) {
                $headers->set($name, $value);
            }
        }

        if ($request->isSecure() && $this->hstsEnabled()) {
            $headers->set('Strict-Transport-Security', 'max-age='.(int) $this->config->get('pagecache.security.hsts_max_age', 31536000)
                .($this->config->get('pagecache.security.hsts_include_subdomains') ? '; includeSubDomains' : ''));
        }

        return $response;
    }

    public function policy(bool $admin, ?string $nonce, bool $secure): string
    {
        $hot = $this->hotOrigins();

        $script = ["'self'"];
        $style = ["'self'"];
        $img = ["'self'", 'data:'];
        $font = ["'self'"];
        $worker = ["'self'"];

        if ($admin) {
            $script = [...$script, "'unsafe-inline'", "'unsafe-eval'"];
            $style[] = "'unsafe-inline'";
            $img[] = 'blob:';
            $font[] = 'data:';
            $worker[] = 'blob:';
        } else {
            if ($nonce !== null) {
                $script[] = "'nonce-{$nonce}'";
            }
            // Inline critical CSS (L9-01): only the exact build-generated <style> blocks, by hash.
            $style = [...$style, ...Assets::criticalStyleHashes($this->vite)];
        }

        if ($hot !== []) {
            $script = [...$script, ...$hot['http']];
            $style = [...$style, ...$hot['http'], "'unsafe-inline'"]; // Vite injects <style> tags in dev
            $font = [...$font, ...$hot['http']];
            $img = [...$img, ...$hot['http']];
        }

        $directives = [
            'default-src' => ["'self'"],
            'base-uri' => ["'self'"],
            'object-src' => ["'none'"],
            'frame-ancestors' => ["'self'"],
            'form-action' => ["'self'"],
            'script-src' => $script,
            'style-src' => $style,
            'img-src' => $img,
            'font-src' => $font,
            'connect-src' => ["'self'", ...($hot['http'] ?? []), ...($hot['ws'] ?? [])],
            'media-src' => $admin ? ["'self'", 'blob:'] : ["'self'"],
            'manifest-src' => ["'self'"],
            'worker-src' => $worker,
            'frame-src' => ["'self'"],
        ];

        $parts = [];
        foreach ($directives as $name => $sources) {
            $parts[] = $name.' '.implode(' ', array_unique($sources));
        }
        if ($secure && $hot === []) {
            $parts[] = 'upgrade-insecure-requests';
        }

        return implode('; ', $parts);
    }

    private function hstsEnabled(): bool
    {
        $hsts = $this->config->get('pagecache.security.hsts');

        return $hsts === null ? $this->app->isProduction() : (bool) $hsts;
    }

    /**
     * Vite dev-server origins while `npm run dev` runs (never in production).
     *
     * @return array{}|array{http: list<string>, ws: list<string>}
     */
    private function hotOrigins(): array
    {
        if ($this->app->isProduction() || ! $this->vite->isRunningHot()) {
            return [];
        }

        $hot = trim((string) @file_get_contents($this->vite->hotFile()));
        $parts = parse_url($hot);
        if (! is_array($parts) || ! isset($parts['host'])) {
            return [];
        }

        $origin = '//'.$parts['host'].(isset($parts['port']) ? ':'.$parts['port'] : '');

        return [
            'http' => [($parts['scheme'] ?? 'http').':'.$origin],
            'ws' => ['ws:'.$origin, 'wss:'.$origin],
        ];
    }
}
