<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Actions;

use App\Domain\Directory\Actions\SyncPlaceAmenities;
use App\Domain\Directory\Actions\SyncPlaceGallery;
use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Join\Enums\JoinRequestStatus;
use App\Domain\Directory\Join\Models\JoinRequest;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceService;
use App\Domain\Directory\Support\DirectoryActivity;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Database\Eloquent\Model;
use InvalidArgumentException;

/**
 * «تبدیل به پیش‌نویس مجموعه» (admin, L5-06): approves a pending join request and creates a DRAFT place from it, in
 * one transaction — name, category, city/district, address, landline, coordinates, «about» (description + a short
 * summary), age range from the chips, opening hours (same JSON shape), booking mode, existing amenities, the photos
 * in upload order (first = cover, the rest = gallery) and one service row per non-empty line of the free-text
 * services (name only — prices are filled in by the team). The contact person's name, mobile and email are NOT
 * copied: they are for the Ritme team, not for the public page. Nothing is published here; the team reviews the
 * draft and publishes it with ChangePlaceStatus. The approval is written to the activity log with the new place id.
 */
final class ApproveJoinRequest
{
    private const SUMMARY_MAX = 300;

    private const SERVICE_NAME_MAX = 191;

    private const SERVICES_MAX = 30;

    public function __construct(
        private readonly ConnectionInterface $db,
        private readonly SyncPlaceAmenities $amenities,
        private readonly SyncPlaceGallery $gallery,
    ) {}

    /**
     * @param  int|null  $categoryId  overrides the request's category (required when the request has none)
     * @param  int|null  $cityId  overrides the request's city (required when the request has none)
     *
     * @throws InvalidArgumentException when the request is not pending or has no category / city
     */
    public function handle(JoinRequest $request, ?int $categoryId = null, ?int $cityId = null, ?Model $causer = null): Place
    {
        if ($request->status !== JoinRequestStatus::Pending) {
            throw new InvalidArgumentException('Only pending join requests can be approved.');
        }

        $categoryId ??= $request->category_id;
        $cityId ??= $request->city_id;
        if ($categoryId === null || $cityId === null) {
            throw new InvalidArgumentException('A place needs a category and a city.');
        }

        /** @var Place $place */
        $place = $this->db->transaction(function () use ($request, $categoryId, $cityId): Place {
            $photoIds = $request->photos()->pluck('media.id')->map(intval(...))->values()->all();
            $age = $request->ageRange();
            $about = trim((string) $request->about);

            $place = new Place([
                'name' => $request->name,
                'category_id' => $categoryId,
                'city_id' => $cityId,
                'district_id' => $cityId === $request->city_id ? $request->district_id : null,
                'summary' => $about === '' ? null : self::summary($about),
                'description' => $about === '' ? null : $about,
                'address' => $request->address,
                'latitude' => $request->latitude,
                'longitude' => $request->longitude,
                'phones' => trim((string) $request->phone) === '' ? null : [trim((string) $request->phone)],
                'age_min_months' => $age->minMonths,
                'age_max_months' => $age->maxMonths,
                'opening_hours' => $request->opening_hours,
                'booking_mode' => $request->booking_mode,
                'cover_media_id' => $photoIds[0] ?? null,
                'status' => PlaceStatus::Draft,
            ]);
            $place->save();

            $amenityIds = Amenity::query()->whereKey(array_map(intval(...), $request->amenity_ids ?? []))->pluck('id')->map(intval(...))->all();
            $this->amenities->handle($place, $amenityIds);
            if (count($photoIds) > 1) {
                $this->gallery->handle($place, array_slice($photoIds, 1));
            }

            foreach (self::serviceNames((string) $request->services) as $position => $name) {
                PlaceService::query()->create(['place_id' => $place->id, 'name' => $name, 'sort_order' => $position]);
            }

            $request->status = JoinRequestStatus::Approved;
            $request->save();

            return $place;
        });

        DirectoryActivity::event($request, 'updated', 'directory.join.approved', [
            'old' => ['status' => JoinRequestStatus::Pending->value],
            'attributes' => ['status' => JoinRequestStatus::Approved->value],
            'place_id' => $place->id,
        ], $causer);
        DirectoryActivity::event($place, 'created', 'directory.place.created', [
            'attributes' => ['status' => PlaceStatus::Draft->value],
            'join_request_id' => $request->id,
        ], $causer);

        return $place;
    }

    private static function summary(string $about): string
    {
        $flat = trim(preg_replace('/\s+/u', ' ', $about) ?? '');

        return mb_strlen($flat) <= self::SUMMARY_MAX ? $flat : rtrim(mb_substr($flat, 0, self::SUMMARY_MAX - 1)).'…';
    }

    /**
     * @return list<string>
     */
    private static function serviceNames(string $services): array
    {
        $names = [];
        foreach (preg_split('/\R/u', $services) ?: [] as $line) {
            $line = trim(preg_replace('/^[\s\-–•*·]+/u', '', $line) ?? '');
            if ($line !== '') {
                $names[] = mb_substr($line, 0, self::SERVICE_NAME_MAX);
            }
        }

        return array_slice(array_values(array_unique($names)), 0, self::SERVICES_MAX);
    }
}
