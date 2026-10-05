<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

/**
 * Store / download URLs (null = hidden). Only absolute http(s) URLs survive (L9-04): every one ends up in an href,
 * so a stored `javascript:`/`data:` value is dropped here instead of in each view.
 */
final readonly class AppLinksSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    public function __construct(
        public ?string $bazaar,
        public ?string $myket,
        public ?string $googlePlay,
        public ?string $appStore,
        public ?string $webApp,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::AppLinks;
    }

    public static function fromArray(array $values): static
    {
        return new self(
            bazaar: self::httpUrl(self::nullableString($values, 'bazaar')),
            myket: self::httpUrl(self::nullableString($values, 'myket')),
            googlePlay: self::httpUrl(self::nullableString($values, 'google_play')),
            appStore: self::httpUrl(self::nullableString($values, 'app_store')),
            webApp: self::httpUrl(self::nullableString($values, 'web_app')),
        );
    }

    private static function httpUrl(?string $url): ?string
    {
        return $url !== null && preg_match('#^https?://[^\s]+$#i', $url) === 1 ? $url : null;
    }

    public function toArray(): array
    {
        return [
            'bazaar' => $this->bazaar,
            'myket' => $this->myket,
            'google_play' => $this->googlePlay,
            'app_store' => $this->appStore,
            'web_app' => $this->webApp,
        ];
    }
}
