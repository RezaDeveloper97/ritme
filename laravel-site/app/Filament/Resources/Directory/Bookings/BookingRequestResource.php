<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Bookings;

use App\Domain\Directory\Booking\Actions\ChangeBookingStatus;
use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Enums\TimeWindow;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Booking\Support\MobileMask;
use App\Filament\Resources\Directory\Bookings\Pages\ListBookingRequests;
use App\Filament\Resources\Directory\Bookings\Pages\ViewBookingRequest;
use App\Filament\Resources\Directory\DirectoryAdmin;
use App\Support\Money\Money;
use BackedEnum;
use Closure;
use Filament\Actions\Action;
use Filament\Actions\ViewAction;
use Filament\Forms\Components\DatePicker;
use Filament\Infolists\Components\TextEntry;
use Filament\Panel;
use Filament\Resources\Resource;
use Filament\Resources\ResourceConfiguration;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\Filter;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Facades\Route;
use UnitEnum;

/**
 * Bookings board (L5-06): tabs per status (new / confirmed / done / cancelled / all), filters by place and preferred
 * day, status flow actions (ChangeBookingStatus: new → confirmed | cancelled, confirmed → done | cancelled; activity
 * log) and a streamed CSV of the current view. Lists show the mobile MASKED; the full number (to call the parent
 * back) and the note are only on the request's own page. Access: BookingRequestPolicy (directory-manager +
 * super-admin; nobody creates, edits or deletes requests here).
 */
final class BookingRequestResource extends Resource
{
    protected static ?string $model = BookingRequest::class;

    protected static ?string $slug = 'directory/bookings';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedCalendarDays;

    protected static string|UnitEnum|null $navigationGroup = DirectoryAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 70;

    protected static ?string $navigationLabel = 'درخواست‌های رزرو';

    protected static ?string $modelLabel = 'درخواست رزرو';

    protected static ?string $pluralModelLabel = 'درخواست‌های رزرو';

    protected static ?string $recordTitleAttribute = 'code';

    public static function getNavigationBadge(): ?string
    {
        $new = BookingRequest::query()->where('status', BookingStatus::New->value)->count();

        return $new > 0 ? fa_digits($new) : null;
    }

    public static function getNavigationBadgeTooltip(): string
    {
        return 'در انتظار تأیید';
    }

    public static function table(Table $table): Table
    {
        return $table
            ->defaultSort('id', 'desc')
            ->recordUrl(static fn (BookingRequest $record): string => self::getUrl('view', ['record' => $record]))
            ->columns([
                TextColumn::make('code')->label('کد')->searchable()->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (BookingStatus $state): string => $state->label())
                    ->color(static fn (BookingStatus $state): string => $state->color()),
                TextColumn::make('place_name')->label('مجموعه')->searchable(),
                TextColumn::make('service_name')->label('خدمت')->placeholder('—')->limit(30),
                TextColumn::make('preferred_date')->label('روز درخواستی')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state, 'l j F')),
                TextColumn::make('time_window')->label('بازه')
                    ->formatStateUsing(static fn (mixed $state): string => $state instanceof TimeWindow ? $state->label() : ''),
                TextColumn::make('parent_name')->label('نام والد')->searchable(),
                TextColumn::make('mobile')->label('موبایل')
                    ->formatStateUsing(static fn (string $state): string => MobileMask::mask($state))
                    ->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('created_at')->label('دریافت')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
            ])
            ->filters([
                SelectFilter::make('place_id')->label('مجموعه')->relationship('place', 'name')->searchable(),
                Filter::make('preferred')
                    ->label('روز درخواستی')
                    ->schema([
                        DatePicker::make('from')->label('از تاریخ'),
                        DatePicker::make('until')->label('تا تاریخ'),
                    ])
                    ->query(static fn (Builder $query, array $data): Builder => $query
                        ->when($data['from'] ?? null, static fn (Builder $q, mixed $date): Builder => $q->whereDate('preferred_date', '>=', (string) $date))
                        ->when($data['until'] ?? null, static fn (Builder $q, mixed $date): Builder => $q->whereDate('preferred_date', '<=', (string) $date)))
                    ->indicateUsing(static function (array $data): array {
                        $indicators = [];
                        if (! empty($data['from'])) {
                            $indicators[] = 'از '.jdate((string) $data['from'], 'Y/m/d');
                        }
                        if (! empty($data['until'])) {
                            $indicators[] = 'تا '.jdate((string) $data['until'], 'Y/m/d');
                        }

                        return $indicators;
                    }),
            ])
            ->recordActions([
                ViewAction::make(),
                ...self::statusActions(),
            ]);
    }

    public static function infolist(Schema $schema): Schema
    {
        return $schema->components([
            TextEntry::make('code')->label('کد پیگیری')->copyable()->extraAttributes(['dir' => 'ltr']),
            TextEntry::make('status')->label('وضعیت')->badge()
                ->formatStateUsing(static fn (BookingStatus $state): string => $state->label())
                ->color(static fn (BookingStatus $state): string => $state->color()),
            TextEntry::make('place_name')->label('مجموعه'),
            TextEntry::make('service_name')->label('خدمت')->placeholder('—'),
            TextEntry::make('service_price')->label('قیمت اعلام‌شده هنگام درخواست')->placeholder('—')
                ->formatStateUsing(static fn (mixed $state, BookingRequest $record): ?string => is_numeric($state)
                    ? trim(Money::fromToman((int) $state)->format().' '.($record->service_price_unit ?? ''))
                    : null),
            TextEntry::make('preferred_date')->label('روز درخواستی')
                ->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state, 'l j F Y')),
            TextEntry::make('time_window')->label('بازه زمانی')
                ->formatStateUsing(static fn (mixed $state): string => $state instanceof TimeWindow ? $state->describe() : ''),
            TextEntry::make('parent_name')->label('نام والد'),
            TextEntry::make('mobile')->label('موبایل')->copyable()
                ->url(static fn (BookingRequest $record): string => 'tel:'.$record->mobile)
                ->extraAttributes(['dir' => 'ltr']),
            TextEntry::make('child_age_months')->label('سن کودک')->placeholder('—')
                ->formatStateUsing(static fn (mixed $state): ?string => is_numeric($state) ? fa_digits((int) $state).' ماه' : null),
            TextEntry::make('created_at')->label('دریافت')->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
            TextEntry::make('note')->label('توضیح والد')->placeholder('—')->columnSpanFull()
                ->formatStateUsing(static fn (string $state): string => e($state))
                ->html()->extraAttributes(['class' => 'whitespace-pre-line']),
        ]);
    }

    /**
     * One action per target status, visible only where the flow allows it.
     *
     * @return list<Action>
     */
    public static function statusActions(): array
    {
        $actions = [];
        foreach ([
            [BookingStatus::Confirmed, 'تأیید رزرو', Heroicon::OutlinedCheckCircle, 'success'],
            [BookingStatus::Done, 'انجام شد', Heroicon::OutlinedCheckBadge, 'info'],
            [BookingStatus::Cancelled, 'لغو', Heroicon::OutlinedXCircle, 'danger'],
        ] as [$status, $label, $icon, $color]) {
            $actions[] = Action::make('status_'.$status->value)
                ->label($label)
                ->icon($icon)
                ->color($color)
                ->requiresConfirmation($status === BookingStatus::Cancelled)
                ->visible(static fn (BookingRequest $record): bool => ChangeBookingStatus::allowed($record->status, $status))
                ->authorize(static fn (BookingRequest $record): bool => DirectoryAdmin::can('update', $record))
                ->action(static function (BookingRequest $record, ChangeBookingStatus $change) use ($status): void {
                    $change->handle($record, $status, DirectoryAdmin::user());
                })
                ->successNotificationTitle('وضعیت رزرو: '.$status->label());
        }

        return $actions;
    }

    /**
     * Header action on the list: a link to the streamed export carrying the current tab, filters and search.
     */
    public static function exportAction(): Action
    {
        return Action::make('export')
            ->label('خروجی CSV')
            ->icon(Heroicon::OutlinedArrowDownTray)
            ->color('gray')
            ->visible(static fn (): bool => DirectoryAdmin::can('export', BookingRequest::class))
            ->url(static fn (ListBookingRequests $livewire): string => self::getUrl('export', $livewire->exportParameters()));
    }

    /**
     * Resource pages plus the `export` download route (same slug prefix, same auth middleware).
     */
    public static function registerRoutes(Panel $panel, ?Closure $registerPageRoutes = null, ?ResourceConfiguration $configuration = null): void
    {
        $registerPageRoutes ??= static function () use ($panel): void {
            Route::get('export', ExportBookingRequestsController::class)->name('export');

            foreach (self::getPages() as $name => $page) {
                $page->registerRoute($panel)?->name($name);
            }
        };

        parent::registerRoutes($panel, $registerPageRoutes, $configuration);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListBookingRequests::route('/'),
            'view' => ViewBookingRequest::route('/{record}'),
        ];
    }
}
