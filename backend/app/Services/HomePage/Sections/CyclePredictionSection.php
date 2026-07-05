<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;
use Carbon\Carbon;

/**
 * Section 4 — prediction mini-cards: fertile window, ovulation and next period,
 * each with an absolute date and a days-until countdown.
 */
class CyclePredictionSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'cycle_prediction';
    }

    public function order(): int
    {
        return 40;
    }

    public function supports(HomeContext $context): bool
    {
        return $context->isCycleMode();
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $calc = $context->cycleData();
        $cycleStart = $context->currentCycleStart();

        if (!$calc || !$cycleStart || ($calc['estimated_ovulation_day'] ?? null) === null) {
            return null;
        }

        $cycleDay = (int) $calc['cycle_day'];
        $cycleLength = (int) ($calc['cycle_length_used'] ?? 28);
        $ovulationDay = (int) $calc['estimated_ovulation_day'];
        $uncertainty = $calc['uncertainty_range'] ?? null;

        // Day N of the cycle falls on cycleStart + (N - 1).
        $dateForDay = fn (int $n): Carbon => $cycleStart->copy()->addDays($n - 1);

        // Resolve the next occurrence of a cycle-day: keep it in the current
        // cycle if it is still ahead (or today), otherwise roll it to the next
        // cycle. The rendered date and the days_until countdown are both derived
        // from this same index, so they can never contradict each other.
        $nextOcc = fn (int $day): int => $day >= $cycleDay ? $day : $day + $cycleLength;

        $ovulationOcc = $nextOcc($ovulationDay);
        // Anchor the whole fertile window to the upcoming (or current) ovulation.
        $fertileStartOcc = $ovulationOcc - 5;
        $fertileEndOcc = $ovulationOcc + 1;

        $cards = [
            [
                'key' => 'fertile_window',
                'label' => $context->t('پنجره باروری', 'Fertile window'),
                'start_date' => $dateForDay($fertileStartOcc)->toDateString(),
                'end_date' => $dateForDay($fertileEndOcc)->toDateString(),
                'days_until' => max(0, $fertileStartOcc - $cycleDay),
                'is_active' => $calc['is_fertile_window'] ?? false,
            ],
            [
                'key' => 'ovulation',
                'label' => $context->t('تخمک‌گذاری', 'Ovulation'),
                'date' => $dateForDay($ovulationOcc)->toDateString(),
                'days_until' => $ovulationOcc - $cycleDay,
            ],
            [
                'key' => 'next_period',
                'label' => $context->t('پریود بعدی', 'Next period'),
                'date' => $dateForDay($cycleLength + 1)->toDateString(),
                'days_until' => max(0, $cycleLength - $cycleDay + 1),
            ],
        ];

        return new HomeSection(
            key: $this->key(),
            type: 'cycle_prediction',
            title: null,
            data: [
                'cards' => $cards,
                'uncertainty_range' => $uncertainty,
            ],
            order: $this->order(),
        );
    }
}
