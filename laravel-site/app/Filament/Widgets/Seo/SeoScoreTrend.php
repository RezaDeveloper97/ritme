<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Seo;

use App\Domain\Seo\Audit\Models\AuditRun;
use App\Domain\Seo\Audit\Queries\AuditHistory;
use Filament\Widgets\ChartWidget;

/**
 * Health score and error count of the last twelve completed audits (L7-05).
 */
final class SeoScoreTrend extends ChartWidget
{
    use SeoWidgetAccess;

    protected static ?int $sort = 11;

    protected ?string $pollingInterval = null;

    protected ?string $heading = 'روند امتیاز سئو';

    protected ?string $description = 'دوازده ممیزی اخیر';

    protected ?string $maxHeight = '240px';

    protected function getType(): string
    {
        return 'line';
    }

    protected function getData(): array
    {
        $runs = app(AuditHistory::class)->trend();

        return [
            'datasets' => [
                [
                    'label' => 'امتیاز',
                    'data' => $runs->map(static fn (AuditRun $run): int => (int) $run->score)->all(),
                    'tension' => 0.3,
                ],
                [
                    'label' => 'خطا',
                    'data' => $runs->map(static fn (AuditRun $run): int => $run->errors_count)->all(),
                    'tension' => 0.3,
                ],
            ],
            'labels' => $runs->map(static fn (AuditRun $run): string => $run->finished_at === null ? '—' : jdate($run->finished_at, 'j F'))->all(),
        ];
    }

    protected function getOptions(): array
    {
        return ['scales' => ['y' => ['beginAtZero' => true, 'suggestedMax' => 100]]];
    }
}
