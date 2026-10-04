<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Reviews;

use App\Domain\Directory\Actions\ModeratePlaceReviews;
use App\Domain\Directory\Enums\ReviewAspect;
use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\PlaceReview;
use App\Filament\Resources\Directory\DirectoryAdmin;
use App\Filament\Resources\Directory\Reviews\Pages\ListPlaceReviews;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Actions\BulkAction;
use Filament\Actions\BulkActionGroup;
use Filament\Actions\DeleteAction;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\ViewAction;
use Filament\Infolists\Components\TextEntry;
use Filament\Resources\Resource;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
use UnitEnum;

/**
 * Review moderation queue (L5-06): tabs pending / approved / rejected / all, approve / reject one or many
 * (ModeratePlaceReviews → rating recalculated, caches bumped, activity log), read-only review view, delete.
 * Access: PlaceReviewPolicy (directory-manager + super-admin).
 */
final class PlaceReviewResource extends Resource
{
    protected static ?string $model = PlaceReview::class;

    protected static ?string $slug = 'directory/reviews';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedChatBubbleLeftRight;

    protected static string|UnitEnum|null $navigationGroup = DirectoryAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 60;

    protected static ?string $navigationLabel = 'نظرها (بررسی)';

    protected static ?string $modelLabel = 'نظر';

    protected static ?string $pluralModelLabel = 'نظرهای مجموعه‌ها';

    public static function getNavigationBadge(): ?string
    {
        $pending = PlaceReview::query()->where('status', ReviewStatus::Pending->value)->count();

        return $pending > 0 ? fa_digits($pending) : null;
    }

    public static function getNavigationBadgeTooltip(): string
    {
        return 'در انتظار بررسی';
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with('place:id,name'))
            ->defaultSort('id', 'desc')
            ->columns([
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (ReviewStatus $state): string => $state->label())
                    ->color(static fn (ReviewStatus $state): string => self::statusColor($state)),
                TextColumn::make('place.name')->label('مجموعه')->searchable(),
                TextColumn::make('author_name')->label('نام')->searchable(),
                TextColumn::make('rating')->label('امتیاز')->formatStateUsing(static fn (int $state): string => fa_digits($state).' از ۵'),
                TextColumn::make('body')->label('متن')->limit(80)->wrap()->searchable(),
                IconColumn::make('is_demo')->label('نمونه')->boolean(),
                TextColumn::make('created_at')->label('ارسال')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
            ])
            ->filters([
                SelectFilter::make('place_id')->label('مجموعه')->relationship('place', 'name')->searchable(),
                SelectFilter::make('rating')->label('امتیاز')->options([1 => '۱', 2 => '۲', 3 => '۳', 4 => '۴', 5 => '۵']),
            ])
            ->recordActions([
                ViewAction::make(),
                self::moderateAction('approve', ReviewStatus::Approved, 'تأیید', Heroicon::OutlinedCheck, 'success'),
                self::moderateAction('reject', ReviewStatus::Rejected, 'رد', Heroicon::OutlinedXMark, 'danger'),
                DeleteAction::make(),
            ])
            ->toolbarActions([
                BulkActionGroup::make([
                    self::bulkModerateAction('approveBulk', ReviewStatus::Approved, 'تأیید انتخاب‌شده‌ها', Heroicon::OutlinedCheck),
                    self::bulkModerateAction('rejectBulk', ReviewStatus::Rejected, 'رد انتخاب‌شده‌ها', Heroicon::OutlinedXMark),
                    DeleteBulkAction::make(),
                ]),
            ]);
    }

    public static function infolist(Schema $schema): Schema
    {
        return $schema->components([
            TextEntry::make('place.name')->label('مجموعه'),
            TextEntry::make('status')->label('وضعیت')->badge()
                ->formatStateUsing(static fn (ReviewStatus $state): string => $state->label())
                ->color(static fn (ReviewStatus $state): string => self::statusColor($state)),
            TextEntry::make('author_name')->label('نام'),
            TextEntry::make('rating')->label('امتیاز')->formatStateUsing(static fn (int $state): string => fa_digits($state).' از ۵'),
            TextEntry::make('aspects')->label('امتیاز جزئی')->placeholder('—')
                ->state(static fn (PlaceReview $record): ?string => self::aspects($record)),
            TextEntry::make('created_at')->label('ارسال')->formatStateUsing(static fn (mixed $state): ?string => DirectoryAdmin::date($state)),
            TextEntry::make('body')->label('متن')->columnSpanFull()
                ->formatStateUsing(static fn (string $state): string => e($state))
                ->html()->extraAttributes(['class' => 'whitespace-pre-line']),
        ]);
    }

    public static function statusColor(ReviewStatus $status): string
    {
        return match ($status) {
            ReviewStatus::Pending => 'warning',
            ReviewStatus::Approved => 'success',
            ReviewStatus::Rejected => 'gray',
        };
    }

    private static function aspects(PlaceReview $review): ?string
    {
        $parts = [];
        foreach ($review->aspects ?? [] as $key => $score) {
            $aspect = ReviewAspect::tryFrom((string) $key);
            if ($aspect !== null) {
                $parts[] = $aspect->label().': '.fa_digits((int) $score);
            }
        }

        return $parts === [] ? null : implode('، ', $parts);
    }

    private static function moderateAction(string $name, ReviewStatus $status, string $label, Heroicon $icon, string $color): Action
    {
        return Action::make($name)
            ->label($label)
            ->icon($icon)
            ->color($color)
            ->visible(static fn (PlaceReview $record): bool => $record->status !== $status)
            ->authorize(static fn (PlaceReview $record): bool => DirectoryAdmin::can('update', $record))
            ->action(static function (PlaceReview $record, ModeratePlaceReviews $moderate) use ($status): void {
                $moderate->handle([$record->id], $status, DirectoryAdmin::user());
            })
            ->successNotificationTitle('نظر '.$status->label());
    }

    private static function bulkModerateAction(string $name, ReviewStatus $status, string $label, Heroicon $icon): BulkAction
    {
        return BulkAction::make($name)
            ->label($label)
            ->icon($icon)
            ->authorize(static fn (): bool => DirectoryAdmin::can('viewAny', PlaceReview::class))
            ->action(static function (Collection $records, ModeratePlaceReviews $moderate) use ($status): void {
                $ids = $records->filter(static fn (mixed $r): bool => $r instanceof PlaceReview && DirectoryAdmin::can('update', $r))
                    ->map(static fn (PlaceReview $r): int => $r->id)->values()->all();
                $moderate->handle($ids, $status, DirectoryAdmin::user());
            })
            ->deselectRecordsAfterCompletion()
            ->successNotificationTitle('انجام شد');
    }

    public static function getPages(): array
    {
        return [
            'index' => ListPlaceReviews::route('/'),
        ];
    }
}
