<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;
use App\Services\HomePage\Support\HealthMetricScorer;

/**
 * Section 12 — "خلاصه هفته": mood / sleep / energy averaged over the last 7 days
 * as percentages, each compared with the 7 days before that so the card can show
 * a trend. Every percentage is null until the user has logged something the
 * metric can be scored from — the client renders that as a dash, never as 0%.
 */
class WeeklySummarySection extends AbstractHomeSection
{
    /** Days per comparison window. */
    private const WINDOW = 7;

    public function key(): string
    {
        return 'weekly_summary';
    }

    public function order(): int
    {
        return 120;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $logs = $context->recentLogs(self::WINDOW);
        $averages = HealthMetricScorer::weeklyAverages($logs);

        // The window immediately before this one, for the per-metric delta.
        $previousLogs = $context->logsBetween(
            $context->date->copy()->subDays(self::WINDOW * 2 - 1),
            $context->date->copy()->subDays(self::WINDOW)
        );
        $previousAverages = HealthMetricScorer::weeklyAverages($previousLogs);

        $meta = [
            'mood' => ['label' => $context->t('روحیه', 'Mood'), 'icon' => 'smile'],
            'sleep' => ['label' => $context->t('خواب', 'Sleep'), 'icon' => 'moon'],
            'energy' => ['label' => $context->t('انرژی', 'Energy'), 'icon' => 'bolt'],
        ];

        $items = [];
        foreach ($meta as $metric => $info) {
            $percent = $averages[$metric];
            $previous = $previousAverages[$metric];

            $items[] = [
                'key' => $metric,
                'label' => $info['label'],
                'icon' => $info['icon'],
                'percent' => $percent,
                'previous_percent' => $previous,
                // Null unless both windows scored — "no comparison" is not "no change".
                'delta' => ($percent !== null && $previous !== null) ? $percent - $previous : null,
            ];
        }

        $scored = array_values(array_filter(
            array_column($items, 'percent'),
            fn ($p) => $p !== null
        ));

        return new HomeSection(
            key: $this->key(),
            type: 'weekly_summary',
            title: $context->t('خلاصه هفته', 'Weekly summary'),
            data: [
                'items' => $items,
                'range' => [
                    'from' => $context->date->copy()->subDays(self::WINDOW - 1)->toDateString(),
                    'to' => $context->date->toDateString(),
                ],
                'logged_days' => $logs->count(),
                'previous_logged_days' => $previousLogs->count(),
                // Headline score across the metrics that could be scored at all.
                'overall_percent' => $scored === [] ? null : (int) round(array_sum($scored) / count($scored)),
            ],
            order: $this->order(),
            action: $this->action('view_all', $context->t('مشاهده کامل', 'View all')),
        );
    }
}
