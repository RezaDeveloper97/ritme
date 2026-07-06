<?php

namespace App\Http\Controllers\Concerns;

use Illuminate\Http\Request;

/**
 * Normalizes the Accept-Language header to a supported locale.
 *
 * Browsers send values like "fa-IR,fa;q=0.9,en;q=0.8" — using the raw header
 * as a locale key breaks translation lookups, so every API controller must
 * resolve it through this trait instead of reading the header directly.
 */
trait ResolvesLocale
{
    protected function resolveLocale(Request $request, string $default = 'fa'): string
    {
        $header = $request->header('Accept-Language', $default);
        $primary = strtolower(strtok($header, ',;'));
        $primary = explode('-', $primary)[0];

        return in_array($primary, ['fa', 'en'], true) ? $primary : $default;
    }
}
