<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

final readonly class GeneralSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    public function __construct(
        public string $siteName,
        public string $alternateName,
        public string $tagline,
        public string $footerNote,
        public string $emergencyNumber,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::General;
    }

    public static function fromArray(array $values): static
    {
        return new self(
            siteName: self::string($values, 'site_name', 'ریتمی'),
            alternateName: self::string($values, 'alternate_name', 'Ritme'),
            tagline: self::string($values, 'tagline'),
            footerNote: self::string($values, 'footer_note'),
            emergencyNumber: self::string($values, 'emergency_number', '115'),
        );
    }

    public function toArray(): array
    {
        return [
            'site_name' => $this->siteName,
            'alternate_name' => $this->alternateName,
            'tagline' => $this->tagline,
            'footer_note' => $this->footerNote,
            'emergency_number' => $this->emergencyNumber,
        ];
    }
}
