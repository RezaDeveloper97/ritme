<?php

declare(strict_types=1);

namespace App\Domain\Contact\Actions;

use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Domain\Contact\Models\ContactMessage;
use Illuminate\Database\Eloquent\Model;

/**
 * Moves an inbox message between unread / read / archived. `read_at` records the first time it was read: kept when
 * the message is archived or restored, cleared by "mark unread". Admin changes are written to the activity log
 * (`contact`); the automatic "read" on opening a message passes `$log = false`.
 */
final class ChangeContactMessageStatus
{
    public const LOG = 'contact';

    public function handle(ContactMessage $message, ContactMessageStatus $status, ?Model $causer = null, bool $log = true): void
    {
        if ($message->status === $status) {
            return;
        }

        $from = $message->status;
        $message->status = $status;
        $message->read_at = match ($status) {
            ContactMessageStatus::Unread => null,
            default => $message->read_at ?? now(),
        };
        $message->save();

        if ($log) {
            activity(self::LOG)
                ->causedBy($causer)
                ->performedOn($message)
                ->event('updated')
                ->withProperties(['old' => ['status' => $from->value], 'attributes' => ['status' => $status->value]])
                ->log('contact.status');
        }
    }
}
