<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\JoinRequests;

use App\Domain\Directory\Booking\Support\MobileMask;
use App\Domain\Directory\Join\Actions\ApproveJoinRequest;
use App\Domain\Directory\Join\Actions\RejectJoinRequest;
use App\Domain\Directory\Join\Enums\AgeGroup;
use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Directory\Join\Enums\JoinRequestStatus;
use App\Domain\Directory\Join\Models\JoinRequest;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Support\OpeningHours;
use App\Filament\Resources\Directory\DirectoryAdmin;
use App\Filament\Resources\Directory\JoinRequests\Pages\ListJoinRequests;
use App\Filament\Resources\Directory\JoinRequests\Pages\ViewJoinRequest;
use App\Filament\Resources\Directory\Places\PlaceResource;
use App\Filament\Resources\Media\MediaPresenter;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Actions\DeleteAction;
use Filament\Actions\ViewAction;
use Filament\Forms\Components\Select;
use Filament\Infolists\Components\TextEntry;
use Filament\Notifications\Notification;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use InvalidArgumentException;
use UnitEnum;

/**
 * «ثبت مجموعه» requests (L5-06): tabs pending / approved / rejected / all, a read-only review page (business data,
 * photos, hours, amenities; the contact person's data only there — lists show the mobile masked), «تبدیل به
 * پیش‌نویس مجموعه» (ApproveJoinRequest: draft place with the request's data and photos, then straight to its edit
 * page to complete and publish it) and reject (RejectJoinRequest); both activity-logged. Access: JoinRequestPolicy.
 */
final class JoinRequestResource extends Resource
{
    protected static ?string $model = JoinRequest::class;

    protected static ?string $slug = 'directory/join-requests';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedClipboardDocumentCheck;

    protected static string|UnitEnum|null $navigationGroup = DirectoryAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 80;

    protected static ?string $navigationLabel = 'درخواست‌های ثبت مجموعه';

    protected static ?string $modelLabel = 'درخواست ثبت مجموعه';

    protected static ?string $pluralModelLabel = 'درخواست‌های ثبت مجموعه';

    protected static ?string $recordTitleAttribute = 'name';

    public static function getNavigationBadge(): ?string
    {
        $pending = JoinRequest::query()->where('status', JoinRequestStatus::Pending->value)->count();

        return $pending > 0 ? fa_digits($pending) : null;
    }

    public static function getNavigationBadgeTooltip(): string
    {
        return 'در انتظار بررسی';
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with(['category:id,name', 'city:id,name']))
            ->defaultSort('id', 'desc')
            ->recordUrl(static fn (JoinRequest $record): string => self::getUrl('view', ['record' => $record]))
            ->columns([
                TextColumn::make('code')->label('کد')->searchable()->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (JoinRequestStatus $state): string => $state->label())
                    ->color(static fn (JoinRequestStatus $state): string => $state->color()),
                TextColumn::make('name')->label('نام مجموعه')->searchable(),
                TextColumn::make('category.name')->label('دسته')->placeholder('—'),
                TextColumn::make('city.name')->label('شهر')->placeholder('—'),
                TextColumn::make('contact_name')->label('رابط'),
                TextColumn::make('mobile')->label('موبایل')
                    ->formatStateUsing(static fn (string $state): string => MobileMask::mask($state))
                    ->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('created_at')->label('دریافت')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
            ])
            ->filters([
                SelectFilter::make('category_id')->label('دسته')->relationship('category', 'name'),
                SelectFilter::make('city_id')->label('شهر')->relationship('city', 'name'),
            ])
            ->recordActions([
                ViewAction::make(),
                self::approveAction(),
                self::rejectAction(),
            ]);
    }

    public static function infolist(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Section::make('مجموعه')->columns(2)->schema([
                TextEntry::make('code')->label('کد پیگیری')->extraAttributes(['dir' => 'ltr']),
                TextEntry::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (JoinRequestStatus $state): string => $state->label())
                    ->color(static fn (JoinRequestStatus $state): string => $state->color()),
                TextEntry::make('name')->label('نام مجموعه'),
                TextEntry::make('category.name')->label('دسته')->placeholder('—'),
                TextEntry::make('city.name')->label('شهر')->placeholder('—'),
                TextEntry::make('district.name')->label('محله')->placeholder('—'),
                TextEntry::make('address')->label('نشانی')->columnSpanFull(),
                TextEntry::make('phone')->label('تلفن ثابت')->placeholder('—')->extraAttributes(['dir' => 'ltr']),
                TextEntry::make('coordinates')->label('مختصات')->placeholder('—')
                    ->state(static fn (JoinRequest $record): ?string => $record->latitude !== null && $record->longitude !== null ? $record->latitude.', '.$record->longitude : null)
                    ->extraAttributes(['dir' => 'ltr']),
                TextEntry::make('booking_mode')->label('نحوه رزرو')
                    ->formatStateUsing(static fn (mixed $state): string => $state instanceof BookingMode ? $state->label() : ''),
                TextEntry::make('age_groups')->label('رده سنی')->placeholder('—')
                    ->state(static fn (JoinRequest $record): ?string => self::ageGroups($record)),
                TextEntry::make('amenities')->label('امکانات')->placeholder('—')->columnSpanFull()
                    ->state(static fn (JoinRequest $record): ?string => self::amenities($record)),
                TextEntry::make('hours')->label('ساعات کاری')->placeholder('—')->columnSpanFull()
                    ->state(static fn (JoinRequest $record): ?string => self::hours($record))
                    ->extraAttributes(['class' => 'whitespace-pre-line']),
                TextEntry::make('about')->label('درباره مجموعه')->placeholder('—')->columnSpanFull()
                    ->formatStateUsing(static fn (string $state): string => e($state))->html()->extraAttributes(['class' => 'whitespace-pre-line']),
                TextEntry::make('services')->label('خدمات و قیمت')->placeholder('—')->columnSpanFull()
                    ->formatStateUsing(static fn (string $state): string => e($state))->html()->extraAttributes(['class' => 'whitespace-pre-line']),
                TextEntry::make('photos')->label('تصاویر')->placeholder('—')->columnSpanFull()->html()
                    ->state(static fn (JoinRequest $record): ?string => self::photos($record)),
            ]),
            Section::make('رابط (فقط برای تیم ریتمی؛ روی صفحه مجموعه نمی‌آید)')->columns(3)->schema([
                TextEntry::make('contact_name')->label('نام'),
                TextEntry::make('mobile')->label('موبایل')->copyable()->url(static fn (JoinRequest $record): string => 'tel:'.$record->mobile)->extraAttributes(['dir' => 'ltr']),
                TextEntry::make('email')->label('ایمیل')->placeholder('—')->copyable(),
                TextEntry::make('created_at')->label('دریافت')->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
                TextEntry::make('terms_accepted_at')->label('پذیرش شرایط')->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
            ]),
        ]);
    }

    /**
     * «تبدیل به پیش‌نویس مجموعه»: category and city default to the request's (required when it has none).
     */
    public static function approveAction(): Action
    {
        return Action::make('approve')
            ->label('تبدیل به پیش‌نویس مجموعه')
            ->icon(Heroicon::OutlinedBuildingStorefront)
            ->color('success')
            ->visible(static fn (JoinRequest $record): bool => $record->status === JoinRequestStatus::Pending)
            ->authorize(static fn (JoinRequest $record): bool => DirectoryAdmin::can('update', $record) && DirectoryAdmin::can('create', Place::class))
            ->modalHeading('تبدیل به پیش‌نویس مجموعه')
            ->modalDescription('اطلاعات و تصاویر درخواست در یک مجموعه پیش‌نویس کپی می‌شود؛ اطلاعات رابط کپی نمی‌شود. بعد از تکمیل، مجموعه را منتشر کنید.')
            ->modalSubmitActionLabel('ساخت پیش‌نویس')
            ->fillForm(static fn (JoinRequest $record): array => ['category_id' => $record->category_id, 'city_id' => $record->city_id])
            ->schema([
                Select::make('category_id')->label('دسته')->required()
                    ->options(static fn (): array => PlaceCategory::query()->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all()),
                Select::make('city_id')->label('شهر')->required()
                    ->options(static fn (): array => City::query()->orderBy('sort_order')->orderBy('name')->pluck('name', 'id')->all()),
            ])
            ->action(static function (JoinRequest $record, array $data, ApproveJoinRequest $approve, Action $action): void {
                try {
                    $place = $approve->handle($record, (int) $data['category_id'], (int) $data['city_id'], DirectoryAdmin::user());
                } catch (InvalidArgumentException) {
                    Notification::make()->danger()->title('این درخواست قابل تبدیل نیست')->send();
                    $action->halt();

                    return;
                }

                Notification::make()->success()->title('پیش‌نویس مجموعه ساخته شد')->send();
                $action->redirect(PlaceResource::getUrl('edit', ['record' => $place]));
            });
    }

    public static function rejectAction(): Action
    {
        return Action::make('reject')
            ->label('رد درخواست')
            ->icon(Heroicon::OutlinedXCircle)
            ->color('danger')
            ->requiresConfirmation()
            ->visible(static fn (JoinRequest $record): bool => $record->status === JoinRequestStatus::Pending)
            ->authorize(static fn (JoinRequest $record): bool => DirectoryAdmin::can('update', $record))
            ->action(static function (JoinRequest $record, RejectJoinRequest $reject): void {
                $reject->handle($record, DirectoryAdmin::user());
            })
            ->successNotificationTitle('درخواست رد شد');
    }

    public static function deleteAction(): DeleteAction
    {
        return DeleteAction::make();
    }

    private static function ageGroups(JoinRequest $record): ?string
    {
        $labels = array_map(static fn (AgeGroup $g): string => $g->label(), array_values(array_filter(array_map(AgeGroup::tryFrom(...), $record->age_groups ?? []))));

        return $labels === [] ? null : implode('، ', $labels);
    }

    private static function amenities(JoinRequest $record): ?string
    {
        $ids = array_map(intval(...), $record->amenity_ids ?? []);
        $names = $ids === [] ? [] : Amenity::query()->whereKey($ids)->orderBy('sort_order')->pluck('name')->all();

        return $names === [] ? null : implode('، ', $names);
    }

    private static function hours(JoinRequest $record): ?string
    {
        $hours = OpeningHours::fromArray($record->opening_hours);
        if ($hours->toArray() === []) {
            return null;
        }

        $lines = [];
        foreach ($hours->rows() as $row) {
            $lines[] = $row['label'].': '.($row['hours'] ?? 'نامشخص');
        }

        return implode("\n", $lines);
    }

    private static function photos(JoinRequest $record): ?string
    {
        $html = '';
        foreach ($record->photos as $media) {
            $html .= '<img src="'.e(MediaPresenter::thumbUrl($media)).'" alt="'.e((string) $media->alt).'" width="120" height="120" class="size-30 rounded-lg object-cover" loading="lazy">';
        }

        return $html === '' ? null : '<span class="flex flex-wrap gap-2">'.$html.'</span>';
    }

    public static function getPages(): array
    {
        return [
            'index' => ListJoinRequests::route('/'),
            'view' => ViewJoinRequest::route('/{record}'),
        ];
    }
}
