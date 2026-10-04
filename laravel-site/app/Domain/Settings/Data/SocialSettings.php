<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

/**
 * Social profile URLs (null = hidden).
 */
final readonly class SocialSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    public function __construct(
        public ?string $instagram,
        public ?string $telegram,
        public ?string $linkedin,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Social;
    }

    public static function fromArray(array $values): static
    {
        return new self(
            instagram: self::nullableString($values, 'instagram'),
            telegram: self::nullableString($values, 'telegram'),
            linkedin: self::nullableString($values, 'linkedin'),
        );
    }

    public function toArray(): array
    {
        return [
            'instagram' => $this->instagram,
            'telegram' => $this->telegram,
            'linkedin' => $this->linkedin,
        ];
    }

    /**
     * Filled profile URLs, for Organization `sameAs` and the footer.
     *
     * @return array<string, string>
     */
    public function filled(): array
    {
        return array_filter($this->toArray(), static fn (?string $url): bool => $url !== null);
    }
}
