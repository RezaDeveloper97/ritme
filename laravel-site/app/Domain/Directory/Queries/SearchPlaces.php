<?php

declare(strict_types=1);

namespace App\Domain\Directory\Queries;

use App\Domain\Directory\Data\PlaceCardData;
use App\Domain\Directory\Data\PlacePage;
use App\Domain\Directory\Data\PlaceSearchCriteria;
use App\Domain\Directory\Enums\PlaceSort;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Support\OpeningHours;
use App\Domain\Search\Support\SearchTerms;
use App\Support\Text\PersianDigits;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Query\Builder as QueryBuilder;

/**
 * Published places matching PlaceSearchCriteria (listing L5-02, site search). Filters run in SQL (city, district,
 * category, child age inside the place's range, every amenity, text tokens via a portable LIKE over normalised
 * name/summary/description/address/category name — SQLite + MySQL). "Open now" depends on per-weekday JSON hours, so it
 * is applied in PHP to the matching id list before paginating (the directory is small: hundreds of places, not
 * millions). No paid placement: `Recommended` = verified first, then real rating.
 */
final class SearchPlaces
{
    /** LIKE escape character; `\` is not portable (MySQL string literals treat it as an escape). */
    private const ESCAPE = '!';

    private const EARTH_RADIUS_KM = 6371.0;

    public function __construct(private readonly PlaceSearchCriteria $criteria) {}

    public function get(): PlacePage
    {
        $c = $this->criteria;
        $table = (new Place)->getTable();

        $rows = $this->sorted($this->filtered())
            ->get(["{$table}.id", "{$table}.opening_hours", "{$table}.latitude", "{$table}.longitude"]);

        if ($c->openAt !== null) {
            $openAt = $c->openAt;
            $rows = $rows->filter(static fn (Place $row): bool => OpeningHours::fromArray($row->opening_hours)->isOpenAt($openAt));
        }

        $total = $rows->count();
        $slice = $rows->values()->slice(($c->page - 1) * $c->perPage, $c->perPage);
        $ids = $slice->map(static fn (Place $row): int => $row->id)->values()->all();

        if ($ids === []) {
            return new PlacePage([], $total, $c->page, $c->perPage);
        }

        $places = Place::query()->whereIn('id', $ids)->with(['category', 'city', 'district'])->get()->keyBy('id');
        $items = [];
        foreach ($ids as $id) {
            $place = $places->get($id);
            if ($place instanceof Place) {
                $items[] = PlaceCardData::fromModel($place, $this->distanceTo($place));
            }
        }

        return new PlacePage($items, $total, $c->page, $c->perPage);
    }

    /**
     * @return Builder<Place>
     */
    private function filtered(): Builder
    {
        $c = $this->criteria;
        $model = new Place;
        $table = $model->getTable();
        $query = Place::query()->published();

        if ($c->cityId !== null) {
            $query->where("{$table}.city_id", $c->cityId);
        }
        if ($c->districtId !== null) {
            $query->where("{$table}.district_id", $c->districtId);
        }
        if ($c->categoryId !== null) {
            $query->where("{$table}.category_id", $c->categoryId);
        }
        if ($c->ageMonths !== null) {
            $age = $c->ageMonths;
            $query->where(static fn (Builder $q) => $q->whereNull("{$table}.age_min_months")->orWhere("{$table}.age_min_months", '<=', $age))
                ->where(static fn (Builder $q) => $q->whereNull("{$table}.age_max_months")->orWhere("{$table}.age_max_months", '>=', $age));
        }
        foreach ($c->amenityIds as $amenityId) {
            $query->whereExists(static function (QueryBuilder $sub) use ($table, $amenityId): void {
                $sub->selectRaw('1')
                    ->from('directory_place_amenity')
                    ->whereColumn('directory_place_amenity.place_id', "{$table}.id")
                    ->where('directory_place_amenity.amenity_id', $amenityId);
            });
        }

        $this->matchText($query, $table);

        return $query;
    }

    /**
     * @param  Builder<Place>  $query
     */
    private function matchText(Builder $query, string $table): void
    {
        if ($this->criteria->tokens === []) {
            return;
        }

        $columns = array_map($this->normalized(...), ["{$table}.name", "{$table}.summary", "{$table}.description", "{$table}.address"]);
        $categories = (new PlaceCategory)->getTable();
        $categoryName = $this->normalized("{$categories}.name");

        foreach ($this->criteria->tokens as $token) {
            $patterns = $this->patterns($token);
            $query->where(static function (Builder $q) use ($columns, $patterns, $table, $categories, $categoryName): void {
                foreach ($columns as $column) {
                    foreach ($patterns as $pattern) {
                        $q->orWhereRaw("{$column} LIKE ? ESCAPE '".self::ESCAPE."'", [$pattern]);
                    }
                }
                $q->orWhereIn("{$table}.category_id", static function (QueryBuilder $sub) use ($categories, $categoryName, $patterns): void {
                    $sub->select("{$categories}.id")->from($categories)->where(static function (QueryBuilder $names) use ($categoryName, $patterns): void {
                        foreach ($patterns as $pattern) {
                            $names->orWhereRaw("{$categoryName} LIKE ? ESCAPE '".self::ESCAPE."'", [$pattern]);
                        }
                    });
                });
            });
        }
    }

    /**
     * @param  Builder<Place>  $query
     * @return Builder<Place>
     */
    private function sorted(Builder $query): Builder
    {
        $c = $this->criteria;
        $table = (new Place)->getTable();
        $sort = $c->sort === PlaceSort::Nearest && ! $c->hasReferencePoint() ? PlaceSort::Recommended : $c->sort;

        switch ($sort) {
            case PlaceSort::Nearest:
                $lat = (float) $c->latitude;
                $lng = (float) $c->longitude;
                $scale = cos(deg2rad($lat)); // equirectangular approximation: plain arithmetic, portable SQL
                $query->orderByRaw("CASE WHEN {$table}.latitude IS NULL OR {$table}.longitude IS NULL THEN 1 ELSE 0 END")
                    ->orderByRaw(
                        "(({$table}.latitude - ?) * ({$table}.latitude - ?)) + (({$table}.longitude - ?) * ?) * (({$table}.longitude - ?) * ?)",
                        [$lat, $lat, $lng, $scale, $lng, $scale],
                    );
                break;
            case PlaceSort::Rating:
                $query->orderByDesc("{$table}.rating_avg")->orderByDesc("{$table}.rating_count");
                break;
            case PlaceSort::Price:
                $query->orderByRaw("CASE WHEN {$table}.price_from IS NULL THEN 1 ELSE 0 END")->orderBy("{$table}.price_from");
                break;
            case PlaceSort::Newest:
                $query->orderByDesc("{$table}.created_at");
                break;
            default:
                $query->orderByDesc("{$table}.is_verified")->orderByDesc("{$table}.rating_avg")->orderByDesc("{$table}.rating_count");
        }

        return $query->orderByDesc("{$table}.id");
    }

    private function distanceTo(Place $place): ?float
    {
        $c = $this->criteria;
        if (! $c->hasReferencePoint() || $place->latitude === null || $place->longitude === null) {
            return null;
        }

        $dLat = deg2rad($place->latitude - (float) $c->latitude);
        $dLng = deg2rad($place->longitude - (float) $c->longitude);
        $a = sin($dLat / 2) ** 2 + cos(deg2rad((float) $c->latitude)) * cos(deg2rad($place->latitude)) * sin($dLng / 2) ** 2;

        return self::EARTH_RADIUS_KM * 2 * atan2(sqrt($a), sqrt(1 - $a));
    }

    private function normalized(string $column): string
    {
        $sql = "LOWER(COALESCE({$column}, ''))";
        foreach (SearchTerms::CHARACTER_MAP as $from => $to) {
            $sql = "REPLACE({$sql}, '{$from}', '{$to}')";
        }

        return $sql;
    }

    /**
     * @return list<string>
     */
    private function patterns(string $token): array
    {
        $variants = array_values(array_unique([$token, PersianDigits::toPersian($token)]));

        return array_map(static fn (string $variant): string => '%'.strtr($variant, [
            self::ESCAPE => self::ESCAPE.self::ESCAPE, '%' => self::ESCAPE.'%', '_' => self::ESCAPE.'_',
        ]).'%', $variants);
    }
}
