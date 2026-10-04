<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Settings\Enums\SettingGroup;
use BackedEnum;
use Filament\Forms\Components\TextInput;
use Filament\Support\Icons\Heroicon;

final class AppLinksSettingsPage extends SettingsPage
{
    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedDevicePhoneMobile;

    protected static ?string $navigationLabel = 'لینک‌های اپ';

    protected static ?string $title = 'لینک‌های دانلود اپ';

    protected static ?string $slug = 'settings/app-links';

    protected static ?int $navigationSort = 40;

    protected static function group(): SettingGroup
    {
        return SettingGroup::AppLinks;
    }

    protected function fields(): array
    {
        return [
            TextInput::make('bazaar')->label('کافه‌بازار')->url()->maxLength(255),
            TextInput::make('myket')->label('مایکت')->url()->maxLength(255),
            TextInput::make('google_play')->label('گوگل‌پلی')->url()->maxLength(255),
            TextInput::make('app_store')->label('اپ‌استور')->url()->maxLength(255),
            TextInput::make('web_app')->label('نسخه وب')->url()->maxLength(255),
        ];
    }
}
