<?php

namespace App\Http\Controllers\Concerns;

use App\Services\Language\LanguageRegistry;
use Illuminate\Http\Request;

/**
 * Normalizes a request's requested language to a locale the app actually ships.
 *
 * Browsers send values like "fa-IR,fa;q=0.9,en;q=0.8" — using the raw header as
 * a locale key breaks translation lookups, so every API controller must resolve
 * it through this trait instead of reading the header directly.
 *
 * The supported set comes from the languages table (LanguageRegistry), never a
 * hardcoded list, so a locale an admin adds is accepted here the moment it
 * exists. `?locale=` wins over the header because browsers forbid JS from
 * overriding Accept-Language, so the web client passes it as a query param.
 */
trait ResolvesLocale
{
    protected function resolveLocale(Request $request, ?string $default = null): string
    {
        $registry = app(LanguageRegistry::class);

        $requested = $request->query('locale') ?? $request->header('Accept-Language');

        // A caller-supplied default only matters when the request said nothing
        // at all; otherwise `resolve()` already falls back to the product
        // default for anything it cannot match.
        if (! is_string($requested) || $requested === '') {
            return $registry->isSupported($default)
                ? $registry->resolve($default)
                : $registry->defaultCode();
        }

        return $registry->resolve($requested);
    }
}
