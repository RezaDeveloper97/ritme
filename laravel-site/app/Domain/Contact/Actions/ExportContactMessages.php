<?php

declare(strict_types=1);

namespace App\Domain\Contact\Actions;

use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Models\ContactMessage;
use DateTimeInterface;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * CSV export of the contact inbox for the admin (L3-10): date (Jalali, Tehran, Latin digits so spreadsheets sort),
 * status, topic, name, email, phone, message. UTF-8 with BOM and Persian headers for Excel; rows are streamed with
 * lazyById(), every cell is guarded against formula injection, and each export is recorded in the activity log.
 */
final class ExportContactMessages
{
    private const BOM = "\xEF\xBB\xBF";

    private const HEADERS = ['تاریخ', 'وضعیت', 'موضوع', 'نام', 'ایمیل', 'تلفن', 'پیام'];

    /**
     * Messages matching the inbox list (status tab, topic filter, search), oldest first.
     *
     * @return Builder<ContactMessage>
     */
    public static function query(?ContactMessageStatus $status = null, ?ContactTopic $topic = null, ?string $search = null): Builder
    {
        $query = ContactMessage::query();
        if ($status !== null) {
            $query->where('status', $status->value);
        }
        if ($topic !== null) {
            $query->where('topic', $topic->value);
        }

        $search = trim((string) $search);
        if ($search !== '') {
            $like = '%'.str_replace(['\\', '%', '_'], ['\\\\', '\\%', '\\_'], $search).'%';
            $query->where(static function (Builder $q) use ($like): void {
                $q->where('name', 'like', $like)->orWhere('email', 'like', $like)->orWhere('phone', 'like', $like)->orWhere('message', 'like', $like);
            });
        }

        return $query;
    }

    public static function filename(): string
    {
        return 'contact-messages-'.jdate(null, 'Y-m-d', false).'.csv';
    }

    /**
     * Logs the export, then writes the CSV to `$out` (e.g. php://output inside a streamed response).
     *
     * @param  resource  $out
     * @return int rows written (without the header)
     */
    public function handle($out, ?ContactMessageStatus $status = null, ?ContactTopic $topic = null, ?string $search = null, ?Model $causer = null): int
    {
        $query = self::query($status, $topic, $search);

        activity(ChangeContactMessageStatus::LOG)
            ->causedBy($causer)
            ->event('exported')
            ->withProperties([
                'rows' => (clone $query)->count(),
                'status' => $status?->value,
                'topic' => $topic?->value,
                'search' => trim((string) $search) !== '' ? trim((string) $search) : null,
            ])
            ->log('contact.exported');

        fwrite($out, self::BOM);
        fputcsv($out, self::HEADERS, ',', '"', '');

        $rows = 0;
        foreach ($query->lazyById(500) as $message) {
            fputcsv($out, [
                self::date($message->created_at),
                $message->status->label(),
                $message->topic->label(),
                self::cell($message->name),
                self::cell((string) $message->email),
                self::cell((string) $message->phone),
                self::cell($message->message),
            ], ',', '"', '');
            $rows++;
        }

        return $rows;
    }

    private static function date(?DateTimeInterface $date): string
    {
        return $date === null ? '' : jdate($date, 'Y/m/d H:i', false);
    }

    /**
     * Neutralises spreadsheet formula injection (a value starting with = + - @ tab or CR is treated as text).
     */
    private static function cell(string $value): string
    {
        return $value !== '' && in_array($value[0], ['=', '+', '-', '@', "\t", "\r"], true) ? "'".$value : $value;
    }
}
