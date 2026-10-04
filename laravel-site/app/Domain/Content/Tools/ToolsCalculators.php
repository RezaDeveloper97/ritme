<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools;

use App\Domain\Content\Tools\Data\CalculatorForm;
use App\Domain\Content\Tools\Data\JalaliDay;
use App\Domain\Content\Tools\Enums\CalculatorError;
use App\Domain\Content\Tools\Enums\CalculatorKind;

/**
 * Builds both calculator cards of /tools from the (optional) no-JS GET submission
 * `?calc=due|fert&lmp=…&cycle=…`. The card that was not submitted shows its example input and the result for it.
 * `today` matters only for the due-date «how far along» line; the example cards never show it, so the parameterless
 * page stays the same all day long (it is full-page cached).
 */
final class ToolsCalculators
{
    public function __construct(
        private readonly DueDateCalculator $due,
        private readonly FertilityWindowCalculator $fertility,
    ) {}

    /** True when the query is a calculator submission (`calc` names a calculator). */
    public static function submitted(mixed $calc): bool
    {
        return is_string($calc) && CalculatorKind::tryFrom($calc) !== null;
    }

    /**
     * @param  array<array-key, mixed>  $query  the request's query string
     * @param  array<string, array{lmp: string, cycle: string}>  $examples  example input per kind value
     * @return array<string, CalculatorForm> keyed by kind value (`due`, `fert`)
     */
    public function forms(array $query, array $examples, ResultText $text, JalaliDay $today): array
    {
        $calc = $query['calc'] ?? null;
        $submitted = self::submitted($calc) ? CalculatorKind::from((string) $calc) : null;

        $forms = [];
        foreach (CalculatorKind::cases() as $kind) {
            $forms[$kind->value] = $kind === $submitted
                ? $this->evaluate($kind, self::string($query['lmp'] ?? null), self::string($query['cycle'] ?? null), $text, $today, true)
                : $this->evaluate($kind, $examples[$kind->value]['lmp'] ?? '', $examples[$kind->value]['cycle'] ?? '', $text, null, false);
        }

        return $forms;
    }

    /**
     * One calculation. `$today` null = do not show the «how far along» line (examples, cacheable output).
     */
    public function evaluate(CalculatorKind $kind, string $lmp, string $cycle, ResultText $text, ?JalaliDay $today, bool $submitted = true): CalculatorForm
    {
        $day = CalculatorInput::parseDate($lmp);
        $length = CalculatorInput::parseCycle($cycle);

        if ($day === null) {
            return new CalculatorForm($kind, $lmp, $cycle, CalculatorError::InvalidDate, submitted: $submitted);
        }
        if ($today !== null && $day->number > $today->number) {
            return new CalculatorForm($kind, $lmp, $cycle, CalculatorError::FutureDate, submitted: $submitted);
        }
        if ($length === null) {
            return new CalculatorForm($kind, $lmp, $cycle, CalculatorError::CycleOutOfRange, submitted: $submitted);
        }

        $lines = $kind === CalculatorKind::Due
            ? $text->due($this->due->calculate($day, $length, $today))
            : $text->fertility($this->fertility->calculate($day, $length));

        return new CalculatorForm($kind, $lmp, $cycle, null, $lines['value'], $lines['detail'], $submitted);
    }

    private static function string(mixed $value): string
    {
        return is_string($value) ? mb_substr(trim($value), 0, 40) : '';
    }
}
