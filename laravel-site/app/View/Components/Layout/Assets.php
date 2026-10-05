<?php

declare(strict_types=1);

namespace App\View\Components\Layout;

use App\Domain\Pwa\Data\InstallMetadata;
use App\Domain\Pwa\Manifest\WebManifest;
use Illuminate\Contracts\View\View;
use Illuminate\Foundation\Vite;
use Illuminate\Support\Str;
use Illuminate\View\Component;
use Throwable;

/**
 * <x-layout.assets /> — the layout's non-SEO head tags: PWA install metadata (manifest link, theme-color, icons —
 * <x-pwa.head>, one cache read), preloads for the two above-the-fold font files (Lalezar for the logo/h1,
 * Vazirmatn 600 for nav and body copy) and the Vite entries. Everything else (other weights, Latin subsets) loads on
 * demand through unicode-range.
 *
 * Critical CSS (L9-01): `npm run build` ends with tools/critical.mjs, which writes public/build/critical/ —
 * `<template>.css` (the above-the-fold subset of the built stylesheet) and `manifest.json` (the stylesheet it was cut
 * from, route-name patterns → template, sha256 per file, page modules). When that manifest belongs to the current
 * build, the template's CSS is inlined in a <style> (its hash is allowed by SecurityHeaders::policy()), the full
 * stylesheet is preloaded in the head and applied by a <link rel="stylesheet"> at the end of <body> (stack
 * `deferred-styles` in layouts/app — no JS, no inline handler), and the template's page modules get a modulepreload.
 * No build, `npm run dev`, a stale/missing manifest or a hash mismatch → the plain render-blocking @vite tags.
 */
final class Assets extends Component
{
    public const PRELOAD_FONTS = [
        'resources/fonts/lalezar-arabic-400-normal.woff2',
        'resources/fonts/vazirmatn-arabic-600-normal.woff2',
    ];

    public const STYLESHEET = 'resources/css/app.css';

    public const SCRIPT = 'resources/js/app.js';

    public const CRITICAL_DIR = 'build/critical';

    /** @var list<string> */
    public readonly array $fonts;

    public readonly InstallMetadata $pwa;

    /** Inline critical CSS for this route (null → plain blocking stylesheet). */
    public readonly ?string $criticalCss;

    /** Full stylesheet URL (only set together with $criticalCss). */
    public readonly ?string $stylesheet;

    /** @var list<string> modulepreload URLs of the template's page modules */
    public readonly array $modules;

    public function __construct(?Vite $vite = null)
    {
        $vite ??= app(Vite::class);
        $fonts = [];
        if (! $vite->isRunningHot()) {
            foreach (self::PRELOAD_FONTS as $font) {
                $url = self::asset($vite, $font);
                if ($url !== '') {
                    $fonts[] = $url;
                }
            }
        }
        $this->fonts = $fonts;

        [$this->criticalCss, $this->stylesheet, $this->modules] = $this->critical($vite, request()->route()?->getName());

        $this->pwa = app(WebManifest::class)->installMetadata();
    }

    public function render(): View
    {
        return view('components.layout.assets');
    }

    /**
     * The critical manifest when it was generated for the stylesheet of the current build, else null.
     *
     * @return array{stylesheet: string, default: string, routes: array<mixed>, templates: array<mixed>}|null
     */
    public static function criticalManifest(Vite $vite): ?array
    {
        if ($vite->isRunningHot()) {
            return null;
        }
        $url = self::asset($vite, self::STYLESHEET);
        $file = public_path(self::CRITICAL_DIR.'/manifest.json');
        if ($url === '' || ! is_file($file)) {
            return null;
        }
        $data = json_decode((string) file_get_contents($file), true);
        if (! is_array($data) || ! is_string($data['stylesheet'] ?? null) || ! is_array($data['templates'] ?? null)
            || ! is_array($data['routes'] ?? null) || ! is_string($data['default'] ?? null)) {
            return null;
        }
        if (! str_ends_with((string) parse_url($url, PHP_URL_PATH), '/'.ltrim($data['stylesheet'], '/'))) {
            return null; // generated for another build
        }

        /** @var array{stylesheet: string, default: string, routes: array<mixed>, templates: array<mixed>} $data */
        return $data;
    }

    /**
     * CSP source expressions ('sha256-…') for every inline critical stylesheet of the current build.
     *
     * @return list<string>
     */
    public static function criticalStyleHashes(Vite $vite): array
    {
        $hashes = [];
        foreach (self::criticalManifest($vite)['templates'] ?? [] as $template) {
            if (is_array($template) && is_string($template['sha256'] ?? null) && preg_match('#^[A-Za-z0-9+/]{43}=$#', $template['sha256']) === 1) {
                $hashes[] = "'sha256-{$template['sha256']}'";
            }
        }

        return array_values(array_unique($hashes));
    }

    /**
     * @return array{0: ?string, 1: ?string, 2: list<string>}
     */
    private function critical(Vite $vite, ?string $route): array
    {
        $manifest = self::criticalManifest($vite);
        if ($manifest === null) {
            return [null, null, []];
        }

        $key = $manifest['default'];
        if ($route !== null) {
            foreach ($manifest['routes'] as $pattern => $template) {
                if (is_string($pattern) && is_string($template) && Str::is($pattern, $route)) {
                    $key = $template;
                    break;
                }
            }
        }
        $template = $manifest['templates'][$key] ?? $manifest['templates'][$manifest['default']] ?? null;
        if (! is_array($template) || ! is_string($template['file'] ?? null) || ! is_string($template['sha256'] ?? null)) {
            return [null, null, []];
        }

        $path = public_path(self::CRITICAL_DIR.'/'.basename($template['file']));
        $css = is_file($path) ? (string) file_get_contents($path) : '';
        // The CSP allows exactly the manifest hash: never inline bytes that would be blocked.
        if ($css === '' || base64_encode(hash('sha256', $css, true)) !== $template['sha256']) {
            return [null, null, []];
        }

        $modules = [];
        $names = is_array($template['modules'] ?? null) ? $template['modules'] : [];
        foreach ($names as $name) {
            if (is_string($name) && preg_match('/^[a-z0-9-]+$/', $name) === 1) {
                $url = self::asset($vite, "resources/js/modules/{$name}.js");
                if ($url !== '') {
                    $modules[] = $url;
                }
            }
        }

        return [$css, self::asset($vite, self::STYLESHEET), $modules];
    }

    private static function asset(Vite $vite, string $entry): string
    {
        try {
            return (string) $vite->asset($entry);
        } catch (Throwable) {
            return ''; // no build yet / not in the manifest
        }
    }
}
