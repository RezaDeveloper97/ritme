<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

final readonly class OrganizationSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    /**
     * @param  list<string>  $sameAs  extra profile URLs (social URLs are added from SocialSettings by the JSON-LD builder)
     */
    public function __construct(
        public string $legalName,
        public ?int $logoMediaId,
        public ?string $foundingDate,
        public array $sameAs,
        public ContactPointData $contactPoint,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Organization;
    }

    public static function fromArray(array $values): static
    {
        $foundingDate = self::nullableString($values, 'founding_date');
        $contactPoint = $values['contact_point'] ?? [];

        return new self(
            legalName: self::string($values, 'legal_name', 'ریتمی'),
            logoMediaId: self::nullableInt($values, 'logo_media_id'),
            foundingDate: $foundingDate !== null && preg_match('/^\d{4}(-\d{2}(-\d{2})?)?$/', $foundingDate) === 1 ? $foundingDate : null,
            sameAs: self::stringList($values, 'same_as'),
            contactPoint: ContactPointData::fromArray(is_array($contactPoint) ? $contactPoint : []),
        );
    }

    public function toArray(): array
    {
        return [
            'legal_name' => $this->legalName,
            'logo_media_id' => $this->logoMediaId,
            'founding_date' => $this->foundingDate,
            'same_as' => $this->sameAs,
            'contact_point' => $this->contactPoint->toArray(),
        ];
    }
}
