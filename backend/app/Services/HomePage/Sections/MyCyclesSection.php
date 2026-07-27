<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 16 — "سیکل‌های من": where the user is in the current cycle, plus the
 * cycles behind it with the length each one actually ran.
 *
 * Every number here is derived: the current cycle comes from the engine (so it
 * agrees with the rest of the page), and each previous cycle's length is the
 * gap to the period that followed it (see
 * {@see \App\Services\HomePage\Support\CycleHistoryDigest}) rather than a
 * stored column that is only sometimes populated.
 */
class MyCyclesSection extends AbstractHomeSection
{
    /** Enough history for the list to feel complete without paging the payload. */
    private const PREVIOUS_LIMIT = 12;

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

        if (! $calc || ! $cycleStart) {
            return null;
        }

        $digest = $context->cycleHistoryDigest();
        $metrics = $context->cycleMetrics();

        // The engine anchors the current cycle; the logged row starting on that
        // same day (when there is one) carries the bleed the user recorded.
        $currentRecord = $digest->startingOn($cycleStart);
        $previous = $digest->previous(self::PREVIOUS_LIMIT);

        return new HomeSection(
            key: $this->key(),
            type: 'my_cycles',
            title: $context->t('سیکل‌های من', 'My cycles'),
            data: [
                'current' => [
                    'id' => $currentRecord['id'] ?? null,
                    'cycle_day' => $calc['cycle_day'],
                    'started_at' => $cycleStart->toDateString(),
                    'period_end_date' => $currentRecord['period_end_date'] ?? null,
                    'period_length' => $currentRecord['period_length'] ?? null,
                    'is_ongoing' => $currentRecord === null ? false : $currentRecord['is_ongoing'],
                    // The length predictions run on, and where it came from — a
                    // learned median, the profile baseline, or the system default.
                    'cycle_length' => $calc['cycle_length_used'] ?? $metrics->effectiveCycleLength,
                    'cycle_length_source' => $metrics->cycleLengthSource->value,
                ],
                'previous_count' => count($digest->previous()),
                'previous' => $previous,
                'averages' => [
                    'cycle_length' => $digest->averageCycleLength(),
                    'period_length' => $digest->averagePeriodLength(),
                    'based_on_cycles' => count($digest->validCycleLengths()),
                ],
            ],
            order: $this->order(),
            action: $this->action('add_previous', $context->t('ثبت سیکل‌های قبلی', 'Add previous cycles')),
        );
    }
}
