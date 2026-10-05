<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Models\Redirect;
use Illuminate\Database\Eloquent\Model;

/**
 * CSV export of every redirect, in the column order ImportRedirects reads back (`from, to, code, regex, note`) plus
 * the hit statistics. UTF-8 with BOM (Excel), rows streamed with lazyById(). Recorded in the activity log. The free-text
 * note is formula-guarded (paths always start with «/», targets with «/» or a scheme).
 */
final class ExportRedirects
{
    private const HEADERS = ['from', 'to', 'code', 'regex', 'note', 'hits', 'last_hit_at', 'auto'];

    public static function filename(): string
    {
        return 'redirects-'.jdate(null, 'Y-m-d', false).'.csv';
    }

    /**
     * @param  resource  $out
     */
    public function handle($out, ?Model $causer = null): int
    {
        activity(SaveRedirect::LOG)->causedBy($causer)->event('exported')->log('خروجی CSV ریدایرکت‌ها');

        fwrite($out, "\xEF\xBB\xBF");
        fputcsv($out, self::HEADERS, ',', '"', '');

        $rows = 0;
        foreach (Redirect::query()->lazyById(500) as $redirect) {
            fputcsv($out, [
                $redirect->from_path,
                (string) $redirect->to_url,
                (string) $redirect->code->value,
                $redirect->is_regex ? '1' : '0',
                self::cell((string) $redirect->note), // free text: formula guard (L9-04)
                (string) $redirect->hits,
                $redirect->last_hit_at?->toDateTimeString() ?? '',
                $redirect->is_auto ? '1' : '0',
            ], ',', '"', '');
            $rows++;
        }

        return $rows;
    }

    /** Spreadsheet formula injection guard (like the other exports): a cell starting with = + - @ TAB CR gets «'». */
    private static function cell(string $value): string
    {
        return $value !== '' && in_array($value[0], ['=', '+', '-', '@', "\t", "\r"], true) ? "'".$value : $value;
    }
}
