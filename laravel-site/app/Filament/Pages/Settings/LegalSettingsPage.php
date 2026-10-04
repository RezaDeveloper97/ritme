<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Settings\Enums\SettingGroup;
use BackedEnum;
use Filament\Forms\Components\TagsInput;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Support\Icons\Heroicon;

/**
 * Legal settings are super-admin only (the enamad snippet is rendered as HTML after tag filtering).
 */
final class LegalSettingsPage extends SettingsPage
{
    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedScale;

    protected static ?string $navigationLabel = 'حقوقی';

    protected static ?string $title = 'تنظیمات حقوقی';

    protected static ?string $slug = 'settings/legal';

    protected static ?int $navigationSort = 50;

    protected static array $roles = [];

    protected static function group(): SettingGroup
    {
        return SettingGroup::Legal;
    }

    protected function fields(): array
    {
        return [
            TextInput::make('enamad_code')->label('کد اینماد')->maxLength(64),
            Textarea::make('enamad_html')->label('کد HTML نشان اینماد')->rows(4)->maxLength(4000)
                ->helperText('فقط برچسب‌های مجاز زیر نگه داشته می‌شوند.'),
            TagsInput::make('enamad_allowed_tags')->label('برچسب‌های مجاز HTML'),
            TextInput::make('data_protection_email')->label('ایمیل حریم خصوصی')->email()->maxLength(255),
        ];
    }
}
