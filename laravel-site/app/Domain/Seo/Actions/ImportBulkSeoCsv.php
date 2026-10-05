<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

/**
 * Reads a bulk SEO CSV (ExportBulkSeoCsv format; `key` required, any of `title, description, robots, cornerstone`)
 * and saves it through BulkSaveSeoMeta — one transaction, one cache bump. Columns missing from the header are left
 * untouched; an empty cell clears the override (inherit). Unknown keys / deleted pages are reported, not fatal.
 */
final class ImportBulkSeoCsv
{
    public const MAX_ROWS = 2000;

    public function __construct(
        private readonly BulkSaveSeoMeta $save,
        private readonly ResolveSeoTarget $targets,
    ) {}

    /**
     * @return array{changed: array<string, array{old: array<string, mixed>, new: array<string, mixed>}>, skipped: int, errors: list<string>}
     */
    public function handle(string $path): array
    {
        $handle = fopen($path, 'rb');
        if ($handle === false) {
            return ['changed' => [], 'skipped' => 0, 'errors' => ['فایل خوانده نشد.']];
        }

        $header = fgetcsv($handle, null, ',', '"', '');
        $columns = array_map(static fn (mixed $h): string => strtolower(trim((string) preg_replace('/^\xEF\xBB\xBF/', '', (string) $h))), is_array($header) ? $header : []);
        if (! in_array('key', $columns, true)) {
            fclose($handle);

            return ['changed' => [], 'skipped' => 0, 'errors' => ['ستون key در سطر اول پیدا نشد.']];
        }

        $changes = [];
        $errors = [];
        $skipped = 0;
        $line = 1;
        while (($cells = fgetcsv($handle, null, ',', '"', '')) !== false && $line <= self::MAX_ROWS) {
            $line++;
            $row = [];
            foreach ($columns as $i => $column) {
                $row[$column] = self::uncell((string) ($cells[$i] ?? ''));
            }

            $key = trim($row['key'] ?? '');
            if ($key === '' || $this->targets->handle($key) === null) {
                $skipped++;
                if ($key !== '' && count($errors) < 20) {
                    $errors[] = "سطر {$line}: کلید «{$key}» شناخته نشد.";
                }

                continue;
            }

            $changes[$key] = array_intersect_key($row, array_flip(BulkSaveSeoMeta::FIELDS));
        }
        fclose($handle);

        $changed = $changes === [] ? [] : $this->save->handle($changes);

        return ['changed' => $changed, 'skipped' => $skipped, 'errors' => $errors];
    }

    /** Undoes the export's formula guard («'=…» → «=…»). */
    private static function uncell(string $value): string
    {
        return strlen($value) > 1 && $value[0] === "'" && in_array($value[1], ['=', '+', '-', '@'], true) ? substr($value, 1) : $value;
    }
}
