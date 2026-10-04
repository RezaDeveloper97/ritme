<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Categories;

use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Support\BlogUrls;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Resources\Blog\BlogPolicy;
use App\Filament\Resources\Blog\Categories\Pages\CreateCategory;
use App\Filament\Resources\Blog\Categories\Pages\EditCategory;
use App\Filament\Resources\Blog\Categories\Pages\ListCategories;
use App\Filament\Resources\Blog\Posts\PostResource;
use BackedEnum;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\EditAction;
use Filament\Facades\Filament;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Tabs;
use Filament\Schemas\Components\Tabs\Tab;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use UnitEnum;

/**
 * Magazine categories (tree-light: one optional parent) with the SEO tab. Access: CategoryPolicy.
 */
final class CategoryResource extends Resource
{
    protected static ?string $model = Category::class;

    protected static ?string $slug = 'blog/categories';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedFolder;

    protected static string|UnitEnum|null $navigationGroup = 'مجله';

    protected static ?int $navigationSort = 20;

    protected static ?string $modelLabel = 'دسته';

    protected static ?string $pluralModelLabel = 'دسته‌ها';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        $locked = static fn (): bool => ! BlogPolicy::canEditContent(Filament::auth()->user());

        return $schema->columns(1)->components([
            Tabs::make('category')->tabs([
                Tab::make('اطلاعات')->disabled($locked)->schema([
                    Grid::make(2)->schema([
                        TextInput::make('name')->label('نام')->required()->maxLength(191)->live(onBlur: true),
                        TextInput::make('slug')->label('نامک (slug)')->maxLength(191)->live(onBlur: true)
                            ->unique(ignoreRecord: true)
                            ->helperText('خالی بماند، از نام ساخته می‌شود.'),
                        TextInput::make('label')->label('برچسب کوتاه روی کارت')->maxLength(64),
                        Select::make('parent_id')->label('دسته والد')
                            ->relationship('parent', 'name', static fn (Builder $query, ?Category $record): Builder => $record === null ? $query : $query->whereKeyNot($record->getKey()))
                            ->searchable()->preload(),
                        Select::make('life_stage')->label('مرحله زندگی')->options(PostResource::lifeStageOptions())->placeholder('—'),
                        TextInput::make('sort_order')->label('ترتیب')->numeric()->minValue(0)->maxValue(65535)->default(0),
                    ]),
                    Textarea::make('description')->label('توضیح')->rows(3)->live(onBlur: true),
                ]),
                Tab::make('سئو')->icon(Heroicon::OutlinedMagnifyingGlass)->schema([
                    SeoFields::make()
                        ->titleFrom('name')
                        ->descriptionFrom('description')
                        ->urlUsing(static fn (Get $get): string => app(BlogUrls::class)->category(trim((string) $get('slug')) ?: 'slug')),
                ]),
            ]),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with('parent')->withCount('posts'))
            ->defaultSort('sort_order')
            ->reorderable('sort_order')
            ->columns([
                TextColumn::make('name')->label('نام')->searchable(),
                TextColumn::make('parent.name')->label('والد')->placeholder('—'),
                TextColumn::make('slug')->label('نامک'),
                TextColumn::make('posts_count')->label('مقاله‌ها')->numeric(),
            ])
            ->recordActions([EditAction::make()])
            ->toolbarActions([DeleteBulkAction::make()]);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListCategories::route('/'),
            'create' => CreateCategory::route('/create'),
            'edit' => EditCategory::route('/{record}/edit'),
        ];
    }
}
