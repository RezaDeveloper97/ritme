<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Places;

use App\Domain\Directory\Actions\ChangePlaceStatus;
use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Support\DirectoryUrls;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Forms\Components\MediaPicker;
use App\Filament\Resources\Directory\DirectoryAdmin;
use App\Filament\Resources\Directory\Places\Pages\CreatePlace;
use App\Filament\Resources\Directory\Places\Pages\EditPlace;
use App\Filament\Resources\Directory\Places\Pages\ListPlaces;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Actions\EditAction;
use Filament\Forms\Components\CheckboxList;
use Filament\Forms\Components\Radio;
use Filament\Forms\Components\Repeater;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TagsInput;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Tabs;
use Filament\Schemas\Components\Tabs\Tab;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Components\Utilities\Set;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Filters\TernaryFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use UnitEnum;

/**
 * Directory places (L5-06): tabs info / location / services & prices / opening hours / gallery / SEO. Writes go
 * through SavePlace (attributes + amenities + ordered gallery) and the services repeater (PlaceServiceObserver keeps
 * `price_from`); status only changes through the publish / draft / suspend actions (ChangePlaceStatus, activity log).
 * Access: DirectoryPolicy (directory-manager + super-admin).
 */
final class PlaceResource extends Resource
{
    protected static ?string $model = Place::class;

    protected static ?string $slug = 'directory/places';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedBuildingStorefront;

    protected static string|UnitEnum|null $navigationGroup = DirectoryAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 10;

    protected static ?string $modelLabel = 'مجموعه';

    protected static ?string $pluralModelLabel = 'مجموعه‌ها';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Tabs::make('place')->persistTabInQueryString()->tabs([
                Tab::make('اطلاعات')->icon(Heroicon::OutlinedInformationCircle)->schema(self::infoFields()),
                Tab::make('مکان')->icon(Heroicon::OutlinedMapPin)->schema([
                    Textarea::make('address')->label('نشانی')->rows(2)->maxLength(500),
                    Grid::make(3)->schema([
                        TextInput::make('postal_code')->label('کد پستی')->maxLength(20)->extraInputAttributes(['dir' => 'ltr']),
                        TextInput::make('latitude')->label('عرض جغرافیایی')->numeric()->minValue(-90)->maxValue(90)->extraInputAttributes(['dir' => 'ltr']),
                        TextInput::make('longitude')->label('طول جغرافیایی')->numeric()->minValue(-180)->maxValue(180)->extraInputAttributes(['dir' => 'ltr']),
                    ]),
                ]),
                Tab::make('خدمات و قیمت‌ها')->icon(Heroicon::OutlinedCurrencyDollar)->schema([
                    Repeater::make('services')
                        ->hiddenLabel()
                        ->relationship()
                        ->orderColumn('sort_order')
                        ->defaultItems(0)
                        ->collapsible()
                        ->itemLabel(static fn (array $state): ?string => is_string($state['name'] ?? null) ? $state['name'] : null)
                        ->addActionLabel('خدمت جدید')
                        ->schema([
                            Grid::make(3)->schema([
                                TextInput::make('name')->label('نام خدمت')->required()->maxLength(191)->columnSpan(2),
                                TextInput::make('duration_minutes')->label('مدت (دقیقه)')->integer()->minValue(1)->maxValue(1440),
                                TextInput::make('price')->label('قیمت (تومان)')->integer()->minValue(0)->maxValue(4_000_000_000)->extraInputAttributes(['dir' => 'ltr']),
                                TextInput::make('price_unit')->label('واحد قیمت')->placeholder('هر جلسه')->maxLength(64),
                                TextInput::make('details')->label('جزئیات')->placeholder('گروهی، اعتبار ۶۰ روز')->maxLength(191),
                                TextInput::make('age_min_months')->label('حداقل سن (ماه)')->integer()->minValue(0)->maxValue(240),
                                TextInput::make('age_max_months')->label('حداکثر سن (ماه)')->integer()->minValue(0)->maxValue(240),
                            ]),
                        ]),
                ]),
                Tab::make('ساعات کاری')->icon(Heroicon::OutlinedClock)->schema([
                    DirectoryAdmin::hoursEditor(),
                ]),
                Tab::make('گالری')->icon(Heroicon::OutlinedPhoto)->schema([
                    MediaPicker::make('cover_media_id')->label('تصویر اصلی (کاور)')->live(),
                    Repeater::make('gallery_items')
                        ->label('تصاویر گالری (به ترتیب نمایش)')
                        ->simple(MediaPicker::make('media_id')->required())
                        ->defaultItems(0)
                        ->maxItems(30)
                        ->addActionLabel('افزودن تصویر'),
                ]),
                Tab::make('سئو')->icon(Heroicon::OutlinedMagnifyingGlass)->schema([
                    SeoFields::make()
                        ->titleFrom('name')
                        ->descriptionFrom('summary')
                        ->imageFrom('cover_media_id')
                        ->urlUsing(static fn (Get $get): string => app(DirectoryUrls::class)->place(trim((string) $get('slug')) ?: 'slug')),
                ]),
            ]),
        ]);
    }

    /**
     * @return array<int, mixed>
     */
    private static function infoFields(): array
    {
        return [
            Grid::make(2)->schema([
                TextInput::make('name')->label('نام مجموعه')->required()->maxLength(191)->live(onBlur: true),
                TextInput::make('slug')->label('نامک (slug)')->maxLength(120)
                    ->unique(ignoreRecord: true)
                    ->helperText('خالی بماند، از نام ساخته می‌شود. تغییر نامک یک مجموعه منتشرشده، نشانی قبلی را ۳۰۱ می‌کند.'),
                Select::make('category_id')->label('دسته')->required()->searchable()->preload()
                    ->options(static fn (): array => PlaceCategory::query()->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all()),
                Select::make('city_id')->label('شهر')->required()->searchable()->preload()->live()
                    ->options(static fn (): array => City::query()->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all())
                    ->afterStateUpdated(static fn (Set $set): mixed => $set('district_id', null)),
                Select::make('district_id')->label('محله / منطقه')->searchable()->placeholder('—')
                    ->options(static fn (Get $get): array => is_numeric($get('city_id'))
                        ? District::query()->where('city_id', (int) $get('city_id'))->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all()
                        : []),
                TagsInput::make('phones')->label('تلفن‌ها')->placeholder('شماره و Enter')->helperText('اولین شماره روی صفحه مجموعه نمایش داده می‌شود.'),
                TextInput::make('website')->label('وب‌سایت')->url()->maxLength(255)->extraInputAttributes(['dir' => 'ltr']),
                Grid::make(2)->schema([
                    TextInput::make('age_min_months')->label('حداقل سن کودک (ماه)')->integer()->minValue(0)->maxValue(240),
                    TextInput::make('age_max_months')->label('حداکثر سن کودک (ماه)')->integer()->minValue(0)->maxValue(240),
                ]),
            ]),
            Textarea::make('summary')->label('خلاصه (کارت و توضیح متا)')->rows(2)->maxLength(300)->live(onBlur: true),
            Textarea::make('description')->label('درباره مجموعه')->rows(6),
            CheckboxList::make('amenity_ids')->label('امکانات')->columns(3)
                ->options(static fn (): array => Amenity::query()->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all()),
            Radio::make('booking_mode')->label('نحوه رزرو')
                ->options(self::bookingModeOptions())
                ->descriptions(self::bookingModeDescriptions())
                ->default(BookingMode::Online->value)
                ->required(),
            Grid::make(2)->schema([
                Textarea::make('rules')->label('قوانین (هر خط یک مورد)')->rows(4),
                Textarea::make('cancellation_policy')->label('شرایط لغو')->rows(4),
            ]),
            Grid::make(2)->schema([
                Toggle::make('is_verified')->label('اطلاعات بررسی و تأیید شده')->default(false),
                Toggle::make('is_demo')->label('مجموعه نمونه (نمایشی)')->default(false)
                    ->helperText('مجموعه‌های نمونه noindex می‌مانند و در امتیاز و نقشه سایت حساب نمی‌شوند.'),
            ]),
        ];
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with(['category', 'city']))
            ->defaultSort('updated_at', 'desc')
            ->columns([
                TextColumn::make('name')->label('نام')->searchable()->sortable(),
                TextColumn::make('category.name')->label('دسته'),
                TextColumn::make('city.name')->label('شهر'),
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (PlaceStatus $state): string => $state->label())
                    ->color(static fn (PlaceStatus $state): string => self::statusColor($state)),
                IconColumn::make('is_demo')->label('نمونه')->boolean(),
                TextColumn::make('rating_count')->label('نظرها')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                TextColumn::make('updated_at')->label('به‌روزرسانی')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
            ])
            ->filters([
                SelectFilter::make('status')->label('وضعیت')->options(self::statusOptions()),
                SelectFilter::make('category_id')->label('دسته')->relationship('category', 'name'),
                SelectFilter::make('city_id')->label('شهر')->relationship('city', 'name'),
                TernaryFilter::make('is_demo')->label('نمونه'),
            ])
            ->recordActions([
                EditAction::make(),
                self::statusAction('publish', PlaceStatus::Published, 'انتشار', Heroicon::OutlinedEye),
            ]);
    }

    /**
     * Publish / back to draft / suspend — through ChangePlaceStatus (cache bump + activity log). Hidden when the
     * place already has that status.
     */
    public static function statusAction(string $name, PlaceStatus $status, string $label, Heroicon $icon): Action
    {
        return Action::make($name)
            ->label($label)
            ->icon($icon)
            ->color($status === PlaceStatus::Published ? 'success' : 'gray')
            ->requiresConfirmation($status !== PlaceStatus::Published)
            ->visible(static fn (Place $record): bool => $record->status !== $status)
            ->authorize(static fn (Place $record): bool => DirectoryAdmin::can('update', $record))
            ->action(static function (Place $record, ChangePlaceStatus $change) use ($status): void {
                $change->handle($record, $status, DirectoryAdmin::user());
            })
            ->successNotificationTitle('وضعیت مجموعه: '.$status->label());
    }

    public static function statusColor(PlaceStatus $status): string
    {
        return match ($status) {
            PlaceStatus::Draft => 'gray',
            PlaceStatus::Published => 'success',
            PlaceStatus::Suspended => 'danger',
        };
    }

    /**
     * @return array<string, string>
     */
    public static function statusOptions(): array
    {
        $options = [];
        foreach (PlaceStatus::cases() as $status) {
            $options[$status->value] = $status->label();
        }

        return $options;
    }

    /**
     * @return array<string, string>
     */
    private static function bookingModeOptions(): array
    {
        $options = [];
        foreach (BookingMode::cases() as $mode) {
            $options[$mode->value] = $mode->label();
        }

        return $options;
    }

    /**
     * @return array<string, string>
     */
    private static function bookingModeDescriptions(): array
    {
        return [
            BookingMode::Online->value => 'فرم درخواست رزرو در صفحه مجموعه نمایش داده می‌شود.',
            BookingMode::Phone->value => 'به‌جای فرم رزرو فقط شماره مجموعه نمایش داده می‌شود.',
        ];
    }

    public static function getPages(): array
    {
        return [
            'index' => ListPlaces::route('/'),
            'create' => CreatePlace::route('/create'),
            'edit' => EditPlace::route('/{record}/edit'),
        ];
    }
}
