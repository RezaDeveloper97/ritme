<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * A previous slug of a published product (written by ProductSlugger when the slug changes).
 *
 * @property int $id
 * @property int $product_id
 * @property string $slug
 * @property Carbon|null $created_at
 */
final class ProductSlug extends Model
{
    public const UPDATED_AT = null;

    protected $table = 'shop_product_slugs';

    protected $fillable = ['product_id', 'slug'];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['product_id' => 'integer'];
    }

    /**
     * @return BelongsTo<Product, $this>
     */
    public function product(): BelongsTo
    {
        return $this->belongsTo(Product::class);
    }
}
