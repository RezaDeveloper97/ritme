<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\Actions;

use App\Domain\Seo\Indexing\HeadCode;
use App\Domain\Seo\Indexing\IndexingType;
use App\Domain\Seo\Indexing\IndexNow\IndexNowKey;
use App\Domain\Seo\Indexing\InvalidIndexingSettings;
use App\Domain\Seo\Indexing\RobotsTxtValidator;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\IndexingSettings;
use App\Domain\Settings\Data\SeoDefaults;
use App\Domain\Settings\Enums\SettingGroup;

/**
 * Validates and stores the indexing controls (robots rules, per-type robots, sitemap settings, verification codes,
 * IndexNow switch, head code) in the `seo` settings group. Only the keys passed are written; `head_code` is dropped
 * unless the caller may edit it (super-admin). Turning IndexNow on creates its key. Cache invalidation: SettingObserver
 * (settings / seo / pages) + IndexingSettingObserver (sitemap). Returns the changed keys with old / new values.
 */
final class SaveIndexingSettings
{
    public const KEYS = [...IndexingSettings::KEYS, 'verification'];

    public function __construct(
        private readonly UpdateSettings $update,
        private readonly SettingsRepository $settings,
        private readonly RobotsTxtValidator $robots,
        private readonly HeadCode $headCode,
    ) {}

    /**
     * @param  array<string, mixed>  $values
     * @return array{old: array<string, mixed>, new: array<string, mixed>}
     *
     * @throws InvalidIndexingSettings
     */
    public function handle(array $values, bool $mayEditHeadCode): array
    {
        $values = array_intersect_key($values, array_flip(self::KEYS));
        if (! $mayEditHeadCode) {
            unset($values['head_code']);
        }

        $errors = $this->errors($values);
        if ($errors !== []) {
            throw new InvalidIndexingSettings($errors);
        }

        if (array_key_exists('head_code', $values) && is_string($values['head_code'])) {
            $values['head_code'] = $this->headCode->sanitize($values['head_code']);
        }

        $current = $this->current();
        if (($values['indexnow_enabled'] ?? false) && ! IndexNowKey::isValid($current['indexnow_key'] ?? null)) {
            $values['indexnow_key'] = IndexNowKey::generate();
        }

        $new = $this->update->handle(SettingGroup::Seo, $values)->toArray();

        $changed = array_keys(array_filter(
            array_intersect_key($new, $values),
            static fn (mixed $value, string $key): bool => ($current[$key] ?? null) !== $value,
            ARRAY_FILTER_USE_BOTH,
        ));

        return [
            'old' => array_intersect_key($current, array_flip($changed)),
            'new' => array_intersect_key($new, array_flip($changed)),
        ];
    }

    /**
     * Validation errors per key (empty = valid). Also used by the admin form for inline field errors.
     *
     * @param  array<string, mixed>  $values
     * @return array<string, list<string>>
     */
    public function errors(array $values): array
    {
        $errors = [];

        if (is_string($values['robots_txt'] ?? null) && ($robots = $this->robots->errors($values['robots_txt'])) !== []) {
            $errors['robots_txt'] = $robots;
        }
        if (is_string($values['head_code'] ?? null) && ($head = $this->headCode->errors($values['head_code'])) !== []) {
            $errors['head_code'] = $head;
        }

        foreach (is_array($values['robots_types'] ?? null) ? $values['robots_types'] : [] as $type => $robots) {
            if (IndexingType::tryFrom((string) $type) === null || ! (is_string($robots) && ($robots === '' || array_key_exists($robots, IndexingType::ROBOTS_OPTIONS)))) {
                $errors['robots_types'][] = 'مقدار robots نوع «'.$type.'» نامعتبر است.';
            }
        }
        foreach (is_array($values['sitemap_exclude'] ?? null) ? $values['sitemap_exclude'] : [] as $key) {
            if (! is_string($key) || IndexingType::tryFrom($key) === null) {
                $errors['sitemap_exclude'][] = 'نوع نقشه سایت نامعتبر است.';
            }
        }
        foreach (is_array($values['sitemap_priorities'] ?? null) ? $values['sitemap_priorities'] : [] as $key => $priority) {
            if (IndexingType::tryFrom((string) $key) === null || ! ($priority === null || $priority === '' || (is_numeric($priority) && (float) $priority >= 0 && (float) $priority <= 1))) {
                $errors['sitemap_priorities'][] = 'اولویت باید عددی بین ۰ تا ۱ باشد.';
            }
        }
        foreach (is_array($values['sitemap_changefreq'] ?? null) ? $values['sitemap_changefreq'] : [] as $key => $freq) {
            if (IndexingType::tryFrom((string) $key) === null || ! ($freq === null || $freq === '' || (is_string($freq) && array_key_exists($freq, IndexingType::CHANGEFREQ_OPTIONS)))) {
                $errors['sitemap_changefreq'][] = 'بسامد تغییر نامعتبر است.';
            }
        }
        foreach (is_array($values['sitemap_ping_urls'] ?? null) ? $values['sitemap_ping_urls'] : [] as $url) {
            if (! is_string($url) || filter_var(str_replace('{sitemap}', 'x', $url), FILTER_VALIDATE_URL) === false || ! str_starts_with($url, 'https://')) {
                $errors['sitemap_ping_urls'][] = 'نشانی پینگ باید یک نشانی کامل https باشد.';
            }
        }
        foreach (is_array($values['verification'] ?? null) ? $values['verification'] : [] as $engine => $code) {
            if ($code !== null && $code !== '' && (! is_string($code) || preg_match('/^[A-Za-z0-9_\-.=+\/:]{1,200}$/', trim($code)) !== 1)) {
                $errors['verification'][] = 'کد تأیید «'.$engine.'» فقط می‌تواند حروف لاتین، عدد و - _ . = + / : داشته باشد (بدون تگ HTML).';
            }
        }

        return array_map(static fn (array $messages): array => array_values(array_unique($messages)), $errors);
    }

    /**
     * @return array<string, mixed>
     */
    private function current(): array
    {
        $seo = $this->settings->group(SettingGroup::Seo);

        return $seo instanceof SeoDefaults ? $seo->toArray() : [];
    }
}
