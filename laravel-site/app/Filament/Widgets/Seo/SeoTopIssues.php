<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Seo;

use App\Domain\Seo\Audit\IssueCodes;
use App\Domain\Seo\Audit\Queries\AuditHistory;
use App\Domain\Seo\Audit\Severity;
use App\Filament\Pages\Seo\AuditReport;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Filament\Widgets\TableWidget;

/**
 * The most frequent findings of the latest audit, errors first (L7-05); each row opens the report filtered by it.
 */
final class SeoTopIssues extends TableWidget
{
    use SeoWidgetAccess;

    protected static ?int $sort = 12;

    public function table(Table $table): Table
    {
        return $table
            ->heading('مشکلات پرتکرار سئو')
            ->records(static function (): array {
                $history = app(AuditHistory::class);
                $run = $history->latest();
                $rows = [];
                foreach ($run === null ? [] : $history->topIssues($run) as $row) {
                    $rows[$row['code'].'|'.$row['severity']->value] = [
                        'code' => $row['code'],
                        'label' => IssueCodes::label($row['code']),
                        'severity' => $row['severity']->value,
                        'count' => $row['count'],
                    ];
                }

                return $rows;
            })
            ->paginated(false)
            ->emptyStateHeading('مشکلی ثبت نشده است.')
            ->recordUrl(static fn (array $record): string => AuditReport::getUrl(['code' => $record['code']]))
            ->columns([
                TextColumn::make('label')->label('مشکل')->description(static fn (array $record): string => (string) $record['code']),
                TextColumn::make('severity')->label('شدت')->badge()
                    ->formatStateUsing(static fn (string $state): string => Severity::from($state)->label())
                    ->color(static fn (string $state): string => Severity::from($state)->color()),
                TextColumn::make('count')->label('تعداد')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
            ]);
    }
}
