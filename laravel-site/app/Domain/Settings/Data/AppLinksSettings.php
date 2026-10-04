<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

/**
 * Store / download URLs (null = hidden).
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
            bazaar: self::nullableString($values, 'bazaar'),
            myket: self::nullableString($values, 'myket'),
            googlePlay: self::nullableString($values, 'google_play'),
            appStore: self::nullableString($values, 'app_store'),
            webApp: self::nullableString($values, 'web_app'),
        );
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
