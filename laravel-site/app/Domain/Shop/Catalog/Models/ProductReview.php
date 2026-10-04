<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Models;

use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use Database\Factories\Shop\ProductReviewFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * A moderated product review. Only approved reviews are shown; only approved reviews that are not demo samples count
 * towards the product's rating (ProductReviewObserver → RecalculateProductRating). `is_verified_purchase` arrives with
 * orders (L6-05).
 *
 * @property int $id
 * @property int $product_id
 * @property string $author_name
 * @property int $rating 1–5
 * @property string $body
 * @property string|null $variant_label
 * @property ReviewStatus $status
 * @property bool $is_verified_purchase
 * @property bool $is_demo
 * @property string|null $ip_hash
 * @property Carbon|null $approved_at
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Product $product
 */
final class ProductReview extends Model
{
    /** @use HasFactory<ProductReviewFactory> */
    use HasFactory;

    protected $table = 'shop_reviews';

    protected $fillable = ['product_id', 'author_name', 'rating', 'body', 'variant_label', 'status', 'is_verified_purchase', 'is_demo', 'ip_hash', 'approved_at'];

    protected $attributes = ['status' => 'pending', 'is_verified_purchase' => false, 'is_demo' => false];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'product_id' => 'integer',
            'rating' => 'integer',
            'status' => ReviewStatus::class,
            'is_verified_purchase' => 'boolean',
            'is_demo' => 'boolean',
            'approved_at' => 'datetime',
        ];
    }

    protected static function newFactory(): ProductReviewFactory
    {
        return ProductReviewFactory::new();
    }

    /**
     * @param  Builder<self>  $query
     */
    public function scopeApproved(Builder $query): void
    {
        $query->where($this->qualifyColumn('status'), ReviewStatus::Approved->value);
    }

    /**
     * Reviews that count towards the rating aggregate: approved and real (not seeded demo samples).
     *
     * @param  Builder<self>  $query
     */
    public function scopeCounted(Builder $query): void
    {
        $query->where($this->qualifyColumn('status'), ReviewStatus::Approved->value)
            ->where($this->qualifyColumn('is_demo'), false);
    }

    /**
     * @return BelongsTo<Product, $this>
     */
    public function product(): BelongsTo
    {
        return $this->belongsTo(Product::class);
    }
}
