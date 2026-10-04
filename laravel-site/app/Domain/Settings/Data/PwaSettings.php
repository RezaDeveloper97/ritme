<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

/**
 * Web app manifest values (L8-01: App\Domain\Pwa builds /manifest.webmanifest and the install head tags from them).
 * `iconMediaId` = square logo (PNG/WebP/JPG, at least 512 px) the PWA icons are generated from; null = the committed
 * defaults in public/icons.
 *
 * Two-tier update (L8-02, served by /pwa/version.json): `minBuildId` = oldest build id still allowed — a visitor on
 * an older build gets the blocking update screen (null = never forced); `updateMessage` = optional text shown in
 * the update toast / screen instead of the default copy.
 */
final readonly class PwaSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    public const BUILD_ID_PATTERN = '/^\d{14}(-[0-9A-Za-z]{1,40})?$/';

    public function __construct(
        public string $name,
        public string $shortName,
        public string $description,
        public string $themeColor,
        public string $backgroundColor,
        public ?int $iconMediaId = null,
        public ?string $minBuildId = null,
        public string $updateMessage = '',
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Pwa;
    }

    public static function fromArray(array $values): static
    {
        return new self(
            name: self::string($values, 'name', 'ریتمی — همراه سلامت زنان'),
            shortName: self::string($values, 'short_name', 'ریتمی'),
            description: self::string($values, 'description'),
            themeColor: self::color($values, 'theme_color', '#17112B'),
            backgroundColor: self::color($values, 'background_color', '#FFFFFF'),
            iconMediaId: self::nullableInt($values, 'icon_media_id'),
            minBuildId: self::buildId($values, 'min_build_id'),
            updateMessage: self::string($values, 'update_message'),
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
            'icon_media_id' => $this->iconMediaId,
            'min_build_id' => $this->minBuildId,
            'update_message' => $this->updateMessage,
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

    /**
     * A build id (`YYYYMMDDHHmmss[-sha]`) or null — anything else would make the forced tier unpredictable.
     *
     * @param  array<string, mixed>  $values
     */
    private static function buildId(array $values, string $key): ?string
    {
        $value = self::nullableString($values, $key);

        return $value !== null && preg_match(self::BUILD_ID_PATTERN, $value) === 1 ? strtolower($value) : null;
    }
}
