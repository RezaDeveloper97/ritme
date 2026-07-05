<?php

namespace App\Models;

use App\Models\Concerns\HasLocalizedContent;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

/**
 * @OA\Schema(
 *     schema="TaskTemplate",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="key", type="string", example="log_bbt"),
 *     @OA\Property(property="title", type="object", @OA\Property(property="fa", type="string"), @OA\Property(property="en", type="string")),
 *     @OA\Property(property="description", type="object", nullable=true),
 *     @OA\Property(property="category", type="string", enum={"tracking","hydration","nutrition","supplement","exercise","mindfulness","medical"}),
 *     @OA\Property(property="icon", type="string", nullable=true),
 *     @OA\Property(property="cycle_phase", type="string", nullable=true, enum={"menstruation","follicular","ovulation","luteal"}),
 *     @OA\Property(property="is_active", type="boolean", example=true),
 *     @OA\Property(property="sort_order", type="integer", example=0)
 * )
 */
class TaskTemplate extends Model
{
    use HasLocalizedContent;

    protected $fillable = [
        'key',
        'title',
        'description',
        'category',
        'icon',
        'cycle_phase',
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
        return $this->hasMany(UserTaskCompletion::class);
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }

    /**
     * Templates that apply to the given phase (phase-specific or universal).
     */
    public function scopeForPhase(Builder $query, ?string $phase): Builder
    {
        return $query->where(function (Builder $q) use ($phase) {
            $q->whereNull('cycle_phase');
            if ($phase) {
                $q->orWhere('cycle_phase', $phase);
            }
        })->orderBy('sort_order');
    }
}
