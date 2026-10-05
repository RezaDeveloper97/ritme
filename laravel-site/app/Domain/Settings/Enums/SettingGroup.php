<?php

declare(strict_types=1);

namespace App\Domain\Settings\Enums;

use App\Domain\Settings\Data\AppLinksSettings;
use App\Domain\Settings\Data\ContactSettings;
use App\Domain\Settings\Data\GeneralSettings;
use App\Domain\Settings\Data\LegalSettings;
use App\Domain\Settings\Data\OrganizationSettings;
use App\Domain\Settings\Data\PwaSettings;
use App\Domain\Settings\Data\SeoDefaults;
use App\Domain\Settings\Data\SettingsGroupData;
use App\Domain\Settings\Data\ShopSettings;
use App\Domain\Settings\Data\SocialSettings;
use App\Domain\Settings\Exceptions\UnknownSettingException;

/**
 * The settings groups. Each maps to a typed readonly DTO whose toArray() keys are the only valid keys.
 */
enum SettingGroup: string
{
    case General = 'general';
    case Contact = 'contact';
    case Social = 'social';
    case AppLinks = 'app_links';
    case Seo = 'seo';
    case Organization = 'organization';
    case Legal = 'legal';
    case Pwa = 'pwa';
    case Shop = 'shop';

    public static function fromName(string $group): self
    {
        return self::tryFrom($group) ?? throw UnknownSettingException::group($group);
    }

    /**
     * @return class-string<SettingsGroupData>
     */
    public function dataClass(): string
    {
        return match ($this) {
            self::General => GeneralSettings::class,
            self::Contact => ContactSettings::class,
            self::Social => SocialSettings::class,
            self::AppLinks => AppLinksSettings::class,
            self::Seo => SeoDefaults::class,
            self::Organization => OrganizationSettings::class,
            self::Legal => LegalSettings::class,
            self::Pwa => PwaSettings::class,
            self::Shop => ShopSettings::class,
        };
    }

    /**
     * @param  array<string, mixed>  $values
     */
    public function data(array $values = []): SettingsGroupData
    {
        return $this->dataClass()::fromArray($values);
    }

    /**
     * @return list<string>
     */
    public function keys(): array
    {
        return array_keys($this->data()->toArray());
    }

    public function assertKey(string $key): void
    {
        if (! in_array($key, $this->keys(), true)) {
            throw UnknownSettingException::key($this, $key);
        }
    }
}
