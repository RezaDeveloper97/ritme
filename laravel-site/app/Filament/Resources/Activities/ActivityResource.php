<?php

declare(strict_types=1);

namespace App\Filament\Resources\Activities;

use App\Filament\Resources\Activities\Pages\ListActivities;
use App\Filament\Resources\Activities\Pages\ViewActivity;
use BackedEnum;
use Filament\Actions\ViewAction;
use Filament\Infolists\Components\KeyValueEntry;
use Filament\Infolists\Components\TextEntry;
use Filament\Resources\Resource;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Spatie\Activitylog\Models\Activity;
use UnitEnum;

/**
 * Read-only audit trail of admin mutations (spatie/laravel-activitylog). Access: ActivityPolicy.
 */
final class ActivityResource extends Resource
{
    protected static ?string $model = Activity::class;

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedClipboardDocumentList;

    protected static string|UnitEnum|null $navigationGroup = 'سیستم';

    protected static ?int $navigationSort = 90;

    protected static ?string $modelLabel = 'رویداد';

    protected static ?string $pluralModelLabel = 'گزارش فعالیت‌ها';

    protected static ?string $slug = 'activity';

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with('causer'))
            ->defaultSort('id', 'desc')
            ->columns([
                TextColumn::make('created_at')->label('زمان')->dateTime('Y-m-d H:i')->sortable(),
                TextColumn::make('causer.name')->label('کاربر')->placeholder('سیستم'),
                TextColumn::make('event')->label('رویداد')->badge()->placeholder('—'),
                TextColumn::make('description')->label('شرح')->searchable()->limit(60),
                TextColumn::make('subject_type')->label('موضوع')
                    ->formatStateUsing(static fn (?string $state): string => $state === null ? '—' : class_basename($state)),
                TextColumn::make('subject_id')->label('شناسه'),
            ])
            ->filters([
                SelectFilter::make('log_name')->label('دسته')->options(['admin' => 'مدیریت', 'settings' => 'تنظیمات']),
                SelectFilter::make('event')->label('رویداد')
                    ->options(['created' => 'ایجاد', 'updated' => 'ویرایش', 'deleted' => 'حذف']),
            ])
            ->recordActions([ViewAction::make()])
            ->toolbarActions([]);
    }

    public static function infolist(Schema $schema): Schema
    {
        return $schema->components([
            TextEntry::make('created_at')->label('زمان')->dateTime('Y-m-d H:i:s'),
            TextEntry::make('causer.name')->label('کاربر')->placeholder('سیستم'),
            TextEntry::make('log_name')->label('دسته'),
            TextEntry::make('event')->label('رویداد')->placeholder('—'),
            TextEntry::make('description')->label('شرح'),
            TextEntry::make('subject_type')->label('موضوع')->placeholder('—'),
            TextEntry::make('subject_id')->label('شناسه')->placeholder('—'),
            KeyValueEntry::make('properties.attributes')->label('مقادیر جدید')->columnSpanFull(),
            KeyValueEntry::make('properties.old')->label('مقادیر قبلی')->columnSpanFull(),
        ]);
    }

    public static function canCreate(): bool
    {
        return false;
    }

    public static function canEdit(Model $record): bool
    {
        return false;
    }

    public static function canDelete(Model $record): bool
    {
        return false;
    }

    public static function getPages(): array
    {
        return [
            'index' => ListActivities::route('/'),
            'view' => ViewActivity::route('/{record}'),
        ];
    }
}
