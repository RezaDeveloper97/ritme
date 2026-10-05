<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

final readonly class SeoDefaults implements SettingsGroupData
{
    use CoercesSettingValues;

    /** Placeholder in the title template replaced by the separator (L7-06), e.g. «%s %sep% ریتمی». */
    public const SEPARATOR_TOKEN = '%sep%';

    /**
     * @param  array<string, string>  $verification  e.g. ['google' => '…', 'bing' => '…']
     * @param  IndexingSettings  $indexing  robots / sitemap / IndexNow / head-code controls (L7-04), flat keys of this group
     */
    public function __construct(
        public string $titleTemplate,
        public string $separator,
        public string $defaultTitle,
        public string $defaultDescription,
        public ?int $defaultOgMediaId,
        public ?string $twitterHandle,
        public array $verification,
        public IndexingSettings $indexing = new IndexingSettings,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Seo;
    }

    public static function fromArray(array $values): static
    {
        $template = self::string($values, 'title_template', '%s — ریتمی');

        return new self(
            titleTemplate: str_contains(str_replace(self::SEPARATOR_TOKEN, '', $template), '%s') ? $template : '%s — ریتمی',
            separator: self::string($values, 'separator', '—'),
            defaultTitle: self::string($values, 'default_title', 'ریتمی'),
            defaultDescription: self::string($values, 'default_description'),
            defaultOgMediaId: self::nullableInt($values, 'default_og_media_id'),
            twitterHandle: self::nullableString($values, 'twitter_handle'),
            verification: self::stringMap($values, 'verification'),
            indexing: IndexingSettings::fromArray($values),
        );
    }

    public function toArray(): array
    {
        return [
            'title_template' => $this->titleTemplate,
            'separator' => $this->separator,
            'default_title' => $this->defaultTitle,
            'default_description' => $this->defaultDescription,
            'default_og_media_id' => $this->defaultOgMediaId,
            'twitter_handle' => $this->twitterHandle,
            'verification' => $this->verification,
            ...$this->indexing->toArray(),
        ];
    }

    /**
     * The title template with the separator token resolved (still contains `%s`).
     */
    public function template(): string
    {
        return str_replace(self::SEPARATOR_TOKEN, $this->separator, $this->titleTemplate);
    }

    /**
     * Applies the title template to a page title; an empty title yields the default title.
     */
    public function title(?string $pageTitle): string
    {
        $pageTitle = trim((string) $pageTitle);

        return $pageTitle === '' ? $this->defaultTitle : str_replace('%s', $pageTitle, $this->template());
    }
}
