<?php

declare(strict_types=1);

namespace App\Domain\Directory\Models;

use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use Database\Factories\Directory\PlaceCategoryFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * A directory category with its schema.org LocalBusiness subtype (استخر → SportsActivityLocation, مهدکودک → ChildCare).
 *
 * @property int $id
 * @property string $name
 * @property string $slug
 * @property string|null $description
 * @property LocalBusinessType $schema_type
 * @property string|null $icon
 * @property int $sort_order
 * @property bool $is_active
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class PlaceCategory extends Model
{
    /** @use HasFactory<PlaceCategoryFactory> */
    use HasFactory;

    protected $table = 'directory_categories';

    protected $fillable = ['name', 'slug', 'description', 'schema_type', 'icon', 'sort_order', 'is_active'];

    protected $attributes = ['schema_type' => 'LocalBusiness', 'sort_order' => 0, 'is_active' => true];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['schema_type' => LocalBusinessType::class, 'sort_order' => 'integer', 'is_active' => 'boolean'];
    }

    protected static function newFactory(): PlaceCategoryFactory
    {
        return PlaceCategoryFactory::new();
    }

    /**
     * @param  Builder<self>  $query
     */
    public function scopeActive(Builder $query): void
    {
        $query->where($this->qualifyColumn('is_active'), true);
    }
}
