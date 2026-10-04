<?php

declare(strict_types=1);

namespace App\Domain\Directory\Models;

use Database\Factories\Directory\DistrictFactory;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * @property int $id
 * @property int $city_id
 * @property string $name
 * @property string $slug
 * @property int $sort_order
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read City $city
 */
final class District extends Model
{
    /** @use HasFactory<DistrictFactory> */
    use HasFactory;

    protected $table = 'directory_districts';

    protected $fillable = ['city_id', 'name', 'slug', 'sort_order'];

    protected $attributes = ['sort_order' => 0];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['city_id' => 'integer', 'sort_order' => 'integer'];
    }

    protected static function newFactory(): DistrictFactory
    {
        return DistrictFactory::new();
    }

    /**
     * @return BelongsTo<City, $this>
     */
    public function city(): BelongsTo
    {
        return $this->belongsTo(City::class);
    }
}
