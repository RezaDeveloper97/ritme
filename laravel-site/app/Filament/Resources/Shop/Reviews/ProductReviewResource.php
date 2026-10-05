<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Reviews;

use App\Domain\Shop\Catalog\Actions\ModerateProductReviews;
use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Filament\Resources\Shop\Reviews\Pages\ListProductReviews;
use App\Filament\Resources\Shop\ShopAdmin;
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
 * Product review moderation (L6-06): tabs pending / approved / rejected / all, approve / reject one or many
 * (ModerateProductReviews → rating recalculated from approved non-demo reviews, caches bumped, activity log),
 * read-only review view, delete. Access: ProductReviewPolicy (shop managers + super-admins).
 */
final class ProductReviewResource extends Resource
{
    protected static ?string $model = ProductReview::class;

    protected static ?string $slug = 'shop/reviews';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedChatBubbleLeftRight;

    protected static string|UnitEnum|null $navigationGroup = ShopAdmin::NAV_GROUP;

    protected static ?int $navigationSort = 50;

    protected static ?string $navigationLabel = 'نظرهای محصولات';

    protected static ?string $modelLabel = 'نظر';

    protected static ?string $pluralModelLabel = 'نظرهای محصولات';

    public static function getNavigationBadge(): ?string
    {
        $pending = ProductReview::query()->where('status', ReviewStatus::Pending->value)->count();

        return $pending > 0 ? fa_digits($pending) : null;
    }

    public static function getNavigationBadgeTooltip(): string
    {
        return 'در انتظار بررسی';
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with('product:id,title'))
            ->defaultSort('id', 'desc')
            ->columns([
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (ReviewStatus $state): string => $state->label())
                    ->color(static fn (ReviewStatus $state): string => self::statusColor($state)),
                TextColumn::make('product.title')->label('محصول')->searchable()->limit(40),
                TextColumn::make('author_name')->label('نام')->searchable(),
                TextColumn::make('rating')->label('امتیاز')->formatStateUsing(static fn (int $state): string => fa_digits($state).' از ۵'),
                TextColumn::make('body')->label('متن')->limit(80)->wrap()->searchable(),
                IconColumn::make('is_verified_purchase')->label('خرید تأییدشده')->boolean(),
                IconColumn::make('is_demo')->label('نمونه')->boolean(),
                TextColumn::make('created_at')->label('ارسال')->sortable()
                    ->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::date($state)),
            ])
            ->filters([
                SelectFilter::make('product_id')->label('محصول')->relationship('product', 'title')->searchable(),
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
            TextEntry::make('product.title')->label('محصول'),
            TextEntry::make('status')->label('وضعیت')->badge()
                ->formatStateUsing(static fn (ReviewStatus $state): string => $state->label())
                ->color(static fn (ReviewStatus $state): string => self::statusColor($state)),
            TextEntry::make('author_name')->label('نام'),
            TextEntry::make('rating')->label('امتیاز')->formatStateUsing(static fn (int $state): string => fa_digits($state).' از ۵'),
            TextEntry::make('variant_label')->label('تنوع خریداری‌شده')->placeholder('—'),
            TextEntry::make('created_at')->label('ارسال')->formatStateUsing(static fn (mixed $state): ?string => ShopAdmin::date($state)),
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

    private static function moderateAction(string $name, ReviewStatus $status, string $label, Heroicon $icon, string $color): Action
    {
        return Action::make($name)
            ->label($label)
            ->icon($icon)
            ->color($color)
            ->visible(static fn (ProductReview $record): bool => $record->status !== $status)
            ->authorize(static fn (ProductReview $record): bool => ShopAdmin::can('update', $record))
            ->action(static function (ProductReview $record, ModerateProductReviews $moderate) use ($status): void {
                $moderate->handle([$record->id], $status, ShopAdmin::user());
            })
            ->successNotificationTitle('نظر '.$status->label());
    }

    private static function bulkModerateAction(string $name, ReviewStatus $status, string $label, Heroicon $icon): BulkAction
    {
        return BulkAction::make($name)
            ->label($label)
            ->icon($icon)
            ->authorize(static fn (): bool => ShopAdmin::can('viewAny', ProductReview::class))
            ->action(static function (Collection $records, ModerateProductReviews $moderate) use ($status): void {
                $ids = $records->filter(static fn (mixed $r): bool => $r instanceof ProductReview && ShopAdmin::can('update', $r))
                    ->map(static fn (ProductReview $r): int => $r->id)->values()->all();
                $moderate->handle($ids, $status, ShopAdmin::user());
            })
            ->deselectRecordsAfterCompletion()
            ->successNotificationTitle('انجام شد');
    }

    public static function getPages(): array
    {
        return [
            'index' => ListProductReviews::route('/'),
        ];
    }
}
