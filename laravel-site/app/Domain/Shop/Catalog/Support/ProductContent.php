<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Support;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Shop\Catalog\Models\Product;
use App\Support\Html\ExternalLinks;
use App\Support\Html\HtmlText;
use App\Support\Html\RichHtmlSanitizer;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Prepares a product's content on save: the long description goes through the allow-list sanitiser (own-media images
 * only, external links get rel="noopener"); short description and badge become plain text; specs become a clean
 * list of {label, value}; the size chart a rectangular {columns, rows} table; life stages valid LifeStage values only.
 */
final class ProductContent
{
    public const MAX_SPECS = 30;

    public const MAX_CHART_COLUMNS = 8;

    public const MAX_CHART_ROWS = 30;

    /**
     * @param  list<string>  $ownHosts
     */
    public function __construct(private readonly RichHtmlSanitizer $sanitizer, private readonly array $ownHosts) {}

    public static function fromConfig(Config $config): self
    {
        $host = parse_url((string) $config->get('app.url'), PHP_URL_HOST);
        $hosts = is_string($host) && $host !== '' ? [strtolower($host)] : [];
        if ($hosts !== [] && ! str_starts_with($hosts[0], 'www.')) {
            $hosts[] = 'www.'.$hosts[0];
        }

        $disk = (string) $config->get('media.disk', 'public');
        $mediaUrl = (string) $config->get("filesystems.disks.{$disk}.url", '/media');
        $mediaPath = parse_url($mediaUrl, PHP_URL_PATH);
        $prefix = rtrim(is_string($mediaPath) && $mediaPath !== '' ? $mediaPath : '/media', '/').'/';

        return new self(new RichHtmlSanitizer($hosts, [$prefix]), $hosts);
    }

    public function prepare(Product $product): void
    {
        if (! $product->exists || $product->isDirty('description')) {
            $html = ExternalLinks::apply($this->sanitizer->sanitize((string) $product->getAttribute('description')), $this->ownHosts);
            $product->description = HtmlText::plain($html) === '' ? null : $html;
        }

        foreach (['short_description', 'badge'] as $plain) {
            if (! $product->exists || $product->isDirty($plain)) {
                $text = HtmlText::plain((string) $product->getAttribute($plain));
                $product->setAttribute($plain, $text === '' ? null : $text);
            }
        }

        if (! $product->exists || $product->isDirty('specs')) {
            $product->specs = self::specs($product->specs);
        }

        if (! $product->exists || $product->isDirty('size_chart')) {
            $product->size_chart = self::sizeChart($product->size_chart);
        }

        if (! $product->exists || $product->isDirty('life_stages')) {
            $stages = array_map(static fn (LifeStage $stage): string => $stage->value, $product->lifeStages());
            $product->life_stages = $stages === [] ? null : array_values(array_unique($stages));
        }
    }

    /**
     * Accepts a list of {label, value} or a label => value map.
     *
     * @return list<array{label: string, value: string}>|null
     */
    public static function specs(mixed $specs): ?array
    {
        if (! is_array($specs)) {
            return null;
        }

        $rows = [];
        foreach ($specs as $key => $row) {
            [$label, $value] = is_array($row)
                ? [$row['label'] ?? null, $row['value'] ?? null]
                : [is_string($key) ? $key : null, $row];
            $label = self::cell($label);
            $value = self::cell($value);
            if ($label !== '' && $value !== '') {
                $rows[] = ['label' => $label, 'value' => $value];
            }
            if (count($rows) === self::MAX_SPECS) {
                break;
            }
        }

        return $rows === [] ? null : $rows;
    }

    /**
     * @return array{columns: list<string>, rows: list<list<string>>}|null
     */
    public static function sizeChart(mixed $chart): ?array
    {
        if (! is_array($chart) || ! is_array($chart['columns'] ?? null)) {
            return null;
        }

        $columns = array_slice(array_values(array_filter(array_map(self::cell(...), $chart['columns']), static fn (string $c): bool => $c !== '')), 0, self::MAX_CHART_COLUMNS);
        if ($columns === []) {
            return null;
        }

        $rows = [];
        foreach (is_array($chart['rows'] ?? null) ? $chart['rows'] : [] as $row) {
            if (! is_array($row)) {
                continue;
            }
            $cells = array_map(self::cell(...), array_slice(array_values($row), 0, count($columns)));
            $cells = array_pad($cells, count($columns), '');
            if (implode('', $cells) !== '') {
                $rows[] = $cells;
            }
            if (count($rows) === self::MAX_CHART_ROWS) {
                break;
            }
        }

        return $rows === [] ? null : ['columns' => $columns, 'rows' => $rows];
    }

    private static function cell(mixed $value): string
    {
        if (! is_scalar($value)) {
            return '';
        }

        return mb_substr(HtmlText::plain((string) $value), 0, 191);
    }
}
