<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Categories;

use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use App\Filament\Resources\Directory\Categories\Pages\CreatePlaceCategory;
use App\Filament\Resources\Directory\Categories\Pages\EditPlaceCategory;
use App\Filament\Resources\Directory\Categories\Pages\ListPlaceCategories;
use App\Filament\Resources\Directory\DirectoryAdmin;
use BackedEnum;
use Filament\Actions\EditAction;
use Filament\Forms\Components\Select;
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
use Illuminate\Database\Eloquent\Model;
use UnitEnum;

/**
 * Directory categories with their schema.org LocalBusiness subtype (JSON-LD of every place in the category) and a
 * sprite icon. A category that still has places cannot be deleted. Access: DirectoryPolicy.
 */
final class PlaceCategoryResource extends Resource
{
    protected static ?string $model = PlaceCategory::class;

    protected static ?string $slug = 'directory/categories';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedSquares2x2;

    protected static string|UnitEnum|null $navigationGroup = DirectoryAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 20;

    protected static ?string $modelLabel = 'دسته مجموعه';

    protected static ?string $pluralModelLabel = 'دسته‌های مجموعه';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Grid::make(2)->schema([
                TextInput::make('name')->label('نام')->required()->maxLength(191),
                TextInput::make('slug')->label('نامک (slug)')->maxLength(191)->unique(ignoreRecord: true)
                    ->helperText('خالی بماند، از نام ساخته می‌شود. در نشانی صفحه‌های شهر × دسته به کار می‌رود.'),
                Select::make('schema_type')->label('نوع schema.org')->required()->native(false)
                    ->options(self::schemaTypes())
                    ->default(LocalBusinessType::LocalBusiness->value)
                    ->helperText('دقیق‌ترین زیرنوع LocalBusiness را انتخاب کنید (مثلاً مهدکودک ← ChildCare).'),
                DirectoryAdmin::iconSelect(),
                TextInput::make('sort_order')->label('ترتیب')->integer()->minValue(0)->maxValue(65535)->default(0),
                Toggle::make('is_active')->label('فعال')->default(true),
            ]),
            Textarea::make('description')->label('توضیح')->rows(3),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->defaultSort('sort_order')
            ->reorderable('sort_order')
            ->columns([
                TextColumn::make('name')->label('نام')->searchable(),
                TextColumn::make('slug')->label('نامک'),
                TextColumn::make('schema_type')->label('schema.org')
                    ->formatStateUsing(static fn (LocalBusinessType $state): string => $state->value),
                IconColumn::make('is_active')->label('فعال')->boolean(),
            ])
            ->recordActions([EditAction::make()]);
    }

    public static function inUse(Model $record): bool
    {
        return Place::query()->where('category_id', $record->getKey())->exists();
    }

    /**
     * @return array<string, string>
     */
    private static function schemaTypes(): array
    {
        $options = [];
        foreach (LocalBusinessType::cases() as $type) {
            $options[$type->value] = $type->value;
        }

        return $options;
    }

    public static function getPages(): array
    {
        return [
            'index' => ListPlaceCategories::route('/'),
            'create' => CreatePlaceCategory::route('/create'),
            'edit' => EditPlaceCategory::route('/{record}/edit'),
        ];
    }
}
