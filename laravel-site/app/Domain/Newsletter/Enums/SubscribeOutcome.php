<?php

declare(strict_types=1);

namespace App\Domain\Newsletter\Enums;

/**
 * What Subscribe did. The visitor sees the same message for every outcome (no address enumeration).
 */
enum SubscribeOutcome: string
{
    case Created = 'created';            // new address, confirmation mail sent
    case Resubscribed = 'resubscribed';  // was unsubscribed: back to pending, new token, mail sent
    case Resent = 'resent';              // still pending: confirmation mail sent again
    case Throttled = 'throttled';        // still pending and a mail went out a moment ago: nothing sent
    case AlreadyActive = 'already-active';

    public function mailed(): bool
    {
        return in_array($this, [self::Created, self::Resubscribed, self::Resent], true);
    }
}
