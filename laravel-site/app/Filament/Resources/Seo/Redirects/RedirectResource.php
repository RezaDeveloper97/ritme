<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\Redirects;

use App\Domain\Seo\Redirects\Actions\DeleteRedirects;
use App\Domain\Seo\Redirects\Actions\ImportRedirects;
use App\Domain\Seo\Redirects\Enums\RedirectCode;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Filament\Resources\Seo\Redirects\Pages\CreateRedirect;
use App\Filament\Resources\Seo\Redirects\Pages\EditRedirect;
use App\Filament\Resources\Seo\Redirects\Pages\ListRedirects;
use BackedEnum;
use Closure;
use Filament\Actions\Action;
use Filament\Actions\BulkAction;
use Filament\Actions\EditAction;
use Filament\Facades\Filament;
use Filament\Forms\Components\FileUpload;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Notifications\Notification;
use Filament\Panel;
use Filament\Resources\Resource;
use Filament\Resources\ResourceConfiguration;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Filters\TernaryFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Gate;
use Illuminate\Support\Facades\Route;
use Livewire\Features\SupportFileUploads\TemporaryUploadedFile;
use UnitEnum;

/**
 * Redirect manager (L7-03): list with hits / last hit, create / edit through SaveRedirect (normalisation, chain
 * collapse, loop check → field errors), delete through DeleteRedirects (activity log), CSV import (ImportRedirects)
 * and a streamed CSV export (ExportRedirectsController). Redirects apply only to URLs that would 404 (or legacy
 * URLs), never on top of a live page. Access: RedirectPolicy (SEO manager + super-admin).
 */
final class RedirectResource extends Resource
{
    protected static ?string $model = Redirect::class;

    protected static ?string $slug = 'seo/redirects';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedArrowUturnRight;

    protected static string|UnitEnum|null $navigationGroup = 'سئو';

    protected static ?int $navigationSort = 40;

    protected static ?string $navigationLabel = 'ریدایرکت‌ها';

    protected static ?string $modelLabel = 'ریدایرکت';

    protected static ?string $pluralModelLabel = 'ریدایرکت‌ها';

    protected static ?string $recordTitleAttribute = 'from_path';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Toggle::make('is_regex')->label('عبارت باقاعده (regex)')->live()
                ->helperText('الگو روی کل مسیر (بدون دامنه و کوئری) و بدون حساسیت به حروف بزرگ و کوچک اعمال می‌شود؛ در مقصد از $1، $2 … استفاده کنید.'),
            TextInput::make('from_path')->label(static fn (Get $get): string => $get('is_regex') ? 'الگوی مبدأ' : 'آدرس مبدأ')
                ->required()->maxLength(700)->extraInputAttributes(['dir' => 'ltr'])
                ->placeholder(static fn (Get $get): string => $get('is_regex') ? '/old-blog/(.*)' : '/old-page')
                ->helperText('فقط وقتی اعمال می‌شود که این آدرس صفحه‌ای نداشته باشد (۴۰۴) یا آدرس قدیمی سایت باشد.'),
            Select::make('code')->label('کد وضعیت')->options(RedirectCode::options())->default(RedirectCode::Permanent->value)
                ->required()->live()->native(false),
            TextInput::make('to_url')->label('مقصد')->maxLength(2048)->extraInputAttributes(['dir' => 'ltr'])
                ->placeholder('/blog یا https://example.com/page')
                ->hidden(static fn (Get $get): bool => (int) $get('code') === RedirectCode::Gone->value)
                ->required(static fn (Get $get): bool => (int) $get('code') !== RedirectCode::Gone->value),
            TextInput::make('note')->label('یادداشت')->maxLength(255),
        ]);
    }

    public static function table(Table $table): Table
    {
        $date = static fn (?Carbon $state): ?string => $state ? jdate($state, 'Y/m/d H:i') : null;

        return $table
            ->defaultSort('id', 'desc')
            ->columns([
                TextColumn::make('from_path')->label('مبدأ')->searchable()->limit(60)->tooltip(static fn (Redirect $r): string => $r->from_path)
                    ->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('to_url')->label('مقصد')->searchable()->limit(60)->placeholder('—')->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('code')->label('کد')->badge()
                    ->formatStateUsing(static fn (RedirectCode $state): string => (string) $state->value)
                    ->color(static fn (RedirectCode $state): string => match ($state) {
                        RedirectCode::Permanent => 'success',
                        RedirectCode::Gone => 'danger',
                        default => 'warning',
                    }),
                IconColumn::make('is_regex')->label('regex')->boolean(),
                IconColumn::make('is_auto')->label('خودکار')->boolean()->toggleable(),
                TextColumn::make('hits')->label('بازدید')->numeric()->sortable(),
                TextColumn::make('last_hit_at')->label('آخرین استفاده')->formatStateUsing($date)->placeholder('—')->sortable(),
                TextColumn::make('note')->label('یادداشت')->limit(40)->placeholder('—')->toggleable(isToggledHiddenByDefault: true),
            ])
            ->filters([
                SelectFilter::make('code')->label('کد')->options(RedirectCode::options()),
                TernaryFilter::make('is_regex')->label('regex'),
                TernaryFilter::make('is_auto')->label('خودکار'),
            ])
            ->recordActions([
                EditAction::make(),
                Action::make('delete')->label('حذف')->icon(Heroicon::OutlinedTrash)->color('danger')->requiresConfirmation()
                    ->authorize(static fn (Redirect $record): bool => self::allows('delete', $record))
                    ->action(static fn (Redirect $record, DeleteRedirects $delete) => $delete->handle([$record->id], Filament::auth()->user()))
                    ->successNotificationTitle('حذف شد'),
            ])
            ->toolbarActions([
                BulkAction::make('deleteSelected')->label('حذف انتخاب‌شده‌ها')->icon(Heroicon::OutlinedTrash)->color('danger')
                    ->requiresConfirmation()
                    ->authorize(static fn (): bool => self::allows('deleteAny', Redirect::class))
                    ->action(static function (Collection $records, DeleteRedirects $delete): void {
                        $ids = $records->filter(static fn (mixed $r): bool => $r instanceof Redirect)->map(static fn (Redirect $r): int => $r->id)->values()->all();
                        $delete->handle($ids, Filament::auth()->user());
                    })
                    ->deselectRecordsAfterCompletion()
                    ->successNotificationTitle('حذف شد'),
            ]);
    }

    public static function importAction(): Action
    {
        return Action::make('import')
            ->label('درون‌ریزی CSV')
            ->icon(Heroicon::OutlinedArrowUpTray)
            ->color('gray')
            ->visible(static fn (): bool => self::allows('import', Redirect::class))
            ->authorize(static fn (): bool => self::allows('import', Redirect::class))
            ->modalDescription('ستون‌ها: from, to, code, regex, note — فقط from الزامی است (کد پیش‌فرض ۳۰۱؛ مقصد خالی = ۴۱۰).')
            ->schema([
                FileUpload::make('file')->label('فایل CSV')->required()->storeFiles(false)
                    ->acceptedFileTypes(['text/csv', 'text/plain', 'application/vnd.ms-excel'])->maxSize(2048),
            ])
            ->action(static function (array $data, ImportRedirects $import): void {
                $file = $data['file'] ?? null;
                if (! $file instanceof TemporaryUploadedFile) {
                    return;
                }

                $result = $import->handle($file->getRealPath(), Filament::auth()->user());
                $body = 'جدید: '.fa_digits($result->created).' — به‌روز: '.fa_digits($result->updated).' — ردشده: '.fa_digits($result->skipped);
                if ($result->errors !== []) {
                    $body .= "\n".implode("\n", $result->errors);
                }

                Notification::make()->title('درون‌ریزی انجام شد')->body($body)
                    ->status($result->skipped > 0 ? 'warning' : 'success')->persistent()->send();
            });
    }

    public static function exportAction(): Action
    {
        return Action::make('export')
            ->label('خروجی CSV')
            ->icon(Heroicon::OutlinedArrowDownTray)
            ->color('gray')
            ->visible(static fn (): bool => self::allows('export', Redirect::class))
            ->url(static fn (): string => self::getUrl('export'));
    }

    public static function allows(string $ability, mixed $subject): bool
    {
        return Gate::forUser(Filament::auth()->user())->allows($ability, $subject);
    }

    /**
     * Resource pages plus the `export` download route (same slug prefix, same auth middleware).
     */
    public static function registerRoutes(Panel $panel, ?Closure $registerPageRoutes = null, ?ResourceConfiguration $configuration = null): void
    {
        $registerPageRoutes ??= static function () use ($panel): void {
            Route::get('export', ExportRedirectsController::class)->name('export');

            foreach (self::getPages() as $name => $page) {
                $page->registerRoute($panel)?->name($name);
            }
        };

        parent::registerRoutes($panel, $registerPageRoutes, $configuration);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListRedirects::route('/'),
            'create' => CreateRedirect::route('/create'),
            'edit' => EditRedirect::route('/{record}/edit'),
        ];
    }
}
