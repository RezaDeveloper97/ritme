<?php

namespace App\Services\HomePage\Sections;

use App\Enums\CycleVariability;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 17 — "خلاصه سیکل": last cycle length, last period duration and cycle
 * variability, each flagged normal / abnormal.
 */
class CycleSummarySection extends AbstractHomeSection
{
    private const NORMAL_CYCLE_RANGE = [21, 35];
    private const NORMAL_PERIOD_RANGE = [2, 8];

    public function key(): string
    {
        return 'cycle_summary';
    }

    public function order(): int
    {
        return 170;
    }

    public function supports(HomeContext $context): bool
    {
        return $context->isCycleMode();
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $calc = $context->cycleData();
        if (!$calc) {
            return null;
        }

        $latest = $context->cycleHistories()->first();
        $lastCycleLength = $latest?->cycle_length;
        $lastPeriodDuration = $latest?->bleeding_length;

        $variabilityValue = $calc['cycle_variability'] ?? null;
        $variability = $variabilityValue ? CycleVariability::tryFrom($variabilityValue) : null;

        $items = [
            [
                'key' => 'last_cycle_length',
                'label' => $context->t('طول سیکل قبلی', 'Last cycle length'),
                'value' => $lastCycleLength,
                'unit' => $context->t('روز', 'days'),
                'status' => $this->rangeStatus($lastCycleLength, self::NORMAL_CYCLE_RANGE),
            ],
            [
                'key' => 'last_period_duration',
                'label' => $context->t('مدت پریود قبلی', 'Last period duration'),
                'value' => $lastPeriodDuration,
                'unit' => $context->t('روز', 'days'),
                'status' => $this->rangeStatus($lastPeriodDuration, self::NORMAL_PERIOD_RANGE),
            ],
            [
                'key' => 'cycle_variability',
                'label' => $context->t('نوسان طول سیکل', 'Cycle length variation'),
                'value' => $variability?->label($context->locale),
                'unit' => null,
                'status' => $variability === null
                    ? 'unknown'
                    : ($variability === CycleVariability::REGULAR ? 'normal' : 'abnormal'),
            ],
        ];

        // Attach a localized status label to each item.
        foreach ($items as &$item) {
            $item['status_label'] = $this->statusLabel($item['status'], $context);
        }
        unset($item);

        return new HomeSection(
            key: $this->key(),
            type: 'cycle_summary',
            title: $context->t('خلاصه سیکل', 'Cycle summary'),
            data: [
                'items' => $items,
            ],
            order: $this->order(),
        );
    }

    /**
     * @param array{0:int,1:int} $range
     */
    private function rangeStatus(?int $value, array $range): string
    {
        if ($value === null) {
            return 'unknown';
        }

        return ($value >= $range[0] && $value <= $range[1]) ? 'normal' : 'abnormal';
    }

    private function statusLabel(string $status, HomeContext $context): string
    {
        return match ($status) {
            'normal' => $context->t('نرمال', 'Normal'),
            'abnormal' => $context->t('غیرطبیعی', 'Abnormal'),
            default => $context->t('نامشخص', 'Unknown'),
        };
    }
}
