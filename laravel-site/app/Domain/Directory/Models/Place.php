<?php

declare(strict_types=1);

namespace App\Domain\Directory\Models;

use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Support\AgeRange;
use App\Domain\Directory\Support\OpeningHours;
use App\Domain\Media\Models\Media;
use App\Domain\Seo\Concerns\HasSeo;
use Database\Factories\Directory\PlaceFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * A place of the mother & child directory. Read through PlaceRepository (DTOs). On save (PlaceObserver): slug +
 * history, opening hours / phones normalised. `rating_avg`/`rating_count` (approved, non-demo reviews) and
 * `price_from` (cheapest service) are maintained by the review / service observers — never set them by hand.
 * Pivot writes go through SyncPlaceAmenities / SyncPlaceGallery.
 *
 * @property int $id
 * @property string $name
 * @property string $slug
 * @property int $category_id
 * @property int $city_id
 * @property int|null $district_id
 * @property string|null $summary
 * @property string|null $description
 * @property string|null $address
 * @property string|null $postal_code
 * @property float|null $latitude
 * @property float|null $longitude
 * @property list<string>|null $phones
 * @property string|null $website
 * @property int|null $age_min_months
 * @property int|null $age_max_months
 * @property array<string, mixed>|null $opening_hours
 * @property string|null $rules
 * @property string|null $cancellation_policy
 * @property int|null $cover_media_id
 * @property PlaceStatus $status
 * @property bool $is_verified
 * @property bool $is_demo
 * @property float $rating_avg
 * @property int $rating_count
 * @property int|null $price_from
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read PlaceCategory $category
 * @property-read City $city
 * @property-read District|null $district
 * @property-read Collection<int, Amenity> $amenities
 * @property-read Collection<int, PlaceService> $services
 * @property-read Collection<int, Media> $gallery
 * @property-read Collection<int, PlaceReview> $reviews
 */
final class Place extends Model
{
    /** @use HasFactory<PlaceFactory> */
    use HasFactory;

    use HasSeo;

    protected $table = 'directory_places';

    protected $fillable = [
        'name', 'slug', 'category_id', 'city_id', 'district_id', 'summary', 'description', 'address', 'postal_code',
        'latitude', 'longitude', 'phones', 'website', 'age_min_months', 'age_max_months', 'opening_hours', 'rules',
        'cancellation_policy', 'cover_media_id', 'status', 'is_verified', 'is_demo',
    ];

    protected $attributes = [
        'status' => 'draft',
        'is_verified' => false,
        'is_demo' => false,
        'rating_avg' => 0,
        'rating_count' => 0,
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'status' => PlaceStatus::class,
            'category_id' => 'integer',
            'city_id' => 'integer',
            'district_id' => 'integer',
            'latitude' => 'float',
            'longitude' => 'float',
            'phones' => 'array',
            'opening_hours' => 'array',
            'age_min_months' => 'integer',
            'age_max_months' => 'integer',
            'cover_media_id' => 'integer',
            'is_verified' => 'boolean',
            'is_demo' => 'boolean',
            'rating_avg' => 'float',
            'rating_count' => 'integer',
            'price_from' => 'integer',
        ];
    }

    protected static function newFactory(): PlaceFactory
    {
        return PlaceFactory::new();
    }

    /**
     * @param  Builder<self>  $query
     */
    public function scopePublished(Builder $query): void
    {
        $query->where($this->qualifyColumn('status'), PlaceStatus::Published->value);
    }

    public function openingHours(): OpeningHours
    {
        return OpeningHours::fromArray($this->opening_hours);
    }

    public function ageRange(): AgeRange
    {
        return new AgeRange($this->age_min_months, $this->age_max_months);
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
        return $this->belongsTo(City::class);
    }

    /**
     * @return BelongsTo<District, $this>
     */
    public function district(): BelongsTo
    {
        return $this->belongsTo(District::class);
    }

    /**
     * @return BelongsToMany<Amenity, $this>
     */
    public function amenities(): BelongsToMany
    {
        return $this->belongsToMany(Amenity::class, 'directory_place_amenity', 'place_id', 'amenity_id');
    }

    /**
     * @return HasMany<PlaceService, $this>
     */
    public function services(): HasMany
    {
        return $this->hasMany(PlaceService::class)->orderBy('sort_order')->orderBy('id');
    }

    /**
     * Ordered gallery images.
     *
     * @return BelongsToMany<Media, $this>
     */
    public function gallery(): BelongsToMany
    {
        return $this->belongsToMany(Media::class, 'directory_place_media', 'place_id', 'media_id')
            ->withPivot('sort_order')
            ->orderByPivot('sort_order');
    }

    /**
     * @return HasMany<PlaceReview, $this>
     */
    public function reviews(): HasMany
    {
        return $this->hasMany(PlaceReview::class);
    }

    /**
     * @return HasMany<PlaceSlug, $this>
     */
    public function previousSlugs(): HasMany
    {
        return $this->hasMany(PlaceSlug::class);
    }
}
