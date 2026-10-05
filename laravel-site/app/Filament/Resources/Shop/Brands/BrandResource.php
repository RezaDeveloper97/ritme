<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Brands;

use App\Domain\Shop\Catalog\Models\Brand;
use App\Filament\Forms\Components\MediaPicker;
use App\Filament\Resources\Shop\Brands\Pages\CreateBrand;
use App\Filament\Resources\Shop\Brands\Pages\EditBrand;
use App\Filament\Resources\Shop\Brands\Pages\ListBrands;
use App\Filament\Resources\Shop\ShopAdmin;
use BackedEnum;
use Filament\Actions\EditAction;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use UnitEnum;

/**
 * Product brands (shown on product cards — single-seller shop). A brand with products cannot be deleted.
 * Access: ShopPolicy (shop managers + super-admins).
 */
final class BrandResource extends Resource
{
    protected static ?string $model = Brand::class;

    protected static ?string $slug = 'shop/brands';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedTag;

    protected static string|UnitEnum|null $navigationGroup = ShopAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 40;

    protected static ?string $modelLabel = 'برند';

    protected static ?string $pluralModelLabel = 'برندها';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Grid::make(2)->schema([
                TextInput::make('name')->label('نام')->required()->maxLength(120),
                TextInput::make('slug')->label('نامک (slug)')->maxLength(191)->unique(ignoreRecord: true)
                    ->helperText('خالی بماند، از نام ساخته می‌شود.'),
                MediaPicker::make('logo_media_id')->label('لوگو'),
                TextInput::make('sort_order')->label('ترتیب')->integer()->minValue(0)->maxValue(65535)->default(0),
                Toggle::make('is_active')->label('فعال')->default(true),
            ]),
            Textarea::make('description')->label('توضیح')->rows(3)->maxLength(2000),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->withCount('products'))
            ->defaultSort('sort_order')
            ->reorderable('sort_order')
            ->columns([
                TextColumn::make('name')->label('نام')->searchable(),
                TextColumn::make('slug')->label('نامک')->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('products_count')->label('محصولات')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                IconColumn::make('is_active')->label('فعال')->boolean(),
            ])
            ->recordActions([EditAction::make()]);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListBrands::route('/'),
            'create' => CreateBrand::route('/create'),
            'edit' => EditBrand::route('/{record}/edit'),
        ];
    }
}
