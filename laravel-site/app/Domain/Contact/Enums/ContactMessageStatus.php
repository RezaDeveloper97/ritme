<?php

declare(strict_types=1);

namespace App\Domain\Contact\Enums;

/**
 * Inbox state of a contact message: new messages are unread; opening one marks it read; archived ones leave the
 * working lists but are kept (and can be restored) until an admin deletes them.
 */
enum ContactMessageStatus: string
{
    case Unread = 'unread';
    case Read = 'read';
    case Archived = 'archived';

    public function label(): string
    {
        return match ($this) {
            self::Unread => 'خوانده‌نشده',
            self::Read => 'خوانده‌شده',
            self::Archived => 'بایگانی',
        };
    }

    public function color(): string
    {
        return match ($this) {
            self::Unread => 'warning',
            self::Read => 'success',
            self::Archived => 'gray',
        };
    }
}
