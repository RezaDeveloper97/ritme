<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Seo;

use App\Domain\Seo\Audit\Queries\TopNotFound;
use App\Domain\Seo\Redirects\Models\NotFoundLog;
use App\Filament\Resources\Seo\NotFoundLogs\NotFoundLogResource;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Filament\Widgets\TableWidget;
use Illuminate\Database\Eloquent\Builder;

/**
 * Most visited missing pages of the last 30 days (404 monitor, L7-03), people before bots.
 */
final class SeoTopNotFound extends TableWidget
{
    use SeoWidgetAccess;

    protected static ?int $sort = 13;

    public function table(Table $table): Table
    {
        return $table
            ->heading('بیشترین خطاهای ۴۰۴ (۳۰ روز)')
            ->query(static fn (): Builder => app(TopNotFound::class)->query())
            ->paginated([5])
            ->emptyStateHeading('خطای ۴۰۴ ثبت نشده است.')
            ->recordUrl(static fn (): string => NotFoundLogResource::getUrl('index'))
            ->columns([
                TextColumn::make('path')->label('نشانی')->limit(60)->tooltip(static fn (NotFoundLog $record): string => $record->path),
                TextColumn::make('hits')->label('بازدید')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
                TextColumn::make('agent')->label('بازدیدکننده')->formatStateUsing(static fn (NotFoundLog $record): string => $record->agent->label()),
                TextColumn::make('last_seen_at')->label('آخرین بار')
                    ->formatStateUsing(static fn (NotFoundLog $record): string => $record->last_seen_at === null ? '—' : jdate($record->last_seen_at, 'j F')),
            ]);
    }
}
