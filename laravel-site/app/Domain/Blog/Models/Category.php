<?php

declare(strict_types=1);

namespace App\Domain\Blog\Models;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Seo\Concerns\HasSeo;
use Database\Factories\Blog\CategoryFactory;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * Magazine category, tree-light (one optional parent). Read through CategoryRepository.
 *
 * @property int $id
 * @property int|null $parent_id
 * @property string $name
 * @property string|null $label
 * @property string $slug
 * @property string|null $description
 * @property LifeStage|null $life_stage
 * @property int $sort_order
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class Category extends Model
{
    /** @use HasFactory<CategoryFactory> */
    use HasFactory;

    use HasSeo;

    protected $table = 'blog_categories';

    protected $fillable = ['parent_id', 'name', 'label', 'slug', 'description', 'life_stage', 'sort_order'];

    protected $attributes = ['sort_order' => 0];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'parent_id' => 'integer',
            'life_stage' => LifeStage::class,
            'sort_order' => 'integer',
        ];
    }

    protected static function newFactory(): CategoryFactory
    {
        return CategoryFactory::new();
    }

    /**
     * @return BelongsTo<self, $this>
     */
    public function parent(): BelongsTo
    {
        return $this->belongsTo(self::class, 'parent_id');
    }

    /**
     * @return HasMany<self, $this>
     */
    public function children(): HasMany
    {
        return $this->hasMany(self::class, 'parent_id');
    }

    /**
     * @return HasMany<Post, $this>
     */
    public function posts(): HasMany
    {
        return $this->hasMany(Post::class);
    }
}
