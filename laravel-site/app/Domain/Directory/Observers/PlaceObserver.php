<?php

declare(strict_types=1);

namespace App\Domain\Directory\Observers;

use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Support\OpeningHours;
use App\Domain\Directory\Support\PlaceSlugger;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use App\Support\Text\PersianDigits;
use Illuminate\Database\Eloquent\Model;

/**
 * Prepares a place on save (slug + history, normalised opening hours and phones, district must belong to the city)
 * and invalidates `directory`, `sitemap` and (cacheaside.always_bump) `pages`.
 */
final class PlaceObserver extends CacheBumpingObserver
{
    public function __construct(NamespaceBumper $bumper, private readonly PlaceSlugger $slugger)
    {
        parent::__construct($bumper);
    }

    protected function namespaces(Model $model): array
    {
        return ['directory', 'sitemap'];
    }

    public function saving(Place $place): void
    {
        $this->slugger->assign($place);

        if ($place->opening_hours !== null) {
            $place->opening_hours = OpeningHours::fromArray($place->opening_hours)->toArray();
        }

        if ($place->phones !== null) {
            $phones = array_map(static fn (mixed $phone): string => PersianDigits::toLatin(trim((string) $phone)), $place->phones);
            $place->phones = array_values(array_unique(array_filter($phones, static fn (string $phone): bool => $phone !== '')));
        }

        if ($place->district_id !== null && ($place->isDirty('district_id') || $place->isDirty('city_id'))) {
            $cityId = District::query()->whereKey($place->district_id)->value('city_id');
            if ($cityId === null || (int) $cityId !== $place->city_id) {
                $place->district_id = null;
            }
        }
    }

    public function updated(Place $place): void
    {
        $this->slugger->recordChange($place);
    }
}
