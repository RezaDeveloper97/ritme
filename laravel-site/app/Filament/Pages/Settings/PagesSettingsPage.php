<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Content\Enums\SiteSection;
use App\Domain\Settings\Enums\SettingGroup;
use BackedEnum;
use Filament\Forms\Components\Toggle;
use Filament\Schemas\Components\Section;
use Filament\Support\Icons\Heroicon;

/**
 * Switches public site sections on and off (SiteSection, settings group `pages`). A section that is off answers 404
 * (EnsureSectionEnabled) and leaves the main nav, footer, sitemap and search; its content stays editable here in the
 * admin. Super-admins only: one click can take a whole module off the site.
 */
final class PagesSettingsPage extends SettingsPage
{
    protected static array $roles = [];

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedEyeSlash;

    protected static ?string $navigationLabel = 'صفحه‌ها';

    protected static ?string $title = 'فعال و غیرفعال کردن صفحه‌ها';

    protected static ?string $slug = 'settings/pages';

    protected static ?int $navigationSort = 15;

    protected static function group(): SettingGroup
    {
        return SettingGroup::Pages;
    }

    protected function fields(): array
    {
        $toggles = static fn (array $sections): array => array_map(
            static fn (SiteSection $section): Toggle => Toggle::make($section->value)
                ->label($section->label())
                ->helperText($section->hint()),
            $sections,
        );

        return [
            Section::make('بخش‌های اصلی')
                ->description('بخش خاموش با همه‌ی صفحه‌هایش از سایت برداشته می‌شود (خطای ۴۰۴) و از منو، فوتر، نقشه‌ی سایت و جستجو هم حذف می‌شود. محتوایش در پنل می‌ماند و با روشن کردن دوباره برمی‌گردد.')
                ->columns(2)
                ->schema($toggles(SiteSection::modules())),
            Section::make('مرحله‌ها')
                ->description('اگر همه خاموش باشند «مرحله‌ها» از منوی اصلی حذف می‌شود.')
                ->columns(3)
                ->schema($toggles(SiteSection::stages())),
            Section::make('صفحه‌ها')
                ->description('صفحه‌ی اصلی، حریم خصوصی و شرایط استفاده همیشه روشن‌اند.')
                ->columns(3)
                ->schema($toggles(SiteSection::pages())),
        ];
    }
}
