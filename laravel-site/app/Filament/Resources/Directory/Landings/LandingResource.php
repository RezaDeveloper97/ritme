<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Landings;

use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\Landing;
use App\Domain\Directory\Models\PlaceCategory;
use App\Filament\Resources\Directory\DirectoryAdmin;
use App\Filament\Resources\Directory\Landings\Pages\CreateLanding;
use App\Filament\Resources\Directory\Landings\Pages\EditLanding;
use App\Filament\Resources\Directory\Landings\Pages\ListLandings;
use BackedEnum;
use Closure;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\EditAction;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use UnitEnum;

/**
 * Landing texts per city (`/directory/{city}`) or city × category (`/directory/{city}/{category}`): title, meta
 * description, h1 and intro. Empty fields fall back to the generated copy (LandingCopy). Access: DirectoryPolicy.
 */
final class LandingResource extends Resource
{
    protected static ?string $model = Landing::class;

    protected static ?string $slug = 'directory/landings';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedDocumentText;

    protected static string|UnitEnum|null $navigationGroup = DirectoryAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 50;

    protected static ?string $modelLabel = 'متن صفحه شهر / دسته';

    protected static ?string $pluralModelLabel = 'متن صفحه‌های شهر × دسته';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Grid::make(2)->schema([
                Select::make('city_id')->label('شهر')->required()->searchable()->preload()->live()
                    ->options(static fn (): array => City::query()->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all())
                    ->rule(static fn (Get $get, ?Landing $record): Closure => static function (string $attribute, mixed $value, Closure $fail) use ($get, $record): void {
                        $taken = Landing::query()
                            ->where('city_id', (int) $value)
                            ->when(is_numeric($get('category_id')), static fn (Builder $q) => $q->where('category_id', (int) $get('category_id')), static fn (Builder $q) => $q->whereNull('category_id'))
                            ->when($record !== null, static fn (Builder $q) => $q->whereKeyNot($record?->getKey()))
                            ->exists();
                        if ($taken) {
                            $fail('برای این شهر و دسته قبلاً متن ثبت شده است.');
                        }
                    }),
                Select::make('category_id')->label('دسته')->searchable()->preload()
                    ->placeholder('همه دسته‌ها (صفحه شهر)')
                    ->options(static fn (): array => PlaceCategory::query()->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all()),
            ]),
            TextInput::make('meta_title')->label('عنوان صفحه (title)')->maxLength(191)->helperText('خالی بماند، عنوان پیش‌فرض ساخته می‌شود.'),
            Textarea::make('meta_description')->label('توضیح متا (description)')->rows(2)->maxLength(320),
            TextInput::make('h1')->label('تیتر اصلی (h1)')->maxLength(191),
            Textarea::make('intro')->label('متن معرفی بالای فهرست')->rows(5),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with(['city', 'category']))
            ->defaultSort('id', 'desc')
            ->columns([
                TextColumn::make('city.name')->label('شهر'),
                TextColumn::make('category.name')->label('دسته')->placeholder('صفحه شهر'),
                TextColumn::make('h1')->label('h1')->placeholder('پیش‌فرض')->limit(50),
                TextColumn::make('updated_at')->label('به‌روزرسانی')->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
            ])
            ->filters([
                SelectFilter::make('city_id')->label('شهر')->relationship('city', 'name'),
            ])
            ->recordActions([EditAction::make()])
            ->toolbarActions([DeleteBulkAction::make()]);
    }

    public static function inUse(Model $record): bool
    {
        return false;
    }

    public static function getPages(): array
    {
        return [
            'index' => ListLandings::route('/'),
            'create' => CreateLanding::route('/create'),
            'edit' => EditLanding::route('/{record}/edit'),
        ];
    }
}
