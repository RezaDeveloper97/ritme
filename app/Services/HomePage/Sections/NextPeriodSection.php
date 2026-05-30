<?php

namespace App\Services\HomePage\Sections;

use App\Enums\CyclePhase;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 3 — hero card: countdown to next period, cycle progress bar,
 * pregnancy-probability badge and the "شروع پریود" CTA.
 */
class NextPeriodSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'next_period';
    }

    public function order(): int
    {
        return 30;
    }

    public function supports(HomeContext $context): bool
    {
        return $context->isCycleMode();
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $calc = $context->cycleData();
        if (!$calc || ($calc['cycle_day'] ?? null) === null) {
            return null;
        }

        $cycleDay = (int) $calc['cycle_day'];
        $cycleLength = (int) ($calc['cycle_length_used'] ?? 28);
        $probability = (float) ($calc['final_probability'] ?? 0); // already a percentage

        $daysUntilPeriod = max(0, $cycleLength - $cycleDay + 1);
        $nextPeriodDate = $context->date->copy()->addDays($daysUntilPeriod);
        $progress = $cycleLength > 0 ? (int) round($cycleDay / $cycleLength * 100) : 0;

        [$probLevel, $probLabel] = $this->probabilityBadge($probability, $context);

        $isInPeriod = ($calc['phase'] ?? null) === CyclePhase::MENSTRUATION->value;

        $title = $context->t(
            "{$daysUntilPeriod} روز تا پریود بعدی",
            "{$daysUntilPeriod} days to next period"
        );

        return new HomeSection(
            key: $this->key(),
            type: 'next_period',
            title: $title,
            data: [
                'days_until_period' => $daysUntilPeriod,
                'next_period_date' => $nextPeriodDate->toDateString(),
                'cycle_day' => $cycleDay,
                'cycle_length' => $cycleLength,
                'progress_percent' => min(100, max(0, $progress)),
                'is_in_period' => $isInPeriod,
                'is_period_tomorrow' => $calc['is_period_tomorrow'] ?? false,
                'is_fertile_window' => $calc['is_fertile_window'] ?? false,
                'uncertainty_range' => $calc['uncertainty_range'] ?? null,
                'pregnancy_probability' => [
                    'percent' => $probability,
                    'level' => $probLevel,
                    'label' => $probLabel,
                ],
            ],
            order: $this->order(),
            subtitle: $context->t(
                'پریود بعدی شما حدوداً ' . $nextPeriodDate->toDateString(),
                'Your next period is around ' . $nextPeriodDate->toDateString()
            ),
            action: $this->action('log_period_start', $context->t('شروع پریود', 'Log period start')),
        );
    }

    /**
     * @return array{0: string, 1: string} [level, label]
     */
    private function probabilityBadge(float $percent, HomeContext $context): array
    {
        if ($percent < 5) {
            return ['low', $context->t('کم', 'Low')];
        }
        if ($percent < 15) {
            return ['medium', $context->t('متوسط', 'Medium')];
        }

        return ['high', $context->t('بالا', 'High')];
    }
}
