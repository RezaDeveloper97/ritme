<?php

namespace App\Services\HomePage\Sections;

use App\Models\CycleHistory;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 16 — "سیکل‌های من": current cycle day & start date plus a list of
 * previous recorded cycles.
 */
class MyCyclesSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'my_cycles';
    }

    public function order(): int
    {
        return 160;
    }

    public function supports(HomeContext $context): bool
    {
        return $context->isCycleMode();
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $calc = $context->cycleData();
        $cycleStart = $context->currentCycleStart();

        if (!$calc || !$cycleStart) {
            return null;
        }

        $histories = $context->cycleHistories();

        $previous = $histories->take(6)->map(fn (CycleHistory $h) => [
            'id' => $h->id,
            'period_start_date' => $h->period_start_date?->toDateString(),
            'period_end_date' => $h->period_end_date?->toDateString(),
            'cycle_length' => $h->cycle_length,
            'bleeding_length' => $h->bleeding_length,
            'is_confirmed' => (bool) $h->is_confirmed,
        ])->all();

        return new HomeSection(
            key: $this->key(),
            type: 'my_cycles',
            title: $context->t('سیکل‌های من', 'My cycles'),
            data: [
                'current' => [
                    'cycle_day' => $calc['cycle_day'],
                    'started_at' => $cycleStart->toDateString(),
                    'cycle_length' => $calc['cycle_length_used'] ?? null,
                ],
                'previous_count' => $histories->count(),
                'previous' => $previous,
            ],
            order: $this->order(),
            action: $this->action('view_previous', $context->t('نمایش سیکل‌های قبلی', 'Show previous cycles')),
        );
    }
}
