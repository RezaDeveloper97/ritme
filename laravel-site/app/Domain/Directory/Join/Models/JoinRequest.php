<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Models;

use App\Domain\Directory\Join\Enums\AgeGroup;
use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Directory\Join\Enums\JoinRequestStatus;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Support\AgeRange;
use App\Domain\Media\Models\Media;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Support\Carbon;

/**
 * A «ثبت مجموعه» request (/directory/join, L5-05), written only by SubmitJoinRequest and reviewed in the admin (L5-06).
 * Photos are pipeline media in upload order (`photos`).
 *
 * @property int $id
 * @property string $code
 * @property JoinRequestStatus $status
 * @property string $name
 * @property int|null $category_id
 * @property int|null $city_id
 * @property int|null $district_id
 * @property string $contact_name
 * @property string $mobile
 * @property string|null $email
 * @property string $address
 * @property string|null $phone
 * @property float|null $latitude
 * @property float|null $longitude
 * @property string|null $about
 * @property list<string>|null $age_groups
 * @property list<int>|null $amenity_ids
 * @property array<string, list<array{opens: string, closes: string}>>|null $opening_hours
 * @property string|null $services
 * @property BookingMode $booking_mode
 * @property Carbon|null $terms_accepted_at
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read PlaceCategory|null $category
 * @property-read City|null $city
 * @property-read District|null $district
 * @property-read Collection<int, Media> $photos
 */
final class JoinRequest extends Model
{
    protected $table = 'directory_join_requests';

    protected $fillable = [
        'code', 'status', 'name', 'category_id', 'city_id', 'district_id', 'contact_name', 'mobile', 'email',
        'address', 'phone', 'latitude', 'longitude', 'about', 'age_groups', 'amenity_ids', 'opening_hours',
        'services', 'booking_mode', 'terms_accepted_at',
    ];

    protected $attributes = [
        'status' => 'pending',
        'booking_mode' => 'online',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'status' => JoinRequestStatus::class,
            'booking_mode' => BookingMode::class,
            'latitude' => 'float',
            'longitude' => 'float',
            'age_groups' => 'array',
            'amenity_ids' => 'array',
            'opening_hours' => 'array',
            'terms_accepted_at' => 'datetime',
        ];
    }

    /**
     * @return BelongsTo<PlaceCategory, $this>
     */
    public function category(): BelongsTo
    {
        return $this->belongsTo(PlaceCategory::class, 'category_id');
    }

    /**
     * @return BelongsTo<City, $this>
     */
    public function city(): BelongsTo
    {
        return $this->belongsTo(City::class, 'city_id');
    }

    /**
     * @return BelongsTo<District, $this>
     */
    public function district(): BelongsTo
    {
        return $this->belongsTo(District::class, 'district_id');
    }

    /**
     * @return BelongsToMany<Media, $this>
     */
    public function photos(): BelongsToMany
    {
        return $this->belongsToMany(Media::class, 'directory_join_request_media', 'join_request_id', 'media_id')
            ->withPivot('sort_order')
            ->orderByPivot('sort_order');
    }

    /** The chosen age chips as the month range a place stores. */
    public function ageRange(): AgeRange
    {
        return AgeGroup::range(array_values(array_filter(array_map(
            AgeGroup::tryFrom(...),
            $this->age_groups ?? [],
        ))));
    }
}
