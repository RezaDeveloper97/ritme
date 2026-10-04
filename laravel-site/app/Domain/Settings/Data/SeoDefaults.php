<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

final readonly class SeoDefaults implements SettingsGroupData
{
    use CoercesSettingValues;

    /**
     * @param  array<string, string>  $verification  e.g. ['google' => '…', 'bing' => '…']
     */
    public function __construct(
        public string $titleTemplate,
        public string $separator,
        public string $defaultTitle,
        public string $defaultDescription,
        public ?int $defaultOgMediaId,
        public ?string $twitterHandle,
        public array $verification,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Seo;
    }

    public static function fromArray(array $values): static
    {
        $template = self::string($values, 'title_template', '%s — ریتمی');

        return new self(
            titleTemplate: str_contains($template, '%s') ? $template : '%s — ریتمی',
            separator: self::string($values, 'separator', '—'),
            defaultTitle: self::string($values, 'default_title', 'ریتمی'),
            defaultDescription: self::string($values, 'default_description'),
            defaultOgMediaId: self::nullableInt($values, 'default_og_media_id'),
            twitterHandle: self::nullableString($values, 'twitter_handle'),
            verification: self::stringMap($values, 'verification'),
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
        ];
    }

    /**
     * Applies the title template to a page title; an empty title yields the default title.
     */
    public function title(?string $pageTitle): string
    {
        $pageTitle = trim((string) $pageTitle);

        return $pageTitle === '' ? $this->defaultTitle : str_replace('%s', $pageTitle, $this->titleTemplate);
    }
}
