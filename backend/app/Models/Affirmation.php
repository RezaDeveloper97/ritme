<?php

namespace App\Models;

use App\Models\Concerns\HasLocalizedContent;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * @OA\Schema(
 *     schema="Affirmation",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="text", type="object", @OA\Property(property="fa", type="string"), @OA\Property(property="en", type="string")),
 *     @OA\Property(property="cycle_phase", type="string", nullable=true),
 *     @OA\Property(property="is_active", type="boolean", example=true)
 * )
 */
class Affirmation extends Model
{
    use HasLocalizedContent;

    protected $fillable = [
        'text',
        'cycle_phase',
        'is_active',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'text' => 'array',
            'is_active' => 'boolean',
            'sort_order' => 'integer',
        ];
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
