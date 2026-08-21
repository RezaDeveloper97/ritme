<?php

namespace App\Models\Concerns;

use App\Support\Translatable;

/**
 * Adds locale-aware accessors for JSON columns that store one value per
 * locale, in the shape ["fa" => "...", "en" => "...", "ar" => "..."].
 *
 * The set of keys is whatever the languages table holds (see
 * App\Services\Language\LanguageRegistry) — never a fixed fa/en pair. Reads
 * fall back so a row that has not been translated yet still renders text.
 */
trait HasLocalizedContent
{
    /**
     * Resolve a localized value from a translatable JSON attribute.
     *
     * Falls back gracefully: requested locale -> default locale -> the first
     * non-empty translation present.
     */
    public function localized(string $field, ?string $locale = null): mixed
    {
        return Translatable::pick($this->{$field}, $locale);
    }

    /**
     * Pick the best matching translation from a translatable value.
     * Accepts a locale-keyed array; returns scalars untouched.
     */
    public static function pickLocale(mixed $value, ?string $locale = null): mixed
    {
        return Translatable::pick($value, $locale);
    }
}
