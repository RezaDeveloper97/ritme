<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Content\Enums\SiteSection;
use App\Domain\Settings\Enums\SettingGroup;

/**
 * Which public site sections are switched on (admin «صفحه‌ها»). One boolean per SiteSection, keyed by its value;
 * anything missing or unreadable counts as on, so a fresh install shows the whole site.
 */
final readonly class PagesSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    /**
     * @param  array<string, bool>  $enabled  SiteSection value => on
     */
    public function __construct(public array $enabled = []) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Pages;
    }

    public static function fromArray(array $values): static
    {
        $enabled = [];
        foreach (SiteSection::cases() as $section) {
            $enabled[$section->value] = self::bool($values, $section->value, true);
        }

        return new self($enabled);
    }

    public function toArray(): array
    {
        $all = [];
        foreach (SiteSection::cases() as $section) {
            $all[$section->value] = $this->enabled($section);
        }

        return $all;
    }

    public function enabled(SiteSection $section): bool
    {
        return $this->enabled[$section->value] ?? true;
    }
}
