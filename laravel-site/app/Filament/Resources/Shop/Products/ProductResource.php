<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Products;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use App\Domain\Shop\Ordering\Models\OrderItem;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Forms\Components\MediaPicker;
use App\Filament\Resources\Shop\LowStock;
use App\Filament\Resources\Shop\Products\Pages\CreateProduct;
use App\Filament\Resources\Shop\Products\Pages\EditProduct;
use App\Filament\Resources\Shop\Products\Pages\ListProducts;
use App\Filament\Resources\Shop\ShopAdmin;
use BackedEnum;
use Filament\Actions\EditAction;
use Filament\Forms\Components\CheckboxList;
use Filament\Forms\Components\Hidden;
use Filament\Forms\Components\Radio;
use Filament\Forms\Components\Repeater;
use Filament\Forms\Components\RichEditor;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TagsInput;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Infolists\Components\TextEntry;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Components\Tabs;
use Filament\Schemas\Components\Tabs\Tab;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\Filter;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Filters\TernaryFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use UnitEnum;

/**
 * Shop products (L6-06): tabs general / pricing (tomans) / inventory + variants / specs + size chart / gallery /
 * categories / SEO. Writes go through SaveProduct (attributes, variants, categories, gallery, cross-sells in one
 * transaction; publish and stock changes logged). Ratings and sales are computed elsewhere and only shown. Access:
 * ShopCatalogPolicy — shop managers + super-admins write; SEO managers edit the SEO tab only.
 */
final class ProductResource extends Resource
{
    protected static ?string $model = Product::class;

    protected static ?string $slug = 'shop/products';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedShoppingBag;

    protected static string|UnitEnum|null $navigationGroup = ShopAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 20;

    protected static ?string $modelLabel = 'محصول';

    protected static ?string $pluralModelLabel = 'محصولات';

    protected static ?string $recordTitleAttribute = 'title';

    public static function form(Schema $schema): Schema
    {
        $locked = static fn (): bool => ! ShopAdmin::canEditContent();

        return $schema->columns(1)->components([
            Tabs::make('product')->persistTabInQueryString()->tabs([
                Tab::make('عمومی')->icon(Heroicon::OutlinedInformationCircle)->disabled($locked)->schema(self::generalFields()),
                Tab::make('قیمت')->icon(Heroicon::OutlinedBanknotes)->disabled($locked)->schema([
                    Grid::make(2)->schema([
                        ShopAdmin::tomanInput('price_toman', 'قیمت فروش')->required()
                            ->helperText('به تومان وارد کنید؛ برای تنوع‌هایی که قیمت جدا ندارند هم همین قیمت به کار می‌رود.'),
                        ShopAdmin::tomanInput('compare_at_toman', 'قیمت قبل از تخفیف')
                            ->helperText('فقط وقتی بیشتر از قیمت فروش باشد نمایش داده می‌شود. بدون شمارش معکوس و فشار فروش.'),
                    ]),
                ]),
                Tab::make('موجودی و تنوع‌ها')->icon(Heroicon::OutlinedArchiveBox)->disabled($locked)->schema(self::inventoryFields()),
                Tab::make('مشخصات و جدول سایز')->icon(Heroicon::OutlinedTableCells)->disabled($locked)->schema(self::specFields()),
                Tab::make('گالری')->icon(Heroicon::OutlinedPhoto)->disabled($locked)->schema([
                    MediaPicker::make('cover_media_id')->label('تصویر اصلی')->live(),
                    Repeater::make('gallery_items')
                        ->label('تصاویر گالری (به ترتیب نمایش)')
                        ->simple(MediaPicker::make('media_id')->required())
                        ->defaultItems(0)
                        ->maxItems(20)
                        ->addActionLabel('افزودن تصویر'),
                ]),
                Tab::make('دسته‌ها')->icon(Heroicon::OutlinedSquares2x2)->disabled($locked)->schema([
                    Select::make('category_ids')->label('دسته‌ها')->multiple()->searchable()->preload()->live()
                        ->options(static fn (): array => self::categoryOptions()),
                    Select::make('primary_category_id')->label('دسته اصلی (مسیر راهنما و نشانی‌ها)')->searchable()
                        ->placeholder('اولین دسته انتخاب‌شده')
                        ->options(static fn (Get $get): array => array_intersect_key(self::categoryOptions(), array_flip(array_map(intval(...), (array) $get('category_ids'))))),
                    Select::make('cross_sell_ids')->label('«معمولاً با این می‌خرند» (به ترتیب)')->multiple()->searchable()
                        ->options(static fn (?Product $record): array => Product::query()
                            ->when($record !== null, static fn (Builder $q): Builder => $q->whereKeyNot($record?->getKey()))
                            ->orderBy('title')->limit(500)->pluck('title', 'id')->all())
                        ->maxItems(8),
                ]),
                Tab::make('سئو')->icon(Heroicon::OutlinedMagnifyingGlass)->schema([
                    SeoFields::make()
                        ->titleFrom('title')
                        ->descriptionFrom('short_description')
                        ->imageFrom('cover_media_id')
                        ->analysisType(ContentType::Product)
                        ->urlUsing(static fn (Get $get): string => app(ShopUrls::class)->product(trim((string) $get('slug')) ?: 'slug')),
                ]),
            ]),
        ]);
    }

    /**
     * @return array<int, mixed>
     */
    private static function generalFields(): array
    {
        return [
            Grid::make(2)->schema([
                TextInput::make('title')->label('نام محصول')->required()->maxLength(191)->live(onBlur: true),
                TextInput::make('slug')->label('نامک (slug)')->maxLength(120)->unique(ignoreRecord: true)
                    ->helperText('خالی بماند، از نام ساخته می‌شود. تغییر نامک یک محصول منتشرشده، نشانی قبلی را ۳۰۱ می‌کند.'),
                TextInput::make('sku')->label('کد کالا (SKU)')->maxLength(64)->unique(ignoreRecord: true)->extraInputAttributes(['dir' => 'ltr']),
                Select::make('brand_id')->label('برند')->searchable()->preload()->placeholder('—')
                    ->options(static fn (): array => Brand::query()->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all()),
                TextInput::make('badge')->label('برچسب روی کارت')->maxLength(40)->placeholder('جدید'),
                TextInput::make('sort_order')->label('ترتیب')->integer()->minValue(0)->maxValue(1_000_000)->default(0),
            ]),
            Textarea::make('short_description')->label('توضیح کوتاه (کارت و توضیح متا)')->rows(2)->maxLength(500)->live(onBlur: true),
            RichEditor::make('description')->label('توضیحات کامل')
                ->toolbarButtons([['bold', 'italic', 'link'], ['h2', 'h3'], ['bulletList', 'orderedList'], ['undo', 'redo']])
                ->fileAttachments(false),
            CheckboxList::make('life_stages')->label('مرحله زندگی')->columns(3)
                ->options(static function (): array {
                    $options = [];
                    foreach (LifeStage::cases() as $stage) {
                        $options[$stage->value] = $stage->label();
                    }

                    return $options;
                }),
            Grid::make(3)->schema([
                Toggle::make('is_published')->label('منتشر شود')->default(false),
                Toggle::make('is_featured')->label('ویژه')->default(false),
                Toggle::make('is_demo')->label('محصول نمونه (نمایشی)')->default(false)
                    ->helperText('محصولات نمونه noindex می‌مانند و در امتیاز و نقشه سایت حساب نمی‌شوند.'),
            ]),
            Section::make('آمار (خودکار)')->collapsed()->visibleOn('edit')->schema([
                TextEntry::make('sales')->label('فروش')
                    ->state(static fn (?Product $record): string => fa_digits($record->sales_count ?? 0)),
                TextEntry::make('rating')->label('امتیاز (نظرهای تأییدشده واقعی)')
                    ->state(static fn (?Product $record): string => $record !== null && $record->rating_count > 0
                        ? fa_digits(number_format($record->rating_avg, 1)).' از ۵ — '.fa_digits($record->rating_count).' نظر'
                        : 'هنوز نظری ثبت نشده'),
            ])->columns(2),
        ];
    }

    /**
     * @return array<int, mixed>
     */
    private static function inventoryFields(): array
    {
        return [
            Grid::make(3)->schema([
                TextInput::make('stock_qty')->label('موجودی (بدون تنوع)')->integer()->minValue(0)->maxValue(1_000_000)->default(0)
                    ->disabled(static fn (Get $get): bool => (array) $get('variants') !== [])
                    ->helperText(static fn (Get $get): ?string => (array) $get('variants') !== [] ? 'با تنوع‌ها، موجودی جمع تنوع‌های فعال است.' : null),
                TextInput::make('weight_grams')->label('وزن (گرم)')->integer()->minValue(0)->maxValue(1_000_000),
                Radio::make('stock_mode')->label('وضعیت موجودی')->default('auto')->required()
                    ->options([
                        'auto' => 'خودکار از موجودی',
                        StockStatus::PreOrder->value => StockStatus::PreOrder->label(),
                        StockStatus::BackOrder->value => StockStatus::BackOrder->label(),
                    ]),
            ]),
            Repeater::make('variants')
                ->label('تنوع‌ها (سایز / رنگ)')
                ->defaultItems(0)
                ->maxItems(60)
                ->collapsible()
                ->live()
                ->itemLabel(static fn (array $state): ?string => trim(implode(' / ', array_filter([$state['size'] ?? null, $state['color'] ?? null], is_string(...)))) ?: null)
                ->addActionLabel('تنوع جدید')
                ->schema([
                    Hidden::make('id'),
                    Grid::make(4)->schema([
                        TextInput::make('size')->label('سایز')->maxLength(40),
                        TextInput::make('color')->label('رنگ')->maxLength(40),
                        TextInput::make('color_hex')->label('کد رنگ')->placeholder('#F4C7C3')->maxLength(7)
                            ->regex('/^#[0-9A-Fa-f]{6}$/')->validationMessages(['regex' => 'به شکل #RRGGBB بنویسید.'])
                            ->extraInputAttributes(['dir' => 'ltr']),
                        TextInput::make('sku')->label('SKU')->maxLength(64)->extraInputAttributes(['dir' => 'ltr']),
                        ShopAdmin::tomanInput('price_toman', 'قیمت جدا')->placeholder('قیمت محصول'),
                        ShopAdmin::tomanInput('compare_at_toman', 'قبل از تخفیف'),
                        TextInput::make('stock_qty')->label('موجودی')->integer()->minValue(0)->maxValue(1_000_000)->default(0)->required(),
                        Toggle::make('is_active')->label('فعال')->default(true)->inline(false),
                    ]),
                ]),
        ];
    }

    /**
     * @return array<int, mixed>
     */
    private static function specFields(): array
    {
        return [
            Repeater::make('specs')
                ->label('مشخصات')
                ->defaultItems(0)
                ->maxItems(30)
                ->addActionLabel('ردیف جدید')
                ->schema([
                    Grid::make(2)->schema([
                        TextInput::make('label')->label('عنوان')->required()->maxLength(191),
                        TextInput::make('value')->label('مقدار')->required()->maxLength(191),
                    ]),
                ]),
            Section::make('جدول سایز')->schema([
                TagsInput::make('size_chart_columns')->label('ستون‌ها')->placeholder('مثلاً سایز، سن، قد (سانتی‌متر) — هر کدام و Enter')
                    ->reorderable(),
                Repeater::make('size_chart_rows')
                    ->label('ردیف‌ها')
                    ->defaultItems(0)
                    ->maxItems(30)
                    ->addActionLabel('ردیف جدید')
                    ->schema([
                        TextInput::make('cells')->hiddenLabel()->required()->maxLength(500)
                            ->placeholder('۰-۳ ماه | ۰ تا ۳ ماه | ۵۰-۶۲')
                            ->helperText('خانه‌ها را به ترتیب ستون‌ها با | جدا کنید.'),
                    ]),
            ]),
        ];
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with(['brand:id,name', 'primaryCategory:id,name']))
            ->defaultSort('updated_at', 'desc')
            ->columns([
                TextColumn::make('title')->label('نام')->searchable()->sortable()->limit(50),
                TextColumn::make('sku')->label('SKU')->searchable()->placeholder('—')->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('primaryCategory.name')->label('دسته')->placeholder('—'),
                TextColumn::make('price')->label('قیمت')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::money($state)),
                TextColumn::make('stock_qty')->label('موجودی')->sortable()
                    ->formatStateUsing(static fn (int $state): string => fa_digits($state))
                    ->color(static fn (Product $record): ?string => $record->stock_qty === 0 ? 'danger' : null),
                TextColumn::make('stock_status')->label('وضعیت موجودی')->badge()
                    ->formatStateUsing(static fn (StockStatus $state): string => $state->label())
                    ->color(static fn (StockStatus $state): string => match ($state) {
                        StockStatus::InStock => 'success',
                        StockStatus::OutOfStock => 'danger',
                        default => 'warning',
                    }),
                IconColumn::make('is_published')->label('منتشر')->boolean(),
                IconColumn::make('is_demo')->label('نمونه')->boolean(),
                TextColumn::make('sales_count')->label('فروش')->sortable()->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                TextColumn::make('updated_at')->label('به‌روزرسانی')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::date($state)),
            ])
            ->filters([
                TernaryFilter::make('is_published')->label('منتشر'),
                SelectFilter::make('stock_status')->label('وضعیت موجودی')->options(self::stockOptions()),
                SelectFilter::make('brand_id')->label('برند')->relationship('brand', 'name'),
                SelectFilter::make('primary_category_id')->label('دسته اصلی')->relationship('primaryCategory', 'name'),
                Filter::make('low_stock')->label('موجودی کم')
                    ->query(static fn (Builder $query): Builder => $query->whereIn('shop_products.id', LowStock::products()->select('shop_products.id'))),
                TernaryFilter::make('is_demo')->label('نمونه'),
            ])
            ->recordActions([EditAction::make()]);
    }

    /**
     * @return array<string, string>
     */
    private static function stockOptions(): array
    {
        $options = [];
        foreach (StockStatus::cases() as $status) {
            $options[$status->value] = $status->label();
        }

        return $options;
    }

    /**
     * Categories with their path («لباس نوزاد › بادی»), in tree order.
     *
     * @return array<int, string>
     */
    public static function categoryOptions(): array
    {
        $categories = Category::query()->orderBy('sort_order')->orderBy('id')->get(['id', 'parent_id', 'name'])->keyBy('id');
        $options = [];
        foreach ($categories as $category) {
            $path = [$category->name];
            $seen = [$category->id => true];
            for ($parent = $categories->get((int) $category->parent_id); $parent !== null && ! isset($seen[$parent->id]); $parent = $categories->get((int) $parent->parent_id)) {
                array_unshift($path, $parent->name);
                $seen[$parent->id] = true;
            }
            $options[$category->id] = implode(' › ', $path);
        }
        asort($options);

        return $options;
    }

    public static function hasOrders(Product $product): bool
    {
        return OrderItem::query()->where('product_id', $product->id)->exists();
    }

    public static function getPages(): array
    {
        return [
            'index' => ListProducts::route('/'),
            'create' => CreateProduct::route('/create'),
            'edit' => EditProduct::route('/{record}/edit'),
        ];
    }
}
