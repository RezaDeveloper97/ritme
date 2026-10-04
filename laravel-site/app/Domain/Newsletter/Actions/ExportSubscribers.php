<?php

declare(strict_types=1);

namespace App\Domain\Newsletter\Actions;

use App\Domain\Newsletter\Enums\SubscriptionStatus;
use App\Domain\Newsletter\Models\Subscriber;
use DateTimeInterface;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * CSV export of newsletter subscribers for the admin (L4-05b). Minimal data only: email, status and the consent /
 * confirmation / unsubscribe dates (Jalali, Tehran, Latin digits so spreadsheets can sort them) — no token or source.
 * UTF-8 with BOM and Persian headers so Excel opens it correctly; rows are read with lazyById() and written straight
 * to the output stream, so memory stays flat however long the list is. Every export is recorded in the activity log.
 */
final class ExportSubscribers
{
    public const LOG = 'newsletter';

    private const BOM = "\xEF\xBB\xBF";

    private const HEADERS = ['ایمیل', 'وضعیت', 'تاریخ ثبت‌نام', 'تاریخ تأیید', 'تاریخ لغو'];

    /**
     * Subscribers matching the admin list filters (status, email search), oldest first.
     *
     * @return Builder<Subscriber>
     */
    public static function query(?SubscriptionStatus $status = null, ?string $search = null): Builder
    {
        $query = Subscriber::query();
        if ($status !== null) {
            self::whereStatus($query, $status);
        }

        $search = trim((string) $search);
        if ($search !== '') {
            $query->where('email', 'like', '%'.mb_strtolower($search).'%');
        }

        return $query;
    }

    /**
     * The derived status as a query constraint (mirrors Subscriber::status()).
     *
     * @param  Builder<Subscriber>  $query
     * @return Builder<Subscriber>
     */
    public static function whereStatus(Builder $query, SubscriptionStatus $status): Builder
    {
        return match ($status) {
            SubscriptionStatus::Unsubscribed => $query->whereNotNull('unsubscribed_at'),
            SubscriptionStatus::Active => $query->whereNotNull('confirmed_at')->whereNull('unsubscribed_at'),
            SubscriptionStatus::Pending => $query->whereNull('confirmed_at')->whereNull('unsubscribed_at'),
        };
    }

    public static function filename(): string
    {
        return 'newsletter-subscribers-'.jdate(null, 'Y-m-d', false).'.csv';
    }

    /**
     * Logs the export, then writes the CSV to `$out` (e.g. php://output inside a streamed response).
     *
     * @param  resource  $out
     * @return int rows written (without the header)
     */
    public function handle($out, ?SubscriptionStatus $status = null, ?string $search = null, ?Model $causer = null): int
    {
        $query = self::query($status, $search);

        activity(self::LOG)
            ->causedBy($causer)
            ->event('exported')
            ->withProperties([
                'rows' => (clone $query)->count(),
                'status' => $status?->value,
                'search' => trim((string) $search) !== '' ? trim((string) $search) : null,
            ])
            ->log('newsletter.exported');

        fwrite($out, self::BOM);
        fputcsv($out, self::HEADERS, ',', '"', '');

        $rows = 0;
        foreach ($query->select(['id', 'email', 'consent_at', 'confirmed_at', 'unsubscribed_at'])->lazyById(500) as $subscriber) {
            fputcsv($out, [
                self::cell($subscriber->email),
                $subscriber->status()->label(),
                self::date($subscriber->consent_at),
                self::date($subscriber->confirmed_at),
                self::date($subscriber->unsubscribed_at),
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
     * Neutralises spreadsheet formula injection (a value starting with = + - @ is treated as text).
     */
    private static function cell(string $value): string
    {
        return $value !== '' && in_array($value[0], ['=', '+', '-', '@', "\t", "\r"], true) ? "'".$value : $value;
    }
}
