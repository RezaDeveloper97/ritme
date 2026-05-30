<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 14 — "وضعیت امروز" trend charts: last-7-days series for the key vital
 * signs. Returns null when there is no vitals history to plot.
 */
class StatusChartsSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'status_charts';
    }

    public function order(): int
    {
        return 140;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $logs = $context->recentLogs(7);

        $series = [
            'heart_rate' => ['label' => $context->t('ضربان قلب', 'Heart rate'), 'unit' => 'bpm', 'field' => 'heart_rate'],
            'blood_pressure' => ['label' => $context->t('فشار خون', 'Blood pressure'), 'unit' => 'mmHg', 'field' => 'systolic_pressure'],
            'blood_sugar' => ['label' => $context->t('قند خون', 'Blood sugar'), 'unit' => 'mg/dl', 'field' => 'blood_sugar'],
        ];

        $charts = [];
        $hasAnyPoint = false;

        foreach ($series as $key => $info) {
            $points = [];
            foreach ($logs as $log) {
                $value = $log->{$info['field']};
                if ($value === null) {
                    continue;
                }
                $hasAnyPoint = true;
                $points[] = [
                    'date' => $log->log_date->toDateString(),
                    'value' => is_numeric($value) ? (float) $value : $value,
                ];
            }

            $charts[] = [
                'key' => $key,
                'label' => $info['label'],
                'unit' => $info['unit'],
                'points' => $points,
            ];
        }

        if (!$hasAnyPoint) {
            return null;
        }

        return new HomeSection(
            key: $this->key(),
            type: 'status_charts',
            title: $context->t('وضعیت امروز', 'Status trends'),
            data: [
                'charts' => $charts,
                'range' => [
                    'from' => $context->date->copy()->subDays(6)->toDateString(),
                    'to' => $context->date->toDateString(),
                ],
            ],
            order: $this->order(),
            action: $this->action('view_all', $context->t('مشاهده کامل', 'View all')),
        );
    }
}
