<?php

declare(strict_types=1);

namespace App\Filament\Resources\Media;

use App\Domain\Media\Actions\DeleteUnusedMedia;
use App\Domain\Media\Actions\FindMediaUsages;
use App\Domain\Media\Models\Media;
use App\Filament\Forms\Components\FocalPointPicker;
use App\Filament\Resources\Media\Pages\EditMedia;
use App\Filament\Resources\Media\Pages\ListMedia;
use BackedEnum;
use Filament\Actions\BulkAction;
use Filament\Actions\EditAction;
use Filament\Facades\Filament;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Infolists\Components\TextEntry;
use Filament\Notifications\Notification;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\ImageColumn;
use Filament\Tables\Columns\Layout\Stack;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\TernaryFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
use UnitEnum;

/**
 * Media library (thin delivery over the Media actions): grid with thumbs and an alt-text warning, multi-upload
 * (ListMedia), alt/title/caption + focal point editing (EditMedia → UpdateMediaDetails), variants with savings,
 * regenerate, find usages and delete-unused-only.
 */
final class MediaResource extends Resource
{
    protected static ?string $model = Media::class;

    protected static ?string $slug = 'media';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedPhoto;

    protected static string|UnitEnum|null $navigationGroup = 'محتوا';

    protected static ?int $navigationSort = 50;

    protected static ?string $modelLabel = 'رسانه';

    protected static ?string $pluralModelLabel = 'کتابخانه رسانه';

    protected static ?string $recordTitleAttribute = 'original_name';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(3)->components([
            Section::make('تصویر')
                ->columnSpan(2)
                ->schema([
                    FocalPointPicker::make('focal')
                        ->imageUrl(static fn (?Media $record): ?string => $record === null ? null : MediaPresenter::previewUrl($record)),
                    TextInput::make('alt')
                        ->label('متن جایگزین (alt)')
                        ->required()
                        ->maxLength(255)
                        ->helperText('آنچه در تصویر دیده می‌شود را کوتاه و دقیق بنویسید؛ برای دسترس‌پذیری و سئو لازم است.'),
                    TextInput::make('title')->label('عنوان')->maxLength(255),
                    Textarea::make('caption')->label('زیرنویس')->rows(3)->maxLength(1000),
                ]),
            Section::make('فایل')
                ->columnSpan(1)
                ->schema([
                    Grid::make(1)->schema([
                        TextEntry::make('original_name')->label('نام فایل')->placeholder('—'),
                        TextEntry::make('mime')->label('نوع'),
                        TextEntry::make('dimensions')->label('ابعاد')
                            ->state(static fn (Media $record): string => MediaPresenter::dimensions($record)),
                        TextEntry::make('size')->label('حجم نسخه اصلی')
                            ->state(static fn (Media $record): string => MediaPresenter::humanSize($record->size)),
                        TextEntry::make('savings')->label('صرفه‌جویی')
                            ->state(static fn (Media $record): string => MediaPresenter::savingsLabel(MediaPresenter::savingsPercent($record))),
                        TextEntry::make('status')->label('وضعیت')->badge()
                            ->state(static fn (Media $record): string => MediaPresenter::status($record))
                            ->color(static fn (Media $record): string => $record->optimized_at === null ? 'warning' : 'success'),
                        TextEntry::make('variants_list')->label('نسخه‌ها')
                            ->state(static fn (Media $record): array => MediaPresenter::variantLines($record))
                            ->listWithLineBreaks()
                            ->placeholder('هنوز نسخه‌ای ساخته نشده است.'),
                        TextEntry::make('usages')->label('استفاده‌ها')
                            ->state(static fn (Media $record): array => app(FindMediaUsages::class)->handle([$record->id])[$record->id] ?? [])
                            ->listWithLineBreaks()
                            ->placeholder('جایی استفاده نشده است.'),
                    ]),
                ])
                ->visible(static fn (?Media $record): bool => $record !== null),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->defaultSort('id', 'desc')
            ->contentGrid(['default' => 2, 'md' => 3, 'xl' => 5])
            ->paginated([20, 40, 80])
            ->columns([
                Stack::make([
                    ImageColumn::make('thumb')
                        ->label('پیش‌نمایش')
                        ->disk(static fn (): string => (string) config('media.disk', 'public'))
                        ->state(static fn (Media $record): string => MediaPresenter::thumbPath($record))
                        ->checkFileExistence(false)
                        ->imageHeight(160)
                        ->extraImgAttributes(static fn (Media $record): array => ['alt' => (string) ($record->alt ?? ''), 'loading' => 'lazy', 'class' => 'w-full rounded-lg object-cover']),
                    TextColumn::make('original_name')
                        ->label('نام فایل')
                        ->searchable(['original_name', 'alt', 'title'])
                        ->limit(32)
                        ->weight('medium'),
                    TextColumn::make('alt')
                        ->label('متن جایگزین')
                        ->badge()
                        ->state(static fn (Media $record): string => filled($record->alt) ? 'دارای متن جایگزین' : 'بدون متن جایگزین')
                        ->color(static fn (Media $record): string => filled($record->alt) ? 'success' : 'warning')
                        ->icon(static fn (Media $record): ?Heroicon => filled($record->alt) ? null : Heroicon::OutlinedExclamationTriangle),
                    TextColumn::make('meta')
                        ->label('جزئیات')
                        ->color('gray')
                        ->size('sm')
                        ->state(static fn (Media $record): string => implode(' · ', array_filter([
                            MediaPresenter::dimensions($record),
                            MediaPresenter::humanSize($record->size),
                            $record->optimized_at === null ? 'در صف بهینه‌سازی' : MediaPresenter::savingsLabel(MediaPresenter::savingsPercent($record)),
                        ]))),
                ])->space(2),
            ])
            ->filters([
                TernaryFilter::make('has_alt')
                    ->label('متن جایگزین')
                    ->placeholder('همه')
                    ->trueLabel('دارای متن جایگزین')
                    ->falseLabel('بدون متن جایگزین')
                    ->queries(
                        true: static fn (Builder $query): Builder => $query->whereNotNull('alt')->where('alt', '!=', ''),
                        false: static fn (Builder $query): Builder => $query->where(static fn (Builder $q): Builder => $q->whereNull('alt')->orWhere('alt', '')),
                    ),
            ])
            ->recordActions([EditAction::make()->label('ویرایش')])
            ->toolbarActions([
                BulkAction::make('deleteUnused')
                    ->label('حذف موارد استفاده‌نشده')
                    ->icon(Heroicon::OutlinedTrash)
                    ->color('danger')
                    ->requiresConfirmation()
                    ->modalDescription('فقط تصاویری حذف می‌شوند که در هیچ جای سایت استفاده نشده‌اند. این کار برگشت‌پذیر نیست.')
                    ->authorize(static fn (): bool => Filament::auth()->user()?->can('deleteAny', Media::class) ?? false)
                    ->action(static function (Collection $records): void {
                        $result = app(DeleteUnusedMedia::class)->handle(array_values(array_map('intval', $records->modelKeys())));
                        self::logDeleted($result['deleted']);

                        $notification = Notification::make()->title(sprintf('%d تصویر حذف شد.', count($result['deleted'])));
                        if ($result['kept'] !== []) {
                            $notification->warning()->body(sprintf('%d تصویر چون در سایت استفاده شده، حذف نشد.', count($result['kept'])));
                        } else {
                            $notification->success();
                        }
                        $notification->send();
                    })
                    ->deselectRecordsAfterCompletion(),
            ]);
    }

    /**
     * @param  list<int>  $ids
     */
    public static function logDeleted(array $ids): void
    {
        if ($ids === []) {
            return;
        }

        activity('media')
            ->causedBy(Filament::auth()->user())
            ->event('deleted')
            ->withProperties(['ids' => $ids])
            ->log('media.deleted');
    }

    public static function getPages(): array
    {
        return [
            'index' => ListMedia::route('/'),
            'edit' => EditMedia::route('/{record}/edit'),
        ];
    }
}
