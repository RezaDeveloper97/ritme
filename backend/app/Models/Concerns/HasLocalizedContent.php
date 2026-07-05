<?php

namespace App\Models\Concerns;

/**
 * Adds locale-aware accessors for JSON columns that store bilingual content
 * in the shape ["fa" => "...", "en" => "..."].
 *
 * Mirrors the convention already used by PregnancyWeeklyContent so all
 * content models resolve translations the same way.
 */
trait HasLocalizedContent
{
    /**
     * Resolve a localized value from a bilingual JSON attribute.
     *
     * Falls back gracefully: requested locale -> fa -> en -> raw value.
     */
    public function localized(string $field, string $locale = 'fa'): mixed
    {
        return static::pickLocale($this->{$field}, $locale);
    }

    /**
     * Pick the best matching translation from a bilingual value.
     *
     * Accepts a ["fa" => ..., "en" => ...] array; returns scalars untouched.
     */
    public static function pickLocale(mixed $value, string $locale = 'fa'): mixed
    {
        if (!is_array($value)) {
            return $value;
        }

        return $value[$locale]
            ?? $value['fa']
            ?? $value['en']
            ?? null;
    }
}
