<?php

namespace App\Services\HomePage\Sections;

use App\Enums\CyclePhase;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;
use App\Services\HomePage\Support\CyclePhasePalette;
use Carbon\Carbon;

/**
 * Section 2 — the 7-day week strip (Saturday→Friday) with per-day phase color.
 */
class WeekCalendarSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'week_calendar';
    }

    public function order(): int
    {
        return 20;
    }

    public function supports(HomeContext $context): bool
    {
        return $context->isCycleMode();
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $hasCycle = $context->hasCycleData();
        $weekStart = $context->date->copy()->startOfWeek(Carbon::SATURDAY);
        $today = $context->date->toDateString();

        $days = [];
        for ($i = 0; $i < 7; $i++) {
            $day = $weekStart->copy()->addDays($i);
            $calc = $hasCycle ? $context->cycleDataFor($day) : null;
            $phase = $calc['phase'] ?? null;

            $days[] = [
                'date' => $day->toDateString(),
                'weekday_iso' => $day->dayOfWeekIso,
                'is_today' => $day->toDateString() === $today,
                'cycle_day' => $calc['cycle_day'] ?? null,
                'phase' => $phase,
                'phase_label' => CyclePhasePalette::label($phase, $context->locale),
                'color' => CyclePhasePalette::color($phase),
                'is_period' => $phase === CyclePhase::MENSTRUATION->value,
                'is_fertile' => $calc['is_fertile_window'] ?? false,
                'is_ovulation' => $phase === CyclePhase::OVULATION->value,
            ];
        }

        return new HomeSection(
            key: $this->key(),
            type: 'week_calendar',
            title: null,
            data: [
                'week_start' => $weekStart->toDateString(),
                'days' => $days,
            ],
            order: $this->order(),
        );
    }
}
