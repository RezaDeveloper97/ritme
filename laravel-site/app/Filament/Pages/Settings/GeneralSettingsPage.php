<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Settings\Enums\SettingGroup;
use BackedEnum;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Support\Icons\Heroicon;

final class GeneralSettingsPage extends SettingsPage
{
    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedCog6Tooth;

    protected static ?string $navigationLabel = 'عمومی';

    protected static ?string $title = 'تنظیمات عمومی';

    protected static ?string $slug = 'settings/general';

    protected static ?int $navigationSort = 10;

    protected static function group(): SettingGroup
    {
        return SettingGroup::General;
    }

    protected function fields(): array
    {
        return [
            TextInput::make('site_name')->label('نام سایت')->required()->maxLength(80),
            TextInput::make('alternate_name')->label('نام جایگزین (لاتین)')->maxLength(80),
            TextInput::make('tagline')->label('شعار')->maxLength(160),
            Textarea::make('footer_note')->label('یادداشت فوتر')->rows(3)->maxLength(500),
            TextInput::make('emergency_number')->label('شماره اورژانس')->maxLength(20),
        ];
    }
}
