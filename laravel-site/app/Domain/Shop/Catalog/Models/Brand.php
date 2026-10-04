<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Models;

use Database\Factories\Shop\BrandFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * A product brand (shown on product cards — single-seller shop).
 *
 * @property int $id
 * @property string $name
 * @property string $slug
 * @property string|null $description
 * @property int|null $logo_media_id
 * @property int $sort_order
 * @property bool $is_active
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class Brand extends Model
{
    /** @use HasFactory<BrandFactory> */
    use HasFactory;

    protected $table = 'shop_brands';

    protected $fillable = ['name', 'slug', 'description', 'logo_media_id', 'sort_order', 'is_active'];

    protected $attributes = ['sort_order' => 0, 'is_active' => true];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['logo_media_id' => 'integer', 'sort_order' => 'integer', 'is_active' => 'boolean'];
    }

    protected static function newFactory(): BrandFactory
    {
        return BrandFactory::new();
    }

    /**
     * @param  Builder<self>  $query
     */
    public function scopeActive(Builder $query): void
    {
        $query->where($this->qualifyColumn('is_active'), true);
    }

    /**
     * @return HasMany<Product, $this>
     */
    public function products(): HasMany
    {
        return $this->hasMany(Product::class);
    }
}
