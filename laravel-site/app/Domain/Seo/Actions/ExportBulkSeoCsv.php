<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

/**
 * CSV of the bulk SEO editor rows (L7-06), in the column order ImportBulkSeoCsv reads back (`key, title,
 * description, robots, cornerstone`) plus read-only context columns. UTF-8 with BOM (Excel); cells that a spreadsheet
 * would run as a formula are prefixed with «'».
 */
final class ExportBulkSeoCsv
{
    public const HEADERS = ['key', 'title', 'description', 'robots', 'cornerstone', 'type', 'name', 'url', 'effective_title', 'effective_description', 'score'];

    public function __construct(private readonly ListBulkSeoRows $rows) {}

    public static function filename(): string
    {
        return 'seo-bulk-'.now()->format('Y-m-d').'.csv';
    }

    /**
     * @param  resource  $out
     * @param  list<array<string, mixed>>|null  $rows  null = every row
     */
    public function handle($out, ?array $rows = null): int
    {
        fwrite($out, "\xEF\xBB\xBF");
        fputcsv($out, self::HEADERS, ',', '"', '');

        $count = 0;
        foreach ($rows ?? $this->rows->handle() as $row) {
            fputcsv($out, array_map(self::cell(...), [
                $row['key'], $row['title'], $row['description'], $row['robots'], $row['cornerstone'] ? '1' : '0',
                $row['type_label'], $row['name'], $row['url'], $row['effective_title'], $row['effective_description'],
                $row['score'] ?? '',
            ]), ',', '"', '');
            $count++;
        }

        return $count;
    }

    private static function cell(mixed $value): string
    {
        $value = is_scalar($value) ? (string) $value : '';

        return $value !== '' && in_array($value[0], ['=', '+', '-', '@', "\t", "\r"], true) ? "'".$value : $value;
    }
}
