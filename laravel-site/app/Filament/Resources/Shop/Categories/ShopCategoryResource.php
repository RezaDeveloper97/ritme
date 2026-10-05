<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Categories;

use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Forms\Components\MediaPicker;
use App\Filament\Resources\Shop\Categories\Pages\CreateShopCategory;
use App\Filament\Resources\Shop\Categories\Pages\EditShopCategory;
use App\Filament\Resources\Shop\Categories\Pages\ListShopCategories;
use App\Filament\Resources\Shop\Products\ProductResource;
use App\Filament\Resources\Shop\ShopAdmin;
use BackedEnum;
use Filament\Actions\EditAction;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Tabs;
use Filament\Schemas\Components\Tabs\Tab;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use UnitEnum;

/**
 * Shop categories (L6-06) — a tree (roots are departments): parent picker that never offers the category itself or
 * one of its subcategories, drag-sort of siblings (filter by parent, then drag), cover image and the SEO tab. A
 * category with products or subcategories cannot be deleted. Access: ShopCatalogPolicy (SEO managers: SEO tab only).
 */
final class ShopCategoryResource extends Resource
{
    protected static ?string $model = Category::class;

    protected static ?string $slug = 'shop/categories';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedSquares2x2;

    protected static string|UnitEnum|null $navigationGroup = ShopAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 30;

    protected static ?string $modelLabel = 'دسته فروشگاه';

    protected static ?string $pluralModelLabel = 'دسته‌های فروشگاه';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        $locked = static fn (): bool => ! ShopAdmin::canEditContent();

        return $schema->columns(1)->components([
            Tabs::make('category')->tabs([
                Tab::make('اطلاعات')->icon(Heroicon::OutlinedInformationCircle)->disabled($locked)->schema([
                    Grid::make(2)->schema([
                        TextInput::make('name')->label('نام')->required()->maxLength(120)->live(onBlur: true),
                        TextInput::make('slug')->label('نامک (slug)')->maxLength(191)->unique(ignoreRecord: true)
                            ->helperText('خالی بماند، از نام ساخته می‌شود. تغییر نامک، نشانی قبلی را ۳۰۱ می‌کند.'),
                        Select::make('parent_id')->label('دسته والد')->searchable()->placeholder('— (دپارتمان اصلی)')
                            ->options(static fn (?Category $record): array => self::parentOptions($record)),
                        TextInput::make('sort_order')->label('ترتیب')->integer()->minValue(0)->maxValue(65535)->default(0),
                        MediaPicker::make('cover_media_id')->label('تصویر دسته')->live(),
                        Toggle::make('is_active')->label('فعال')->default(true)->inline(false),
                    ]),
                    Textarea::make('intro')->label('متن معرفی (بالای فهرست محصولات)')->rows(3)->maxLength(2000)->live(onBlur: true),
                ]),
                Tab::make('سئو')->icon(Heroicon::OutlinedMagnifyingGlass)->schema([
                    SeoFields::make()
                        ->titleFrom('name')
                        ->descriptionFrom('intro')
                        ->imageFrom('cover_media_id')
                        ->contentFrom('intro')
                        ->analysisType(ContentType::Archive)
                        ->urlUsing(static fn (Get $get): string => app(ShopUrls::class)->category(trim((string) $get('slug')) ?: 'slug')),
                ]),
            ]),
        ]);
    }

    public static function table(Table $table): Table
    {
        $paths = null;

        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->withCount(['products', 'children']))
            ->defaultSort('sort_order')
            ->reorderable('sort_order', static fn (): bool => ShopAdmin::canEditContent())
            ->columns([
                TextColumn::make('name')->label('نام (مسیر)')->searchable()
                    ->formatStateUsing(static function (string $state, Category $record) use (&$paths): string {
                        $paths ??= ProductResource::categoryOptions();

                        return $paths[$record->id] ?? $state;
                    }),
                TextColumn::make('slug')->label('نامک')->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('products_count')->label('محصولات')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                TextColumn::make('children_count')->label('زیردسته‌ها')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                IconColumn::make('is_active')->label('فعال')->boolean(),
            ])
            ->filters([
                SelectFilter::make('parent')
                    ->label('والد (برای مرتب‌سازی هم‌سطح‌ها)')
                    ->options(static fn (): array => ['root' => '— دپارتمان‌های اصلی', ...self::parentOptions(null)])
                    ->query(static function (Builder $query, array $data): Builder {
                        $value = $data['value'] ?? null;

                        return match (true) {
                            $value === 'root' => $query->whereNull('parent_id'),
                            is_numeric($value) => $query->where('parent_id', (int) $value),
                            default => $query,
                        };
                    }),
            ])
            ->recordActions([EditAction::make()]);
    }

    /**
     * Every category with its path, except the given one and its subcategories (a parent there would make a cycle).
     *
     * @return array<int|string, string>
     */
    public static function parentOptions(?Category $record): array
    {
        $options = ProductResource::categoryOptions();
        if ($record === null) {
            return $options;
        }

        $excluded = [$record->id => true];
        $children = Category::query()->get(['id', 'parent_id'])->groupBy('parent_id');
        $queue = [$record->id];
        while ($queue !== []) {
            $id = array_shift($queue);
            foreach ($children->get($id, collect()) as $child) {
                if (! isset($excluded[$child->id])) {
                    $excluded[$child->id] = true;
                    $queue[] = $child->id;
                }
            }
        }

        return array_diff_key($options, $excluded);
    }

    public static function inUse(Category $category): bool
    {
        return $category->products()->exists() || $category->children()->exists();
    }

    public static function getPages(): array
    {
        return [
            'index' => ListShopCategories::route('/'),
            'create' => CreateShopCategory::route('/create'),
            'edit' => EditShopCategory::route('/{record}/edit'),
        ];
    }
}
