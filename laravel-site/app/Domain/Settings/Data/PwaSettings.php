<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

/**
 * Web app manifest values (consumed by the PWA milestone).
 */
final readonly class PwaSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    public function __construct(
        public string $name,
        public string $shortName,
        public string $description,
        public string $themeColor,
        public string $backgroundColor,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Pwa;
    }

    public static function fromArray(array $values): static
    {
        return new self(
            name: self::string($values, 'name', 'ریتمی'),
            shortName: self::string($values, 'short_name', 'ریتمی'),
            description: self::string($values, 'description'),
            themeColor: self::color($values, 'theme_color', '#17112B'),
            backgroundColor: self::color($values, 'background_color', '#FFFFFF'),
        );
    }

    public function toArray(): array
    {
        return [
            'name' => $this->name,
            'short_name' => $this->shortName,
            'description' => $this->description,
            'theme_color' => $this->themeColor,
            'background_color' => $this->backgroundColor,
        ];
    }

    /**
     * @param  array<string, mixed>  $values
     */
    private static function color(array $values, string $key, string $default): string
    {
        $value = self::string($values, $key, $default);

        return preg_match('/^#[0-9A-Fa-f]{6}$/', $value) === 1 ? $value : $default;
    }
}
