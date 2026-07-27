<?php

namespace App\Services\HomePage\Sections;

use App\Enums\RegularityStatus;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 17 — "خلاصه سیکل": the user's own cycle numbers, each read against the
 * range that is typical for most people.
 *
 * The rows grow with the data: the last cycle/period appear as soon as there is
 * a completed one, and the average and spread rows only once enough valid
 * cycles exist to mean anything — an average of a single cycle would just be
 * that cycle restated. Nothing here is a diagnosis (§11): a value outside the
 * usual range is reported as exactly that, never as "abnormal".
 */
class CycleSummarySection extends AbstractHomeSection
{
    private const NORMAL_CYCLE_RANGE = [21, 35];

    private const NORMAL_PERIOD_RANGE = [2, 8];

    /** Below this many valid cycles an "average" or "spread" isn't meaningful. */
    private const MIN_CYCLES_FOR_AGGREGATE = 2;

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
        if (! $context->cycleData()) {
            return null;
        }

        $digest = $context->cycleHistoryDigest();
        $metrics = $context->cycleMetrics();

        $lastCycleLength = $digest->lastCycleLength();
        $lastPeriodLength = $digest->lastPeriodLength();
        $validCycles = count($digest->validCycleLengths());

        $items = [
            $this->numericItem(
                context: $context,
                key: 'last_cycle_length',
                label: $context->t('طول سیکل قبلی', 'Last cycle length'),
                value: $lastCycleLength,
                range: self::NORMAL_CYCLE_RANGE,
                emptyHint: $context->t(
                    'با ثبت پریود بعدی، طول این سیکل مشخص می‌شود.',
                    'Log your next period to learn this cycle’s length.'
                ),
            ),
            $this->numericItem(
                context: $context,
                key: 'last_period_duration',
                label: $context->t('مدت پریود قبلی', 'Last period duration'),
                value: $lastPeriodLength,
                range: self::NORMAL_PERIOD_RANGE,
                emptyHint: $context->t(
                    'پایان پریود را ثبت کنید تا مدت آن محاسبه شود.',
                    'Record when your period ends to see its duration.'
                ),
            ),
            $this->regularityItem($context, $metrics->regularityStatus, $metrics->cycleVariabilityRange),
        ];

        // Aggregates are only shown once they summarize more than one cycle.
        if ($validCycles >= self::MIN_CYCLES_FOR_AGGREGATE) {
            $items[] = $this->numericItem(
                context: $context,
                key: 'average_cycle_length',
                label: $context->t('میانگین طول سیکل', 'Average cycle length'),
                value: $digest->averageCycleLength(),
                range: self::NORMAL_CYCLE_RANGE,
                hint: $context->t(
                    "بر پایه {$context->num($validCycles)} سیکل ثبت‌شده",
                    "Based on {$validCycles} recorded cycles"
                ),
            );

            $items[] = $this->rangeItem(
                $context,
                $digest->shortestCycle(),
                $digest->longestCycle(),
            );
        }

        return new HomeSection(
            key: $this->key(),
            type: 'cycle_summary',
            title: $context->t('خلاصه سیکل', 'Cycle summary'),
            data: [
                'items' => $items,
                'based_on_cycles' => $validCycles,
                'has_history' => $lastCycleLength !== null || $lastPeriodLength !== null,
                'normal_ranges' => [
                    'cycle_length' => ['min' => self::NORMAL_CYCLE_RANGE[0], 'max' => self::NORMAL_CYCLE_RANGE[1]],
                    'period_duration' => ['min' => self::NORMAL_PERIOD_RANGE[0], 'max' => self::NORMAL_PERIOD_RANGE[1]],
                ],
            ],
            order: $this->order(),
            action: $this->action('view_more', $context->t('مشاهده بیشتر', 'View more')),
        );
    }

    /**
     * A day-valued row. `value` stays a raw number so each client formats it in
     * its own locale's digits; `unit` is a machine key with a localized twin.
     *
     * @param  array{0:int,1:int}  $range
     * @return array<string, mixed>
     */
    private function numericItem(
        HomeContext $context,
        string $key,
        string $label,
        ?int $value,
        array $range,
        ?string $hint = null,
        ?string $emptyHint = null,
    ): array {
        $status = $this->rangeStatus($value, $range);

        return $this->item(
            context: $context,
            key: $key,
            label: $label,
            value: $value,
            unit: 'days',
            unitLabel: $context->t('روز', 'days'),
            status: $status,
            hint: $value === null ? $emptyHint : $hint,
            normalRange: ['min' => $range[0], 'max' => $range[1]],
        );
    }

    /**
     * The regularity row. It reports the spec's user-facing regularity (§11),
     * which stays "not enough data" until three valid cycles exist, and carries
     * the observed spread as its hint so the label isn't an unexplained verdict.
     *
     * @return array<string, mixed>
     */
    private function regularityItem(HomeContext $context, RegularityStatus $status, ?int $spread): array
    {
        // Until regularity can be judged, the observed spread is withheld — a
        // number next to "not enough data" would read as a contradiction.
        $known = $status !== RegularityStatus::NOT_ENOUGH_DATA;
        $spread = $known ? $spread : null;

        return $this->item(
            context: $context,
            key: 'cycle_variability',
            label: $context->t('نوسان طول سیکل', 'Cycle length variation'),
            value: $spread,
            text: $status->label($context->locale),
            unit: $spread === null ? null : 'days',
            unitLabel: $spread === null ? null : $context->t('روز', 'days'),
            status: match ($status) {
                RegularityStatus::NOT_ENOUGH_DATA => 'unknown',
                RegularityStatus::RELATIVELY_REGULAR => 'normal',
                RegularityStatus::IRREGULAR_POSSIBLE => 'outside_range',
            },
            hint: $spread !== null
                ? $context->t(
                    "اختلاف کوتاه‌ترین و بلندترین سیکل اخیر: {$context->num($spread)} روز",
                    "Recent cycles differ by {$spread} days"
                )
                : $context->t(
                    'برای سنجش نظم، حداقل سه سیکل کامل لازم است.',
                    'At least three complete cycles are needed to judge regularity.'
                ),
        );
    }

    /**
     * The shortest–longest spread of recorded cycles, as a two-ended value the
     * client renders as a range.
     *
     * @return array<string, mixed>
     */
    private function rangeItem(HomeContext $context, ?int $shortest, ?int $longest): array
    {
        return $this->item(
            context: $context,
            key: 'cycle_length_range',
            label: $context->t('کوتاه‌ترین تا بلندترین سیکل', 'Shortest to longest cycle'),
            valueMin: $shortest,
            valueMax: $longest,
            unit: 'days',
            unitLabel: $context->t('روز', 'days'),
            status: $this->rangeStatus($shortest, self::NORMAL_CYCLE_RANGE) === 'normal'
                && $this->rangeStatus($longest, self::NORMAL_CYCLE_RANGE) === 'normal'
                    ? 'normal'
                    : 'outside_range',
        );
    }

    /**
     * Normalize one row so every item carries the same keys — clients can render
     * the list without testing for optional fields.
     *
     * @param  array{min:int,max:int}|null  $normalRange
     * @return array<string, mixed>
     */
    private function item(
        HomeContext $context,
        string $key,
        string $label,
        ?int $value = null,
        ?int $valueMin = null,
        ?int $valueMax = null,
        ?string $text = null,
        ?string $unit = null,
        ?string $unitLabel = null,
        string $status = 'unknown',
        ?string $hint = null,
        ?array $normalRange = null,
    ): array {
        return [
            'key' => $key,
            'label' => $label,
            'value' => $value,
            'value_min' => $valueMin,
            'value_max' => $valueMax,
            'text' => $text,
            'unit' => $unit,
            'unit_label' => $unitLabel,
            'status' => $status,
            'status_label' => $this->statusLabel($status, $context),
            'hint' => $hint,
            'normal_range' => $normalRange,
        ];
    }

    /**
     * @param  array{0:int,1:int}  $range
     */
    private function rangeStatus(?int $value, array $range): string
    {
        if ($value === null) {
            return 'unknown';
        }

        return ($value >= $range[0] && $value <= $range[1]) ? 'normal' : 'outside_range';
    }

    /**
     * Wording stays descriptive, never diagnostic (§11) — a value can sit
     * outside the usual range without anything being wrong.
     */
    private function statusLabel(string $status, HomeContext $context): string
    {
        return match ($status) {
            'normal' => $context->t('در محدودهٔ معمول', 'Within the usual range'),
            'outside_range' => $context->t('خارج از محدودهٔ معمول', 'Outside the usual range'),
            default => $context->t('هنوز مشخص نیست', 'Not known yet'),
        };
    }
}
