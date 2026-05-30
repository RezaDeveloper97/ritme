<?php

namespace App\Models;

use App\Models\Concerns\HasLocalizedContent;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * @OA\Schema(
 *     schema="Article",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="slug", type="string", example="period-blood-clots"),
 *     @OA\Property(property="title", type="object", @OA\Property(property="fa", type="string"), @OA\Property(property="en", type="string")),
 *     @OA\Property(property="excerpt", type="object", nullable=true),
 *     @OA\Property(property="body", type="object", nullable=true),
 *     @OA\Property(property="cycle_phase", type="string", nullable=true, enum={"menstruation","follicular","ovulation","luteal"}),
 *     @OA\Property(property="category", type="string", nullable=true),
 *     @OA\Property(property="read_time_minutes", type="integer", nullable=true, example=4),
 *     @OA\Property(property="image_url", type="string", nullable=true),
 *     @OA\Property(property="is_published", type="boolean", example=true)
 * )
 */
class Article extends Model
{
    use HasLocalizedContent;

    protected $fillable = [
        'slug',
        'title',
        'excerpt',
        'body',
        'cycle_phase',
        'category',
        'read_time_minutes',
        'image_url',
        'is_published',
        'published_at',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'title' => 'array',
            'excerpt' => 'array',
            'body' => 'array',
            'read_time_minutes' => 'integer',
            'is_published' => 'boolean',
            'published_at' => 'datetime',
            'sort_order' => 'integer',
        ];
    }

    public function scopePublished(Builder $query): Builder
    {
        return $query->where('is_published', true);
    }

    /**
     * Articles for the given phase (phase-specific first, then general).
     */
    public function scopeForPhase(Builder $query, ?string $phase): Builder
    {
        return $query->where(function (Builder $q) use ($phase) {
            $q->whereNull('cycle_phase');
            if ($phase) {
                $q->orWhere('cycle_phase', $phase);
            }
        })->orderByRaw('cycle_phase IS NULL')->orderBy('sort_order');
    }
}
