<?php

declare(strict_types=1);

namespace App\Domain\Media\Actions;

use Illuminate\Database\Connection;
use Illuminate\Database\DatabaseManager;

/**
 * Where media items are referenced: settings values whose key ends in `media_id` (logo, default OG image…), SEO
 * overrides (`seo_meta.og_media_id`), every column registered with `FindMediaUsages::column()` — contexts that
 * add a media foreign key (posts, products, places…) register it from their service provider's boot() — and every
 * rich-HTML column registered with `FindMediaUsages::html()`, scanned for `<img data-media-id="N">` (post bodies…).
 * Tables that do not exist (yet) are skipped. Used by the admin library ("find usages") and DeleteUnusedMedia.
 */
final class FindMediaUsages
{
    /**
     * @var array<string, array{table: string, column: string, label: string, title: string|null}>
     */
    private static array $columns = [
        'seo_meta.og_media_id' => ['table' => 'seo_meta', 'column' => 'og_media_id', 'label' => 'تصویر اشتراک‌گذاری سئو', 'title' => 'route_name'],
    ];

    /**
     * @var array<string, array{table: string, column: string, label: string, title: string|null}>
     */
    private static array $htmlColumns = [];

    private const HTML_CHUNK = 200;

    private readonly Connection $db;

    public function __construct(DatabaseManager $databases)
    {
        $this->db = $databases->connection();
    }

    /**
     * Registers a media reference column. `$titleColumn` (e.g. `title`) is shown next to the label.
     */
    public static function column(string $table, string $column, string $label, ?string $titleColumn = null): void
    {
        self::$columns["{$table}.{$column}"] = ['table' => $table, 'column' => $column, 'label' => $label, 'title' => $titleColumn];
    }

    /**
     * Registers a rich-HTML column whose `data-media-id="N"` attributes count as usages (e.g. `blog_posts.body`).
     */
    public static function html(string $table, string $column, string $label, ?string $titleColumn = null): void
    {
        self::$htmlColumns["{$table}.{$column}"] = ['table' => $table, 'column' => $column, 'label' => $label, 'title' => $titleColumn];
    }

    /**
     * @param  list<int>  $ids
     * @return array<int, list<string>> media id => human-readable usages (Persian); unused ids are omitted
     */
    public function handle(array $ids): array
    {
        $ids = array_values(array_unique(array_map('intval', $ids)));
        if ($ids === []) {
            return [];
        }

        $usages = [];
        $this->settings($ids, $usages);
        foreach (self::$columns as $source) {
            $this->scanColumn($source, $ids, $usages);
        }
        foreach (self::$htmlColumns as $source) {
            $this->scanHtml($source, $ids, $usages);
        }

        return $usages;
    }

    /**
     * @param  list<int>  $ids
     * @return list<int> the subset of `$ids` that is referenced somewhere
     */
    public function used(array $ids): array
    {
        return array_keys($this->handle($ids));
    }

    /**
     * @param  list<int>  $ids
     * @param  array<int, list<string>>  $usages
     */
    private function settings(array $ids, array &$usages): void
    {
        if (! $this->db->getSchemaBuilder()->hasTable('settings')) {
            return;
        }

        $wanted = array_flip($ids);
        $rows = $this->db->table('settings')->where('key', 'like', '%media_id')->get(['group', 'key', 'value']);
        foreach ($rows as $row) {
            $value = json_decode((string) $row->value, true);
            if (is_numeric($value) && isset($wanted[(int) $value])) {
                $usages[(int) $value][] = "تنظیمات: {$row->group}.{$row->key}";
            }
        }
    }

    /**
     * @param  array{table: string, column: string, label: string, title: string|null}  $source
     * @param  list<int>  $ids
     * @param  array<int, list<string>>  $usages
     */
    private function scanColumn(array $source, array $ids, array &$usages): void
    {
        $schema = $this->db->getSchemaBuilder();
        if (! $schema->hasTable($source['table']) || ! $schema->hasColumn($source['table'], $source['column'])) {
            return;
        }

        $title = $source['title'] !== null && $schema->hasColumn($source['table'], $source['title']) ? $source['title'] : null;
        $select = array_values(array_filter(['id', $source['column'], $title]));
        $rows = $this->db->table($source['table'])->whereIn($source['column'], $ids)->get($select);

        foreach ($rows as $row) {
            $row = (array) $row;
            $name = $title !== null && filled($row[$title] ?? null) ? (string) $row[$title] : '#'.(string) ($row['id'] ?? '');
            $usages[(int) $row[$source['column']]][] = "{$source['label']}: {$name}";
        }
    }

    /**
     * Reads only rows that contain a media reference at all, in id chunks, and collects every `data-media-id` value.
     *
     * @param  array{table: string, column: string, label: string, title: string|null}  $source
     * @param  list<int>  $ids
     * @param  array<int, list<string>>  $usages
     */
    private function scanHtml(array $source, array $ids, array &$usages): void
    {
        $schema = $this->db->getSchemaBuilder();
        if (! $schema->hasTable($source['table']) || ! $schema->hasColumn($source['table'], $source['column'])) {
            return;
        }

        $title = $source['title'] !== null && $schema->hasColumn($source['table'], $source['title']) ? $source['title'] : null;
        $select = array_values(array_filter(['id', $source['column'], $title]));
        $wanted = array_flip($ids);

        $this->db->table($source['table'])
            ->where($source['column'], 'like', '%data-media-id%')
            ->select($select)
            ->chunkById(self::HTML_CHUNK, function ($rows) use ($source, $title, $wanted, &$usages): void {
                foreach ($rows as $row) {
                    $row = (array) $row;
                    if (! preg_match_all('/data-media-id\s*=\s*["\']?\s*(\d+)/i', (string) $row[$source['column']], $matches)) {
                        continue;
                    }

                    $name = $title !== null && filled($row[$title] ?? null) ? (string) $row[$title] : '#'.(string) ($row['id'] ?? '');
                    foreach (array_unique(array_map('intval', $matches[1])) as $id) {
                        if (isset($wanted[$id])) {
                            $usages[$id][] = "{$source['label']}: {$name}";
                        }
                    }
                }
            });
    }
}
