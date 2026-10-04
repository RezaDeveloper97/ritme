<?php

declare(strict_types=1);

namespace App\Domain\Directory\Models;

use Database\Factories\Directory\AmenityFactory;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * @property int $id
 * @property string $name
 * @property string $slug
 * @property string|null $icon sprite icon name
 * @property bool $is_filter
 * @property int $sort_order
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class Amenity extends Model
{
    /** @use HasFactory<AmenityFactory> */
    use HasFactory;

    protected $table = 'directory_amenities';

    protected $fillable = ['name', 'slug', 'icon', 'is_filter', 'sort_order'];

    protected $attributes = ['is_filter' => false, 'sort_order' => 0];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['is_filter' => 'boolean', 'sort_order' => 'integer'];
    }

    protected static function newFactory(): AmenityFactory
    {
        return AmenityFactory::new();
    }
}
