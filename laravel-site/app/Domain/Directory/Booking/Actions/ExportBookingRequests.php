<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Actions;

use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Booking\Support\MobileMask;
use App\Domain\Directory\Support\DirectoryActivity;
use App\Support\Text\PersianDigits;
use DateTimeInterface;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * CSV export of the bookings board (L5-06): code, received (Jalali, Tehran), status, place, service, preferred day,
 * time window, parent name, MASKED mobile, child age. Personal data is kept minimal — the full number and the free
 * note stay in the admin view (needed to call back), not in files that travel. UTF-8 with BOM and Persian headers;
 * Latin digits so spreadsheets sort; rows streamed with lazyById(); cells guarded against formula injection; each
 * export is written to the activity log (`directory.booking.exported`).
 */
final class ExportBookingRequests
{
    private const BOM = "\xEF\xBB\xBF";

    private const HEADERS = ['کد', 'دریافت', 'وضعیت', 'مجموعه', 'خدمت', 'روز درخواستی', 'بازه زمانی', 'نام والد', 'موبایل', 'سن کودک (ماه)'];

    /**
     * Requests matching the board (status tab, place filter, preferred-day range, search on code / place / name).
     *
     * @return Builder<BookingRequest>
     */
    public static function query(?BookingStatus $status = null, ?int $placeId = null, ?string $from = null, ?string $until = null, ?string $search = null): Builder
    {
        $query = BookingRequest::query();
        if ($status !== null) {
            $query->where('status', $status->value);
        }
        if ($placeId !== null) {
            $query->where('place_id', $placeId);
        }
        if ($from !== null && preg_match('/^\d{4}-\d{2}-\d{2}$/', $from) === 1) {
            $query->whereDate('preferred_date', '>=', $from);
        }
        if ($until !== null && preg_match('/^\d{4}-\d{2}-\d{2}$/', $until) === 1) {
            $query->whereDate('preferred_date', '<=', $until);
        }

        $search = trim(PersianDigits::toLatin((string) $search));
        if ($search !== '') {
            $like = '%'.str_replace(['\\', '%', '_'], ['\\\\', '\\%', '\\_'], $search).'%';
            $query->where(static function (Builder $q) use ($like): void {
                $q->where('code', 'like', $like)->orWhere('place_name', 'like', $like)->orWhere('parent_name', 'like', $like);
            });
        }

        return $query;
    }

    public static function filename(): string
    {
        return 'directory-bookings-'.jdate(null, 'Y-m-d', false).'.csv';
    }

    /**
     * Logs the export, then writes the CSV to `$out` (php://output inside a streamed response).
     *
     * @param  resource  $out
     * @return int rows written (without the header)
     */
    public function handle($out, ?BookingStatus $status = null, ?int $placeId = null, ?string $from = null, ?string $until = null, ?string $search = null, ?Model $causer = null): int
    {
        $query = self::query($status, $placeId, $from, $until, $search);

        DirectoryActivity::event(null, 'exported', 'directory.booking.exported', [
            'rows' => (clone $query)->count(),
            'status' => $status?->value,
            'place_id' => $placeId,
            'from' => $from,
            'until' => $until,
        ], $causer);

        fwrite($out, self::BOM);
        fputcsv($out, self::HEADERS, ',', '"', '');

        $rows = 0;
        foreach ($query->lazyById(500) as $booking) {
            fputcsv($out, [
                $booking->code,
                self::date($booking->created_at, 'Y/m/d H:i'),
                $booking->status->label(),
                self::cell($booking->place_name),
                self::cell((string) $booking->service_name),
                self::date($booking->preferred_date, 'Y/m/d'),
                $booking->time_window->label(),
                self::cell($booking->parent_name),
                PersianDigits::toLatin(MobileMask::mask($booking->mobile)),
                $booking->child_age_months === null ? '' : (string) $booking->child_age_months,
            ], ',', '"', '');
            $rows++;
        }

        return $rows;
    }

    private static function date(?DateTimeInterface $date, string $format): string
    {
        return $date === null ? '' : jdate($date, $format, false);
    }

    /**
     * Neutralises spreadsheet formula injection (a value starting with = + - @ tab or CR is treated as text).
     */
    private static function cell(string $value): string
    {
        return $value !== '' && in_array($value[0], ['=', '+', '-', '@', "\t", "\r"], true) ? "'".$value : $value;
    }
}
