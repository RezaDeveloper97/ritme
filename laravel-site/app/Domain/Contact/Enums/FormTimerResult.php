<?php

declare(strict_types=1);

namespace App\Domain\Contact\Enums;

/** Outcome of the contact form's time trap (FormTimer). Invalid and TooFast are spam signals. */
enum FormTimerResult
{
    case Ok;
    case TooFast;
    case Expired;
    case Invalid;

    public function isSpam(): bool
    {
        return $this === self::TooFast || $this === self::Invalid;
    }
}
