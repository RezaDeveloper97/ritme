<?php

declare(strict_types=1);

namespace App\Domain\Directory\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * Admin-edited copy of a landing page: a city (`category_id` null) or a city × category combination.
 *
 * @property int $id
 * @property int $city_id
 * @property int|null $category_id
 * @property string|null $h1
 * @property string|null $meta_title
 * @property string|null $meta_description
 * @property string|null $intro
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read City $city
 * @property-read PlaceCategory|null $category
 */
final class Landing extends Model
{
    protected $table = 'directory_landings';

    protected $fillable = ['city_id', 'category_id', 'h1', 'meta_title', 'meta_description', 'intro'];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['city_id' => 'integer', 'category_id' => 'integer'];
    }

    /**
     * @return BelongsTo<City, $this>
     */
    public function city(): BelongsTo
    {
        return $this->belongsTo(City::class);
    }

    /**
     * @return BelongsTo<PlaceCategory, $this>
     */
    public function category(): BelongsTo
    {
        return $this->belongsTo(PlaceCategory::class, 'category_id');
    }
}
