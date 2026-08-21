<?php

namespace App\Services\Language;

use App\Enums\TextDirection;
use App\Models\Language;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Cache;
use Throwable;

/**
 * The one place that answers "which locales exist, and what is the default?".
 *
 * Every other layer — admin forms, validation, API locale resolution, content
 * fallbacks — asks this service instead of hardcoding fa/en. Results are cached
 * because they are read on nearly every request and change only when an admin
 * edits the language list (which calls {@see flush()}).
 *
 * It degrades safely: before the migration runs (or if the table is empty) it
 * returns the two bootstrap locales, so a fresh checkout and the installer both
 * work without a seeded database.
 */
class LanguageRegistry
{
    private const CACHE_KEY = 'languages.registry';

    /** Used only when the table is missing or empty — mirrors LanguageSeeder. */
    public const BOOTSTRAP = [
        ['code' => 'fa', 'name' => 'فارسی', 'english_name' => 'Persian', 'direction' => 'rtl', 'is_default' => true],
        ['code' => 'en', 'name' => 'English', 'english_name' => 'English', 'direction' => 'ltr', 'is_default' => false],
    ];

    /** @var array<int, array{code: string, name: string, english_name: string, direction: string, is_default: bool}>|null */
    private ?array $memo = null;

    /**
     * All active locales, in display order.
     *
     * @return array<int, array{code: string, name: string, english_name: string, direction: string, is_default: bool}>
     */
    public function all(): array
    {
        if ($this->memo !== null) {
            return $this->memo;
        }

        return $this->memo = Cache::rememberForever(self::CACHE_KEY, fn (): array => $this->load());
    }

    /** @return array<int, string> */
    public function codes(): array
    {
        return array_column($this->all(), 'code');
    }

    /** The default locale's code — the one content must always be filled in. */
    public function defaultCode(): string
    {
        foreach ($this->all() as $language) {
            if ($language['is_default']) {
                return $language['code'];
            }
        }

        return $this->all()[0]['code'] ?? 'fa';
    }

    public function isSupported(?string $code): bool
    {
        return $code !== null && in_array(Language::normalizeCode($code), $this->codes(), true);
    }

    /**
     * Resolve an arbitrary client-supplied locale (query param, Accept-Language)
     * to a supported code, falling back to the default. Accepts full
     * Accept-Language values like "en-US,en;q=0.9" and region-only matches.
     */
    public function resolve(?string $requested): string
    {
        if ($requested === null || $requested === '') {
            return $this->defaultCode();
        }

        $codes = $this->codes();

        foreach (explode(',', $requested) as $part) {
            $candidate = Language::normalizeCode(explode(';', $part)[0]);

            if ($candidate === '' || $candidate === '*') {
                continue;
            }

            if (in_array($candidate, $codes, true)) {
                return $candidate;
            }

            // "en-US" should still match a plain "en" (and vice versa).
            $base = explode('-', $candidate)[0];

            foreach ($codes as $code) {
                if ($code === $base || explode('-', $code)[0] === $base) {
                    return $code;
                }
            }
        }

        return $this->defaultCode();
    }

    public function direction(string $code): TextDirection
    {
        foreach ($this->all() as $language) {
            if ($language['code'] === $code) {
                return TextDirection::from($language['direction']);
            }
        }

        return TextDirection::LTR;
    }

    public function name(string $code): string
    {
        foreach ($this->all() as $language) {
            if ($language['code'] === $code) {
                return $language['name'];
            }
        }

        return $code;
    }

    /**
     * Locales as a keyed collection for Blade loops: code => label.
     *
     * @return Collection<string, string>
     */
    public function labels(): Collection
    {
        return collect($this->all())->mapWithKeys(
            fn (array $language): array => [$language['code'] => $language['name']]
        );
    }

    /** Drop the cache after any write to the languages table. */
    public function flush(): void
    {
        $this->memo = null;
        Cache::forget(self::CACHE_KEY);
    }

    /** @return array<int, array{code: string, name: string, english_name: string, direction: string, is_default: bool}> */
    private function load(): array
    {
        try {
            $rows = Language::query()->active()->ordered()->get();
        } catch (Throwable) {
            // Table not migrated yet (fresh checkout, `artisan migrate` itself).
            return self::BOOTSTRAP;
        }

        if ($rows->isEmpty()) {
            return self::BOOTSTRAP;
        }

        return $rows->map(fn (Language $language): array => [
            'code' => $language->code,
            'name' => $language->name,
            'english_name' => $language->english_name,
            'direction' => $language->direction->value,
            'is_default' => $language->is_default,
        ])->all();
    }
}
