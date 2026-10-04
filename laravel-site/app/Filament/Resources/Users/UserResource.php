<?php

declare(strict_types=1);

namespace App\Filament\Resources\Users;

use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Users\Pages\CreateUser;
use App\Filament\Resources\Users\Pages\EditUser;
use App\Filament\Resources\Users\Pages\ListUsers;
use App\Models\User;
use BackedEnum;
use Filament\Actions\EditAction;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\Pages\CreateRecord;
use Filament\Resources\Resource;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Validation\Rules\Password;
use UnitEnum;

/**
 * Admin accounts (super-admin only, see UserPolicy). Passwords are never shown; leave empty on edit to keep it.
 */
final class UserResource extends Resource
{
    protected static ?string $model = User::class;

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedUsers;

    protected static string|UnitEnum|null $navigationGroup = 'سیستم';

    protected static ?int $navigationSort = 80;

    protected static ?string $modelLabel = 'کاربر مدیریت';

    protected static ?string $pluralModelLabel = 'کاربران مدیریت';

    public static function form(Schema $schema): Schema
    {
        return $schema->components([
            TextInput::make('name')->label('نام')->required()->maxLength(255),
            TextInput::make('email')->label('ایمیل')->email()->required()->maxLength(255)
                ->unique(ignoreRecord: true),
            TextInput::make('password')->label('رمز عبور')->password()->revealable()
                ->rule(Password::min(12)->letters()->numbers())
                ->required(static fn ($livewire): bool => $livewire instanceof CreateRecord)
                ->dehydrated(static fn (?string $state): bool => filled($state))
                ->helperText('حداقل ۱۲ نویسه شامل حرف و عدد. برای حفظ رمز فعلی خالی بگذارید.'),
            Select::make('roles')->label('نقش‌ها')->multiple()->preload()->required()
                ->relationship('roles', 'name')
                ->getOptionLabelFromRecordUsing(static fn ($record): string => AdminRole::tryFrom((string) $record->name)?->label() ?? (string) $record->name),
            Toggle::make('is_active')->label('فعال')->default(true),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with('roles'))
            ->columns([
                TextColumn::make('name')->label('نام')->searchable()->sortable(),
                TextColumn::make('email')->label('ایمیل')->searchable(),
                TextColumn::make('roles.name')->label('نقش‌ها')->badge()
                    ->formatStateUsing(static fn (string $state): string => AdminRole::tryFrom($state)?->label() ?? $state),
                IconColumn::make('is_active')->label('فعال')->boolean(),
                TextColumn::make('last_login_at')->label('آخرین ورود')->dateTime('Y-m-d H:i')->placeholder('—')->sortable(),
            ])
            ->recordActions([EditAction::make()]);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListUsers::route('/'),
            'create' => CreateUser::route('/create'),
            'edit' => EditUser::route('/{record}/edit'),
        ];
    }
}
