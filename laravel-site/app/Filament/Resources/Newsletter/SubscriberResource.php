<?php

declare(strict_types=1);

namespace App\Filament\Resources\Newsletter;

use App\Domain\Newsletter\Actions\ExportSubscribers;
use App\Domain\Newsletter\Enums\SubscriptionStatus;
use App\Domain\Newsletter\Models\Subscriber;
use App\Filament\Resources\Newsletter\Pages\ListSubscribers;
use BackedEnum;
use Closure;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Panel;
use Filament\Resources\Resource;
use Filament\Resources\ResourceConfiguration;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Gate;
use Illuminate\Support\Facades\Route;
use UnitEnum;

/**
 * Newsletter subscribers (L4-05b): read-only list with email search and status filter, plus a streamed CSV export of
 * the filtered list (ExportSubscribersController → Domain ExportSubscribers). Access: SubscriberPolicy.
 */
final class SubscriberResource extends Resource
{
    protected static ?string $model = Subscriber::class;

    protected static ?string $slug = 'newsletter/subscribers';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedEnvelope;

    protected static string|UnitEnum|null $navigationGroup = 'مجله';

    protected static ?int $navigationSort = 50;

    protected static ?string $modelLabel = 'مشترک خبرنامه';

    protected static ?string $pluralModelLabel = 'مشترکان خبرنامه';

    protected static ?string $recordTitleAttribute = 'email';

    public static function table(Table $table): Table
    {
        $date = static fn (?Carbon $state): ?string => $state ? jdate($state, 'Y/m/d H:i') : null;

        return $table
            ->defaultSort('id', 'desc')
            ->columns([
                TextColumn::make('email')->label('ایمیل')->searchable()->copyable(),
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->state(static fn (Subscriber $record): string => $record->status()->label())
                    ->color(static fn (Subscriber $record): string => match ($record->status()) {
                        SubscriptionStatus::Active => 'success',
                        SubscriptionStatus::Pending => 'warning',
                        SubscriptionStatus::Unsubscribed => 'gray',
                    }),
                TextColumn::make('source')->label('منبع')->placeholder('—')->toggleable(isToggledHiddenByDefault: true),
                TextColumn::make('consent_at')->label('ثبت‌نام')->formatStateUsing($date)->sortable(),
                TextColumn::make('confirmed_at')->label('تأیید')->formatStateUsing($date)->placeholder('—')->sortable(),
                TextColumn::make('unsubscribed_at')->label('لغو')->formatStateUsing($date)->placeholder('—')->sortable(),
            ])
            ->filters([
                SelectFilter::make('status')->label('وضعیت')
                    ->options(collect(SubscriptionStatus::cases())->mapWithKeys(static fn (SubscriptionStatus $s): array => [$s->value => $s->label()])->all())
                    ->query(static function (Builder $query, array $data): Builder {
                        $status = SubscriptionStatus::tryFrom((string) ($data['value'] ?? ''));

                        return $status === null ? $query : ExportSubscribers::whereStatus($query, $status);
                    }),
            ]);
    }

    /**
     * Header action on the list: a link to the streamed export carrying the current status filter and search.
     */
    public static function exportAction(): Action
    {
        return Action::make('export')
            ->label('خروجی CSV')
            ->icon(Heroicon::OutlinedArrowDownTray)
            ->color('gray')
            ->visible(static fn (): bool => Gate::forUser(Filament::auth()->user())->allows('export', Subscriber::class))
            ->url(static fn (ListSubscribers $livewire): string => self::getUrl('export', array_filter([
                'status' => $livewire->exportStatus(),
                'search' => $livewire->exportSearch(),
            ])));
    }

    /**
     * Resource pages plus the `export` download route (same slug prefix, same auth middleware).
     */
    public static function registerRoutes(Panel $panel, ?Closure $registerPageRoutes = null, ?ResourceConfiguration $configuration = null): void
    {
        $registerPageRoutes ??= static function () use ($panel): void {
            foreach (self::getPages() as $name => $page) {
                $page->registerRoute($panel)?->name($name);
            }

            Route::get('export', ExportSubscribersController::class)->name('export');
        };

        parent::registerRoutes($panel, $registerPageRoutes, $configuration);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListSubscribers::route('/'),
        ];
    }
}
