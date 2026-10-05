<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Orders;

use App\Domain\Directory\Booking\Support\MobileMask;
use App\Domain\Shop\Ordering\Actions\AddOrderNote;
use App\Domain\Shop\Ordering\Actions\ChangeOrderStatus;
use App\Domain\Shop\Ordering\Actions\ExportOrders;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Ordering\Models\OrderItem;
use App\Domain\Shop\Payment\Enums\PaymentStatus;
use App\Filament\Resources\Shop\Orders\Pages\ListOrders;
use App\Filament\Resources\Shop\Orders\Pages\ViewOrder;
use App\Filament\Resources\Shop\ShopAdmin;
use BackedEnum;
use Closure;
use Filament\Actions\Action;
use Filament\Actions\ViewAction;
use Filament\Forms\Components\DatePicker;
use Filament\Forms\Components\Textarea;
use Filament\Infolists\Components\IconEntry;
use Filament\Infolists\Components\RepeatableEntry;
use Filament\Infolists\Components\TextEntry;
use Filament\Notifications\Notification;
use Filament\Panel;
use Filament\Resources\Resource;
use Filament\Resources\ResourceConfiguration;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\Filter;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Filters\TernaryFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\HtmlString;
use InvalidArgumentException;
use UnitEnum;

/**
 * Shop orders (L6-06): tabs per status (with counts), filters by placed date / payment / province / demo, search by
 * order code or full mobile number, lifecycle actions through ChangeOrderStatus (state machine — only allowed
 * transitions are offered and the action refuses the rest; cancellation returns the tracked stock; activity log),
 * internal notes (AddOrderNote), a printable invoice and a streamed CSV of the current view. Lists show the
 * recipient's name and mobile MASKED and never the address; the full data is only on the order page.
 * Access: OrderPolicy (shop managers + super-admins; nobody creates, edits or deletes orders here).
 */
final class OrderResource extends Resource
{
    protected static ?string $model = Order::class;

    protected static ?string $slug = 'shop/orders';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedClipboardDocumentList;

    protected static string|UnitEnum|null $navigationGroup = ShopAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 10;

    protected static ?string $navigationLabel = 'سفارش‌ها';

    protected static ?string $modelLabel = 'سفارش';

    protected static ?string $pluralModelLabel = 'سفارش‌ها';

    protected static ?string $recordTitleAttribute = 'code';

    public static function getNavigationBadge(): ?string
    {
        $pending = Order::query()->where('status', OrderStatus::Pending->value)->count();

        return $pending > 0 ? fa_digits($pending) : null;
    }

    public static function getNavigationBadgeTooltip(): string
    {
        return 'در انتظار تأیید';
    }

    public static function table(Table $table): Table
    {
        return $table
            ->defaultSort('id', 'desc')
            ->recordUrl(static fn (Order $record): string => self::getUrl('view', ['record' => $record]))
            ->columns([
                TextColumn::make('code')->label('کد سفارش')->extraAttributes(['dir' => 'ltr'])
                    ->searchable(query: static fn (Builder $query, string $search): Builder => ExportOrders::search($query, $search)),
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (OrderStatus $state): string => $state->label())
                    ->color(static fn (OrderStatus $state): string => self::statusColor($state)),
                TextColumn::make('payment_status')->label('پرداخت')->badge()
                    ->formatStateUsing(static fn (PaymentStatus $state): string => $state->label())
                    ->color(static fn (PaymentStatus $state): string => $state === PaymentStatus::Paid ? 'success' : 'gray'),
                TextColumn::make('recipient_name')->label('گیرنده')
                    ->formatStateUsing(static fn (string $state): string => ShopAdmin::maskName($state)),
                TextColumn::make('mobile')->label('موبایل')
                    ->formatStateUsing(static fn (string $state): string => MobileMask::mask($state))
                    ->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('city')->label('شهر')
                    ->formatStateUsing(static fn (string $state, Order $record): string => $record->province.' / '.$state),
                TextColumn::make('items_count')->label('اقلام')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                TextColumn::make('total')->label('مبلغ کل')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::money($state)),
                TextColumn::make('delivery_date')->label('تحویل')->sortable()
                    ->formatStateUsing(static fn (mixed $state, Order $record): string => trim((ShopAdmin::date($state, 'l j F') ?? '').' '.$record->delivery_window->hours())),
                IconColumn::make('is_demo')->label('نمونه')->boolean()->toggleable(isToggledHiddenByDefault: true),
                TextColumn::make('created_at')->label('ثبت')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::date($state)),
            ])
            ->filters([
                Filter::make('placed')
                    ->label('تاریخ ثبت')
                    ->schema([
                        DatePicker::make('from')->label('از تاریخ'),
                        DatePicker::make('until')->label('تا تاریخ'),
                    ])
                    ->query(static fn (Builder $query, array $data): Builder => $query
                        ->when($data['from'] ?? null, static fn (Builder $q, mixed $date): Builder => $q->whereDate('created_at', '>=', (string) $date))
                        ->when($data['until'] ?? null, static fn (Builder $q, mixed $date): Builder => $q->whereDate('created_at', '<=', (string) $date)))
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
                SelectFilter::make('payment_status')->label('پرداخت')->options(self::paymentOptions()),
                SelectFilter::make('province')->label('استان')->searchable()
                    ->options(static fn (): array => Order::query()->distinct()->orderBy('province')->pluck('province', 'province')->all()),
                TernaryFilter::make('is_demo')->label('نمونه'),
            ])
            ->recordActions([
                ViewAction::make(),
                ...self::statusActions(),
            ]);
    }

    public static function infolist(Schema $schema): Schema
    {
        return $schema->components([
            Section::make('سفارش')->columns(3)->schema([
                TextEntry::make('code')->label('کد سفارش')->copyable()->extraAttributes(['dir' => 'ltr']),
                TextEntry::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (OrderStatus $state): string => $state->label())
                    ->color(static fn (OrderStatus $state): string => self::statusColor($state)),
                TextEntry::make('payment_status')->label('پرداخت')
                    ->formatStateUsing(static fn (PaymentStatus $state, Order $record): string => $record->payment_method->label().' — '.$state->label()),
                TextEntry::make('created_at')->label('ثبت')->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::date($state)),
                TextEntry::make('delivery_date')->label('تحویل ترجیحی')
                    ->formatStateUsing(static fn (mixed $state, Order $record): string => trim((ShopAdmin::date($state, 'l j F Y') ?? '').' — ساعت '.$record->delivery_window->hours())),
                IconEntry::make('discreet_packaging')->label('بسته‌بندی بدون نام محصول')->boolean(),
            ]),
            Section::make('گیرنده')->columns(3)->schema([
                TextEntry::make('recipient_name')->label('نام'),
                TextEntry::make('mobile')->label('موبایل')->copyable()
                    ->url(static fn (Order $record): string => 'tel:'.$record->mobile)
                    ->extraAttributes(['dir' => 'ltr']),
                TextEntry::make('postal_code')->label('کد پستی')->placeholder('—')->extraAttributes(['dir' => 'ltr']),
                TextEntry::make('province')->label('استان / شهر')
                    ->formatStateUsing(static fn (string $state, Order $record): string => $state.' / '.$record->city),
                TextEntry::make('address')->label('نشانی')->columnSpan(2),
                TextEntry::make('note')->label('توضیح مشتری')->placeholder('—')->columnSpanFull()
                    ->formatStateUsing(static fn (string $state): string => e($state))
                    ->html()->extraAttributes(['class' => 'whitespace-pre-line']),
            ]),
            Section::make('اقلام')->schema([
                RepeatableEntry::make('items')->hiddenLabel()->columns(5)->schema([
                    TextEntry::make('title')->label('محصول')->columnSpan(2)
                        ->formatStateUsing(static fn (string $state, OrderItem $record): string => $record->variant_label !== null ? $state.' — '.$record->variant_label : $state),
                    TextEntry::make('quantity')->label('تعداد')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                    TextEntry::make('unit_price')->label('قیمت واحد')->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::money($state)),
                    TextEntry::make('line_total')->label('جمع')
                        ->formatStateUsing(static fn (mixed $state, OrderItem $record): string => (ShopAdmin::money($state) ?? '').($record->stock_tracked ? '' : ' (پیش‌سفارش)')),
                ]),
                Grid::make(3)->schema([
                    TextEntry::make('subtotal')->label('جمع کالاها')->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::money($state)),
                    TextEntry::make('shipping_fee')->label('هزینه ارسال')->placeholder('هنگام تماس اعلام می‌شود')
                        ->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::money($state)),
                    TextEntry::make('total')->label('مبلغ کل')->weight('bold')->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::money($state)),
                ]),
            ]),
            Section::make('یادداشت‌ها و تاریخچه')->schema([
                TextEntry::make('history')->hiddenLabel()->placeholder('هنوز یادداشتی ثبت نشده است.')
                    ->state(static fn (Order $record): ?HtmlString => self::history($record))
                    ->html(),
            ]),
        ]);
    }

    /**
     * One action per target status, visible only where the state machine allows it. ChangeOrderStatus still checks
     * (another admin may have moved the order) — a refused change shows an error and changes nothing.
     *
     * @return list<Action>
     */
    public static function statusActions(): array
    {
        $actions = [];
        foreach ([
            [OrderStatus::Confirmed, 'تأیید سفارش', Heroicon::OutlinedCheckCircle, 'success'],
            [OrderStatus::Shipped, 'ارسال شد', Heroicon::OutlinedTruck, 'info'],
            [OrderStatus::Delivered, 'تحویل شد (وجه دریافت شد)', Heroicon::OutlinedCheckBadge, 'success'],
            [OrderStatus::Cancelled, 'لغو سفارش', Heroicon::OutlinedXCircle, 'danger'],
        ] as [$status, $label, $icon, $color]) {
            $actions[] = Action::make('status_'.$status->value)
                ->label($label)
                ->icon($icon)
                ->color($color)
                ->requiresConfirmation($status === OrderStatus::Cancelled || $status === OrderStatus::Delivered)
                ->modalDescription($status === OrderStatus::Cancelled ? 'موجودی اقلام این سفارش به انبار برمی‌گردد. این کار برگشت‌پذیر نیست.' : null)
                ->visible(static fn (Order $record): bool => ChangeOrderStatus::allowed($record->status, $status))
                ->authorize(static fn (Order $record): bool => ShopAdmin::can('update', $record))
                ->action(static function (Order $record, ChangeOrderStatus $change, Action $action) use ($status): void {
                    try {
                        $change->handle($record, $status, ShopAdmin::user());
                    } catch (InvalidArgumentException) {
                        Notification::make()->danger()->title('این تغییر وضعیت مجاز نیست.')
                            ->body('ممکن است وضعیت سفارش هم‌زمان تغییر کرده باشد؛ صفحه را دوباره باز کنید.')->send();
                        $action->halt();
                    }
                })
                ->successNotificationTitle('وضعیت سفارش: '.$status->label());
        }

        return $actions;
    }

    public static function noteAction(): Action
    {
        return Action::make('note')
            ->label('یادداشت داخلی')
            ->icon(Heroicon::OutlinedPencilSquare)
            ->color('gray')
            ->authorize(static fn (Order $record): bool => ShopAdmin::can('update', $record))
            ->schema([
                Textarea::make('note')->label('یادداشت (فقط برای تیم؛ مشتری نمی‌بیند)')->required()->rows(3)->maxLength(AddOrderNote::MAX_LENGTH),
            ])
            ->action(static function (Order $record, array $data, AddOrderNote $add): void {
                $add->handle($record, (string) ($data['note'] ?? ''), ShopAdmin::user());
            })
            ->successNotificationTitle('یادداشت ثبت شد');
    }

    public static function invoiceAction(): Action
    {
        return Action::make('invoice')
            ->label('فاکتور چاپی')
            ->icon(Heroicon::OutlinedPrinter)
            ->color('gray')
            ->url(static fn (Order $record): string => self::getUrl('invoice', ['record' => $record]), shouldOpenInNewTab: true);
    }

    /**
     * Header action on the list: a link to the streamed export carrying the current tab, date range and search.
     */
    public static function exportAction(): Action
    {
        return Action::make('export')
            ->label('خروجی CSV')
            ->icon(Heroicon::OutlinedArrowDownTray)
            ->color('gray')
            ->visible(static fn (): bool => ShopAdmin::can('export', Order::class))
            ->url(static fn (ListOrders $livewire): string => self::getUrl('export', $livewire->exportParameters()));
    }

    public static function statusColor(OrderStatus $status): string
    {
        return match ($status) {
            OrderStatus::Pending => 'warning',
            OrderStatus::Confirmed => 'info',
            OrderStatus::Shipped => 'primary',
            OrderStatus::Delivered => 'success',
            OrderStatus::Cancelled => 'gray',
        };
    }

    /**
     * Notes and status changes of the order, newest first, as escaped HTML.
     */
    private static function history(Order $order): ?HtmlString
    {
        $rows = [];
        foreach (AddOrderNote::history($order) as $entry) {
            $who = $entry->causer?->getAttribute('name');
            $meta = e(trim((ShopAdmin::date($entry->created_at) ?? '').' — '.(is_string($who) && $who !== '' ? $who : 'سیستم')));
            if ($entry->description === AddOrderNote::DESCRIPTION) {
                $text = e((string) $entry->getExtraProperty('note'));
                $rows[] = '<li class="py-2"><div class="text-xs text-gray-500">'.$meta.' — یادداشت</div><div class="whitespace-pre-line">'.$text.'</div></li>';
            } else {
                $from = OrderStatus::tryFrom((string) data_get($entry->properties, 'old.status'));
                $to = OrderStatus::tryFrom((string) data_get($entry->properties, 'attributes.status'));
                $skipped = count((array) data_get($entry->properties, 'stock.skipped', []));
                $change = e(($from?->label() ?? '?').' ← '.($to?->label() ?? '?'));
                $note = $skipped > 0 ? ' <span class="text-danger-600">('.e(fa_digits($skipped)).' قلم به انبار برنگشت)</span>' : '';
                $rows[] = '<li class="py-2"><div class="text-xs text-gray-500">'.$meta.' — تغییر وضعیت</div><div>'.$change.$note.'</div></li>';
            }
        }

        return $rows === [] ? null : new HtmlString('<ul class="divide-y divide-gray-200 dark:divide-white/10">'.implode('', $rows).'</ul>');
    }

    /**
     * @return array<string, string>
     */
    private static function paymentOptions(): array
    {
        $options = [];
        foreach (PaymentStatus::cases() as $status) {
            $options[$status->value] = $status->label();
        }

        return $options;
    }

    /**
     * Resource pages plus the `export` download and the printable `invoice` (same slug prefix, same auth middleware).
     */
    public static function registerRoutes(Panel $panel, ?Closure $registerPageRoutes = null, ?ResourceConfiguration $configuration = null): void
    {
        $registerPageRoutes ??= static function () use ($panel): void {
            Route::get('export', ExportOrdersController::class)->name('export');
            Route::get('{record}/invoice', OrderInvoiceController::class)->name('invoice');

            foreach (self::getPages() as $name => $page) {
                $page->registerRoute($panel)?->name($name);
            }
        };

        parent::registerRoutes($panel, $registerPageRoutes, $configuration);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListOrders::route('/'),
            'view' => ViewOrder::route('/{record}'),
        ];
    }
}
