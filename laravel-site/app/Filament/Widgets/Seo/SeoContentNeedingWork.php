<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Seo;

use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Seo\Audit\Models\AuditRunPage;
use App\Domain\Seo\Audit\Queries\AuditHistory;
use App\Domain\Seo\Audit\SeoAuditEngine;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Filament\Widgets\TableWidget;
use Illuminate\Database\Eloquent\Builder;

/**
 * Posts, products and places whose content-analyser score in the latest audit is under 60 (L7-05), weakest first,
 * each linking to its edit form.
 */
final class SeoContentNeedingWork extends TableWidget
{
    use SeoWidgetAccess;

    protected static ?int $sort = 15;

    public function table(Table $table): Table
    {
        return $table
            ->heading('محتوای نیازمند کار (امتیاز تحلیل زیر '.fa_digits(SeoAuditEngine::ANALYSIS_THRESHOLD).')')
            ->query(static function (): Builder {
                $run = app(AuditHistory::class)->latest();

                return AuditRunPage::query()
                    ->where('run_id', $run->id ?? 0)
                    ->whereNotNull('analysis_score')
                    ->where('analysis_score', '<', SeoAuditEngine::ANALYSIS_THRESHOLD);
            })
            ->defaultSort('analysis_score')
            ->paginated([5])
            ->emptyStateHeading('همه محتواها امتیاز قابل قبول دارند.')
            ->recordUrl(static fn (AuditRunPage $record): ?string => $record->edit_url)
            ->columns([
                TextColumn::make('title')->label('صفحه')->limit(50)->description(static fn (AuditRunPage $record): string => $record->path),
                TextColumn::make('content_type')->label('نوع')
                    ->formatStateUsing(static fn (?string $state): string => ContentType::tryFrom((string) $state)?->label() ?? '—'),
                TextColumn::make('analysis_score')->label('امتیاز')->badge()->color('danger')
                    ->formatStateUsing(static fn (int $state): string => fa_digits($state)),
            ]);
    }
}
