<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Actions;

use App\Domain\Directory\Booking\Support\MobileMask;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use App\Support\Money\Money;
use App\Support\Text\PersianDigits;
use DateTimeInterface;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * CSV export of the orders list (admin, L6-06): code, placed (Jalali, Tehran), status, payment status, items, subtotal
 * / shipping / total in TOMANS, province, city, MASKED mobile, preferred delivery day. Personal data stays minimal —
 * the recipient's name, full number and street address are only on the order page (needed to deliver), not in files
 * that travel. UTF-8 with BOM, Persian headers, Latin digits so spreadsheets sort and sum; rows streamed with
 * lazyById(); cells guarded against formula injection; each export is written to the activity log
 * (`shop`, `shop.order.exported`).
 */
final class ExportOrders
{
    private const BOM = "\xEF\xBB\xBF";

    private const HEADERS = [
        'کد سفارش', 'ثبت', 'وضعیت', 'پرداخت', 'تعداد کالا', 'جمع کالاها (تومان)', 'هزینه ارسال (تومان)', 'مبلغ کل (تومان)',
        'استان', 'شهر', 'موبایل', 'روز تحویل', 'نمونه',
    ];

    /**
     * Orders matching the admin list (status tab, placed-date range, search on code / exact mobile).
     *
     * @return Builder<Order>
     */
    public static function query(?OrderStatus $status = null, ?string $from = null, ?string $until = null, ?string $search = null): Builder
    {
        $query = Order::query();
        if ($status !== null) {
            $query->where('status', $status->value);
        }
        if ($from !== null && preg_match('/^\d{4}-\d{2}-\d{2}$/', $from) === 1) {
            $query->whereDate('created_at', '>=', $from);
        }
        if ($until !== null && preg_match('/^\d{4}-\d{2}-\d{2}$/', $until) === 1) {
            $query->whereDate('created_at', '<=', $until);
        }

        return self::search($query, $search);
    }

    /**
     * Search by order code (partial, case-insensitive) or by the full mobile number (exact — a partial number search
     * would let the list be used to guess numbers).
     *
     * @param  Builder<Order>  $query
     * @return Builder<Order>
     */
    public static function search(Builder $query, ?string $search): Builder
    {
        $search = trim(PersianDigits::toLatin((string) $search));
        if ($search === '') {
            return $query;
        }

        $like = '%'.str_replace(['\\', '%', '_'], ['\\\\', '\\%', '\\_'], strtoupper($search)).'%';
        $digits = preg_replace('/\D/', '', $search) ?? '';

        return $query->where(static function (Builder $q) use ($like, $digits): void {
            $q->where('code', 'like', $like);
            if (strlen($digits) === 11) {
                $q->orWhere('mobile', $digits);
            }
        });
    }

    public static function filename(): string
    {
        return 'shop-orders-'.jdate(null, 'Y-m-d', false).'.csv';
    }

    /**
     * Logs the export, then writes the CSV to `$out` (php://output inside a streamed response).
     *
     * @param  resource  $out
     * @return int rows written (without the header)
     */
    public function handle($out, ?OrderStatus $status = null, ?string $from = null, ?string $until = null, ?string $search = null, ?Model $causer = null): int
    {
        $query = self::query($status, $from, $until, $search);

        activity(ChangeOrderStatus::LOG)
            ->causedBy($causer)
            ->event('exported')
            ->withProperties([
                'rows' => (clone $query)->count(),
                'status' => $status?->value,
                'from' => $from,
                'until' => $until,
            ])
            ->log('shop.order.exported');

        fwrite($out, self::BOM);
        fputcsv($out, self::HEADERS, ',', '"', '');

        $rows = 0;
        foreach ($query->lazyById(500) as $order) {
            fputcsv($out, [
                $order->code,
                self::date($order->created_at, 'Y/m/d H:i'),
                $order->status->label(),
                $order->payment_status->label(),
                (string) $order->items_count,
                self::toman($order->subtotal),
                $order->shipping_fee === null ? '' : self::toman($order->shipping_fee),
                self::toman($order->total),
                self::cell($order->province),
                self::cell($order->city),
                PersianDigits::toLatin(MobileMask::mask($order->mobile)),
                self::date($order->delivery_date, 'Y/m/d'),
                $order->is_demo ? 'بله' : '',
            ], ',', '"', '');
            $rows++;
        }

        return $rows;
    }

    private static function toman(Money $amount): string
    {
        return (string) $amount->toToman();
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
