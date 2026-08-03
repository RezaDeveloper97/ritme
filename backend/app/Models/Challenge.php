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
 *
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="title", type="object", @OA\Property(property="fa", type="string"), @OA\Property(property="en", type="string")),
 *     @OA\Property(property="description", type="object", nullable=true),
 *     @OA\Property(property="cycle_day_from", type="integer", nullable=true, example=6),
 *     @OA\Property(property="cycle_day_to", type="integer", nullable=true, example=12),
 *     @OA\Property(property="category", type="string", nullable=true),
 *     @OA\Property(property="is_active", type="boolean", example=true)
 * )
 */
class Challenge extends Model
{
    use HasLocalizedContent;

    /**
     * Longest cycle a challenge can be targeted at. Cycles longer than this
     * exist, but content authored past day 35 would practically never be seen,
     * so the admin form caps the range here.
     */
    public const MAX_CYCLE_DAY = 35;

    protected $fillable = [
        'slug',
        'title',
        'description',
        'cycle_day_from',
        'cycle_day_to',
        'category',
        'is_active',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'title' => 'array',
            'description' => 'array',
            'cycle_day_from' => 'integer',
            'cycle_day_to' => 'integer',
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

    /**
     * Challenges that may run on the given cycle day — the ones whose range
     * contains it, plus every untargeted challenge (both bounds null).
     *
     * A null `$day` means we don't know where in her cycle the user is, so no
     * day-specific challenge can be justified and only untargeted ones match.
     */
    public function scopeForCycleDay(Builder $query, ?int $day): Builder
    {
        return $query->where(function (Builder $q) use ($day) {
            $q->where(fn (Builder $u) => $u->whereNull('cycle_day_from')->whereNull('cycle_day_to'));

            if ($day !== null) {
                $q->orWhere(fn (Builder $r) => $r
                    ->where(fn (Builder $lo) => $lo->whereNull('cycle_day_from')->orWhere('cycle_day_from', '<=', $day))
                    ->where(fn (Builder $hi) => $hi->whereNull('cycle_day_to')->orWhere('cycle_day_to', '>=', $day)));
            }
        });
    }

    /**
     * Whether this challenge is targeted at a specific stretch of the cycle at
     * all. Untargeted challenges are the pool's safe fallback.
     */
    public function isDayTargeted(): bool
    {
        return $this->cycle_day_from !== null || $this->cycle_day_to !== null;
    }

    /**
     * The targeting range as a Persian label for the admin panel. Digits stay
     * Latin, matching the rest of the panel's tables.
     */
    public function cycleDayLabel(): string
    {
        return match (true) {
            ! $this->isDayTargeted() => 'همه روزها',
            $this->cycle_day_from === null => "تا روز {$this->cycle_day_to}",
            $this->cycle_day_to === null => "از روز {$this->cycle_day_from}",
            $this->cycle_day_from === $this->cycle_day_to => "روز {$this->cycle_day_from}",
            default => "روز {$this->cycle_day_from} تا {$this->cycle_day_to}",
        };
    }

    /**
     * The in-memory twin of {@see scopeForCycleDay}, used by the picker, which
     * loads the pool once and then narrows it with soft filters.
     */
    public function appliesToCycleDay(?int $day): bool
    {
        if (! $this->isDayTargeted()) {
            return true;
        }

        if ($day === null) {
            return false;
        }

        return ($this->cycle_day_from === null || $day >= $this->cycle_day_from)
            && ($this->cycle_day_to === null || $day <= $this->cycle_day_to);
    }
}
