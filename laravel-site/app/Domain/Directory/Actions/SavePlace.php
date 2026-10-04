<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Support\DirectoryActivity;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Database\Eloquent\Model;

/**
 * Creates or updates a place from the admin form: attributes (never status / rating / price — those have their own
 * paths), then amenities and the ordered gallery through their sync actions, in one transaction. Status changes go
 * through ChangePlaceStatus; a new place starts as a draft. Creation is written to the activity log.
 */
final class SavePlace
{
    /** Columns the form may never write: status has its own logged action, the rest is computed. */
    private const GUARDED = ['status', 'rating_avg', 'rating_count', 'price_from'];

    public function __construct(
        private readonly ConnectionInterface $db,
        private readonly SyncPlaceAmenities $amenities,
        private readonly SyncPlaceGallery $gallery,
    ) {}

    /**
     * @param  array<string, mixed>  $attributes
     * @param  list<int>|null  $amenityIds  null = leave as is
     * @param  list<int>|null  $galleryMediaIds  null = leave as is
     */
    public function handle(Place $place, array $attributes, ?array $amenityIds = null, ?array $galleryMediaIds = null, ?Model $causer = null): Place
    {
        $creating = ! $place->exists;

        /** @var Place $place */
        $place = $this->db->transaction(function () use ($place, $attributes, $amenityIds, $galleryMediaIds): Place {
            $place->fill(array_diff_key($attributes, array_flip(self::GUARDED)));
            $place->save();

            if ($amenityIds !== null) {
                $this->amenities->handle($place, $amenityIds);
            }
            if ($galleryMediaIds !== null) {
                $this->gallery->handle($place, $galleryMediaIds);
            }

            return $place;
        });

        if ($creating) {
            DirectoryActivity::event($place, 'created', 'directory.place.created', ['attributes' => ['status' => $place->status->value]], $causer);
        }

        return $place;
    }
}
