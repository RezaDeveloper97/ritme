<?php

namespace App\Services\Language;

use Illuminate\Support\Arr;
use Illuminate\Support\Facades\File;

/**
 * Reads and writes the app's UI-string bundles — the JSON files the frontend
 * renders its interface from, one folder per locale and one file per namespace
 * ("home", "profile", …).
 *
 * Two layers, in priority order:
 *
 *   1. storage/app/translations/{code}/{ns}.json  — live, admin-editable, and
 *      what `php artisan` generates when a language is created. This lives on
 *      the persisted `backend-storage` volume, so edits survive a redeploy.
 *   2. resources/translations/{code}/{ns}.json    — the seed shipped in the
 *      repo, kept in sync with `frontend/messages` by `translations:import`.
 *      Read-only at runtime (the image's resources/ is not writable).
 *
 * Every read is finally backfilled from the DEFAULT locale, so a key an admin
 * has not translated yet renders the default language's text instead of a blank
 * string. That is what makes adding a language safe at any moment.
 */
class TranslationStore
{
    public function __construct(private readonly LanguageRegistry $registry) {}

    /** Where generated / edited bundles live (persisted volume). */
    public function livePath(string $code = '', string $namespace = ''): string
    {
        return rtrim(implode('/', array_filter([
            storage_path('app/translations'),
            $code,
            $namespace === '' ? '' : "{$namespace}.json",
        ])), '/');
    }

    /** Where the repo-shipped seed lives (read-only). */
    public function seedPath(string $code = '', string $namespace = ''): string
    {
        return rtrim(implode('/', array_filter([
            resource_path('translations'),
            $code,
            $namespace === '' ? '' : "{$namespace}.json",
        ])), '/');
    }

    /**
     * Every namespace the app has strings for, sorted.
     *
     * Derived from the union of the default locale's seed and live folders, so
     * a namespace added by a frontend sync shows up in the editor immediately.
     *
     * @return array<int, string>
     */
    public function namespaces(): array
    {
        $default = $this->registry->defaultCode();

        $names = collect([$this->seedPath($default), $this->livePath($default)])
            ->filter(fn (string $dir): bool => File::isDirectory($dir))
            ->flatMap(fn (string $dir): array => File::files($dir))
            ->filter(fn ($file): bool => $file->getExtension() === 'json')
            ->map(fn ($file): string => $file->getFilenameWithoutExtension())
            ->unique()
            ->sort()
            ->values();

        return $names->all();
    }

    /**
     * One namespace for one locale, with the default locale backfilled underneath.
     *
     * @return array<string, mixed>
     */
    public function namespaceMessages(string $code, string $namespace): array
    {
        $default = $this->registry->defaultCode();

        $base = $code === $default ? [] : $this->rawNamespace($default, $namespace);

        return $this->deepMerge($base, $this->rawNamespace($code, $namespace));
    }

    /**
     * The complete message bundle for a locale: namespace => nested messages.
     * This is exactly what the frontend consumes.
     *
     * @return array<string, array<string, mixed>>
     */
    public function bundle(string $code): array
    {
        $bundle = [];

        foreach ($this->namespaces() as $namespace) {
            $bundle[$namespace] = $this->namespaceMessages($code, $namespace);
        }

        return $bundle;
    }

    /**
     * The locale's OWN stored strings for a namespace — no default backfill.
     * The editor needs this to tell "translated" from "inherited".
     *
     * @return array<string, mixed>
     */
    public function rawNamespace(string $code, string $namespace): array
    {
        $seed = $this->readJson($this->seedPath($code, $namespace));
        $live = $this->readJson($this->livePath($code, $namespace));

        return $this->deepMerge($seed, $live);
    }

    /**
     * Persist one namespace for one locale. Values are stored as-is; empty
     * strings are dropped so the key falls back to the default locale rather
     * than rendering blank.
     *
     * @param  array<string, mixed>  $messages  nested (not dot-notation)
     */
    public function writeNamespace(string $code, string $namespace, array $messages): void
    {
        $path = $this->livePath($code, $namespace);
        File::ensureDirectoryExists(dirname($path));
        File::put(
            $path,
            json_encode($this->prune($messages), JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)."\n"
        );
    }

    /**
     * Create a brand-new locale's files by copying another locale's text.
     *
     * The copy (rather than a set of empty keys) is deliberate: the app is
     * fully usable in the new language the moment it is created, and the admin
     * replaces strings one namespace at a time in /admin/languages/{code}/translations
     * instead of facing a blank slate.
     *
     * @return int number of namespace files written
     */
    public function generateFor(string $code, ?string $copyFrom = null): int
    {
        $source = $copyFrom ?? $this->registry->defaultCode();
        $written = 0;

        foreach ($this->namespaces() as $namespace) {
            // The source's own strings, already backfilled from the default,
            // so the new locale starts complete even if the source was partial.
            $messages = $this->namespaceMessages($source, $namespace);

            if ($messages === []) {
                continue;
            }

            $this->writeNamespace($code, $namespace, $messages);
            $written++;
        }

        return $written;
    }

    /** Remove a locale's generated files (called when a language is deleted). */
    public function deleteFor(string $code): void
    {
        $dir = $this->livePath($code);

        if (File::isDirectory($dir)) {
            File::deleteDirectory($dir);
        }
    }

    /**
     * Flatten a nested namespace to dot-notation for the editor form, keeping
     * a stable order so the page doesn't reshuffle between saves.
     *
     * @param  array<string, mixed>  $messages
     * @return array<string, string>
     */
    public function flatten(array $messages): array
    {
        $flat = [];

        foreach (Arr::dot($messages) as $key => $value) {
            // Arrays of strings (e.g. profileInfo sections) flatten to indexed
            // keys, which round-trip fine through Arr::undot.
            $flat[$key] = is_scalar($value) ? (string) $value : '';
        }

        return $flat;
    }

    /**
     * Rebuild a nested namespace from the editor's dot-notation payload.
     *
     * @param  array<string, string>  $flat
     * @return array<string, mixed>
     */
    public function unflatten(array $flat): array
    {
        return Arr::undot($flat);
    }

    /** @return array<string, mixed> */
    private function readJson(string $path): array
    {
        if (! is_file($path)) {
            return [];
        }

        $decoded = json_decode((string) file_get_contents($path), true);

        return is_array($decoded) ? $decoded : [];
    }

    /**
     * Recursive merge where `$override` wins, but only for keys it actually
     * defines — a partially translated namespace keeps the base's other keys.
     *
     * @param  array<string, mixed>  $base
     * @param  array<string, mixed>  $override
     * @return array<string, mixed>
     */
    private function deepMerge(array $base, array $override): array
    {
        foreach ($override as $key => $value) {
            if (is_array($value) && isset($base[$key]) && is_array($base[$key])) {
                $base[$key] = $this->deepMerge($base[$key], $value);

                continue;
            }

            if ($value === null || $value === '') {
                continue;
            }

            $base[$key] = $value;
        }

        return $base;
    }

    /**
     * Drop empty leaves so they inherit from the default locale on read.
     *
     * @param  array<string, mixed>  $messages
     * @return array<string, mixed>
     */
    private function prune(array $messages): array
    {
        $clean = [];

        foreach ($messages as $key => $value) {
            if (is_array($value)) {
                $nested = $this->prune($value);

                if ($nested !== []) {
                    $clean[$key] = $nested;
                }

                continue;
            }

            if ($value === null || $value === '') {
                continue;
            }

            $clean[$key] = $value;
        }

        return $clean;
    }
}
