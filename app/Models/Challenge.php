<?php

namespace App\Models;

use App\Models\Concerns\HasLocalizedContent;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

/**
 * @OA\Schema(
 *     schema="Challenge",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="title", type="object", @OA\Property(property="fa", type="string"), @OA\Property(property="en", type="string")),
 *     @OA\Property(property="description", type="object", nullable=true),
 *     @OA\Property(property="cycle_phase", type="string", nullable=true),
 *     @OA\Property(property="category", type="string", nullable=true),
 *     @OA\Property(property="difficulty", type="string", nullable=true, enum={"easy","medium","hard"}),
 *     @OA\Property(property="is_active", type="boolean", example=true)
 * )
 */
class Challenge extends Model
{
    use HasLocalizedContent;

    protected $fillable = [
        'title',
        'description',
        'cycle_phase',
        'category',
        'difficulty',
        'is_active',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'title' => 'array',
            'description' => 'array',
            'is_active' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    public function completions(): HasMany
    {
        return $this->hasMany(UserChallengeCompletion::class);
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }

    public function scopeForPhase(Builder $query, ?string $phase): Builder
    {
        return $query->where(function (Builder $q) use ($phase) {
            $q->whereNull('cycle_phase');
            if ($phase) {
                $q->orWhere('cycle_phase', $phase);
            }
        });
    }
}
