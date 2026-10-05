<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Data\ImportResult;
use App\Domain\Seo\Redirects\InvalidRedirect;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Domain\Seo\Redirects\Support\RedirectPath;
use Illuminate\Database\Eloquent\Model;

/**
 * CSV import of redirects (migration from the old WordPress site, or a round trip of ExportRedirects).
 *
 * Columns: `from, to, code, regex, note` — only `from` is required (code defaults to 301, an empty `to` with 410).
 * A header row is detected and skipped; comma, semicolon and tab delimiters and a UTF-8 BOM are accepted. A row
 * whose source already exists updates that redirect. Each row goes through SaveRedirect (normalisation, chain
 * collapse, loop check); invalid rows are skipped and reported. One activity-log entry for the whole import.
 */
final class ImportRedirects
{
    public const MAX_ROWS = 5000;

    private const MAX_ERRORS = 20;

    public function __construct(private readonly SaveRedirect $save) {}

    public function handle(string $file, ?Model $causer = null): ImportResult
    {
        $handle = @fopen($file, 'rb');
        if ($handle === false) {
            return new ImportResult(0, 0, 0, ['فایل خوانده نشد.']);
        }

        $created = $updated = $skipped = 0;
        $errors = [];
        $delimiter = null;
        $line = 0;

        try {
            while (($raw = fgets($handle)) !== false) {
                $line++;
                if ($line === 1) {
                    $raw = (string) preg_replace('/^\xEF\xBB\xBF/', '', $raw);
                }
                if (trim($raw) === '') {
                    continue;
                }

                $delimiter ??= $this->delimiter($raw);
                $row = array_map(static fn (?string $v): string => trim((string) $v), str_getcsv($raw, $delimiter, '"', ''));

                if ($line === 1 && $this->isHeader($row)) {
                    continue;
                }
                if ($created + $updated + $skipped >= self::MAX_ROWS) {
                    $errors[] = 'فقط '.self::MAX_ROWS.' ردیف اول وارد شد.';
                    break;
                }

                $isRegex = in_array(mb_strtolower($row[3] ?? ''), ['1', 'yes', 'true', 'regex', 'بله'], true);
                $from = $row[0];
                $to = $row[1] ?? '';
                $code = ($row[2] ?? '') !== '' ? $row[2] : ($to === '' ? '410' : '301');

                $hash = $isRegex ? sha1($from) : RedirectPath::hash($from);
                $existing = $from === '' ? null : Redirect::query()->where('from_hash', $hash)->where('is_regex', $isRegex)->first();

                try {
                    $this->save->handle($existing ?? new Redirect, [
                        'from_path' => $from,
                        'to_url' => $to,
                        'code' => $code,
                        'is_regex' => $isRegex,
                        'note' => ($row[4] ?? '') !== '' ? $row[4] : 'درون‌ریزی CSV',
                    ], $causer, log: false);
                    $existing === null ? $created++ : $updated++;
                } catch (InvalidRedirect $e) {
                    $skipped++;
                    if (count($errors) < self::MAX_ERRORS) {
                        $errors[] = 'ردیف '.$line.': '.$e->getMessage();
                    }
                }
            }
        } finally {
            fclose($handle);
        }

        activity(SaveRedirect::LOG)
            ->causedBy($causer)
            ->event('imported')
            ->withProperties(['created' => $created, 'updated' => $updated, 'skipped' => $skipped])
            ->log('درون‌ریزی ریدایرکت‌ها از CSV');

        return new ImportResult($created, $updated, $skipped, $errors);
    }

    private function delimiter(string $line): string
    {
        $counts = [',' => substr_count($line, ','), ';' => substr_count($line, ';'), "\t" => substr_count($line, "\t")];
        arsort($counts);

        return (string) array_key_first($counts);
    }

    /**
     * @param  list<string>  $row
     */
    private function isHeader(array $row): bool
    {
        $first = mb_strtolower($row[0] ?? '');

        return in_array($first, ['from', 'source', 'from_path', 'old', 'old url', 'مبدأ', 'مبدا'], true);
    }
}
