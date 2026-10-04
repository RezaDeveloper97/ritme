<?php

declare(strict_types=1);

namespace App\Domain\Directory\Support;

use Illuminate\Database\Eloquent\Model;

/**
 * Activity-log entries of the directory admin (log `directory`): every status change of a place, review, booking
 * request or join request is recorded with its old and new value and the admin who made it. Properties never carry
 * personal data (names, phone numbers) — the subject id is enough to look the record up.
 */
final class DirectoryActivity
{
    public const LOG = 'directory';

    public static function status(Model $subject, string $description, string $from, string $to, ?Model $causer = null): void
    {
        activity(self::LOG)
            ->causedBy($causer)
            ->performedOn($subject)
            ->event('updated')
            ->withProperties(['old' => ['status' => $from], 'attributes' => ['status' => $to]])
            ->log($description);
    }

    /**
     * @param  array<string, mixed>  $properties
     */
    public static function event(?Model $subject, string $event, string $description, array $properties = [], ?Model $causer = null): void
    {
        $log = activity(self::LOG)->causedBy($causer)->event($event)->withProperties($properties);
        if ($subject !== null) {
            $log->performedOn($subject);
        }
        $log->log($description);
    }
}
