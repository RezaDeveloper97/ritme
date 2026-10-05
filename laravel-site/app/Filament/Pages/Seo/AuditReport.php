<?php

declare(strict_types=1);

namespace App\Filament\Pages\Seo;

use App\Domain\Seo\Audit\Actions\QueueSeoAudit;
use App\Domain\Seo\Audit\Enums\RunTrigger;
use App\Domain\Seo\Audit\IssueCodes;
use App\Domain\Seo\Audit\Models\AuditRun;
use App\Domain\Seo\Audit\Models\AuditRunIssue;
use App\Domain\Seo\Audit\Queries\AuditHistory;
use App\Domain\Seo\Audit\Severity;
use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use App\Filament\Widgets\Seo\SeoHealthOverview;
use App\Filament\Widgets\Seo\SeoScoreTrend;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Notifications\Notification;
use Filament\Pages\Page;
use Filament\Schemas\Components\EmbeddedTable;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Concerns\InteractsWithTable;
use Filament\Tables\Contracts\HasTable;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use UnitEnum;

/**
 * SEO audit report (L7-05): findings of an audit run (latest by default) with severity / problem / run filters,
 * search by URL or message, a "fix" link per finding (the page's edit form, the redirect manager or the indexing
 * controls) and "run again" (queued; one run at a time). Header: health stats + score trend. Opening with
 * `?code=link.broken` pre-filters by problem (dashboard links). Access: SEO managers + super-admins.
 */
final class AuditReport extends Page implements HasTable
{
    use InteractsWithTable;

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedClipboardDocumentCheck;

    protected static string|UnitEnum|null $navigationGroup = 'سئو';

    protected static ?int $navigationSort = 5;

    protected static ?string $navigationLabel = 'ممیزی سئو';

    protected static ?string $title = 'گزارش ممیزی سئو';

    protected static ?string $slug = 'seo/audit';

    public static function canAccess(): bool
    {
        return AdminAccess::allows(Filament::auth()->user(), [AdminRole::SeoManager]);
    }

    public function mount(): void
    {
        abort_unless(self::canAccess(), 403);

        $code = request()->query('code');
        if (is_string($code) && isset(IssueCodes::LABELS[$code])) {
            $this->tableFilters = ['code' => ['value' => $code]];
        }
    }

    public function getSubheading(): string
    {
        $history = app(AuditHistory::class);
        $active = $history->active();
        $run = $history->latest();

        $parts = [];
        if ($run !== null) {
            $parts[] = 'آخرین ممیزی: '.($run->finished_at === null ? '—' : jdate($run->finished_at, 'j F Y، H:i'))
                .' · '.fa_digits($run->pages_count).' صفحه در '.fa_digits(round(($run->duration_ms ?? 0) / 1000, 1)).' ثانیه'
                .' · '.$run->trigger->label()
                .($run->truncated ? ' · نیمه‌کاره (به سقف '.($run->truncated_by === 'time' ? 'زمان' : 'تعداد صفحه').' رسید)' : '');
        } else {
            $parts[] = 'هنوز ممیزی‌ای ثبت نشده است؛ «اجرای دوباره» را بزنید.';
        }
        if ($active !== null) {
            $parts[] = 'یک ممیزی '.$active->status->label().' است.';
        }

        return implode(' — ', $parts);
    }

    protected function getHeaderWidgets(): array
    {
        return [SeoHealthOverview::class, SeoScoreTrend::class];
    }

    public function getHeaderWidgetsColumns(): int
    {
        return 1;
    }

    protected function getHeaderActions(): array
    {
        return [
            Action::make('rerun')
                ->label('اجرای دوباره')
                ->icon(Heroicon::OutlinedArrowPath)
                ->requiresConfirmation()
                ->modalDescription('همه صفحه‌های سایت در پس‌زمینه بررسی می‌شوند؛ نتیجه چند دقیقه بعد همین‌جا دیده می‌شود.')
                ->disabled(static fn (): bool => app(AuditHistory::class)->active() !== null)
                ->action(static function (): void {
                    $user = Filament::auth()->user();
                    $run = app(QueueSeoAudit::class)->handle(RunTrigger::Admin, is_object($user) && is_numeric($user->getAuthIdentifier()) ? (int) $user->getAuthIdentifier() : null);
                    $run === null
                        ? Notification::make()->title('یک ممیزی در حال اجراست.')->warning()->send()
                        : Notification::make()->title('ممیزی در صف اجرا قرار گرفت.')->body('صفحه را چند دقیقه بعد تازه کنید.')->success()->send();
                }),
        ];
    }

    public function content(Schema $schema): Schema
    {
        return $schema->components([EmbeddedTable::make()]);
    }

    public function table(Table $table): Table
    {
        return $table
            ->query(static fn (): Builder => AuditRunIssue::query())
            ->defaultSort(static fn (Builder $query): Builder => $query
                ->orderByRaw("CASE severity WHEN 'error' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END")
                ->orderBy('path')
                ->orderBy('id'))
            ->paginated([25, 50, 100])
            ->emptyStateHeading('موردی پیدا نشد.')
            ->emptyStateDescription('یا ممیزی هنوز اجرا نشده، یا این فیلترها مشکلی ندارند.')
            ->columns([
                TextColumn::make('severity')
                    ->label('شدت')
                    ->badge()
                    ->formatStateUsing(static fn (Severity $state): string => $state->label())
                    ->color(static fn (Severity $state): string => $state->color()),
                TextColumn::make('code')
                    ->label('مشکل')
                    ->formatStateUsing(static fn (string $state): string => IssueCodes::label($state))
                    ->description(static fn (AuditRunIssue $record): string => $record->code),
                TextColumn::make('path')
                    ->label('نشانی')
                    ->searchable()
                    ->limit(60)
                    ->tooltip(static fn (AuditRunIssue $record): string => $record->path)
                    ->url(static fn (AuditRunIssue $record): string => url($record->path))
                    ->openUrlInNewTab(),
                TextColumn::make('message')
                    ->label('شرح')
                    ->searchable()
                    ->wrap(),
            ])
            ->filters([
                SelectFilter::make('run_id')
                    ->label('ممیزی')
                    ->options(static fn (): array => app(AuditHistory::class)->trend(30)->reverse()
                        ->mapWithKeys(static fn (AuditRun $run): array => [$run->id => '#'.fa_digits($run->id).' — '
                            .($run->finished_at === null ? '—' : jdate($run->finished_at, 'j F Y، H:i')).' — امتیاز '.fa_digits((int) $run->score)])
                        ->all())
                    ->placeholder('آخرین ممیزی')
                    ->query(static function (Builder $query, array $data): Builder {
                        $id = is_numeric($data['value'] ?? null) ? (int) $data['value'] : app(AuditHistory::class)->latest()?->id;

                        return $query->where('run_id', $id ?? 0);
                    }),
                SelectFilter::make('severity')
                    ->label('شدت')
                    ->options(Severity::options()),
                SelectFilter::make('code')
                    ->label('مشکل')
                    ->options(IssueCodes::options())
                    ->searchable(),
            ])
            ->recordActions([
                Action::make('fix')
                    ->label('رفع')
                    ->icon(Heroicon::OutlinedWrenchScrewdriver)
                    ->url(static fn (AuditRunIssue $record): ?string => $record->fix_url)
                    ->visible(static fn (AuditRunIssue $record): bool => $record->fix_url !== null),
            ]);
    }
}
