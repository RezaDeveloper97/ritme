<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools\Enums;

/**
 * The two calculators on /tools. The value is the `calc` query parameter of the no-JS GET form
 * (`/tools?calc=due&lmp=1405/02/12&cycle=28`); `anchor()` is the stable fragment the footer and home page link to.
 */
enum CalculatorKind: string
{
    case Due = 'due';
    case Fertility = 'fert';

    public function anchor(): string
    {
        return match ($this) {
            self::Due => 'due-date',
            self::Fertility => 'fertility',
        };
    }
}
