<?php

declare(strict_types=1);

namespace App\Domain\Directory\Models;

use App\Domain\Directory\Support\AgeRange;
use Database\Factories\Directory\PlaceServiceFactory;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * A service of a place with its announced price (whole Toman). Changes recalculate the place's `price_from`.
 *
 * @property int $id
 * @property int $place_id
 * @property string $name
 * @property int|null $duration_minutes
 * @property int|null $price
 * @property string|null $price_unit e.g. «هر جلسه»
 * @property string|null $details e.g. «گروهی», «اعتبار ۶۰ روز»
 * @property int|null $age_min_months
 * @property int|null $age_max_months
 * @property int $sort_order
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Place $place
 */
final class PlaceService extends Model
{
    /** @use HasFactory<PlaceServiceFactory> */
    use HasFactory;

    protected $table = 'directory_place_services';

    protected $fillable = [
        'place_id', 'name', 'duration_minutes', 'price', 'price_unit', 'details', 'age_min_months', 'age_max_months', 'sort_order',
    ];

    protected $attributes = ['sort_order' => 0];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'place_id' => 'integer',
            'duration_minutes' => 'integer',
            'price' => 'integer',
            'age_min_months' => 'integer',
            'age_max_months' => 'integer',
            'sort_order' => 'integer',
        ];
    }

    protected static function newFactory(): PlaceServiceFactory
    {
        return PlaceServiceFactory::new();
    }

    public function ageRange(): AgeRange
    {
        return new AgeRange($this->age_min_months, $this->age_max_months);
    }

    /**
     * @return BelongsTo<Place, $this>
     */
    public function place(): BelongsTo
    {
        return $this->belongsTo(Place::class);
    }
}
