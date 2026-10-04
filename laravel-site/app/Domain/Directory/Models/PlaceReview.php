<?php

declare(strict_types=1);

namespace App\Domain\Directory\Models;

use App\Domain\Directory\Enums\ReviewStatus;
use Database\Factories\Directory\PlaceReviewFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * A moderated visitor review. Only approved reviews are shown; only approved, non-demo reviews count towards the
 * place's rating (PlaceReviewObserver → RecalculatePlaceRating).
 *
 * @property int $id
 * @property int $place_id
 * @property string $author_name
 * @property int $rating 1–5
 * @property array<string, int>|null $aspects ReviewAspect value => 1–5
 * @property string $body
 * @property ReviewStatus $status
 * @property bool $is_demo
 * @property string|null $ip_hash
 * @property Carbon|null $approved_at
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Place $place
 */
final class PlaceReview extends Model
{
    /** @use HasFactory<PlaceReviewFactory> */
    use HasFactory;

    protected $table = 'directory_reviews';

    protected $fillable = ['place_id', 'author_name', 'rating', 'aspects', 'body', 'status', 'is_demo', 'ip_hash', 'approved_at'];

    protected $attributes = ['status' => 'pending', 'is_demo' => false];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'place_id' => 'integer',
            'rating' => 'integer',
            'aspects' => 'array',
            'status' => ReviewStatus::class,
            'is_demo' => 'boolean',
            'approved_at' => 'datetime',
        ];
    }

    protected static function newFactory(): PlaceReviewFactory
    {
        return PlaceReviewFactory::new();
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
     * @return BelongsTo<Place, $this>
     */
    public function place(): BelongsTo
    {
        return $this->belongsTo(Place::class);
    }
}
