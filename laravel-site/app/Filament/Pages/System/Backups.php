<?php

declare(strict_types=1);

namespace App\Filament\Pages\System;

use App\Filament\Auth\AdminAccess;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Pages\Page;
use Filament\Pages\PageConfiguration;
use Filament\Panel;
use Filament\Schemas\Components\EmbeddedTable;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Concerns\InteractsWithTable;
use Filament\Tables\Contracts\HasTable;
use Filament\Tables\Table;
use Illuminate\Support\Facades\Route;
use Spatie\Backup\BackupDestination\Backup;
use Spatie\Backup\BackupDestination\BackupDestination;
use Spatie\Backup\Tasks\Monitor\HealthChecks\MaximumAgeInDays;
use Throwable;
use UnitEnum;

/**
 * Backups (L10-02): the zips spatie/laravel-backup writes every night (config/backup.php — database + media), newest
 * first, with a download link per file. **Super-admins only**: a backup holds every order, booking and contact message.
 * Downloads go through BackupDownloadController (registered next to this page, so it sits behind the panel's auth,
 * inactivity-timeout and MFA middleware), are streamed (no memory spike on big archives) and are logged in the
 * activity log. Creating and deleting backups stays with the scheduler (backup:run / backup:clean): a web request on
 * shared hosting would time out on a large media folder.
 */
final class Backups extends Page implements HasTable
{
    use InteractsWithTable;

    public const DOWNLOAD_ROUTE = 'system.backups.download';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedArchiveBox;

    protected static string|UnitEnum|null $navigationGroup = 'سیستم';

    protected static ?int $navigationSort = 95;

    protected static ?string $navigationLabel = 'پشتیبان‌ها';

    protected static ?string $title = 'پشتیبان‌ها';

    protected static ?string $slug = 'system/backups';

    public static function canAccess(): bool
    {
        return AdminAccess::allows(Filament::auth()->user());
    }

    public static function routes(Panel $panel, ?PageConfiguration $configuration = null): void
    {
        parent::routes($panel, $configuration);

        Route::get(self::getRoutePath($panel).'/{disk}/{file}', BackupDownloadController::class)
            ->where('disk', '[A-Za-z0-9_-]+')
            ->where('file', BackupDownloadController::FILE_PATTERN)
            ->middleware(self::getRouteMiddleware($panel))
            ->name('system.backups.download');
    }

    public static function downloadUrl(string $disk, string $file): string
    {
        return route(Filament::getPanel('admin')->generateRouteName('pages.'.self::DOWNLOAD_ROUTE), ['disk' => $disk, 'file' => $file]);
    }

    public function mount(): void
    {
        abort_unless(self::canAccess(), 403);
    }

    public function getSubheading(): string
    {
        $maxDays = self::maxAgeDays();
        $parts = [];
        foreach (self::destinations() as $destination) {
            try {
                $newest = $destination->newestBackup();
                $parts[] = $destination->diskName().': '.($newest === null
                    ? 'هنوز پشتیبانی ساخته نشده'
                    : 'آخرین '.jdate($newest->date(), 'j F Y، H:i').' · '.fa_digits($destination->backups()->count()).' فایل · '
                        .fa_digits(number_format($destination->usedStorage() / 1048576, 1)).' مگابایت'
                        .($newest->date()->lt(now()->subDays($maxDays)) ? ' · قدیمی‌تر از '.fa_digits($maxDays).' روز!' : ''));
            } catch (Throwable) {
                $parts[] = $destination->diskName().': در دسترس نیست';
            }
        }

        return implode(' — ', $parts).' — ساخت خودکار هر شب ساعت ۲:۱۰؛ فایل‌ها اطلاعات شخصی دارند، فقط در جای امن نگه دارید.';
    }

    public function content(Schema $schema): Schema
    {
        return $schema->components([EmbeddedTable::make()]);
    }

    public function table(Table $table): Table
    {
        return $table
            ->records(static function (): array {
                $records = [];
                foreach (self::destinations() as $destination) {
                    try {
                        foreach ($destination->backups() as $backup) {
                            /** @var Backup $backup */
                            $file = basename($backup->path());
                            $records[$destination->diskName().'/'.$file] = [
                                'disk' => $destination->diskName(),
                                'file' => $file,
                                'date' => $backup->date(),
                                'size' => $backup->sizeInBytes(),
                            ];
                        }
                    } catch (Throwable) {
                        continue;
                    }
                }

                return $records;
            })
            ->paginated(false)
            ->emptyStateHeading('هنوز پشتیبانی ساخته نشده است.')
            ->emptyStateDescription('زمان‌بند هر شب پشتیبان می‌گیرد؛ برای ساخت فوری در Terminal هاست: php artisan backup:run')
            ->columns([
                TextColumn::make('date')
                    ->label('زمان')
                    ->formatStateUsing(static fn (mixed $state): string => $state instanceof \DateTimeInterface ? jdate($state, 'j F Y، H:i') : '—'),
                TextColumn::make('file')
                    ->label('فایل'),
                TextColumn::make('size')
                    ->label('حجم')
                    ->formatStateUsing(static fn (mixed $state): string => fa_digits(number_format((float) $state / 1048576, 1)).' مگابایت'),
                TextColumn::make('disk')
                    ->label('مقصد'),
            ])
            ->recordActions([
                Action::make('download')
                    ->label('دانلود')
                    ->icon(Heroicon::OutlinedArrowDownTray)
                    ->url(static fn (array $record): string => self::downloadUrl((string) $record['disk'], (string) $record['file'])),
            ]);
    }

    /** @return list<BackupDestination> */
    public static function destinations(): array
    {
        $name = (string) config('backup.backup.name');
        $destinations = [];
        foreach ((array) config('backup.backup.destination.disks', []) as $disk) {
            $destinations[] = BackupDestination::create((string) $disk, $name);
        }

        return $destinations;
    }

    private static function maxAgeDays(): int
    {
        $monitor = config('backup.monitor_backups.0.health_checks');

        return is_array($monitor) ? max(1, (int) ($monitor[MaximumAgeInDays::class] ?? 1)) : 1;
    }
}
