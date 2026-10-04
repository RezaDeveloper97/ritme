<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Settings\Enums\SettingGroup;
use BackedEnum;
use Filament\Forms\Components\TextInput;
use Filament\Support\Icons\Heroicon;

final class SocialSettingsPage extends SettingsPage
{
    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedShare;

    protected static ?string $navigationLabel = 'شبکه‌های اجتماعی';

    protected static ?string $title = 'شبکه‌های اجتماعی';

    protected static ?string $slug = 'settings/social';

    protected static ?int $navigationSort = 30;

    protected static function group(): SettingGroup
    {
        return SettingGroup::Social;
    }

    protected function fields(): array
    {
        return [
            TextInput::make('instagram')->label('اینستاگرام')->url()->maxLength(255),
            TextInput::make('telegram')->label('تلگرام')->url()->maxLength(255),
            TextInput::make('linkedin')->label('لینکدین')->url()->maxLength(255),
        ];
    }
}
