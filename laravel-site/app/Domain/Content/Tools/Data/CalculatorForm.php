<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools\Data;

use App\Domain\Content\Tools\Enums\CalculatorError;
use App\Domain\Content\Tools\Enums\CalculatorKind;

/**
 * Everything pages/tools.blade.php needs for one calculator card: the field values to show, the error (if the
 * submitted input was rejected) and the ready result lines (null while there is an error). `submitted` is true
 * when this card was the one sent through the no-JS GET form.
 */
final readonly class CalculatorForm
{
    public function __construct(
        public CalculatorKind $kind,
        public string $lmp,
        public string $cycle,
        public ?CalculatorError $error = null,
        public ?string $value = null,
        public ?string $detail = null,
        public bool $submitted = false,
    ) {}

    public function hasResult(): bool
    {
        return $this->error === null && $this->value !== null;
    }
}
