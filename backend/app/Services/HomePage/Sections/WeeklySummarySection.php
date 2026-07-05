<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;
use App\Services\HomePage\Support\HealthMetricScorer;

/**
 * Section 12 — "خلاصه هفته": mood / sleep / energy averaged over the last 7 days
 * as percentages.
 */
class WeeklySummarySection extends AbstractHomeSection
{
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
        $logs = $context->recentLogs(7);
        $averages = HealthMetricScorer::weeklyAverages($logs);

        $meta = [
            'mood' => ['label' => $context->t('روحیه', 'Mood'), 'icon' => 'smile'],
            'sleep' => ['label' => $context->t('خواب', 'Sleep'), 'icon' => 'moon'],
            'energy' => ['label' => $context->t('انرژی', 'Energy'), 'icon' => 'bolt'],
        ];

        $items = [];
        foreach ($meta as $metric => $info) {
            $items[] = [
                'key' => $metric,
                'label' => $info['label'],
                'icon' => $info['icon'],
                'percent' => $averages[$metric],
            ];
        }

        return new HomeSection(
            key: $this->key(),
            type: 'weekly_summary',
            title: $context->t('خلاصه هفته', 'Weekly summary'),
            data: [
                'items' => $items,
                'range' => [
                    'from' => $context->date->copy()->subDays(6)->toDateString(),
                    'to' => $context->date->toDateString(),
                ],
                'logged_days' => $logs->count(),
            ],
            order: $this->order(),
            action: $this->action('view_all', $context->t('مشاهده کامل', 'View all')),
        );
    }
}
