<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 13 — "وضعیت امروز": today's vital signs (heart rate, blood pressure,
 * blood sugar). Cards are always present; values are null until logged.
 */
class VitalsSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'vitals';
    }

    public function order(): int
    {
        return 130;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $log = $context->dailyLog();

        $heartRate = $log?->heart_rate;
        $systolic = $log?->systolic_pressure;
        $diastolic = $log?->diastolic_pressure;
        $bloodSugar = $log?->blood_sugar;

        $bloodPressure = ($systolic !== null && $diastolic !== null)
            ? "{$systolic}/{$diastolic}"
            : null;

        $items = [
            [
                'key' => 'heart_rate',
                'label' => $context->t('ضربان قلب', 'Heart rate'),
                'value' => $heartRate,
                'unit' => 'bpm',
                'icon' => 'heart-pulse',
            ],
            [
                'key' => 'blood_pressure',
                'label' => $context->t('فشار خون', 'Blood pressure'),
                'value' => $bloodPressure,
                'unit' => 'mmHg',
                'icon' => 'gauge',
            ],
            [
                'key' => 'blood_sugar',
                'label' => $context->t('قند خون', 'Blood sugar'),
                'value' => $bloodSugar !== null ? (float) $bloodSugar : null,
                'unit' => 'mg/dl',
                'icon' => 'droplet',
            ],
        ];

        $hasData = $heartRate !== null || $bloodPressure !== null || $bloodSugar !== null;

        return new HomeSection(
            key: $this->key(),
            type: 'vitals',
            title: $context->t('وضعیت امروز', "Today's vitals"),
            data: [
                'items' => $items,
                'has_data' => $hasData,
                'date' => $context->date->toDateString(),
            ],
            order: $this->order(),
            action: $this->action('view_all', $context->t('مشاهده کامل', 'View all')),
        );
    }
}
