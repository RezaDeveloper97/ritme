<?php

namespace App\Support;

use App\Services\Language\LanguageRegistry;

/**
 * Helpers for the JSON columns that hold one value per locale
 * (`{"fa": "...", "en": "...", "ar": "..."}`).
 *
 * The shape grows with the languages table: add a locale in /admin/languages
 * and every admin form, validator and reader below covers it with no code
 * change. Nothing in the app may enumerate locales by hand — go through here.
 */
class Translatable
{
    /**
     * Validation rules for one translatable field.
     *
     * Only the DEFAULT locale is required: an admin must be able to publish
     * content before it has been translated, and readers fall back to the
     * default locale anyway (see {@see pick()}). Every other active locale is
     * optional but validated with the same constraints.
     *
     * @param  array<int, string>  $extra  per-locale rules, e.g. ['max:255']
     * @return array<string, array<int, string>>
     */
    public static function rules(string $field, bool $required = true, array $extra = []): array
    {
        $registry = app(LanguageRegistry::class);
        $default = $registry->defaultCode();

        $rules = [$field => [$required ? 'required' : 'nullable', 'array']];

        foreach ($registry->codes() as $code) {
            $isDefault = $code === $default;

            $rules["{$field}.{$code}"] = array_merge(
                [$required && $isDefault ? 'required' : 'nullable', 'string'],
                $extra
            );
        }

        return $rules;
    }

    /**
     * Merge rule sets for several translatable fields into one array.
     *
     * @param  array<string, array{required?: bool, extra?: array<int, string>}>  $fields
     * @return array<string, array<int, string>>
     */
    public static function rulesFor(array $fields): array
    {
        $rules = [];

        foreach ($fields as $field => $options) {
            $rules += self::rules(
                $field,
                $options['required'] ?? true,
                $options['extra'] ?? []
            );
        }

        return $rules;
    }

    /**
     * Read the best available translation.
     *
     * Order: the requested locale, then the default locale, then the first
     * non-empty value present. A half-translated row therefore renders real
     * text rather than a blank, in the admin list and in the API alike.
     */
    public static function pick(mixed $value, ?string $locale = null): mixed
    {
        if (! is_array($value)) {
            return $value;
        }

        $registry = app(LanguageRegistry::class);
        $locale ??= app()->getLocale();

        foreach ([$locale, $registry->defaultCode()] as $candidate) {
            if (isset($value[$candidate]) && $value[$candidate] !== '') {
                return $value[$candidate];
            }
        }

        foreach ($value as $translation) {
            if ($translation !== null && $translation !== '') {
                return $translation;
            }
        }

        return null;
    }

    /**
     * Strip locales that carry no text, so an untouched input never persists
     * as `"en": ""` and shadows the fallback.
     *
     * @param  array<string, mixed>|null  $value
     * @return array<string, mixed>|null null when nothing was filled in
     */
    public static function clean(mixed $value): ?array
    {
        if (! is_array($value)) {
            return null;
        }

        $clean = array_filter(
            $value,
            fn ($translation): bool => $translation !== null && $translation !== ''
        );

        return $clean === [] ? null : $clean;
    }
}
