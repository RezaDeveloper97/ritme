<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Models;

use App\Domain\Seo\Concerns\HasSeo;
use Database\Factories\Shop\CategoryFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * A shop category in a tree (roots are departments). Read through CatalogRepository::categoryTree(). The observer
 * fills the slug and refuses a parent that would create a cycle.
 *
 * @property int $id
 * @property int|null $parent_id
 * @property string $name
 * @property string $slug
 * @property string|null $intro
 * @property int|null $cover_media_id
 * @property string|null $illustration
 * @property int $sort_order
 * @property bool $is_active
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Category|null $parent
 * @property-read Collection<int, Category> $children
 * @property-read Collection<int, Product> $products
 */
final class Category extends Model
{
    /** @use HasFactory<CategoryFactory> */
    use HasFactory;

    use HasSeo;

    protected $table = 'shop_categories';

    protected $fillable = ['parent_id', 'name', 'slug', 'intro', 'cover_media_id', 'illustration', 'sort_order', 'is_active'];

    protected $attributes = ['sort_order' => 0, 'is_active' => true];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['parent_id' => 'integer', 'cover_media_id' => 'integer', 'sort_order' => 'integer', 'is_active' => 'boolean'];
    }

    protected static function newFactory(): CategoryFactory
    {
        return CategoryFactory::new();
    }

    /**
     * @param  Builder<self>  $query
     */
    public function scopeActive(Builder $query): void
    {
        $query->where($this->qualifyColumn('is_active'), true);
    }

    /**
     * @return BelongsTo<Category, $this>
     */
    public function parent(): BelongsTo
    {
        return $this->belongsTo(self::class, 'parent_id');
    }

    /**
     * @return HasMany<Category, $this>
     */
    public function children(): HasMany
    {
        return $this->hasMany(self::class, 'parent_id')->orderBy('sort_order')->orderBy('id');
    }

    /**
     * @return BelongsToMany<Product, $this>
     */
    public function products(): BelongsToMany
    {
        return $this->belongsToMany(Product::class, 'shop_category_product', 'category_id', 'product_id');
    }
}
