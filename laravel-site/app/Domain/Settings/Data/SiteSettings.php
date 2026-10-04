<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Settings\Exceptions\UnknownSettingException;

/**
 * Every settings group, loaded together (one query, or one cache read).
 */
final readonly class SiteSettings
{
    public function __construct(
        public GeneralSettings $general,
        public ContactSettings $contact,
        public SocialSettings $social,
        public AppLinksSettings $appLinks,
        public SeoDefaults $seo,
        public OrganizationSettings $organization,
        public LegalSettings $legal,
        public PwaSettings $pwa,
    ) {}

    /**
     * @param  array<string, array<string, mixed>>  $values  group => key => value; unknown groups/keys are ignored
     */
    public static function fromArray(array $values): self
    {
        $group = static fn (SettingGroup $group): array => is_array($values[$group->value] ?? null) ? $values[$group->value] : [];

        return new self(
            general: GeneralSettings::fromArray($group(SettingGroup::General)),
            contact: ContactSettings::fromArray($group(SettingGroup::Contact)),
            social: SocialSettings::fromArray($group(SettingGroup::Social)),
            appLinks: AppLinksSettings::fromArray($group(SettingGroup::AppLinks)),
            seo: SeoDefaults::fromArray($group(SettingGroup::Seo)),
            organization: OrganizationSettings::fromArray($group(SettingGroup::Organization)),
            legal: LegalSettings::fromArray($group(SettingGroup::Legal)),
            pwa: PwaSettings::fromArray($group(SettingGroup::Pwa)),
        );
    }

    /**
     * @return array<string, array<string, mixed>>
     */
    public function toArray(): array
    {
        $all = [];
        foreach (SettingGroup::cases() as $group) {
            $all[$group->value] = $this->group($group)->toArray();
        }

        return $all;
    }

    public function group(SettingGroup $group): SettingsGroupData
    {
        return match ($group) {
            SettingGroup::General => $this->general,
            SettingGroup::Contact => $this->contact,
            SettingGroup::Social => $this->social,
            SettingGroup::AppLinks => $this->appLinks,
            SettingGroup::Seo => $this->seo,
            SettingGroup::Organization => $this->organization,
            SettingGroup::Legal => $this->legal,
            SettingGroup::Pwa => $this->pwa,
        };
    }

    /**
     * @throws UnknownSettingException
     */
    public function get(SettingGroup $group, string $key): mixed
    {
        $group->assertKey($key);

        return $this->group($group)->toArray()[$key];
    }
}
