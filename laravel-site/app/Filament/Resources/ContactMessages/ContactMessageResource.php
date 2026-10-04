<?php

declare(strict_types=1);

namespace App\Filament\Resources\ContactMessages;

use App\Domain\Contact\Actions\ChangeContactMessageStatus;
use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Models\ContactMessage;
use App\Filament\Resources\ContactMessages\Pages\ListContactMessages;
use App\Filament\Resources\ContactMessages\Pages\ViewContactMessage;
use BackedEnum;
use Closure;
use Filament\Actions\Action;
use Filament\Actions\BulkAction;
use Filament\Actions\BulkActionGroup;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\ViewAction;
use Filament\Facades\Filament;
use Filament\Infolists\Components\TextEntry;
use Filament\Panel;
use Filament\Resources\Resource;
use Filament\Resources\ResourceConfiguration;
use Filament\Schemas\Schema;
use Filament\Support\Enums\FontWeight;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Gate;
use Illuminate\Support\Facades\Route;
use UnitEnum;

/**
 * Contact inbox (L3-10): messages from the public form in tabs (unread / read / archived / all) with topic filter and
 * search, a read-only message view (opening it marks it read), reply by mail (`mailto:` with a prefilled subject) or
 * phone (`tel:`), read/unread/archive actions (single + bulk, activity log `contact`), delete, and a streamed CSV
 * export of the current tab/filter/search. Access: ContactMessagePolicy (support + super-admin).
 */
final class ContactMessageResource extends Resource
{
    protected static ?string $model = ContactMessage::class;

    protected static ?string $slug = 'contact-messages';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedInbox;

    protected static string|UnitEnum|null $navigationGroup = 'پشتیبانی';

    protected static ?int $navigationSort = 10;

    protected static ?string $navigationLabel = 'پیام‌های تماس';

    protected static ?string $modelLabel = 'پیام تماس';

    protected static ?string $pluralModelLabel = 'پیام‌های تماس';

    protected static ?string $recordTitleAttribute = 'name';

    public static function getNavigationBadge(): ?string
    {
        $unread = ContactMessage::query()->where('status', ContactMessageStatus::Unread->value)->count();

        return $unread > 0 ? fa_digits($unread) : null;
    }

    public static function getNavigationBadgeTooltip(): string
    {
        return 'خوانده‌نشده';
    }

    public static function table(Table $table): Table
    {
        $date = static fn (?Carbon $state): ?string => $state ? jdate($state, 'Y/m/d H:i') : null;
        $unread = static fn (ContactMessage $record): bool => $record->status === ContactMessageStatus::Unread;

        return $table
            ->defaultSort('id', 'desc')
            ->recordUrl(static fn (ContactMessage $record): string => self::getUrl('view', ['record' => $record]))
            ->columns([
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (ContactMessageStatus $state): string => $state->label())
                    ->color(static fn (ContactMessageStatus $state): string => $state->color()),
                TextColumn::make('topic')->label('موضوع')
                    ->formatStateUsing(static fn (ContactTopic $state): string => $state->label()),
                TextColumn::make('name')->label('نام')->searchable()
                    ->weight(static fn (ContactMessage $record): ?FontWeight => $unread($record) ? FontWeight::Bold : null),
                TextColumn::make('reply')->label('راه پاسخ')
                    ->state(static fn (ContactMessage $record): string => $record->replyTo())
                    ->searchable(['email', 'phone'])->copyable(),
                TextColumn::make('message')->label('پیام')->limit(60)->searchable()->wrap(),
                TextColumn::make('created_at')->label('دریافت')->formatStateUsing($date)->sortable(),
            ])
            ->filters([
                SelectFilter::make('topic')->label('موضوع')->options(ContactTopic::options()),
            ])
            ->recordActions([
                ViewAction::make(),
                self::statusAction('markRead', ContactMessageStatus::Read, 'خوانده شد', Heroicon::OutlinedEnvelopeOpen)
                    ->visible(static fn (ContactMessage $record): bool => $record->status === ContactMessageStatus::Unread),
                self::statusAction('markUnread', ContactMessageStatus::Unread, 'خوانده‌نشده', Heroicon::OutlinedEnvelope)
                    ->visible(static fn (ContactMessage $record): bool => $record->status === ContactMessageStatus::Read),
                self::statusAction('archive', ContactMessageStatus::Archived, 'بایگانی', Heroicon::OutlinedArchiveBox)
                    ->visible(static fn (ContactMessage $record): bool => $record->status !== ContactMessageStatus::Archived),
                self::statusAction('unarchive', ContactMessageStatus::Read, 'خروج از بایگانی', Heroicon::OutlinedArrowUturnLeft)
                    ->visible(static fn (ContactMessage $record): bool => $record->status === ContactMessageStatus::Archived),
            ])
            ->toolbarActions([
                BulkActionGroup::make([
                    self::bulkStatusAction('markReadBulk', ContactMessageStatus::Read, 'علامت خوانده‌شده', Heroicon::OutlinedEnvelopeOpen),
                    self::bulkStatusAction('markUnreadBulk', ContactMessageStatus::Unread, 'علامت خوانده‌نشده', Heroicon::OutlinedEnvelope),
                    self::bulkStatusAction('archiveBulk', ContactMessageStatus::Archived, 'بایگانی', Heroicon::OutlinedArchiveBox),
                    DeleteBulkAction::make(),
                ]),
            ]);
    }

    public static function infolist(Schema $schema): Schema
    {
        return $schema->components([
            TextEntry::make('topic')->label('موضوع')->formatStateUsing(static fn (ContactTopic $state): string => $state->label()),
            TextEntry::make('status')->label('وضعیت')->badge()
                ->formatStateUsing(static fn (ContactMessageStatus $state): string => $state->label())
                ->color(static fn (ContactMessageStatus $state): string => $state->color()),
            TextEntry::make('name')->label('نام'),
            TextEntry::make('email')->label('ایمیل')->placeholder('—')->copyable(),
            TextEntry::make('phone')->label('تلفن')->placeholder('—')->copyable(),
            TextEntry::make('created_at')->label('دریافت')->formatStateUsing(static fn (?Carbon $state): ?string => $state ? jdate($state, 'Y/m/d H:i') : null),
            TextEntry::make('message')->label('پیام')->columnSpanFull()
                ->formatStateUsing(static fn (string $state): string => e($state))
                ->html()->extraAttributes(['class' => 'whitespace-pre-line']),
        ]);
    }

    /**
     * Reply link for the view page: `mailto:` (subject prefilled with the topic) when the sender left an email,
     * `tel:` when they left a phone number.
     */
    public static function replyUrl(ContactMessage $record): ?string
    {
        if ($record->email !== null) {
            return 'mailto:'.$record->email.'?subject='.rawurlencode('پاسخ ریتمی: '.$record->topic->label());
        }

        return $record->phone !== null ? 'tel:'.$record->phone : null;
    }

    public static function statusAction(string $name, ContactMessageStatus $status, string $label, Heroicon $icon): Action
    {
        return Action::make($name)
            ->label($label)
            ->icon($icon)
            ->color('gray')
            ->authorize(static fn (ContactMessage $record): bool => Gate::forUser(Filament::auth()->user())->allows('update', $record))
            ->action(static function (ContactMessage $record, ChangeContactMessageStatus $change) use ($status): void {
                $change->handle($record, $status, Filament::auth()->user());
            })
            ->successNotificationTitle('انجام شد');
    }

    /**
     * Header action on the list: a link to the streamed export carrying the current tab, topic filter and search.
     */
    public static function exportAction(): Action
    {
        return Action::make('export')
            ->label('خروجی CSV')
            ->icon(Heroicon::OutlinedArrowDownTray)
            ->color('gray')
            ->visible(static fn (): bool => Gate::forUser(Filament::auth()->user())->allows('export', ContactMessage::class))
            ->url(static fn (ListContactMessages $livewire): string => self::getUrl('export', $livewire->exportParameters()));
    }

    /**
     * Resource pages plus the `export` download route (same slug prefix, same auth middleware).
     */
    public static function registerRoutes(Panel $panel, ?Closure $registerPageRoutes = null, ?ResourceConfiguration $configuration = null): void
    {
        $registerPageRoutes ??= static function () use ($panel): void {
            Route::get('export', ExportContactMessagesController::class)->name('export');

            foreach (self::getPages() as $name => $page) {
                $page->registerRoute($panel)?->name($name);
            }
        };

        parent::registerRoutes($panel, $registerPageRoutes, $configuration);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListContactMessages::route('/'),
            'view' => ViewContactMessage::route('/{record}'),
        ];
    }

    private static function bulkStatusAction(string $name, ContactMessageStatus $status, string $label, Heroicon $icon): BulkAction
    {
        return BulkAction::make($name)
            ->label($label)
            ->icon($icon)
            ->authorize(static fn (): bool => Gate::forUser(Filament::auth()->user())->allows('viewAny', ContactMessage::class))
            ->action(static function (Collection $records, ChangeContactMessageStatus $change) use ($status): void {
                $user = Filament::auth()->user();
                foreach ($records as $record) {
                    if ($record instanceof ContactMessage && Gate::forUser($user)->allows('update', $record)) {
                        $change->handle($record, $status, $user);
                    }
                }
            })
            ->deselectRecordsAfterCompletion()
            ->successNotificationTitle('انجام شد');
    }
}
