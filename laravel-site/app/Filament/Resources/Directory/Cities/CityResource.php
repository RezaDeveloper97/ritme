<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Cities;

use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\Place;
use App\Filament\Resources\Directory\Cities\Pages\CreateCity;
use App\Filament\Resources\Directory\Cities\Pages\EditCity;
use App\Filament\Resources\Directory\Cities\Pages\ListCities;
use App\Filament\Resources\Directory\DirectoryAdmin;
use BackedEnum;
use Filament\Actions\EditAction;
use Filament\Forms\Components\Repeater;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use UnitEnum;

/**
 * Cities (`/directory/{city}` landings) with their districts (repeater, ordered). City slugs never shadow the fixed
 * `/directory/*` routes (TaxonomyObserver). A city that still has places cannot be deleted. Access: DirectoryPolicy.
 */
final class CityResource extends Resource
{
    protected static ?string $model = City::class;

    protected static ?string $slug = 'directory/cities';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedMap;

    protected static string|UnitEnum|null $navigationGroup = DirectoryAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 30;

    protected static ?string $modelLabel = 'شهر';

    protected static ?string $pluralModelLabel = 'شهرها و محله‌ها';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Grid::make(2)->schema([
                TextInput::make('name')->label('نام شهر')->required()->maxLength(191),
                TextInput::make('slug')->label('نامک (slug)')->maxLength(191)->unique(ignoreRecord: true)
                    ->helperText('خالی بماند، از نام ساخته می‌شود؛ نشانی صفحه شهر: /directory/{slug}'),
                TextInput::make('province')->label('استان')->maxLength(191),
                TextInput::make('sort_order')->label('ترتیب')->integer()->minValue(0)->maxValue(65535)->default(0),
                Toggle::make('is_active')->label('فعال')->default(true),
            ]),
            Section::make('محله‌ها / مناطق')->schema([
                Repeater::make('districts')
                    ->hiddenLabel()
                    ->relationship()
                    ->orderColumn('sort_order')
                    ->defaultItems(0)
                    ->addActionLabel('محله جدید')
                    ->columns(2)
                    ->schema([
                        TextInput::make('name')->label('نام')->required()->maxLength(191),
                        TextInput::make('slug')->label('نامک')->maxLength(191)->helperText('خالی بماند، از نام ساخته می‌شود.'),
                    ]),
            ]),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->withCount('districts'))
            ->defaultSort('sort_order')
            ->reorderable('sort_order')
            ->columns([
                TextColumn::make('name')->label('نام')->searchable(),
                TextColumn::make('province')->label('استان')->placeholder('—'),
                TextColumn::make('slug')->label('نامک'),
                TextColumn::make('districts_count')->label('محله‌ها')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                IconColumn::make('is_active')->label('فعال')->boolean(),
            ])
            ->recordActions([EditAction::make()]);
    }

    public static function inUse(Model $record): bool
    {
        return Place::query()->where('city_id', $record->getKey())->exists();
    }

    public static function getPages(): array
    {
        return [
            'index' => ListCities::route('/'),
            'create' => CreateCity::route('/create'),
            'edit' => EditCity::route('/{record}/edit'),
        ];
    }
}
