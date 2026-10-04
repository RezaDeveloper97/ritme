<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Amenities;

use App\Domain\Directory\Models\Amenity;
use App\Filament\Resources\Directory\Amenities\Pages\CreateAmenity;
use App\Filament\Resources\Directory\Amenities\Pages\EditAmenity;
use App\Filament\Resources\Directory\Amenities\Pages\ListAmenities;
use App\Filament\Resources\Directory\DirectoryAdmin;
use BackedEnum;
use Filament\Actions\EditAction;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Model;
use UnitEnum;

/**
 * Amenities (پارکینگ، اتاق شیردهی، …) with a sprite icon; `is_filter` shows them in the listing's filter panel.
 * Deleting one detaches it from every place. Access: DirectoryPolicy.
 */
final class AmenityResource extends Resource
{
    protected static ?string $model = Amenity::class;

    protected static ?string $slug = 'directory/amenities';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedSparkles;

    protected static string|UnitEnum|null $navigationGroup = DirectoryAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 40;

    protected static ?string $modelLabel = 'امکان';

    protected static ?string $pluralModelLabel = 'امکانات';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Grid::make(2)->schema([
                TextInput::make('name')->label('نام')->required()->maxLength(191),
                TextInput::make('slug')->label('نامک (slug)')->maxLength(191)->unique(ignoreRecord: true)
                    ->helperText('خالی بماند، از نام ساخته می‌شود؛ در پارامتر فیلتر فهرست به کار می‌رود.'),
                DirectoryAdmin::iconSelect(),
                TextInput::make('sort_order')->label('ترتیب')->integer()->minValue(0)->maxValue(65535)->default(0),
                Toggle::make('is_filter')->label('در فیلترهای فهرست نمایش داده شود')->default(false),
            ]),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->defaultSort('sort_order')
            ->reorderable('sort_order')
            ->columns([
                TextColumn::make('name')->label('نام')->searchable(),
                TextColumn::make('icon')->label('آیکن')->placeholder('—'),
                IconColumn::make('is_filter')->label('فیلتر')->boolean(),
            ])
            ->recordActions([EditAction::make()]);
    }

    public static function inUse(Model $record): bool
    {
        return false;
    }

    public static function getPages(): array
    {
        return [
            'index' => ListAmenities::route('/'),
            'create' => CreateAmenity::route('/create'),
            'edit' => EditAmenity::route('/{record}/edit'),
        ];
    }
}
