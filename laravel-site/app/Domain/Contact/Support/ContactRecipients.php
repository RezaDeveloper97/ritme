<?php

declare(strict_types=1);

namespace App\Domain\Contact\Support;

use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Settings\Data\SiteSettings;

/**
 * The mailbox notified about a new message, from the contact settings (admin → تنظیمات → تماس): partnership topics
 * go to the partnership email, privacy requests to the data-protection email (legal settings), everything else —
 * and any topic whose own mailbox is empty or still a «[…]» placeholder — to the support email. null when no valid
 * address is configured (the message is still stored in the inbox).
 */
final class ContactRecipients
{
    public static function for(ContactTopic $topic, SiteSettings $settings): ?string
    {
        $specific = match ($topic) {
            ContactTopic::Partnership => $settings->contact->partnershipEmail,
            ContactTopic::Privacy => $settings->legal->dataProtectionEmail,
            default => null,
        };

        return self::valid($specific) ?? self::valid($settings->contact->supportEmail);
    }

    public static function valid(?string $email): ?string
    {
        $email = trim((string) $email);

        return $email !== '' && filter_var($email, FILTER_VALIDATE_EMAIL) !== false ? $email : null;
    }
}
