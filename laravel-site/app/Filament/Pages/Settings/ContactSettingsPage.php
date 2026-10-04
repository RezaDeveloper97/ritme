<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Settings\Enums\SettingGroup;
use BackedEnum;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Support\Icons\Heroicon;

final class ContactSettingsPage extends SettingsPage
{
    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedEnvelope;

    protected static ?string $navigationLabel = 'تماس';

    protected static ?string $title = 'اطلاعات تماس';

    protected static ?string $slug = 'settings/contact';

    protected static ?int $navigationSort = 20;

    protected static function group(): SettingGroup
    {
        return SettingGroup::Contact;
    }

    protected function fields(): array
    {
        return [
            TextInput::make('support_email')->label('ایمیل پشتیبانی')->email()->maxLength(255),
            TextInput::make('partnership_email')->label('ایمیل همکاری')->email()->maxLength(255),
            TextInput::make('phone')->label('تلفن')->tel()->maxLength(30),
            TextInput::make('working_hours')->label('ساعات پاسخ‌گویی')->maxLength(120),
            TextInput::make('response_time')->label('زمان پاسخ')->maxLength(120),
            Textarea::make('address')->label('نشانی')->rows(2)->maxLength(500),
            TextInput::make('address_note')->label('توضیح نشانی')->maxLength(255),
        ];
    }
}
