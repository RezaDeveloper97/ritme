<?php

declare(strict_types=1);

namespace App\Domain\Directory\Queries;

use App\Domain\Directory\Data\LandingComboData;
use App\Domain\Directory\Models\Place;
use Carbon\CarbonImmutable;
use Illuminate\Support\Facades\DB;

/**
 * Landing pages that have content: every active city and city × active category with at least one published place,
 * with place counts and the newest place update as lastmod. In city / category sort order; each city row comes right
 * before its categories. `$includeDemo` false (sitemaps) ignores demo places.
 */
final class LandingCombos
{
    public function __construct(private readonly bool $includeDemo = true) {}

    /**
     * @return list<LandingComboData>
     */
    public function get(): array
    {
        $places = (new Place)->getTable();

        $rows = Place::query()
            ->published()
            ->when(! $this->includeDemo, static fn ($q) => $q->where("{$places}.is_demo", false))
            ->join('directory_cities as c', 'c.id', '=', "{$places}.city_id")
            ->join('directory_categories as k', 'k.id', '=', "{$places}.category_id")
            ->where('c.is_active', true)
            ->where('k.is_active', true)
            ->groupBy('c.id', 'c.slug', 'c.name', 'c.sort_order', 'k.id', 'k.slug', 'k.name', 'k.sort_order')
            ->orderBy('c.sort_order')->orderBy('c.id')->orderBy('k.sort_order')->orderBy('k.id')
            ->toBase()
            ->get([
                'c.slug as city_slug', 'c.name as city_name', 'k.slug as category_slug', 'k.name as category_name',
                DB::raw('COUNT(*) as place_count'), DB::raw("MAX({$places}.updated_at) as lastmod"),
            ]);

        /** @var array<string, array{city: LandingComboData, categories: list<LandingComboData>}> $cities */
        $cities = [];
        foreach ($rows as $row) {
            $citySlug = (string) $row->city_slug;
            $lastmod = $row->lastmod === null ? null : CarbonImmutable::parse((string) $row->lastmod);
            $combo = new LandingComboData($citySlug, (string) $row->city_name, (string) $row->category_slug, (string) $row->category_name, (int) $row->place_count, $lastmod);

            $current = $cities[$citySlug]['city'] ?? null;
            $cities[$citySlug] = [
                'city' => new LandingComboData(
                    $citySlug,
                    (string) $row->city_name,
                    null,
                    null,
                    ($current->placeCount ?? 0) + $combo->placeCount,
                    self::latest($current?->lastmod, $lastmod),
                ),
                'categories' => [...($cities[$citySlug]['categories'] ?? []), $combo],
            ];
        }

        $combos = [];
        foreach ($cities as $city) {
            $combos[] = $city['city'];
            array_push($combos, ...$city['categories']);
        }

        return $combos;
    }

    private static function latest(?CarbonImmutable $a, ?CarbonImmutable $b): ?CarbonImmutable
    {
        if ($a === null || $b === null) {
            return $a ?? $b;
        }

        return $a->greaterThan($b) ? $a : $b;
    }
}
